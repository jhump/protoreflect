package protoresolve

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/jhump/protoreflect/v2/internal/wrappers"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// DescriptorConverter is a type that can be used to convert between descriptors and the
// other representations of types and services: google.protobuf.Type, google.protobuf.Enum,
// and google.protobuf.Api.
//
// It uses a RemoteRegistry to convert the alternate representations, which may involve
// fetching remote types by type URL in order to create a complete representation of the
// transitive closure of the type or service.
//
// See RemoteRegistry.AsDescriptorConverter.
type DescriptorConverter RemoteRegistry

func (dc *DescriptorConverter) ToServiceDescriptor(ctx context.Context, api *apipb.Api) (protoreflect.ServiceDescriptor, error) {
	msgs := map[protoreflect.FullName]protoreflect.MessageDescriptor{}
	unresolved := map[string]struct{}{}
	reg := (*RemoteRegistry)(dc)
	for _, m := range api.Methods {
		// request type
		md, err := reg.findMessageByURLContext(ctx, m.RequestTypeUrl, nil)
		if errors.Is(err, ErrNotFound) {
			if dc.TypeFetcher == nil {
				return nil, fmt.Errorf("could not resolve type URL %s for request of method %s.%s", m.RequestTypeUrl, api.Name, m.Name)
			}
			unresolved[m.RequestTypeUrl] = struct{}{}
		} else if err != nil {
			return nil, err
		} else {
			msgs[TypeNameFromURL(m.RequestTypeUrl)] = md
		}
		// and response type
		md, err = reg.findMessageByURLContext(ctx, m.ResponseTypeUrl, nil)
		if errors.Is(err, ErrNotFound) {
			if dc.TypeFetcher == nil {
				return nil, fmt.Errorf("could not resolve type URL %s for response of method %s.%s", m.ResponseTypeUrl, api.Name, m.Name)
			}
			unresolved[m.ResponseTypeUrl] = struct{}{}
		} else if err != nil {
			return nil, err
		} else {
			msgs[TypeNameFromURL(m.ResponseTypeUrl)] = md
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
			msgs[TypeNameFromURL(u)] = md
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
		addDescriptors(fileName, files, md, msgs, added)
	}

	// build resulting file descriptor(s) and return the final service descriptor
	fileDescriptors, err := toFileDescriptors(files, (*typeTrie).rewriteDescriptor)
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
		return nil, fmt.Errorf("expecting a service; instead got %s", descKindWithArticle(desc))
	}
	return sd, nil
}

func (dc *DescriptorConverter) ToMessageDescriptor(ctx context.Context, msg *typepb.Type) (protoreflect.MessageDescriptor, error) {
	reg := (*RemoteRegistry)(dc)
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
		return nil, fmt.Errorf("expecting a message; instead got %s", descKindWithArticle(desc))
	}
	return md, nil
}

func (dc *DescriptorConverter) ToEnumDescriptor(ctx context.Context, enum *typepb.Enum) (protoreflect.EnumDescriptor, error) {
	// NB: We keep ctx in context for consistency... and just in case we need it in the future. Ideally,
	//     we'd use the context to fetch extension descriptions for enum options. But there's no spec
	//     for discovering/downloading extensions, only types.
	_ = ctx

	reg := (*RemoteRegistry)(dc)
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
		return nil, fmt.Errorf("expecting an enum; instead got %s", descKindWithArticle(desc))
	}
	return ed, nil
}

func (dc *DescriptorConverter) DescriptorAsApi(sd protoreflect.ServiceDescriptor) *apipb.Api {
	ms := sd.Methods()
	reg := (*RemoteRegistry)(dc)
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
			Syntax:            syntax(mtd.ParentFile().Syntax()),
		}
	}
	return &apipb.Api{
		Name:          string(sd.FullName()),
		Methods:       methods,
		Options:       dc.options(sd.Options()),
		Syntax:        syntax(sd.ParentFile().Syntax()),
		SourceContext: &sourcecontextpb.SourceContext{FileName: sd.ParentFile().Path()},
	}
}

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
		oneOf = int32(oo.Index())
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

	reg := (*RemoteRegistry)(dc)
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

func (dc *DescriptorConverter) options(options proto.Message) []*typepb.Option {
	rv := reflect.ValueOf(options)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	var opts []*typepb.Option
	options.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		o := dc.option(fd, val)
		if len(o) > 0 {
			opts = append(opts, o...)
		}
		return true
	})
	return opts
}

