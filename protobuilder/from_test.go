package protobuilder

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
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
		AddService(NewService("Svc").AddMethod(NewMethod("Do", RpcTypeMessage(msg, false), RpcTypeMessage(msg, false))))

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
