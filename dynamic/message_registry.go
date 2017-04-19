package dynamic

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/genproto/protobuf/api"
	"google.golang.org/genproto/protobuf/ptype"
	"google.golang.org/genproto/protobuf/source_context"

	"github.com/jhump/protoreflect/desc"
)

const googleApisDomain = "type.googleapis.com"

// MessageRegistry is a registry that maps URLs to message types. It allows for marsalling
// and unmarshalling Any types to and from dynamic messages.
type MessageRegistry struct {
	includeDefault bool
	resolver       typeResolver
	mf             *MessageFactory
	er             *ExtensionRegistry
	mu             sync.RWMutex
	messages       map[string]desc.Descriptor
	domains        map[string]string
	defaultDomain  string
}

func NewMessageRegistryWithDefaults() *MessageRegistry {
	mf := NewMessageFactoryWithDefaults()
	return &MessageRegistry{
		includeDefault: true,
		mf:             mf,
		er:             mf.er,
	}
}

// WithFetcher sets the TypeFetcher that this registry uses to resolve unknown
// URLs. This method is not thread-safe and is intended to be used for one-time
// initialization of the registry, before it published for use by other threads.
func (r *MessageRegistry) WithFetcher(fetcher TypeFetcher) *MessageRegistry {
	r.resolver = typeResolver{fetcher: fetcher, mr: r}
	return r
}

// WithMessageFactory sets the MessageFactory used to instantiate any messages.
// This method is not thread-safe and is intended to be used for one-time
// initialization of the registry, before it published for use by other threads.
func (r *MessageRegistry) WithMessageFactory(mf *MessageFactory) *MessageRegistry {
	r.mf = mf
	if mf == nil {
		r.er = nil
	} else {
		r.er = mf.er
	}
	return r
}

// WithDefaultDomain sets the default domain used when constructing type URLs for
// marshalling messages as Any types. If unspecified, the default domain is
// "type.googleapis.com". This method is not thread-safe and is intended to be used
// for one-time initialization of the registry, before it published for use by other
// threads.
func (r *MessageRegistry) WithDefaultDomain(domain string) *MessageRegistry {
	domain = canonicalizeDomain(domain)
	r.defaultDomain = domain
	return r
}

func canonicalizeDomain(domain string) string {
	domain = ensureScheme(domain)
	if domain[len(domain)-1] == '/' {
		return domain[:len(domain)-1]
	}
	return domain
}

func (r *MessageRegistry) AddMessage(url string, md *desc.MessageDescriptor) error {
	if !strings.HasSuffix(url, "/"+md.GetFullyQualifiedName()) {
		return fmt.Errorf("URL %s is invalid: it should end with path element %s", url, md.GetFullyQualifiedName())
	}
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages[url] = md
	return nil
}

func (r *MessageRegistry) AddEnum(url string, ed *desc.EnumDescriptor) error {
	if !strings.HasSuffix(url, "/"+ed.GetFullyQualifiedName()) {
		return fmt.Errorf("URL %s is invalid: it should end with path element %s", url, ed.GetFullyQualifiedName())
	}
	url = ensureScheme(url)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.messages[url] = ed
	return nil
}

func (r *MessageRegistry) AddFile(domain string, fd *desc.FileDescriptor) {
	domain = canonicalizeDomain(domain)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.addEnumTypesLocked(domain, fd.GetEnumTypes())
	r.addMessageTypesLocked(domain, fd.GetMessageTypes())
}

func (r *MessageRegistry) addEnumTypesLocked(domain string, enums []*desc.EnumDescriptor) {
	for _, ed := range enums {
		r.messages[fmt.Sprintf("%s/%s", domain, ed.GetFullyQualifiedName())] = ed
	}
}

func (r *MessageRegistry) addMessageTypesLocked(domain string, msgs []*desc.MessageDescriptor) {
	for _, md := range msgs {
		r.messages[fmt.Sprintf("%s/%s", domain, md.GetFullyQualifiedName())] = md
		r.addEnumTypesLocked(domain, md.GetNestedEnumTypes())
		r.addMessageTypesLocked(domain, md.GetNestedMessageTypes())
	}
}

