package dynamic

import (
	"sort"
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/desc_test"
	"github.com/jhump/protoreflect/testutil"
)

func TestExtensionRegistry_AddExtension(t *testing.T) {
	er := &ExtensionRegistry{}
	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	er.AddExtension(file.GetExtensions()...)

	fds := er.AllExtensionsForType("desc_test.AnotherTestMessage")
	sort.Sort(fields(fds))

	testutil.Eq(t, []desc.Descriptor{
		file.FindSymbol("desc_test.xtm"),
		file.FindSymbol("desc_test.xs"),
		file.FindSymbol("desc_test.xi"),
		file.FindSymbol("desc_test.xui"),
	}, fds)

	checkFindExtension(t, er, fds)
}

func TestExtensionRegistry_AddExtensionDesc(t *testing.T) {
	er := &ExtensionRegistry{}

	er.AddExtensionDesc(desc_test.E_Xtm, desc_test.E_Xs, desc_test.E_Xi)

	fds := er.AllExtensionsForType("desc_test.AnotherTestMessage")
	sort.Sort(fields(fds))

	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	testutil.Eq(t, 3, len(fds))
	testutil.Eq(t, file.FindSymbol("desc_test.xtm"), fds[0])
	testutil.Eq(t, file.FindSymbol("desc_test.xs"), fds[1])
	testutil.Eq(t, file.FindSymbol("desc_test.xi"), fds[2])

	checkFindExtension(t, er, fds)
}

func TestExtensionRegistry_AddExtensionsFromFile(t *testing.T) {
	er := &ExtensionRegistry{}
	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	er.AddExtensionsFromFile(file)

	fds := er.AllExtensionsForType("desc_test.AnotherTestMessage")
	sort.Sort(fields(fds))

	testutil.Eq(t, 5, len(fds))
	testutil.Eq(t, file.FindSymbol("desc_test.xtm"), fds[0])
	testutil.Eq(t, file.FindSymbol("desc_test.xs"), fds[1])
	testutil.Eq(t, file.FindSymbol("desc_test.xi"), fds[2])
	testutil.Eq(t, file.FindSymbol("desc_test.xui"), fds[3])
	testutil.Eq(t, file.FindSymbol("desc_test.TestMessage.NestedMessage.AnotherNestedMessage.flags"), fds[4])

	checkFindExtension(t, er, fds)
}

func TestExtensionRegistry_Empty(t *testing.T) {
	er := ExtensionRegistry{}
	fds := er.AllExtensionsForType("desc_test.AnotherTestMessage")
	testutil.Eq(t, 0, len(fds))
}

func TestExtensionRegistry_Defaults(t *testing.T) {
	er := NewRegistryWithDefaults()

	fds := er.AllExtensionsForType("desc_test.AnotherTestMessage")
	sort.Sort(fields(fds))

	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	testutil.Eq(t, 5, len(fds))
	testutil.Eq(t, file.FindSymbol("desc_test.xtm").AsProto(), fds[0].AsProto())
	testutil.Eq(t, file.FindSymbol("desc_test.xs").AsProto(), fds[1].AsProto())
	testutil.Eq(t, file.FindSymbol("desc_test.xi").AsProto(), fds[2].AsProto())
	testutil.Eq(t, file.FindSymbol("desc_test.xui").AsProto(), fds[3].AsProto())
	testutil.Eq(t, file.FindSymbol("desc_test.TestMessage.NestedMessage.AnotherNestedMessage.flags").AsProto(), fds[4].AsProto())

	checkFindExtension(t, er, fds)
}

func checkFindExtension(t *testing.T, er *ExtensionRegistry, fds []*desc.FieldDescriptor) {
	for _, fd := range fds {
		testutil.Eq(t, fd, er.FindExtension(fd.GetOwner().GetFullyQualifiedName(), fd.GetNumber()))
		testutil.Eq(t, fd, er.FindExtensionByName(fd.GetOwner().GetFullyQualifiedName(), fd.GetFullyQualifiedName()))
	}
}

type fields []*desc.FieldDescriptor

func (f fields) Len() int {
	return len(f)
}

func (f fields) Less(i, j int) bool {
	return f[i].GetNumber() < f[j].GetNumber()
}

func (f fields) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
