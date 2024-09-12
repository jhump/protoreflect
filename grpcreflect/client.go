package grpcreflect

//lint:file-ignore SA1019 The refv1alpha package is deprecated, but we need it in order to adapt it to new version

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	refv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/protoresolve"
	"github.com/jhump/protoreflect/v2/protowrap"
)

// If we try the v1 reflection API and get back "not implemented", we'll wait
// this long before trying v1 again. This allows a long-lived client to
// dynamically switch from v1alpha to v1 if the underlying server is updated
// to support it. But it also prevents every stream request from always trying
// v1 first: if we try it and see it fail, we shouldn't continually retry it
// if we expect it will fail again.
const durationBetweenV1Attempts = time.Hour

// elementNotFoundError is the error returned by reflective operations where the
// server does not recognize a given file name, symbol name, or extension.
type elementNotFoundError struct {
	path string
	name protoreflect.FullName
	kind elementKind
	tag  protoreflect.FieldNumber // only used when kind == elementKindExtension

	// only errors with a kind of elementKindFile will have a cause, which means
	// the named file count not be resolved because of a dependency that could
	// not be found where cause describes the missing dependency
	cause *elementNotFoundError
}

type elementKind int

const (
	elementKindSymbol elementKind = iota
	elementKindFile
	elementKindExtension
)

func symbolNotFound(symbol protoreflect.FullName, cause *elementNotFoundError) error {
	if cause != nil && cause.kind == elementKindSymbol && cause.name == symbol {
		// no need to wrap
		return cause
	}
	return &elementNotFoundError{name: symbol, kind: elementKindSymbol, cause: cause}
}

func extensionNotFound(extendee protoreflect.FullName, tag protoreflect.FieldNumber, cause *elementNotFoundError) error {
	if cause != nil && cause.kind == elementKindExtension && cause.name == extendee && cause.tag == tag {
		// no need to wrap
		return cause
	}
	return &elementNotFoundError{name: extendee, tag: tag, kind: elementKindExtension, cause: cause}
}

func fileNotFound(file string, cause *elementNotFoundError) error {
	if cause != nil && cause.kind == elementKindFile && cause.path == file {
		// no need to wrap
		return cause
	}
	return &elementNotFoundError{path: file, kind: elementKindFile, cause: cause}
}

func (e *elementNotFoundError) Error() string {
	first := true
	var b bytes.Buffer
	for ; e != nil; e = e.cause {
		if first {
			first = false
		} else {
			_, _ = fmt.Fprint(&b, "\ncaused by: ")
		}
		switch e.kind {
		case elementKindSymbol:
			_, _ = fmt.Fprintf(&b, "symbol not found: %s", e.name)
		case elementKindExtension:
			_, _ = fmt.Fprintf(&b, "extension not found: tag %d for %s", e.tag, e.name)
		default:
			_, _ = fmt.Fprintf(&b, "file not found: %s", e.path)
		}
	}
	return b.String()
}

// IsElementNotFoundError determines if the given error indicates that a file
// name, symbol name, or extension field was could not be found by the server.
func IsElementNotFoundError(err error) bool {
	var enfe *elementNotFoundError
	return errors.As(err, &enfe)
}

// ProtocolError is an error returned when the server sends a response of the
// wrong type.
type ProtocolError struct {
	missingType reflect.Type
}

func (p ProtocolError) Error() string {
	return fmt.Sprintf("Protocol error: response was missing %v", p.missingType)
}

// Client is a client connection to a server for performing reflection calls
// and resolving remote symbols.
type Client struct {
	ctx                 context.Context
	now                 func() time.Time
	stubV1              refv1.ServerReflectionClient
	stubV1Alpha         refv1alpha.ServerReflectionClient
	allowMissing        bool
	fallbackResolver    protodesc.Resolver
	fallbackExtResolver protoregistry.ExtensionTypeResolver

	connMu      sync.Mutex
	cancel      context.CancelFunc
	stream      refv1.ServerReflection_ServerReflectionInfoClient
	useV1Alpha  bool
	lastTriedV1 time.Time

	cacheMu      sync.RWMutex
	protosByName map[string]*descriptorpb.FileDescriptorProto
	descriptors  protoresolve.Registry
}

