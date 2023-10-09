package protoresolve_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/jhump/protoreflect/v2/internal/testdata"
	. "github.com/jhump/protoreflect/v2/protoresolve"
)

func TestMessageRegistry_LookupTypes(t *testing.T) {
	rr := &RemoteRegistry{}

	// register some types
	md := (*descriptorpb.DescriptorProto)(nil).ProtoReflect().Descriptor()
	err := rr.RegisterMessageWithURL(md, "foo.bar/google.protobuf.DescriptorProto")
	require.NoError(t, err)
	ed := md.ParentFile().Messages().ByName("FieldDescriptorProto").Enums().ByName("Type")
	require.NotNil(t, ed)
	err = rr.RegisterEnumWithURL(ed, "foo.bar/google.protobuf.FieldDescriptorProto.Type")
	require.NoError(t, err)

	// lookups succeed
	msg, err := rr.FindMessageByURL("foo.bar/google.protobuf.DescriptorProto")
	require.NoError(t, err)
	require.Equal(t, md, msg)
	require.Equal(t, "https://foo.bar/google.protobuf.DescriptorProto", rr.URLForType(md))
	en, err := rr.FindEnumByURL("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	require.NoError(t, err)
	require.Equal(t, ed, en)
	require.Equal(t, "https://foo.bar/google.protobuf.FieldDescriptorProto.Type", rr.URLForType(ed))

	// right name but wrong domain? not found
	msg, err = rr.FindMessageByURL("type.googleapis.com/google.protobuf.DescriptorProto")
	require.NoError(t, err)
	require.Nil(t, msg)
	en, err = rr.FindEnumByURL("type.googleapis.com/google.protobuf.FieldDescriptorProto.Type")
	require.NoError(t, err)
	require.Nil(t, en == nil)

	// wrong type
	_, err = rr.FindMessageByURL("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	_, ok := err.(*ErrUnexpectedType)
	require.True(t, ok)
	_, err = rr.FindEnumByURL("foo.bar/google.protobuf.DescriptorProto")
	_, ok = err.(*ErrUnexpectedType)
	require.True(t, ok)

	// unmarshal any successfully finds the registered type
	b, err := proto.Marshal(md.AsProto())
	require.NoError(t, err)
	a := &anypb.Any{TypeUrl: "foo.bar/google.protobuf.DescriptorProto", Value: b}
	pm, err := rr.UnmarshalAny(a)
	require.NoError(t, err)
	protosEqual(t, md.AsProto(), pm)
	// we didn't configure the registry with a message factory, so it would have
	// produced a dynamic message instead of a generated message
	require.Equal(t, reflect.TypeOf((*dynamicpb.Message)(nil)), reflect.TypeOf(pm))

	// by default, message registry knows about well-known types
	dur := &durationpb.Duration{Nanos: 100, Seconds: 1000}
	b, err = proto.Marshal(dur)
	require.NoError(t, err)
	a = &anypb.Any{TypeUrl: "foo.bar/google.protobuf.Duration", Value: b}
	pm, err = rr.UnmarshalAny(a)
	require.NoError(t, err)
	protosEqual(t, dur, pm)
	require.Equal(t, reflect.TypeOf((*durationpb.Duration)(nil)), reflect.TypeOf(pm))

	fd, err := protoregistry.GlobalFiles.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)
	rr.AddFile("frob.nitz/foo.bar", fd)
	msgCount, enumCount := 0, 0
	mds := fd.GetMessageTypes()
	for i := 0; i < len(mds); i++ {
		md := mds[i]
		msgCount++
		mds = append(mds, md.GetNestedMessageTypes()...)
		exp := fmt.Sprintf("https://frob.nitz/foo.bar/%s", md.GetFullyQualifiedName())
		require.Equal(t, exp, rr.ComputeURL(md))
		for _, ed := range md.GetNestedEnumTypes() {
			enumCount++
			exp := fmt.Sprintf("https://frob.nitz/foo.bar/%s", ed.GetFullyQualifiedName())
			require.Equal(t, exp, rr.ComputeURL(ed))
		}
	}
	for _, ed := range fd.GetEnumTypes() {
		enumCount++
		exp := fmt.Sprintf("https://frob.nitz/foo.bar/%s", ed.GetFullyQualifiedName())
		require.Equal(t, exp, rr.ComputeURL(ed))
	}
	// sanity check
	require.Equal(t, 11, msgCount)
	require.Equal(t, 2, enumCount)
}

func TestMessageRegistry_LookupTypes_WithDefaults(t *testing.T) {
	mr := NewMessageRegistryWithDefaults()

	md := (*descriptorpb.DescriptorProto)(nil).ProtoReflect().Descriptor()
	ed := md.GetFile().FindEnum("google.protobuf.FieldDescriptorProto.Type")
	require.NotNil(t, ed)

	// lookups succeed
	msg, err := mr.FindMessageTypeByUrl("type.googleapis.com/google.protobuf.DescriptorProto")
	require.NoError(t, err)
	require.Equal(t, md, msg)
	// default types don't know their base URL, so will resolve even w/ wrong name
	// (just have to get fully-qualified message name right)
	msg, err = mr.FindMessageTypeByUrl("foo.bar/google.protobuf.DescriptorProto")
	require.NoError(t, err)
	require.Equal(t, md, msg)

	// sad trombone: no way to lookup "default" enum types, so enums don't resolve
	// without being explicitly registered :(
	en, err := mr.FindEnumTypeByUrl("type.googleapis.com/google.protobuf.FieldDescriptorProto.Type")
	require.NoError(t, err)
	require.Nil(t, en)
	en, err = mr.FindEnumTypeByUrl("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	require.NoError(t, err)
	require.Nil(t, en)

	// unmarshal any successfully finds the registered type
	b, err := proto.Marshal(md.AsProto())
	require.NoError(t, err)
	a := &anypb.Any{TypeUrl: "foo.bar/google.protobuf.DescriptorProto", Value: b}
	pm, err := mr.UnmarshalAny(a)
	require.NoError(t, err)
	protosEqual(t, md.AsProto(), pm)
	// message registry with defaults implies known-type registry with defaults, so
	// it should have marshalled the message into a generated message
	require.Equal(t, reflect.TypeOf((*descriptorpb.DescriptorProto)(nil)), reflect.TypeOf(pm))
}

func TestMessageRegistry_FindMessage_WithFetcher(t *testing.T) {
	tf := createFetcher(t)
	// we want "defaults" for the message factory so that we can properly process
	// known extensions (which the type fetcher puts into the descriptor options)
	mr := &RemoteRegistry{TypeFetcher: tf}

	md, err := mr.FindMessageTypeByUrl("foo.bar/some.Type")
	require.NoError(t, err)

	// Fairly in-depth check of the returned message descriptor:

	require.Equal(t, "Type", md.GetName())
	require.Equal(t, "some.Type", md.GetFullyQualifiedName())
	require.Equal(t, "some", md.GetFile().GetPackage())
	require.Equal(t, true, md.GetFile().IsProto3())
	require.Equal(t, true, md.IsProto3())

	mo := &descriptorpb.MessageOptions{
		Deprecated: proto.Bool(true),
	}
	err = proto.SetExtension(mo, testdata.E_Mfubar, proto.Bool(true))
	require.NoError(t, err)
	protosEqual(t, mo, md.GetMessageOptions())

	flds := md.GetFields()
	require.Equal(t, 4, len(flds))
	require.Equal(t, "a", flds[0].GetName())
	require.Equal(t, int32(1), flds[0].GetNumber())
	require.Nil(t, flds[0].GetOneOf())
	require.Equal(t, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, flds[0].GetLabel())
	require.Equal(t, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, flds[0].GetType())

	fo := &descriptorpb.FieldOptions{
		Deprecated: proto.Bool(true),
	}
	err = proto.SetExtension(fo, testdata.E_Ffubar, []string{"foo", "bar", "baz"})
	require.NoError(t, err)
	err = proto.SetExtension(fo, testdata.E_Ffubarb, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	require.NoError(t, err)
	protosEqual(t, fo, flds[0].GetFieldOptions())

	require.Equal(t, "b", flds[1].GetName())
	require.Equal(t, int32(2), flds[1].GetNumber())
	require.Nil(t, flds[1].GetOneOf())
	require.Equal(t, descriptorpb.FieldDescriptorProto_LABEL_REPEATED, flds[1].GetLabel())
	require.Equal(t, descriptorpb.FieldDescriptorProto_TYPE_STRING, flds[1].GetType())

	require.Equal(t, "c", flds[2].GetName())
	require.Equal(t, int32(3), flds[2].GetNumber())
	require.Equal(t, "un", flds[2].GetOneOf().GetName())
	require.Equal(t, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, flds[2].GetLabel())
	require.Equal(t, descriptorpb.FieldDescriptorProto_TYPE_ENUM, flds[2].GetType())

	require.Equal(t, "d", flds[3].GetName())
	require.Equal(t, int32(4), flds[3].GetNumber())
	require.Equal(t, "un", flds[3].GetOneOf().GetName())
	require.Equal(t, descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL, flds[3].GetLabel())
	require.Equal(t, descriptorpb.FieldDescriptorProto_TYPE_INT32, flds[3].GetType())

	oos := md.GetOneOfs()
	require.Equal(t, 1, len(oos))
	require.Equal(t, "un", oos[0].GetName())
	ooflds := oos[0].GetChoices()
	require.Equal(t, 2, len(ooflds))
	require.Equal(t, flds[2], ooflds[0])
	require.Equal(t, flds[3], ooflds[1])

	// Quick, shallow check of the linked descriptors:

	md2 := md.FindFieldByName("a").GetMessageType()
	require.Equal(t, "OtherType", md2.GetName())
	require.Equal(t, "some.OtherType", md2.GetFullyQualifiedName())
	require.Equal(t, "some", md2.GetFile().GetPackage())
	require.Equal(t, false, md2.GetFile().IsProto3())
	require.Equal(t, false, md2.IsProto3())

	nmd := md2.GetNestedMessageTypes()[0]
	protosEqual(t, nmd.AsProto(), md2.FindFieldByName("a").GetMessageType().AsProto())
	require.Equal(t, "AnotherType", nmd.GetName())
	require.Equal(t, "some.OtherType.AnotherType", nmd.GetFullyQualifiedName())
	require.Equal(t, "some", nmd.GetFile().GetPackage())
	require.Equal(t, false, nmd.GetFile().IsProto3())
	require.Equal(t, false, nmd.IsProto3())

	en := md.FindFieldByName("c").GetEnumType()
	require.Equal(t, "Enum", en.GetName())
	require.Equal(t, "some.Enum", en.GetFullyQualifiedName())
	require.Equal(t, "some", en.GetFile().GetPackage())
	require.Equal(t, true, en.GetFile().IsProto3())

	// Ask for another one. This one has a name that looks like "some.YetAnother"
	// package in this context.
	md3, err := mr.FindMessageTypeByUrl("foo.bar/some.YetAnother.MessageType")
	require.NoError(t, err)
	require.Equal(t, "MessageType", md3.GetName())
	require.Equal(t, "some.YetAnother.MessageType", md3.GetFullyQualifiedName())
	require.Equal(t, "some.YetAnother", md3.GetFile().GetPackage())
	require.Equal(t, false, md3.GetFile().IsProto3())
	require.Equal(t, false, md3.IsProto3())
}

func TestMessageRegistry_FindMessage_Mixed(t *testing.T) {
	msgType := &typepb.Type{
		Name:   "foo.Bar",
		Oneofs: []string{"baz"},
		Fields: []*typepb.Field{
			{
				Name:        "id",
				Number:      1,
				Kind:        typepb.Field_TYPE_UINT64,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "id",
			},
			{
				Name:        "name",
				Number:      2,
				Kind:        typepb.Field_TYPE_STRING,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "name",
			},
			{
				Name:        "count",
				Number:      3,
				OneofIndex:  1,
				Kind:        typepb.Field_TYPE_INT32,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "count",
			},
			{
				Name:        "data",
				Number:      4,
				OneofIndex:  1,
				Kind:        typepb.Field_TYPE_BYTES,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "data",
			},
			{
				Name:        "other",
				Number:      5,
				OneofIndex:  1,
				Kind:        typepb.Field_TYPE_MESSAGE,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "other",
				TypeUrl:     "type.googleapis.com/google.protobuf.Empty",
			},
			{
				Name:        "created",
				Number:      6,
				Kind:        typepb.Field_TYPE_MESSAGE,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "created",
				TypeUrl:     "type.googleapis.com/google.protobuf.Timestamp",
			},
			{
				Name:        "updated",
				Number:      7,
				Kind:        typepb.Field_TYPE_MESSAGE,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "updated",
				TypeUrl:     "type.googleapis.com/google.protobuf.Timestamp",
			},
			{
				Name:        "tombstone",
				Number:      8,
				Kind:        typepb.Field_TYPE_BOOL,
				Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
				JsonName:    "tombstone",
			},
		},
		SourceContext: &sourcecontextpb.SourceContext{
			FileName: "test/foo.proto",
		},
		Syntax: typepb.Syntax_SYNTAX_PROTO3,
	}

	var mr MessageRegistry
	mr.WithFetcher(func(url string, enum bool) (proto.Message, error) {
		if url == "https://foo.test.com/foo.Bar" && !enum {
			return msgType, nil
		}
		return nil, fmt.Errorf("unknown type: %s", url)
	})

	// Make sure we successfully get back a descriptor
	md, err := mr.FindMessageTypeByUrl("foo.test.com/foo.Bar")
	require.NoError(t, err)

	// Check its properties. It should have the fields from the type
	// description above, but also correctly refer to google/protobuf
	// dependencies (which came from resolver, not the fetcher).

	require.Equal(t, "foo.Bar", md.GetFullyQualifiedName())
	require.Equal(t, "Bar", md.GetName())
	require.Equal(t, "test/foo.proto", md.GetFile().GetName())
	require.Equal(t, "foo", md.GetFile().GetPackage())

	fd := md.FindFieldByName("created")
	require.Equal(t, "google.protobuf.Timestamp", fd.GetMessageType().GetFullyQualifiedName())
	require.Equal(t, "google/protobuf/timestamp.proto", fd.GetMessageType().GetFile().GetName())

	ood := md.GetOneOfs()[0]
	require.Equal(t, 3, len(ood.GetChoices()))
	fd = ood.GetChoices()[2]
	require.Equal(t, "google.protobuf.Empty", fd.GetMessageType().GetFullyQualifiedName())
	require.Equal(t, "google/protobuf/empty.proto", fd.GetMessageType().GetFile().GetName())
}

func TestMessageRegistry_FindEnum_WithFetcher(t *testing.T) {
	tf := createFetcher(t)
	// we want "defaults" for the message factory so that we can properly process
	// known extensions (which the type fetcher puts into the descriptor options)
	mr := &RemoteRegistry{TypeFetcher: tf}

	ed, err := mr.FindEnumTypeByUrl("foo.bar/some.Enum")
	require.NoError(t, err)

	require.Equal(t, "Enum", ed.GetName())
	require.Equal(t, "some.Enum", ed.GetFullyQualifiedName())
	require.Equal(t, "some", ed.GetFile().GetPackage())
	require.Equal(t, true, ed.GetFile().IsProto3())

	eo := &descriptorpb.EnumOptions{
		Deprecated: proto.Bool(true),
		AllowAlias: proto.Bool(true),
	}
	err = proto.SetExtension(eo, testdata.E_Efubar, proto.Int32(-42))
	require.NoError(t, err)
	err = proto.SetExtension(eo, testdata.E_Efubars, proto.Int32(-42))
	require.NoError(t, err)
	err = proto.SetExtension(eo, testdata.E_Efubarsf, proto.Int32(-42))
	require.NoError(t, err)
	err = proto.SetExtension(eo, testdata.E_Efubaru, proto.Uint32(42))
	require.NoError(t, err)
	err = proto.SetExtension(eo, testdata.E_Efubaruf, proto.Uint32(42))
	require.NoError(t, err)
	protosEqual(t, eo, ed.GetEnumOptions())

	vals := ed.GetValues()
	require.Equal(t, 3, len(vals))
	require.Equal(t, "ABC", vals[0].GetName())
	require.Equal(t, int32(0), vals[0].GetNumber())

	evo := &descriptorpb.EnumValueOptions{
		Deprecated: proto.Bool(true),
	}
	err = proto.SetExtension(evo, testdata.E_Evfubar, proto.Int64(-420420420420))
	require.NoError(t, err)
	err = proto.SetExtension(evo, testdata.E_Evfubars, proto.Int64(-420420420420))
	require.NoError(t, err)
	err = proto.SetExtension(evo, testdata.E_Evfubarsf, proto.Int64(-420420420420))
	require.NoError(t, err)
	err = proto.SetExtension(evo, testdata.E_Evfubaru, proto.Uint64(420420420420))
	require.NoError(t, err)
	err = proto.SetExtension(evo, testdata.E_Evfubaruf, proto.Uint64(420420420420))
	require.NoError(t, err)
	protosEqual(t, evo, vals[0].GetEnumValueOptions())

	require.Equal(t, "XYZ", vals[1].GetName())
	require.Equal(t, int32(1), vals[1].GetNumber())

	require.Equal(t, "WXY", vals[2].GetName())
	require.Equal(t, int32(1), vals[2].GetNumber())
}

func createFetcher(t *testing.T) TypeFetcher {
	var bol anypb.Any
	err := anypb.MarshalFrom(&bol, &wrapperspb.BoolValue{Value: true}, proto.MarshalOptions{})
	require.NoError(t, err)
	var in32 anypb.Any
	err = anypb.MarshalFrom(&in32, &wrapperspb.Int32Value{Value: -42}, proto.MarshalOptions{})
	require.NoError(t, err)
	var uin32 anypb.Any
	err = anypb.MarshalFrom(&uin32, &wrapperspb.UInt32Value{Value: 42}, proto.MarshalOptions{})
	require.NoError(t, err)
	var in64 anypb.Any
	err = anypb.MarshalFrom(&in64, &wrapperspb.Int64Value{Value: -420420420420}, proto.MarshalOptions{})
	require.NoError(t, err)
	var uin64 anypb.Any
	err = anypb.MarshalFrom(&uin64, &wrapperspb.UInt64Value{Value: 420420420420}, proto.MarshalOptions{})
	require.NoError(t, err)
	var byt anypb.Any
	err = anypb.MarshalFrom(&byt, &wrapperspb.BytesValue{Value: []byte{1, 2, 3, 4, 5, 6, 7, 8}}, proto.MarshalOptions{})
	require.NoError(t, err)
	var str1 anypb.Any
	err = anypb.MarshalFrom(&str1, &wrapperspb.StringValue{Value: "foo"}, proto.MarshalOptions{})
	require.NoError(t, err)
	var str2 anypb.Any
	err = anypb.MarshalFrom(&str2, &wrapperspb.StringValue{Value: "bar"}, proto.MarshalOptions{})
	require.NoError(t, err)
	var str3 anypb.Any
	err = anypb.MarshalFrom(&str3, &wrapperspb.StringValue{Value: "baz"}, proto.MarshalOptions{})
	require.NoError(t, err)

	types := map[string]proto.Message{
		"https://foo.bar/some.Type": &typepb.Type{
			Name:   "some.Type",
			Oneofs: []string{"un"},
			Fields: []*typepb.Field{
				{
					Name:        "a",
					JsonName:    "a",
					Number:      1,
					Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
					Kind:        typepb.Field_TYPE_MESSAGE,
					TypeUrl:     "foo.bar/some.OtherType",
					Options: []*typepb.Option{
						{
							Name:  "deprecated",
							Value: &bol,
						},
						{
							Name:  "testdata.ffubar",
							Value: &str1,
						},
						{
							Name:  "testdata.ffubar",
							Value: &str2,
						},
						{
							Name:  "testdata.ffubar",
							Value: &str3,
						},
						{
							Name:  "testdata.ffubarb",
							Value: &byt,
						},
					},
				},
				{
					Name:        "b",
					JsonName:    "b",
					Number:      2,
					Cardinality: typepb.Field_CARDINALITY_REPEATED,
					Kind:        typepb.Field_TYPE_STRING,
				},
				{
					Name:        "c",
					JsonName:    "c",
					Number:      3,
					Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
					Kind:        typepb.Field_TYPE_ENUM,
					TypeUrl:     "foo.bar/some.Enum",
					OneofIndex:  1,
				},
				{
					Name:        "d",
					JsonName:    "d",
					Number:      4,
					Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
					Kind:        typepb.Field_TYPE_INT32,
					OneofIndex:  1,
				},
			},
			Options: []*typepb.Option{
				{
					Name:  "deprecated",
					Value: &bol,
				},
				{
					Name:  "testdata.mfubar",
					Value: &bol,
				},
			},
			SourceContext: &sourcecontextpb.SourceContext{FileName: "foo.proto"},
			Syntax:        typepb.Syntax_SYNTAX_PROTO3,
		},
		"https://foo.bar/some.OtherType": &typepb.Type{
			Name: "some.OtherType",
			Fields: []*typepb.Field{
				{
					Name:        "a",
					JsonName:    "a",
					Number:      1,
					Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
					Kind:        typepb.Field_TYPE_MESSAGE,
					TypeUrl:     "foo.bar/some.OtherType.AnotherType",
				},
			},
			SourceContext: &sourcecontextpb.SourceContext{FileName: "bar.proto"},
			Syntax:        typepb.Syntax_SYNTAX_PROTO2,
		},
		"https://foo.bar/some.OtherType.AnotherType": &typepb.Type{
			Name: "some.OtherType.AnotherType",
			Fields: []*typepb.Field{
				{
					Name:        "a",
					JsonName:    "a",
					Number:      1,
					Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
					Kind:        typepb.Field_TYPE_BYTES,
				},
			},
			SourceContext: &sourcecontextpb.SourceContext{FileName: "bar.proto"},
			Syntax:        typepb.Syntax_SYNTAX_PROTO2,
		},
		"https://foo.bar/some.Enum": &typepb.Enum{
			Name: "some.Enum",
			Enumvalue: []*typepb.EnumValue{
				{
					Name:   "ABC",
					Number: 0,
					Options: []*typepb.Option{
						{
							Name:  "deprecated",
							Value: &bol,
						},
						{
							Name:  "testdata.evfubar",
							Value: &in64,
						},
						{
							Name:  "testdata.evfubars",
							Value: &in64,
						},
						{
							Name:  "testdata.evfubarsf",
							Value: &in64,
						},
						{
							Name:  "testdata.evfubaru",
							Value: &uin64,
						},
						{
							Name:  "testdata.evfubaruf",
							Value: &uin64,
						},
					},
				},
				{
					Name:   "XYZ",
					Number: 1,
				},
				{
					Name:   "WXY",
					Number: 1,
				},
			},
			Options: []*typepb.Option{
				{
					Name:  "deprecated",
					Value: &bol,
				},
				{
					Name:  "allow_alias",
					Value: &bol,
				},
				{
					Name:  "testdata.efubar",
					Value: &in32,
				},
				{
					Name:  "testdata.efubars",
					Value: &in32,
				},
				{
					Name:  "testdata.efubarsf",
					Value: &in32,
				},
				{
					Name:  "testdata.efubaru",
					Value: &uin32,
				},
				{
					Name:  "testdata.efubaruf",
					Value: &uin32,
				},
			},
			SourceContext: &sourcecontextpb.SourceContext{FileName: "foo.proto"},
			Syntax:        typepb.Syntax_SYNTAX_PROTO3,
		},
		"https://foo.bar/some.YetAnother.MessageType": &typepb.Type{
			// in a separate file, so it will look like package some.YetAnother
			Name: "some.YetAnother.MessageType",
			Fields: []*typepb.Field{
				{
					Name:        "a",
					JsonName:    "a",
					Number:      1,
					Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
					Kind:        typepb.Field_TYPE_STRING,
				},
			},
			SourceContext: &sourcecontextpb.SourceContext{FileName: "baz.proto"},
			Syntax:        typepb.Syntax_SYNTAX_PROTO2,
		},
	}
	return func(url string, enum bool) (proto.Message, error) {
		t := types[url]
		if t == nil {
			return nil, nil
		}
		if _, ok := t.(*typepb.Enum); ok == enum {
			return t, nil
		} else {
			return nil, fmt.Errorf("bad type for %s", url)
		}
	}
}

func TestMessageRegistry_ResolveApiIntoServiceDescriptor(t *testing.T) {
	tf := createFetcher(t)
	// we want "defaults" for the message factory so that we can properly process
	// known extensions (which the type fetcher puts into the descriptor options)
	mr := &RemoteRegistry{TypeFetcher: tf}

	sd, err := mr.ResolveApiIntoServiceDescriptor(getApi(t))
	require.NoError(t, err)

	require.Equal(t, "Service", sd.GetName())
	require.Equal(t, "some.Service", sd.GetFullyQualifiedName())
	require.Equal(t, "some", sd.GetFile().GetPackage())
	require.Equal(t, true, sd.GetFile().IsProto3())

	so := &descriptorpb.ServiceOptions{
		Deprecated: proto.Bool(true),
	}
	err = proto.SetExtension(so, testdata.E_Sfubar, &testdata.ReallySimpleMessage{Id: proto.Uint64(100), Name: proto.String("deuce")})
	require.NoError(t, err)
	err = proto.SetExtension(so, testdata.E_Sfubare, testdata.ReallySimpleEnum_VALUE.Enum())
	require.NoError(t, err)
	protosEqual(t, so, sd.GetServiceOptions())

	methods := sd.GetMethods()
	require.Equal(t, 4, len(methods))
	require.Equal(t, "UnaryMethod", methods[0].GetName())
	require.Equal(t, "some.Type", methods[0].GetInputType().GetFullyQualifiedName())
	require.Equal(t, "some.OtherType", methods[0].GetOutputType().GetFullyQualifiedName())

	mto := &descriptorpb.MethodOptions{
		Deprecated: proto.Bool(true),
	}
	err = proto.SetExtension(mto, testdata.E_Mtfubar, []float32{3.14159, 2.71828})
	require.NoError(t, err)
	err = proto.SetExtension(mto, testdata.E_Mtfubard, proto.Float64(10203040.506070809))
	require.NoError(t, err)
	protosEqual(t, mto, methods[0].GetMethodOptions())

	require.Equal(t, "ClientStreamMethod", methods[1].GetName())
	require.Equal(t, "some.OtherType", methods[1].GetInputType().GetFullyQualifiedName())
	require.Equal(t, "some.Type", methods[1].GetOutputType().GetFullyQualifiedName())

	require.Equal(t, "ServerStreamMethod", methods[2].GetName())
	require.Equal(t, "some.OtherType.AnotherType", methods[2].GetInputType().GetFullyQualifiedName())
	require.Equal(t, "some.YetAnother.MessageType", methods[2].GetOutputType().GetFullyQualifiedName())

	require.Equal(t, "BidiStreamMethod", methods[3].GetName())
	require.Equal(t, "some.YetAnother.MessageType", methods[3].GetInputType().GetFullyQualifiedName())
	require.Equal(t, "some.OtherType.AnotherType", methods[3].GetOutputType().GetFullyQualifiedName())

	// check linked message types

	require.Equal(t, methods[0].GetInputType(), methods[1].GetOutputType())
	require.Equal(t, methods[0].GetOutputType(), methods[1].GetInputType())
	require.Equal(t, methods[2].GetInputType(), methods[3].GetOutputType())
	require.Equal(t, methods[2].GetOutputType(), methods[3].GetInputType())

	md1 := methods[0].GetInputType()
	md2 := methods[0].GetOutputType()
	md3 := methods[2].GetInputType()
	md4 := methods[2].GetOutputType()

	require.Equal(t, "Type", md1.GetName())
	require.Equal(t, "some.Type", md1.GetFullyQualifiedName())
	require.Equal(t, "some", md1.GetFile().GetPackage())
	require.Equal(t, true, md1.GetFile().IsProto3())
	require.Equal(t, true, md1.IsProto3())

	require.Equal(t, "OtherType", md2.GetName())
	require.Equal(t, "some.OtherType", md2.GetFullyQualifiedName())
	require.Equal(t, "some", md2.GetFile().GetPackage())
	require.Equal(t, false, md2.GetFile().IsProto3())
	require.Equal(t, false, md2.IsProto3())

	require.Equal(t, md3, md2.GetNestedMessageTypes()[0])
	require.Equal(t, "AnotherType", md3.GetName())
	require.Equal(t, "some.OtherType.AnotherType", md3.GetFullyQualifiedName())
	require.Equal(t, "some", md3.GetFile().GetPackage())
	require.Equal(t, false, md3.GetFile().IsProto3())
	require.Equal(t, false, md3.IsProto3())

	require.Equal(t, "MessageType", md4.GetName())
	require.Equal(t, "some.YetAnother.MessageType", md4.GetFullyQualifiedName())
	require.Equal(t, "some", md4.GetFile().GetPackage())
	require.Equal(t, true, md4.GetFile().IsProto3())
	require.Equal(t, true, md4.IsProto3())
}

func getApi(t *testing.T) *apipb.Api {
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
	err = anypb.MarshalFrom(&enu, &wrapperspb.Int32Value{Value: int32(testdata.ReallySimpleEnum_VALUE)}, proto.MarshalOptions{})
	require.NoError(t, err)
	var msg anypb.Any
	err = anypb.MarshalFrom(&msg, &testdata.ReallySimpleMessage{Id: proto.Uint64(100), Name: proto.String("deuce")}, proto.MarshalOptions{})
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
						Name:  "testdata.mtfubar",
						Value: &flt1,
					},
					{
						Name:  "testdata.mtfubar",
						Value: &flt2,
					},
					{
						Name:  "testdata.mtfubard",
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
				Name:  "testdata.sfubar",
				Value: &msg,
			},
			{
				Name:  "testdata.sfubare",
				Value: &enu,
			},
		},
		SourceContext: &sourcecontextpb.SourceContext{FileName: "baz.proto"},
		Syntax:        typepb.Syntax_SYNTAX_PROTO3,
	}
}