func (dc *DescriptorConverter) option(field protoreflect.FieldDescriptor, value protoreflect.Value) []*typepb.Option {
	switch {
	case field.IsList():
		listVal := value.List()
		opts := make([]*typepb.Option, 0, listVal.Len())
		for i, length := 0, listVal.Len(); i < length; i++ {
			if opt := dc.singleOption(field, value); opt != nil {
				opts = append(opts, opt)
			}
		}
		return opts
	case field.IsMap():
		mapVal := value.Map()
		opts := make([]*typepb.Option, 0, mapVal.Len())
		mapVal.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
			entry := dynamicpb.NewMessage(field.Message())
			entry.Set(field.MapKey(), k.Value())
			entry.Set(field.MapValue(), v)
			if opt := dc.singleOption(field, protoreflect.ValueOfMessage(entry)); opt != nil {
				opts = append(opts, opt)
			}
			return true
		})
		return opts
	default:
		if opt := dc.singleOption(field, value); opt != nil {
			return []*typepb.Option{opt}
		}
		return nil
	}
}

func (dc *DescriptorConverter) singleOption(field protoreflect.FieldDescriptor, value protoreflect.Value) *typepb.Option {
	pm := maybeWrap(field.Kind(), value)
	if pm == nil {
		return nil
	}
	var a anypb.Any
	if err := anypb.MarshalFrom(&a, pm, proto.MarshalOptions{}); err != nil {
		return nil
	}
	var name string
	if field.IsExtension() {
		name = string(field.FullName())
	} else {
		name = string(field.Name())
	}
	return &typepb.Option{
		Name:  name,
		Value: &a,
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
		s := make([]byte, len(b))
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

func maybeWrap(k protoreflect.Kind, v protoreflect.Value) proto.Message {
	if !v.IsValid() {
		return nil
	}
	if k == protoreflect.MessageKind || k == protoreflect.GroupKind {
		return v.Message().Interface()
	}
	switch k {
	case protoreflect.BoolKind:
		return &wrapperspb.BoolValue{Value: v.Bool()}
	case protoreflect.BytesKind:
		return &wrapperspb.BytesValue{Value: v.Bytes()}
	case protoreflect.StringKind:
		return &wrapperspb.StringValue{Value: v.String()}
	case protoreflect.FloatKind:
		return &wrapperspb.FloatValue{Value: float32(v.Float())}
	case protoreflect.DoubleKind:
		return &wrapperspb.DoubleValue{Value: v.Float()}
	case protoreflect.EnumKind:
		return &wrapperspb.Int32Value{Value: int32(v.Enum())}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return &wrapperspb.Int32Value{Value: int32(v.Int())}
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return &wrapperspb.Int64Value{Value: v.Int()}
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return &wrapperspb.UInt32Value{Value: uint32(v.Uint())}
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return &wrapperspb.UInt64Value{Value: v.Uint()}
	default:
		return nil
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

func addDescriptors(ref string, files map[string]*fileEntry, d protoreflect.Descriptor, msgs map[protoreflect.FullName]protoreflect.MessageDescriptor, onAdd tracker) {
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

	fe.types.addType(string(name), protoFromDescriptor(d))

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
				addDescriptors(fileName, files, md, msgs, onAdd)
			} else if fld.Kind() == protoreflect.EnumKind {
				addDescriptors(fileName, files, fld.Enum(), msgs, onAdd)
			}
		}
	}
}

// convertContext provides the state for a resolution operation, accumulating details about
// type descriptions and the files that contain them.
type convertContext struct {
	reg     *RemoteRegistry
	res     SerializationResolver
	fetcher TypeFetcher

	mu sync.Mutex
	// map of file names to details regarding the files' contents
	files map[string]*fileEntry
	// map of type URLs to the file name that defines them
	typeLocations map[string]string
}

func newConvertContext(reg *RemoteRegistry, fetcher TypeFetcher) *convertContext {
	return &convertContext{
		reg:           reg,
		res:           (*remoteSubResolver)(reg),
		fetcher:       fetcher,
		typeLocations: map[string]string{},
		files:         map[string]*fileEntry{},
	}
}

// addType adds the type at the given URL to the context, using the given fetcher to download the type's
// description. This function will recursively add dependencies (e.g. types referenced by the given type's
// fields if it is a message type), fetching their type descriptions concurrently.
func (cc *convertContext) addType(ctx context.Context, url string, enum bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if enum {
		var err error
		et, err := cc.fetcher.FetchEnumType(ctx, url)
		if err != nil {
			return err
		}
		cc.recordEnum(url, et)
		return nil
	}

	mt, err := cc.fetcher.FetchMessageType(ctx, url)
	if err != nil {
		return err
	}
	return cc.recordTypeAndDependencies(ctx, url, mt)
}

func (cc *convertContext) recordEnum(url string, e *typepb.Enum) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	var fileName string
	if e.SourceContext != nil && e.SourceContext.FileName != "" {
		fileName = e.SourceContext.FileName
	} else {
		fileName = fmt.Sprintf("--unknown--%d.proto", cc.reg.fileCounter.Add(1))
	}
	cc.typeLocations[url] = fileName

	fe := cc.files[fileName]
	if fe == nil {
		fe = &fileEntry{}
		cc.files[fileName] = fe
	}
	fe.types.addType(e.Name, e)
	if e.Syntax == typepb.Syntax_SYNTAX_PROTO3 {
		fe.proto3 = true
	}
}

