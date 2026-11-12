package remotereg_test

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	_ "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	_ "google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	. "github.com/jhump/protoreflect/v2/protoresolve/remotereg"
)

func TestDescriptorConverter_ToServiceDescriptorKeepsNestedTypes(t *testing.T) {
	t.Parallel()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("req.proto"),
		Package: proto.String("foo"),
		Syntax:  proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Request"),
			Field: []*descriptorpb.FieldDescriptorProto{{
				Name:     proto.String("used"),
				Number:   proto.Int32(1),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".foo.Request.Used"),
				JsonName: proto.String("used"),
			}},
			NestedType: []*descriptorpb.DescriptorProto{
				{Name: proto.String("Used")},
				// Nothing refers to this one, but it must still be retained.
				{Name: proto.String("Unused")},
			},
		}},
	}
	fd, err := protodesc.NewFile(fdp, nil)
	require.NoError(t, err)
	var registry Registry
	require.NoError(t, registry.RegisterMessage(fd.Messages().Get(0)))

	api := &apipb.Api{
		Name: "foo.Service",
		Methods: []*apipb.Method{{
			Name:            "Method",
			RequestTypeUrl:  "type.googleapis.com/foo.Request",
			ResponseTypeUrl: "type.googleapis.com/foo.Request",
		}},
		Syntax: typepb.Syntax_SYNTAX_PROTO3,
	}
	sd, err := registry.AsDescriptorConverter().ToServiceDescriptor(context.Background(), api)
	require.NoError(t, err)
	input := sd.Methods().Get(0).Input()
	require.Equal(t, 2, input.Messages().Len())
	require.NotNil(t, input.Messages().ByName("Used"))
	require.NotNil(t, input.Messages().ByName("Unused"))
}

