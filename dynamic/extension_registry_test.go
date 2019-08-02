package dynamic

import (
	"sort"
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestExtensionRegistry_AddExtension(t *testing.T) {
	er := &ExtensionRegistry{}
	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	err = er.AddExtension(file.GetExtensions()...)
	testutil.Ok(t, err)

	fds := er.AllExtensionsForType("testprotos.AnotherTestMessage")
	sort.Sort(fields(fds))

	testutil.Eq(t, []desc.Descriptor{
		file.FindSymbol("testprotos.xtm"),
		file.FindSymbol("testprotos.xs"),
		file.FindSymbol("testprotos.xi"),
		file.FindSymbol("testprotos.xui"),
	}, fds)

	checkFindExtension(t, er, fds)
}

func TestExtensionRegistry_AddExtensionDesc(t *testing.T) {
	er := &ExtensionRegistry{}

	err := er.AddExtensionDesc(testprotos.E_Xtm, testprotos.E_Xs, testprotos.E_Xi)
	testutil.Ok(t, err)

	fds := er.AllExtensionsForType("testprotos.AnotherTestMessage")
	sort.Sort(fields(fds))

	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	testutil.Eq(t, 3, len(fds))
	testutil.Eq(t, file.FindSymbol("testprotos.xtm"), fds[0])
	testutil.Eq(t, file.FindSymbol("testprotos.xs"), fds[1])
	testutil.Eq(t, file.FindSymbol("testprotos.xi"), fds[2])

	checkFindExtension(t, er, fds)
}

func TestExtensionRegistry_AddExtensionsFromFile(t *testing.T) {
	er := &ExtensionRegistry{}
	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	er.AddExtensionsFromFile(file)

	fds := er.AllExtensionsForType("testprotos.AnotherTestMessage")
	sort.Sort(fields(fds))

	testutil.Eq(t, 5, len(fds))
	testutil.Eq(t, file.FindSymbol("testprotos.xtm"), fds[0])
	testutil.Eq(t, file.FindSymbol("testprotos.xs"), fds[1])
	testutil.Eq(t, file.FindSymbol("testprotos.xi"), fds[2])
	testutil.Eq(t, file.FindSymbol("testprotos.xui"), fds[3])
	testutil.Eq(t, file.FindSymbol("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.flags"), fds[4])

	checkFindExtension(t, er, fds)
}

func TestExtensionRegistry_Empty(t *testing.T) {
	er := ExtensionRegistry{}
	fds := er.AllExtensionsForType("testprotos.AnotherTestMessage")
	testutil.Eq(t, 0, len(fds))
}

func TestExtensionRegistry_Defaults(t *testing.T) {
	er := NewExtensionRegistryWithDefaults()

	fds := er.AllExtensionsForType("testprotos.AnotherTestMessage")
	sort.Sort(fields(fds))

	file, err := desc.LoadFileDescriptor("desc_test1.proto")
	testutil.Ok(t, err)

	testutil.Eq(t, 5, len(fds))
	testutil.Eq(t, file.FindSymbol("testprotos.xtm").AsProto(), fds[0].AsProto())
	testutil.Eq(t, file.FindSymbol("testprotos.xs").AsProto(), fds[1].AsProto())
	testutil.Eq(t, file.FindSymbol("testprotos.xi").AsProto(), fds[2].AsProto())
	testutil.Eq(t, file.FindSymbol("testprotos.xui").AsProto(), fds[3].AsProto())
	testutil.Eq(t, file.FindSymbol("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.flags").AsProto(), fds[4].AsProto())

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
