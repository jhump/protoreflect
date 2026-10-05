package protoresolve_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestFindExtensionByNumber(t *testing.T) {
	var files protoregistry.Files
	err := files.RegisterFile(testprotos.File_desc_test1_proto)
	require.NoError(t, err)
	err = files.RegisterFile(testprotos.File_desc_test2_proto)
	require.NoError(t, err)
	err = files.RegisterFile(testprotos.File_desc_test_complex_proto)
	require.NoError(t, err)

	extd := protoresolve.FindExtensionByNumber(&files, "testprotos.AnotherTestMessage", 100)
	require.NotNil(t, extd)
	assert.Equal(t, protoreflect.FullName("testprotos.xtm"), extd.FullName())
	assert.Equal(t, protoreflect.MessageKind, extd.Kind())
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage"), extd.Message().FullName())
	assert.Equal(t, "desc_test1.proto", extd.ParentFile().Path())

	extd = protoresolve.FindExtensionByNumber(&files, "testprotos.AnotherTestMessage", 102)
	require.NotNil(t, extd)
	assert.Equal(t, protoreflect.FullName("testprotos.xi"), extd.FullName())
	assert.Equal(t, protoreflect.Int32Kind, extd.Kind())
	assert.Equal(t, "desc_test1.proto", extd.ParentFile().Path())

	extd = protoresolve.FindExtensionByNumber(&files, "testprotos.AnotherTestMessage", 999)
	require.Nil(t, extd)

	extd = protoresolve.FindExtensionByNumber(&files, "google.protobuf.ExtensionRangeOptions", 20000)
	require.NotNil(t, extd)
	assert.Equal(t, protoreflect.FullName("foo.bar.label"), extd.FullName())
	assert.Equal(t, protoreflect.StringKind, extd.Kind())
	assert.Equal(t, "desc_test_complex.proto", extd.ParentFile().Path())
}

func TestFindExtensionByNumberInFile(t *testing.T) {
	extd := protoresolve.FindExtensionByNumberInFile(testprotos.File_desc_test1_proto, "testprotos.AnotherTestMessage", 100)
	require.NotNil(t, extd)
	assert.Equal(t, protoreflect.FullName("testprotos.xtm"), extd.FullName())
	assert.Equal(t, protoreflect.MessageKind, extd.Kind())
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage"), extd.Message().FullName())

	extd = protoresolve.FindExtensionByNumberInFile(testprotos.File_desc_test1_proto, "testprotos.AnotherTestMessage", 102)
	require.NotNil(t, extd)
	assert.Equal(t, protoreflect.FullName("testprotos.xi"), extd.FullName())
	assert.Equal(t, protoreflect.Int32Kind, extd.Kind())

	extd = protoresolve.FindExtensionByNumberInFile(testprotos.File_desc_test1_proto, "testprotos.AnotherTestMessage", 999)
	require.Nil(t, extd)

	extd = protoresolve.FindExtensionByNumberInFile(testprotos.File_desc_test1_proto, "google.protobuf.ExtensionRangeOptions", 20000)
	require.Nil(t, extd)
}

