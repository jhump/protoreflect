// Package grpcdynamic provides a dynamic RPC stub. It can be used to invoke RPC
// methods that are unknown at compile time, using method descriptors to drive the
// invocations at runtime. The actual request and response messages may be (and
// likely often are) dynamic messages.
package grpcdynamic

import (
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/protomessage"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

// Stub is an RPC client stub, used for dynamically dispatching RPCs to a server.
type Stub struct {
	channel  grpc.ClientConnInterface
	resolver protoresolve.SerializationResolver
}

// NewStub creates a new RPC stub that uses the given channel for dispatching RPCs.
func NewStub(channel grpc.ClientConnInterface, opts ...StubOption) *Stub {
	stub := &Stub{channel: channel}
	for _, opt := range opts {
		opt.apply(stub)
	}
	return stub
}

// StubOption is an option that can be used to customize behavior when creating a Stub.
type StubOption interface {
	apply(*Stub)
}

type stubOptionFunc func(*Stub)

func (s stubOptionFunc) apply(stub *Stub) {
	s(stub)
}

// WithResolver returns a StubOption that causes a Stub to use the given resolver for
// de-serializing response messages. If not specified, [protoregistry.GlobalTypes] is
// used. If the given resolver does not support the response message type, a dynamic
// message is used. The given resolver is also used for recognizing extensions in
// response messages.
func WithResolver(res protoresolve.SerializationResolver) StubOption {
	return stubOptionFunc(func(s *Stub) {
		s.resolver = res
	})
}

func requestMethod(md protoreflect.MethodDescriptor) string {
	return fmt.Sprintf("/%s/%s", md.Parent().FullName(), md.Name())
}

// InvokeRpc sends a unary RPC and returns the response. Use this for unary methods.
func (s *Stub) InvokeRpc(ctx context.Context, method protoreflect.MethodDescriptor, request proto.Message, opts ...grpc.CallOption) (proto.Message, error) {
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, fmt.Errorf("InvokeRpc is for unary methods; %q is %s", method.FullName(), methodType(method))
	}
	if err := checkMessageType(method.Input(), request); err != nil {
		return nil, err
	}
	resp := newMessage(method.Output(), s.resolver)
	if err := s.channel.Invoke(ctx, requestMethod(method), request, resp, opts...); err != nil {
		return nil, err
	}
	if s.resolver != nil {
		protomessage.ReparseUnrecognized(resp, s.resolver)
	}
	return resp, nil
}

// InvokeRpcServerStream sends a unary RPC and returns the response stream. Use this for server-streaming methods.
//
// To release the stream's resources, callers must either call RecvMsg until it
// returns an error (which is io.EOF when the stream completes normally) or
// cancel ctx.
func (s *Stub) InvokeRpcServerStream(ctx context.Context, method protoreflect.MethodDescriptor, request proto.Message, opts ...grpc.CallOption) (*ServerStream, error) {
	if method.IsStreamingClient() || !method.IsStreamingServer() {
		return nil, fmt.Errorf("InvokeRpcServerStream is for server-streaming methods; %q is %s", method.FullName(), methodType(method))
	}
	if err := checkMessageType(method.Input(), request); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	cs, err := s.channel.NewStream(ctx, streamDesc(method), requestMethod(method), opts...)
	if err != nil {
		cancel()
		return nil, err
	}
	err = cs.SendMsg(request)
	if errors.Is(err, io.EOF) {
		// The server already ended the call. The actual error, with the
		// call's status, comes from receiving, and the caller has no stream
		// with which to do that. So we do it here.
		if recvErr := cs.RecvMsg(newMessage(method.Output(), s.resolver)); recvErr != nil {
			err = recvErr
		}
	}
	if err != nil {
		cancel()
		return nil, err
	}
	err = cs.CloseSend()
	if err != nil {
		cancel()
		return nil, err
	}
	return &ServerStream{baseStream{cs}, method.Output(), s.resolver, cancel}, nil
}

// InvokeRpcClientStream creates a new stream that is used to send request messages and, at the end,
// receive the response message. Use this for client-streaming methods.
func (s *Stub) InvokeRpcClientStream(ctx context.Context, method protoreflect.MethodDescriptor, opts ...grpc.CallOption) (*ClientStream, error) {
	if !method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, fmt.Errorf("InvokeRpcClientStream is for client-streaming methods; %q is %s", method.FullName(), methodType(method))
	}
	ctx, cancel := context.WithCancel(ctx)
	cs, err := s.channel.NewStream(ctx, streamDesc(method), requestMethod(method), opts...)
	if err != nil {
		cancel()
		return nil, err
	}
	return &ClientStream{baseStream{cs}, method, s.resolver, cancel}, nil
}

// InvokeRpcBidiStream creates a new stream that is used to both send request messages and receive response
// messages. Use this for bidi-streaming methods.
func (s *Stub) InvokeRpcBidiStream(ctx context.Context, method protoreflect.MethodDescriptor, opts ...grpc.CallOption) (*BidiStream, error) {
	if !method.IsStreamingClient() || !method.IsStreamingServer() {
		return nil, fmt.Errorf("InvokeRpcBidiStream is for bidi-streaming methods; %q is %s", method.FullName(), methodType(method))
	}
	cs, err := s.channel.NewStream(ctx, streamDesc(method), requestMethod(method), opts...)
	if err != nil {
		return nil, err
	}
	return &BidiStream{baseStream{cs}, method.Input(), method.Output(), s.resolver}, nil
}

func streamDesc(md protoreflect.MethodDescriptor) *grpc.StreamDesc {
	return &grpc.StreamDesc{
		StreamName:    string(md.Name()),
		ServerStreams: md.IsStreamingServer(),
		ClientStreams: md.IsStreamingClient(),
	}
}

