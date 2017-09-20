package msgregistry

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/genproto/protobuf/api"
	"google.golang.org/genproto/protobuf/ptype"
	"google.golang.org/genproto/protobuf/source_context"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
)

const googleApisDomain = "type.googleapis.com"

// MessageRegistry is a registry that maps URLs to message types. It allows for marshalling
// and unmarshalling Any types to and from dynamic messages.
type MessageRegistry struct {
	includeDefault bool
	resolver       typeResolver
	mf             *dynamic.MessageFactory
	er             *dynamic.ExtensionRegistry
	mu             sync.RWMutex
	types          map[string]desc.Descriptor
	baseUrls       map[string]string
	defaultBaseUrl string
}

// NewMessageRegistryWithDefaults is a registry that includes all "default" message types,
// which are those that are statically linked into the current program (e.g. registered by
// protoc-generated code via proto.RegisterType). Note that it cannot resolve "default" enum
// types since those don't actually get registered by protoc-generated code the same way.
// Any types explicitly added to the registry will override any default message types with
// the same URL.
func NewMessageRegistryWithDefaults() *MessageRegistry {
	mf := dynamic.NewMessageFactoryWithDefaults()
	return &MessageRegistry{
		includeDefault: true,
		mf:             mf,
		er:             mf.GetExtensionRegistry(),
	}
}

// WithFetcher sets the TypeFetcher that this registry uses to resolve unknown URLs. If no fetcher
// is configured for the registry then unknown URLs cannot be resolved. Known URLs are those for
// explicitly registered types and, if the registry includes "default" types, those for statically
// linked message types. This method is not thread-safe and is intended to be used for one-time
// initialization of the registry, before it is published for use by other threads.
func (r *MessageRegistry) WithFetcher(fetcher TypeFetcher) *MessageRegistry {
	r.resolver = typeResolver{fetcher: fetcher, mr: r}
	return r
}

// WithMessageFactory sets the MessageFactory used to instantiate any messages.
// This method is not thread-safe and is intended to be used for one-time
// initialization of the registry, before it is published for use by other threads.
func (r *MessageRegistry) WithMessageFactory(mf *dynamic.MessageFactory) *MessageRegistry {
	r.mf = mf
	if mf == nil {
		r.er = nil
	} else {
		r.er = mf.GetExtensionRegistry()
	}
	return r
}

// WithDefaultBaseUrl sets the default base URL used when constructing type URLs for
// marshalling messages as Any types and converting descriptors to well-known type
// descriptions (ptypes). If unspecified, the default base URL will be "type.googleapis.com".
// This method is not thread-safe and is intended to be used for one-time initialization
// of the registry, before it is published for use by other threads.
func (r *MessageRegistry) WithDefaultBaseUrl(baseUrl string) *MessageRegistry {
	baseUrl = stripTrailingSlash(baseUrl)
	r.defaultBaseUrl = baseUrl
	return r
}

func stripTrailingSlash(url string) string {
	if url[len(url)-1] == '/' {
		return url[:len(url)-1]
	}
	return url
}

// AddMessage adds the given URL and associated message descriptor to the registry.
func (r *MessageRegistry) AddMessage(url string, md *desc.MessageDescriptor) error {
	if !strings.HasSuffix(url, "/"+md.GetFullyQualifiedName()) {
		return fmt.Errorf("URL %s is invalid: it should end with path element %s", url, md.GetFullyQualifiedName())
	}
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.types == nil {
		r.types = map[string]desc.Descriptor{}
	}
	r.types[url] = md
	return nil
}

// AddEnum adds the given URL and associated enum descriptor to the registry.
func (r *MessageRegistry) AddEnum(url string, ed *desc.EnumDescriptor) error {
	if !strings.HasSuffix(url, "/"+ed.GetFullyQualifiedName()) {
		return fmt.Errorf("URL %s is invalid: it should end with path element %s", url, ed.GetFullyQualifiedName())
	}
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.types == nil {
		r.types = map[string]desc.Descriptor{}
	}
	r.types[url] = ed
	return nil
}