// ClientOption is an option that can be used to configure the behavior of
// a reflection client created with one of the various NewClient* functions
// in this package.
type ClientOption func(*Client)

// NewClientV1 creates a new Client using the v1 version of reflection
// with the given root context and using the given RPC stub for talking to the
// server.
func NewClientV1(ctx context.Context, stub refv1.ServerReflectionClient, opts ...ClientOption) *Client {
	return newClient(ctx, stub, nil, opts)
}

// NewClientV1Alpha creates a new Client using the v1alpha version of reflection
// with the given root context and using the given RPC stub for talking to the
// server.
func NewClientV1Alpha(ctx context.Context, stub refv1alpha.ServerReflectionClient, opts ...ClientOption) *Client {
	return newClient(ctx, nil, stub, opts)
}

func newClient(ctx context.Context, stubv1 refv1.ServerReflectionClient, stubv1alpha refv1alpha.ServerReflectionClient, opts []ClientOption) *Client {
	cr := &Client{
		ctx:          ctx,
		now:          time.Now,
		stubV1:       stubv1,
		stubV1Alpha:  stubv1alpha,
		protosByName: map[string]*descriptorpb.FileDescriptorProto{},
	}
	for _, opt := range opts {
		opt(cr)
	}
	// don't leak a grpc stream
	runtime.SetFinalizer(cr, (*Client).Reset)
	return cr
}

// NewClientAuto creates a new Client that will use either v1 or v1alpha version
// of reflection (based on what the server supports) with the given root context
// and using the given client connection.
//
// It will first the v1 version of the reflection service. If it gets back an
// "Unimplemented" error, it will fall back to using the v1alpha version. It
// will remember which version the server supports for any subsequent operations
// that need to re-invoke the streaming RPC. But, if it's a very long-lived
// client, it will periodically retry the v1 version (in case the server is
// updated to support it also). The period for these retries is every hour.
func NewClientAuto(ctx context.Context, cc grpc.ClientConnInterface, opts ...ClientOption) *Client {
	stubv1 := refv1.NewServerReflectionClient(cc)
	stubv1alpha := refv1alpha.NewServerReflectionClient(cc)
	return newClient(ctx, stubv1, stubv1alpha, opts)
}

// WithAllowMissingFileDescriptors returns an option that configures a client
// to allow missing files when building descriptors when possible. Missing files
// are often fatal errors, but with this option they can sometimes be worked
// around. Building a schema can only succeed with some files missing if the
// files in question only provide custom options and/or other unused types.
func WithAllowMissingFileDescriptors() ClientOption {
	return func(c *Client) {
		c.allowMissing = true
	}
}

// WithFallbackResolvers returns an option that configures the client to allow
// falling back to the given resolvers if the server is unable to supply
// descriptors for a particular query. This allows working around issues where
// servers' reflection service provides an incomplete set of descriptors, but
// the client has knowledge of the missing descriptors from another source. It
// is usually most appropriate to pass [protoregistry.GlobalFiles] and
// [protoregistry.GlobalTypes] as the resolver values.
//
// The first value is used as a fallback for FileByFilename and FileContainingSymbol
// queries. The second value is used as a fallback for FileContainingExtension. It
// can also be used as a fallback for AllExtensionNumbersForType if it provides
// a method with the following signature (which *[protoregistry.Types] provides):
//
//	RangeExtensionsByMessage(message protoreflect.FullName, f func(protoreflect.ExtensionType) bool)
func WithFallbackResolvers(descriptors protodesc.Resolver, exts protoregistry.ExtensionTypeResolver) ClientOption {
	return func(c *Client) {
		c.fallbackResolver = descriptors
		c.fallbackExtResolver = exts
	}
}