func TestMessageRegistry_MarshalAndUnmarshalAny(t *testing.T) {
	mr := NewMessageRegistryWithDefaults()

	md := (*descriptorpb.DescriptorProto)(nil).ProtoReflect().Descriptor()

	// marshal with default base URL
	a, err := mr.MarshalAny(md.AsProto())
	require.NoError(t, err)
	require.Equal(t, "type.googleapis.com/google.protobuf.DescriptorProto", a.TypeUrl)

	// check that we can unmarshal it with normal ptypes library
	var umd descriptorpb.DescriptorProto
	err = anypb.UnmarshalTo(a, &umd, proto.UnmarshalOptions{})
	require.NoError(t, err)
	protosEqual(t, md.AsProto(), &umd)

	// and that we can unmarshal it with a message registry
	pm, err := mr.UnmarshalAny(a)
	require.NoError(t, err)
	_, ok := pm.(*descriptorpb.DescriptorProto)
	require.True(t, ok)
	protosEqual(t, md.AsProto(), pm)

	// and that we can unmarshal it as a dynamic message, using a
	// message registry that doesn't know about the generated type
	mrWithoutDefaults := &MessageRegistry{}
	err = mrWithoutDefaults.AddMessage("type.googleapis.com/google.protobuf.DescriptorProto", md)
	require.NoError(t, err)
	pm, err = mrWithoutDefaults.UnmarshalAny(a)
	require.NoError(t, err)
	dm, ok := pm.(*dynamicpb.Message)
	require.True(t, ok)
	protosEqual(t, md.AsProto(), dm)

	// now test generation of type URLs with other settings

	// - different default
	mr.WithDefaultBaseUrl("foo.com/some/path/")
	a, err = mr.MarshalAny(md.AsProto())
	require.NoError(t, err)
	require.Equal(t, "foo.com/some/path/google.protobuf.DescriptorProto", a.TypeUrl)

	// - custom base URL for package
	mr.AddBaseUrlForElement("bar.com/other/", "google.protobuf")
	a, err = mr.MarshalAny(md.AsProto())
	require.NoError(t, err)
	require.Equal(t, "bar.com/other/google.protobuf.DescriptorProto", a.TypeUrl)

	// - custom base URL for type
	mr.AddBaseUrlForElement("http://baz.com/another/", "google.protobuf.DescriptorProto")
	a, err = mr.MarshalAny(md.AsProto())
	require.NoError(t, err)
	require.Equal(t, "http://baz.com/another/google.protobuf.DescriptorProto", a.TypeUrl)
}

