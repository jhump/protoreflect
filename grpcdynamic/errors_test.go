package grpcdynamic

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"

	grpctestprotos "github.com/jhump/protoreflect/v2/internal/testprotos/grpc"
)

func TestWrongMethodType(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	_, err := stub.InvokeRpc(ctx, clientStreamingMd, &grpctestprotos.StreamingInputCallRequest{})
	assert.ErrorContains(t, err, "is client-streaming")
	_, err = stub.InvokeRpcServerStream(ctx, unaryMd, &grpctestprotos.SimpleRequest{})
	assert.ErrorContains(t, err, "is unary")
	_, err = stub.InvokeRpcClientStream(ctx, bidiStreamingMd)
	assert.ErrorContains(t, err, "is bidi-streaming")
	_, err = stub.InvokeRpcBidiStream(ctx, serverStreamingMd)
	assert.ErrorContains(t, err, "is server-streaming")
}

func TestWrongMessageType(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	wrongMsg := &grpctestprotos.Payload{}
	const errSubstr = "got grpc.testing.Payload"
	_, err := stub.InvokeRpc(ctx, unaryMd, wrongMsg)
	assert.ErrorContains(t, err, errSubstr)
	_, err = stub.InvokeRpcServerStream(ctx, serverStreamingMd, wrongMsg)
	assert.ErrorContains(t, err, errSubstr)

	clientStream, err := stub.InvokeRpcClientStream(ctx, clientStreamingMd)
	require.NoError(t, err)
	assert.ErrorContains(t, clientStream.SendMsg(wrongMsg), errSubstr)
	_, err = clientStream.CloseAndReceive()
	require.NoError(t, err)

	bidiStream, err := stub.InvokeRpcBidiStream(ctx, bidiStreamingMd)
	require.NoError(t, err)
	assert.ErrorContains(t, bidiStream.SendMsg(wrongMsg), errSubstr)
	require.NoError(t, bidiStream.CloseSend())
	_, err = bidiStream.RecvMsg()
	assert.Equal(t, io.EOF, err)
}

func TestNilMessage(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	const errSubstr = "got nil"
	_, err := stub.InvokeRpc(ctx, unaryMd, nil)
	assert.ErrorContains(t, err, errSubstr)
	_, err = stub.InvokeRpcServerStream(ctx, serverStreamingMd, nil)
	assert.ErrorContains(t, err, errSubstr)

	clientStream, err := stub.InvokeRpcClientStream(ctx, clientStreamingMd)
	require.NoError(t, err)
	assert.ErrorContains(t, clientStream.SendMsg(nil), errSubstr)
	_, err = clientStream.CloseAndReceive()
	require.NoError(t, err)
}

func TestChannelErrors(t *testing.T) {
	t.Parallel()
	errChannel := errors.New("channel failure")
	ctx := t.Context()

	t.Run("invoke", func(t *testing.T) {
		t.Parallel()
		fakeStub := NewStub(&fakeChannel{invokeErr: errChannel})
		_, err := fakeStub.InvokeRpc(ctx, unaryMd, &grpctestprotos.SimpleRequest{})
		assert.ErrorIs(t, err, errChannel)
	})
	t.Run("new stream", func(t *testing.T) {
		t.Parallel()
		fakeStub := NewStub(&fakeChannel{newStreamErr: errChannel})
		_, err := fakeStub.InvokeRpcServerStream(ctx, serverStreamingMd, &grpctestprotos.StreamingOutputCallRequest{})
		assert.ErrorIs(t, err, errChannel)
		_, err = fakeStub.InvokeRpcClientStream(ctx, clientStreamingMd)
		assert.ErrorIs(t, err, errChannel)
		_, err = fakeStub.InvokeRpcBidiStream(ctx, bidiStreamingMd)
		assert.ErrorIs(t, err, errChannel)
	})
	t.Run("server stream send", func(t *testing.T) {
		t.Parallel()
		fakeStub := NewStub(&fakeChannel{stream: &fakeStream{sendErr: errChannel}})
		_, err := fakeStub.InvokeRpcServerStream(ctx, serverStreamingMd, &grpctestprotos.StreamingOutputCallRequest{})
		assert.ErrorIs(t, err, errChannel)
	})
	t.Run("server stream close send", func(t *testing.T) {
		t.Parallel()
		fakeStub := NewStub(&fakeChannel{stream: &fakeStream{closeSendErr: errChannel}})
		_, err := fakeStub.InvokeRpcServerStream(ctx, serverStreamingMd, &grpctestprotos.StreamingOutputCallRequest{})
		assert.ErrorIs(t, err, errChannel)
	})
}