func (cc *convertContext) recordTypeAndDependencies(ctx context.Context, url string, mt *typepb.Type) error {
	fe, fileName := cc.recordType(url, mt)
	if fe == nil {
		// already resolved this one
		return nil
	}

	// Resolve dependencies in parallel.
	grp, ctx := errgroup.WithContext(ctx)
	for _, f := range mt.Fields {
		if f.Kind == typepb.Field_TYPE_GROUP || f.Kind == typepb.Field_TYPE_MESSAGE || f.Kind == typepb.Field_TYPE_ENUM {
			typeURL := ensureScheme(f.TypeUrl)
			kind := f.Kind
			grp.Go(func() error {
				// first check the registry for descriptors
				cc.reg.mu.Lock()
				d := cc.reg.typeCache[typeURL]
				cc.reg.mu.Unlock()

				if d != nil {
					// found it!
					cc.recordDescriptor(typeURL, fileName, d)
					return nil
				}

				// not in registry, so we have to recursively fetch
				if err := cc.addType(ctx, typeURL, kind == typepb.Field_TYPE_ENUM); err != nil {
					return err
				}
				return nil
			})
		}
	}
	if err := grp.Wait(); err != nil {
		return err
	}
	// double-check if context has been cancelled
	if ctx.Err() != nil {
		return ctx.Err()
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	for _, f := range mt.Fields {
		if f.Kind == typepb.Field_TYPE_GROUP || f.Kind == typepb.Field_TYPE_MESSAGE || f.Kind == typepb.Field_TYPE_ENUM {
			typeUrl := ensureScheme(f.TypeUrl)
			if fe.deps == nil {
				fe.deps = map[string]struct{}{}
			}
			dep := cc.typeLocations[typeUrl]
			if dep != fileName {
				fe.deps[dep] = struct{}{}
			}
		}
	}
	return nil
}

func (cc *convertContext) recordType(url string, t *typepb.Type) (*fileEntry, string) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if _, ok := cc.typeLocations[url]; ok {
		return nil, ""
	}

	var fileName string
	if t.SourceContext != nil && t.SourceContext.FileName != "" {
		fileName = t.SourceContext.FileName
	} else {
		fileName = fmt.Sprintf("--unknown--%d.proto", cc.reg.fileCounter.Add(1))
	}
	cc.typeLocations[url] = fileName

	fe := cc.files[fileName]
	if fe == nil {
		fe = &fileEntry{}
		cc.files[fileName] = fe
	}
	fe.types.addType(t.Name, t)
	if t.Syntax == typepb.Syntax_SYNTAX_PROTO3 {
		fe.proto3 = true
	}

	return fe, fileName
}

func (cc *convertContext) recordDescriptor(url, ref string, d protoreflect.Descriptor) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	addDescriptors(ref, cc.files, d, nil, func(dsc protoreflect.Descriptor) bool {
		u := ensureScheme(cc.reg.urlForType(dsc.FullName(), dsc.Parent().FullName()))
		if _, ok := cc.typeLocations[u]; ok {
			// already seen this one
			return false
		}
		fileName := dsc.ParentFile().Path()
		cc.typeLocations[u] = fileName
		if dsc == d {
			// make sure we're also adding the actual URL reference used
			cc.typeLocations[url] = fileName
		}
		return true
	})
}

