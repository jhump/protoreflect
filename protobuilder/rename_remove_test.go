package protobuilder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
)

func TestRenamingBuilders(t *testing.T) {
	t.Parallel()
	t.Run("file elements", testRenameFileElements)
	t.Run("message elements", testRenameMessageElements)
	t.Run("enum values and methods", testRenameEnumValuesAndMethods)
	t.Run("group and map types", testRenameGroupAndMapTypes)
}

func testRenameFileElements(t *testing.T) {
	t.Parallel()
	msg := NewMessage("Msg")
	enum := NewEnum("Enum").AddValue(NewEnumValue("ENUM_ZERO"))
	ext := NewExtension("ext", 100, FieldTypeString(), msg)
	svc := NewService("Svc")
	msg.AddExtensionRange(100, 200)
	file := NewFile("test.proto").SetPackageName("test").
		AddMessage(msg).AddEnum(enum).AddExtension(ext).AddService(svc)

	msg.SetName("Msg2")
	enum.SetName("Enum2")
	ext.SetName("ext2")
	svc.SetName("Svc2")
	assert.Nil(t, file.GetMessage("Msg"))
	assert.Same(t, msg, file.GetMessage("Msg2"))
	assert.Nil(t, file.GetEnum("Enum"))
	assert.Same(t, enum, file.GetEnum("Enum2"))
	assert.Nil(t, file.GetExtension("ext"))
	assert.Same(t, ext, file.GetExtension("ext2"))
	assert.Nil(t, file.GetService("Svc"))
	assert.Same(t, svc, file.GetService("Svc2"))

	// Names must be unique across all kinds of elements.
	assert.ErrorContains(t, msg.TrySetName("Enum2"), "already contains element")
	assert.Equal(t, protoreflect.Name("Msg2"), msg.Name(), "failed rename should be reverted")
	assert.Same(t, msg, file.GetMessage("Msg2"))
	assert.ErrorContains(t, svc.TrySetName("Msg2"), "already contains element")
	// The old names are free for re-use.
	file.AddMessage(NewMessage("Msg"))

	fd, err := file.Build()
	require.NoError(t, err)
	assert.NotNil(t, fd.Messages().ByName("Msg2"))
	assert.NotNil(t, fd.Messages().ByName("Msg"))
	assert.NotNil(t, fd.Enums().ByName("Enum2"))
	assert.NotNil(t, fd.Extensions().ByName("ext2"))
	assert.NotNil(t, fd.Services().ByName("Svc2"))
}

func testRenameMessageElements(t *testing.T) {
	t.Parallel()
	field := NewField("field", FieldTypeString()).SetNumber(1)
	oneof := NewOneof("oneof").AddChoice(NewField("choice", FieldTypeInt32()).SetNumber(2))
	nested := NewMessage("Nested")
	nestedEnum := NewEnum("NestedEnum").AddValue(NewEnumValue("NESTED_ZERO"))
	msg := NewMessage("Msg").AddField(field).AddOneOf(oneof).
		AddNestedMessage(nested).AddNestedEnum(nestedEnum).AddExtensionRange(100, 200)
	nestedExt := NewExtension("nested_ext", 100, FieldTypeString(), msg)
	msg.AddNestedExtension(nestedExt)
	file := NewFile("test.proto").SetPackageName("test").AddMessage(msg)

	field.SetName("field2")
	oneof.SetName("oneof2")
	nested.SetName("Nested2")
	nestedEnum.SetName("NestedEnum2")
	nestedExt.SetName("nested_ext2")
	assert.Nil(t, msg.GetField("field"))
	assert.Same(t, field, msg.GetField("field2"))
	assert.Nil(t, msg.GetOneOf("oneof"))
	assert.Same(t, oneof, msg.GetOneOf("oneof2"))
	assert.Nil(t, msg.GetNestedMessage("Nested"))
	assert.Same(t, nested, msg.GetNestedMessage("Nested2"))
	assert.Nil(t, msg.GetNestedEnum("NestedEnum"))
	assert.Same(t, nestedEnum, msg.GetNestedEnum("NestedEnum2"))
	assert.Nil(t, msg.GetNestedExtension("nested_ext"))
	assert.Same(t, nestedExt, msg.GetNestedExtension("nested_ext2"))

	assert.ErrorContains(t, field.TrySetName("Nested2"), "already contains element")
	assert.Equal(t, protoreflect.Name("field2"), field.Name())

	// Oneof choices share the message's namespace.
	choice := oneof.GetChoice("choice")
	require.NotNil(t, choice)
	assert.ErrorContains(t, choice.TrySetName("field2"), "already contains element")
	assert.Equal(t, protoreflect.Name("choice"), choice.Name())
	assert.Same(t, choice, oneof.GetChoice("choice"))
	choice.SetName("choice2")
	assert.Nil(t, oneof.GetChoice("choice"))
	assert.Same(t, choice, oneof.GetChoice("choice2"))
	assert.Same(t, choice, msg.GetField("choice2"))
	// The old name is free for re-use.
	msg.AddField(NewField("choice", FieldTypeBool()).SetNumber(3))

	md, err := msg.Build()
	require.NoError(t, err)
	assert.NotNil(t, md.Fields().ByName("field2"))
	assert.NotNil(t, md.Fields().ByName("choice2"))
	assert.NotNil(t, md.Fields().ByName("choice"))
	assert.NotNil(t, md.Oneofs().ByName("oneof2"))
	assert.NotNil(t, md.Messages().ByName("Nested2"))
	assert.NotNil(t, md.Enums().ByName("NestedEnum2"))
	assert.NotNil(t, md.Extensions().ByName("nested_ext2"))
	_, err = file.Build()
	require.NoError(t, err)
}

