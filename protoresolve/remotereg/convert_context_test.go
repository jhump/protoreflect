package remotereg

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"
)

func TestFetchSyntheticFilesWithCycles(t *testing.T) {
	t.Parallel()

	// None of these types indicate their file. Types that refer to one another
	// must be put into the same file, but other types are kept separate.
	registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
		// mutual recursion
		"foo.A": newTestType("foo.A", "", "foo.B"),
		"foo.B": newTestType("foo.B", "", "foo.A", "foo.C"),
		// a longer cycle, entered from a type that is not part of it
		"foo.C": newTestType("foo.C", "", "foo.D"),
		"foo.D": newTestType("foo.D", "", "foo.E"),
		"foo.E": newTestType("foo.E", "", "foo.C", "foo.Enum"),
		// not part of any cycle
		"foo.Enum": &typepb.Enum{
			Name:      "foo.Enum",
			Enumvalue: []*typepb.EnumValue{{Name: "ZERO"}},
		},
	})}

	msgA, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
	require.NoError(t, err)
	fileA := msgA.ParentFile()
	require.Equal(t, "foo", string(fileA.Package()))
	require.Equal(t, 2, fileA.Messages().Len())
	require.NotNil(t, fileA.Messages().ByName("A"))
	require.NotNil(t, fileA.Messages().ByName("B"))
	require.Equal(t, 1, fileA.Imports().Len())

	fileC := fileA.Imports().Get(0).FileDescriptor
	require.Equal(t, "foo", string(fileC.Package()))
	require.Equal(t, 3, fileC.Messages().Len())
	for _, name := range []string{"C", "D", "E"} {
		require.NotNil(t, fileC.Messages().ByName(protoreflect.Name(name)))
	}
	require.Equal(t, 1, fileC.Imports().Len())

	fileEnum := fileC.Imports().Get(0).FileDescriptor
	require.Equal(t, 0, fileEnum.Messages().Len())
	require.Equal(t, 1, fileEnum.Enums().Len())
	require.Equal(t, "foo.Enum", string(fileEnum.Enums().Get(0).FullName()))
}

func TestFetchNotFound(t *testing.T) {
	t.Parallel()

	registry := &Registry{
		TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A": newTestType("foo.A", "a.proto", "foo.B"),
			"foo.B": newTestType("foo.B", "b.proto", "foo.Missing"),
		}),
		Fallback: &protoregistry.Files{},
	}
	t.Run("requested type", func(t *testing.T) {
		t.Parallel()
		// Resolvers must return exactly ErrNotFound for an unknown type, not
		// an error that wraps it.
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.Missing")
		require.Equal(t, protoregistry.NotFound, err)
		_, err = registry.AsTypeResolver().FindMessageByURL(testURLPrefix + "foo.Missing")
		require.Equal(t, protoregistry.NotFound, err)
	})
	t.Run("dependency", func(t *testing.T) {
		t.Parallel()
		// foo.A exists, but a type it refers to does not. That must not be
		// reported as foo.A being unknown.
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.ErrorContains(t, err, "could not resolve type "+testURLPrefix+"foo.Missing for field foo.B.f1")
		require.NotErrorIs(t, err, protoregistry.NotFound)
	})
}

func TestFetchCachedTypeWithNestedTypeDependencies(t *testing.T) {
	t.Parallel()

	registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
		"foo.X":   newTestType("foo.X", "x.proto", "foo.A", "foo.A.B"),
		"foo.A":   newTestType("foo.A", "a.proto"),
		"foo.A.B": newTestType("foo.A.B", "a.proto", "foo.Z"),
		"foo.Z":   newTestType("foo.Z", "z.proto"),
		"foo.D":   newTestType("foo.D", "d.proto", "foo.A"),
	})}
	// This caches foo.A, with foo.A.B nested inside it.
	_, err := registry.FindMessageByURL(testURLPrefix + "foo.X")
	require.NoError(t, err)
	// foo.D refers only to foo.A. But the cached foo.A includes foo.A.B, so
	// the type that foo.A.B refers to must be included, too.
	msgD, err := registry.FindMessageByURL(testURLPrefix + "foo.D")
	require.NoError(t, err)
	msgB := msgD.Fields().ByName("f1").Message().Messages().ByName("B")
	require.NotNil(t, msgB)
	require.Equal(t, "foo.Z", string(msgB.Fields().ByName("f1").Message().FullName()))
}