// WithFallbackResolver returns an option that configures the client to allow
// falling back to the given resolvers if the server is unable to supply
// descriptors for a particular query.
//
// This is the same as the plural version, WithFallbackResolvers, except that
// a single resolver instance can be used for both files and descriptors as
// well as extension types.
func WithFallbackResolver(res interface {
	protoresolve.DependencyResolver
	protoresolve.ExtensionResolver
}) ClientOption {
	var extRes protoregistry.ExtensionTypeResolver
	if asPool, ok := res.(interface{ AsTypePool() protoresolve.TypePool }); ok {
		extRes = asPool.AsTypePool()
	} else if asTypes, ok := res.(protoresolve.Resolver); ok {
		extRes = asTypes.AsTypeResolver()
	} else {
		extRes = protoresolve.TypesFromResolver(res)
	}
	return WithFallbackResolvers(res, extRes)
}

// FileByFilename asks the server for a file descriptor for the proto file with
// the given name.
func (cr *Client) FileByFilename(filename string) (protoreflect.FileDescriptor, error) {
	cr.cacheMu.RLock()
	// hit the cache first
	if fd, err := cr.descriptors.FindFileByPath(filename); err == nil {
		cr.cacheMu.RUnlock()
		return fd, nil
	}
	// not there? see if we've downloaded the proto
	fdp, ok := cr.protosByName[filename]
	cr.cacheMu.RUnlock()
	if ok {
		return cr.descriptorFromProto(fdp)
	}

	req := &refv1.ServerReflectionRequest{
		MessageRequest: &refv1.ServerReflectionRequest_FileByFilename{
			FileByFilename: filename,
		},
	}
	accept := func(fd protoreflect.FileDescriptor) bool {
		return fd.Path() == filename
	}

	fd, err := cr.getAndCacheFileDescriptors(req, accept)
	if isNotFound(err) && cr.fallbackResolver != nil {
		if fd, err := cr.fallbackResolver.FindFileByPath(filename); err == nil {
			return fd, nil
		}
	}
	if isNotFound(err) {
		err = fileNotFound(filename, nil)
	} else if e, ok := err.(*elementNotFoundError); ok {
		err = fileNotFound(filename, e)
	}
	return fd, err
}

// FileContainingSymbol asks the server for a file descriptor for the proto file
// that declares the given fully-qualified symbol.
func (cr *Client) FileContainingSymbol(symbol protoreflect.FullName) (protoreflect.FileDescriptor, error) {
	// hit the cache first
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindDescriptorByName(symbol)
	cr.cacheMu.RUnlock()
	if err == nil {
		return d.ParentFile(), nil
	}

	req := &refv1.ServerReflectionRequest{
		MessageRequest: &refv1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: string(symbol),
		},
	}
	accept := func(fd protoreflect.FileDescriptor) bool {
		return protoresolve.FindDescriptorByNameInFile(fd, symbol) != nil
	}
	fd, err := cr.getAndCacheFileDescriptors(req, accept)
	if isNotFound(err) && cr.fallbackResolver != nil {
		if d, err := cr.fallbackResolver.FindDescriptorByName(symbol); err == nil {
			return d.ParentFile(), nil
		}
	}
	if isNotFound(err) {
		err = symbolNotFound(symbol, nil)
	} else if e, ok := err.(*elementNotFoundError); ok {
		err = symbolNotFound(symbol, e)
	}
	return fd, err
}

// FileContainingExtension asks the server for a file descriptor for the proto
// file that declares an extension with the given number for the given
// fully-qualified message name.
func (cr *Client) FileContainingExtension(extendedMessageName protoreflect.FullName, extensionNumber protoreflect.FieldNumber) (protoreflect.FileDescriptor, error) {
	// hit the cache first
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindExtensionByNumber(extendedMessageName, extensionNumber)
	cr.cacheMu.RUnlock()
	if err == nil {
		return d.ParentFile(), nil
	}

	req := &refv1.ServerReflectionRequest{
		MessageRequest: &refv1.ServerReflectionRequest_FileContainingExtension{
			FileContainingExtension: &refv1.ExtensionRequest{
				ContainingType:  string(extendedMessageName),
				ExtensionNumber: int32(extensionNumber),
			},
		},
	}
	accept := func(fd protoreflect.FileDescriptor) bool {
		return protoresolve.FindExtensionByNumberInFile(fd, extendedMessageName, extensionNumber) != nil
	}
	fd, err := cr.getAndCacheFileDescriptors(req, accept)
	if isNotFound(err) && cr.fallbackExtResolver != nil {
		if xt, err := cr.fallbackExtResolver.FindExtensionByNumber(extendedMessageName, extensionNumber); err == nil {
			return xt.TypeDescriptor().ParentFile(), nil
		}
	}
	if isNotFound(err) {
		err = extensionNotFound(extendedMessageName, extensionNumber, nil)
	} else if e, ok := err.(*elementNotFoundError); ok {
		err = extensionNotFound(extendedMessageName, extensionNumber, e)
	}
	return fd, err
}

