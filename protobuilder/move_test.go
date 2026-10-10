package protobuilder

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Adding an element to a builder moves it from its current parent, if it has
// one. These tests verify that the old parent no longer refers to the element
// and that the element's names and numbers are free for re-use in the old
// parent.
func TestMovingBuilders(t *testing.T) {
	t.Parallel()

	t.Run("field between messages", func(t *testing.T) {
		t.Parallel()
		field := NewField("field", FieldTypeString()).SetNumber(1)
		from := NewMessage("From").AddField(field)
		to := NewMessage("To")
		to.AddField(field)
		assert.Same(t, to, field.Parent())
		assert.Same(t, field, to.GetField("field"))
		checkMovedFrom(t, from, "field", 1)
		buildAll(t, from, to)
	})
	t.Run("field into oneof of same message", func(t *testing.T) {
		t.Parallel()
		field := NewField("field", FieldTypeString()).SetNumber(1)
		oneof := NewOneof("oneof")
		msg := NewMessage("Msg").AddField(field).AddOneOf(oneof)
		oneof.AddChoice(field)
		assert.Same(t, oneof, field.Parent())
		assert.Same(t, field, oneof.GetChoice("field"))
		// Choices are still in the message's namespace.
		assert.Same(t, field, msg.GetField("field"))
		checkChildren(t, msg, oneof)
		md, err := msg.Build()
		require.NoError(t, err)
		assert.Equal(t, protoreflect.Name("oneof"), md.Fields().ByName("field").ContainingOneof().Name())

		// And back again.
		msg.AddField(field)
		assert.Same(t, msg, field.Parent())
		assert.Nil(t, oneof.GetChoice("field"))
		checkChildren(t, msg, oneof, field)
		msg.RemoveOneOf("oneof") // an empty oneof can't be built
		md, err = msg.Build()
		require.NoError(t, err)
		assert.Nil(t, md.Fields().ByName("field").ContainingOneof())
	})
	t.Run("field between oneofs of same message", func(t *testing.T) {
		t.Parallel()
		field := NewField("field", FieldTypeString()).SetNumber(1)
		from := NewOneof("from").AddChoice(field)
		to := NewOneof("to")
		msg := NewMessage("Msg").AddOneOf(from).AddOneOf(to)
		to.AddChoice(field)
		assert.Same(t, to, field.Parent())
		assert.Nil(t, from.GetChoice("field"))
		assert.Same(t, field, msg.GetField("field"))
		from.AddChoice(NewField("other", FieldTypeString()).SetNumber(2))
		md, err := msg.Build()
		require.NoError(t, err)
		assert.Equal(t, protoreflect.Name("to"), md.Fields().ByName("field").ContainingOneof().Name())
	})
	t.Run("field from oneof to other message", func(t *testing.T) {
		t.Parallel()
		field := NewField("field", FieldTypeString()).SetNumber(1)
		oneof := NewOneof("oneof").AddChoice(field)
		from := NewMessage("From").AddOneOf(oneof)
		to := NewMessage("To")
		to.AddField(field)
		assert.Same(t, to, field.Parent())
		assert.Nil(t, oneof.GetChoice("field"))
		checkMovedFrom(t, from, "field", 1)
		buildAll(t, to)
	})
	t.Run("field into oneof without message", func(t *testing.T) {
		t.Parallel()
		field := NewField("field", FieldTypeString()).SetNumber(1)
		from := NewMessage("From").AddField(field)
		oneof := NewOneof("oneof").AddChoice(field)
		assert.Same(t, oneof, field.Parent())
		assert.Empty(t, from.Children())
		checkMovedFrom(t, from, "field", 1)
		buildAll(t, from)
	})
	t.Run("oneof between messages", func(t *testing.T) {
		t.Parallel()
		oneof := NewOneof("oneof").
			AddChoice(NewField("choice1", FieldTypeString()).SetNumber(1)).
			AddChoice(NewField("choice2", FieldTypeString()).SetNumber(2))
		from := NewMessage("From").AddOneOf(oneof)
		to := NewMessage("To")
		to.AddOneOf(oneof)
		assert.Same(t, to, oneof.Parent())
		assert.Same(t, oneof.GetChoice("choice1"), to.GetField("choice1"))
		assert.Nil(t, from.GetOneOf("oneof"))
		checkMovedFrom(t, from, "choice1", 1)
		checkMovedFrom(t, from, "choice2", 2)
		from.AddOneOf(NewOneof("oneof").AddChoice(NewField("choice3", FieldTypeString()).SetNumber(3)))
		buildAll(t, from, to)
	})
	t.Run("group type out of its field", func(t *testing.T) {
		t.Parallel()
		group := NewMessage("Group")
		groupField := NewGroupField(group).SetNumber(1)
		msg := NewMessage("Msg").AddField(groupField)
		msg.AddNestedMessage(group)
		assert.Same(t, msg, group.Parent())
		assert.Same(t, group, msg.GetNestedMessage("Group"))
		assert.Empty(t, groupField.Children())
		checkChildren(t, msg, groupField, group)
	})
	t.Run("nested types between messages", func(t *testing.T) {
		t.Parallel()
		nested := NewMessage("Nested")
		enum := NewEnum("Enum").AddValue(NewEnumValue("ZERO"))
		from := NewMessage("From").AddNestedMessage(nested).AddNestedEnum(enum).AddExtensionRange(100, 200)
		ext := NewExtension("ext", 100, FieldTypeString(), from)
		from.AddNestedExtension(ext)
		to := NewMessage("To").AddNestedMessage(nested).AddNestedEnum(enum).AddNestedExtension(ext)
		assert.Same(t, to, nested.Parent())
		assert.Same(t, to, enum.Parent())
		assert.Same(t, to, ext.Parent())
		assert.Empty(t, from.Children())
		// The names are free for re-use.
		from.AddNestedMessage(NewMessage("Nested")).
			AddNestedEnum(NewEnum("Enum").AddValue(NewEnumValue("ZERO"))).
			AddNestedExtension(NewExtension("ext", 101, FieldTypeString(), from))
		NewFile("from.proto").AddMessage(from)
		NewFile("to.proto").AddMessage(to)
		buildAll(t, from, to)
	})
	t.Run("elements between files and messages", func(t *testing.T) {
		t.Parallel()
		msg := NewMessage("Msg").AddExtensionRange(100, 200)
		enum := NewEnum("Enum").AddValue(NewEnumValue("ZERO"))
		ext := NewExtension("ext", 100, FieldTypeString(), msg)
		svc := NewService("Svc")
		from := NewFile("from.proto").AddMessage(msg).AddEnum(enum).AddExtension(ext).AddService(svc)
		to := NewFile("to.proto").AddMessage(msg).AddEnum(enum).AddExtension(ext).AddService(svc)
		for _, b := range []Builder{msg, enum, ext, svc} {
			assert.Same(t, to, b.Parent(), "parent of %s", b.Name())
		}
		assert.Empty(t, from.Children())

		// From a file into a message, and back.
		container := NewMessage("Container")
		to.AddMessage(container)
		container.AddNestedEnum(enum)
		assert.Same(t, container, enum.Parent())
		assert.Nil(t, to.GetEnum("Enum"))
		to.AddEnum(enum)
		assert.Same(t, to, enum.Parent())
		assert.Nil(t, container.GetNestedEnum("Enum"))
		_, err := to.Build()
		require.NoError(t, err)
	})
	t.Run("enum values and methods", func(t *testing.T) {
		t.Parallel()
		value := NewEnumValue("VALUE").SetNumber(1)
		fromEnum := NewEnum("From").AddValue(NewEnumValue("FROM_ZERO")).AddValue(value)
		toEnum := NewEnum("To").AddValue(NewEnumValue("TO_ZERO")).AddValue(value)
		assert.Same(t, toEnum, value.Parent())
		assert.Nil(t, fromEnum.GetValue("VALUE"))
		fromEnum.AddValue(NewEnumValue("VALUE").SetNumber(1))

		req := RPCTypeMessage(NewMessage("Req"), false)
		method := NewMethod("Method", req, req)
		fromSvc := NewService("From").AddMethod(method)
		toSvc := NewService("To").AddMethod(method)
		assert.Same(t, toSvc, method.Parent())
		assert.Nil(t, fromSvc.GetMethod("Method"))
		fromSvc.AddMethod(NewMethod("Method", req, req))
	})
	t.Run("failed move", func(t *testing.T) {
		t.Parallel()
		// When an element can't be added, it stays where it was.
		field := NewField("field", FieldTypeString()).SetNumber(1)
		from := NewMessage("From").AddField(field)
		to := NewMessage("To").AddField(NewField("field", FieldTypeInt32()).SetNumber(2))
		require.ErrorContains(t, to.TryAddField(field), "already contains element")
		assert.Same(t, from, field.Parent())
		assert.Same(t, field, from.GetField("field"))

		oneof := NewOneof("oneof")
		to.AddOneOf(oneof)
		require.ErrorContains(t, oneof.TryAddChoice(field), "already contains element")
		assert.Same(t, from, field.Parent())
		assert.Nil(t, oneof.GetChoice("field"))
		oneof.AddChoice(NewField("choice", FieldTypeString()).SetNumber(3))

		numbered := NewMessage("Numbered").AddField(NewField("other", FieldTypeInt32()).SetNumber(1))
		require.ErrorContains(t, numbered.TryAddField(field), "already contains field with tag 1")
		assert.Same(t, from, field.Parent())
		assert.Nil(t, numbered.GetField("field"))
		buildAll(t, from, to, numbered)
	})
}