// toFileDescriptors converts the information in the context into a map of file names to file descriptors.
func (cc *convertContext) toFileDescriptors() (*protoregistry.Files, error) {
	return toFileDescriptors(cc.files, func(tt *typeTrie, name string) (proto.Message, error) {
		mdp, edp := tt.typeToDescriptor(name, cc.res)
		if mdp != nil {
			return mdp, nil
		} else {
			return edp, nil
		}
	})
}

// converts a map of file entries into a map of file descriptors using the given function to convert
// each trie node into a descriptor proto.
func toFileDescriptors(files map[string]*fileEntry, trieFn func(*typeTrie, string) (proto.Message, error)) (*protoregistry.Files, error) {
	fdps := map[string]*descriptorpb.FileDescriptorProto{}
	for name, file := range files {
		fdp, err := file.toFileDescriptor(name, trieFn)
		if err != nil {
			return nil, err
		}
		fdps[name] = fdp
	}
	var reg protoregistry.Files
	for _, fdp := range fdps {
		if err := addToRegistry(fdp, &reg, fdps); err != nil {
			return nil, err
		}
	}
	return &reg, nil
}

func addToRegistry(fdp *descriptorpb.FileDescriptorProto, reg *protoregistry.Files, fdps map[string]*descriptorpb.FileDescriptorProto) error {
	if _, err := reg.FindFileByPath(fdp.GetName()); err == nil {
		return nil // already registered
	}
	for _, dep := range fdp.Dependency {
		depFd := fdps[dep]
		if depFd == nil {
			return fmt.Errorf("missing dependency: %s", dep)
		}
		if err := addToRegistry(depFd, reg, fdps); err != nil {
			return err
		}
	}
	fd, err := protodesc.NewFile(fdp, reg)
	if err == nil {
		err = reg.RegisterFile(wrappers.WrapFile(fd, fdp))
	}
	return err
}

// fileEntry represents the contents of a single file.
type fileEntry struct {
	types  typeTrie
	deps   map[string]struct{}
	proto3 bool
}

// toFileDescriptor converts this file entry into a file descriptor proto. The given function
// is used to transform nodes in a typeTrie into message and/or enum descriptor protos.
func (fe *fileEntry) toFileDescriptor(name string, trieFn func(*typeTrie, string) (proto.Message, error)) (*descriptorpb.FileDescriptorProto, error) {
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
		if len(tt.children) != 1 {
			break
		}
		for last, tt = range tt.children {
		}
	}
	fd := createFileDescriptor(name, pkg.String(), fe.proto3, fe.deps)
	if tt.typ != nil {
		pm, err := trieFn(tt, last)
		if err != nil {
			return nil, err
		}
		if mdp, ok := pm.(*descriptorpb.DescriptorProto); ok {
			fd.MessageType = append(fd.MessageType, mdp)
		} else if edp, ok := pm.(*descriptorpb.EnumDescriptorProto); ok {
			fd.EnumType = append(fd.EnumType, edp)
		} else {
			sdp := pm.(*descriptorpb.ServiceDescriptorProto)
			fd.Service = append(fd.Service, sdp)
		}
	} else {
		for name, nested := range tt.children {
			pm, err := trieFn(nested, name)
			if err != nil {
				return nil, err
			}
			if mdp, ok := pm.(*descriptorpb.DescriptorProto); ok {
				fd.MessageType = append(fd.MessageType, mdp)
			} else if edp, ok := pm.(*descriptorpb.EnumDescriptorProto); ok {
				fd.EnumType = append(fd.EnumType, edp)
			} else {
				sdp := pm.(*descriptorpb.ServiceDescriptorProto)
				fd.Service = append(fd.Service, sdp)
			}
		}
	}
	return fd, nil
}

// typeTrie is a prefix trie where each key component is part of a fully-qualified type name. So key components
// will either be package name components or element names.
type typeTrie struct {
	// successor key components
	children map[string]*typeTrie
	// if non-nil, the element whose fully-qualified name is the path from the trie root to this node
	typ proto.Message
}

// addType recursively adds an element to the trie.
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