func TestMessageRegistry_MessageDescriptorToPType(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto2"),
		Package: proto.String("foo"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Bar"),
				OneofDecl: []*descriptorpb.OneofDescriptorProto{
					{
						Name: proto.String("oo"),
					},
				},
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("abc"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						Options:  &descriptorpb.FieldOptions{Deprecated: proto.Bool(true)},
						JsonName: proto.String("abc"),
					},
					{
						Name:     proto.String("def"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
						Options:  &descriptorpb.FieldOptions{Packed: proto.Bool(true)},
						JsonName: proto.String("def"),
					},
					{
						Name:         proto.String("ghi"),
						Number:       proto.Int32(3),
						Label:        descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:         descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						DefaultValue: proto.String("foobar"),
						JsonName:     proto.String("ghi"),
					},
					{
						Name:       proto.String("nid"),
						Number:     proto.Int32(4),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
						JsonName:   proto.String("nid"),
						OneofIndex: proto.Int32(0),
					},
					{
						Name:       proto.String("sid"),
						Number:     proto.Int32(5),
						Label:      descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:       descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						JsonName:   proto.String("sid"),
						OneofIndex: proto.Int32(0),
					},
				},
			},
		},
	}
	fd, err := protodesc.NewFile(fdp, nil)
	require.NoError(t, err)

	msg := NewMessageRegistryWithDefaults().MessageAsPType(fd.GetMessageTypes()[0])

	// quick check of the resulting message's properties
	require.Equal(t, "foo.Bar", msg.Name)
	require.Equal(t, []string{"oo"}, msg.Oneofs)
	require.Equal(t, typepb.Syntax_SYNTAX_PROTO2, msg.Syntax)
	require.Equal(t, "test.proto", msg.SourceContext.GetFileName())
	require.Equal(t, 0, len(msg.Options))
	require.Equal(t, 5, len(msg.Fields))

	require.Equal(t, "abc", msg.Fields[0].Name)
	require.Equal(t, typepb.Field_CARDINALITY_OPTIONAL, msg.Fields[0].Cardinality)
	require.Equal(t, typepb.Field_TYPE_STRING, msg.Fields[0].Kind)
	require.Equal(t, "", msg.Fields[0].DefaultValue)
	require.Equal(t, int32(1), msg.Fields[0].Number)
	require.Equal(t, int32(0), msg.Fields[0].OneofIndex)
	require.Equal(t, 1, len(msg.Fields[0].Options))
	require.Equal(t, "deprecated", msg.Fields[0].Options[0].Name)
	// make sure the value is a wrapped bool
	v, err := anypb.UnmarshalNew(msg.Fields[0].Options[0].Value, proto.UnmarshalOptions{})
	require.NoError(t, err)
	protosEqual(t, &wrapperspb.BoolValue{Value: true}, v)

	require.Equal(t, "def", msg.Fields[1].Name)
	require.Equal(t, typepb.Field_CARDINALITY_REPEATED, msg.Fields[1].Cardinality)
	require.Equal(t, typepb.Field_TYPE_INT32, msg.Fields[1].Kind)
	require.Equal(t, "", msg.Fields[1].DefaultValue)
	require.Equal(t, int32(2), msg.Fields[1].Number)
	require.Equal(t, int32(0), msg.Fields[1].OneofIndex)
	require.Equal(t, true, msg.Fields[1].Packed)
	require.Equal(t, 0, len(msg.Fields[1].Options))

	require.Equal(t, "ghi", msg.Fields[2].Name)
	require.Equal(t, typepb.Field_CARDINALITY_OPTIONAL, msg.Fields[2].Cardinality)
	require.Equal(t, typepb.Field_TYPE_STRING, msg.Fields[2].Kind)
	require.Equal(t, "foobar", msg.Fields[2].DefaultValue)
	require.Equal(t, int32(3), msg.Fields[2].Number)
	require.Equal(t, int32(0), msg.Fields[2].OneofIndex)
	require.Equal(t, 0, len(msg.Fields[2].Options))

	require.Equal(t, "nid", msg.Fields[3].Name)
	require.Equal(t, typepb.Field_CARDINALITY_OPTIONAL, msg.Fields[3].Cardinality)
	require.Equal(t, typepb.Field_TYPE_UINT64, msg.Fields[3].Kind)
	require.Equal(t, "", msg.Fields[3].DefaultValue)
	require.Equal(t, int32(4), msg.Fields[3].Number)
	require.Equal(t, int32(1), msg.Fields[3].OneofIndex)
	require.Equal(t, 0, len(msg.Fields[3].Options))

	require.Equal(t, "sid", msg.Fields[4].Name)
	require.Equal(t, typepb.Field_CARDINALITY_OPTIONAL, msg.Fields[4].Cardinality)
	require.Equal(t, typepb.Field_TYPE_STRING, msg.Fields[4].Kind)
	require.Equal(t, "", msg.Fields[4].DefaultValue)
	require.Equal(t, int32(5), msg.Fields[4].Number)
	require.Equal(t, int32(1), msg.Fields[4].OneofIndex)
	require.Equal(t, 0, len(msg.Fields[4].Options))
}