// AddFile adds to the registry all message and enum types in the given file. The URL for each type
// is derived using the given base URL as "baseURL/full.qualified.type.name".
func (r *MessageRegistry) AddFile(baseUrl string, fd *desc.FileDescriptor) {
	baseUrl = stripTrailingSlash(ensureScheme(baseUrl))
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.types == nil {
		r.types = map[string]desc.Descriptor{}
	}
	r.addEnumTypesLocked(baseUrl, fd.GetEnumTypes())
	r.addMessageTypesLocked(baseUrl, fd.GetMessageTypes())
}

func (r *MessageRegistry) addEnumTypesLocked(domain string, enums []*desc.EnumDescriptor) {
	for _, ed := range enums {
		r.types[fmt.Sprintf("%s/%s", domain, ed.GetFullyQualifiedName())] = ed
	}
}

func (r *MessageRegistry) addMessageTypesLocked(domain string, msgs []*desc.MessageDescriptor) {
	for _, md := range msgs {
		r.types[fmt.Sprintf("%s/%s", domain, md.GetFullyQualifiedName())] = md
		r.addEnumTypesLocked(domain, md.GetNestedEnumTypes())
		r.addMessageTypesLocked(domain, md.GetNestedMessageTypes())
	}
}

// FindMessageTypeByUrl finds a message descriptor for the type at the given URL. It may
// return nil if the registry is empty and cannot resolve unknown URLs. If an error occurs
// while resolving the URL, it is returned.
func (r *MessageRegistry) FindMessageTypeByUrl(url string) (*desc.MessageDescriptor, error) {
	md, err := r.getRegisteredMessageTypeByUrl(url)
	if err != nil {
		return nil, err
	} else if md != nil {
		return md, err
	}

	if r.resolver.fetcher == nil {
		return nil, nil
	}
	if md, err := r.resolver.resolveUrlToMessageDescriptor(url); err != nil {
		return nil, err
	} else {
		return md, nil
	}

}

func (r *MessageRegistry) getRegisteredMessageTypeByUrl(url string) (*desc.MessageDescriptor, error) {
	if r == nil {
		return nil, nil
	}
	r.mu.RLock()
	m := r.types[ensureScheme(url)]
	r.mu.RUnlock()
	if m != nil {
		if md, ok := m.(*desc.MessageDescriptor); ok {
			return md, nil
		} else {
			return nil, fmt.Errorf("Type for url %v is the wrong type: wanted message, got enum", url)
		}
	}
	if r.includeDefault {
		pos := strings.LastIndex(url, "/")
		var msgName string
		if pos >= 0 {
			msgName = url[pos+1:]
		} else {
			msgName = url
		}
		if md, err := desc.LoadMessageDescriptor(msgName); err != nil {
			return nil, err
		} else if md != nil {
			return md, nil
		}
	}
	return nil, nil
}

// FindEnumTypeByUrl finds an enum descriptor for the type at the given URL. It may return nil
// if the registry is empty and cannot resolve unknown URLs. If an error occurs while resolving
// the URL, it is returned.
func (r *MessageRegistry) FindEnumTypeByUrl(url string) (*desc.EnumDescriptor, error) {
	if r == nil {
		return nil, nil
	}
	r.mu.RLock()
	m := r.types[ensureScheme(url)]
	r.mu.RUnlock()
	if m != nil {
		if ed, ok := m.(*desc.EnumDescriptor); ok {
			return ed, nil
		} else {
			return nil, fmt.Errorf("Type for url %v is the wrong type: wanted enum, got message", url)
		}
	}
	if r.resolver.fetcher == nil {
		return nil, nil
	}
	if ed, err := r.resolver.resolveUrlToEnumDescriptor(url); err != nil {
		return nil, err
	} else {
		return ed, nil
	}
}