// typeToDescriptor converts this level of the trie into a message or enum
// descriptor proto, requiring that the element stored in t.typ is a *ptype.Type
// or *ptype.Enum. If t.typ is nil, a placeholder message (with no fields) is
// returned that contains the trie's children as nested message and/or enum
// types.
//
// If the value in t.typ is already a *descriptor.DescriptorProto or a
// *descriptor.EnumDescriptorProto then it is returned as is. This function
// should not be used in type tries that may have service descriptors. That will
// result in a panic.
func (t *typeTrie) typeToDescriptor(name string, res SerializationResolver) (*descriptorpb.DescriptorProto, *descriptorpb.EnumDescriptorProto) {
	switch typ := t.typ.(type) {
	case *descriptorpb.EnumDescriptorProto:
		return nil, typ
	case *typepb.Enum:
		return nil, createEnumDescriptor(typ, res)
	case *descriptorpb.DescriptorProto:
		return typ, nil
	default:
		var msg *descriptorpb.DescriptorProto
		if t.typ == nil {
			msg = createIntermediateMessageDescriptor(name)
		} else {
			msg = createMessageDescriptor(t.typ.(*typepb.Type), res)
		}
		// sort children for deterministic output
		var keys []string
		for k := range t.children {
			keys = append(keys, k)
		}
		for _, name := range keys {
			nested := t.children[name]
			chMsg, chEnum := nested.typeToDescriptor(name, res)
			if chMsg != nil {
				msg.NestedType = append(msg.NestedType, chMsg)
			}
			if chEnum != nil {
				msg.EnumType = append(msg.EnumType, chEnum)
			}
		}
		return msg, nil
	}
}

// rewriteDescriptor converts this level of the trie into a new descriptor
// proto, requiring that the element stored in t.type is already a service,
// message, or enum descriptor proto. If this trie has children then t.typ must
// be a message descriptor proto. The returned descriptor proto is the same as
// .type but with possibly new nested elements to represent this trie node's
// children.
func (t *typeTrie) rewriteDescriptor(name string) (proto.Message, error) {
	if len(t.children) == 0 && t.typ != nil {
		if mdp, ok := t.typ.(*descriptorpb.DescriptorProto); ok {
			if len(mdp.NestedType) == 0 && len(mdp.EnumType) == 0 {
				return mdp, nil
			}
			mdp = proto.Clone(mdp).(*descriptorpb.DescriptorProto)
			mdp.NestedType = nil
			mdp.EnumType = nil
			return mdp, nil
		}
		return t.typ, nil
	}
	var mdp *descriptorpb.DescriptorProto
	if t.typ == nil {
		mdp = createIntermediateMessageDescriptor(name)
	} else {
		mdp = t.typ.(*descriptorpb.DescriptorProto)
		mdp = proto.Clone(mdp).(*descriptorpb.DescriptorProto)
		mdp.NestedType = nil
		mdp.EnumType = nil
	}
	// sort children for deterministic output
	var keys []string
	for k := range t.children {
		keys = append(keys, k)
	}
	for _, n := range keys {
		ch := t.children[n]
		typ, err := ch.rewriteDescriptor(n)
		if err != nil {
			return nil, err
		}
		switch typ := typ.(type) {
		case (*descriptorpb.DescriptorProto):
			mdp.NestedType = append(mdp.NestedType, typ)
		case (*descriptorpb.EnumDescriptorProto):
			mdp.EnumType = append(mdp.EnumType, typ)
		default:
			// TODO: this should probably panic instead
			return nil, fmt.Errorf("invalid descriptor trie: message cannot have child of type %v", reflect.TypeOf(typ))
		}
	}
	return mdp, nil
}

func split(s string) (string, string) {
	pos := strings.Index(s, ".")
	if pos >= 0 {
		return s[:pos], s[pos+1:]
	} else {
		return s, ""
	}
}

func createEnumDescriptor(e *typepb.Enum, res SerializationResolver) *descriptorpb.EnumDescriptorProto {
	opts := &descriptorpb.EnumOptions{}
	if len(e.Options) > 0 {
		processOptions(e.Options, opts.ProtoReflect(), res)
	}

	var vals []*descriptorpb.EnumValueDescriptorProto
	for _, v := range e.Enumvalue {
		evd := createEnumValueDescriptor(v, res)
		vals = append(vals, evd)
	}

	return &descriptorpb.EnumDescriptorProto{
		Name:    proto.String(base(e.Name)),
		Options: opts,
		Value:   vals,
	}
}