func TestMessageRegistry_EnumDescriptorToPType(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto2"),
		Package: proto.String("foo"),
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name:    proto.String("Bar"),
				Options: &descriptorpb.EnumOptions{AllowAlias: proto.Bool(true)},
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:   proto.String("ZERO"),
						Number: proto.Int32(0),
					},
					{
						Name:    proto.String("__UNSET__"),
						Number:  proto.Int32(0),
						Options: &descriptorpb.EnumValueOptions{Deprecated: proto.Bool(true)},
					},
					{
						Name:   proto.String("ONE"),
						Number: proto.Int32(1),
					},
					{
						Name:   proto.String("TWO"),
						Number: proto.Int32(2),
					},
					{
						Name:   proto.String("THREE"),
						Number: proto.Int32(3),
					},
				},
			},
		},
	}
	fd, err := protodesc.NewFile(fdp, nil)
	require.NoError(t, err)

	enum := NewMessageRegistryWithDefaults().EnumAsPType(fd.GetEnumTypes()[0])

	// quick check of the resulting message's properties
	require.Equal(t, "foo.Bar", enum.Name)
	require.Equal(t, typepb.Syntax_SYNTAX_PROTO2, enum.Syntax)
	require.Equal(t, "test.proto", enum.SourceContext.GetFileName())
	require.Equal(t, 5, len(enum.Enumvalue))
	require.Equal(t, 1, len(enum.Options))
	require.Equal(t, "allow_alias", enum.Options[0].Name)
	// make sure the value is a wrapped bool
	v, err := anypb.UnmarshalNew(enum.Options[0].Value, proto.UnmarshalOptions{})
	require.NoError(t, err)
	protosEqual(t, &wrapperspb.BoolValue{Value: true}, v)

	require.Equal(t, "ZERO", enum.Enumvalue[0].Name)
	require.Equal(t, int32(0), enum.Enumvalue[0].Number)
	require.Equal(t, 0, len(enum.Enumvalue[0].Options))

	require.Equal(t, "__UNSET__", enum.Enumvalue[1].Name)
	require.Equal(t, int32(0), enum.Enumvalue[1].Number)
	require.Equal(t, 1, len(enum.Enumvalue[1].Options))
	require.Equal(t, "deprecated", enum.Enumvalue[1].Options[0].Name)
	// make sure the value is a wrapped bool
	v, err = anypb.UnmarshalNew(enum.Enumvalue[1].Options[0].Value, proto.UnmarshalOptions{})
	require.NoError(t, err)
	protosEqual(t, &wrapperspb.BoolValue{Value: true}, v)

	require.Equal(t, "ONE", enum.Enumvalue[2].Name)
	require.Equal(t, int32(1), enum.Enumvalue[2].Number)
	require.Equal(t, 0, len(enum.Enumvalue[2].Options))

	require.Equal(t, "TWO", enum.Enumvalue[3].Name)
	require.Equal(t, int32(2), enum.Enumvalue[3].Number)
	require.Equal(t, 0, len(enum.Enumvalue[3].Options))

	require.Equal(t, "THREE", enum.Enumvalue[4].Name)
	require.Equal(t, int32(3), enum.Enumvalue[4].Number)
	require.Equal(t, 0, len(enum.Enumvalue[4].Options))
}

func TestMessageRegistry_ServiceDescriptorToApi(t *testing.T) {
	// TODO
}

func protosEqual(t *testing.T, a, b proto.Message) {
	t.Helper()
	diff := cmp.Diff(a, b, protocmp.Transform())
	require.Empty(t, diff)
}
