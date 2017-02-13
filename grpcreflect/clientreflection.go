package grpcreflect

import (
	"fmt"
	"reflect"
	"runtime"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"

	"golang.org/x/net/context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/jhump/protoreflect/desc"
)

// FileOrSymbolNotFound is the error returned by reflective operations where
// the server does not recognize a given file name, symbol name, or extension
// number.
const FileOrSymbolNotFound = notFoundError(0)

type notFoundError int32

// Error implements the error interface
func (_ notFoundError) Error() string {
	return "File or symbol not found"
}

// ProtocolError is an error returned when the server sends a response of the
// wrong type.
type ProtocolError struct {
	missingType reflect.Type
}

func (p ProtocolError) Error() string {
	return fmt.Sprintf("Protocol error: response was missing %v", p.missingType)
}

type extDesc struct {
	extendedMessageName string
	extensionNumber     int32
}

// ClientReflection is a client connection to a server for performing reflection
// calls and resolving remote symbols. ClientReflection is *not* thread-safe.
type ClientReflection struct {
	ctx  context.Context
	stub rpb.ServerReflectionClient

	cancel context.CancelFunc
	stream rpb.ServerReflection_ServerReflectionInfoClient

	protosByName     map[string]*dpb.FileDescriptorProto
	filesByName      map[string]*desc.FileDescriptor
	filesBySymbol    map[string]*desc.FileDescriptor
	filesByExtension map[extDesc]*desc.FileDescriptor
}

// NewClientReflection creates a new ClientReflection with the given root
// context and using the given RPC stub for talking to the server.
func NewClientReflection(ctx context.Context, stub rpb.ServerReflectionClient) *ClientReflection {
	cr := &ClientReflection{ctx: ctx, stub: stub}
	// don't leak a grpc stream
	runtime.SetFinalizer(cr, cr.Reset)
	return cr
}

// FileByFilename asks the server for a file descriptor for the proto file with
// the given name.
func (cr *ClientReflection) FileByFilename(filename string) (*desc.FileDescriptor, error) {
	// hit the cache first
	if fd, ok := cr.filesByName[filename]; ok {
		return fd, nil
	}

	// not there? see if we've downloaded the proto
	if fdp, ok := cr.protosByName[filename]; ok {
		return cr.descriptorFromProto(fdp)
	}

	req := &rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_FileByFilename{
			FileByFilename: filename,
		},
	}
	return cr.getAndCacheFileDescriptors(req)
}

// FileContainingSymbol asks the server for a file descriptor for the proto file
// that declares the given fully-qualified symbol.
func (cr *ClientReflection) FileContainingSymbol(symbol string) (*desc.FileDescriptor, error) {
	// hit the cache first
	if fd, ok := cr.filesBySymbol[symbol]; ok {
		return fd, nil
	}

	req := &rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: symbol,
		},
	}
	return cr.getAndCacheFileDescriptors(req)
}

// FileContainingExtension asks the server for a file descriptor for the proto
// file that declares an extension with the given number for the given
// fully-qualified message name.
func (cr *ClientReflection) FileContainingExtension(extendedMessageName string, extensionNumber int32) (*desc.FileDescriptor, error) {
	// hit the cache first
	if fd, ok := cr.filesByExtension[extDesc{extendedMessageName, extensionNumber}]; ok {
		return fd, nil
	}

	req := &rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_FileContainingExtension{
			FileContainingExtension: &rpb.ExtensionRequest{
				ContainingType:  extendedMessageName,
				ExtensionNumber: extensionNumber,
			},
		},
	}
	return cr.getAndCacheFileDescriptors(req)
}

func (cr *ClientReflection) getAndCacheFileDescriptors(req *rpb.ServerReflectionRequest) (*desc.FileDescriptor, error) {
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
	// should be the answer).
	var firstFd *dpb.FileDescriptorProto
	for _, fdBytes := range fdResp.FileDescriptorProto {
		fd := &dpb.FileDescriptorProto{}
		if err = proto.Unmarshal(fdBytes, fd); err != nil {
			return nil, err
		}
		cr.protosByName[fd.GetName()] = fd
		if firstFd == nil {
			firstFd = fd
		}
	}
	if firstFd == nil {
		return nil, &ProtocolError{reflect.TypeOf(firstFd).Elem()}
	}

	return cr.descriptorFromProto(firstFd)
}

func (cr *ClientReflection) descriptorFromProto(fd *dpb.FileDescriptorProto) (*desc.FileDescriptor, error) {
	deps := make([]*desc.FileDescriptor, len(fd.GetDependency()))
	for i, depName := range(fd.GetDependency()) {
		if dep, err := cr.FileByFilename(depName); err != nil {
			return nil, err
		} else {
			deps[i] = dep
		}
	}
	d, err := desc.CreateFileDescriptor(fd, deps)
	if err != nil {
		return nil, err
	}
	cr.cacheFile(d)
	return d, nil
}

