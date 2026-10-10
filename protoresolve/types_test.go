package protoresolve_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestTypePools(t *testing.T) {
	t.Parallel()
	corpus := resolvertest.Corpus()
	files := newFiles(t, corpus)
	types := newTypes(t, files)
	inputs := newCombinedInputs(t, corpus)

	testCases := []struct {
		name string
		pool protoresolve.TypeResolver
		opts []resolvertest.Option
	}{
		{
			// Sanity check of the test suite against the reference implementation.
			name: "protoregistry.GlobalTypes",
			pool: protoregistry.GlobalTypes,
			opts: []resolvertest.Option{resolvertest.WithExtraFiles(), resolvertest.WithLenientErrors()},
		},
		{
			name: "GlobalDescriptors",
			pool: protoresolve.GlobalDescriptors.AsTypePool(),
			opts: []resolvertest.Option{resolvertest.WithExtraFiles(), resolvertest.WithLenientErrors()},
		},
		{
			name: "Registry.AsTypePool",
			pool: newRegistry(t, corpus).AsTypePool(),
		},
		{
			name: "Registry.AsTypeResolver",
			pool: newRegistry(t, corpus).AsTypeResolver(),
		},
		{
			name: "TypesFromDescriptorPool",
			pool: protoresolve.TypesFromDescriptorPool(files),
		},
		{
			name: "TypesFromResolver(DescriptorPool)",
			pool: protoresolve.TypesFromResolver(protoresolve.ResolverFromPool(files)),
		},
		{
			name: "ResolverFromPool",
			pool: protoresolve.ResolverFromPool(files).AsTypeResolver(),
		},
		{
			// The type pool is the given *protoregistry.Types, so its errors
			// are not *protoresolve.ErrUnexpectedType.
			name: "ResolverFromPools",
			pool: protoresolve.ResolverFromPools(files, types).AsTypePool(),
			opts: []resolvertest.Option{resolvertest.WithLenientErrors()},
		},
		{
			name: "Combine",
			pool: protoresolve.Combine(inputs.resolvers...).AsTypeResolver(),
		},
		{
			name: "Combine(non-pools)",
			pool: protoresolve.Combine(inputs.nonPools...).AsTypeResolver(),
		},
		{
			name: "CombinePools.AsTypePool",
			pool: protoresolve.CombinePools(inputs.pools...).AsTypePool(),
		},
		{
			name: "CombinePools.AsTypeResolver",
			pool: protoresolve.CombinePools(inputs.pools...).AsTypeResolver(),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			pool, ok := testCase.pool.(protoresolve.TypePool)
			require.True(t, ok, "%T should implement TypePool", testCase.pool)
			resolvertest.CheckTypePool(t, pool, corpus, testCase.opts...)
		})
	}
}

func TestTypesFromResolver(t *testing.T) {
	t.Parallel()
	corpus := resolvertest.Corpus()
	reg := newRegistry(t, corpus)

	t.Run("resolver", func(t *testing.T) {
		t.Parallel()
		res := protoresolve.TypesFromResolver(struct {
			protoresolve.DescriptorResolver
			protoresolve.ExtensionResolver
		}{reg, reg})
		resolvertest.CheckTypeResolver(t, res, corpus)
	})
	t.Run("extension pool", func(t *testing.T) {
		t.Parallel()
		res := protoresolve.TypesFromResolver(struct {
			protoresolve.DescriptorResolver
			protoresolve.ExtensionPool
		}{reg, reg})
		_, ok := res.(interface {
			RangeExtensionsByMessage(protoreflect.FullName, func(protoreflect.ExtensionType) bool)
		})
		require.True(t, ok, "%T should have RangeExtensionsByMessage method", res)
		resolvertest.CheckTypeResolver(t, res, corpus)
	})
}
