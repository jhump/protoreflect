package grpcdynamic

import (
	"fmt"
	"io"
	"net"
	"os"
	"testing"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/grpc_testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/jhump/protoreflect/internal/testutil"
)

var unaryMd *desc.MethodDescriptor
var clientStreamingMd *desc.MethodDescriptor
var serverStreamingMd *desc.MethodDescriptor
var bidiStreamingMd *desc.MethodDescriptor
var stub Stub

func TestMain(m *testing.M) {
	code := 1
	defer func() {
		p := recover()
		if p != nil {
			fmt.Fprintf(os.Stderr, "PANIC: %v\n", p)
		}
		os.Exit(code)
	}()

	// Start up a server on an ephemeral port
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("Failed to listen to port: %s", err.Error()))
	}
	svr := grpc.NewServer()
	grpc_testing.RegisterTestServiceServer(svr, testutil.TestService{})
	go svr.Serve(l)
	defer svr.Stop()

	svcs, err := grpcreflect.LoadServiceDescriptors(svr)
	if err != nil {
		panic(fmt.Sprintf("Failed to load service descriptor: %s", err.Error()))
	}
	sd := svcs["grpc.testing.TestService"]
	unaryMd = sd.FindMethodByName("UnaryCall")
	clientStreamingMd = sd.FindMethodByName("StreamingInputCall")
	serverStreamingMd = sd.FindMethodByName("StreamingOutputCall")
	bidiStreamingMd = sd.FindMethodByName("FullDuplexCall")

	// Start up client that talks to the same port
	cc, err := grpc.Dial(l.Addr().String(), grpc.WithInsecure())
	if err != nil {
		panic(fmt.Sprintf("Failed to create client to %s: %s", l.Addr().String(), err.Error()))
	}
	defer cc.Close()

	stub = NewStub(cc)

	code = m.Run()
}

var payload = &grpc_testing.Payload{
	Type: grpc_testing.PayloadType_RANDOM,
	Body: []byte{3, 14, 159, 2, 65, 35, 9},
}

func TestUnaryRpc(t *testing.T) {
	resp, err := stub.InvokeRpc(context.Background(), unaryMd, &grpc_testing.SimpleRequest{Payload: payload})
	testutil.Ok(t, err, "Failed to invoke unary RPC")
	dm := resp.(*dynamic.Message)
	fd := dm.GetMessageDescriptor().FindFieldByName("payload")
	p := dm.GetField(fd)
	testutil.Require(t, dynamic.MessagesEqual(p.(proto.Message), payload), "Incorrect payload returned from RPC: %v != %v", p, payload)
}

func TestClientStreamingRpc(t *testing.T) {
	cs, err := stub.InvokeRpcClientStream(context.Background(), clientStreamingMd)
	testutil.Ok(t, err, "Failed to invoke client-streaming RPC")
	req := &grpc_testing.StreamingInputCallRequest{Payload: payload}
	for i := 0; i < 3; i++ {
		err = cs.SendMsg(req)
		testutil.Ok(t, err, "Failed to send request message")
	}
	resp, err := cs.CloseAndReceive()
	testutil.Ok(t, err, "Failed to receive response")
	dm := resp.(*dynamic.Message)
	fd := dm.GetMessageDescriptor().FindFieldByName("aggregated_payload_size")
	sz := dm.GetField(fd)
	expectedSz := 3 * len(payload.Body)
	testutil.Eq(t, expectedSz, int(sz.(int32)), "Incorrect response returned from RPC")
}

func TestServerStreamingRpc(t *testing.T) {
	ss, err := stub.InvokeRpcServerStream(context.Background(), serverStreamingMd, &grpc_testing.StreamingOutputCallRequest{
		Payload: payload,
		ResponseParameters: []*grpc_testing.ResponseParameters{
			{}, {}, {}, // three entries means we'll get back three responses
		},
	})
	testutil.Ok(t, err, "Failed to invoke server-streaming RPC")
	for i := 0; i < 3; i++ {
		resp, err := ss.RecvMsg()
		testutil.Ok(t, err, "Failed to receive response message")
		dm := resp.(*dynamic.Message)
		fd := dm.GetMessageDescriptor().FindFieldByName("payload")
		p := dm.GetField(fd)
		testutil.Require(t, dynamic.MessagesEqual(p.(proto.Message), payload), "Incorrect payload returned from RPC: %v != %v", p, payload)
	}
	_, err = ss.RecvMsg()
	testutil.Eq(t, io.EOF, err, "Incorrect number of messages in response")
}

func TestBidiStreamingRpc(t *testing.T) {
	bds, err := stub.InvokeRpcBidiStream(context.Background(), bidiStreamingMd)
	testutil.Ok(t, err)
	req := &grpc_testing.StreamingOutputCallRequest{Payload: payload}
	for i := 0; i < 3; i++ {
		err = bds.SendMsg(req)
		testutil.Ok(t, err, "Failed to send request message")
		resp, err := bds.RecvMsg()
		testutil.Ok(t, err, "Failed to receive response message")
		dm := resp.(*dynamic.Message)
		fd := dm.GetMessageDescriptor().FindFieldByName("payload")
		p := dm.GetField(fd)
		testutil.Require(t, dynamic.MessagesEqual(p.(proto.Message), payload), "Incorrect payload returned from RPC: %v != %v", p, payload)
	}
	err = bds.CloseSend()
	testutil.Ok(t, err, "Failed to receive response")
	_, err = bds.RecvMsg()
	testutil.Eq(t, io.EOF, err, "Incorrect number of messages in response")
}
