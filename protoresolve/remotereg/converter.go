package remotereg

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// DescriptorConverter is a type that can be used to convert between descriptors and the
// other representations of types and services: google.protobuf.Type, google.protobuf.Enum,
// and google.protobuf.Api.
//
// It uses a Registry to convert the alternate representations, which may involve
// fetching remote types by type URL in order to create a complete representation of the
// transitive closure of the type or service.
//
// See Registry.AsDescriptorConverter.
type DescriptorConverter Registry

// ToServiceDescriptor converts the given Api message into a service descriptor. Since
// the service's referenced types may not yet be known, they may be fetched, which could
// warrant interruption by providing a cancellable context.
func (dc *DescriptorConverter) ToServiceDescriptor(ctx context.Context, api *apipb.Api) (protoreflect.ServiceDescriptor, error) {
	msgs := map[protoreflect.FullName]protoreflect.MessageDescriptor{}
	unresolved := map[string]struct{}{}
	reg := (*Registry)(dc)
	for _, m := range api.Methods {
		// request type
		md, err := reg.FindMessageByURLContext(ctx, m.RequestTypeUrl)
		if errors.Is(err, protoresolve.ErrNotFound) {
			if dc.TypeFetcher == nil {
				return nil, fmt.Errorf("could not resolve type URL %s for request of method %s.%s", m.RequestTypeUrl, api.Name, m.Name)
			}
			unresolved[m.RequestTypeUrl] = struct{}{}
		} else if err != nil {
			return nil, err
		} else {
			msgs[protoresolve.TypeNameFromURL(m.RequestTypeUrl)] = md
		}
		// and response type
		md, err = reg.FindMessageByURLContext(ctx, m.ResponseTypeUrl)
		if errors.Is(err, protoresolve.ErrNotFound) {
			if dc.TypeFetcher == nil {
				return nil, fmt.Errorf("could not resolve type URL %s for response of method %s.%s", m.ResponseTypeUrl, api.Name, m.Name)
			}
			unresolved[m.ResponseTypeUrl] = struct{}{}
		} else if err != nil {
			return nil, err
		} else {
			msgs[protoresolve.TypeNameFromURL(m.ResponseTypeUrl)] = md
		}
	}

	if len(unresolved) > 0 {
		unresolvedSlice := make([]string, 0, len(unresolved))
		for k := range unresolved {
			unresolvedSlice = append(unresolvedSlice, k)
		}
		mp, err := reg.findMessageTypesByURL(ctx, unresolvedSlice)
		if err != nil {
			return nil, err
		}
		for u, md := range mp {
			msgs[protoresolve.TypeNameFromURL(u)] = md
		}
	}

	var fileName string
	if api.SourceContext != nil && api.SourceContext.FileName != "" {
		fileName = api.SourceContext.FileName
	} else {
		fileName = fmt.Sprintf("--unknown--%d.proto", reg.fileCounter.Add(1))
	}

	// now we add all types we care about to a typeTrie and use that to generate file descriptors
	files := map[string]*fileEntry{}
	fe := &fileEntry{}
	fe.proto3 = api.Syntax == typepb.Syntax_SYNTAX_PROTO3
	files[fileName] = fe
	fe.types.addType(api.Name, createServiceDescriptor(api, (*remoteSubResolver)(reg)))
	added := newNameTracker()
	for _, md := range msgs {
		dc.addDescriptors(fileName, files, md, msgs, added)
	}

	// build resulting file descriptor(s) and return the final service descriptor
	var fallback protoresolve.FileResolver
	if dc.Fallback != nil {
		fallback, _ = dc.Fallback.(protoresolve.FileResolver)
	} else {
		fallback = protoregistry.GlobalFiles
	}
	fileDescriptors, err := toFileDescriptors(files, fallback, (*typeTrie).rewriteDescriptor)
	if err != nil {
		return nil, err
	}
	desc, err := fileDescriptors.FindDescriptorByName(protoreflect.FullName(api.Name))
	if err != nil {
		return nil, err
	}
	sd, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		// should not be possible?
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindService, desc, "")
	}
	return sd, nil
}