func testRenameEnumValuesAndMethods(t *testing.T) {
	t.Parallel()
	val := NewEnumValue("VAL_ZERO")
	enum := NewEnum("Enum").AddValue(val).AddValue(NewEnumValue("VAL_ONE"))
	val.SetName("VAL_NONE")
	assert.Nil(t, enum.GetValue("VAL_ZERO"))
	assert.Same(t, val, enum.GetValue("VAL_NONE"))
	assert.ErrorContains(t, val.TrySetName("VAL_ONE"), "already contains")
	assert.Equal(t, protoreflect.Name("VAL_NONE"), val.Name())

	req := RpcTypeMessage(NewMessage("Req"), false)
	method := NewMethod("Method", req, req)
	svc := NewService("Svc").AddMethod(method).AddMethod(NewMethod("Other", req, req))
	method.SetName("Method2")
	assert.Nil(t, svc.GetMethod("Method"))
	assert.Same(t, method, svc.GetMethod("Method2"))
	assert.ErrorContains(t, method.TrySetName("Other"), "already contains")
	assert.Equal(t, protoreflect.Name("Method2"), method.Name())
}

func testRenameGroupAndMapTypes(t *testing.T) {
	t.Parallel()
	group := NewMessage("Group")
	groupField := NewGroupField(group).SetNumber(1)
	mapField := NewMapField("vals", FieldTypeString(), FieldTypeInt32()).SetNumber(2)
	msg := NewMessage("Msg").AddField(groupField).AddField(mapField)
	NewFile("test.proto").SetSyntax(protoreflect.Proto2).AddMessage(msg)

	// Renaming a group message renames its field.
	group.SetName("Renamed")
	assert.Equal(t, protoreflect.Name("renamed"), groupField.Name())
	assert.Same(t, groupField, msg.GetField("renamed"))
	assert.Same(t, group, msg.GetNestedMessage("Renamed"))
	assert.ErrorContains(t, group.TrySetName("lowercase"), "must start with capital letter")
	assert.Equal(t, protoreflect.Name("Renamed"), group.Name())

	// The old name is free for re-use.
	assert.Nil(t, msg.GetNestedMessage("Group"))
	require.NoError(t, msg.TryAddNestedMessage(NewMessage("Group")))

	// Map entry names are derived from the field name.
	entry := msg.GetNestedMessage("ValsEntry")
	require.NotNil(t, entry)
	assert.ErrorContains(t, entry.TrySetName("Other"), "cannot change name of map entry")
	mapField.SetName("values")
	assert.Equal(t, protoreflect.Name("ValuesEntry"), entry.Name())
	assert.Same(t, entry, msg.GetNestedMessage("ValuesEntry"))
	assert.Nil(t, msg.GetNestedMessage("ValsEntry"))
	require.NoError(t, msg.TryAddNestedMessage(NewMessage("ValsEntry")))

	// A failed rename of a map field leaves the field and entry unchanged.
	msg.AddField(NewField("other", FieldTypeString()).SetNumber(3))
	assert.ErrorContains(t, mapField.TrySetName("other"), "already contains element")
	assert.Equal(t, protoreflect.Name("values"), mapField.Name())
	assert.Equal(t, protoreflect.Name("ValuesEntry"), entry.Name())
	assert.Same(t, entry, msg.GetNestedMessage("ValuesEntry"))
	assert.Nil(t, msg.GetNestedMessage("OtherEntry"))

	md, err := msg.Build()
	require.NoError(t, err)
	assert.NotNil(t, md.Fields().ByName("renamed"))
	assert.NotNil(t, md.Messages().ByName("Renamed"))
	assert.NotNil(t, md.Fields().ByName("values"))
	assert.NotNil(t, md.Messages().ByName("ValuesEntry"))
}

