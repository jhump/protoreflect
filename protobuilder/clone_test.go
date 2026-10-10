package protobuilder

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestClone_Files(t *testing.T) {
	t.Parallel()
	for _, path := range []string{
		"desc_test1.proto", "desc_test2.proto",
		"desc_test_defaults.proto", "desc_test_editions.proto", "desc_test_options.proto",
		"desc_test_proto3.proto", "desc_test_wellknowntypes.proto",
		"nopkg/desc_test_nopkg.proto", "nopkg/desc_test_nopkg_new.proto", "pkg/desc_test_pkg.proto"} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			fd, err := protoregistry.GlobalFiles.FindFileByPath(path)
			require.NoError(t, err)
			fb, err := FromFile(fd)
			require.NoError(t, err)
			expected, err := fb.Build()
			require.NoError(t, err)

			clone := Clone(fb)
			// Comparing these large trees with checkClonedTree is slow, so
			// we compare the built descriptors instead.
			checkNoSharedBuilders(t, fb, clone)
			cloned, err := clone.Build()
			require.NoError(t, err)
			checkDescriptors(t, expected, cloned)

			// Change the copy. This must not affect the original.
			clone.SetPath("copy/" + path)
			for _, mb := range clone.messages {
				mb.AddField(NewField("clone_only", FieldTypeString()).SetNumber(9999))
			}
			cloned, err = clone.Build()
			require.NoError(t, err)
			// The copy's references to its own elements must refer to the
			// copies. Otherwise, the original file would be a dependency.
			for i := range cloned.Imports().Len() {
				require.NotEqual(t, path, cloned.Imports().Get(i).Path())
			}
			for i := range cloned.Messages().Len() {
				require.NotNil(t, cloned.Messages().Get(i).Fields().ByName("clone_only"))
			}

			actual, err := fb.Build()
			require.NoError(t, err)
			checkDescriptors(t, expected, actual)
		})
	}
}

func TestClone_Message(t *testing.T) {
	t.Parallel()
	other := NewMessage("Other")
	inner := NewMessage("Inner")
	nestedEnum := NewEnum("Kind").AddValue(NewEnumValue("KIND_UNSPECIFIED"))
	oneof := NewOneof("choice").
		AddChoice(NewField("choice_inner", FieldTypeMessage(inner))).
		AddChoice(NewField("choice_str", FieldTypeString()))
	outer := NewMessage("Outer").
		AddNestedMessage(inner).
		AddNestedEnum(nestedEnum).
		AddField(NewField("inner", FieldTypeMessage(inner))).
		AddField(NewField("other", FieldTypeMessage(other))).
		AddField(NewField("kind", FieldTypeEnum(nestedEnum))).
		AddField(NewMapField("inners", FieldTypeString(), FieldTypeMessage(inner))).
		AddField(NewGroupField(NewMessage("Group").AddField(NewField("inner", FieldTypeMessage(inner))))).
		AddOneOf(oneof).
		AddExtensionRangeWithOptions(100, 200, &descriptorpb.ExtensionRangeOptions{Verification: new(descriptorpb.ExtensionRangeOptions_UNVERIFIED)}).
		SetOptions(&descriptorpb.MessageOptions{Deprecated: new(true)}).
		SetComments(Comments{LeadingDetachedComments: []string{"detached"}, LeadingComment: "leading"})
	file := NewFile("clone_test.proto").SetPackageName("clone.test").AddMessage(outer).AddMessage(other)
	expected, err := file.Build()
	require.NoError(t, err)

	clone := Clone(outer)
	require.Nil(t, clone.Parent())
	checkClonedTree(t, outer, clone)

	// References to elements in the copied tree refer to the copies.
	cloneInner := clone.GetNestedMessage("Inner")
	cloneEnum := clone.GetNestedEnum("Kind")
	require.Same(t, cloneInner, clone.GetField("inner").fieldType.localMsgType)
	require.Same(t, cloneEnum, clone.GetField("kind").fieldType.localEnumType)
	require.Same(t, cloneInner, clone.GetOneOf("choice").GetChoice("choice_inner").fieldType.localMsgType)
	mapField := clone.GetField("inners")
	require.Same(t, mapField.msgType, mapField.fieldType.localMsgType)
	require.Same(t, cloneInner, mapField.msgType.GetField("value").fieldType.localMsgType)
	groupField := clone.GetField("group")
	require.Same(t, groupField.msgType, groupField.fieldType.localMsgType)
	require.Same(t, cloneInner, groupField.msgType.GetField("inner").fieldType.localMsgType)
	// Other references are unchanged.
	require.Same(t, other, clone.GetField("other").fieldType.localMsgType)

	// The copy can be changed without affecting the original.
	clone.Options.Deprecated = new(false)
	clone.ExtensionRanges[0].Options.Verification = new(descriptorpb.ExtensionRangeOptions_DECLARATION)
	clone.Comments().LeadingDetachedComments[0] = "changed"
	cloneInner.SetName("Changed")
	clone.GetField("other").SetNumber(1000)
	clone.RemoveField("inner")
	clone.GetOneOf("choice").RemoveChoice("choice_str")
	cloneEnum.AddValue(NewEnumValue("KIND_CHANGED"))
	other.AddNestedMessage(clone.SetName("OuterCopy"))
	actual, err := file.Build()
	require.NoError(t, err)
	// The original's file now also has the copy, nested in Other.
	require.NotNil(t, actual.Messages().ByName("Other").Messages().ByName("OuterCopy"))
	other.TryRemoveNestedMessage("OuterCopy")
	actual, err = file.Build()
	require.NoError(t, err)
	checkDescriptors(t, expected, actual)
}

