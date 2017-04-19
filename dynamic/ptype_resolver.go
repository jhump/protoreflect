package dynamic

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/wrappers"
	"golang.org/x/net/context"
	"google.golang.org/genproto/protobuf/api"
	"google.golang.org/genproto/protobuf/ptype"

	"github.com/jhump/protoreflect/desc"
)

var (
	enumOptionsDesc, enumValueOptionsDesc *desc.MessageDescriptor
	msgOptionsDesc, fieldOptionsDesc      *desc.MessageDescriptor
	svcOptionsDesc, methodOptionsDesc     *desc.MessageDescriptor
)

func init() {
	var err error
	enumOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*descriptor.EnumOptions)(nil))
	if err != nil {
		panic("Failed to load descriptor for EnumOptions")
	}
	enumValueOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*descriptor.EnumValueOptions)(nil))
	if err != nil {
		panic("Failed to load descriptor for EnumValueOptions")
	}
	msgOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*descriptor.MessageOptions)(nil))
	if err != nil {
		panic("Failed to load descriptor for MessageOptions")
	}
	fieldOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*descriptor.FieldOptions)(nil))
	if err != nil {
		panic("Failed to load descriptor for FieldOptions")
	}
	svcOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*descriptor.ServiceOptions)(nil))
	if err != nil {
		panic("Failed to load descriptor for ServiceOptions")
	}
	methodOptionsDesc, err = desc.LoadMessageDescriptorForMessage((*descriptor.MethodOptions)(nil))
	if err != nil {
		panic("Failed to load descriptor for MethodOptions")
	}
}

// TypeFetcher is a simple operation that retrieves a type definition for a given type URL.
// The returned proto message will be either a *ptype.Enum or a *ptype.Type, depending on
// whether the enum flag is true or not.
type TypeFetcher func(url string, enum bool) (proto.Message, error)

// CachingTypeFetcher adds a caching layer to the given type fetcher. Queries for
// types that have already been fetched will not result in another call to the
// underlying fetcher and instead are retrieved from the cache.
func CachingTypeFetcher(fetcher TypeFetcher) TypeFetcher {
	c := protoCache{entries: map[string]*protoCacheEntry{}}
	return func(typeUrl string, enum bool) (proto.Message, error) {
		m, err := c.getOrLoad(typeUrl, func() (proto.Message, error) {
			return fetcher(typeUrl, enum)
		})
		if err != nil {
			return nil, err
		}
		if _, isEnum := m.(*ptype.Enum); enum != isEnum {
			var want, got string
			if enum {
				want = "enum"
				got = "message"
			} else {
				want = "message"
				got = "enum"
			}
			return nil, fmt.Errorf("Type for url %v is the wrong type: wanted %s, got %s", typeUrl, want, got)
		}
		return m.(proto.Message), nil
	}
}

type protoCache struct {
	mu      sync.RWMutex
	entries map[string]*protoCacheEntry
}

type protoCacheEntry struct {
	msg proto.Message
	err error
	wg  sync.WaitGroup
}

func (c *protoCache) getOrLoad(key string, loader func() (proto.Message, error)) (m proto.Message, err error) {
	// see if it's cached
	c.mu.RLock()
	cached, ok := c.entries[key]
	c.mu.RUnlock()
	if ok {
		cached.wg.Wait()
		return cached.msg, cached.err
	}

	// must delegate and cache the result
	c.mu.Lock()
	// double-check, in case it was added concurrently while we were upgrading lock
	cached, ok = c.entries[key]
	if ok {
		c.mu.Unlock()
		cached.wg.Wait()
		return cached.msg, cached.err
	}
	e := &protoCacheEntry{}
	e.wg.Add(1)
	c.entries[key] = e
	c.mu.Unlock()
	defer func() {
		if err != nil {
			// don't leave broken entry in the cache
			c.mu.Lock()
			delete(c.entries, key)
			c.mu.Unlock()
		}
		e.msg, e.err = m, err
		e.wg.Done()
	}()

	return loader()
}

func (c *protoCache) put(key string, m proto.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &protoCacheEntry{msg: m}
}