func TestFetchSyntheticCycleMixedSyntax(t *testing.T) {
	t.Parallel()

	typeA := newTestType("foo.A", "", "foo.B")
	typeB := newTestType("foo.B", "", "foo.A")
	typeB.Syntax = typepb.Syntax_SYNTAX_PROTO3
	registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
		"foo.A": typeA,
		"foo.B": typeB,
	})}
	_, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
	require.ErrorContains(t, err, "types with cyclic references have mixed syntax")
}

func TestFetchSyntheticNestedTypes(t *testing.T) {
	t.Parallel()

	typeA := newTestType("foo.A", "", "foo.A.B", "foo.C")
	typeAB := newTestType("foo.A.B", "", "foo.A.B.Enum")
	typeABEnum := &typepb.Enum{Name: "foo.A.B.Enum", Enumvalue: []*typepb.EnumValue{{Name: "ZERO"}}}
	typeC := newTestType("foo.C", "")
	// requireNested checks that the named message is nested in the given
	// message and is in the same file.
	requireNested := func(t *testing.T, parent protoreflect.MessageDescriptor, name protoreflect.Name) protoreflect.MessageDescriptor {
		t.Helper()
		nested := parent.Messages().ByName(name)
		require.NotNil(t, nested, "%s should have nested message %s", parent.FullName(), name)
		require.Equal(t, parent.ParentFile().Path(), nested.ParentFile().Path())
		return nested
	}

	t.Run("together", func(t *testing.T) {
		t.Parallel()
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A": typeA, "foo.A.B": typeAB, "foo.A.B.Enum": typeABEnum, "foo.C": typeC,
		})}
		msgA, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.NoError(t, err)
		require.Equal(t, "foo", string(msgA.ParentFile().Package()))
		msgB := requireNested(t, msgA, "B")
		require.NotNil(t, msgB.Enums().ByName("Enum"))
		// C is not nested, so is in a separate file
		require.Equal(t, 1, msgA.ParentFile().Imports().Len())
		require.Equal(t, "foo.C", string(msgA.Fields().ByName("f2").Message().FullName()))
	})
	t.Run("merge creates cycle", func(t *testing.T) {
		t.Parallel()
		// Once A.B is moved into A's file, A's file and C's file would
		// import each other, so they must be merged, too.
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A":   newTestType("foo.A", "", "foo.C"),
			"foo.A.B": newTestType("foo.A.B", ""),
			"foo.C":   newTestType("foo.C", "", "foo.A.B"),
		})}
		msgA, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.NoError(t, err)
		requireNested(t, msgA, "B")
		require.NotNil(t, msgA.ParentFile().Messages().ByName("C"))
		require.Equal(t, 0, msgA.ParentFile().Imports().Len())
	})
	t.Run("nested type cached first", func(t *testing.T) {
		t.Parallel()
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A": typeA, "foo.A.B": typeAB, "foo.A.B.Enum": typeABEnum, "foo.C": typeC,
		})}
		// Resolved alone, A.B's enclosing type is unknown, so it is assumed
		// to be a package.
		msgB, err := registry.FindMessageByURL(testURLPrefix + "foo.A.B")
		require.NoError(t, err)
		require.Equal(t, "foo.A", string(msgB.ParentFile().Package()))
		// Now A must be combined with the cached A.B.
		msgA, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.NoError(t, err)
		requireNested(t, msgA, "B")
	})
	t.Run("enclosing type cached first", func(t *testing.T) {
		t.Parallel()
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A":   newTestType("foo.A", ""),
			"foo.A.B": newTestType("foo.A.B", ""),
			"foo.D":   newTestType("foo.D", "", "foo.A", "foo.A.B"),
		})}
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.NoError(t, err)
		// Now A.B must be added to the cached A.
		msgD, err := registry.FindMessageByURL(testURLPrefix + "foo.D")
		require.NoError(t, err)
		msgA := msgD.Fields().ByName("f1").Message()
		msgB := requireNested(t, msgA, "B")
		require.Equal(t, msgB, msgD.Fields().ByName("f2").Message())
	})
	t.Run("enclosing type in known file", func(t *testing.T) {
		t.Parallel()
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A":   newTestType("foo.A", "a.proto", "foo.A.B"),
			"foo.A.B": newTestType("foo.A.B", ""),
		})}
		msgA, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.NoError(t, err)
		require.Equal(t, "a.proto", msgA.ParentFile().Path())
		requireNested(t, msgA, "B")
	})
	t.Run("nested type in known file", func(t *testing.T) {
		t.Parallel()
		// The enclosing type does not indicate its file, but the nested type
		// does. So the enclosing type must be in that file, too.
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A":   newTestType("foo.A", "", "foo.A.B"),
			"foo.A.B": newTestType("foo.A.B", "b.proto"),
		})}
		msgA, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.NoError(t, err)
		require.Equal(t, "b.proto", msgA.ParentFile().Path())
		require.Equal(t, "foo", string(msgA.ParentFile().Package()))
		requireNested(t, msgA, "B")
	})
	t.Run("nested types in multiple known files", func(t *testing.T) {
		t.Parallel()
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A":   newTestType("foo.A", "", "foo.A.B", "foo.A.C"),
			"foo.A.B": newTestType("foo.A.B", "b.proto"),
			"foo.A.C": newTestType("foo.A.C", "c.proto"),
		})}
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.ErrorContains(t, err, "nested types and their enclosing types are defined in multiple files: b.proto, c.proto")
	})
	t.Run("enclosing type never fetched", func(t *testing.T) {
		t.Parallel()
		// foo.A is never fetched, but a.proto has other types, so it gets a
		// placeholder for foo.A. That shows foo.A is a message, so foo.A.Y
		// must be in a.proto, too.
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.B":   newTestType("foo.B", "a.proto", "foo.A.X", "foo.A.Y"),
			"foo.A.X": newTestType("foo.A.X", "a.proto"),
			"foo.A.Y": newTestType("foo.A.Y", ""),
		})}
		msgB, err := registry.FindMessageByURL(testURLPrefix + "foo.B")
		require.NoError(t, err)
		msgX := msgB.Fields().ByName("f1").Message()
		msgY := msgB.Fields().ByName("f2").Message()
		require.Equal(t, "a.proto", msgY.ParentFile().Path())
		require.Equal(t, "foo.A", string(msgY.Parent().FullName()))
		require.Equal(t, msgX.Parent(), msgY.Parent())
	})
	t.Run("cycle creates placeholder", func(t *testing.T) {
		t.Parallel()
		// Merging the cycle between foo.C and foo.A.X results in a file with
		// a placeholder for foo.A. So foo.A.Y must then be merged into it.
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.C":   newTestType("foo.C", "", "foo.A.X", "foo.A.Y"),
			"foo.A.X": newTestType("foo.A.X", "", "foo.C"),
			"foo.A.Y": newTestType("foo.A.Y", ""),
		})}
		msgC, err := registry.FindMessageByURL(testURLPrefix + "foo.C")
		require.NoError(t, err)
		msgY := msgC.Fields().ByName("f2").Message()
		require.Equal(t, msgC.ParentFile().Path(), msgY.ParentFile().Path())
		require.Equal(t, "foo.A", string(msgY.Parent().FullName()))
	})
	t.Run("new type nested in cached nested type", func(t *testing.T) {
		t.Parallel()
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.X":     newTestType("foo.X", "", "foo.A", "foo.A.B"),
			"foo.A":     newTestType("foo.A", ""),
			"foo.A.B":   newTestType("foo.A.B", "", "foo.Z"),
			"foo.Z":     newTestType("foo.Z", ""),
			"foo.D":     newTestType("foo.D", "", "foo.A", "foo.A.B.C"),
			"foo.A.B.C": newTestType("foo.A.B.C", ""),
		})}
		// This caches foo.A, with foo.A.B nested inside it.
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.X")
		require.NoError(t, err)
		// Nothing in foo.D refers to foo.A.B. So when the new foo.A.B.C is
		// added to the cached foo.A, the existing foo.A.B must be preserved.
		msgD, err := registry.FindMessageByURL(testURLPrefix + "foo.D")
		require.NoError(t, err)
		msgC := msgD.Fields().ByName("f2").Message()
		msgB, ok := msgC.Parent().(protoreflect.MessageDescriptor)
		require.True(t, ok)
		require.Equal(t, "foo.A.B", string(msgB.FullName()))
		require.Equal(t, 1, msgB.Fields().Len())
	})
	t.Run("cached copies in different files", func(t *testing.T) {
		t.Parallel()
		// The order in which foo.D's fields are processed determines which
		// copy of foo.A.B is used below, and only one order triggered a bug.
		// Fields are processed concurrently, so we try both field orders and
		// repeat each several times.
		for _, fieldTypes := range [][]string{{"foo.A", "foo.A.B"}, {"foo.A.B", "foo.A"}} {
			types := map[string]proto.Message{
				"foo.A":   newTestType("foo.A", "a.proto", "foo.A.B"),
				"foo.A.B": newTestType("foo.A.B", ""),
				"foo.D":   newTestType("foo.D", "", fieldTypes...),
			}
			for range 10 {
				registry := &Registry{TypeFetcher: newTestFetcher(types)}
				// This caches foo.A.B in its own synthetic file.
				_, err := registry.FindMessageByURL(testURLPrefix + "foo.A.B")
				require.NoError(t, err)
				// This caches foo.A, which refers to a copy of foo.A.B that
				// has been moved into a.proto.
				_, err = registry.FindMessageByURL(testURLPrefix + "foo.A")
				require.NoError(t, err)
				// Now both copies of foo.A.B are reachable, and the resulting
				// files must only refer to the copy that gets used.
				msgD, err := registry.FindMessageByURL(testURLPrefix + "foo.D")
				require.NoError(t, err)
				msgB := msgD.Fields().ByName(protoreflect.Name(fmt.Sprintf("f%d", slices.Index(fieldTypes, "foo.A.B")+1))).Message()
				require.Equal(t, "foo.A", string(msgB.Parent().FullName()))
			}
		}
	})
	t.Run("mixed syntax", func(t *testing.T) {
		t.Parallel()
		typeB := newTestType("foo.A.B", "")
		typeB.Syntax = typepb.Syntax_SYNTAX_PROTO3
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A":   newTestType("foo.A", "", "foo.A.B"),
			"foo.A.B": typeB,
		})}
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.ErrorContains(t, err, "nested types and their enclosing types have mixed syntax")
	})
}

