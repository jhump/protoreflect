package testutil

import (
	"io"

	"golang.org/x/net/context"
	"google.golang.org/grpc/test/grpc_testing"
)

// TestService is a very simple test service that just echos back request payloads
type TestService struct{}

// EmptyCall satisfies the grpc_testing.TestServiceServer interface. It always succeeds.
func (TestService) EmptyCall(context.Context, *grpc_testing.Empty) (*grpc_testing.Empty, error) {
	return &grpc_testing.Empty{}, nil
}

// UnaryCall satisfies the grpc_testing.TestServiceServer interface. It always succeeds, echoing
// back the payload present in the request.
func (TestService) UnaryCall(_ context.Context, req *grpc_testing.SimpleRequest) (*grpc_testing.SimpleResponse, error) {
	return &grpc_testing.SimpleResponse{
		Payload: req.Payload,
	}, nil
}

// StreamingOutputCall satisfies the grpc_testing.TestServiceServer interface. It only fails if the
// client cancels or disconnects (thus causing ss.Send to return an error). It echoes a number of
// responses equal to the request's number of response parameters. The requested parameter details,
// however, ignored. The response payload is always an echo of the request payload.
func (TestService) StreamingOutputCall(req *grpc_testing.StreamingOutputCallRequest, ss grpc_testing.TestService_StreamingOutputCallServer) error {
	for i := 0; i < len(req.GetResponseParameters()); i++ {
		ss.Send(&grpc_testing.StreamingOutputCallResponse{
			Payload: req.Payload,
		})
	}
	return nil
}

// StreamingInputCall satisfies the grpc_testing.TestServiceServer interface. It always succeeds,
// sending back the total observed size of all request payloads.
func (TestService) StreamingInputCall(ss grpc_testing.TestService_StreamingInputCallServer) error {
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
	return ss.SendAndClose(&grpc_testing.StreamingInputCallResponse{
		AggregatedPayloadSize: int32(sz),
	})
}

// FullDuplexCall satisfies the grpc_testing.TestServiceServer interface. It only fails if the
// client cancels or disconnects (thus causing ss.Send to return an error). For each request
// message it receives, it sends back a response message with the same payload.
func (TestService) FullDuplexCall(ss grpc_testing.TestService_FullDuplexCallServer) error {
	for {
		req, err := ss.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = ss.Send(&grpc_testing.StreamingOutputCallResponse{
			Payload: req.Payload,
		})
		if err != nil {
			return err
		}
	}
}

// HalfDuplexCall satisfies the grpc_testing.TestServiceServer interface. It only fails if the
// client cancels or disconnects (thus causing ss.Send to return an error). For each request
// message it receives, it sends back a response message with the same payload. But since it is
// half-duplex, all of the request payloads are buffered and responses will only be sent after
// the request stream is half-closed.
func (TestService) HalfDuplexCall(ss grpc_testing.TestService_HalfDuplexCallServer) error {
	var data []*grpc_testing.Payload
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
		err := ss.Send(&grpc_testing.StreamingOutputCallResponse{
			Payload: d,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