func TestDescriptorConverter_ToServiceDescriptor(t *testing.T) {
	tf := createFetcher(t)
	// we want "defaults" for the message factory so that we can properly process
	// known extensions (which the type fetcher puts into the descriptor options)
	rr := &Registry{TypeFetcher: tf}
	dc := rr.AsDescriptorConverter()

	sd, err := dc.ToServiceDescriptor(context.Background(), getAPI(t))
	require.NoError(t, err)

	require.Equal(t, "Service", string(sd.Name()))
	require.Equal(t, "some.Service", string(sd.FullName()))
	require.Equal(t, "some", string(sd.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, sd.ParentFile().Syntax())

	so := &descriptorpb.ServiceOptions{
		Deprecated: new(true),
	}
	proto.SetExtension(so, testprotos.E_Sfubar, &testprotos.ReallySimpleMessage{Id: new(uint64(100)), Name: new("deuce")})
	proto.SetExtension(so, testprotos.E_Sfubare, testprotos.ReallySimpleEnum_VALUE)
	protosEqual(t, so, sd.Options())

	methods := sd.Methods()
	require.Equal(t, 4, methods.Len())
	require.Equal(t, "UnaryMethod", string(methods.Get(0).Name()))
	require.Equal(t, "some.Type", string(methods.Get(0).Input().FullName()))
	require.Equal(t, "some.OtherType", string(methods.Get(0).Output().FullName()))

	mto := &descriptorpb.MethodOptions{
		Deprecated: new(true),
	}
	proto.SetExtension(mto, testprotos.E_Mtfubar, []float32{3.14159, 2.71828})
	proto.SetExtension(mto, testprotos.E_Mtfubard, 10203040.506070809)
	protosEqual(t, mto, methods.Get(0).Options())

	require.Equal(t, "ClientStreamMethod", string(methods.Get(1).Name()))
	require.Equal(t, "some.OtherType", string(methods.Get(1).Input().FullName()))
	require.Equal(t, "some.Type", string(methods.Get(1).Output().FullName()))

	require.Equal(t, "ServerStreamMethod", string(methods.Get(2).Name()))
	require.Equal(t, "some.OtherType.AnotherType", string(methods.Get(2).Input().FullName()))
	require.Equal(t, "some.YetAnother.MessageType", string(methods.Get(2).Output().FullName()))

	require.Equal(t, "BidiStreamMethod", string(methods.Get(3).Name()))
	require.Equal(t, "some.YetAnother.MessageType", string(methods.Get(3).Input().FullName()))
	require.Equal(t, "some.OtherType.AnotherType", string(methods.Get(3).Output().FullName()))

	// check linked message types

	require.Equal(t, methods.Get(0).Input(), methods.Get(1).Output())
	require.Equal(t, methods.Get(0).Output(), methods.Get(1).Input())
	require.Equal(t, methods.Get(2).Input(), methods.Get(3).Output())
	require.Equal(t, methods.Get(2).Output(), methods.Get(3).Input())

	md1 := methods.Get(0).Input()
	md2 := methods.Get(0).Output()
	md3 := methods.Get(2).Input()
	md4 := methods.Get(2).Output()

	require.Equal(t, "Type", string(md1.Name()))
	require.Equal(t, "some.Type", string(md1.FullName()))
	require.Equal(t, "some", string(md1.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, md1.ParentFile().Syntax())

	require.Equal(t, "OtherType", string(md2.Name()))
	require.Equal(t, "some.OtherType", string(md2.FullName()))
	require.Equal(t, "some", string(md2.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto2, md2.ParentFile().Syntax())

	require.Equal(t, md3, md2.Messages().Get(0))
	require.Equal(t, "AnotherType", string(md3.Name()))
	require.Equal(t, "some.OtherType.AnotherType", string(md3.FullName()))
	require.Equal(t, "some", string(md3.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto2, md3.ParentFile().Syntax())

	require.Equal(t, "MessageType", string(md4.Name()))
	require.Equal(t, "some.YetAnother.MessageType", string(md4.FullName()))
	require.Equal(t, "some", string(md4.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, md4.ParentFile().Syntax())
}

func getAPI(t *testing.T) *apipb.Api {
	var bol anypb.Any
	err := anypb.MarshalFrom(&bol, &wrapperspb.BoolValue{Value: true}, proto.MarshalOptions{})
	require.NoError(t, err)
	var dbl anypb.Any
	err = anypb.MarshalFrom(&dbl, &wrapperspb.DoubleValue{Value: 10203040.506070809}, proto.MarshalOptions{})
	require.NoError(t, err)
	var flt1 anypb.Any
	err = anypb.MarshalFrom(&flt1, &wrapperspb.FloatValue{Value: 3.14159}, proto.MarshalOptions{})
	require.NoError(t, err)
	var flt2 anypb.Any
	err = anypb.MarshalFrom(&flt2, &wrapperspb.FloatValue{Value: 2.71828}, proto.MarshalOptions{})
	require.NoError(t, err)
	var enu anypb.Any
	err = anypb.MarshalFrom(&enu, &wrapperspb.Int32Value{Value: int32(testprotos.ReallySimpleEnum_VALUE)}, proto.MarshalOptions{})
	require.NoError(t, err)
	var msg anypb.Any
	err = anypb.MarshalFrom(&msg, &testprotos.ReallySimpleMessage{Id: new(uint64(100)), Name: new("deuce")}, proto.MarshalOptions{})
	require.NoError(t, err)
	return &apipb.Api{
		Name: "some.Service",
		Methods: []*apipb.Method{
			{
				Name:            "UnaryMethod",
				RequestTypeUrl:  "foo.bar/some.Type",
				ResponseTypeUrl: "foo.bar/some.OtherType",
				Options: []*typepb.Option{
					{
						Name:  "deprecated",
						Value: &bol,
					},
					{
						Name:  "testprotos.mtfubar",
						Value: &flt1,
					},
					{
						Name:  "testprotos.mtfubar",
						Value: &flt2,
					},
					{
						Name:  "testprotos.mtfubard",
						Value: &dbl,
					},
				},
				Syntax: typepb.Syntax_SYNTAX_PROTO3,
			},
			{
				Name:             "ClientStreamMethod",
				RequestStreaming: true,
				RequestTypeUrl:   "foo.bar/some.OtherType",
				ResponseTypeUrl:  "foo.bar/some.Type",
				Syntax:           typepb.Syntax_SYNTAX_PROTO3,
			},
			{
				Name:              "ServerStreamMethod",
				ResponseStreaming: true,
				RequestTypeUrl:    "foo.bar/some.OtherType.AnotherType",
				ResponseTypeUrl:   "foo.bar/some.YetAnother.MessageType",
				Syntax:            typepb.Syntax_SYNTAX_PROTO3,
			},
			{
				Name:              "BidiStreamMethod",
				RequestStreaming:  true,
				ResponseStreaming: true,
				RequestTypeUrl:    "foo.bar/some.YetAnother.MessageType",
				ResponseTypeUrl:   "foo.bar/some.OtherType.AnotherType",
				Syntax:            typepb.Syntax_SYNTAX_PROTO3,
			},
		},
		Options: []*typepb.Option{
			{
				Name:  "deprecated",
				Value: &bol,
			},
			{
				Name:  "testprotos.sfubar",
				Value: &msg,
			},
			{
				Name:  "testprotos.sfubare",
				Value: &enu,
			},
		},
		SourceContext: &sourcecontextpb.SourceContext{FileName: "baz.proto"},
		Syntax:        typepb.Syntax_SYNTAX_PROTO3,
	}
}

func TestDescriptorConverter_DescriptorAsAPI(t *testing.T) {
	svcOpts := &descriptorpb.ServiceOptions{
		Deprecated: new(true),
	}
	proto.SetExtension(svcOpts, testprotos.E_Sfubar, &testprotos.ReallySimpleMessage{Id: new(uint64(1234)), Name: new("abc")})
	proto.SetExtension(svcOpts, testprotos.E_Sfubare, testprotos.ReallySimpleEnum_VALUE)
	mtdOpts := &descriptorpb.MethodOptions{
		Deprecated: new(true),
	}
	proto.SetExtension(mtdOpts, testprotos.E_Mtfubar, []float32{0, 102.3040506, float32(math.Inf(-1)), 2030.40506})
	proto.SetExtension(mtdOpts, testprotos.E_Mtfubard, -98765.4321)
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    new("test.proto"),
		Syntax:  new("proto3"),
		Package: new("foo"),
		Dependency: []string{
			"google/protobuf/empty.proto",
			"desc_test_options.proto",
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name:    new("FooService"),
				Options: svcOpts,
				Method: []*descriptorpb.MethodDescriptorProto{
					{
						Name:            new("Do"),
						Options:         mtdOpts,
						InputType:       new(".foo.Request"),
						ClientStreaming: new(true),
						OutputType:      new(".google.protobuf.Empty"),
					},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: new("Request"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     new("id"),
						Number:   new(int32(1)),
						Label:    new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						JsonName: new("id"),
					},
				},
			},
		},
	}
	fd, err := protodesc.NewFile(fdp, protoregistry.GlobalFiles)
	require.NoError(t, err)

	rr := &Registry{DefaultBaseURL: "foo.com"}
	api := rr.AsDescriptorConverter().DescriptorAsAPI(fd.Services().Get(0))

	expected := &apipb.Api{
		Name:   "foo.FooService",
		Syntax: typepb.Syntax_SYNTAX_PROTO3,
		SourceContext: &sourcecontextpb.SourceContext{
			FileName: "test.proto",
		},
		Options: []*typepb.Option{
			{Name: "deprecated", Value: asAny(t, &wrapperspb.BoolValue{Value: true})},
			{Name: "testprotos.sfubar", Value: asAny(t, &testprotos.ReallySimpleMessage{Id: new(uint64(1234)), Name: new("abc")})},
			{Name: "testprotos.sfubare", Value: asAny(t, &wrapperspb.Int32Value{Value: int32(testprotos.ReallySimpleEnum_VALUE)})},
		},
		Methods: []*apipb.Method{
			{
				Name:   "Do",
				Syntax: typepb.Syntax_SYNTAX_PROTO3,
				Options: []*typepb.Option{
					{Name: "deprecated", Value: asAny(t, &wrapperspb.BoolValue{Value: true})},
					{Name: "testprotos.mtfubar", Value: asAny(t, &wrapperspb.FloatValue{Value: 0})},
					{Name: "testprotos.mtfubar", Value: asAny(t, &wrapperspb.FloatValue{Value: 102.3040506})},
					{Name: "testprotos.mtfubar", Value: asAny(t, &wrapperspb.FloatValue{Value: float32(math.Inf(-1))})},
					{Name: "testprotos.mtfubar", Value: asAny(t, &wrapperspb.FloatValue{Value: 2030.40506})},
					{Name: "testprotos.mtfubard", Value: asAny(t, &wrapperspb.DoubleValue{Value: -98765.4321})},
				},
				RequestStreaming: true,
				RequestTypeUrl:   "foo.com/foo.Request",
				ResponseTypeUrl:  "foo.com/google.protobuf.Empty",
			},
		},
	}

	protosEqual(t, expected, api)
}

func TestDescriptorConverter_ToMessageDescriptor(t *testing.T) {
	tf := createFetcher(t)
	msg, err := tf.FetchMessageType(context.Background(), "https://foo.bar/some.Type")
	require.NoError(t, err)

	md, err := (&Registry{TypeFetcher: tf}).AsDescriptorConverter().ToMessageDescriptor(context.Background(), msg)
	require.NoError(t, err)

	require.Equal(t, "foo.proto", md.ParentFile().Path())
	require.Equal(t, "some", string(md.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, md.ParentFile().Syntax())
	mdProto := protodesc.ToDescriptorProto(md)

	msgOpts := &descriptorpb.MessageOptions{
		Deprecated: new(true),
	}
	proto.SetExtension(msgOpts, testprotos.E_Mfubar, true)
	fldOpts := &descriptorpb.FieldOptions{
		Deprecated: new(true),
	}
	proto.SetExtension(fldOpts, testprotos.E_Ffubar, []string{"foo", "bar", "baz"})
	proto.SetExtension(fldOpts, testprotos.E_Ffubarb, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	expected := &descriptorpb.DescriptorProto{
		Name:    new("Type"),
		Options: msgOpts,
		OneofDecl: []*descriptorpb.OneofDescriptorProto{
			{
				Name: new("un"),
			},
		},
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:     new("a"),
				Label:    new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
				Type:     new(descriptorpb.FieldDescriptorProto_TYPE_MESSAGE),
				TypeName: new(".some.OtherType"),
				Number:   new(int32(1)),
				Options:  fldOpts,
				JsonName: new("a"),
			},
			{
				Name:     new("b"),
				Label:    new(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
				Type:     new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
				Number:   new(int32(2)),
				JsonName: new("b"),
			},
			{
				Name:       new("c"),
				Label:      new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
				Type:       new(descriptorpb.FieldDescriptorProto_TYPE_ENUM),
				TypeName:   new(".some.Enum"),
				Number:     new(int32(3)),
				OneofIndex: new(int32(0)),
				JsonName:   new("c"),
			},
			{
				Name:       new("d"),
				Label:      new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
				Type:       new(descriptorpb.FieldDescriptorProto_TYPE_INT32),
				Number:     new(int32(4)),
				OneofIndex: new(int32(0)),
				JsonName:   new("d"),
			},
		},
	}

	protosEqual(t, expected, mdProto)
}

func TestDescriptorConverter_DescriptorAsType(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    new("test.proto"),
		Syntax:  new("proto2"),
		Package: new("foo"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: new("Bar"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{
						Name: new("oo"),
					},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     new("abc"),
						Number:   new(int32(1)),
						Label:    new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:     new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						Options:  &descriptorpb.FieldOptions{Deprecated: new(true)},
						JsonName: new("abc"),
					},
					{
						Name:     new("def"),
						Number:   new(int32(2)),
						Label:    new(descriptorpb.FieldDescriptorProto_LABEL_REPEATED),
						Type:     new(descriptorpb.FieldDescriptorProto_TYPE_INT32),
						Options:  &descriptorpb.FieldOptions{Packed: new(true)},
						JsonName: new("def"),
					},
					{
						Name:         new("ghi"),
						Number:       new(int32(3)),
						Label:        new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:         new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						DefaultValue: new("foobar"),
						JsonName:     new("ghi"),
					},
					{
						Name:       new("nid"),
						Number:     new(int32(4)),
						Label:      new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:       new(descriptorpb.FieldDescriptorProto_TYPE_UINT64),
						JsonName:   new("nid"),
						OneofIndex: new(int32(0)),
					},
					{
						Name:       new("sid"),
						Number:     new(int32(5)),
						Label:      new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
						Type:       new(descriptorpb.FieldDescriptorProto_TYPE_STRING),
						JsonName:   new("_SID_"),
						OneofIndex: new(int32(0)),
					},
				},
			},
		},
	}
	fd, err := protodesc.NewFile(fdp, nil)
	require.NoError(t, err)

	msg := (&Registry{}).AsDescriptorConverter().DescriptorAsType(fd.Messages().Get(0))

	expected := &typepb.Type{
		Name:   "foo.Bar",
		Syntax: typepb.Syntax_SYNTAX_PROTO2,
		SourceContext: &sourcecontextpb.SourceContext{
			FileName: "test.proto",
		},
		Oneofs: []string{"oo"},
		Fields: []*typepb.Field{
			{
				Name:        "abc",
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				Kind:        typepb.Field_TYPE_STRING,
				Number:      1,
				Options: []*typepb.Option{
					{Name: "deprecated", Value: asAny(t, &wrapperspb.BoolValue{Value: true})},
				},
				JsonName: "abc",
			},
			{
				Name:        "def",
				Cardinality: typepb.Field_CARDINALITY_REPEATED,
				Kind:        typepb.Field_TYPE_INT32,
				Number:      2,
				Packed:      true,
				JsonName:    "def",
			},
			{
				Name:         "ghi",
				Cardinality:  typepb.Field_CARDINALITY_OPTIONAL,
				Kind:         typepb.Field_TYPE_STRING,
				Number:       3,
				DefaultValue: "foobar",
				JsonName:     "ghi",
			},
			{
				Name:        "nid",
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				Kind:        typepb.Field_TYPE_UINT64,
				Number:      4,
				OneofIndex:  1,
				JsonName:    "nid",
			},
			{
				Name:        "sid",
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				Kind:        typepb.Field_TYPE_STRING,
				Number:      5,
				OneofIndex:  1,
				JsonName:    "_SID_",
			},
		},
	}

	protosEqual(t, expected, msg)
}

func TestDescriptorConverter_ToEnumDescriptor(t *testing.T) {
	tf := createFetcher(t)
	enum, err := tf.FetchEnumType(context.Background(), "https://foo.bar/some.Enum")
	require.NoError(t, err)

	ed, err := (&Registry{TypeFetcher: tf}).AsDescriptorConverter().ToEnumDescriptor(context.Background(), enum)
	require.NoError(t, err)

	require.Equal(t, "foo.proto", ed.ParentFile().Path())
	require.Equal(t, "some", string(ed.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, ed.ParentFile().Syntax())
	edProto := protodesc.ToEnumDescriptorProto(ed)

	enumOpts := &descriptorpb.EnumOptions{
		Deprecated: new(true),
		AllowAlias: new(true),
	}
	proto.SetExtension(enumOpts, testprotos.E_Efubar, int32(-42))
	proto.SetExtension(enumOpts, testprotos.E_Efubars, int32(-42))
	proto.SetExtension(enumOpts, testprotos.E_Efubarsf, int32(-42))
	proto.SetExtension(enumOpts, testprotos.E_Efubaru, uint32(42))
	proto.SetExtension(enumOpts, testprotos.E_Efubaruf, uint32(42))
	enumValOpts := &descriptorpb.EnumValueOptions{
		Deprecated: new(true),
	}
	proto.SetExtension(enumValOpts, testprotos.E_Evfubar, int64(-420420420420))
	proto.SetExtension(enumValOpts, testprotos.E_Evfubars, int64(-420420420420))
	proto.SetExtension(enumValOpts, testprotos.E_Evfubarsf, int64(-420420420420))
	proto.SetExtension(enumValOpts, testprotos.E_Evfubaru, uint64(420420420420))
	proto.SetExtension(enumValOpts, testprotos.E_Evfubaruf, uint64(420420420420))
	expected := &descriptorpb.EnumDescriptorProto{
		Name:    new("Enum"),
		Options: enumOpts,
		Value: []*descriptorpb.EnumValueDescriptorProto{
			{
				Name:    new("ABC"),
				Number:  new(int32(0)),
				Options: enumValOpts,
			},
			{
				Name:   new("XYZ"),
				Number: new(int32(1)),
			},
			{
				Name:   new("WXY"),
				Number: new(int32(1)),
			},
		},
	}

	protosEqual(t, expected, edProto)
}

func TestDescriptorConverter_DescriptorAsEnum(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    new("test.proto"),
		Syntax:  new("proto2"),
		Package: new("foo"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name:    new("Bar"),
				Options: &descriptorpb.EnumOptions{AllowAlias: new(true)},
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:   new("ZERO"),
						Number: new(int32(0)),
					},
					{
						Name:    new("__UNSET__"),
						Number:  new(int32(0)),
						Options: &descriptorpb.EnumValueOptions{Deprecated: new(true)},
					},
					{
						Name:   new("ONE"),
						Number: new(int32(1)),
					},
					{
						Name:   new("TWO"),
						Number: new(int32(2)),
					},
					{
						Name:   new("THREE"),
						Number: new(int32(3)),
					},
				},
			},
		},
	}
	fd, err := protodesc.NewFile(fdp, nil)
	require.NoError(t, err)

	enum := (&Registry{}).AsDescriptorConverter().DescriptorAsEnum(fd.Enums().Get(0))

	expected := &typepb.Enum{
		Name:   "foo.Bar",
		Syntax: typepb.Syntax_SYNTAX_PROTO2,
		SourceContext: &sourcecontextpb.SourceContext{
			FileName: "test.proto",
		},
		Options: []*typepb.Option{
			{Name: "allow_alias", Value: asAny(t, &wrapperspb.BoolValue{Value: true})},
		},
		Enumvalue: []*typepb.EnumValue{
			{
				Name:   "ZERO",
				Number: 0,
			},
			{
				Name:   "__UNSET__",
				Number: 0,
				Options: []*typepb.Option{
					{Name: "deprecated", Value: asAny(t, &wrapperspb.BoolValue{Value: true})},
				},
			},
			{
				Name:   "ONE",
				Number: 1,
			},
			{
				Name:   "TWO",
				Number: 2,
			},
			{
				Name:   "THREE",
				Number: 3,
			},
		},
	}

	protosEqual(t, expected, enum)
}

func asAny(t *testing.T, msg proto.Message) *anypb.Any {
	var a anypb.Any
	err := anypb.MarshalFrom(&a, msg, proto.MarshalOptions{})
	require.NoError(t, err)
	return &a
}