func TestFetchCyclicFiles(t *testing.T) {
	t.Parallel()

	t.Run("known files", func(t *testing.T) {
		t.Parallel()
		// Types in a real file cannot refer to types in another file that
		// refers back to the first. So the server is providing bogus data.
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A": newTestType("foo.A", "a.proto", "foo.B"),
			"foo.B": newTestType("foo.B", "b.proto", "foo.C"),
			"foo.C": newTestType("foo.C", "a.proto"),
		})}
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.ErrorContains(t, err, "file a.proto has cyclic import path: a.proto -> b.proto -> a.proto")
	})
	t.Run("known and synthetic files", func(t *testing.T) {
		t.Parallel()
		// B does not indicate its file. But since it is in a cycle with A,
		// it must be in A's file.
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A": newTestType("foo.A", "a.proto", "foo.B"),
			"foo.B": newTestType("foo.B", "", "foo.A"),
		})}
		msgA, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.NoError(t, err)
		require.Equal(t, "a.proto", msgA.ParentFile().Path())
		msgB := msgA.Fields().ByName("f1").Message()
		require.Equal(t, "a.proto", msgB.ParentFile().Path())
		require.Equal(t, 0, msgA.ParentFile().Imports().Len())
	})
	t.Run("multiple known and synthetic files", func(t *testing.T) {
		t.Parallel()
		// A cycle that includes multiple known files cannot be reconciled.
		registry := &Registry{TypeFetcher: newTestFetcher(map[string]proto.Message{
			"foo.A": newTestType("foo.A", "a.proto", "foo.B"),
			"foo.B": newTestType("foo.B", "", "foo.C"),
			"foo.C": newTestType("foo.C", "c.proto", "foo.A"),
		})}
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.A")
		require.ErrorContains(t, err, "has cyclic import path")
	})
}