func TestGroupInOneof(t *testing.T) {
	t.Parallel()
	newMessageWithGroupInOneof := func() (*MessageBuilder, *OneofBuilder, *MessageBuilder) {
		group := NewMessage("Group")
		oneof := NewOneof("oneof").AddChoice(NewGroupField(group).SetNumber(1))
		msg := NewMessage("Msg").AddOneOf(oneof)
		NewFile("test.proto").SetSyntax(protoreflect.Proto2).AddMessage(msg)
		return msg, oneof, group
	}

	t.Run("rename", func(t *testing.T) {
		t.Parallel()
		msg, oneof, group := newMessageWithGroupInOneof()
		group.SetName("Renamed")
		assert.Same(t, group, msg.GetNestedMessage("Renamed"))
		assert.Same(t, oneof.GetChoice("renamed"), msg.GetField("renamed"))
		assert.Nil(t, msg.GetNestedMessage("Group"))
		require.NoError(t, msg.TryAddNestedMessage(NewMessage("Group")))
		md, err := msg.Build()
		require.NoError(t, err)
		assert.NotNil(t, md.Oneofs().ByName("oneof").Fields().ByName("renamed"))
	})
	t.Run("remove choice", func(t *testing.T) {
		t.Parallel()
		msg, oneof, _ := newMessageWithGroupInOneof()
		require.True(t, oneof.TryRemoveChoice("group"))
		assert.Nil(t, msg.GetNestedMessage("Group"))
		require.NoError(t, msg.TryAddNestedMessage(NewMessage("Group")))
	})
	t.Run("remove oneof", func(t *testing.T) {
		t.Parallel()
		msg, _, _ := newMessageWithGroupInOneof()
		require.True(t, msg.TryRemoveOneOf("oneof"))
		assert.Nil(t, msg.GetNestedMessage("Group"))
		require.NoError(t, msg.TryAddNestedMessage(NewMessage("Group")))
	})
}

func TestRemoveMapEntryClearsParent(t *testing.T) {
	t.Parallel()
	mapField := NewMapField("vals", FieldTypeString(), FieldTypeInt32()).SetNumber(1)
	msg := NewMessage("Msg").AddField(mapField)
	entry := msg.GetNestedMessage("ValsEntry")
	require.NotNil(t, entry)
	require.True(t, msg.TryRemoveNestedMessage("ValsEntry"))
	assert.Nil(t, entry.Parent())
}

func TestBuildWithoutFile(t *testing.T) {
	t.Parallel()
	// Messages without a parent file are built in a synthetic file.
	msg := NewMessage("Msg")
	msg.AddField(NewField("self", FieldTypeMessage(msg)).SetNumber(1))
	other := NewMessage("Other").AddField(NewField("msg", FieldTypeMessage(msg)).SetNumber(1))
	md, err := msg.Build()
	require.NoError(t, err)
	assert.Equal(t, md.FullName(), md.Fields().ByName("self").Message().FullName())
	otherMd, err := other.Build()
	require.NoError(t, err)
	assert.Equal(t, md.FullName(), otherMd.Fields().ByName("msg").Message().FullName())

	// Elements that must have a parent return errors.
	_, err = NewField("field", FieldTypeString()).Build()
	assert.ErrorContains(t, err, "field must be added to message")
	_, err = NewOneof("oneof").Build()
	assert.ErrorContains(t, err, "one-of must be added to message")
	_, err = NewEnumValue("VALUE").Build()
	assert.ErrorContains(t, err, "enum value must be added to enum")
	req := RpcTypeMessage(NewMessage("Req"), false)
	_, err = NewMethod("Method", req, req).Build()
	assert.ErrorContains(t, err, "method must be added to service")
}