func (cr *Client) getAndCacheFileDescriptors(req *refv1.ServerReflectionRequest, accept func(protoreflect.FileDescriptor) bool) (protoreflect.FileDescriptor, error) {
	resp, err := cr.send(req)
	if err != nil {
		return nil, err
	}

	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		return nil, &ProtocolError{reflect.TypeOf(fdResp).Elem()}
	}

	// Response can contain the result file descriptor, but also its transitive
	// deps. Furthermore, protocol states that subsequent requests do not need
	// to send transitive deps that have been sent in prior responses. So we
	// need to cache all file descriptors and then return the first one (which
	// should be the answer). If we're looking for a file by name, we can be
	// smarter and make sure to grab one by name instead of just grabbing the
	// first one.
	var fds []*descriptorpb.FileDescriptorProto
	for _, fdBytes := range fdResp.FileDescriptorProto {
		fd := &descriptorpb.FileDescriptorProto{}
		if err = proto.Unmarshal(fdBytes, fd); err != nil {
			return nil, err
		}

		cr.cacheMu.Lock()
		// store in cache of raw descriptor protos, but don't overwrite existing protos
		if existingFd, ok := cr.protosByName[fd.GetName()]; ok {
			fd = existingFd
		} else {
			cr.protosByName[fd.GetName()] = fd
		}
		cr.cacheMu.Unlock()

		fds = append(fds, fd)
	}

	// find the right result from the files returned
	for _, fd := range fds {
		result, err := cr.descriptorFromProto(fd)
		if err != nil {
			return nil, err
		}
		if accept(result) {
			return result, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "response does not include expected file")
}

func (cr *Client) descriptorFromProto(fd *descriptorpb.FileDescriptorProto) (protoreflect.FileDescriptor, error) {
	var deferredErr error
	var missingDeps []int
	for i, depName := range fd.GetDependency() {
		if _, err := cr.FileByFilename(depName); err != nil {
			if _, ok := err.(*elementNotFoundError); !ok || !cr.allowMissing {
				return nil, err
			}
			// We'll ignore for now to see if the file is really necessary.
			// (If it only supplies custom options, we can get by without it.)
			if deferredErr == nil {
				deferredErr = err
			}
			missingDeps = append(missingDeps, i)
		}
	}
	if len(missingDeps) > 0 {
		fd = fileWithoutDeps(fd, missingDeps)
	}
	cr.cacheMu.Lock()
	defer cr.cacheMu.Unlock()
	if fd, err := cr.descriptors.FindFileByPath(fd.GetName()); err == nil {
		return fd, nil
	}
	d, err := protowrap.AddToRegistry(fd, &cr.descriptors)
	if err != nil {
		if deferredErr != nil {
			// assume the issue is the missing dep
			return nil, deferredErr
		}
		return nil, err
	}
	return d, nil
}