func createEnumValueDescriptor(v *typepb.EnumValue, res SerializationResolver) *descriptorpb.EnumValueDescriptorProto {
	opts := &descriptorpb.EnumValueOptions{}
	if len(v.Options) > 0 {
		processOptions(v.Options, opts.ProtoReflect(), res)
	}

	return &descriptorpb.EnumValueDescriptorProto{
		Name:    proto.String(v.Name),
		Number:  proto.Int32(v.Number),
		Options: opts,
	}
}

func createMessageDescriptor(m *typepb.Type, res SerializationResolver) *descriptorpb.DescriptorProto {
	opts := &descriptorpb.MessageOptions{}
	if len(m.Options) > 0 {
		processOptions(m.Options, opts.ProtoReflect(), res)
	}

	var fields []*descriptorpb.FieldDescriptorProto
	for _, f := range m.Fields {
		fields = append(fields, createFieldDescriptor(f, res))
	}

	var oneOfs []*descriptorpb.OneofDescriptorProto
	for _, o := range m.Oneofs {
		oneOfs = append(oneOfs, &descriptorpb.OneofDescriptorProto{
			Name: proto.String(o),
		})
	}

	return &descriptorpb.DescriptorProto{
		Name:      proto.String(base(m.Name)),
		Options:   opts,
		Field:     fields,
		OneofDecl: oneOfs,
	}
}

func createFieldDescriptor(f *typepb.Field, res SerializationResolver) *descriptorpb.FieldDescriptorProto {
	opts := &descriptorpb.FieldOptions{}
	if len(f.Options) > 0 {
		processOptions(f.Options, opts.ProtoReflect(), res)
	}
	if f.Packed {
		if opts == nil {
			opts = &descriptorpb.FieldOptions{Packed: proto.Bool(true)}
		} else {
			opts.Packed = proto.Bool(true)
		}
	}

	var oneOf *int32
	if f.OneofIndex > 0 {
		oneOf = proto.Int32(f.OneofIndex - 1)
	}

	var typeName string
	if f.Kind == typepb.Field_TYPE_GROUP || f.Kind == typepb.Field_TYPE_MESSAGE || f.Kind == typepb.Field_TYPE_ENUM {
		pos := strings.LastIndex(f.TypeUrl, "/")
		typeName = "." + f.TypeUrl[pos+1:]
	}

	var label descriptorpb.FieldDescriptorProto_Label
	switch f.Cardinality {
	case typepb.Field_CARDINALITY_OPTIONAL:
		label = descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	case typepb.Field_CARDINALITY_REPEATED:
		label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	case typepb.Field_CARDINALITY_REQUIRED:
		label = descriptorpb.FieldDescriptorProto_LABEL_REQUIRED
	}

	var typ descriptorpb.FieldDescriptorProto_Type
	switch f.Kind {
	case typepb.Field_TYPE_ENUM:
		typ = descriptorpb.FieldDescriptorProto_TYPE_ENUM
	case typepb.Field_TYPE_GROUP:
		typ = descriptorpb.FieldDescriptorProto_TYPE_GROUP
	case typepb.Field_TYPE_MESSAGE:
		typ = descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	case typepb.Field_TYPE_BYTES:
		typ = descriptorpb.FieldDescriptorProto_TYPE_BYTES
	case typepb.Field_TYPE_STRING:
		typ = descriptorpb.FieldDescriptorProto_TYPE_STRING
	case typepb.Field_TYPE_BOOL:
		typ = descriptorpb.FieldDescriptorProto_TYPE_BOOL
	case typepb.Field_TYPE_DOUBLE:
		typ = descriptorpb.FieldDescriptorProto_TYPE_DOUBLE
	case typepb.Field_TYPE_FLOAT:
		typ = descriptorpb.FieldDescriptorProto_TYPE_FLOAT
	case typepb.Field_TYPE_FIXED32:
		typ = descriptorpb.FieldDescriptorProto_TYPE_FIXED32
	case typepb.Field_TYPE_FIXED64:
		typ = descriptorpb.FieldDescriptorProto_TYPE_FIXED64
	case typepb.Field_TYPE_INT32:
		typ = descriptorpb.FieldDescriptorProto_TYPE_INT32
	case typepb.Field_TYPE_INT64:
		typ = descriptorpb.FieldDescriptorProto_TYPE_INT64
	case typepb.Field_TYPE_SFIXED32:
		typ = descriptorpb.FieldDescriptorProto_TYPE_SFIXED32
	case typepb.Field_TYPE_SFIXED64:
		typ = descriptorpb.FieldDescriptorProto_TYPE_SFIXED64
	case typepb.Field_TYPE_SINT32:
		typ = descriptorpb.FieldDescriptorProto_TYPE_SINT32
	case typepb.Field_TYPE_SINT64:
		typ = descriptorpb.FieldDescriptorProto_TYPE_SINT64
	case typepb.Field_TYPE_UINT32:
		typ = descriptorpb.FieldDescriptorProto_TYPE_UINT32
	case typepb.Field_TYPE_UINT64:
		typ = descriptorpb.FieldDescriptorProto_TYPE_UINT64
	}
	var defaultVal *string
	if f.DefaultValue != "" {
		defaultVal = proto.String(f.DefaultValue)
	}
	return &descriptorpb.FieldDescriptorProto{
		Name:         proto.String(f.Name),
		Number:       proto.Int32(f.Number),
		DefaultValue: defaultVal,
		JsonName:     proto.String(f.JsonName),
		OneofIndex:   oneOf,
		TypeName:     proto.String(typeName),
		Label:        label.Enum(),
		Type:         typ.Enum(),
		Options:      opts,
	}
}

