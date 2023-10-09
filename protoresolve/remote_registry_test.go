package protoresolve_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/jhump/protoreflect/v2/internal/testdata"
	. "github.com/jhump/protoreflect/v2/protoresolve"
	"github.com/jhump/protoreflect/v2/protowrap"
)

func TestRemoteRegistry_Basic(t *testing.T) {
	rr := &RemoteRegistry{Fallback: &Registry{} /* empty fallback */}

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
	_, err = rr.FindMessageByURL("type.googleapis.com/google.protobuf.DescriptorProto")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = rr.FindEnumByURL("type.googleapis.com/google.protobuf.FieldDescriptorProto.Type")
	require.ErrorIs(t, err, ErrNotFound)

	// wrong type
	_, err = rr.FindMessageByURL("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	var unexpectedTypeErr *ErrUnexpectedType
	require.ErrorAs(t, err, &unexpectedTypeErr)
	_, err = rr.FindEnumByURL("foo.bar/google.protobuf.DescriptorProto")
	require.ErrorAs(t, err, &unexpectedTypeErr)

	// unmarshal any successfully finds the registered type
	b, err := proto.Marshal(protowrap.ProtoFromMessageDescriptor(md))
	require.NoError(t, err)
	a := &anypb.Any{TypeUrl: "foo.bar/google.protobuf.DescriptorProto", Value: b}
	pm, err := anypb.UnmarshalNew(a, proto.UnmarshalOptions{Resolver: rr.AsTypeResolver()})
	require.NoError(t, err)
	protosEqual(t, protowrap.ProtoFromMessageDescriptor(md), pm)
	require.Equal(t, reflect.TypeOf((*dynamicpb.Message)(nil)), reflect.TypeOf(pm))

	fd, err := protoregistry.GlobalFiles.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)
	err = rr.RegisterTypesInFileWithBaseURL(fd, "frob.nitz/foo.bar")
	require.NoError(t, err)
	msgCount, enumCount := 0, 0
	msgsQueue := []protoreflect.MessageDescriptors{fd.Messages()}
	for len(msgsQueue) > 0 {
		mds := msgsQueue[0]
		msgsQueue = msgsQueue[1:]
		for i, length := 0, mds.Len(); i < length; i++ {
			md := mds.Get(i)
			msgCount++
			msgsQueue = append(msgsQueue, md.Messages())
			exp := fmt.Sprintf("https://frob.nitz/foo.bar/%s", md.FullName())
			require.Equal(t, exp, rr.URLForType(md))
			eds := md.Enums()
			for i, length := 0, eds.Len(); i < length; i++ {
				ed := eds.Get(i)
				enumCount++
				exp := fmt.Sprintf("https://frob.nitz/foo.bar/%s", ed.FullName())
				require.Equal(t, exp, rr.URLForType(ed))
			}
		}
	}
	eds := md.Enums()
	for i, length := 0, eds.Len(); i < length; i++ {
		ed := eds.Get(i)
		enumCount++
		exp := fmt.Sprintf("https://frob.nitz/foo.bar/%s", ed.FullName())
		require.Equal(t, exp, rr.URLForType(ed))
	}
	// sanity check
	require.Equal(t, 11, msgCount)
	require.Equal(t, 2, enumCount)
}

func TestRemoteRegistry_Fallback(t *testing.T) {
	rr := &RemoteRegistry{}

	md := (*descriptorpb.DescriptorProto)(nil).ProtoReflect().Descriptor()
	ed := md.ParentFile().Messages().ByName("FieldDescriptorProto").Enums().ByName("Type")
	require.NotNil(t, ed)

	// lookups without registration or type fetcher use fallback
	msg, err := rr.FindMessageByURL("type.googleapis.com/google.protobuf.DescriptorProto")
	require.NoError(t, err)
	require.Equal(t, md, msg)
	// default types don't know their base URL, so will resolve even w/ wrong name
	// (just have to get fully-qualified message name right)
	msg, err = rr.FindMessageByURL("foo.bar/google.protobuf.DescriptorProto")
	require.NoError(t, err)
	require.Equal(t, md, msg)

	en, err := rr.FindEnumByURL("type.googleapis.com/google.protobuf.FieldDescriptorProto.Type")
	require.NoError(t, err)
	require.Equal(t, ed, en)
	en, err = rr.FindEnumByURL("foo.bar/google.protobuf.FieldDescriptorProto.Type")
	require.NoError(t, err)
	require.Equal(t, ed, en)
}

