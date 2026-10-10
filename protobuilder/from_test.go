package protobuilder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestFindElement(t *testing.T) {
	t.Parallel()
	msg := NewMessage("Msg").
		AddNestedMessage(NewMessage("Nested")).
		AddNestedEnum(NewEnum("NestedEnum").AddValue(NewEnumValue("NESTED_VALUE"))).
		AddField(NewField("field", FieldTypeString())).
		AddField(NewMapField("map_field", FieldTypeString(), FieldTypeString())).
		AddOneOf(NewOneof("choice").AddChoice(NewField("choice_field", FieldTypeBool()))).
		AddExtensionRange(100, 200)
	msg.AddNestedExtension(NewExtension("nested_ext", 100, FieldTypeString(), msg))
	file := NewFile("find.proto").
		SetPackageName("foo.bar").
		AddMessage(msg).
		AddEnum(NewEnum("Enum").AddValue(NewEnumValue("VALUE"))).
		AddService(NewService("Svc").AddMethod(NewMethod("Do", RPCTypeMessage(msg, false), RPCTypeMessage(msg, false))))

	testCases := []struct {
		name     protoreflect.FullName
		expected Builder
	}{
		{"foo.bar.Msg", msg},
		{"foo.bar.Msg.Nested", msg.GetNestedMessage("Nested")},
		{"foo.bar.Msg.NestedEnum", msg.GetNestedEnum("NestedEnum")},
		// Enum values are in the scope of the enum's parent.
		{"foo.bar.Msg.NESTED_VALUE", msg.GetNestedEnum("NestedEnum").GetValue("NESTED_VALUE")},
		{"foo.bar.Msg.field", msg.GetField("field")},
		{"foo.bar.Msg.MapFieldEntry", msg.GetField("map_field").msgType},
		{"foo.bar.Msg.MapFieldEntry.key", msg.GetField("map_field").msgType.GetField("key")},
		{"foo.bar.Msg.choice", msg.GetOneOf("choice")},
		{"foo.bar.Msg.choice_field", msg.GetOneOf("choice").GetChoice("choice_field")},
		{"foo.bar.Msg.nested_ext", msg.GetNestedExtension("nested_ext")},
		{"foo.bar.Enum", file.GetEnum("Enum")},
		{"foo.bar.VALUE", file.GetEnum("Enum").GetValue("VALUE")},
		{"foo.bar.Svc", file.GetService("Svc")},
		{"foo.bar.Svc.Do", file.GetService("Svc").GetMethod("Do")},
		// Not found
		{"foo.bar.Msg.choice.choice_field", nil},
		{"foo.bar.Msg.Unknown", nil},
		{"foo.bar.Msg.field.nothing", nil},
		{"foo.bar", nil},
		{"foo.Msg", nil},
		{"Msg", nil},
		{"other.foo.bar.Msg", nil},
	}
	for _, testCase := range testCases {
		t.Run(string(testCase.name), func(t *testing.T) {
			t.Parallel()
			actual := file.FindElement(testCase.name)
			if testCase.expected == nil {
				require.Nil(t, actual)
			} else {
				require.Same(t, testCase.expected, actual)
			}
		})
	}

	// Also works for files with no package.
	noPkg := NewFile("no_pkg.proto").AddMessage(NewMessage("Msg").AddNestedMessage(NewMessage("Nested")))
	require.Same(t, noPkg.GetMessage("Msg"), noPkg.FindElement("Msg"))
	require.Same(t, noPkg.GetMessage("Msg").GetNestedMessage("Nested"), noPkg.FindElement("Msg.Nested"))
	require.Nil(t, noPkg.FindElement("foo.Msg"))
}

func TestFindElement_FromFile(t *testing.T) {
	t.Parallel()
	fd, err := protoregistry.GlobalFiles.FindFileByPath("desc_test1.proto")
	require.NoError(t, err)
	fb, err := FromFile(fd)
	require.NoError(t, err)
	checkFindElements(t, fb, fd.Messages())
}

func checkFindElements(t *testing.T, fb *FileBuilder, msgs protoreflect.MessageDescriptors) {
	t.Helper()
	for i := range msgs.Len() {
		md := msgs.Get(i)
		b := fb.FindElement(md.FullName())
		require.IsType(t, (*MessageBuilder)(nil), b, "%s", md.FullName())
		require.Equal(t, md.Name(), b.Name())
		for j := range md.Fields().Len() {
			fld := md.Fields().Get(j)
			b := fb.FindElement(fld.FullName())
			require.IsType(t, (*FieldBuilder)(nil), b, "%s", fld.FullName())
			require.Equal(t, fld.Name(), b.Name())
		}
		checkFindElements(t, fb, md.Messages())
	}
}