func TestBuildMethodWithoutTypes(t *testing.T) {
	t.Parallel()
	svc := NewService("Svc").AddMethod(NewMethod("Method", nil, nil))
	_, err := NewFile("test.proto").AddService(svc).Build()
	assert.ErrorContains(t, err, "method Svc.Method must have both request and response types")
}

func TestGroupOptionsAddImports(t *testing.T) {
	t.Parallel()
	opts := &descriptorpb.MessageOptions{}
	proto.SetExtension(opts, testprotos.E_Mfubar, true)
	group := NewMessage("Group").SetOptions(opts)
	msg := NewMessage("Msg").AddField(NewGroupField(group).SetNumber(1))
	fd, err := NewFile("test.proto").SetSyntax(protoreflect.Proto2).AddMessage(msg).Build()
	require.NoError(t, err)
	var imports []string
	for i := range fd.Imports().Len() {
		imports = append(imports, fd.Imports().Get(i).Path())
	}
	assert.Contains(t, imports, testprotos.File_desc_test_options_proto.Path())
}

func TestRemoveFileElements(t *testing.T) {
	t.Parallel()
	msg := NewMessage("Msg").AddExtensionRange(100, 200)
	file := NewFile("test.proto").SetPackageName("test").
		AddMessage(msg).
		AddMessage(NewMessage("Removed")).
		AddEnum(NewEnum("Enum").AddValue(NewEnumValue("ENUM_ZERO"))).
		AddExtension(NewExtension("ext", 100, FieldTypeString(), msg)).
		AddService(NewService("Svc"))
	removed := file.GetMessage("Removed")

	assert.True(t, file.TryRemoveMessage("Removed"))
	assert.False(t, file.TryRemoveMessage("Removed"))
	assert.Nil(t, removed.Parent())
	assert.True(t, file.TryRemoveEnum("Enum"))
	assert.False(t, file.TryRemoveEnum("Enum"))
	assert.True(t, file.TryRemoveExtension("ext"))
	assert.False(t, file.TryRemoveExtension("ext"))
	assert.True(t, file.TryRemoveService("Svc"))
	assert.False(t, file.TryRemoveService("Svc"))
	// The non-Try variants are no-ops for unknown names.
	file.RemoveMessage("Removed").RemoveEnum("Enum").RemoveExtension("ext").RemoveService("Svc")

	// Names are free for re-use, even for other kinds of elements.
	file.AddEnum(NewEnum("Removed").AddValue(NewEnumValue("REMOVED_ZERO")))
	file.AddMessage(NewMessage("Svc"))
	file.AddExtension(NewExtension("ext", 100, FieldTypeInt32(), msg))

	fd, err := file.Build()
	require.NoError(t, err)
	assert.Equal(t, 2, fd.Messages().Len())
	assert.NotNil(t, fd.Enums().ByName("Removed"))
	assert.Equal(t, protoreflect.Int32Kind, fd.Extensions().ByName("ext").Kind())
	assert.Zero(t, fd.Services().Len())
}