// ToMessageDescriptor converts the given Type message into a message descriptor. Since
// the message's fields may reference other types that are not yet known, other types
// may be fetched, which could warrant interruption by providing a cancellable context.
func (dc *DescriptorConverter) ToMessageDescriptor(ctx context.Context, msg *typepb.Type) (protoreflect.MessageDescriptor, error) {
	reg := (*Registry)(dc)
	cc := newConvertContext(reg, dc.TypeFetcher)
	typeName := protoreflect.FullName(msg.Name)
	typeURL := reg.urlForType(typeName, typeName.Parent())
	if err := cc.recordTypeAndDependencies(ctx, typeURL, msg); err != nil {
		return nil, err
	}
	desc, err := reg.resolveURLFromConvertContext(cc, typeURL)
	if err != nil {
		return nil, err
	}
	md, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		// should not be possible?
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindMessage, desc, "")
	}
	return md, nil
}

// ToEnumDescriptor converts the given Enum message into an enum descriptor.
func (dc *DescriptorConverter) ToEnumDescriptor(ctx context.Context, enum *typepb.Enum) (protoreflect.EnumDescriptor, error) {
	// NB: We keep ctx in signature for consistency... and just in case we need it in the future.
	//     Ideally, we'd use the context to fetch extension descriptions for enum custom options.
	//     But there's no spec for discovering/downloading extensions, only types.
	_ = ctx

	reg := (*Registry)(dc)
	cc := newConvertContext(reg, dc.TypeFetcher)
	typeName := protoreflect.FullName(enum.Name)
	typeURL := reg.urlForType(typeName, typeName.Parent())
	cc.recordEnum(typeURL, enum)
	desc, err := reg.resolveURLFromConvertContext(cc, typeURL)
	if err != nil {
		return nil, err
	}
	ed, ok := desc.(protoreflect.EnumDescriptor)
	if !ok {
		// should not be possible?
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindEnum, desc, "")
	}
	return ed, nil
}

// DescriptorAsAPI produces an Api message that represents the given service descriptor.
func (dc *DescriptorConverter) DescriptorAsAPI(sd protoreflect.ServiceDescriptor) *apipb.Api {
	ms := sd.Methods()
	reg := (*Registry)(dc)
	methods := make([]*apipb.Method, ms.Len())
	for i, length := 0, ms.Len(); i < length; i++ {
		mtd := ms.Get(i)
		methods[i] = &apipb.Method{
			Name:              string(mtd.Name()),
			RequestStreaming:  mtd.IsStreamingClient(),
			ResponseStreaming: mtd.IsStreamingServer(),
			RequestTypeUrl:    reg.URLForType(mtd.Input()),
			ResponseTypeUrl:   reg.URLForType(mtd.Output()),
			Options:           dc.options(mtd.Options()),
		}
		//lint:ignore SA1019 readers should use Api.syntax instead, but we still populate this for older readers
		methods[i].Syntax = syntax(mtd.ParentFile().Syntax())
	}
	return &apipb.Api{
		Name:          string(sd.FullName()),
		Methods:       methods,
		Options:       dc.options(sd.Options()),
		Syntax:        syntax(sd.ParentFile().Syntax()),
		SourceContext: &sourcecontextpb.SourceContext{FileName: sd.ParentFile().Path()},
	}
}

// DescriptorAsType produces a Type message that represents the given message descriptor.
func (dc *DescriptorConverter) DescriptorAsType(md protoreflect.MessageDescriptor) *typepb.Type {
	fs := md.Fields()
	fields := make([]*typepb.Field, fs.Len())
	for i, length := 0, fs.Len(); i < length; i++ {
		fields[i] = dc.descriptorAsField(fs.Get(i))
	}
	oos := md.Oneofs()
	oneOfs := make([]string, oos.Len())
	for i, length := 0, oos.Len(); i < length; i++ {
		oneOfs[i] = string(oos.Get(i).Name())
	}
	return &typepb.Type{
		Name:          string(md.FullName()),
		Fields:        fields,
		Oneofs:        oneOfs,
		Options:       dc.options(md.Options()),
		Syntax:        syntax(md.ParentFile().Syntax()),
		SourceContext: &sourcecontextpb.SourceContext{FileName: md.ParentFile().Path()},
	}
}

