package protoresolve_test

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/internal/testprotos/nopkg"
	"github.com/jhump/protoreflect/v2/internal/testprotos/pkg"
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

func TestTypeNameFromURL(t *testing.T) {
	t.Parallel()
	testCases := map[string]protoreflect.FullName{
		"type.googleapis.com/foo.Bar": "foo.Bar",
		"example.com/a/b/c/foo.Bar":   "foo.Bar",
		"foo.Bar":                     "foo.Bar",
		"example.com/":                "",
		"":                            "",
	}
	for url, name := range testCases {
		assert.Equal(t, name, protoresolve.TypeNameFromURL(url), "URL %q", url)
	}
}

func TestExtensionType(t *testing.T) {
	t.Parallel()
	// Descriptors that know their type return that type.
	assert.Same(t, testprotos.E_Xtm, protoresolve.ExtensionType(testprotos.E_Xtm.TypeDescriptor()))
	// Otherwise, a dynamic type is returned.
	ext := testprotos.File_desc_test1_proto.Extensions().ByName("xtm")
	require.NotNil(t, ext)
	extType := protoresolve.ExtensionType(ext)
	assert.IsType(t, dynamicpb.NewExtensionType(ext), extType)
	assert.Equal(t, ext, extType.TypeDescriptor().Descriptor())
}

func TestTypeKindString(t *testing.T) {
	t.Parallel()
	testCases := map[protoresolve.TypeKind]string{
		0:                                   "<none>",
		protoresolve.TypeKindMessage:        "message",
		protoresolve.TypeKindEnum:           "enum",
		protoresolve.TypeKindExtension:      "extension",
		protoresolve.TypeKindsAll:           "message,enum,extension",
		protoresolve.TypeKindsSerialization: "message,extension",
		protoresolve.TypeKind(8):            "unknown kind (8)",
		protoresolve.TypeKindEnum | 8:       "enum,unknown kind (8)",
	}
	for kind, str := range testCases {
		assert.Equal(t, str, kind.String())
	}
}

func TestRegisterTypesInFile(t *testing.T) {
	t.Parallel()
	file := testprotos.File_desc_test1_proto
	const (
		msgName        = "testprotos.TestMessage.NestedMessage"
		enumName       = "testprotos.SomeEnum"
		nestedEnumName = "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.YetAnotherNestedMessage.DeeplyNestedEnum"
		extName        = "testprotos.xtm"
		nestedExtName  = "testprotos.TestMessage.NestedMessage.AnotherNestedMessage.flags"
	)
	testCases := []struct {
		kinds                       protoresolve.TypeKind
		messages, enums, extensions bool
	}{
		{kinds: protoresolve.TypeKindMessage, messages: true},
		{kinds: protoresolve.TypeKindEnum, enums: true},
		{kinds: protoresolve.TypeKindExtension, extensions: true},
		{kinds: protoresolve.TypeKindsSerialization, messages: true, extensions: true},
		{kinds: protoresolve.TypeKindsAll, messages: true, enums: true, extensions: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.kinds.String(), func(t *testing.T) {
			t.Parallel()
			var types protoregistry.Types
			require.NoError(t, protoresolve.RegisterTypesInFile(file, &types, testCase.kinds))
			// Registering again is a no-op.
			require.NoError(t, protoresolve.RegisterTypesInFile(file, &types, testCase.kinds))
			_, err := types.FindMessageByName(msgName)
			assert.Equal(t, testCase.messages, err == nil, "message %s: %v", msgName, err)
			for _, name := range []protoreflect.FullName{enumName, nestedEnumName} {
				_, err = types.FindEnumByName(name)
				assert.Equal(t, testCase.enums, err == nil, "enum %s: %v", name, err)
			}
			for _, name := range []protoreflect.FullName{extName, nestedExtName} {
				_, err = types.FindExtensionByName(name)
				assert.Equal(t, testCase.extensions, err == nil, "extension %s: %v", name, err)
			}
		})
	}

	t.Run("not recursive", func(t *testing.T) {
		t.Parallel()
		var types protoregistry.Types
		require.NoError(t, protoresolve.RegisterTypesInFile(testprotos.File_desc_test2_proto, &types, protoresolve.TypeKindsAll))
		_, err := types.FindMessageByName(msgName)
		assert.ErrorIs(t, err, protoresolve.ErrNotFound, "types from imports should not be registered")
	})
}

