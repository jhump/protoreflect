package grpcreflect

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
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

func TestConcurrentRequestsForSameElement(t *testing.T) {
	t.Parallel()
	const numGoroutines = 20
	file := testprotos.File_desc_test_complex_proto
	testCases := []struct {
		name   string
		lookup func(*Client) error
		// Identifies the request for the element.
		isRequest func(*refv1.ServerReflectionRequest) bool
	}{
		{
			name: "file",
			lookup: func(client *Client) error {
				_, err := client.FileByFilename(file.Path())
				return err
			},
			isRequest: func(req *refv1.ServerReflectionRequest) bool {
				return req.GetFileByFilename() == file.Path()
			},
		},
		{
			name: "symbol",
			lookup: func(client *Client) error {
				_, err := client.FileContainingSymbol(file.Messages().Get(0).FullName())
				return err
			},
			isRequest: func(req *refv1.ServerReflectionRequest) bool {
				return req.GetFileContainingSymbol() == string(file.Messages().Get(0).FullName())
			},
		},
		{
			name: "extension",
			lookup: func(client *Client) error {
				ext := file.Extensions().Get(0)
				_, err := client.FileContainingExtension(ext.ContainingMessage().FullName(), ext.Number())
				return err
			},
			isRequest: func(req *refv1.ServerReflectionRequest) bool {
				ext := file.Extensions().Get(0)
				extReq := req.GetFileContainingExtension()
				return extReq.GetContainingType() == string(ext.ContainingMessage().FullName()) &&
					extReq.GetExtensionNumber() == int32(ext.Number())
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var numRequests atomic.Int32
			// Counts requests for the element, and slows down responses, so
			// that concurrent lookups overlap.
			interceptor := func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
				return handler(srv, &countingServerStream{ServerStream: stream, isRequest: testCase.isRequest, numRequests: &numRequests})
			}
			clientConn := startReflectionServer(t, resolvertest.Corpus(), grpc.StreamInterceptor(interceptor))
			client := NewClientV1(context.Background(), refv1.NewServerReflectionClient(clientConn))
			t.Cleanup(client.Reset)

			var waitGroup sync.WaitGroup
			for range numGoroutines {
				waitGroup.Go(func() {
					assert.NoError(t, testCase.lookup(client))
				})
			}
			waitGroup.Wait()
			assert.Equal(t, int32(1), numRequests.Load(), "concurrent lookups should share one request")
		})
	}
}

type countingServerStream struct {
	grpc.ServerStream
	isRequest   func(*refv1.ServerReflectionRequest) bool
	numRequests *atomic.Int32
}

func (s *countingServerStream) RecvMsg(msg any) error {
	if err := s.ServerStream.RecvMsg(msg); err != nil {
		return err
	}
	if req, ok := msg.(*refv1.ServerReflectionRequest); ok && s.isRequest(req) {
		s.numRequests.Add(1)
	}
	return nil
}

func (s *countingServerStream) SendMsg(msg any) error {
	time.Sleep(10 * time.Millisecond)
	return s.ServerStream.SendMsg(msg)
}