// HttpTypeFetcher returns a TypeFetcher that uses the given HTTP transport to query and
// download type definitions. The given szLimit is the maximum response size accepted. If
// used from multiple goroutines (like when a type's dependency graph is resolved in
// parallel), this resolver limits the number of parallel queries/downloads to the given
// parLimit.
func HttpTypeFetcher(transport http.RoundTripper, szLimit, parLimit int) TypeFetcher {
	sem := semaphore{count: parLimit, permits: parLimit}
	return CachingTypeFetcher(func(typeUrl string, enum bool) (proto.Message, error) {
		sem.Acquire()
		defer sem.Release()

		// build URL
		u, err := url.Parse(ensureScheme(typeUrl))
		if err != nil {
			return nil, err
		}

		resp, err := transport.RoundTrip(&http.Request{URL: u})
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.ContentLength > int64(szLimit) {
			return nil, fmt.Errorf("Type definition size %d is larger than limit of %d", resp.ContentLength, szLimit)
		}

		// download the response, up to the given size limit, into a buffer
		buf := bufferPool.Get().([]byte)
		defer bufferPool.Put(buf)
		var b bytes.Buffer
		for {
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if n > 0 {
				if b.Len()+n > szLimit {
					return nil, fmt.Errorf("Type definition size %d is larger than limit of %d", resp.ContentLength, szLimit)
				}
				b.Write(buf[:n])
			}
			if err == io.EOF {
				break
			}
		}

		// now we can de-serialize the type definition
		if enum {
			var ret ptype.Enum
			if err = proto.Unmarshal(b.Bytes(), &ret); err != nil {
				return nil, err
			}
			return &ret, nil
		} else {
			var ret ptype.Type
			if err = proto.Unmarshal(b.Bytes(), &ret); err != nil {
				return nil, err
			}
			return &ret, nil
		}
	})
}

func ensureScheme(url string) string {
	pos := strings.Index(url, "://")
	if pos < 0 {
		return "https://" + url
	}
	return url
}

var bufferPool = sync.Pool{New: func() interface{} {
	return make([]byte, 8192)
}}

type semaphore struct {
	lock    sync.Mutex
	count   int
	permits int
	cond    sync.Cond
}

func (s *semaphore) Acquire() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.cond.L == nil {
		s.cond.L = &s.lock
	}

	for s.count == 0 {
		s.cond.Wait()
	}
	s.count--
}

func (s *semaphore) Release() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.cond.L == nil {
		s.cond.L = &s.lock
	}

	if s.count == s.permits {
		panic("call to Release() without corresponding call to Acquire()")
	}
	s.count++
	s.cond.Signal()
}

type typeResolver struct {
	fetcher TypeFetcher
	mr      *MessageRegistry
	mu      sync.RWMutex
	cache   map[string]desc.Descriptor
}

func (r *typeResolver) resolveUrlToMessageDescriptor(url string) (md *desc.MessageDescriptor, err error) {
	r.mu.RLock()
	cached := r.cache[url]
	r.mu.RUnlock()
	if cached != nil {
		var ok bool
		if md, ok = cached.(*desc.MessageDescriptor); ok {
			return
		} else {
			err = fmt.Errorf("Type for url %v is the wrong type: wanted message, got enum", url)
			return
		}
	}

	rc := newResolutionContext()
	if err = rc.addType(url, r.fetcher, false); err != nil {
		return
	}

	var files map[string]*desc.FileDescriptor
	files, err = rc.toFileDescriptors(r.mr)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for typeUrl, fileName := range rc.typeLocations {
		fd := files[fileName]
		sym := fd.FindSymbol(typeName(typeUrl))
		r.cache[typeUrl] = sym
		if url == typeUrl {
			md = sym.(*desc.MessageDescriptor)
		}
	}
	return
}

func (r *typeResolver) resolveUrlToEnumDescriptor(url string) (ed *desc.EnumDescriptor, err error) {
	r.mu.RLock()
	cached := r.cache[url]
	r.mu.RUnlock()
	if cached != nil {
		var ok bool
		if ed, ok = cached.(*desc.EnumDescriptor); ok {
			return
		} else {
			err = fmt.Errorf("Type for url %v is the wrong type: wanted enum, got message", url)
			return
		}
	}

	rc := newResolutionContext()
	if err = rc.addType(url, r.fetcher, true); err != nil {
		return
	}

	var files map[string]*desc.FileDescriptor
	files, err = rc.toFileDescriptors(r.mr)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for typeUrl, fileName := range rc.typeLocations {
		fd := files[fileName]
		sym := fd.FindSymbol(typeName(typeUrl))
		r.cache[typeUrl] = sym
		if url == typeUrl {
			ed = sym.(*desc.EnumDescriptor)
		}
	}
	return
}