func TestRemoveMessageElements(t *testing.T) {
	t.Parallel()
	msg := NewMessage("Msg").
		AddField(NewField("field", FieldTypeString()).SetNumber(1)).
		AddField(NewMapField("vals", FieldTypeString(), FieldTypeInt32()).SetNumber(2)).
		AddOneOf(NewOneof("oneof").
			AddChoice(NewField("choice1", FieldTypeInt32()).SetNumber(3)).
			AddChoice(NewField("choice2", FieldTypeInt32()).SetNumber(4))).
		AddNestedMessage(NewMessage("Nested")).
		AddNestedEnum(NewEnum("NestedEnum").AddValue(NewEnumValue("NESTED_ZERO"))).
		AddExtensionRange(100, 200)
	msg.AddNestedExtension(NewExtension("nested_ext", 100, FieldTypeString(), msg))
	NewFile("test.proto").SetPackageName("test").AddMessage(msg)

	assert.True(t, msg.TryRemoveField("field"))
	assert.False(t, msg.TryRemoveField("field"))
	assert.True(t, msg.TryRemoveOneOf("oneof"))
	assert.False(t, msg.TryRemoveOneOf("oneof"))
	assert.True(t, msg.TryRemoveNestedMessage("Nested"))
	assert.False(t, msg.TryRemoveNestedMessage("Nested"))
	assert.True(t, msg.TryRemoveNestedEnum("NestedEnum"))
	assert.False(t, msg.TryRemoveNestedEnum("NestedEnum"))
	assert.True(t, msg.TryRemoveNestedExtension("nested_ext"))
	assert.False(t, msg.TryRemoveNestedExtension("nested_ext"))
	msg.RemoveField("field").RemoveOneOf("oneof").RemoveNestedMessage("Nested").
		RemoveNestedEnum("NestedEnum").RemoveNestedExtension("nested_ext")

	// Removing a map entry message removes it from its field.
	mapField := msg.GetField("vals")
	require.NotNil(t, mapField)
	require.NotNil(t, msg.GetNestedMessage("ValsEntry"))
	assert.True(t, msg.TryRemoveNestedMessage("ValsEntry"))
	assert.Empty(t, mapField.Children())
	assert.Nil(t, msg.GetNestedMessage("ValsEntry"))
	assert.True(t, msg.TryRemoveField("vals"))

	// Names and numbers of all removed elements, including oneof choices,
	// are free for re-use.
	for i, name := range []protoreflect.Name{"field", "choice1", "choice2", "oneof", "Nested", "NestedEnum", "nested_ext", "ValsEntry"} {
		require.NoError(t, msg.TryAddField(NewField(name, FieldTypeBool()).SetNumber(protoreflect.FieldNumber(i+1))), "re-using name %s", name)
	}

	md, err := msg.Build()
	require.NoError(t, err)
	assert.Equal(t, 8, md.Fields().Len())
	assert.Zero(t, md.Oneofs().Len())
	assert.Zero(t, md.Messages().Len())
	assert.Zero(t, md.Enums().Len())
	assert.Zero(t, md.Extensions().Len())
}

func TestRemoveOneofChoiceEnumValueAndMethod(t *testing.T) {
	t.Parallel()
	oneof := NewOneof("oneof").
		AddChoice(NewField("choice1", FieldTypeInt32()).SetNumber(1)).
		AddChoice(NewField("choice2", FieldTypeInt32()).SetNumber(2))
	msg := NewMessage("Msg").AddOneOf(oneof)
	assert.True(t, oneof.TryRemoveChoice("choice1"))
	assert.False(t, oneof.TryRemoveChoice("choice1"))
	oneof.RemoveChoice("choice1")
	assert.Nil(t, msg.GetField("choice1"))
	msg.AddField(NewField("choice1", FieldTypeBool()).SetNumber(1))
	md, err := msg.Build()
	require.NoError(t, err)
	assert.Equal(t, 1, md.Oneofs().ByName("oneof").Fields().Len())

	enum := NewEnum("Enum").AddValue(NewEnumValue("ZERO")).AddValue(NewEnumValue("ONE"))
	assert.True(t, enum.TryRemoveValue("ONE"))
	assert.False(t, enum.TryRemoveValue("ONE"))
	enum.RemoveValue("ONE")
	assert.Nil(t, enum.GetValue("ONE"))

	req := RpcTypeMessage(NewMessage("Req"), false)
	svc := NewService("Svc").AddMethod(NewMethod("Method", req, req))
	assert.True(t, svc.TryRemoveMethod("Method"))
	assert.False(t, svc.TryRemoveMethod("Method"))
	svc.RemoveMethod("Method")
	assert.Nil(t, svc.GetMethod("Method"))
}

