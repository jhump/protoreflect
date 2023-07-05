package sourcelocation

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
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