func (r *typeResolver) resolveApiToServiceDescriptor(a *api.Api) (*desc.ServiceDescriptor, error) {
	rc := newResolutionContext()

	serviceName := base(a.Name)
	var packageName string
	if serviceName == a.Name {
		packageName = ""
	} else {
		packageName = a.Name[:len(a.Name)-len(serviceName)-1]
	}

	var fileName string
	if a.SourceContext != nil && a.SourceContext.FileName != "" {
		fileName = a.SourceContext.FileName
	} else {
		fileName = "--unknown--.proto"
	}
	rc.files[fileName] = &fileEntry{pkgHint: packageName, svc: a, svcName: serviceName}

	for _, m := range a.Methods {
		if err := rc.addType(m.RequestTypeUrl, r.fetcher, false); err != nil {
			return nil, err
		}
		if err := rc.addType(m.ResponseTypeUrl, r.fetcher, false); err != nil {
			return nil, err
		}
	}

	files, err := rc.toFileDescriptors(r.mr)
	if err != nil {
		return nil, err
	}
	fd := files[fileName]
	return fd.FindService(a.Name), nil
}

type resolutionContext struct {
	ctx           context.Context
	cancel        func()
	mu            sync.Mutex
	files         map[string]*fileEntry
	typeLocations map[string]string
	unknownCount  int
}

func newResolutionContext() *resolutionContext {
	ctx, cancel := context.WithCancel(context.Background())
	return &resolutionContext{
		ctx:           ctx,
		cancel:        cancel,
		typeLocations: map[string]string{},
		files:         map[string]*fileEntry{},
	}
}