func methodType(md protoreflect.MethodDescriptor) string {
	switch {
	case md.IsStreamingClient() && md.IsStreamingServer():
		return "bidi-streaming"
	case md.IsStreamingClient():
		return "client-streaming"
	case md.IsStreamingServer():
		return "server-streaming"
	default:
		return "unary"
	}
}

func checkMessageType(md protoreflect.MessageDescriptor, msg proto.Message) error {
	if msg == nil {
		return fmt.Errorf("expecting message of type %s; got nil", md.FullName())
	}
	typeName := msg.ProtoReflect().Descriptor().FullName()
	if typeName != md.FullName() {
		return fmt.Errorf("expecting message of type %s; got %s", md.FullName(), typeName)
	}
	return nil
}

// baseStream provides the behavior shared by all of the dynamic stream types.
type baseStream struct {
	stream grpc.ClientStream
}

// Header returns any header metadata sent by the server (blocks if necessary until headers are
// received).
func (s baseStream) Header() (metadata.MD, error) {
	return s.stream.Header()
}

// Trailer returns the trailer metadata sent by the server. It must only be called after
// the stream has finished: after RecvMsg returns a non-nil error (which may be EOF for
// normal completion of stream) or, for client-streaming calls, after CloseAndReceive
// returns.
func (s baseStream) Trailer() metadata.MD {
	return s.stream.Trailer()
}

// Context returns the context associated with this streaming operation.
func (s baseStream) Context() context.Context {
	return s.stream.Context()
}

// recvMessage receives the next message from the stream, as a message of the
// given type. If a resolver is given, it is used to re-parse unrecognized fields,
// which may be extensions.
func (s baseStream) recvMessage(md protoreflect.MessageDescriptor, resolver protoresolve.SerializationResolver) (proto.Message, error) {
	msg := newMessage(md, resolver)
	if err := s.stream.RecvMsg(msg); err != nil {
		return nil, err
	}
	if resolver != nil {
		protomessage.ReparseUnrecognized(msg, resolver)
	}
	return msg, nil
}

// ServerStream represents a response stream from a server. Messages in the stream can be queried
// as can header and trailer metadata sent by the server.
type ServerStream struct {
	baseStream
	respType protoreflect.MessageDescriptor
	resolver protoresolve.SerializationResolver
	// Cancels the stream's context, to release its resources once the
	// stream is finished.
	cancel context.CancelFunc
}

// RecvMsg returns the next message in the response stream or an error. If the stream
// has completed normally, the error is io.EOF. Otherwise, the error indicates the
// nature of the abnormal termination of the stream.
func (s *ServerStream) RecvMsg() (proto.Message, error) {
	resp, err := s.recvMessage(s.respType, s.resolver)
	if err != nil {
		// The stream is finished.
		s.cancel()
		return nil, err
	}
	return resp, nil
}

// ClientStream represents the client side of a client-streaming call. Messages can be sent
// and, when done, the unary server message and header and trailer metadata can be queried.
type ClientStream struct {
	baseStream
	method   protoreflect.MethodDescriptor
	resolver protoresolve.SerializationResolver
	// Cancels the stream's context, to release its resources once the
	// stream is finished.
	cancel context.CancelFunc
}

// SendMsg sends a request message to the server. If it returns io.EOF, the
// server has already ended the call; use CloseAndReceive to get the call's
// status.
func (s *ClientStream) SendMsg(m proto.Message) error {
	if err := checkMessageType(s.method.Input(), m); err != nil {
		return err
	}
	return s.stream.SendMsg(m)
}

// CloseAndReceive closes the outgoing request stream and then blocks for the server's response.
func (s *ClientStream) CloseAndReceive() (proto.Message, error) {
	// The stream is finished when this returns.
	defer s.cancel()
	if err := s.stream.CloseSend(); err != nil {
		return nil, err
	}
	resp, err := s.recvMessage(s.method.Output(), s.resolver)
	if err != nil {
		return nil, err
	}

	// make sure we get EOF for a second message
	if err := s.stream.RecvMsg(resp.ProtoReflect().New().Interface()); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("client-streaming method %q returned more than one response message", s.method.FullName())
		}
		return nil, err
	}
	return resp, nil
}

// BidiStream represents a bi-directional stream for sending messages to and receiving
// messages from a server. The header and trailer metadata sent by the server can also be
// queried.
type BidiStream struct {
	baseStream
	reqType  protoreflect.MessageDescriptor
	respType protoreflect.MessageDescriptor
	resolver protoresolve.SerializationResolver
}

// SendMsg sends a request message to the server. If it returns io.EOF, the
// server has already ended the call; use RecvMsg to get the call's status.
func (s *BidiStream) SendMsg(m proto.Message) error {
	if err := checkMessageType(s.reqType, m); err != nil {
		return err
	}
	return s.stream.SendMsg(m)
}

// CloseSend indicates the request stream has ended. Invoke this after all request messages
// are sent (even if there are zero such messages).
func (s *BidiStream) CloseSend() error {
	return s.stream.CloseSend()
}

// RecvMsg returns the next message in the response stream or an error. If the stream
// has completed normally, the error is io.EOF. Otherwise, the error indicates the
// nature of the abnormal termination of the stream.
func (s *BidiStream) RecvMsg() (proto.Message, error) {
	return s.recvMessage(s.respType, s.resolver)
}

func newMessage(md protoreflect.MessageDescriptor, resolver protoresolve.SerializationResolver) proto.Message {
	if resolver == nil {
		resolver = protoregistry.GlobalTypes
	}
	msgType, err := resolver.FindMessageByName(md.FullName())
	if err == nil {
		return msgType.New().Interface()
	}
	return dynamicpb.NewMessage(md)
}