func fileWithoutDeps(fd *descriptorpb.FileDescriptorProto, missingDeps []int) *descriptorpb.FileDescriptorProto {
	// We need to rebuild the file without the missing deps.
	fd = proto.Clone(fd).(*descriptorpb.FileDescriptorProto)
	newNumDeps := len(fd.GetDependency()) - len(missingDeps)
	newDeps := make([]string, 0, newNumDeps)
	remapped := make(map[int]int, newNumDeps)
	missingIdx := 0
	for i, dep := range fd.GetDependency() {
		if missingIdx < len(missingDeps) {
			if i == missingDeps[missingIdx] {
				// This dep was missing. Skip it.
				missingIdx++
				continue
			}
		}
		remapped[i] = len(newDeps)
		newDeps = append(newDeps, dep)
	}
	// Also rebuild public and weak import slices.
	newPublic := make([]int32, 0, len(fd.GetPublicDependency()))
	for _, idx := range fd.GetPublicDependency() {
		newIdx, ok := remapped[int(idx)]
		if ok {
			newPublic = append(newPublic, int32(newIdx))
		}
	}
	newWeak := make([]int32, 0, len(fd.GetWeakDependency()))
	for _, idx := range fd.GetWeakDependency() {
		newIdx, ok := remapped[int(idx)]
		if ok {
			newWeak = append(newWeak, int32(newIdx))
		}
	}

	fd.Dependency = newDeps
	fd.PublicDependency = newPublic
	fd.WeakDependency = newWeak
	return fd
}

// AllExtensionNumbersForType asks the server for all known extension numbers
// for the given fully-qualified message name.
func (cr *Client) AllExtensionNumbersForType(extendedMessageName protoreflect.FullName) ([]protoreflect.FieldNumber, error) {
	req := &refv1.ServerReflectionRequest{
		MessageRequest: &refv1.ServerReflectionRequest_AllExtensionNumbersOfType{
			AllExtensionNumbersOfType: string(extendedMessageName),
		},
	}
	resp, err := cr.send(req)
	if err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	extResp := resp.GetAllExtensionNumbersResponse()
	if extResp == nil {
		return nil, &ProtocolError{reflect.TypeOf(extResp).Elem()}
	}
	nums := make([]protoreflect.FieldNumber, len(extResp.ExtensionNumber))
	for i := range extResp.ExtensionNumber {
		nums[i] = protoreflect.FieldNumber(extResp.ExtensionNumber[i])
	}
	if extRanger, ok := cr.fallbackExtResolver.(interface {
		RangeExtensionsByMessage(protoreflect.FullName, func(protoreflect.ExtensionType) bool)
	}); ok {
		numSet := make(map[protoreflect.FieldNumber]struct{}, len(nums))
		for _, num := range nums {
			numSet[num] = struct{}{}
		}
		extRanger.RangeExtensionsByMessage(extendedMessageName, func(xt protoreflect.ExtensionType) bool {
			fieldNum := xt.TypeDescriptor().Number()
			if _, exists := numSet[fieldNum]; exists {
				return true // already know about extension with this number
			}
			numSet[fieldNum] = struct{}{}
			nums = append(nums, fieldNum)
			return true
		})
	}
	return nums, nil
}

// ListServices asks the server for the fully-qualified names of all exposed
// services.
func (cr *Client) ListServices() ([]protoreflect.FullName, error) {
	req := &refv1.ServerReflectionRequest{
		MessageRequest: &refv1.ServerReflectionRequest_ListServices{
			// proto doesn't indicate any purpose for this value and server impl
			// doesn't actually use it...
			ListServices: "*",
		},
	}
	resp, err := cr.send(req)
	if err != nil {
		return nil, err
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		return nil, &ProtocolError{reflect.TypeOf(listResp).Elem()}
	}
	serviceNames := make([]protoreflect.FullName, len(listResp.Service))
	for i, s := range listResp.Service {
		serviceNames[i] = protoreflect.FullName(s.Name)
	}
	return serviceNames, nil
}

func (cr *Client) send(req *refv1.ServerReflectionRequest) (*refv1.ServerReflectionResponse, error) {
	// we allow one immediate retry, in case we have a stale stream
	// (e.g. closed by server)
	resp, err := cr.doSend(req)
	if err != nil {
		return nil, err
	}

	// convert error response messages into errors
	errResp := resp.GetErrorResponse()
	if errResp != nil {
		return nil, status.Errorf(codes.Code(errResp.ErrorCode), "%s", errResp.ErrorMessage)
	}

	return resp, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	s, ok := status.FromError(err)
	return ok && s.Code() == codes.NotFound
}