func createServiceDescriptor(a *apipb.Api, res SerializationResolver) *descriptorpb.ServiceDescriptorProto {
	opts := &descriptorpb.ServiceOptions{}
	if len(a.Options) > 0 {
		processOptions(a.Options, opts.ProtoReflect(), res)
	}

	methods := make([]*descriptorpb.MethodDescriptorProto, len(a.Methods))
	for i, m := range a.Methods {
		methods[i] = createMethodDescriptor(m, res)
	}

	return &descriptorpb.ServiceDescriptorProto{
		Name:    proto.String(base(a.Name)),
		Method:  methods,
		Options: opts,
	}
}

func createMethodDescriptor(m *apipb.Method, res SerializationResolver) *descriptorpb.MethodDescriptorProto {
	opts := &descriptorpb.MethodOptions{}
	if len(m.Options) > 0 {
		processOptions(m.Options, opts.ProtoReflect(), res)
	}

	var reqType, respType string
	pos := strings.LastIndex(m.RequestTypeUrl, "/")
	reqType = "." + m.RequestTypeUrl[pos+1:]
	pos = strings.LastIndex(m.ResponseTypeUrl, "/")
	respType = "." + m.ResponseTypeUrl[pos+1:]

	return &descriptorpb.MethodDescriptorProto{
		Name:            proto.String(m.Name),
		Options:         opts,
		ClientStreaming: proto.Bool(m.RequestStreaming),
		ServerStreaming: proto.Bool(m.ResponseStreaming),
		InputType:       proto.String(reqType),
		OutputType:      proto.String(respType),
	}
}

func createIntermediateMessageDescriptor(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: proto.String(name),
	}
}

func createFileDescriptor(name, pkg string, proto3 bool, deps map[string]struct{}) *descriptorpb.FileDescriptorProto {
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
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String(name),
		Package:    proto.String(pkg),
		Syntax:     proto.String(syntax),
		Dependency: imports,
	}
}

func processOptions(options []*typepb.Option, optsMsg protoreflect.Message, res SerializationResolver) {
	// these are created "best effort" so entries which are unresolvable
	// (or seemingly invalid) are simply ignored...
	optsDesc := optsMsg.Descriptor()
	fields := optsDesc.Fields()
	for _, o := range options {
		field := fields.ByName(protoreflect.Name(o.Name))
		if field == nil {
			// must be an extension
			extType, err := res.FindExtensionByName(protoreflect.FullName(o.Name))
			if err != nil {
				continue
			}
			field = extType.TypeDescriptor()
			if field.ContainingMessage() != optsDesc {
				continue
			}
		}
		msgValue := newMessageValueForField(optsMsg, field)
		if msgValue == nil {
			continue
		}
		if TypeNameFromURL(o.Value.TypeUrl) != msgValue.Descriptor().FullName() {
			continue
		}
		err := o.Value.UnmarshalTo(msgValue.Interface())
		if err != nil {
			// can't interpret value? skip it
			continue
		}

		if field.IsMap() {
			// Value is a dynamic message representing entry type. So unpack it.
			k := msgValue.Get(field.MapKey()).MapKey()
			v := msgValue.Get(field.MapValue())
			optsMsg.Mutable(field).Map().Set(k, v)
			continue
		}

		var fv protoreflect.Value
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			fv = unwrap(msgValue.Interface(), field.Kind() == protoreflect.EnumKind)
			if !fv.IsValid() {
				// It should not be possible to get here...
				continue
			}
		} else {
			fv = protoreflect.ValueOfMessage(msgValue)
		}
		if field.IsList() {
			optsMsg.Mutable(field).List().Append(protoreflect.ValueOf(fv))
		} else {
			optsMsg.Set(field, protoreflect.ValueOf(fv))
		}
	}
}