// TestBuild_CopiesOfSameFile is a regression test for a bug where building
// failed with an error that a file was already registered when the builders
// referred to two separate but identical copies of the same file.
func TestBuild_CopiesOfSameFile(t *testing.T) {
	t.Parallel()
	doubleValue := (*wrapperspb.DoubleValue)(nil).ProtoReflect().Descriptor()
	stringValue := (*wrapperspb.StringValue)(nil).ProtoReflect().Descriptor()
	newFile := func(types ...*FieldType) *FileBuilder {
		msg := NewMessage("TestMessage")
		for i, typ := range types {
			msg.AddField(NewField(protoreflect.Name("value_"+string(rune('a'+i))), typ))
		}
		return NewFile("test.proto").SetSyntax(protoreflect.Proto3).AddMessage(msg)
	}

	t.Run("identical copies", func(t *testing.T) {
		t.Parallel()
		// Each of these has its own copy of wrappers.proto.
		doubleBuilder, err := FromMessage(doubleValue)
		require.NoError(t, err)
		stringBuilder, err := FromMessage(stringValue)
		require.NoError(t, err)
		require.NotSame(t, doubleBuilder.ParentFile(), stringBuilder.ParentFile())

		fd, err := newFile(FieldTypeMessage(doubleBuilder), FieldTypeMessage(stringBuilder)).Build()
		require.NoError(t, err)
		require.Equal(t, 1, fd.Imports().Len())
		fields := fd.Messages().Get(0).Fields()
		require.Equal(t, doubleValue.FullName(), fields.Get(0).Message().FullName())
		require.Equal(t, stringValue.FullName(), fields.Get(1).Message().FullName())
		require.Same(t, fields.Get(0).Message().ParentFile(), fields.Get(1).Message().ParentFile())
	})

	t.Run("copy and descriptor", func(t *testing.T) {
		t.Parallel()
		doubleBuilder, err := FromMessage(doubleValue)
		require.NoError(t, err)
		for _, types := range [][]*FieldType{
			{FieldTypeMessage(doubleBuilder), FieldTypeImportedMessage(stringValue)},
			{FieldTypeImportedMessage(stringValue), FieldTypeMessage(doubleBuilder)},
		} {
			fd, err := newFile(types...).Build()
			require.NoError(t, err)
			require.Equal(t, 1, fd.Imports().Len())
		}
	})

	t.Run("copy after descriptor", func(t *testing.T) {
		t.Parallel()
		// The original wrappers.proto is registered as a dependency of other.proto
		// before the copy is built.
		other := NewMessage("Other").AddField(NewField("value", FieldTypeImportedMessage(doubleValue)))
		NewFile("other.proto").AddMessage(other)
		stringBuilder, err := FromMessage(stringValue)
		require.NoError(t, err)
		fd, err := newFile(FieldTypeMessage(other), FieldTypeMessage(stringBuilder)).Build()
		require.NoError(t, err)
		require.Equal(t, 2, fd.Imports().Len())
	})

	t.Run("different copies", func(t *testing.T) {
		t.Parallel()
		doubleBuilder, err := FromMessage(doubleValue)
		require.NoError(t, err)
		stringBuilder, err := FromMessage(stringValue)
		require.NoError(t, err)
		stringBuilder.AddField(NewField("extra", FieldTypeString()))

		_, err = newFile(FieldTypeMessage(doubleBuilder), FieldTypeMessage(stringBuilder)).Build()
		require.ErrorContains(t, err, "multiple versions of descriptors found with same file path: google/protobuf/wrappers.proto")
		require.ErrorContains(t, err, "use FromFile and FileBuilder.FindElement")

		// The suggested alternative works.
		file, err := FromFile(doubleValue.ParentFile())
		require.NoError(t, err)
		doubleBuilder = file.FindElement(doubleValue.FullName()).(*MessageBuilder)
		stringBuilder = file.FindElement(stringValue.FullName()).(*MessageBuilder)
		stringBuilder.AddField(NewField("extra", FieldTypeString()))
		fd, err := newFile(FieldTypeMessage(doubleBuilder), FieldTypeMessage(stringBuilder)).Build()
		require.NoError(t, err)
		require.NotNil(t, fd.Messages().Get(0).Fields().Get(1).Message().Fields().ByName("extra"))
	})
}