func TestRenumberingFields(t *testing.T) {
	t.Parallel()

	t.Run("invalid numbers", func(t *testing.T) {
		t.Parallel()
		field := NewField("field", FieldTypeString()).SetNumber(1)
		NewMessage("Msg").AddField(field)
		assert.ErrorContains(t, field.TrySetNumber(-1), "negative value")
		assert.ErrorContains(t, field.TrySetNumber(19000), "special reserved range")
		assert.ErrorContains(t, field.TrySetNumber(19999), "special reserved range")
		assert.ErrorContains(t, field.TrySetNumber(math.MaxInt32), "cannot be above max")
		assert.Equal(t, protoreflect.FieldNumber(1), field.Number())
		assert.Panics(t, func() { field.SetNumber(-1) })

		// Numbers above the max for normal messages are allowed, since the
		// message could use the message set wire format, but fail to build.
		require.NoError(t, field.TrySetNumber(536870912))
		_, err := field.Parent().(*MessageBuilder).Build()
		assert.ErrorContains(t, err, "cannot be above max 536870911")
		field.SetNumber(1)

		extendee := NewMessage("Extendee").AddExtensionRange(100, 200)
		ext := NewExtension("ext", 100, FieldTypeString(), extendee)
		assert.ErrorContains(t, ext.TrySetNumber(0), "only regular fields can be auto-assigned")
		assert.Equal(t, protoreflect.FieldNumber(100), ext.Number())
	})
	t.Run("in message", func(t *testing.T) {
		t.Parallel()
		field1 := NewField("field1", FieldTypeString()).SetNumber(1)
		field2 := NewField("field2", FieldTypeString()).SetNumber(2)
		msg := NewMessage("Msg").AddField(field1).AddField(field2)
		require.ErrorContains(t, field2.TrySetNumber(1), "already contains field with tag 1")
		assert.Equal(t, protoreflect.FieldNumber(2), field2.Number())

		require.NoError(t, field2.TrySetNumber(3))
		require.NoError(t, field2.TrySetNumber(3), "setting same number is a no-op")
		// The old number is free for re-use.
		msg.AddField(NewField("field3", FieldTypeString()).SetNumber(2))
		// But the new one isn't.
		require.ErrorContains(t, msg.TryAddField(NewField("field4", FieldTypeString()).SetNumber(3)), "already contains field with tag 3")

		md, err := msg.Build()
		require.NoError(t, err)
		assert.Equal(t, protoreflect.FieldNumber(3), md.Fields().ByName("field2").Number())
	})
	t.Run("in oneof", func(t *testing.T) {
		t.Parallel()
		choice := NewField("choice", FieldTypeString()).SetNumber(2)
		oneof := NewOneof("oneof").AddChoice(choice)
		msg := NewMessage("Msg").AddField(NewField("field", FieldTypeString()).SetNumber(1)).AddOneOf(oneof)
		require.ErrorContains(t, choice.TrySetNumber(1), "already contains field with tag 1")
		assert.Equal(t, protoreflect.FieldNumber(2), choice.Number())

		require.NoError(t, choice.TrySetNumber(3))
		msg.AddField(NewField("other", FieldTypeString()).SetNumber(2))
		require.ErrorContains(t, msg.TryAddField(NewField("another", FieldTypeString()).SetNumber(3)), "already contains field with tag 3")
		md, err := msg.Build()
		require.NoError(t, err)
		assert.Equal(t, protoreflect.FieldNumber(3), md.Fields().ByName("choice").Number())
	})
	t.Run("in oneof without message", func(t *testing.T) {
		t.Parallel()
		choice := NewField("choice", FieldTypeString()).SetNumber(1)
		oneof := NewOneof("oneof").AddChoice(choice)
		require.NoError(t, choice.TrySetNumber(2))
		// The number is checked when the oneof is added to a message.
		msg := NewMessage("Msg").AddField(NewField("field", FieldTypeString()).SetNumber(2))
		require.ErrorContains(t, msg.TryAddOneOf(oneof), "already contains field with tag 2")
	})
	t.Run("auto-assigned", func(t *testing.T) {
		t.Parallel()
		field1 := NewField("field1", FieldTypeString()).SetNumber(1)
		field2 := NewField("field2", FieldTypeString()).SetNumber(2)
		msg := NewMessage("Msg").AddField(field1).AddField(field2)
		// Zero means auto-assign, so several fields can have it.
		require.NoError(t, field1.TrySetNumber(0))
		require.NoError(t, field2.TrySetNumber(0))
		msg.AddField(NewField("field3", FieldTypeString()).SetNumber(1))
		md, err := msg.Build()
		require.NoError(t, err)
		numbers := map[protoreflect.FieldNumber]struct{}{}
		for i := range md.Fields().Len() {
			numbers[md.Fields().Get(i).Number()] = struct{}{}
		}
		assert.Len(t, numbers, 3, "fields should have distinct numbers")
	})
	t.Run("extension", func(t *testing.T) {
		t.Parallel()
		// Extension numbers are not tracked in the extended message.
		extendee := NewMessage("Extendee").AddExtensionRange(100, 200)
		ext := NewExtension("ext", 100, FieldTypeString(), extendee)
		NewFile("test.proto").AddMessage(extendee).AddExtension(ext)
		require.NoError(t, ext.TrySetNumber(101))
		fd, err := ext.ParentFile().Build()
		require.NoError(t, err)
		assert.Equal(t, protoreflect.FieldNumber(101), fd.Extensions().ByName("ext").Number())
	})
}

// checkMovedFrom verifies that the given message no longer has a field with
// the given name and number, and that both are free for re-use.
func checkMovedFrom(t *testing.T, msg *MessageBuilder, name protoreflect.Name, number protoreflect.FieldNumber) {
	t.Helper()
	assert.Nil(t, msg.GetField(name))
	require.NoError(t, msg.TryAddField(NewField(name, FieldTypeBool()).SetNumber(number)), "re-using name %s and number %d", name, number)
}

func buildAll(t *testing.T, msgs ...*MessageBuilder) {
	t.Helper()
	for _, msg := range msgs {
		_, err := msg.Build()
		require.NoError(t, err, "building %s", msg.Name())
	}
}