func TestFindDescriptorByNameInFile(t *testing.T) {
	d := protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.TestMessage")
	require.NotNil(t, d)
	md, ok := d.(protoreflect.MessageDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage"), md.FullName())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.TestMessage.ne")
	require.NotNil(t, d)
	fld, ok := d.(protoreflect.FieldDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage.ne"), fld.FullName())
	assert.Equal(t, protoreflect.FieldNumber(4), fld.Number())
	assert.False(t, fld.IsExtension())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.AnotherTestMessage.atmoo")
	require.NotNil(t, d)
	ood, ok := d.(protoreflect.OneofDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.AnotherTestMessage.atmoo"), ood.FullName())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.SomeEnum")
	require.NotNil(t, d)
	ed, ok := d.(protoreflect.EnumDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.SomeEnum"), ed.FullName())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.SOME_VAL")
	require.NotNil(t, d)
	evd, ok := d.(protoreflect.EnumValueDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.SOME_VAL"), evd.FullName())
	assert.Equal(t, protoreflect.FullName("testprotos.SomeEnum"), evd.Parent().FullName())
	assert.Equal(t, protoreflect.EnumNumber(0), evd.Number())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.xtm")
	require.NotNil(t, d)
	extd, ok := d.(protoreflect.ExtensionDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.xtm"), extd.FullName())
	assert.Equal(t, protoreflect.FieldNumber(100), extd.Number())
	assert.True(t, extd.IsExtension())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.SomeService")
	require.NotNil(t, d)
	sd, ok := d.(protoreflect.ServiceDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.SomeService"), sd.FullName())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.SomeService.SomeMethod")
	require.NotNil(t, d)
	mtd, ok := d.(protoreflect.MethodDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.SomeService.SomeMethod"), mtd.FullName())

	// Nested elements

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.TestMessage.NestedMessage.AnotherNestedMessage")
	require.NotNil(t, d)
	md, ok = d.(protoreflect.MessageDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage.NestedMessage.AnotherNestedMessage"), md.FullName())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.TestMessage.NestedMessage.yanm")
	require.NotNil(t, d)
	fld, ok = d.(protoreflect.FieldDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage.NestedMessage.yanm"), fld.FullName())
	assert.Equal(t, protoreflect.FieldNumber(2), fld.Number())
	assert.False(t, fld.IsExtension())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum")
	require.NotNil(t, d)
	ed, ok = d.(protoreflect.EnumDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum"), ed.FullName())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.VALUE1")
	require.NotNil(t, d)
	evd, ok = d.(protoreflect.EnumValueDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.VALUE1"), evd.FullName())
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum"), evd.Parent().FullName())
	assert.Equal(t, protoreflect.EnumNumber(1), evd.Number())

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.flags")
	require.NotNil(t, d)
	extd, ok = d.(protoreflect.ExtensionDescriptor)
	assert.True(t, ok)
	assert.Equal(t, protoreflect.FullName("testprotos.TestMessage.NestedMessage.AnotherNestedMessage.flags"), extd.FullName())
	assert.True(t, extd.IsExtension())

	// Not found

	d = protoresolve.FindDescriptorByNameInFile(testprotos.File_desc_test1_proto, "foo.bar")
	require.Nil(t, d)
}

func TestRangeExtensionsByMessage(t *testing.T) {
	var files protoregistry.Files
	err := files.RegisterFile(testprotos.File_desc_test1_proto)
	require.NoError(t, err)
	err = files.RegisterFile(testprotos.File_desc_test2_proto)
	require.NoError(t, err)
	err = files.RegisterFile(testprotos.File_desc_test_complex_proto)
	require.NoError(t, err)

	var exts []protoreflect.ExtensionDescriptor
	// stops when func returns false
	protoresolve.RangeExtensionsByMessage(&files, "testprotos.AnotherTestMessage", func(extd protoreflect.ExtensionDescriptor) bool {
		exts = append(exts, extd)
		return false
	})
	assert.Equal(t, 1, len(exts))

	exts = nil
	protoresolve.RangeExtensionsByMessage(&files, "testprotos.AnotherTestMessage", func(extd protoreflect.ExtensionDescriptor) bool {
		exts = append(exts, extd)
		return true
	})
	assert.Equal(t, 5, len(exts))
	names := make([]string, 5)
	for i, ext := range exts {
		names[i] = string(ext.FullName())
	}
	sort.Strings(names)
	expected := []string{
		"testprotos.TestMessage.NestedMessage.AnotherNestedMessage.flags",
		"testprotos.xi",
		"testprotos.xs",
		"testprotos.xtm",
		"testprotos.xui",
	}
	assert.Equal(t, expected, names)

	exts = nil
	protoresolve.RangeExtensionsByMessage(&files, "google.protobuf.MessageOptions", func(extd protoreflect.ExtensionDescriptor) bool {
		exts = append(exts, extd)
		return true
	})
	assert.Equal(t, 5, len(exts))
	for i, ext := range exts {
		names[i] = string(ext.FullName())
	}
	sort.Strings(names)
	expected = []string{
		"foo.bar.Test.Nested.fooblez",
		"foo.bar.a",
		"foo.bar.eee",
		"foo.bar.map_vals",
		"foo.bar.rept",
	}
	assert.Equal(t, expected, names)

	// Message with no extensions
	exts = nil
	protoresolve.RangeExtensionsByMessage(&files, "testprotos.TestMessage", func(extd protoreflect.ExtensionDescriptor) bool {
		exts = append(exts, extd)
		return true
	})
	assert.Equal(t, 0, len(exts))

	// Unknown message
	exts = nil
	protoresolve.RangeExtensionsByMessage(&files, "foo.bar.baz.Buzz", func(extd protoreflect.ExtensionDescriptor) bool {
		exts = append(exts, extd)
		return true
	})
	assert.Equal(t, 0, len(exts))
}

func TestGlobalDescriptors(t *testing.T) {
	// TODO
	testResolver(t, protoresolve.GlobalDescriptors)
}

func TestResolverFromPool(t *testing.T) {
	var files protoregistry.Files
	require.NoError(t, files.RegisterFile(testprotos.File_desc_test1_proto))
	require.NoError(t, files.RegisterFile(testprotos.File_desc_test2_proto))
	require.NoError(t, files.RegisterFile(testprotos.File_desc_test_complex_proto))

	// Wrap the registry in the ResolverFromPool adapter
	res := protoresolve.ResolverFromPool(&files)

	// Run the standard validation suite (defined in this package)
	testResolver(t, res)
}

func TestResolverFromPools(t *testing.T) {
	var files protoregistry.Files
	require.NoError(t, files.RegisterFile(testprotos.File_desc_test1_proto))
	require.NoError(t, files.RegisterFile(testprotos.File_desc_test2_proto))
	require.NoError(t, files.RegisterFile(testprotos.File_desc_test_complex_proto))

	// Wrap the registry in the ResolverFromPools adapter using GlobalTypes
	res := protoresolve.ResolverFromPools(&files, protoregistry.GlobalTypes)

	// Run the standard validation suite (defined in this package)
	testResolver(t, res)
}