// ResolveApiIntoServiceDescriptor constructs a service descriptor that describes the given API.
// If any of the service's request or response type URLs cannot be resolved by this registry, a
// nil descriptor is returned.
func (r *MessageRegistry) ResolveApiIntoServiceDescriptor(a *api.Api) (*desc.ServiceDescriptor, error) {
	if r == nil {
		return nil, nil
	}

	msgs := map[string]*desc.MessageDescriptor{}
	unresolved := map[string]struct{}{}
	for _, m := range a.Methods {
		// request type
		md, err := r.getRegisteredMessageTypeByUrl(m.RequestTypeUrl)
		if err != nil {
			return nil, err
		} else if md == nil {
			if r.resolver.fetcher == nil {
				return nil, nil
			}
			unresolved[m.RequestTypeUrl] = struct{}{}
		} else {
			msgs[m.RequestTypeUrl] = md
		}
		// and response type
		md, err = r.getRegisteredMessageTypeByUrl(m.ResponseTypeUrl)
		if err != nil {
			return nil, err
		} else if md == nil {
			if r.resolver.fetcher == nil {
				return nil, nil
			}
			unresolved[m.ResponseTypeUrl] = struct{}{}
		} else {
			msgs[m.ResponseTypeUrl] = md
		}
	}

	if len(unresolved) > 0 {
		unresolvedSlice := make([]string, 0, len(unresolved))
		for k := range unresolved {
			unresolvedSlice = append(unresolvedSlice, k)
		}
		mp, err := r.resolver.resolveUrlsToMessageDescriptors(unresolvedSlice...)
		if err != nil {
			return nil, err
		}
		for u, md := range mp {
			msgs[u] = md
		}
	}

	var fileName string
	if a.SourceContext != nil && a.SourceContext.FileName != "" {
		fileName = a.SourceContext.FileName
	} else {
		fileName = "--unknown--.proto"
	}

	// now we add all types we care about to a typeTrie and use that to generate file descriptors
	files := map[string]*fileEntry{}
	fe := &fileEntry{}
	fe.proto3 = a.Syntax == ptype.Syntax_SYNTAX_PROTO3
	files[fileName] = fe
	fe.types.addType(a.Name, createServiceDescriptor(a, r))
	added := map[string]struct{}{}
	for _, md := range msgs {
		addDescriptors(fileName, files, md, msgs, added)
	}

	// build resulting file descriptor(s) and return the final service descriptor
	fileDescriptors, err := toFileDescriptors(files, (*typeTrie).rewriteDescriptor)
	if err != nil {
		return nil, err
	}
	return fileDescriptors[fileName].FindService(a.Name), nil
}

func addDescriptors(ref string, files map[string]*fileEntry, d desc.Descriptor, msgs map[string]*desc.MessageDescriptor, added map[string]struct{}) {
	name := d.GetFullyQualifiedName()

	fileName := d.GetFile().GetName()
	if fileName != ref {
		dependee := files[ref]
		if dependee.deps == nil {
			dependee.deps = map[string]struct{}{}
		}
		dependee.deps[fileName] = struct{}{}
	}

	if _, ok := added[name]; ok {
		// already added this one
		return
	}
	added[name] = struct{}{}

	fe := files[fileName]
	if fe == nil {
		fe = &fileEntry{}
		fe.proto3 = d.GetFile().IsProto3()
		files[fileName] = fe
	}
	fe.types.addType(name, d.AsProto())

	if md, ok := d.(*desc.MessageDescriptor); ok {
		for _, fld := range md.GetFields() {
			if fld.GetType() == descriptor.FieldDescriptorProto_TYPE_MESSAGE || fld.GetType() == descriptor.FieldDescriptorProto_TYPE_GROUP {
				// prefer descriptor in msgs map over what the field descriptor indicates
				md := msgs[fld.GetMessageType().GetFullyQualifiedName()]
				if md == nil {
					md = fld.GetMessageType()
				}
				addDescriptors(fileName, files, md, msgs, added)
			} else if fld.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
				addDescriptors(fileName, files, fld.GetEnumType(), msgs, added)
			}
		}
	}
}

// UnmarshalAny will unmarshal the value embedded in the given Any value. This will use this
// registry to resolve the given value's type URL. Use this instead of ptypes.UnmarshalAny for
// cases where the type might not be statically linked into the current program.
func (r *MessageRegistry) UnmarshalAny(any *any.Any) (proto.Message, error) {
	return r.unmarshalAny(any, r.FindMessageTypeByUrl)
}