func TestToFileDescriptorsCyclicImports(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		imports        map[string][]string
		expectedSuffix string
	}{
		{
			name:           "self",
			imports:        map[string][]string{"self.proto": {"self.proto"}},
			expectedSuffix: "cyclic import path: self.proto -> self.proto",
		},
		{
			name: "mutual",
			imports: map[string][]string{
				"mutual1.proto": {"mutual2.proto"},
				"mutual2.proto": {"mutual1.proto"},
			},
			expectedSuffix: "cyclic import path: mutual1.proto -> mutual2.proto -> mutual1.proto",
		},
		{
			// the cycle does not include the first file, and the reported
			// path starts at the first file in the cycle
			name: "ring",
			imports: map[string][]string{
				"entry.proto": {"ring1.proto"},
				"ring1.proto": {"ring2.proto"},
				"ring2.proto": {"ring3.proto"},
				"ring3.proto": {"ring1.proto"},
			},
			expectedSuffix: "cyclic import path: ring1.proto -> ring2.proto -> ring3.proto -> ring1.proto",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			files := map[string]*fileEntry{}
			for fileName, deps := range testCase.imports {
				entry := &fileEntry{deps: map[string]struct{}{}}
				for _, dep := range deps {
					entry.deps[dep] = struct{}{}
				}
				msgName := strings.TrimSuffix(fileName, ".proto")
				entry.types.addType("foo."+msgName, &descriptorpb.DescriptorProto{Name: proto.String(msgName)})
				files[fileName] = entry
			}
			_, err := toFileDescriptors(files, nil, (*typeTrie).rewriteDescriptor)
			require.Error(t, err)
			require.True(t, strings.HasSuffix(err.Error(), testCase.expectedSuffix),
				"expected error to end with %q; got: %v", testCase.expectedSuffix, err)
		})
	}
}