func (cr *ClientReflection) cacheFile(fd *desc.FileDescriptor) {
	// cache file descriptor by name
	cr.filesByName[fd.GetName()] = fd

	// also cache by symbols and extensions
	for _, m := range(fd.GetMessageTypes()) {
		cr.cacheMessage(fd, m)
	}
	for _, e := range(fd.GetEnumTypes()) {
		cr.filesBySymbol[e.GetFullyQualifiedName()] = fd
		for _, v := range(e.GetValues()) {
			cr.filesBySymbol[v.GetFullyQualifiedName()] = fd
		}
	}
	for _, e := range(fd.GetExtensions()) {
		cr.filesBySymbol[e.GetFullyQualifiedName()] = fd
		cr.filesByExtension[extDesc{e.GetOwner().GetFullyQualifiedName(), e.GetNumber()}] = fd
	}
	for _, s := range(fd.GetServices()) {
		cr.filesBySymbol[s.GetFullyQualifiedName()] = fd
		for _, m := range(s.GetMethods()) {
			cr.filesBySymbol[m.GetFullyQualifiedName()] = fd
		}
	}
}

func (cr *ClientReflection) cacheMessage(fd *desc.FileDescriptor, md *desc.MessageDescriptor) {
	cr.filesBySymbol[md.GetFullyQualifiedName()] = fd
	for _, f := range(md.GetFields()) {
		cr.filesBySymbol[f.GetFullyQualifiedName()] = fd
	}
	for _, o := range(md.GetOneOfs()) {
		cr.filesBySymbol[o.GetFullyQualifiedName()] = fd
	}
	for _, e := range(md.GetNestedEnumTypes()) {
		cr.filesBySymbol[e.GetFullyQualifiedName()] = fd
		for _, v := range(e.GetValues()) {
			cr.filesBySymbol[v.GetFullyQualifiedName()] = fd
		}
	}
	for _, e := range(md.GetNestedExtensions()) {
		cr.filesBySymbol[e.GetFullyQualifiedName()] = fd
		cr.filesByExtension[extDesc{e.GetOwner().GetFullyQualifiedName(), e.GetNumber()}] = fd
	}
	for _, m := range(md.GetNestedMessageTypes()) {
		cr.cacheMessage(fd, m) // recurse
	}
}

// AllExtensionNumbersForType asks the server for all known extension numbers
// for the given fully-qualified message name.
func (cr *ClientReflection) AllExtensionNumbersForType(extendedMessageName string) ([]int32, error) {
	req := &rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_AllExtensionNumbersOfType{
			AllExtensionNumbersOfType: extendedMessageName,
		},
	}
	resp, err := cr.send(req)
	if err != nil {
		return nil, err
	}

	extResp := resp.GetAllExtensionNumbersResponse()
	if extResp == nil {
		return nil, &ProtocolError{reflect.TypeOf(extResp).Elem()}
	}
	return extResp.ExtensionNumber, nil
}