func (r *MessageRegistry) FindMessageTypeByUrl(url string) (*desc.MessageDescriptor, error) {
	if r == nil {
		return nil, nil
	}
	url = ensureScheme(url)
	r.mu.RLock()
	m := r.messages[url]
	r.mu.RUnlock()
	if md, ok := m.(*desc.MessageDescriptor); ok {
		return md, nil
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
	if r.resolver.fetcher == nil {
		return nil, nil
	}
	if md, err := r.resolver.resolveUrlToMessageDescriptor(url); err != nil {
		return nil, err
	} else {
		return md, nil
	}
}

func (r *MessageRegistry) FindEnumTypeByUrl(url string) (*desc.EnumDescriptor, error) {
	if r == nil {
		return nil, nil
	}
	url = ensureScheme(url)
	r.mu.RLock()
	m := r.messages[url]
	r.mu.RUnlock()
	if ed, ok := m.(*desc.EnumDescriptor); ok {
		return ed, nil
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

func (r *MessageRegistry) ResolveApiIntoServiceDescriptor(a *api.Api) (*desc.ServiceDescriptor, error) {
	if r.resolver.fetcher == nil {
		return nil, nil
	}
	return r.resolver.resolveApiToServiceDescriptor(a)
}

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
		if msg = (*KnownTypeRegistry)(nil).CreateIfKnown(name); msg == nil {
			return nil, fmt.Errorf("Unknown message type: %s", any.TypeUrl)
		}
	} else {
		var ktr *KnownTypeRegistry
		if r.mf != nil {
			ktr = r.mf.ktr
		}
		if msg = ktr.CreateIfKnown(name); msg == nil {
			if md, err := fetch(any.TypeUrl); err != nil {
				return nil, err
			} else if md == nil {
				return nil, fmt.Errorf("Unknown message type: %s", any.TypeUrl)
			} else {
				msg = newMessageWithMessageFactory(md, r.mf)
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

func (r *MessageRegistry) AddDomainForElement(domain, packageOrTypeName string) {
	if domain[len(domain)-1] == '/' {
		domain = domain[:len(domain)-1]
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.domains[packageOrTypeName] = domain
}

func (r *MessageRegistry) MarshalAny(m proto.Message) (*any.Any, error) {
	name := MessageName(m)
	if name == "" {
		return nil, fmt.Errorf("could not determine message name for %v", reflect.TypeOf(m))
	}

	if b, err := proto.Marshal(m); err != nil {
		return nil, err
	} else {
		return &any.Any{TypeUrl: r.asUrl(name), Value: b}, nil
	}
}

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

	var card ptype.Field_Cardinality
	switch fd.GetLabel() {
	case descriptor.FieldDescriptorProto_LABEL_OPTIONAL:
		card = ptype.Field_CARDINALITY_OPTIONAL
	case descriptor.FieldDescriptorProto_LABEL_REPEATED:
		card = ptype.Field_CARDINALITY_REPEATED
	case descriptor.FieldDescriptorProto_LABEL_REQUIRED:
		card = ptype.Field_CARDINALITY_REQUIRED
	}

	var kind ptype.Field_Kind
	switch fd.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		kind = ptype.Field_TYPE_ENUM
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		kind = ptype.Field_TYPE_GROUP
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		kind = ptype.Field_TYPE_MESSAGE
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
		OneofIndex:   fd.AsFieldDescriptorProto().GetOneofIndex(),
		DefaultValue: fd.AsFieldDescriptorProto().GetDefaultValue(),
		Options:      opts,
		Packed:       fd.GetFieldOptions().GetPacked(),
		TypeUrl:      r.asUrl(fd.GetMessageType().GetFullyQualifiedName()),
		Cardinality:  card,
		Kind:         kind,
	}
}

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
		RequestTypeUrl:    r.asUrl(md.GetInputType().GetFullyQualifiedName()),
		ResponseTypeUrl:   r.asUrl(md.GetOutputType().GetFullyQualifiedName()),
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
			opts = append(opts, o)
		}
	}
	for _, ext := range proto.RegisteredExtensions(options) {
		if proto.HasExtension(options, ext) {
			v, err := proto.GetExtension(options, ext)
			if err == nil && v != nil {
				o := r.option(ext.Name, reflect.ValueOf(v))
				if o != nil {
					opts = append(opts, o)
				}
			}
		}
	}
	return opts
}

func (r *MessageRegistry) option(name string, value reflect.Value) *ptype.Option {
	// ignoring unsupported types or values that cannot be marshalled
	// TODO(jh): error or panic?
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

func (r *MessageRegistry) asUrl(name string) string {
	r.mu.RLock()
	domain := r.domains[name]
	if domain == "" {
		// lookup domain for the package
		domain = r.domains[name[strings.LastIndex(name, ".")+1:]]
	}
	r.mu.RUnlock()

	if domain == "" {
		domain = r.defaultDomain
		if domain == "" {
			domain = googleApisDomain
		}
	}

	return fmt.Sprintf("%s/%s", domain, name)
}