func (cr *Client) doSend(req *refv1.ServerReflectionRequest) (*refv1.ServerReflectionResponse, error) {
	// TODO: Streams are thread-safe, so we shouldn't need to lock. But without locking, we'll need more machinery
	// (goroutines and channels) to ensure that responses are correctly correlated with their requests and thus
	// delivered in correct oder.
	cr.connMu.Lock()
	defer cr.connMu.Unlock()
	return cr.doSendLocked(0, nil, req)
}

func (cr *Client) doSendLocked(attemptCount int, prevErr error, req *refv1.ServerReflectionRequest) (*refv1.ServerReflectionResponse, error) {
	if attemptCount >= 3 && prevErr != nil {
		return nil, prevErr
	}
	if (status.Code(prevErr) == codes.Unimplemented ||
		status.Code(prevErr) == codes.Unavailable) &&
		cr.useV1() {
		// If v1 is unimplemented, fallback to v1alpha.
		// We also fallback on unavailable because some servers have been
		// observed to close the connection/cancel the stream, w/out sending
		// back status or headers, when the service name is not known. When
		// this happens, the RPC status code is unavailable.
		// See https://github.com/fullstorydev/grpcurl/issues/434
		cr.useV1Alpha = true
		cr.lastTriedV1 = cr.now()
	}
	attemptCount++

	if err := cr.initStreamLocked(); err != nil {
		return nil, err
	}

	if err := cr.stream.Send(req); err != nil {
		if err == io.EOF {
			// if send returns EOF, must call Recv to get real underlying error
			_, err = cr.stream.Recv()
		}
		cr.resetLocked()
		return cr.doSendLocked(attemptCount, err, req)
	}

	resp, err := cr.stream.Recv()
	if err != nil {
		cr.resetLocked()
		return cr.doSendLocked(attemptCount, err, req)
	}
	return resp, nil
}

func (cr *Client) initStreamLocked() error {
	if cr.stream != nil {
		return nil
	}
	var newCtx context.Context
	newCtx, cr.cancel = context.WithCancel(cr.ctx)
	if cr.useV1Alpha && cr.now().Sub(cr.lastTriedV1) > durationBetweenV1Attempts {
		// we're due for periodic retry of v1
		cr.useV1Alpha = false
	}
	if cr.useV1() {
		// try the v1 API
		streamv1, err := cr.stubV1.ServerReflectionInfo(newCtx)
		if err == nil {
			cr.stream = streamv1
			return nil
		}
		if status.Code(err) != codes.Unimplemented {
			return err
		}
		// oh well, fall through below to try v1alpha and update state
		// so we skip straight to v1alpha next time
		cr.useV1Alpha = true
		cr.lastTriedV1 = cr.now()
	}
	var err error
	streamv1alpha, err := cr.stubV1Alpha.ServerReflectionInfo(newCtx)
	if err == nil {
		cr.stream = adaptStreamFromV1Alpha{streamv1alpha}
		return nil
	}
	return err
}

func (cr *Client) useV1() bool {
	return cr.stubV1Alpha == nil || (!cr.useV1Alpha && cr.stubV1 != nil)
}

// Reset ensures that any active stream with the server is closed, releasing any
// resources.
func (cr *Client) Reset() {
	cr.connMu.Lock()
	defer cr.connMu.Unlock()
	cr.resetLocked()
}

func (cr *Client) resetLocked() {
	if cr.stream != nil {
		_ = cr.stream.CloseSend()
		for {
			// drain the stream, this covers io.EOF too
			if _, err := cr.stream.Recv(); err != nil {
				break
			}
		}
		cr.stream = nil
	}
	if cr.cancel != nil {
		cr.cancel()
		cr.cancel = nil
	}
}

// AsResolver returns a protoresolve.Resolver that is backed by this client.
// Iteration via the various Range methods will only enumerate the snapshot of
// known elements at the time iteration starts. If more elements are discovered,
// via subsequent calls to the server to handle other queries, they will then be
// available to later iterations. That means that calls to NumFiles and
// NumFilesByPackage are not necessarily authoritative as the actual number
// could change concurrently.
func (cr *Client) AsResolver() protoresolve.Resolver {
	return (*clientResolver)(cr)
}

type clientResolver Client

