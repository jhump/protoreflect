package grpcreflect

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	refv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestClientAsResolver(t *testing.T) {
	t.Parallel()
	corpus := resolvertest.Corpus()
	clientConn := startReflectionServer(t, corpus)

	ctx := context.Background()
	newClients := map[string]func() *Client{
		"v1": func() *Client {
			return NewClientV1(ctx, refv1.NewServerReflectionClient(clientConn))
		},
		"v1alpha": func() *Client {
			return NewClientV1Alpha(ctx, refv1alpha.NewServerReflectionClient(clientConn))
		},
	}
	for name, newClient := range newClients {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := newClient()
			t.Cleanup(client.Reset)
			res := client.AsResolver()

			// Queries download files on demand.
			assert.Zero(t, res.NumFiles())
			_, err := res.FindMessageByName("testprotos.TestMessage")
			require.NoError(t, err)
			assert.NotZero(t, res.NumFiles())

			// But enumeration only includes files already downloaded, so
			// download the whole corpus before checking.
			for _, file := range corpus {
				_, err := client.FileByFilename(file.Path())
				require.NoError(t, err)
			}
			resolvertest.CheckResolver(t, res, corpus)
		})
	}
}

func TestClientAsResolverWithFallback(t *testing.T) {
	t.Parallel()
	// The server does not have this file, but the fallback resolvers do.
	fallbackFile := testprotos.File_desc_test_complex_proto
	served := slices.DeleteFunc(resolvertest.Corpus(), func(file protoreflect.FileDescriptor) bool {
		return file.Path() == fallbackFile.Path()
	})
	clientConn := startReflectionServer(t, served)
	client := NewClientV1(context.Background(), refv1.NewServerReflectionClient(clientConn),
		WithFallbackResolvers(protoregistry.GlobalFiles, protoregistry.GlobalTypes))
	t.Cleanup(client.Reset)
	res := client.AsResolver()

	msg := fallbackFile.Messages().Get(0)
	found, err := res.FindMessageByName(msg.FullName())
	require.NoError(t, err)
	assert.Equal(t, msg.FullName(), found.FullName())
	foundDesc, err := res.FindDescriptorByName(msg.FullName())
	require.NoError(t, err)
	assert.Equal(t, msg.FullName(), foundDesc.FullName())
	found, err = res.FindMessageByURL("type.googleapis.com/" + string(msg.FullName()))
	require.NoError(t, err)
	assert.Equal(t, msg.FullName(), found.FullName())

	ext := fallbackFile.Extensions().Get(0)
	foundExt, err := res.FindExtensionByName(ext.FullName())
	require.NoError(t, err)
	assert.Equal(t, ext.FullName(), foundExt.FullName())
	foundExt, err = res.FindExtensionByNumber(ext.ContainingMessage().FullName(), ext.Number())
	require.NoError(t, err)
	assert.Equal(t, ext.FullName(), foundExt.FullName())

	// The server does not know about this message, but the fallback knows
	// about extensions for it.
	extendee := fallbackFile.Messages().ByName("Test")
	require.NotNil(t, extendee)
	var expected []protoreflect.FieldNumber
	protoregistry.GlobalTypes.RangeExtensionsByMessage(extendee.FullName(), func(extType protoreflect.ExtensionType) bool {
		expected = append(expected, extType.TypeDescriptor().Number())
		return true
	})
	require.NotEmpty(t, expected)
	nums, err := client.AllExtensionNumbersForType(extendee.FullName())
	require.NoError(t, err)
	assert.ElementsMatch(t, expected, nums)
}

// startReflectionServer starts a reflection server that serves exactly the
// given files and returns a connection to it.
func startReflectionServer(t *testing.T, files []protoreflect.FileDescriptor) *grpc.ClientConn {
	t.Helper()
	var reg protoregistry.Files
	for _, file := range files {
		require.NoError(t, reg.RegisterFile(file))
	}
	var types protoregistry.Types
	require.NoError(t, protoresolve.RegisterTypesInFilesRecursive(&reg, &types, protoresolve.TypeKindsAll))
	serverOpts := reflection.ServerOptions{DescriptorResolver: &reg, ExtensionResolver: &types}
	svr := grpc.NewServer()
	refv1.RegisterServerReflectionServer(svr, reflection.NewServerV1(serverOpts))
	refv1alpha.RegisterServerReflectionServer(svr, reflection.NewServer(serverOpts))
	return prototesting.StartServer(t, svr)
}