func TestCloseAndReceiveErrors(t *testing.T) {
	t.Parallel()
	errChannel := errors.New("channel failure")
	response := &grpctestprotos.StreamingInputCallResponse{AggregatedPayloadSize: 123}
	testCases := []struct {
		name      string
		stream    *fakeStream
		errSubstr string
	}{
		{
			name:      "close send",
			stream:    &fakeStream{closeSendErr: errChannel},
			errSubstr: errChannel.Error(),
		},
		{
			name:      "no response",
			stream:    &fakeStream{recvErr: errChannel},
			errSubstr: errChannel.Error(),
		},
		{
			name:      "error after response",
			stream:    &fakeStream{responses: []proto.Message{response}, recvErr: errChannel},
			errSubstr: errChannel.Error(),
		},
		{
			name:      "too many responses",
			stream:    &fakeStream{responses: []proto.Message{response, response}},
			errSubstr: "returned more than one response message",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			fakeStub := NewStub(&fakeChannel{stream: testCase.stream})
			clientStream, err := fakeStub.InvokeRpcClientStream(t.Context(), clientStreamingMd)
			require.NoError(t, err)
			_, err = clientStream.CloseAndReceive()
			assert.ErrorContains(t, err, testCase.errSubstr)
		})
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		fakeStub := NewStub(&fakeChannel{stream: &fakeStream{responses: []proto.Message{response}}})
		clientStream, err := fakeStub.InvokeRpcClientStream(t.Context(), clientStreamingMd)
		require.NoError(t, err)
		resp, err := clientStream.CloseAndReceive()
		require.NoError(t, err)
		assert.True(t, proto.Equal(response, resp))
	})
}

func TestStreamCleanup(t *testing.T) {
	t.Parallel()
	checkCleanup := func(t *testing.T, stream *fakeStream) {
		t.Helper()
		assert.Error(t, stream.ctx.Err(), "stream's context should be cancelled once the stream is finished")
		// Calling Context on a gRPC stream commits to the current attempt,
		// which disables retries.
		assert.False(t, stream.contextCalled.Load(), "stream's Context method should not be called")
	}

	t.Run("server stream", func(t *testing.T) {
		t.Parallel()
		stream := &fakeStream{responses: []proto.Message{&grpctestprotos.StreamingOutputCallResponse{}}}
		fakeStub := NewStub(&fakeChannel{stream: stream})
		serverStream, err := fakeStub.InvokeRpcServerStream(t.Context(), serverStreamingMd, &grpctestprotos.StreamingOutputCallRequest{})
		require.NoError(t, err)
		_, err = serverStream.RecvMsg()
		require.NoError(t, err)
		assert.NoError(t, stream.ctx.Err(), "stream's context should not be cancelled before stream is finished")
		_, err = serverStream.RecvMsg()
		require.Equal(t, io.EOF, err)
		checkCleanup(t, stream)
	})
	t.Run("client stream", func(t *testing.T) {
		t.Parallel()
		stream := &fakeStream{responses: []proto.Message{&grpctestprotos.StreamingInputCallResponse{}}}
		fakeStub := NewStub(&fakeChannel{stream: stream})
		clientStream, err := fakeStub.InvokeRpcClientStream(t.Context(), clientStreamingMd)
		require.NoError(t, err)
		require.NoError(t, clientStream.SendMsg(&grpctestprotos.StreamingInputCallRequest{}))
		assert.NoError(t, stream.ctx.Err(), "stream's context should not be cancelled before stream is finished")
		_, err = clientStream.CloseAndReceive()
		require.NoError(t, err)
		checkCleanup(t, stream)
	})
}

// fakeChannel is a channel whose streams are backed by the given fakeStream.
type fakeChannel struct {
	invokeErr    error
	newStreamErr error
	stream       *fakeStream
}

func (c *fakeChannel) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return c.invokeErr
}

func (c *fakeChannel) NewStream(ctx context.Context, _ *grpc.StreamDesc, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
	if c.newStreamErr != nil {
		return nil, c.newStreamErr
	}
	c.stream.ctx = ctx
	return c.stream, nil
}

// fakeStream is a stream that returns the given responses, in order, from
// RecvMsg. After that, RecvMsg returns recvErr or, if nil, io.EOF.
type fakeStream struct {
	ctx          context.Context
	sendErr      error
	closeSendErr error
	responses    []proto.Message
	recvErr      error
	// Set when Context is called.
	contextCalled atomic.Bool
}

func (s *fakeStream) Header() (metadata.MD, error) { return nil, nil }

func (s *fakeStream) Trailer() metadata.MD { return nil }

func (s *fakeStream) CloseSend() error { return s.closeSendErr }

func (s *fakeStream) Context() context.Context {
	s.contextCalled.Store(true)
	return s.ctx
}

func (s *fakeStream) SendMsg(any) error { return s.sendErr }

func (s *fakeStream) RecvMsg(msg any) error {
	if len(s.responses) == 0 {
		if s.recvErr != nil {
			return s.recvErr
		}
		return io.EOF
	}
	protoMsg, ok := msg.(proto.Message)
	if !ok {
		return fmt.Errorf("unexpected message type %T", msg)
	}
	data, err := proto.Marshal(s.responses[0])
	if err != nil {
		return err
	}
	s.responses = s.responses[1:]
	return proto.Unmarshal(data, protoMsg)
}