func TestClone_RecursiveTypes(t *testing.T) {
	t.Parallel()
	// Node refers to itself, A and B refer to each other, and B refers to its
	// ancestor, Node.
	node := NewMessage("Node")
	a := NewMessage("A")
	b := NewMessage("B")
	a.AddField(NewField("b", FieldTypeMessage(b)))
	b.AddField(NewField("a", FieldTypeMessage(a))).
		AddField(NewField("node", FieldTypeMessage(node)))
	node.AddNestedMessage(a).
		AddNestedMessage(b).
		AddField(NewField("parent", FieldTypeMessage(node))).
		AddField(NewField("children", FieldTypeMessage(node)).SetRepeated()).
		AddField(NewMapField("by_name", FieldTypeString(), FieldTypeMessage(node))).
		AddField(NewField("a", FieldTypeMessage(a)))
	svc := NewService("NodeService").
		AddMethod(NewMethod("Walk", RpcTypeMessage(node, false), RpcTypeMessage(node, true)))
	file := NewFile("recursive.proto").SetPackageName("recursive").AddMessage(node).AddService(svc)
	expected, err := file.Build()
	require.NoError(t, err)

	clone := Clone(node)
	checkClonedTree(t, node, clone)
	cloneA := clone.GetNestedMessage("A")
	cloneB := clone.GetNestedMessage("B")
	require.Same(t, clone, clone.GetField("parent").fieldType.localMsgType)
	require.Same(t, clone, clone.GetField("children").fieldType.localMsgType)
	require.Same(t, clone, clone.GetField("by_name").msgType.GetField("value").fieldType.localMsgType)
	require.Same(t, cloneA, clone.GetField("a").fieldType.localMsgType)
	require.Same(t, cloneB, cloneA.GetField("b").fieldType.localMsgType)
	require.Same(t, cloneA, cloneB.GetField("a").fieldType.localMsgType)
	require.Same(t, clone, cloneB.GetField("node").fieldType.localMsgType)

	// The copy builds on its own, in a different file, without depending on
	// the original.
	cloneFile := NewFile("copy.proto").SetPackageName("copy").AddMessage(clone)
	cloneFd, err := cloneFile.Build()
	require.NoError(t, err)
	require.Equal(t, 0, cloneFd.Imports().Len())
	cloneNode := cloneFd.Messages().ByName("Node")
	require.Equal(t, cloneNode, cloneNode.Fields().ByName("parent").Message())
	require.Equal(t, cloneNode, cloneNode.Messages().ByName("B").Fields().ByName("node").Message())

	// The same goes for a copy of the whole file, including the service.
	fileClone := Clone(file).SetPath("recursive_copy.proto")
	checkNoSharedBuilders(t, file, fileClone)
	walk := fileClone.GetService("NodeService").GetMethod("Walk")
	require.Same(t, fileClone.GetMessage("Node"), walk.ReqType.localType)
	require.Same(t, fileClone.GetMessage("Node"), walk.RespType.localType)
	fileCloneFd, err := fileClone.SetPackageName("recursive_copy").Build()
	require.NoError(t, err)
	require.Equal(t, 0, fileCloneFd.Imports().Len())

	actual, err := file.Build()
	require.NoError(t, err)
	checkDescriptors(t, expected, actual)
}