func TestTryAddOneofRollback(t *testing.T) {
	t.Parallel()
	msg := NewMessage("Msg").AddField(NewField("existing", FieldTypeString()).SetNumber(3))
	// The first choice is a group, whose message is also in the message's namespace.
	oneof := NewOneof("oneof").
		AddChoice(NewGroupField(NewMessage("Choice1")).SetNumber(1)).
		AddChoice(NewField("choice2", FieldTypeInt32()).SetNumber(2)).
		AddChoice(NewField("choice3", FieldTypeInt32()).SetNumber(3))
	require.ErrorContains(t, msg.TryAddOneOf(oneof), "already contains field with tag 3")
	assert.Nil(t, oneof.Parent())

	// None of the oneof's names or numbers should have been retained.
	msg.AddOneOf(NewOneof("oneof"))
	for i, name := range []protoreflect.Name{"choice1", "choice2"} {
		require.NoError(t, msg.TryAddField(NewField(name, FieldTypeBool()).SetNumber(protoreflect.FieldNumber(i+1))), "re-using name %s", name)
	}
	require.NoError(t, msg.TryAddNestedMessage(NewMessage("Choice1")))
}

func TestBuilderSetters(t *testing.T) {
	t.Parallel()
	comments := func(s string) Comments {
		return Comments{LeadingComment: " " + s + "\n"}
	}
	enumVal := NewEnumValue("ONE").SetNumber(1).SetComments(comments("enum value")).
		SetOptions(&descriptorpb.EnumValueOptions{Deprecated: new(true)})
	assert.True(t, enumVal.HasNumber())
	assert.Equal(t, protoreflect.EnumNumber(1), enumVal.Number())
	enumVal.ClearNumber()
	assert.False(t, enumVal.HasNumber())
	enumVal.SetNumber(1)
	enum := NewEnum("Enum").AddValue(NewEnumValue("ZERO")).AddValue(enumVal).
		SetComments(comments("enum")).
		SetOptions(&descriptorpb.EnumOptions{Deprecated: new(true)}).
		AddReservedRange(10, 20).AddReservedName("TEN")
	enum.SetReservedRanges(append(enum.ReservedRanges, EnumRange{30, 40}))
	enum.SetReservedNames(append(enum.ReservedNames, "THIRTY"))

	field := NewField("field", FieldTypeInt32()).SetNumber(1).SetJsonName("customName").
		SetComments(comments("field")).
		SetOptions(&descriptorpb.FieldOptions{Deprecated: new(true)})
	field.SetType(FieldTypeEnum(enum))
	oneof := NewOneof("oneof").AddChoice(NewField("choice", FieldTypeBytes()).SetNumber(2)).
		SetComments(comments("oneof")).
		SetOptions(&descriptorpb.OneofOptions{})
	msg := NewMessage("Msg").AddField(field).AddOneOf(oneof).
		SetComments(comments("message")).
		SetOptions(&descriptorpb.MessageOptions{Deprecated: new(true)}).
		SetReservedRanges([]FieldRange{{10, 20}}).
		SetReservedNames([]protoreflect.Name{"ten"})
	req := RpcTypeMessage(msg, false)
	method := NewMethod("Method", req, req).
		SetRequestType(RpcTypeMessage(msg, true)).
		SetResponseType(RpcTypeMessage(msg, true)).
		SetComments(comments("method")).
		SetOptions(&descriptorpb.MethodOptions{Deprecated: new(true)})
	svc := NewService("Svc").AddMethod(method).
		SetComments(comments("service")).
		SetOptions(&descriptorpb.ServiceOptions{Deprecated: new(true)})
	file := NewFile("before.proto").SetPath("test.proto").SetPackageName("test").
		SetEdition(descriptorpb.Edition_EDITION_2023).
		SetComments(comments("file")).
		SetSyntaxComments(comments("syntax")).
		SetPackageComments(comments("package")).
		SetOptions(&descriptorpb.FileOptions{Deprecated: new(true)}).
		AddEnum(enum).AddMessage(msg).AddService(svc)
	assert.Equal(t, "test.proto", file.Path())
	assert.Same(t, file, field.ParentFile())
	assert.Same(t, file, file.ParentFile())

	fd, err := file.Build()
	require.NoError(t, err)
	assert.Equal(t, "test.proto", fd.Path())
	assert.Equal(t, protoreflect.Editions, fd.Syntax())
	assert.True(t, fd.Options().(*descriptorpb.FileOptions).GetDeprecated())

	ed := fd.Enums().ByName("Enum")
	assert.True(t, ed.Options().(*descriptorpb.EnumOptions).GetDeprecated())
	assert.Equal(t, 2, ed.ReservedRanges().Len())
	assert.Equal(t, 2, ed.ReservedNames().Len())
	assert.True(t, ed.Values().ByName("ONE").Options().(*descriptorpb.EnumValueOptions).GetDeprecated())

	md := fd.Messages().ByName("Msg")
	assert.True(t, md.Options().(*descriptorpb.MessageOptions).GetDeprecated())
	assert.Equal(t, 1, md.ReservedRanges().Len())
	assert.Equal(t, 1, md.ReservedNames().Len())
	fld := md.Fields().ByName("field")
	assert.Equal(t, "customName", fld.JSONName())
	assert.Equal(t, protoreflect.EnumKind, fld.Kind())
	assert.True(t, fld.Options().(*descriptorpb.FieldOptions).GetDeprecated())

	mtd := fd.Services().ByName("Svc").Methods().ByName("Method")
	assert.True(t, mtd.IsStreamingClient())
	assert.True(t, mtd.IsStreamingServer())
	assert.True(t, mtd.Options().(*descriptorpb.MethodOptions).GetDeprecated())

	// Comments are recorded in source info.
	for _, descriptor := range []protoreflect.Descriptor{fd, ed, ed.Values().ByName("ONE"), md, fld, md.Oneofs().ByName("oneof"), fd.Services().ByName("Svc"), mtd} {
		loc := fd.SourceLocations().ByDescriptor(descriptor)
		assert.NotEmpty(t, loc.LeadingComments, "comments for %s", descriptor.FullName())
	}
}

