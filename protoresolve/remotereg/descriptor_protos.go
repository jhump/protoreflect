package remotereg

import (
	"sort"
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

func createEnumDescriptor(e *typepb.Enum, res protoresolve.SerializationResolver) *descriptorpb.EnumDescriptorProto {
	var opts *descriptorpb.EnumOptions
	if len(e.Options) > 0 {
		opts = &descriptorpb.EnumOptions{}
		processOptions(e.Options, opts.ProtoReflect(), res)
	}

	var vals []*descriptorpb.EnumValueDescriptorProto
	for _, v := range e.Enumvalue {
		evd := createEnumValueDescriptor(v, res)
		vals = append(vals, evd)
	}

	return &descriptorpb.EnumDescriptorProto{
		Name:    new(base(e.Name)),
		Options: opts,
		Value:   vals,
	}
}

func createEnumValueDescriptor(v *typepb.EnumValue, res protoresolve.SerializationResolver) *descriptorpb.EnumValueDescriptorProto {
	var opts *descriptorpb.EnumValueOptions
	if len(v.Options) > 0 {
		opts = &descriptorpb.EnumValueOptions{}
		processOptions(v.Options, opts.ProtoReflect(), res)
	}

	return &descriptorpb.EnumValueDescriptorProto{
		Name:    new(v.Name),
		Number:  new(v.Number),
		Options: opts,
	}
}

func createMessageDescriptor(m *typepb.Type, res protoresolve.SerializationResolver) *descriptorpb.DescriptorProto {
	var opts *descriptorpb.MessageOptions
	if len(m.Options) > 0 {
		opts = &descriptorpb.MessageOptions{}
		processOptions(m.Options, opts.ProtoReflect(), res)
	}

	var fields []*descriptorpb.FieldDescriptorProto
	for _, f := range m.Fields {
		fields = append(fields, createFieldDescriptor(f, res))
	}

	var oneOfs []*descriptorpb.OneofDescriptorProto
	for _, o := range m.Oneofs {
		oneOfs = append(oneOfs, &descriptorpb.OneofDescriptorProto{
			Name: new(o),
		})
	}

	return &descriptorpb.DescriptorProto{
		Name:      new(base(m.Name)),
		Options:   opts,
		Field:     fields,
		OneofDecl: oneOfs,
	}
}

func createFieldDescriptor(f *typepb.Field, res protoresolve.SerializationResolver) *descriptorpb.FieldDescriptorProto {
	var opts *descriptorpb.FieldOptions
	if len(f.Options) > 0 {
		opts = &descriptorpb.FieldOptions{}
		processOptions(f.Options, opts.ProtoReflect(), res)
	}
	if f.Packed {
		if opts == nil {
			opts = &descriptorpb.FieldOptions{Packed: new(true)}
		} else {
			opts.Packed = new(true)
		}
	}

	var oneOf *int32
	if f.OneofIndex > 0 {
		oneOf = new(f.OneofIndex - 1)
	}

	var typeName *string
	if f.Kind == typepb.Field_TYPE_GROUP || f.Kind == typepb.Field_TYPE_MESSAGE || f.Kind == typepb.Field_TYPE_ENUM {
		pos := strings.LastIndex(f.TypeUrl, "/")
		typeName = new("." + f.TypeUrl[pos+1:])
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
		defaultVal = new(f.DefaultValue)
	}
	return &descriptorpb.FieldDescriptorProto{
		Name:         new(f.Name),
		Number:       new(f.Number),
		DefaultValue: defaultVal,
		JsonName:     new(f.JsonName),
		OneofIndex:   oneOf,
		TypeName:     typeName,
		Label:        new(label),
		Type:         new(typ),
		Options:      opts,
	}
}

func createServiceDescriptor(a *apipb.Api, res protoresolve.SerializationResolver) *descriptorpb.ServiceDescriptorProto {
	var opts *descriptorpb.ServiceOptions
	if len(a.Options) > 0 {
		opts = &descriptorpb.ServiceOptions{}
		processOptions(a.Options, opts.ProtoReflect(), res)
	}

	methods := make([]*descriptorpb.MethodDescriptorProto, len(a.Methods))
	for i, m := range a.Methods {
		methods[i] = createMethodDescriptor(m, res)
	}

	return &descriptorpb.ServiceDescriptorProto{
		Name:    new(base(a.Name)),
		Method:  methods,
		Options: opts,
	}
}

func createMethodDescriptor(m *apipb.Method, res protoresolve.SerializationResolver) *descriptorpb.MethodDescriptorProto {
	var opts *descriptorpb.MethodOptions
	if len(m.Options) > 0 {
		opts = &descriptorpb.MethodOptions{}
		processOptions(m.Options, opts.ProtoReflect(), res)
	}

	var reqType, respType string
	pos := strings.LastIndex(m.RequestTypeUrl, "/")
	reqType = "." + m.RequestTypeUrl[pos+1:]
	pos = strings.LastIndex(m.ResponseTypeUrl, "/")
	respType = "." + m.ResponseTypeUrl[pos+1:]

	return &descriptorpb.MethodDescriptorProto{
		Name:            new(m.Name),
		Options:         opts,
		ClientStreaming: new(m.RequestStreaming),
		ServerStreaming: new(m.ResponseStreaming),
		InputType:       new(reqType),
		OutputType:      new(respType),
	}
}

func createIntermediateMessageDescriptor(name string) *descriptorpb.DescriptorProto {
	return &descriptorpb.DescriptorProto{
		Name: new(name),
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
		Name:       new(name),
		Package:    new(pkg),
		Syntax:     new(syntax),
		Dependency: imports,
	}
}

func base(name string) string {
	pos := strings.LastIndex(name, ".")
	if pos >= 0 {
		return name[pos+1:]
	}
	return name
}