func (r *MessageRegistry) unmarshalAny(any *any.Any, fetch func(string) (*desc.MessageDescriptor, error)) (proto.Message, error) {
	name, err := ptypes.AnyMessageName(any)
	if err != nil {
		return nil, err
	}

	var msg proto.Message
	if r == nil {
		// a nil registry only knows about well-known types
		if msg = (*dynamic.KnownTypeRegistry)(nil).CreateIfKnown(name); msg == nil {
			return nil, fmt.Errorf("Unknown message type: %s", any.TypeUrl)
		}
	} else {
		var ktr *dynamic.KnownTypeRegistry
		if r.mf != nil {
			ktr = r.mf.GetKnownTypeRegistry()
		}
		if msg = ktr.CreateIfKnown(name); msg == nil {
			if md, err := fetch(any.TypeUrl); err != nil {
				return nil, err
			} else if md == nil {
				return nil, fmt.Errorf("Unknown message type: %s", any.TypeUrl)
			} else {
				msg = r.mf.NewDynamicMessage(md)
			}
		}
	}

	err = proto.Unmarshal(any.Value, msg)
	if err != nil {
		return nil, err
	} else {
		return msg, nil
	}
}

// AddBaseUrlForElement adds a base URL for the given package or fully-qualified type name.
// This is used to construct type URLs for message types. If a given type has an associated
// base URL, it is used. Otherwise, the base URL for the type's package is used. If that is
// also absent, the registry's default base URL is used.
func (r *MessageRegistry) AddBaseUrlForElement(baseUrl, packageOrTypeName string) {
	if baseUrl[len(baseUrl)-1] == '/' {
		baseUrl = baseUrl[:len(baseUrl)-1]
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.baseUrls == nil {
		r.baseUrls = map[string]string{}
	}
	r.baseUrls[packageOrTypeName] = baseUrl
}

// MarshalAny wraps the given message in an Any value.
func (r *MessageRegistry) MarshalAny(m proto.Message) (*any.Any, error) {
	var md *desc.MessageDescriptor
	if dm, ok := m.(*dynamic.Message); ok {
		md = dm.GetMessageDescriptor()
	} else {
		var err error
		md, err = desc.LoadMessageDescriptorForMessage(m)
		if err != nil {
			return nil, err
		}
	}
	typeName := md.GetFullyQualifiedName()
	packageName := md.GetFile().GetPackage()

	if b, err := proto.Marshal(m); err != nil {
		return nil, err
	} else {
		return &any.Any{TypeUrl: r.asUrl(typeName, packageName), Value: b}, nil
	}
}

// MessageAsPType converts the given message descriptor into a ptype.Type. Registered
// base URLs are used to compute type URLs for any fields that have message or enum
// types.
func (r *MessageRegistry) MessageAsPType(md *desc.MessageDescriptor) *ptype.Type {
	fs := md.GetFields()
	fields := make([]*ptype.Field, len(fs))
	for i, f := range fs {
		fields[i] = r.fieldAsPType(f)
	}
	oos := md.GetOneOfs()
	oneOfs := make([]string, len(oos))
	for i, oo := range oos {
		oneOfs[i] = oo.GetName()
	}
	return &ptype.Type{
		Name:          md.GetFullyQualifiedName(),
		Fields:        fields,
		Oneofs:        oneOfs,
		Options:       r.options(md.GetOptions()),
		Syntax:        syntax(md.GetFile()),
		SourceContext: &source_context.SourceContext{FileName: md.GetFile().GetName()},
	}
}

func (r *MessageRegistry) fieldAsPType(fd *desc.FieldDescriptor) *ptype.Field {
	opts := r.options(fd.GetOptions())
	// remove the "packed" option as that is represented via separate field in ptype.Field
	for i, o := range opts {
		if o.Name == "packed" {
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	var oneOf int32
	if fd.AsFieldDescriptorProto().OneofIndex != nil {
		oneOf = fd.AsFieldDescriptorProto().GetOneofIndex() + 1
	}

	var card ptype.Field_Cardinality
	switch fd.GetLabel() {
	case descriptor.FieldDescriptorProto_LABEL_OPTIONAL:
		card = ptype.Field_CARDINALITY_OPTIONAL
	case descriptor.FieldDescriptorProto_LABEL_REPEATED:
		card = ptype.Field_CARDINALITY_REPEATED
	case descriptor.FieldDescriptorProto_LABEL_REQUIRED:
		card = ptype.Field_CARDINALITY_REQUIRED
	}

	var url string
	var kind ptype.Field_Kind
	switch fd.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		kind = ptype.Field_TYPE_ENUM
		url = r.asUrl(fd.GetEnumType().GetFullyQualifiedName(), fd.GetFile().GetPackage())
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		kind = ptype.Field_TYPE_GROUP
		url = r.asUrl(fd.GetMessageType().GetFullyQualifiedName(), fd.GetFile().GetPackage())
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		kind = ptype.Field_TYPE_MESSAGE
		url = r.asUrl(fd.GetMessageType().GetFullyQualifiedName(), fd.GetFile().GetPackage())
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		kind = ptype.Field_TYPE_BYTES
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		kind = ptype.Field_TYPE_STRING
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		kind = ptype.Field_TYPE_BOOL
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		kind = ptype.Field_TYPE_DOUBLE
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		kind = ptype.Field_TYPE_FLOAT
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		kind = ptype.Field_TYPE_FIXED32
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		kind = ptype.Field_TYPE_FIXED64
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		kind = ptype.Field_TYPE_INT32
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		kind = ptype.Field_TYPE_INT64
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		kind = ptype.Field_TYPE_SFIXED32
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		kind = ptype.Field_TYPE_SFIXED64
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		kind = ptype.Field_TYPE_SINT32
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		kind = ptype.Field_TYPE_SINT64
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		kind = ptype.Field_TYPE_UINT32
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		kind = ptype.Field_TYPE_UINT64
	}

	return &ptype.Field{
		Name:         fd.GetName(),
		Number:       fd.GetNumber(),
		JsonName:     fd.AsFieldDescriptorProto().GetJsonName(),
		OneofIndex:   oneOf,
		DefaultValue: fd.AsFieldDescriptorProto().GetDefaultValue(),
		Options:      opts,
		Packed:       fd.GetFieldOptions().GetPacked(),
		TypeUrl:      url,
		Cardinality:  card,
		Kind:         kind,
	}
}

// EnumAsPType converts the given enum descriptor into a ptype.Enum.
func (r *MessageRegistry) EnumAsPType(ed *desc.EnumDescriptor) *ptype.Enum {
	vs := ed.GetValues()
	vals := make([]*ptype.EnumValue, len(vs))
	for i, v := range vs {
		vals[i] = r.enumValueAsPType(v)
	}
	return &ptype.Enum{
		Name:          ed.GetFullyQualifiedName(),
		Enumvalue:     vals,
		Options:       r.options(ed.GetOptions()),
		Syntax:        syntax(ed.GetFile()),
		SourceContext: &source_context.SourceContext{FileName: ed.GetFile().GetName()},
	}
}

func (r *MessageRegistry) enumValueAsPType(vd *desc.EnumValueDescriptor) *ptype.EnumValue {
	return &ptype.EnumValue{
		Name:    vd.GetName(),
		Number:  vd.GetNumber(),
		Options: r.options(vd.GetOptions()),
	}
}

// ServiceAsApi converts the given service descriptor into a ptype API description.
func (r *MessageRegistry) ServiceAsApi(sd *desc.ServiceDescriptor) *api.Api {
	ms := sd.GetMethods()
	methods := make([]*api.Method, len(ms))
	for i, m := range ms {
		methods[i] = r.methodAsApi(m)
	}
	return &api.Api{
		Name:          sd.GetFullyQualifiedName(),
		Methods:       methods,
		Options:       r.options(sd.GetOptions()),
		Syntax:        syntax(sd.GetFile()),
		SourceContext: &source_context.SourceContext{FileName: sd.GetFile().GetName()},
	}
}

func (r *MessageRegistry) methodAsApi(md *desc.MethodDescriptor) *api.Method {
	return &api.Method{
		Name:              md.GetName(),
		RequestStreaming:  md.IsClientStreaming(),
		ResponseStreaming: md.IsServerStreaming(),
		RequestTypeUrl:    r.asUrl(md.GetInputType().GetFullyQualifiedName(), md.GetInputType().GetFile().GetPackage()),
		ResponseTypeUrl:   r.asUrl(md.GetOutputType().GetFullyQualifiedName(), md.GetOutputType().GetFile().GetPackage()),
		Options:           r.options(md.GetOptions()),
		Syntax:            syntax(md.GetFile()),
	}
}

func (r *MessageRegistry) options(options proto.Message) []*ptype.Option {
	rv := reflect.ValueOf(options)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	var opts []*ptype.Option
	for _, p := range proto.GetProperties(rv.Type()).Prop {
		o := r.option(p.OrigName, rv.FieldByName(p.Name))
		if o != nil {
			opts = append(opts, o...)
		}
	}
	for _, ext := range proto.RegisteredExtensions(options) {
		if proto.HasExtension(options, ext) {
			v, err := proto.GetExtension(options, ext)
			if err == nil && v != nil {
				o := r.option(ext.Name, reflect.ValueOf(v))
				if o != nil {
					opts = append(opts, o...)
				}
			}
		}
	}
	return opts
}

var typeOfBytes = reflect.TypeOf([]byte(nil))

func (r *MessageRegistry) option(name string, value reflect.Value) []*ptype.Option {
	if value.Kind() == reflect.Slice && value.Type() != typeOfBytes {
		// repeated field
		ret := make([]*ptype.Option, value.Len())
		for i := 0; i < value.Len(); i++ {
			ret[i] = r.singleOption(name, value.Index(i))
		}
		return ret
	} else {
		return []*ptype.Option{r.singleOption(name, value)}
	}
}

func (r *MessageRegistry) singleOption(name string, value reflect.Value) *ptype.Option {
	pm := wrap(value)
	if pm == nil {
		return nil
	}
	a, err := r.MarshalAny(pm)
	if err != nil {
		return nil
	}
	return &ptype.Option{
		Name:  name,
		Value: a,
	}
}

func wrap(v reflect.Value) proto.Message {
	if pm, ok := v.Interface().(proto.Message); ok {
		return pm
	}
	switch v.Kind() {
	case reflect.Bool:
		return &wrappers.BoolValue{Value: v.Bool()}
	case reflect.Slice:
		if v.Type() != typeOfBytes {
			return nil
		}
		return &wrappers.BytesValue{Value: v.Bytes()}
	case reflect.String:
		return &wrappers.StringValue{Value: v.String()}
	case reflect.Float32:
		return &wrappers.FloatValue{Value: float32(v.Float())}
	case reflect.Float64:
		return &wrappers.DoubleValue{Value: v.Float()}
	case reflect.Int32:
		return &wrappers.Int32Value{Value: int32(v.Int())}
	case reflect.Int64:
		return &wrappers.Int64Value{Value: v.Int()}
	case reflect.Uint32:
		return &wrappers.UInt32Value{Value: uint32(v.Uint())}
	case reflect.Uint64:
		return &wrappers.UInt64Value{Value: v.Uint()}
	default:
		return nil
	}
}

func syntax(fd *desc.FileDescriptor) ptype.Syntax {
	if fd.IsProto3() {
		return ptype.Syntax_SYNTAX_PROTO3
	} else {
		return ptype.Syntax_SYNTAX_PROTO2
	}
}

func (r *MessageRegistry) asUrl(name, pkgName string) string {
	r.mu.RLock()
	baseUrl := r.baseUrls[name]
	if baseUrl == "" {
		// lookup domain for the package
		baseUrl = r.baseUrls[pkgName]
	}
	r.mu.RUnlock()

	if baseUrl == "" {
		baseUrl = r.defaultBaseUrl
		if baseUrl == "" {
			baseUrl = googleApisDomain
		}
	}

	return fmt.Sprintf("%s/%s", baseUrl, name)
}

// Resolve resolves the given type URL into an instance of a message. This
// implements the jsonpb.AnyResolver interface, for use with marshaling and
// unmarshaling Any messages to/from JSON.
func (r *MessageRegistry) Resolve(typeUrl string) (proto.Message, error) {
	mr := (*MessageRegistry)(r)
	md, err := mr.FindMessageTypeByUrl(typeUrl)
	if err != nil {
		return nil, err
	}
	return mr.mf.NewMessage(md), nil
}

var _ jsonpb.AnyResolver = (*MessageRegistry)(nil)
