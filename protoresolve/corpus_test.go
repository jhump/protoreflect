package protoresolve_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

func newRegistry(t *testing.T, files []protoreflect.FileDescriptor) *protoresolve.Registry {
	t.Helper()
	reg := &protoresolve.Registry{}
	for _, file := range files {
		require.NoError(t, reg.RegisterFile(file))
	}
	return reg
}

func newFiles(t *testing.T, files []protoreflect.FileDescriptor) *protoregistry.Files {
	t.Helper()
	var reg protoregistry.Files
	for _, file := range files {
		require.NoError(t, reg.RegisterFile(file))
	}
	return &reg
}

// splitCorpus splits the given corpus into three parts, for testing resolvers
// that combine other resolvers. Some files appear in more than one part, to
// verify that combined resolvers de-duplicate elements.
func splitCorpus(t *testing.T, corpus []protoreflect.FileDescriptor) [][]protoreflect.FileDescriptor {
	t.Helper()
	parts := [][]string{
		{"desc_test1.proto", "pkg/desc_test_pkg.proto", "nopkg/desc_test_nopkg_new.proto", "nopkg/desc_test_nopkg.proto", "desc_test2.proto"},
		{"google/protobuf/descriptor.proto", "desc_test_complex.proto", "desc_test1.proto"},
		{"desc_test_proto3.proto", "desc_test_editions.proto", "pkg/desc_test_pkg.proto"},
	}
	byPath := make(map[string]protoreflect.FileDescriptor, len(corpus))
	for _, file := range corpus {
		byPath[file.Path()] = file
	}
	result := make([][]protoreflect.FileDescriptor, len(parts))
	covered := map[string]struct{}{}
	for i, paths := range parts {
		for _, path := range paths {
			file, ok := byPath[path]
			require.True(t, ok, "corpus has no file %q", path)
			result[i] = append(result[i], file)
			covered[path] = struct{}{}
		}
	}
	require.Len(t, covered, len(corpus), "split should cover entire corpus")
	return result
}

// newTypes returns a type registry with dynamic types for all files in the
// given pool.
func newTypes(t *testing.T, files protoresolve.FilePool) *protoregistry.Types {
	t.Helper()
	var types protoregistry.Types
	require.NoError(t, protoresolve.RegisterTypesInFilesRecursive(files, &types, protoresolve.TypeKindsAll))
	return &types
}

type resolverWithTypePool = interface {
	protoresolve.Resolver
	AsTypePool() protoresolve.TypePool
}

// combinedInputs are inputs for Combine and CombinePools: the corpus split
// across several resolvers.
type combinedInputs struct {
	// Registries, which also have an AsTypePool method.
	resolvers []protoresolve.Resolver
	// The same registries, as a different static type for CombinePools.
	pools []resolverWithTypePool
	// The same registries, wrapped with ResolverFromPool, so they have no
	// AsTypePool method.
	nonPools []protoresolve.Resolver
}

// newCombinedInputs returns the given corpus split across several resolvers.
// The first resolver is empty, to verify that combined resolvers consult the
// others even when the first has nothing to offer.
func newCombinedInputs(t *testing.T, corpus []protoreflect.FileDescriptor) combinedInputs {
	t.Helper()
	regs := []*protoresolve.Registry{{}}
	for _, part := range splitCorpus(t, corpus) {
		regs = append(regs, newRegistry(t, part))
	}
	var inputs combinedInputs
	for _, reg := range regs {
		inputs.resolvers = append(inputs.resolvers, reg)
		inputs.pools = append(inputs.pools, reg)
		inputs.nonPools = append(inputs.nonPools, protoresolve.ResolverFromPool(reg))
	}
	return inputs
}