func (dc *DescriptorConverter) descriptorAsField(fld protoreflect.FieldDescriptor) *typepb.Field {
	opts := dc.options(fld.Options())
	// remove the "packed" option as that is represented via separate field in ptype.Field
	for i, o := range opts {
		if o.Name == "packed" {
			opts = append(opts[:i], opts[i+1:]...)
			break
		}
	}

	var oneOf int32
	if oo := fld.ContainingOneof(); oo != nil {
		// NB: the typepb representation uses zero to mean "not in a oneof" so
		//     the oneof indexes start at one.
		oneOf = int32(oo.Index()) + 1
	}

	var card typepb.Field_Cardinality
	switch fld.Cardinality() {
	case protoreflect.Optional:
		card = typepb.Field_CARDINALITY_OPTIONAL
	case protoreflect.Repeated:
		card = typepb.Field_CARDINALITY_REPEATED
	case protoreflect.Required:
		card = typepb.Field_CARDINALITY_REQUIRED
	}

	reg := (*Registry)(dc)
	var url string
	var kind typepb.Field_Kind
	switch fld.Kind() {
	case protoreflect.EnumKind:
		kind = typepb.Field_TYPE_ENUM
		url = reg.URLForType(fld.Enum())
	case protoreflect.GroupKind:
		kind = typepb.Field_TYPE_GROUP
		url = reg.URLForType(fld.Message())
	case protoreflect.MessageKind:
		kind = typepb.Field_TYPE_MESSAGE
		url = reg.URLForType(fld.Message())
	case protoreflect.BytesKind:
		kind = typepb.Field_TYPE_BYTES
	case protoreflect.StringKind:
		kind = typepb.Field_TYPE_STRING
	case protoreflect.BoolKind:
		kind = typepb.Field_TYPE_BOOL
	case protoreflect.DoubleKind:
		kind = typepb.Field_TYPE_DOUBLE
	case protoreflect.FloatKind:
		kind = typepb.Field_TYPE_FLOAT
	case protoreflect.Fixed32Kind:
		kind = typepb.Field_TYPE_FIXED32
	case protoreflect.Fixed64Kind:
		kind = typepb.Field_TYPE_FIXED64
	case protoreflect.Int32Kind:
		kind = typepb.Field_TYPE_INT32
	case protoreflect.Int64Kind:
		kind = typepb.Field_TYPE_INT64
	case protoreflect.Sfixed32Kind:
		kind = typepb.Field_TYPE_SFIXED32
	case protoreflect.Sfixed64Kind:
		kind = typepb.Field_TYPE_SFIXED64
	case protoreflect.Sint32Kind:
		kind = typepb.Field_TYPE_SINT32
	case protoreflect.Sint64Kind:
		kind = typepb.Field_TYPE_SINT64
	case protoreflect.Uint32Kind:
		kind = typepb.Field_TYPE_UINT32
	case protoreflect.Uint64Kind:
		kind = typepb.Field_TYPE_UINT64
	}
	var defVal string
	if fld.HasDefault() {
		defVal = defaultValueString(fld.Kind(), fld.Default(), fld.DefaultEnumValue())
	}

	return &typepb.Field{
		Name:         string(fld.Name()),
		Number:       int32(fld.Number()),
		JsonName:     fld.JSONName(),
		OneofIndex:   oneOf,
		DefaultValue: defVal,
		Options:      opts,
		Packed:       fld.IsPacked(),
		TypeUrl:      url,
		Cardinality:  card,
		Kind:         kind,
	}
}

// DescriptorAsEnum produces an Enum message that represents the given enum descriptor.
func (dc *DescriptorConverter) DescriptorAsEnum(ed protoreflect.EnumDescriptor) *typepb.Enum {
	vs := ed.Values()
	vals := make([]*typepb.EnumValue, vs.Len())
	for i, length := 0, vs.Len(); i < length; i++ {
		evd := vs.Get(i)
		vals[i] = &typepb.EnumValue{
			Name:    string(evd.Name()),
			Number:  int32(evd.Number()),
			Options: dc.options(evd.Options()),
		}
	}
	return &typepb.Enum{
		Name:          string(ed.FullName()),
		Enumvalue:     vals,
		Options:       dc.options(ed.Options()),
		Syntax:        syntax(ed.ParentFile().Syntax()),
		SourceContext: &sourcecontextpb.SourceContext{FileName: ed.ParentFile().Path()},
	}
}