func base(name string) string {
	pos := strings.LastIndex(name, ".")
	if pos >= 0 {
		return name[pos+1:]
	}
	return name
}

func newMessageValueForField(msg protoreflect.Message, field protoreflect.FieldDescriptor) protoreflect.Message {
	isMessageKind := field.Kind() == protoreflect.MessageKind || field.Kind() == protoreflect.GroupKind
	switch {
	case field.IsList() && isMessageKind:
		return msg.NewField(field).List().NewElement().Message()
	case field.IsMap():
		// For maps, create a dynamic message representing the map entry
		return dynamicpb.NewMessage(field.Message())
	case isMessageKind:
		return msg.NewField(field).Message()
	default:
		switch field.Kind() {
		case protoreflect.BoolKind:
			return (&wrapperspb.BoolValue{}).ProtoReflect()
		case protoreflect.FloatKind:
			return (&wrapperspb.FloatValue{}).ProtoReflect()
		case protoreflect.DoubleKind:
			return (&wrapperspb.DoubleValue{}).ProtoReflect()
		case protoreflect.Int32Kind, protoreflect.EnumKind:
			return (&wrapperspb.Int32Value{}).ProtoReflect()
		case protoreflect.Int64Kind:
			return (&wrapperspb.Int64Value{}).ProtoReflect()
		case protoreflect.Uint32Kind:
			return (&wrapperspb.UInt32Value{}).ProtoReflect()
		case protoreflect.Uint64Kind:
			return (&wrapperspb.UInt64Value{}).ProtoReflect()
		case protoreflect.BytesKind:
			return (&wrapperspb.BytesValue{}).ProtoReflect()
		case protoreflect.StringKind:
			return (&wrapperspb.StringValue{}).ProtoReflect()
		}
	}
	return nil
}

func unwrap(msg proto.Message, isEnum bool) protoreflect.Value {
	switch m := msg.(type) {
	case *wrapperspb.BoolValue:
		return protoreflect.ValueOfBool(m.Value)
	case *wrapperspb.FloatValue:
		return protoreflect.ValueOfFloat32(m.Value)
	case *wrapperspb.DoubleValue:
		return protoreflect.ValueOfFloat64(m.Value)
	case *wrapperspb.Int32Value:
		if isEnum {
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(m.Value))
		}
		return protoreflect.ValueOfInt32(m.Value)
	case *wrapperspb.Int64Value:
		return protoreflect.ValueOfInt64(m.Value)
	case *wrapperspb.UInt32Value:
		return protoreflect.ValueOfUint32(m.Value)
	case *wrapperspb.UInt64Value:
		return protoreflect.ValueOfUint64(m.Value)
	case *wrapperspb.BytesValue:
		return protoreflect.ValueOfBytes(m.Value)
	case *wrapperspb.StringValue:
		return protoreflect.ValueOfString(m.Value)
	default:
		return protoreflect.Value{}
	}
}

func protoFromDescriptor(d protoreflect.Descriptor) proto.Message {
	switch d := d.(type) {
	case protoreflect.MessageDescriptor:
		if res, ok := d.(interface {
			MessageDescriptorProto() *descriptorpb.DescriptorProto
		}); ok {
			return res.MessageDescriptorProto()
		}
		if res, ok := d.(interface{ AsProto() proto.Message }); ok {
			if md, ok := res.AsProto().(*descriptorpb.DescriptorProto); ok {
				return md
			}
		}
		return protodesc.ToDescriptorProto(d)
	case protoreflect.EnumDescriptor:
		if res, ok := d.(interface {
			EnumDescriptorProto() *descriptorpb.EnumDescriptorProto
		}); ok {
			return res.EnumDescriptorProto()
		}
		if res, ok := d.(interface{ AsProto() proto.Message }); ok {
			if ed, ok := res.AsProto().(*descriptorpb.EnumDescriptorProto); ok {
				return ed
			}
		}
		return protodesc.ToEnumDescriptorProto(d)
	default:
		return nil
	}
}