func TestScalarFieldTypes(t *testing.T) {
	t.Parallel()
	fieldTypes := map[protoreflect.Kind]*FieldType{
		protoreflect.Int32Kind:    FieldTypeInt32(),
		protoreflect.Uint32Kind:   FieldTypeUint32(),
		protoreflect.Sint32Kind:   FieldTypeSint32(),
		protoreflect.Fixed32Kind:  FieldTypeFixed32(),
		protoreflect.Sfixed32Kind: FieldTypeSfixed32(),
		protoreflect.Int64Kind:    FieldTypeInt64(),
		protoreflect.Uint64Kind:   FieldTypeUint64(),
		protoreflect.Sint64Kind:   FieldTypeSint64(),
		protoreflect.Fixed64Kind:  FieldTypeFixed64(),
		protoreflect.Sfixed64Kind: FieldTypeSfixed64(),
		protoreflect.FloatKind:    FieldTypeFloat(),
		protoreflect.DoubleKind:   FieldTypeDouble(),
		protoreflect.BoolKind:     FieldTypeBool(),
		protoreflect.StringKind:   FieldTypeString(),
		protoreflect.BytesKind:    FieldTypeBytes(),
	}
	msg := NewMessage("Msg")
	number := protoreflect.FieldNumber(1)
	for kind, fieldType := range fieldTypes {
		msg.AddField(NewField(protoreflect.Name("f_"+kind.String()), fieldType).SetNumber(number))
		number++
	}
	md, err := msg.Build()
	require.NoError(t, err)
	for kind := range fieldTypes {
		fld := md.Fields().ByName(protoreflect.Name("f_" + kind.String()))
		if assert.NotNil(t, fld, "field for %v", kind) {
			assert.Equal(t, kind, fld.Kind())
		}
	}
}

func TestProto3OptionalField(t *testing.T) {
	t.Parallel()
	field := NewField("field", FieldTypeString()).SetNumber(1)
	assert.False(t, field.IsOptional())
	field.SetOptional()
	assert.True(t, field.IsOptional())
	md, err := NewMessage("Msg").AddField(field).Build()
	require.NoError(t, err)
	assert.True(t, md.Fields().ByName("field").HasPresence())
}