func TestRegisterTypesInFileRecursive(t *testing.T) {
	t.Parallel()
	// desc_test2.proto and all of its transitive imports.
	expected := []protoreflect.FileDescriptor{
		testprotos.File_desc_test1_proto,
		pkg.File_pkg_desc_test_pkg_proto,
		nopkg.File_nopkg_desc_test_nopkg_new_proto,
		nopkg.File_nopkg_desc_test_nopkg_proto,
		testprotos.File_desc_test2_proto,
	}
	opts := []resolvertest.Option{resolvertest.WithLenientErrors()}

	var types protoregistry.Types
	require.NoError(t, protoresolve.RegisterTypesInFileRecursive(testprotos.File_desc_test2_proto, &types, protoresolve.TypeKindsAll))
	resolvertest.CheckTypePool(t, &types, expected, opts...)

	types = protoregistry.Types{}
	files := newFiles(t, []protoreflect.FileDescriptor{testprotos.File_desc_test1_proto, testprotos.File_desc_test2_proto})
	require.NoError(t, protoresolve.RegisterTypesInFilesRecursive(files, &types, protoresolve.TypeKindsAll))
	resolvertest.CheckTypePool(t, &types, expected, opts...)
}

func TestRegisterTypesInFileConflicts(t *testing.T) {
	t.Parallel()
	file := testprotos.File_desc_test1_proto
	deps := newFiles(t, []protoreflect.FileDescriptor{file})
	newFile := func(t *testing.T, fileProto *descriptorpb.FileDescriptorProto) protoreflect.FileDescriptor {
		t.Helper()
		other, err := protodesc.NewFile(fileProto, deps)
		require.NoError(t, err)
		return other
	}
	// Files with elements whose names conflict with those in file.
	otherMessage := fileProto("other.proto", "testprotos")
	otherMessage.MessageType = []*descriptorpb.DescriptorProto{{Name: proto.String("TestMessage")}}
	otherEnum := fileProto("other.proto", "testprotos")
	otherEnum.EnumType = []*descriptorpb.EnumDescriptorProto{{
		Name:  proto.String("SomeEnum"),
		Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String("OTHER"), Number: proto.Int32(0)}},
	}}
	otherExtension := fileProto("other.proto", "testprotos", file.Path())
	otherExtension.Extension = []*descriptorpb.FieldDescriptorProto{int32Extension("xtm", 150, "testprotos.AnotherTestMessage")}
	testCases := []struct {
		name      string
		other     *descriptorpb.FileDescriptorProto
		kinds     protoresolve.TypeKind
		errSubstr string
	}{
		{
			name:      "message",
			other:     otherMessage,
			kinds:     protoresolve.TypeKindMessage,
			errSubstr: `type testprotos.TestMessage is defined in both "other.proto" and "desc_test1.proto"`,
		},
		{
			name:      "enum",
			other:     otherEnum,
			kinds:     protoresolve.TypeKindEnum,
			errSubstr: `type testprotos.SomeEnum is defined in both "other.proto" and "desc_test1.proto"`,
		},
		{
			name:      "extension name",
			other:     otherExtension,
			kinds:     protoresolve.TypeKindExtension,
			errSubstr: `type testprotos.xtm is defined in both "other.proto" and "desc_test1.proto"`,
		},
		{
			name:      "extension number",
			other:     conflictingExtensionFileProto(),
			kinds:     protoresolve.TypeKindExtension,
			errSubstr: `extension number 100 for testprotos.AnotherTestMessage is defined in both "conflict.proto" and "desc_test1.proto"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			other := newFile(t, testCase.other)
			var types protoregistry.Types
			require.NoError(t, protoresolve.RegisterTypesInFile(other, &types, testCase.kinds))
			err := protoresolve.RegisterTypesInFile(file, &types, testCase.kinds)
			assert.ErrorContains(t, err, testCase.errSubstr)
			err = protoresolve.RegisterTypesInFilesRecursive(deps, &types, testCase.kinds)
			assert.ErrorContains(t, err, testCase.errSubstr)
		})
	}
}
