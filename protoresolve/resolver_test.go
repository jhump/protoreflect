package protoresolve_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestResolvers(t *testing.T) {
	t.Parallel()
	corpus := resolvertest.Corpus()
	files := newFiles(t, corpus)
	types := newTypes(t, files)

	regFromProtos := &protoresolve.Registry{}
	for _, fileProto := range fileProtos(corpus) {
		_, err := regFromProtos.RegisterFileProto(fileProto)
		require.NoError(t, err)
	}
	// Reverse the files, so they are not in topological order, to verify
	// that FromFileDescriptorSet sorts them.
	fileSet := &descriptorpb.FileDescriptorSet{File: fileProtos(corpus)}
	slices.Reverse(fileSet.File)
	regFromSet, err := protoresolve.FromFileDescriptorSet(fileSet)
	require.NoError(t, err)
	regFromFiles, err := protoresolve.FromFiles(newFiles(t, corpus))
	require.NoError(t, err)
	regFromGlobalFiles, err := protoresolve.FromFiles(protoregistry.GlobalFiles)
	require.NoError(t, err)

	inputs := newCombinedInputs(t, corpus)

	testCases := []struct {
		name     string
		resolver protoresolve.Resolver
		opts     []resolvertest.Option
	}{
		{
			name:     "Registry",
			resolver: newRegistry(t, corpus),
		},
		{
			name:     "Registry.RegisterFileProto",
			resolver: regFromProtos,
		},
		{
			name:     "FromFileDescriptorSet",
			resolver: regFromSet,
		},
		{
			name:     "FromFiles",
			resolver: regFromFiles,
		},
		{
			name:     "FromFiles(GlobalFiles)",
			resolver: regFromGlobalFiles,
			opts:     []resolvertest.Option{resolvertest.WithExtraFiles()},
		},
		{
			name:     "GlobalDescriptors",
			resolver: protoresolve.GlobalDescriptors,
			opts:     []resolvertest.Option{resolvertest.WithExtraFiles(), resolvertest.WithLenientTypeErrors()},
		},
		{
			name:     "ResolverFromPool",
			resolver: protoresolve.ResolverFromPool(files),
		},
		{
			name:     "ResolverFromPools",
			resolver: protoresolve.ResolverFromPools(files, types),
			opts:     []resolvertest.Option{resolvertest.WithLenientTypeErrors()},
		},
		{
			// Combined resolvers only report the number of files in the first.
			name:     "Combine",
			resolver: protoresolve.Combine(inputs.resolvers...),
			opts:     []resolvertest.Option{resolvertest.WithInexactFileCounts()},
		},
		{
			name:     "Combine(non-pools)",
			resolver: protoresolve.Combine(inputs.nonPools...),
			opts:     []resolvertest.Option{resolvertest.WithInexactFileCounts()},
		},
		{
			name:     "CombinePools",
			resolver: protoresolve.CombinePools(inputs.pools...),
			opts:     []resolvertest.Option{resolvertest.WithInexactFileCounts()},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			resolvertest.CheckResolver(t, testCase.resolver, corpus, testCase.opts...)
		})
	}
}

// fileProtos returns descriptor protos for the given files. Each call returns
// new protos, since registering a file proto takes ownership of it.
func fileProtos(files []protoreflect.FileDescriptor) []*descriptorpb.FileDescriptorProto {
	protos := make([]*descriptorpb.FileDescriptorProto, len(files))
	for i, file := range files {
		protos[i] = protodesc.ToFileDescriptorProto(file)
	}
	return protos
}