func TestRemoteRegistry_FindMessage_TypeFetcher(t *testing.T) {
	tf := createFetcher(t)
	// we want "defaults" for the message factory so that we can properly process
	// known extensions (which the type fetcher puts into the descriptor options)
	rr := &RemoteRegistry{TypeFetcher: tf}

	md, err := rr.FindMessageByURL("foo.bar/some.Type")
	require.NoError(t, err)

	// Fairly in-depth check of the returned message descriptor:

	require.Equal(t, "Type", string(md.Name()))
	require.Equal(t, "some.Type", string(md.FullName()))
	require.Equal(t, "some", string(md.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, md.ParentFile().Syntax())

	mo := &descriptorpb.MessageOptions{
		Deprecated: proto.Bool(true),
	}
	proto.SetExtension(mo, testdata.E_Mfubar, true)
	protosEqual(t, mo, md.Options())

	flds := md.Fields()
	require.Equal(t, 4, flds.Len())
	require.Equal(t, "a", string(flds.Get(0).Name()))
	require.Equal(t, protoreflect.FieldNumber(1), flds.Get(0).Number())
	require.Nil(t, flds.Get(0).ContainingOneof())
	require.Equal(t, protoreflect.Optional, flds.Get(0).Cardinality())
	require.Equal(t, protoreflect.MessageKind, flds.Get(0).Kind())

	fo := &descriptorpb.FieldOptions{
		Deprecated: proto.Bool(true),
	}
	proto.SetExtension(fo, testdata.E_Ffubar, []string{"foo", "bar", "baz"})
	proto.SetExtension(fo, testdata.E_Ffubarb, []byte{1, 2, 3, 4, 5, 6, 7, 8})
	protosEqual(t, fo, flds.Get(0).Options())

	require.Equal(t, "b", string(flds.Get(1).Name()))
	require.Equal(t, protoreflect.FieldNumber(2), flds.Get(1).Number())
	require.Nil(t, flds.Get(1).ContainingOneof())
	require.Equal(t, protoreflect.Repeated, flds.Get(1).Cardinality())
	require.Equal(t, protoreflect.StringKind, flds.Get(1).Kind())

	require.Equal(t, "c", string(flds.Get(2).Name()))
	require.Equal(t, protoreflect.FieldNumber(3), flds.Get(2).Number())
	require.Equal(t, "un", string(flds.Get(2).ContainingOneof().Name()))
	require.Equal(t, protoreflect.Optional, flds.Get(2).Cardinality())
	require.Equal(t, protoreflect.EnumKind, flds.Get(2).Kind())

	require.Equal(t, "d", string(flds.Get(3).Name()))
	require.Equal(t, protoreflect.FieldNumber(4), flds.Get(3).Number())
	require.Equal(t, "un", string(flds.Get(3).ContainingOneof().Name()))
	require.Equal(t, protoreflect.Optional, flds.Get(3).Cardinality())
	require.Equal(t, protoreflect.Int32Kind, flds.Get(3).Kind())

	oos := md.Oneofs()
	require.Equal(t, 1, oos.Len())
	require.Equal(t, "un", string(oos.Get(0).Name()))
	ooflds := oos.Get(0).Fields()
	require.Equal(t, 2, ooflds.Len())
	require.Equal(t, flds.Get(2), ooflds.Get(0))
	require.Equal(t, flds.Get(3), ooflds.Get(1))

	// Quick, shallow check of the linked descriptors:

	md2 := md.Fields().ByName("a").Message()
	require.Equal(t, "OtherType", string(md2.Name()))
	require.Equal(t, "some.OtherType", string(md2.FullName()))
	require.Equal(t, "some", string(md2.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, md2.ParentFile().Syntax())

	nmd := md2.Messages().Get(0)
	protosEqual(t, protowrap.ProtoFromMessageDescriptor(nmd), protowrap.ProtoFromMessageDescriptor(md2.Fields().ByName("a").Message()))
	require.Equal(t, "AnotherType", string(nmd.Name()))
	require.Equal(t, "some.OtherType.AnotherType", string(nmd.FullName()))
	require.Equal(t, "some", string(nmd.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, nmd.ParentFile().Syntax())

	en := md.Fields().ByName("c").Enum()
	require.Equal(t, "Enum", string(en.Name()))
	require.Equal(t, "some.Enum", string(en.FullName()))
	require.Equal(t, "some", string(en.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, en.ParentFile().Syntax())

	// Ask for another one. This one has a name that looks like "some.YetAnother"
	// package in this context.
	md3, err := rr.FindMessageByURL("foo.bar/some.YetAnother.MessageType")
	require.NoError(t, err)
	require.Equal(t, "MessageType", string(md3.Name()))
	require.Equal(t, "some.YetAnother.MessageType", string(md3.FullName()))
	require.Equal(t, "some.YetAnother", string(md3.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, md3.ParentFile().Syntax())
}

func TestRemoteRegistry_FindMessage_Mixed(t *testing.T) {
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

	rr := &RemoteRegistry{TypeFetcher: TypeFetcherFunc(func(_ context.Context, url string, enum bool) (proto.Message, error) {
		if url == "https://foo.test.com/foo.Bar" && !enum {
			return msgType, nil
		}
		return nil, ErrNotFound
	})}

	// Make sure we successfully get back a descriptor
	md, err := rr.FindMessageByURL("foo.test.com/foo.Bar")
	require.NoError(t, err)

	// Check its properties. It should have the fields from the type
	// description above, but also correctly refer to google/protobuf
	// dependencies (which came from resolver, not the fetcher).

	require.Equal(t, "foo.Bar", string(md.FullName()))
	require.Equal(t, "Bar", string(md.Name()))
	require.Equal(t, "test/foo.proto", md.ParentFile().Path())
	require.Equal(t, "foo", string(md.ParentFile().Package()))

	fd := md.Fields().ByName("created")
	require.Equal(t, "google.protobuf.Timestamp", string(fd.Message().FullName()))
	require.Equal(t, "google/protobuf/timestamp.proto", fd.Message().ParentFile().Path())

	ood := md.Oneofs().Get(0)
	require.Equal(t, 3, ood.Fields().Len())
	fd = ood.Fields().Get(2)
	require.Equal(t, "google.protobuf.Empty", string(fd.Message().FullName()))
	require.Equal(t, "google/protobuf/empty.proto", fd.Message().ParentFile().Path())
}

func TestRemoteRegistry_FindEnum_TypeFetcher(t *testing.T) {
	tf := createFetcher(t)
	// we want "defaults" for the message factory so that we can properly process
	// known extensions (which the type fetcher puts into the descriptor options)
	mr := &RemoteRegistry{TypeFetcher: tf}

	ed, err := mr.FindEnumByURL("foo.bar/some.Enum")
	require.NoError(t, err)

	require.Equal(t, "Enum", string(ed.Name()))
	require.Equal(t, "some.Enum", string(ed.FullName()))
	require.Equal(t, "some", string(ed.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, ed.ParentFile().Syntax())

	eo := &descriptorpb.EnumOptions{
		Deprecated: proto.Bool(true),
		AllowAlias: proto.Bool(true),
	}
	proto.SetExtension(eo, testdata.E_Efubar, int32(-42))
	require.NoError(t, err)
	proto.SetExtension(eo, testdata.E_Efubars, int32(-42))
	require.NoError(t, err)
	proto.SetExtension(eo, testdata.E_Efubarsf, int32(-42))
	require.NoError(t, err)
	proto.SetExtension(eo, testdata.E_Efubaru, uint32(42))
	require.NoError(t, err)
	proto.SetExtension(eo, testdata.E_Efubaruf, uint32(42))
	require.NoError(t, err)
	protosEqual(t, eo, ed.Options())

	vals := ed.Values()
	require.Equal(t, 3, vals.Len())
	require.Equal(t, "ABC", string(vals.Get(0).Name()))
	require.Equal(t, protoreflect.EnumNumber(0), vals.Get(0).Number())

	evo := &descriptorpb.EnumValueOptions{
		Deprecated: proto.Bool(true),
	}
	proto.SetExtension(evo, testdata.E_Evfubar, int64(-420420420420))
	require.NoError(t, err)
	proto.SetExtension(evo, testdata.E_Evfubars, int64(-420420420420))
	require.NoError(t, err)
	proto.SetExtension(evo, testdata.E_Evfubarsf, int64(-420420420420))
	require.NoError(t, err)
	proto.SetExtension(evo, testdata.E_Evfubaru, uint64(420420420420))
	require.NoError(t, err)
	proto.SetExtension(evo, testdata.E_Evfubaruf, uint64(420420420420))
	require.NoError(t, err)
	protosEqual(t, evo, vals.Get(0).Options())

	require.Equal(t, "XYZ", string(vals.Get(1).Name()))
	require.Equal(t, protoreflect.EnumNumber(1), vals.Get(1).Number())

	require.Equal(t, "WXY", string(vals.Get(2).Name()))
	require.Equal(t, protoreflect.EnumNumber(1), vals.Get(2).Number())
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
	return TypeFetcherFunc(func(_ context.Context, url string, enum bool) (proto.Message, error) {
		t := types[url]
		if t == nil {
			return nil, nil
		}
		if _, ok := t.(*typepb.Enum); ok == enum {
			return t, nil
		} else {
			return nil, fmt.Errorf("bad type for %s", url)
		}
	})
}

func TestDescriptorConverter_ToServiceDescriptor(t *testing.T) {
	tf := createFetcher(t)
	// we want "defaults" for the message factory so that we can properly process
	// known extensions (which the type fetcher puts into the descriptor options)
	rr := &RemoteRegistry{TypeFetcher: tf}
	dc := rr.AsDescriptorConverter()

	sd, err := dc.ToServiceDescriptor(context.Background(), getApi(t))
	require.NoError(t, err)

	require.Equal(t, "Service", string(sd.Name()))
	require.Equal(t, "some.Service", string(sd.FullName()))
	require.Equal(t, "some", string(sd.ParentFile().Package()))
	require.Equal(t, protoreflect.Proto3, sd.ParentFile().Syntax())

	so := &descriptorpb.ServiceOptions{
		Deprecated: proto.Bool(true),
	}
	proto.SetExtension(so, testdata.E_Sfubar, &testdata.ReallySimpleMessage{Id: proto.Uint64(100), Name: proto.String("deuce")})
	require.NoError(t, err)
	proto.SetExtension(so, testdata.E_Sfubare, testdata.ReallySimpleEnum_VALUE)
	require.NoError(t, err)
	protosEqual(t, so, sd.Options())

	methods := sd.Methods()
	require.Equal(t, 4, methods.Len())
	require.Equal(t, "UnaryMethod", string(methods.Get(0).Name()))
	require.Equal(t, "some.Type", string(methods.Get(0).Input().FullName()))
	require.Equal(t, "some.OtherType", string(methods.Get(0).Output().FullName()))

	mto := &descriptorpb.MethodOptions{
		Deprecated: proto.Bool(true),
	}
	proto.SetExtension(mto, testdata.E_Mtfubar, []float32{3.14159, 2.71828})
	require.NoError(t, err)
	proto.SetExtension(mto, testdata.E_Mtfubard, 10203040.506070809)
	require.NoError(t, err)
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

func TestDescriptorConverter_DescriptorAsApi(t *testing.T) {
	// TODO
}

func TestDescriptorConverter_ToMessageDescriptor(t *testing.T) {
	// TODO
}

func TestDescriptorConverter_DescriptorAsType(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
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

	msg := (&RemoteRegistry{}).AsDescriptorConverter().DescriptorAsType(fd.Messages().Get(0))

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
	require.Equal(t, int32(0), msg.Fields[3].OneofIndex)
	require.Equal(t, 0, len(msg.Fields[3].Options))

	require.Equal(t, "sid", msg.Fields[4].Name)
	require.Equal(t, typepb.Field_CARDINALITY_OPTIONAL, msg.Fields[4].Cardinality)
	require.Equal(t, typepb.Field_TYPE_STRING, msg.Fields[4].Kind)
	require.Equal(t, "", msg.Fields[4].DefaultValue)
	require.Equal(t, int32(5), msg.Fields[4].Number)
	require.Equal(t, int32(0), msg.Fields[4].OneofIndex)
	require.Equal(t, 0, len(msg.Fields[4].Options))
}

func TestDescriptorConverter_ToEnumDescriptor(t *testing.T) {
	// TODO
}

func TestDescriptorConverter_DescriptorAsEnum(t *testing.T) {
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("test.proto"),
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

	enum := (&RemoteRegistry{}).AsDescriptorConverter().DescriptorAsEnum(fd.Enums().Get(0))

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

func protosEqual(t *testing.T, a, b proto.Message) {
	t.Helper()
	diff := cmp.Diff(a, b, protocmp.Transform())
	require.Empty(t, diff)
}
