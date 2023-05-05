package testing

import (
	"context"
	"io"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/jhump/protoreflect/v2/internal/testdata/grpc"
)

// TestService is a very simple test service that just echos back request payloads
type TestService struct {
	grpc.UnimplementedTestServiceServer
}

// EmptyCall satisfies the grpc.TestServiceServer interface. It always succeeds.
func (TestService) EmptyCall(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// UnaryCall satisfies the grpc.TestServiceServer interface. It always succeeds, echoing
// back the payload present in the request.
func (TestService) UnaryCall(_ context.Context, req *grpc.SimpleRequest) (*grpc.SimpleResponse, error) {
	return &grpc.SimpleResponse{
		Payload: req.Payload,
	}, nil
}

// StreamingOutputCall satisfies the grpc.TestServiceServer interface. It only fails if the
// client cancels or disconnects (thus causing ss.Send to return an error). It echoes a number of
// responses equal to the request's number of response parameters. The requested parameter details,
// however, ignored. The response payload is always an echo of the request payload.
func (TestService) StreamingOutputCall(req *grpc.StreamingOutputCallRequest, ss grpc.TestService_StreamingOutputCallServer) error {
	for i := 0; i < len(req.GetResponseParameters()); i++ {
		err := ss.Send(&grpc.StreamingOutputCallResponse{
			Payload: req.Payload,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// StreamingInputCall satisfies the grpc.TestServiceServer interface. It always succeeds,
// sending back the total observed size of all request payloads.
func (TestService) StreamingInputCall(ss grpc.TestService_StreamingInputCallServer) error {
	sz := 0
	for {
		req, err := ss.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		sz += len(req.Payload.GetBody())
	}
	return ss.SendAndClose(&grpc.StreamingInputCallResponse{
		AggregatedPayloadSize: int32(sz),
	})
}

// FullDuplexCall satisfies the grpc.TestServiceServer interface. It only fails if the
// client cancels or disconnects (thus causing ss.Send to return an error). For each request
// message it receives, it sends back a response message with the same payload.
func (TestService) FullDuplexCall(ss grpc.TestService_FullDuplexCallServer) error {
	for {
		req, err := ss.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = ss.Send(&grpc.StreamingOutputCallResponse{
			Payload: req.Payload,
		})
		if err != nil {
			return err
		}
	}
}

// HalfDuplexCall satisfies the grpc.TestServiceServer interface. It only fails if the
// client cancels or disconnects (thus causing ss.Send to return an error). For each request
// message it receives, it sends back a response message with the same payload. But since it is
// half-duplex, all of the request payloads are buffered and responses will only be sent after
// the request stream is half-closed.
func (TestService) HalfDuplexCall(ss grpc.TestService_HalfDuplexCallServer) error {
	var data []*grpc.Payload
	for {
		req, err := ss.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		data = append(data, req.Payload)
	}

	for _, d := range data {
		err := ss.Send(&grpc.StreamingOutputCallResponse{
			Payload: d,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
