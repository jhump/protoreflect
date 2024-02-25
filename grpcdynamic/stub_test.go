package grpcdynamic

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/grpcreflect"
	grpctestdata "github.com/jhump/protoreflect/v2/internal/testdata/grpc"
	grpctesting "github.com/jhump/protoreflect/v2/internal/testing"
)

var (
	unaryMd           protoreflect.MethodDescriptor
	clientStreamingMd protoreflect.MethodDescriptor
	serverStreamingMd protoreflect.MethodDescriptor
	bidiStreamingMd   protoreflect.MethodDescriptor
	stub              *Stub
)

func TestMain(m *testing.M) {
	code := 1
	defer func() {
		p := recover()
		if p != nil {
			_, _ = fmt.Fprintf(os.Stderr, "PANIC: %v\n", p)
		}
		os.Exit(code)
	}()

	// Start up a server on an ephemeral port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("Failed to listen to port: %s", err.Error()))
	}
	svr := grpc.NewServer()
	grpctestdata.RegisterTestServiceServer(svr, grpctesting.TestService{})
	go func() {
		_ = svr.Serve(l)
	}()
	defer svr.Stop()

	svcs, err := grpcreflect.LoadServiceDescriptors(svr)
	if err != nil {
		panic(fmt.Sprintf("Failed to load service descriptor: %s", err.Error()))
	}
	sd := svcs["grpc.testing.TestService"]
	unaryMd = sd.Methods().ByName("UnaryCall")
	clientStreamingMd = sd.Methods().ByName("StreamingInputCall")
	serverStreamingMd = sd.Methods().ByName("StreamingOutputCall")
	bidiStreamingMd = sd.Methods().ByName("FullDuplexCall")

	// Start up client that talks to the same port
	cc, err := grpc.Dial(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to create client to %s: %s", l.Addr().String(), err.Error()))
	}
	defer func() {
		_ = cc.Close()
	}()

	stub = NewStub(cc)

	code = m.Run()
}

var payload = &grpctestdata.Payload{
	Type: grpctestdata.PayloadType_RANDOM,
	Body: []byte{3, 14, 159, 2, 65, 35, 9},
}

func TestUnaryRpc(t *testing.T) {
	resp, err := stub.InvokeRpc(context.Background(), unaryMd, &grpctestdata.SimpleRequest{Payload: payload})
	require.NoError(t, err, "Failed to invoke unary RPC")
	refMsg := resp.ProtoReflect()
	fd := refMsg.Descriptor().Fields().ByName("payload")
	p := refMsg.Get(fd)
	require.True(t, proto.Equal(p.Message().Interface(), payload), "Incorrect payload returned from RPC: %v != %v", p, payload)
}

func TestClientStreamingRpc(t *testing.T) {
	cs, err := stub.InvokeRpcClientStream(context.Background(), clientStreamingMd)
	require.NoError(t, err, "Failed to invoke client-streaming RPC")
	req := &grpctestdata.StreamingInputCallRequest{Payload: payload}
	for i := 0; i < 3; i++ {
		err = cs.SendMsg(req)
		require.NoError(t, err, "Failed to send request message")
	}
	resp, err := cs.CloseAndReceive()
	require.NoError(t, err, "Failed to receive response")
	refMsg := resp.ProtoReflect()
	fd := refMsg.Descriptor().Fields().ByName("aggregated_payload_size")
	sz := refMsg.Get(fd)
	expectedSz := 3 * len(payload.Body)
	require.Equal(t, expectedSz, int(sz.Int()), "Incorrect response returned from RPC")
}

func TestServerStreamingRpc(t *testing.T) {
	ss, err := stub.InvokeRpcServerStream(context.Background(), serverStreamingMd, &grpctestdata.StreamingOutputCallRequest{
		Payload: payload,
		ResponseParameters: []*grpctestdata.ResponseParameters{
			{}, {}, {}, // three entries means we'll get back three responses
		},
	})
	require.NoError(t, err, "Failed to invoke server-streaming RPC")
	for i := 0; i < 3; i++ {
		resp, err := ss.RecvMsg()
		require.NoError(t, err, "Failed to receive response message")
		refMsg := resp.ProtoReflect()
		fd := refMsg.Descriptor().Fields().ByName("payload")
		p := refMsg.Get(fd)
		require.True(t, proto.Equal(p.Message().Interface(), payload), "Incorrect payload returned from RPC: %v != %v", p, payload)
	}
	_, err = ss.RecvMsg()
	require.Equal(t, io.EOF, err, "Incorrect number of messages in response")
}

func TestBidiStreamingRpc(t *testing.T) {
	bds, err := stub.InvokeRpcBidiStream(context.Background(), bidiStreamingMd)
	require.NoError(t, err)
	req := &grpctestdata.StreamingOutputCallRequest{Payload: payload}
	for i := 0; i < 3; i++ {
		err = bds.SendMsg(req)
		require.NoError(t, err, "Failed to send request message")
		resp, err := bds.RecvMsg()
		require.NoError(t, err, "Failed to receive response message")
		refMsg := resp.ProtoReflect()
		fd := refMsg.Descriptor().Fields().ByName("payload")
		p := refMsg.Get(fd)
		require.True(t, proto.Equal(p.Message().Interface(), payload), "Incorrect payload returned from RPC: %v != %v", p, payload)
	}
	err = bds.CloseSend()
	require.NoError(t, err, "Failed to receive response")
	_, err = bds.RecvMsg()
	require.Equal(t, io.EOF, err, "Incorrect number of messages in response")
}
