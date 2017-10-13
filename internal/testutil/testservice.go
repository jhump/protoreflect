package testutil

import (
	"io"

	"golang.org/x/net/context"
	"google.golang.org/grpc/test/grpc_testing"
)

// very simple test service that just echos back request payloads
type TestService struct{}

func (_ TestService) EmptyCall(context.Context, *grpc_testing.Empty) (*grpc_testing.Empty, error) {
	return &grpc_testing.Empty{}, nil
}

func (_ TestService) UnaryCall(_ context.Context, req *grpc_testing.SimpleRequest) (*grpc_testing.SimpleResponse, error) {
	return &grpc_testing.SimpleResponse{
		Payload: req.Payload,
	}, nil
}

func (_ TestService) StreamingOutputCall(req *grpc_testing.StreamingOutputCallRequest, ss grpc_testing.TestService_StreamingOutputCallServer) error {
	for i := 0; i < len(req.GetResponseParameters()); i++ {
		ss.Send(&grpc_testing.StreamingOutputCallResponse{
			Payload: req.Payload,
		})
	}
	return nil
}

func (_ TestService) StreamingInputCall(ss grpc_testing.TestService_StreamingInputCallServer) error {
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

func (_ TestService) FullDuplexCall(ss grpc_testing.TestService_FullDuplexCallServer) error {
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

func (_ TestService) HalfDuplexCall(ss grpc_testing.TestService_HalfDuplexCallServer) error {
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