func (rc *resolutionContext) addType(url string, fetcher TypeFetcher, enum bool) error {
	if err := rc.ctx.Err(); err != nil {
		return err
	}
	m, err := fetcher(url, enum)
	if err != nil {
		return err
	}

	if enum {
		e := m.(*ptype.Enum)

		rc.mu.Lock()
		defer rc.mu.Unlock()

		var fileName string
		if e.SourceContext != nil && e.SourceContext.FileName != "" {
			fileName = e.SourceContext.FileName
		} else {
			fileName = fmt.Sprintf("--unknown--%d.proto", rc.unknownCount)
			rc.unknownCount++
		}
		rc.typeLocations[url] = fileName

		fe := rc.files[fileName]
		if fe == nil {
			fe = &fileEntry{}
			rc.files[fileName] = fe
		}
		if fe.pkgHint != "" && !strings.HasPrefix(e.Name, fe.pkgHint+".") {
			return fmt.Errorf("File %q should have package %s and is supposed to include incompatible element %s", fileName, fe.pkgHint, e.Name)
		}
		fe.types.addType(e.Name, e)
		if e.Syntax == ptype.Syntax_SYNTAX_PROTO3 {
			fe.proto3 = true
		}
		return nil
	}

	// for messages, resolve dependencies in parallel
	t := m.(*ptype.Type)
	var wg sync.WaitGroup
	var failed int32
	for _, f := range t.Fields {
		if f.Kind == ptype.Field_TYPE_GROUP || f.Kind == ptype.Field_TYPE_MESSAGE || f.Kind == ptype.Field_TYPE_ENUM {
			typeUrl := ensureScheme(f.TypeUrl)
			kind := f.Kind
			wg.Add(1)
			go func() {
				defer wg.Done()
				innerErr := rc.addType(typeUrl, fetcher, kind == ptype.Field_TYPE_ENUM)
				// We want the "real" error to ultimately propagate to root, not
				// one of the resulting cancellations (from any concurrent goroutines
				// working in the same resolution context).
				if innerErr != nil && (rc.ctx.Err() == nil || innerErr != context.Canceled) {
					if atomic.CompareAndSwapInt32(&failed, 0, 1) {
						err = innerErr
					}
					rc.cancel()
				}
			}()
		}
	}
	wg.Wait()
	if err != nil {
		return err
	}
	// double-check if context has been cancelled
	if err = rc.ctx.Err(); err != nil {
		return err
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	var fileName string
	if t.SourceContext != nil && t.SourceContext.FileName != "" {
		fileName = t.SourceContext.FileName
	} else {
		fileName = fmt.Sprintf("--unknown--%d.proto", rc.unknownCount)
		rc.unknownCount++
	}
	rc.typeLocations[url] = fileName

	fe := rc.files[fileName]
	if fe == nil {
		fe = &fileEntry{}
		rc.files[fileName] = fe
	}
	fe.types.addType(t.Name, t)
	if t.Syntax == ptype.Syntax_SYNTAX_PROTO3 {
		fe.proto3 = true
	}

	for _, f := range t.Fields {
		if f.Kind == ptype.Field_TYPE_GROUP || f.Kind == ptype.Field_TYPE_MESSAGE || f.Kind == ptype.Field_TYPE_ENUM {
			typeUrl := ensureScheme(f.TypeUrl)
			if fe.deps == nil {
				fe.deps = map[string]struct{}{}
			}
			dep := rc.typeLocations[typeUrl]
			if dep != fileName {
				fe.deps[dep] = struct{}{}
			}
		}
	}
	return nil
}

func (rc *resolutionContext) toFileDescriptors(mr *MessageRegistry) (map[string]*desc.FileDescriptor, error) {
	fdps := map[string]*descriptor.FileDescriptorProto{}
	for name, file := range rc.files {
		fdp := file.toFileDescriptor(name, mr)
		if file.svc != nil {
			fdp.Service = append(fdp.Service, createServiceDescriptor(file.svc, mr))
		}
		fdps[name] = fdp
	}
	fds := map[string]*desc.FileDescriptor{}
	for name, fdp := range fdps {
		if _, ok := fds[name]; ok {
			continue
		}
		var err error
		if fds[name], err = makeFileDesc(fdp, fds, fdps); err != nil {
			return nil, err
		}
	}
	return fds, nil
}

func makeFileDesc(fdp *descriptor.FileDescriptorProto, fds map[string]*desc.FileDescriptor, fdps map[string]*descriptor.FileDescriptorProto) (*desc.FileDescriptor, error) {
	deps := make([]*desc.FileDescriptor, len(fdp.Dependency))
	for i, dep := range fdp.Dependency {
		d := fds[dep]
		if d == nil {
			var err error
			d, err = makeFileDesc(fdps[dep], fds, fdps)
			if err != nil {
				return nil, err
			}
		}
		deps[i] = d
	}
	if fd, err := desc.CreateFileDescriptor(fdp, deps...); err != nil {
		return nil, err
	} else {
		fds[fdp.GetName()] = fd
		return fd, nil
	}
}

type fileEntry struct {
	types   typeTrie
	svc     *api.Api
	svcName string
	pkgHint string
	deps    map[string]struct{}
	proto3  bool
}

func (fe fileEntry) toFileDescriptor(name string, mr *MessageRegistry) *descriptor.FileDescriptorProto {
	var pkg bytes.Buffer
	tt := &fe.types
	first := true
	last := ""
	for tt.typ == nil {
		if last != "" {
			if first {
				first = false
			} else {
				pkg.WriteByte('.')
			}
			pkg.WriteString(last)
		}
		if len(tt.children) != 1 || pkg.Len() == len(fe.pkgHint) {
			break
		}
		for last, tt = range tt.children {
		}
	}
	fd := createFileDescriptor(name, pkg.String(), fe.proto3, fe.deps)
	if tt.typ != nil {
		msg, enum := tt.toDescriptor(last, mr)
		if msg != nil {
			fd.MessageType = append(fd.MessageType, msg)
		}
		if enum != nil {
			fd.EnumType = append(fd.EnumType, enum)
		}
	} else {
		for name, nested := range tt.children {
			msg, enum := nested.toDescriptor(name, mr)
			if msg != nil {
				fd.MessageType = append(fd.MessageType, msg)
			}
			if enum != nil {
				fd.EnumType = append(fd.EnumType, enum)
			}
		}
	}
	return fd
}

type typeTrie struct {
	children map[string]*typeTrie
	typ      proto.Message
}

func (t *typeTrie) addType(key string, typ proto.Message) {
	if key == "" {
		t.typ = typ
		return
	}
	if t.children == nil {
		t.children = map[string]*typeTrie{}
	}
	curr, rest := split(key)
	child := t.children[curr]
	if child == nil {
		child = &typeTrie{}
		t.children[curr] = child
	}
	child.addType(rest, typ)
}

func (t *typeTrie) toDescriptor(name string, mr *MessageRegistry) (*descriptor.DescriptorProto, *descriptor.EnumDescriptorProto) {
	if en, ok := t.typ.(*ptype.Enum); ok {
		return nil, createEnumDescriptor(en, mr)
	} else {
		var msg *descriptor.DescriptorProto
		if t.typ == nil {
			msg = createIntermediateMessageDescriptor(name)
		} else {
			msg = createMessageDescriptor(t.typ.(*ptype.Type), mr)
		}
		for name, nested := range t.children {
			msg, enum := nested.toDescriptor(name, mr)
			if msg != nil {
				msg.NestedType = append(msg.NestedType, msg)
			}
			if enum != nil {
				msg.EnumType = append(msg.EnumType, enum)
			}
		}
		return msg, nil
	}
}

func split(s string) (string, string) {
	pos := strings.Index(s, ".")
	if pos >= 0 {
		return s[:pos], s[pos+1:]
	} else {
		return s, ""
	}
}

func createEnumDescriptor(e *ptype.Enum, mr *MessageRegistry) *descriptor.EnumDescriptorProto {
	var opts *descriptor.EnumOptions
	if len(e.Options) > 0 {
		dopts := createOptions(e.Options, enumOptionsDesc, mr)
		dopts.ConvertTo(opts)
	}

	var vals []*descriptor.EnumValueDescriptorProto
	for _, v := range e.Enumvalue {
		evd := createEnumValueDescriptor(v, mr)
		vals = append(vals, evd)
	}

	return &descriptor.EnumDescriptorProto{
		Name:    proto.String(base(e.Name)),
		Options: opts,
		Value:   vals,
	}
}

func createEnumValueDescriptor(v *ptype.EnumValue, mr *MessageRegistry) *descriptor.EnumValueDescriptorProto {
	var opts *descriptor.EnumValueOptions
	if len(v.Options) > 0 {
		dopts := createOptions(v.Options, enumValueOptionsDesc, mr)
		dopts.ConvertTo(opts)
	}

	return &descriptor.EnumValueDescriptorProto{
		Name:    proto.String(v.Name),
		Number:  proto.Int32(v.Number),
		Options: opts,
	}
}

func createMessageDescriptor(m *ptype.Type, mr *MessageRegistry) *descriptor.DescriptorProto {
	var opts *descriptor.MessageOptions
	if len(m.Options) > 0 {
		dopts := createOptions(m.Options, msgOptionsDesc, mr)
		dopts.ConvertTo(opts) // ignore any error
	}

	var fields []*descriptor.FieldDescriptorProto
	for _, f := range m.Fields {
		fields = append(fields, createFieldDescriptor(f, mr))
	}

	var oneOfs []*descriptor.OneofDescriptorProto
	for _, o := range m.Oneofs {
		oneOfs = append(oneOfs, &descriptor.OneofDescriptorProto{
			Name: proto.String(o),
		})
	}

	return &descriptor.DescriptorProto{
		Name:      proto.String(base(m.Name)),
		Options:   opts,
		Field:     fields,
		OneofDecl: oneOfs,
	}
}

func createFieldDescriptor(f *ptype.Field, mr *MessageRegistry) *descriptor.FieldDescriptorProto {
	var opts *descriptor.FieldOptions
	if len(f.Options) > 0 {
		dopts := createOptions(f.Options, fieldOptionsDesc, mr)
		dopts.ConvertTo(opts) // ignore any error
	}
	if f.Packed {
		if opts == nil {
			opts = &descriptor.FieldOptions{Packed: proto.Bool(true)}
		} else {
			opts.Packed = proto.Bool(true)
		}
	}

	var typeName string
	if f.Kind == ptype.Field_TYPE_GROUP || f.Kind == ptype.Field_TYPE_MESSAGE || f.Kind == ptype.Field_TYPE_ENUM {
		pos := strings.LastIndex(f.TypeUrl, "/")
		typeName = "." + f.TypeUrl[pos+1:]
	}

	var label descriptor.FieldDescriptorProto_Label
	switch f.Cardinality {
	case ptype.Field_CARDINALITY_OPTIONAL:
		label = descriptor.FieldDescriptorProto_LABEL_OPTIONAL
	case ptype.Field_CARDINALITY_REPEATED:
		label = descriptor.FieldDescriptorProto_LABEL_REPEATED
	case ptype.Field_CARDINALITY_REQUIRED:
		label = descriptor.FieldDescriptorProto_LABEL_REQUIRED
	}

	var typ descriptor.FieldDescriptorProto_Type
	switch f.Kind {
	case ptype.Field_TYPE_ENUM:
		typ = descriptor.FieldDescriptorProto_TYPE_ENUM
	case ptype.Field_TYPE_GROUP:
		typ = descriptor.FieldDescriptorProto_TYPE_GROUP
	case ptype.Field_TYPE_MESSAGE:
		typ = descriptor.FieldDescriptorProto_TYPE_MESSAGE
	case ptype.Field_TYPE_BYTES:
		typ = descriptor.FieldDescriptorProto_TYPE_BYTES
	case ptype.Field_TYPE_STRING:
		typ = descriptor.FieldDescriptorProto_TYPE_STRING
	case ptype.Field_TYPE_BOOL:
		typ = descriptor.FieldDescriptorProto_TYPE_BOOL
	case ptype.Field_TYPE_DOUBLE:
		typ = descriptor.FieldDescriptorProto_TYPE_DOUBLE
	case ptype.Field_TYPE_FLOAT:
		typ = descriptor.FieldDescriptorProto_TYPE_FLOAT
	case ptype.Field_TYPE_FIXED32:
		typ = descriptor.FieldDescriptorProto_TYPE_FIXED32
	case ptype.Field_TYPE_FIXED64:
		typ = descriptor.FieldDescriptorProto_TYPE_FIXED64
	case ptype.Field_TYPE_INT32:
		typ = descriptor.FieldDescriptorProto_TYPE_INT32
	case ptype.Field_TYPE_INT64:
		typ = descriptor.FieldDescriptorProto_TYPE_INT64
	case ptype.Field_TYPE_SFIXED32:
		typ = descriptor.FieldDescriptorProto_TYPE_SFIXED32
	case ptype.Field_TYPE_SFIXED64:
		typ = descriptor.FieldDescriptorProto_TYPE_SFIXED64
	case ptype.Field_TYPE_SINT32:
		typ = descriptor.FieldDescriptorProto_TYPE_SINT32
	case ptype.Field_TYPE_SINT64:
		typ = descriptor.FieldDescriptorProto_TYPE_SINT64
	case ptype.Field_TYPE_UINT32:
		typ = descriptor.FieldDescriptorProto_TYPE_UINT32
	case ptype.Field_TYPE_UINT64:
		typ = descriptor.FieldDescriptorProto_TYPE_UINT64
	}

	return &descriptor.FieldDescriptorProto{
		Name:         proto.String(f.Name),
		Number:       proto.Int32(f.Number),
		DefaultValue: proto.String(f.DefaultValue),
		JsonName:     proto.String(f.JsonName),
		OneofIndex:   proto.Int32(f.OneofIndex),
		TypeName:     proto.String(typeName),
		Label:        label.Enum(),
		Type:         typ.Enum(),
	}
}

func createServiceDescriptor(a *api.Api, mr *MessageRegistry) *descriptor.ServiceDescriptorProto {
	var opts *descriptor.ServiceOptions
	if len(a.Options) > 0 {
		dopts := createOptions(a.Options, svcOptionsDesc, mr)
		dopts.ConvertTo(opts) // ignore any error
	}

	methods := make([]*descriptor.MethodDescriptorProto, len(a.Methods))
	for i, m := range a.Methods {
		methods[i] = createMethodDescriptor(m, mr)
	}

	return &descriptor.ServiceDescriptorProto{
		Name:    proto.String(base(a.Name)),
		Method:  methods,
		Options: opts,
	}
}

func createMethodDescriptor(m *api.Method, mr *MessageRegistry) *descriptor.MethodDescriptorProto {
	var opts *descriptor.MethodOptions
	if len(m.Options) > 0 {
		dopts := createOptions(m.Options, methodOptionsDesc, mr)
		dopts.ConvertTo(opts) // ignore any error
	}

	var reqType, respType string
	pos := strings.LastIndex(m.RequestTypeUrl, "/")
	reqType = "." + m.RequestTypeUrl[pos+1:]
	pos = strings.LastIndex(m.ResponseTypeUrl, "/")
	respType = "." + m.ResponseTypeUrl[pos+1:]

	return &descriptor.MethodDescriptorProto{
		Name:            proto.String(m.Name),
		Options:         opts,
		ClientStreaming: proto.Bool(m.RequestStreaming),
		ServerStreaming: proto.Bool(m.ResponseStreaming),
		InputType:       proto.String(reqType),
		OutputType:      proto.String(respType),
	}
}

func createIntermediateMessageDescriptor(name string) *descriptor.DescriptorProto {
	return &descriptor.DescriptorProto{
		Name: proto.String(name),
	}
}

func createFileDescriptor(name, pkg string, proto3 bool, deps map[string]struct{}) *descriptor.FileDescriptorProto {
	imports := make([]string, 0, len(deps))
	for k := range deps {
		imports = append(imports, k)
	}
	sort.Strings(imports)
	var syntax string
	if proto3 {
		syntax = "proto3"
	} else {
		syntax = "proto2"
	}
	return &descriptor.FileDescriptorProto{
		Name:       proto.String(name),
		Package:    proto.String(pkg),
		Syntax:     proto.String(syntax),
		Dependency: imports,
	}
}

func createOptions(options []*ptype.Option, optionsDesc *desc.MessageDescriptor, mr *MessageRegistry) *Message {
	// these are created "best effort" so entries which are unresolvable
	// (or seemingly invalid) are simply ignored...
	dopts := newMessageWithMessageFactory(optionsDesc, mr.mf)
	for _, o := range options {
		field := optionsDesc.FindFieldByName(o.Name)
		if field == nil {
			field = mr.er.FindExtensionByName(optionsDesc.GetFullyQualifiedName(), o.Name)
			if field == nil && o.Name[0] != '[' {
				field = mr.er.FindExtensionByName(optionsDesc.GetFullyQualifiedName(), fmt.Sprintf("[%s]", o.Name))
			}
			if field == nil {
				// can't resolve option name? skip it
				continue
			}
		}
		v, err := mr.unmarshalAny(o.Value, func(url string) (*desc.MessageDescriptor, error) {
			// we don't want to try to recursively fetch this value's type, so if it doesn't
			// match the type of the extension field, we'll skip it
			if (field.GetType() == descriptor.FieldDescriptorProto_TYPE_GROUP ||
				field.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE) &&
				typeName(url) == field.GetMessageType().GetFullyQualifiedName() {

				return field.GetMessageType(), nil
			}
			return nil, nil
		})
		if err != nil {
			// can't interpret value? skip it
			continue
		}
		var fv interface{}
		if field.GetType() != descriptor.FieldDescriptorProto_TYPE_MESSAGE && field.GetType() != descriptor.FieldDescriptorProto_TYPE_GROUP {
			fv = unwrap(v)
			if v == nil {
				// non-wrapper type for scalar field? skip it
				continue
			}
		} else {
			fv = v
		}
		dopts.setField(field, fv) // ignore any error
	}
	return dopts
}

func base(name string) string {
	pos := strings.LastIndex(name, ".")
	if pos >= 0 {
		return name[pos+1:]
	}
	return name
}

func unwrap(msg proto.Message) interface{} {
	switch m := msg.(type) {
	case (*wrappers.BoolValue):
		return m.Value
	case (*wrappers.FloatValue):
		return m.Value
	case (*wrappers.DoubleValue):
		return m.Value
	case (*wrappers.Int32Value):
		return m.Value
	case (*wrappers.Int64Value):
		return m.Value
	case (*wrappers.UInt32Value):
		return m.Value
	case (*wrappers.UInt64Value):
		return m.Value
	case (*wrappers.BytesValue):
		return m.Value
	case (*wrappers.StringValue):
		return m.Value
	default:
		return nil
	}
}

func typeName(url string) string {
	return url[strings.LastIndex(url, "/")+1:]
}