func TestClone_SameFieldInTwoMessages(t *testing.T) {
	t.Parallel()
	field := NewField("id", FieldTypeString()).SetNumber(1)
	req := NewMessage("Request").AddField(field)
	resp := NewMessage("Response").AddField(Clone(field))
	file := NewFile("rpc.proto").AddMessage(req).AddMessage(resp)
	fd, err := file.Build()
	require.NoError(t, err)
	require.NotNil(t, fd.Messages().ByName("Request").Fields().ByName("id"))
	require.NotNil(t, fd.Messages().ByName("Response").Fields().ByName("id"))
}

func TestClone_EnumAndService(t *testing.T) {
	t.Parallel()
	enum := NewEnum("Color").
		AddValue(NewEnumValue("RED")).
		AddValue(NewEnumValue("GREEN").SetOptions(&descriptorpb.EnumValueOptions{Deprecated: new(true)})).
		AddReservedName("BLUE")
	clone := Clone(enum)
	checkClonedTree(t, enum, clone)
	require.Same(t, clone.GetValue("RED"), clone.symbols["RED"])
	clone.GetValue("GREEN").Options.Deprecated = new(false)
	clone.RemoveValue("RED")
	clone.ReservedNames[0] = "PURPLE"
	require.NotNil(t, enum.GetValue("RED"))
	require.True(t, enum.GetValue("GREEN").Options.GetDeprecated())
	require.Equal(t, []protoreflect.Name{"BLUE"}, enum.ReservedNames)

	msg := NewMessage("Msg")
	method := NewMethod("Do", RpcTypeMessage(msg, false), RpcTypeMessage(msg, true))
	svc := NewService("Svc").AddMethod(method)
	cloneSvc := Clone(svc)
	checkClonedTree(t, svc, cloneSvc)
	cloneMethod := cloneSvc.GetMethod("Do")
	require.Same(t, msg, cloneMethod.ReqType.localType)
	require.Same(t, msg, cloneMethod.RespType.localType)
	cloneMethod.ReqType.IsStream = true
	cloneMethod.SetName("DoOver")
	require.False(t, method.ReqType.IsStream)
	require.Same(t, method, svc.GetMethod("Do"))
	require.Nil(t, svc.GetMethod("DoOver"))

	// The generic interface type can also be used.
	var b Builder = method
	require.IsType(t, (*MethodBuilder)(nil), Clone(b))
}

func TestClone_FileDependencies(t *testing.T) {
	t.Parallel()
	dep := NewFile("dep.proto")
	file := NewFile("file.proto").AddDependency(dep).AddDependency(NewFile("other.proto"))
	file.AddDependency(file) // erroneous, but allowed
	clone := Clone(file)
	require.Len(t, clone.explicitDeps, 3)
	require.Contains(t, clone.explicitDeps, dep)
	require.Contains(t, clone.explicitDeps, clone)
	clone.PruneUnusedDependencies()
	require.Len(t, file.explicitDeps, 3)
}

// checkClonedTree checks that clone is a deep copy of orig: it is equal to
// orig but has no builders in common with it.
func checkClonedTree(t *testing.T, orig, clone Builder) {
	t.Helper()
	checkNoSharedBuilders(t, orig, clone)
	builderPkg := reflect.TypeFor[FileBuilder]().PkgPath()
	diff := cmp.Diff(orig, clone,
		cmp.Exporter(func(typ reflect.Type) bool { return typ.PkgPath() == builderPkg }),
		// Parents are checked above. The root of the copy has no parent.
		cmpopts.IgnoreFields(baseBuilder{}, "parent"),
		cmpopts.IgnoreFields(FileBuilder{}, "origExts"),
		cmpopts.EquateEmpty(),
		cmp.Comparer(func(a, b protoreflect.FileDescriptor) bool { return a == b }),
		cmp.Comparer(func(a, b protoreflect.MessageDescriptor) bool { return a == b }),
		cmp.Comparer(func(a, b protoreflect.EnumDescriptor) bool { return a == b }),
		protocmp.Transform(),
	)
	require.Empty(t, diff)
}

func checkNoSharedBuilders(t *testing.T, orig, clone Builder) {
	t.Helper()
	require.NotSame(t, orig, clone)
	origChildren, cloneChildren := orig.Children(), clone.Children()
	require.Len(t, cloneChildren, len(origChildren))
	for i := range origChildren {
		require.Same(t, clone, cloneChildren[i].Parent())
		checkNoSharedBuilders(t, origChildren[i], cloneChildren[i])
	}
}
