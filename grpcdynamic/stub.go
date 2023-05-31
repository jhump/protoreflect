// Package grpcdynamic provides a dynamic RPC stub. It can be used to invoke RPC
// method where only method descriptors are known. The actual request and response
// messages may be dynamic messages.
package grpcdynamic

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

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
		protoresolve.ReparseUnrecognized(resp, s.resolver)
	}
	return resp, nil
}

// InvokeRpcServerStream sends a unary RPC and returns the response stream. Use this for server-streaming methods.
func (s *Stub) InvokeRpcServerStream(ctx context.Context, method protoreflect.MethodDescriptor, request proto.Message, opts ...grpc.CallOption) (*ServerStream, error) {
	if method.IsStreamingClient() || !method.IsStreamingServer() {
		return nil, fmt.Errorf("InvokeRpcServerStream is for server-streaming methods; %q is %s", method.FullName(), methodType(method))
	}
	if err := checkMessageType(method.Input(), request); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	sd := grpc.StreamDesc{
		StreamName:    string(method.Name()),
		ServerStreams: method.IsStreamingServer(),
		ClientStreams: method.IsStreamingClient(),
	}
	cs, err := s.channel.NewStream(ctx, &sd, requestMethod(method), opts...)
	if err != nil {
		cancel()
		return nil, err
	}
	err = cs.SendMsg(request)
	if err != nil {
		cancel()
		return nil, err
	}
	err = cs.CloseSend()
	if err != nil {
		cancel()
		return nil, err
	}
	go func() {
		// when the new stream is finished, also cleanup the parent context
		<-cs.Context().Done()
		cancel()
	}()
	return &ServerStream{cs, method.Output(), s.resolver}, nil
}

// InvokeRpcClientStream creates a new stream that is used to send request messages and, at the end,
// receive the response message. Use this for client-streaming methods.
func (s *Stub) InvokeRpcClientStream(ctx context.Context, method protoreflect.MethodDescriptor, opts ...grpc.CallOption) (*ClientStream, error) {
	if !method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, fmt.Errorf("InvokeRpcClientStream is for client-streaming methods; %q is %s", method.FullName(), methodType(method))
	}
	ctx, cancel := context.WithCancel(ctx)
	sd := grpc.StreamDesc{
		StreamName:    string(method.Name()),
		ServerStreams: method.IsStreamingServer(),
		ClientStreams: method.IsStreamingClient(),
	}
	cs, err := s.channel.NewStream(ctx, &sd, requestMethod(method), opts...)
	if err != nil {
		cancel()
		return nil, err
	}
	go func() {
		// when the new stream is finished, also cleanup the parent context
		<-cs.Context().Done()
		cancel()
	}()
	return &ClientStream{cs, method, s.resolver, cancel}, nil
}

// InvokeRpcBidiStream creates a new stream that is used to both send request messages and receive response
// messages. Use this for bidi-streaming methods.
func (s *Stub) InvokeRpcBidiStream(ctx context.Context, method protoreflect.MethodDescriptor, opts ...grpc.CallOption) (*BidiStream, error) {
	if !method.IsStreamingClient() || !method.IsStreamingServer() {
		return nil, fmt.Errorf("InvokeRpcBidiStream is for bidi-streaming methods; %q is %s", method.FullName(), methodType(method))
	}
	sd := grpc.StreamDesc{
		StreamName:    string(method.Name()),
		ServerStreams: method.IsStreamingServer(),
		ClientStreams: method.IsStreamingClient(),
	}
	cs, err := s.channel.NewStream(ctx, &sd, requestMethod(method), opts...)
	if err != nil {
		return nil, err
	}
	return &BidiStream{cs, method.Input(), method.Output(), s.resolver}, nil
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
	typeName := msg.ProtoReflect().Descriptor().FullName()
	if typeName != md.FullName() {
		return fmt.Errorf("expecting message of type %s; got %s", md.FullName(), typeName)
	}
	return nil
}

