package grpcreflect

import (
	"testing"

	"google.golang.org/grpc"

	testprotosgrpc "github.com/jhump/protoreflect/internal/testprotos/grpc"
	"github.com/jhump/protoreflect/internal/testutil"
)

type testService struct {
	testprotosgrpc.DummyServiceServer
}

func TestLoadServiceDescriptors(t *testing.T) {
	s := grpc.NewServer()
	testprotosgrpc.RegisterDummyServiceServer(s, testService{})
	sds, err := LoadServiceDescriptors(s)
	testutil.Ok(t, err)
	testutil.Eq(t, 1, len(sds))
	sd := sds["testprotos.DummyService"]

	cases := []struct{ method, request, response string }{
		{"DoSomething", "testprotos.DummyRequest", "jhump.protoreflect.desc.Bar"},
		{"DoSomethingElse", "testprotos.TestMessage", "testprotos.DummyResponse"},
		{"DoSomethingAgain", "jhump.protoreflect.desc.Bar", "testprotos.AnotherTestMessage"},
		{"DoSomethingForever", "testprotos.DummyRequest", "testprotos.DummyResponse"},
	}

	testutil.Eq(t, len(cases), len(sd.GetMethods()))

	for i, c := range cases {
		md := sd.GetMethods()[i]
		testutil.Eq(t, c.method, md.GetName())
		testutil.Eq(t, c.request, md.GetInputType().GetFullyQualifiedName())
		testutil.Eq(t, c.response, md.GetOutputType().GetFullyQualifiedName())
	}
}

func TestLoadServiceDescriptor(t *testing.T) {
	sd, err := LoadServiceDescriptor(&testprotosgrpc.DummyService_ServiceDesc)
	testutil.Ok(t, err)

	cases := []struct{ method, request, response string }{
		{"DoSomething", "testprotos.DummyRequest", "jhump.protoreflect.desc.Bar"},
		{"DoSomethingElse", "testprotos.TestMessage", "testprotos.DummyResponse"},
		{"DoSomethingAgain", "jhump.protoreflect.desc.Bar", "testprotos.AnotherTestMessage"},
		{"DoSomethingForever", "testprotos.DummyRequest", "testprotos.DummyResponse"},
	}

	testutil.Eq(t, len(cases), len(sd.GetMethods()))

	for i, c := range cases {
		md := sd.GetMethods()[i]
		testutil.Eq(t, c.method, md.GetName())
		testutil.Eq(t, c.request, md.GetInputType().GetFullyQualifiedName())
		testutil.Eq(t, c.response, md.GetOutputType().GetFullyQualifiedName())
	}
}