func TestFetchReachableTypesLimit(t *testing.T) {
	t.Parallel()

	t.Run("endless", func(t *testing.T) {
		t.Parallel()
		registry := &Registry{TypeFetcher: newChainFetcher(-1)}
		_, err := registry.FindMessageByURL(testURLPrefix + "foo.T0")
		require.ErrorContains(t, err, fmt.Sprintf("too many types reachable from requested type(s): limit is %d", defaultMaxReachableTypes))
	})
	testCases := []struct {
		name       string
		configured int
		limit      int
	}{
		{name: "default", configured: 0, limit: defaultMaxReachableTypes},
		{name: "negative uses default", configured: -1, limit: defaultMaxReachableTypes},
		{name: "lower", configured: 10, limit: 10},
		{name: "higher", configured: 3000, limit: 3000},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			t.Run("at limit", func(t *testing.T) {
				t.Parallel()
				// This also results in an import graph that is very deep.
				registry := &Registry{
					TypeFetcher:       newChainFetcher(testCase.limit - 1),
					MaxReachableTypes: testCase.configured,
				}
				_, err := registry.FindMessageByURL(testURLPrefix + "foo.T0")
				require.NoError(t, err)
			})
			t.Run("over limit", func(t *testing.T) {
				t.Parallel()
				registry := &Registry{
					TypeFetcher:       newChainFetcher(testCase.limit),
					MaxReachableTypes: testCase.configured,
				}
				_, err := registry.FindMessageByURL(testURLPrefix + "foo.T0")
				require.ErrorContains(t, err, fmt.Sprintf("too many types reachable from requested type(s): limit is %d", testCase.limit))
			})
		})
	}
	t.Run("over limit with cached types", func(t *testing.T) {
		t.Parallel()
		// Types already in the registry count towards the limit.
		half := defaultMaxReachableTypes / 2
		registry := &Registry{TypeFetcher: newChainFetcher(defaultMaxReachableTypes)}
		_, err := registry.FindMessageByURL(fmt.Sprintf("%sfoo.T%d", testURLPrefix, half))
		require.NoError(t, err)
		_, err = registry.FindMessageByURL(testURLPrefix + "foo.T0")
		require.ErrorContains(t, err, "too many types reachable")
	})
}

