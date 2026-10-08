package grpcreflect

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"

	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
)

func TestResetDoesNotWaitForServer(t *testing.T) {
	t.Parallel()
	client := newStubClient(t, func(*refv1.ServerReflectionRequest) *refv1.ServerReflectionResponse {
		return &refv1.ServerReflectionResponse{
			MessageResponse: &refv1.ServerReflectionResponse_ListServicesResponse{
				ListServicesResponse: &refv1.ListServiceResponse{},
			},
		}
	})
	_, err := client.ListServices()
	require.NoError(t, err)

	// The server never ends the stream, even after the client half-closes it.
	done := make(chan struct{})
	go func() {
		defer close(done)
		client.Reset()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Reset blocked waiting for the server")
	}
}

func TestErrorResponseWithOKCode(t *testing.T) {
	t.Parallel()
	client := newStubClient(t, func(*refv1.ServerReflectionRequest) *refv1.ServerReflectionResponse {
		return &refv1.ServerReflectionResponse{
			MessageResponse: &refv1.ServerReflectionResponse_ErrorResponse{
				ErrorResponse: &refv1.ErrorResponse{ErrorCode: int32(codes.OK), ErrorMessage: "server failure"},
			},
		}
	})
	t.Cleanup(client.Reset)
	_, err := client.ListServices()
	require.Error(t, err)
	assert.Equal(t, codes.Unknown, status.Code(err))
	assert.ErrorContains(t, err, "server failure")
}

// newStubClient returns a client for a reflection server that responds to every
// request using the given function. The server never ends a stream on its own.
func newStubClient(t *testing.T, respond func(*refv1.ServerReflectionRequest) *refv1.ServerReflectionResponse) *Client {
	t.Helper()
	svr := grpc.NewServer()
	refv1.RegisterServerReflectionServer(svr, stubReflectionServer{respond: respond})
	clientConn := prototesting.StartServer(t, svr)
	return NewClientV1(context.Background(), refv1.NewServerReflectionClient(clientConn))
}

type stubReflectionServer struct {
	refv1.UnimplementedServerReflectionServer
	respond func(*refv1.ServerReflectionRequest) *refv1.ServerReflectionResponse
}

func (s stubReflectionServer) ServerReflectionInfo(stream refv1.ServerReflection_ServerReflectionInfoServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			// Wait for the client to cancel, instead of ending the stream.
			<-stream.Context().Done()
			return stream.Context().Err()
		}
		if err := stream.Send(s.respond(req)); err != nil {
			return err
		}
	}
}