func (c *clientResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	cr := (*Client)(c)
	return cr.FileByFilename(path)
}

func (c *clientResolver) NumFiles() int {
	cr := (*Client)(c)
	cr.cacheMu.RLock()
	n := c.descriptors.NumFiles()
	cr.cacheMu.RUnlock()
	return n
}

func (c *clientResolver) RangeFiles(fn func(protoreflect.FileDescriptor) bool) {
	cr := (*Client)(c)
	var files []protoreflect.FileDescriptor
	func() {
		cr.cacheMu.RLock()
		defer cr.cacheMu.RUnlock()
		cr.descriptors.RangeFiles(func(file protoreflect.FileDescriptor) bool {
			files = append(files, file)
			return true
		})
	}()
	for _, file := range files {
		if !fn(file) {
			return
		}
	}
}

func (c *clientResolver) NumFilesByPackage(name protoreflect.FullName) int {
	cr := (*Client)(c)
	cr.cacheMu.RLock()
	n := c.descriptors.NumFilesByPackage(name)
	cr.cacheMu.RUnlock()
	return n
}

func (c *clientResolver) RangeFilesByPackage(name protoreflect.FullName, fn func(protoreflect.FileDescriptor) bool) {
	cr := (*Client)(c)
	var files []protoreflect.FileDescriptor
	func() {
		cr.cacheMu.RLock()
		defer cr.cacheMu.RUnlock()
		cr.descriptors.RangeFilesByPackage(name, func(file protoreflect.FileDescriptor) bool {
			files = append(files, file)
			return true
		})
	}()
	for _, file := range files {
		if !fn(file) {
			return
		}
	}
}

func (c *clientResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindDescriptorByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindMessageByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindFieldByName(name protoreflect.FullName) (protoreflect.FieldDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindFieldByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindExtensionByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindOneofByName(name protoreflect.FullName) (protoreflect.OneofDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindOneofByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindEnumByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindEnumValueByName(name protoreflect.FullName) (protoreflect.EnumValueDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindEnumValueByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindServiceByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindMethodByName(name)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) FindMessageByURL(url string) (protoreflect.MessageDescriptor, error) {
	return c.FindMessageByName(protoresolve.TypeNameFromURL(url))
}

func (c *clientResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionDescriptor, error) {
	cr := (*Client)(c)
	_, err := cr.FileContainingExtension(message, field)
	if err != nil {
		return nil, err
	}
	cr.cacheMu.RLock()
	d, err := cr.descriptors.FindExtensionByNumber(message, field)
	cr.cacheMu.RUnlock()
	return d, err
}

func (c *clientResolver) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionDescriptor) bool) {
	cr := (*Client)(c)
	var exts []protoreflect.ExtensionDescriptor
	func() {
		cr.cacheMu.RLock()
		defer cr.cacheMu.RUnlock()
		cr.descriptors.RangeExtensionsByMessage(message, func(ext protoreflect.ExtensionDescriptor) bool {
			exts = append(exts, ext)
			return true
		})
	}()
	for _, ext := range exts {
		if !fn(ext) {
			return
		}
	}
}

func (c *clientResolver) AsTypeResolver() protoresolve.TypeResolver {
	return protoresolve.TypesFromResolver(c)
}

// depResolver is a view of the client's registries as a single resolver. It
// is backed by the client's descriptors field (which contains all items already
// downloaded and resolved) and its fallbackResolver field (which is optional
// and provided by the code that configured the client).
type depResolver Client

func (d *depResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	file, err := d.descriptors.FindFileByPath(path)
	if err == nil || d.fallbackResolver == nil {
		return file, err
	}
	file, fallbackErr := d.fallbackResolver.FindFileByPath(path)
	if fallbackErr == nil {
		return file, nil
	}
	return file, err
}

func (d *depResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	desc, err := d.descriptors.FindDescriptorByName(name)
	if err == nil || d.fallbackResolver == nil {
		return desc, err
	}
	desc, fallbackErr := d.fallbackResolver.FindDescriptorByName(name)
	if fallbackErr == nil {
		return desc, nil
	}
	return desc, err
}