func defaultValueString(k protoreflect.Kind, v protoreflect.Value, evd protoreflect.EnumValueDescriptor) string {
	switch k {
	case protoreflect.BoolKind:
		if v.Bool() {
			return "true"
		}
		return "false"
	case protoreflect.EnumKind:
		return string(evd.Name())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(v.Int(), 10)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(v.Uint(), 10)
	case protoreflect.FloatKind, protoreflect.DoubleKind:
		f := v.Float()
		switch {
		case math.IsInf(f, -1):
			return "-inf"
		case math.IsInf(f, +1):
			return "inf"
		case math.IsNaN(f):
			return "nan"
		}
		if k == protoreflect.FloatKind {
			return strconv.FormatFloat(f, 'g', -1, 32)
		}
		return strconv.FormatFloat(f, 'g', -1, 64)
	case protoreflect.StringKind:
		// String values are serialized as is without any escaping.
		return v.String()
	case protoreflect.BytesKind:
		b := v.Bytes()
		s := make([]byte, 0, len(b))
		for _, c := range b {
			switch c {
			case '\n':
				s = append(s, '\\', 'n')
			case '\r':
				s = append(s, '\\', 'r')
			case '\t':
				s = append(s, '\\', 't')
			case '"':
				s = append(s, '\\', '"')
			case '\'':
				s = append(s, '\\', '\'')
			case '\\':
				s = append(s, '\\', '\\')
			default:
				if printableASCII := c >= 0x20 && c <= 0x7e; printableASCII {
					s = append(s, c)
				} else {
					s = append(s, fmt.Sprintf(`\%03o`, c)...)
				}
			}
		}
		return string(s)
	default:
		return ""
	}
}

func syntax(s protoreflect.Syntax) typepb.Syntax {
	switch s {
	case protoreflect.Proto3:
		return typepb.Syntax_SYNTAX_PROTO3
	case protoreflect.Proto2:
		return typepb.Syntax_SYNTAX_PROTO2
	default:
		// TODO: This really should be an "UNSET" constant. But type.proto doesn't declare any such value for the Syntax enum.
		return 0
	}
}

type tracker func(d protoreflect.Descriptor) bool

func newNameTracker() tracker {
	names := map[protoreflect.FullName]struct{}{}
	return func(d protoreflect.Descriptor) bool {
		name := d.FullName()
		if _, ok := names[name]; ok {
			return false
		}
		names[name] = struct{}{}
		return true
	}
}

func (dc *DescriptorConverter) addDescriptors(ref string, files map[string]*fileEntry, d protoreflect.Descriptor, msgs map[protoreflect.FullName]protoreflect.MessageDescriptor, onAdd tracker) {
	name := d.FullName()

	fileName := d.ParentFile().Path()
	if fileName != ref {
		dependee := files[ref]
		if dependee.deps == nil {
			dependee.deps = map[string]struct{}{}
		}
		dependee.deps[fileName] = struct{}{}
	}

	if !onAdd(d) {
		// already added this one
		return
	}

	fe := files[fileName]
	if fe == nil {
		fe = &fileEntry{}
		fe.proto3 = d.ParentFile().Syntax() == protoreflect.Proto3
		files[fileName] = fe
	}

	dc.mu.RLock()
	descProto := dc.descProtos[d]
	dc.mu.RUnlock()
	if descProto == nil {
		switch d := d.(type) {
		case protoreflect.MessageDescriptor:
			descProto = protodesc.ToDescriptorProto(d)
		case protoreflect.EnumDescriptor:
			descProto = protodesc.ToEnumDescriptorProto(d)
		}
	}
	if descProto != nil {
		fe.types.addType(string(name), descProto)
	}

	if md, ok := d.(protoreflect.MessageDescriptor); ok {
		fields := md.Fields()
		for i, length := 0, fields.Len(); i < length; i++ {
			fld := fields.Get(i)
			if fld.Kind() == protoreflect.MessageKind || fld.Kind() == protoreflect.GroupKind {
				// prefer descriptor in msgs map over what the field descriptor indicates
				md := msgs[fld.Message().FullName()]
				if md == nil {
					md = fld.Message()
				}
				dc.addDescriptors(fileName, files, md, msgs, onAdd)
			} else if fld.Kind() == protoreflect.EnumKind {
				dc.addDescriptors(fileName, files, fld.Enum(), msgs, onAdd)
			}
		}
	}
}
