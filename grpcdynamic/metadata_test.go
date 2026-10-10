package grpcdynamic

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/emptypb"

	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	grpctestprotos "github.com/jhump/protoreflect/v2/internal/testprotos/grpc"
)

func TestStreamMetadata(t *testing.T) {
	t.Parallel()
	metadataStub := newMetadataStub(t)
	ctx := t.Context()
	checkMetadata := func(t *testing.T, header metadata.MD, headerErr error, trailer metadata.MD) {
		t.Helper()
		require.NoError(t, headerErr)
		assert.Equal(t, []string{"header-value"}, header.Get("header-key"))
		assert.Equal(t, []string{"trailer-value"}, trailer.Get("trailer-key"))
	}

	t.Run("server stream", func(t *testing.T) {
		t.Parallel()
		stream, err := metadataStub.InvokeRPCServerStream(ctx, serverStreamingMd, &grpctestprotos.StreamingOutputCallRequest{})
		require.NoError(t, err)
		assert.NoError(t, stream.Context().Err())
		header, headerErr := stream.Header()
		_, err = stream.RecvMsg()
		require.NoError(t, err)
		_, err = stream.RecvMsg()
		require.Equal(t, io.EOF, err)
		checkMetadata(t, header, headerErr, stream.Trailer())
	})
	t.Run("client stream", func(t *testing.T) {
		t.Parallel()
		stream, err := metadataStub.InvokeRPCClientStream(ctx, clientStreamingMd)
		require.NoError(t, err)
		assert.NoError(t, stream.Context().Err())
		require.NoError(t, stream.SendMsg(&grpctestprotos.StreamingInputCallRequest{}))
		header, headerErr := stream.Header()
		_, err = stream.CloseAndReceive()
		require.NoError(t, err)
		checkMetadata(t, header, headerErr, stream.Trailer())
	})
	t.Run("bidi stream", func(t *testing.T) {
		t.Parallel()
		stream, err := metadataStub.InvokeRPCBidiStream(ctx, bidiStreamingMd)
		require.NoError(t, err)
		assert.NoError(t, stream.Context().Err())
		require.NoError(t, stream.SendMsg(&grpctestprotos.StreamingOutputCallRequest{}))
		header, headerErr := stream.Header()
		_, err = stream.RecvMsg()
		require.NoError(t, err)
		require.NoError(t, stream.CloseSend())
		_, err = stream.RecvMsg()
		require.Equal(t, io.EOF, err)
		checkMetadata(t, header, headerErr, stream.Trailer())
	})
}

func TestWithResolver(t *testing.T) {
	t.Parallel()
	req := &grpctestprotos.SimpleRequest{Payload: payload}

	// A resolver that knows the response type produces that type.
	globalStub := NewStub(stub.channel, WithResolver(protoregistry.GlobalTypes))
	resp, err := globalStub.InvokeRPC(t.Context(), unaryMd, req)
	require.NoError(t, err)
	assert.IsType(t, &grpctestprotos.SimpleResponse{}, resp)

	// Otherwise, the response is a dynamic message.
	emptyStub := NewStub(stub.channel, WithResolver(&protoregistry.Types{}))
	resp, err = emptyStub.InvokeRPC(t.Context(), unaryMd, req)
	require.NoError(t, err)
	assert.IsType(t, &dynamicpb.Message{}, resp)
	assert.Equal(t, unaryMd.Output().FullName(), resp.ProtoReflect().Descriptor().FullName())

	stream, err := emptyStub.InvokeRPCServerStream(t.Context(), serverStreamingMd, &grpctestprotos.StreamingOutputCallRequest{
		Payload:            payload,
		ResponseParameters: []*grpctestprotos.ResponseParameters{{}},
	})
	require.NoError(t, err)
	resp, err = stream.RecvMsg()
	require.NoError(t, err)
	assert.IsType(t, &dynamicpb.Message{}, resp)
}

// newMetadataStub returns a stub for a server that sends header and trailer
// metadata for every call. It handles every method by sending one empty
// response message for every request message.
func newMetadataStub(t *testing.T) *Stub {
	t.Helper()
	handler := func(_ any, stream grpc.ServerStream) error {
		if err := stream.SendHeader(metadata.Pairs("header-key", "header-value")); err != nil {
			return err
		}
		stream.SetTrailer(metadata.Pairs("trailer-key", "trailer-value"))
		for {
			if err := stream.RecvMsg(&emptypb.Empty{}); err == io.EOF {
				return nil
			} else if err != nil {
				return err
			}
			if err := stream.SendMsg(&emptypb.Empty{}); err != nil {
				return err
			}
		}
	}
	svr := grpc.NewServer(grpc.UnknownServiceHandler(handler))
	clientConn := prototesting.StartServer(t, svr)
	return NewStub(clientConn)
}