// ServerStream represents a response stream from a server. Messages in the stream can be queried
// as can header and trailer metadata sent by the server.
type ServerStream struct {
	stream   grpc.ClientStream
	respType protoreflect.MessageDescriptor
	resolver protoresolve.SerializationResolver
}

// Header returns any header metadata sent by the server (blocks if necessary until headers are
// received).
func (s *ServerStream) Header() (metadata.MD, error) {
	return s.stream.Header()
}

// Trailer returns the trailer metadata sent by the server. It must only be called after
// RecvMsg returns a non-nil error (which may be EOF for normal completion of stream).
func (s *ServerStream) Trailer() metadata.MD {
	return s.stream.Trailer()
}

// Context returns the context associated with this streaming operation.
func (s *ServerStream) Context() context.Context {
	return s.stream.Context()
}

// RecvMsg returns the next message in the response stream or an error. If the stream
// has completed normally, the error is io.EOF. Otherwise, the error indicates the
// nature of the abnormal termination of the stream.
func (s *ServerStream) RecvMsg() (proto.Message, error) {
	resp := newMessage(s.respType, s.resolver)
	if err := s.stream.RecvMsg(resp); err != nil {
		return nil, err
	}
	if s.resolver != nil {
		protoresolve.ReparseUnrecognized(resp, s.resolver)
	}
	return resp, nil
}

// ClientStream represents a response stream from a client. Messages in the stream can be sent
// and, when done, the unary server message and header and trailer metadata can be queried.
type ClientStream struct {
	stream   grpc.ClientStream
	method   protoreflect.MethodDescriptor
	resolver protoresolve.SerializationResolver
	cancel   context.CancelFunc
}

// Header returns any header metadata sent by the server (blocks if necessary until headers are
// received).
func (s *ClientStream) Header() (metadata.MD, error) {
	return s.stream.Header()
}

// Trailer returns the trailer metadata sent by the server. It must only be called after
// RecvMsg returns a non-nil error (which may be EOF for normal completion of stream).
func (s *ClientStream) Trailer() metadata.MD {
	return s.stream.Trailer()
}

// Context returns the context associated with this streaming operation.
func (s *ClientStream) Context() context.Context {
	return s.stream.Context()
}

// SendMsg sends a request message to the server.
func (s *ClientStream) SendMsg(m proto.Message) error {
	if err := checkMessageType(s.method.Input(), m); err != nil {
		return err
	}
	return s.stream.SendMsg(m)
}

// CloseAndReceive closes the outgoing request stream and then blocks for the server's response.
func (s *ClientStream) CloseAndReceive() (proto.Message, error) {
	if err := s.stream.CloseSend(); err != nil {
		return nil, err
	}
	resp := newMessage(s.method.Output(), s.resolver)
	if err := s.stream.RecvMsg(resp); err != nil {
		return nil, err
	}
	if s.resolver != nil {
		protoresolve.ReparseUnrecognized(resp, s.resolver)
	}

	// make sure we get EOF for a second message
	if err := s.stream.RecvMsg(resp.ProtoReflect().New().Interface()); err != io.EOF {
		if err == nil {
			s.cancel()
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
	stream   grpc.ClientStream
	reqType  protoreflect.MessageDescriptor
	respType protoreflect.MessageDescriptor
	resolver protoresolve.SerializationResolver
}

// Header returns any header metadata sent by the server (blocks if necessary until headers are
// received).
func (s *BidiStream) Header() (metadata.MD, error) {
	return s.stream.Header()
}

// Trailer returns the trailer metadata sent by the server. It must only be called after
// RecvMsg returns a non-nil error (which may be EOF for normal completion of stream).
func (s *BidiStream) Trailer() metadata.MD {
	return s.stream.Trailer()
}

// Context returns the context associated with this streaming operation.
func (s *BidiStream) Context() context.Context {
	return s.stream.Context()
}

// SendMsg sends a request message to the server.
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
	resp := newMessage(s.respType, s.resolver)
	if err := s.stream.RecvMsg(resp); err != nil {
		return nil, err
	}
	if s.resolver != nil {
		protoresolve.ReparseUnrecognized(resp, s.resolver)
	}
	return resp, nil
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