// ListServices asks the server for the fully-qualified names of all exposed
// services.
func (cr *ClientReflection) ListServices() ([]string, error) {
	req := &rpb.ServerReflectionRequest{
		MessageRequest: &rpb.ServerReflectionRequest_ListServices{
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
	serviceNames := make([]string, len(listResp.Service))
	for i, s := range listResp.Service {
		serviceNames[i] = s.Name
	}
	return serviceNames, nil
}

func (cr *ClientReflection) send(req *rpb.ServerReflectionRequest) (*rpb.ServerReflectionResponse, error) {
	// we allow one immediate retry, in case we have a stale stream
	// (e.g. closed by server)
	resp, err := cr.doSend(true, req)
	if err != nil {
		return nil, err
	}

	// convert error response messages into errors
	errResp := resp.GetErrorResponse()
	if errResp != nil {
		if errResp.ErrorCode == int32(codes.NotFound) {
			return nil, FileOrSymbolNotFound
		}
		return nil, grpc.Errorf(codes.Code(errResp.ErrorCode), "%s", errResp.ErrorMessage)
	}

	return resp, nil
}

func (cr *ClientReflection) doSend(retry bool, req *rpb.ServerReflectionRequest) (*rpb.ServerReflectionResponse, error) {
	if err := cr.initStream(); err != nil {
		return nil, err
	}

	if err := cr.stream.Send(req); err != nil {
		cr.Reset()
		if retry {
			return cr.doSend(false, req)
		}
		return nil, err
	}

	if resp, err := cr.stream.Recv(); err != nil {
		cr.Reset()
		if retry {
			return cr.doSend(false, req)
		}
		return nil, err
	} else {
		return resp, nil
	}
}

func (cr *ClientReflection) initStream() error {
	if cr.stream != nil {
		return nil
	}
	var newCtx context.Context
	newCtx, cr.cancel = context.WithCancel(cr.ctx)
	var err error
	cr.stream, err = cr.stub.ServerReflectionInfo(newCtx)
	return err
}

// Reset ensures that any active stream with the server is closed, releasing any
// resources.
func (cr *ClientReflection) Reset() {
	if cr.stream != nil {
		cr.stream.CloseSend()
		cr.stream = nil
	}
	if cr.cancel != nil {
		cr.cancel()
		cr.cancel = nil
	}
}

// ResolveService asks the server to resolve the given fully-qualified service
// name into a service descriptor.
func (cr *ClientReflection) ResolveService(serviceName string) (*desc.ServiceDescriptor, error) {
	file, err := cr.FileContainingSymbol(serviceName)
	if err != nil {
		return nil, err
	}
	d := file.FindSymbol(serviceName)
	if d == nil {
		return nil, FileOrSymbolNotFound
	}
	if s, ok := d.(*desc.ServiceDescriptor); ok {
		return s, nil
	} else {
		return nil, FileOrSymbolNotFound
	}
}

// ResolveMessage asks the server to resolve the given fully-qualified message
// name into a message descriptor.
func (cr *ClientReflection) ResolveMessage(messageName string) (*desc.MessageDescriptor, error) {
	file, err := cr.FileContainingSymbol(messageName)
	if err != nil {
		return nil, err
	}
	d := file.FindSymbol(messageName)
	if d == nil {
		return nil, FileOrSymbolNotFound
	}
	if s, ok := d.(*desc.MessageDescriptor); ok {
		return s, nil
	} else {
		return nil, FileOrSymbolNotFound
	}
}

// ResolveEnum asks the server to resolve the given fully-qualified enum name
// into an enum descriptor.
func (cr *ClientReflection) ResolveEnum(enumName string) (*desc.EnumDescriptor, error) {
	file, err := cr.FileContainingSymbol(enumName)
	if err != nil {
		return nil, err
	}
	d := file.FindSymbol(enumName)
	if d == nil {
		return nil, FileOrSymbolNotFound
	}
	if s, ok := d.(*desc.EnumDescriptor); ok {
		return s, nil
	} else {
		return nil, FileOrSymbolNotFound
	}
}

// ResolveEnumValues asks the server to resolve the given fully-qualified enum
// name into a map of names to numbers that represents the enum's values.
func (cr *ClientReflection) ResolveEnumValues(enumName string) (map[string]int32, error) {
	enumDesc, err := cr.ResolveEnum(enumName)
	if err != nil {
		return nil, err
	}
	vals := map[string]int32{}
	for _, valDesc := range enumDesc.GetValues() {
		vals[valDesc.GetName()] = valDesc.GetNumber()
	}
	return vals, nil
}

// ResolveExtension asks the server to resolve the given extension number and
// fully-qualified message name into a field descriptor.
func (cr *ClientReflection) ResolveExtension(extendedType string, extensionNumber int32) (*desc.FieldDescriptor, error) {
	file, err := cr.FileContainingExtension(extendedType, extensionNumber)
	if err != nil {
		return nil, err
	}
	d := findExtension(extendedType, extensionNumber, fileDescriptorExtensions{file})
	if d == nil {
		return nil, FileOrSymbolNotFound
	} else {
		return d, nil
	}
}

func findExtension(extendedType string, extensionNumber int32, scope extensionScope) *desc.FieldDescriptor {
	// search extensions in this scope
	for _, ext := range scope.extensions() {
		if ext.GetNumber() == extensionNumber && ext.GetOwner().GetFullyQualifiedName() == extendedType {
			return ext
		}
	}

	// if not found, search nested scopes
	for _, nested := range scope.nestedScopes() {
		ext := findExtension(extendedType, extensionNumber, nested)
		if ext != nil {
			return ext
		}
	}

	return nil
}

type extensionScope interface {
	extensions() []*desc.FieldDescriptor
	nestedScopes() []extensionScope
}

// fileDescriptorExtensions implements extensionHolder interface on top of
// FileDescriptorProto
type fileDescriptorExtensions struct {
	proto *desc.FileDescriptor
}

func (fde fileDescriptorExtensions) extensions() []*desc.FieldDescriptor {
	return fde.proto.GetExtensions()
}

func (fde fileDescriptorExtensions) nestedScopes() []extensionScope {
	scopes := make([]extensionScope, len(fde.proto.GetMessageTypes()))
	for i, m := range fde.proto.GetMessageTypes() {
		scopes[i] = msgDescriptorExtensions{m}
	}
	return scopes
}

// msgDescriptorExtensions implements extensionHolder interface on top of
// DescriptorProto
type msgDescriptorExtensions struct {
	proto *desc.MessageDescriptor
}

func (mde msgDescriptorExtensions) extensions() []*desc.FieldDescriptor {
	return mde.proto.GetNestedExtensions()
}

func (mde msgDescriptorExtensions) nestedScopes() []extensionScope {
	scopes := make([]extensionScope, len(mde.proto.GetNestedMessageTypes()))
	for i, m := range mde.proto.GetNestedMessageTypes() {
		scopes[i] = msgDescriptorExtensions{m}
	}
	return scopes
}