func TestFetchValidatesNames(t *testing.T) {
	t.Parallel()

	// This is a payload that was found via fuzzing. It has no name, and its
	// fields refer to the URLs "0000" and "".
	var fuzzType typepb.Type
	err := proto.Unmarshal([]byte("\x12\f\b\v80002\x040000\x12\f\b\v0000000000"), &fuzzType)
	require.NoError(t, err)

	testCases := []struct {
		name          string
		url           string
		enum          bool
		typ           proto.Message
		expectedError string
	}{
		{
			name:          "fuzz payload",
			url:           testURLPrefix + "foo.Bar",
			typ:           &fuzzType,
			expectedError: `has wrong name: wanted "foo.Bar", got ""`,
		},
		{
			name:          "wrong message name",
			url:           testURLPrefix + "foo.Bar",
			typ:           newTestType("foo.Baz", ""),
			expectedError: `has wrong name: wanted "foo.Bar", got "foo.Baz"`,
		},
		{
			name:          "wrong enum name",
			url:           testURLPrefix + "foo.Bar",
			enum:          true,
			typ:           &typepb.Enum{Name: "foo.Baz"},
			expectedError: `has wrong name: wanted "foo.Bar", got "foo.Baz"`,
		},
		{
			name:          "invalid message name",
			url:           testURLPrefix + "foo..Bar",
			typ:           newTestType("foo..Bar", ""),
			expectedError: `has invalid name "foo..Bar"`,
		},
		{
			name:          "invalid enum name",
			url:           testURLPrefix + "foo.Bar-Baz",
			enum:          true,
			typ:           &typepb.Enum{Name: "foo.Bar-Baz"},
			expectedError: `has invalid name "foo.Bar-Baz"`,
		},
		{
			name:          "empty name",
			url:           testURLPrefix,
			typ:           newTestType("", ""),
			expectedError: `has invalid name ""`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			registry := &Registry{TypeFetcher: TypeFetcherFunc(func(context.Context, string, bool) (proto.Message, error) {
				return testCase.typ, nil
			})}
			var err error
			if testCase.enum {
				_, err = registry.FindEnumByURL(testCase.url)
			} else {
				_, err = registry.FindMessageByURL(testCase.url)
			}
			require.ErrorContains(t, err, testCase.expectedError)
		})
	}
}

func TestToMessageDescriptorValidatesName(t *testing.T) {
	t.Parallel()
	var registry Registry
	_, err := registry.AsDescriptorConverter().ToMessageDescriptor(context.Background(), newTestType("foo..Bar", ""))
	require.ErrorContains(t, err, `has invalid name "foo..Bar"`)
	_, err = registry.AsDescriptorConverter().ToEnumDescriptor(context.Background(), &typepb.Enum{Name: ""})
	require.ErrorContains(t, err, `has invalid name ""`)
}

const testURLPrefix = "https://type.googleapis.com/"

// newTestType returns a message type with the given name and a message field
// for each of the given type names. If fileName is not empty, the type has a
// source context that indicates it is defined in that file.
func newTestType(name, fileName string, fieldTypes ...string) *typepb.Type {
	typ := &typepb.Type{Name: name}
	if fileName != "" {
		typ.SourceContext = &sourcecontextpb.SourceContext{FileName: fileName}
	}
	for i, fieldType := range fieldTypes {
		kind := typepb.Field_TYPE_MESSAGE
		if strings.HasSuffix(fieldType, "Enum") {
			kind = typepb.Field_TYPE_ENUM
		}
		typ.Fields = append(typ.Fields, &typepb.Field{
			Name:        fmt.Sprintf("f%d", i+1),
			Number:      int32(i + 1),
			Kind:        kind,
			Cardinality: typepb.Field_CARDINALITY_OPTIONAL,
			TypeUrl:     testURLPrefix + fieldType,
		})
	}
	return typ
}

// newTestFetcher returns a fetcher that serves the given types, keyed by name.
func newTestFetcher(types map[string]proto.Message) TypeFetcher {
	return TypeFetcherFunc(func(_ context.Context, url string, _ bool) (proto.Message, error) {
		typ, ok := types[strings.TrimPrefix(url, testURLPrefix)]
		if !ok {
			return nil, protoregistry.NotFound
		}
		return typ, nil
	})
}

// newChainFetcher returns a fetcher that serves a chain of types: "foo.T0"
// refers to "foo.T1", which refers to "foo.T2", and so on. The type "foo.T<end>"
// refers to nothing. If end is negative, the chain never ends.
func newChainFetcher(end int) TypeFetcher {
	return TypeFetcherFunc(func(_ context.Context, url string, _ bool) (proto.Message, error) {
		name := strings.TrimPrefix(url, testURLPrefix)
		num, err := strconv.Atoi(strings.TrimPrefix(name, "foo.T"))
		if err != nil {
			return nil, protoregistry.NotFound
		}
		if num == end {
			return newTestType(name, ""), nil
		}
		return newTestType(name, "", fmt.Sprintf("foo.T%d", num+1)), nil
	})
}
