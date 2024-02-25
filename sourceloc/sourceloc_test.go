package sourceloc_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal"
	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	. "github.com/jhump/protoreflect/v2/sourceloc"
)

func TestIsZero(t *testing.T) {
	loc := protoreflect.SourceLocation{}
	require.True(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		Path: []int32{},
	}
	require.False(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		StartLine: 1,
	}
	require.False(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		StartColumn: 1,
	}
	require.False(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		EndLine: 1,
	}
	require.False(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		EndColumn: 1,
	}
	require.False(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		LeadingDetachedComments: []string{},
	}
	require.False(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		LeadingComments: "a",
	}
	require.False(t, IsZero(loc))
	loc = protoreflect.SourceLocation{
		TrailingComments: "a",
	}
	require.False(t, IsZero(loc))
}

func TestPathsEqual(t *testing.T) {
	path1 := protoreflect.SourcePath{1, 2, 3}
	path2 := protoreflect.SourcePath{}
	require.False(t, PathsEqual(path1, path2))
	require.True(t, PathsEqual(path1, path1))
	require.True(t, PathsEqual(path2, path2))
	path2 = protoreflect.SourcePath{1, 2, 4}
	require.False(t, PathsEqual(path1, path2))
	path2 = protoreflect.SourcePath{1, 2}
	require.False(t, PathsEqual(path1, path2))
	path2 = protoreflect.SourcePath{1, 2, 3, 4}
	require.False(t, PathsEqual(path1, path2))
}

func TestIsSubpath(t *testing.T) {
	path1 := protoreflect.SourcePath{1, 2, 3}
	path2 := protoreflect.SourcePath{}
	require.True(t, IsSubpathOf(path1, path2))
	require.False(t, IsSubpathOf(path2, path1))
	// paths are sub-paths of themselves
	require.True(t, IsSubpathOf(path1, path1))
	require.True(t, IsSubpathOf(path2, path2))
	path2 = protoreflect.SourcePath{1, 2, 4}
	require.False(t, IsSubpathOf(path1, path2))
	require.False(t, IsSubpathOf(path2, path1))
	path2 = protoreflect.SourcePath{1, 2}
	require.True(t, IsSubpathOf(path1, path2))
	require.False(t, IsSubpathOf(path2, path1))
	path2 = protoreflect.SourcePath{1, 2, 3, 4}
	require.False(t, IsSubpathOf(path1, path2))
	require.True(t, IsSubpathOf(path2, path1))
}

func TestPathFor(t *testing.T) {
	fd, err := prototesting.LoadProtoset("../internal/testdata/desc_test_complex_source_info.protoset")
	require.NoError(t, err)
	checkPathsForFile(t, fd)
}

func checkPathsForFile(t *testing.T, fd protoreflect.FileDescriptor) {
	path := protoreflect.SourcePath{}
	require.Equal(t, path, PathFor(fd))

	msgs := fd.Messages()
	path = protoreflect.SourcePath{internal.FileMessagesTag}
	for i, length := 0, msgs.Len(); i < length; i++ {
		path := append(path, int32(i))
		checkPathsForMessage(t, path, msgs.Get(i))
	}

	enums := fd.Enums()
	path = protoreflect.SourcePath{internal.FileEnumsTag}
	for i, length := 0, enums.Len(); i < length; i++ {
		path := append(path, int32(i))
		checkPathsForEnum(t, path, enums.Get(i))
	}

	exts := fd.Extensions()
	path = protoreflect.SourcePath{internal.FileExtensionsTag}
	for i, length := 0, exts.Len(); i < length; i++ {
		path := append(path, int32(i))
		require.Equal(t, path, PathFor(exts.Get(i)))
	}

	svcs := fd.Services()
	path = protoreflect.SourcePath{internal.FileServicesTag}
	for i, length := 0, svcs.Len(); i < length; i++ {
		svc := svcs.Get(i)
		path := append(path, int32(i))
		require.Equal(t, path, PathFor(svc))
		path = append(path, internal.ServiceMethodsTag)
		mtds := svc.Methods()
		for j, length := 0, mtds.Len(); j < length; j++ {
			path := append(path, int32(j))
			require.Equal(t, path, PathFor(mtds.Get(j)))
		}
	}
}

func checkPathsForMessage(t *testing.T, msgPath protoreflect.SourcePath, md protoreflect.MessageDescriptor) {
	require.Equal(t, msgPath, PathFor(md))

	flds := md.Fields()
	path := append(msgPath, internal.MessageFieldsTag)
	for i, length := 0, flds.Len(); i < length; i++ {
		path := append(path, int32(i))
		require.Equal(t, path, PathFor(flds.Get(i)))
	}

	oos := md.Oneofs()
	path = append(msgPath, internal.MessageOneofsTag)
	for i, length := 0, oos.Len(); i < length; i++ {
		path := append(path, int32(i))
		require.Equal(t, path, PathFor(oos.Get(i)))
	}

	msgs := md.Messages()
	path = append(msgPath, internal.MessageNestedMessagesTag)
	for i, length := 0, msgs.Len(); i < length; i++ {
		path := append(path, int32(i))
		checkPathsForMessage(t, path, msgs.Get(i))
	}

	enums := md.Enums()
	path = append(msgPath, internal.MessageEnumsTag)
	for i, length := 0, enums.Len(); i < length; i++ {
		path := append(path, int32(i))
		checkPathsForEnum(t, path, enums.Get(i))
	}

	exts := md.Extensions()
	path = append(msgPath, internal.MessageExtensionsTag)
	for i, length := 0, exts.Len(); i < length; i++ {
		path := append(path, int32(i))
		require.Equal(t, path, PathFor(exts.Get(i)))
	}
}

func checkPathsForEnum(t *testing.T, enumPath protoreflect.SourcePath, ed protoreflect.EnumDescriptor) {
	require.Equal(t, enumPath, PathFor(ed))

	vals := ed.Values()
	path := append(enumPath, internal.EnumValuesTag)
	for i, length := 0, vals.Len(); i < length; i++ {
		path := append(path, int32(i))
		require.Equal(t, path, PathFor(vals.Get(i)))
	}
}
