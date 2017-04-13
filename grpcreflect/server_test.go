package grpcreflect

import (
	"testing"

	"google.golang.org/grpc"

	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

type testService struct {
	testprotos.TestServiceServer
}

func TestLoadServiceDescriptors(t *testing.T) {
	s := grpc.NewServer()
	testprotos.RegisterTestServiceServer(s, testService{})
	sds, err := LoadServiceDescriptors(s)
	testutil.Ok(t, err)
	testutil.Eq(t, 1, len(sds))
	sd := sds["testprotos.TestService"]

	cases := []struct{ method, request, response string }{
		{"DoSomething", "testprotos.TestRequest", "jhump.protoreflect.desc.Bar"},
		{"DoSomethingElse", "testprotos.TestMessage", "testprotos.TestResponse"},
		{"DoSomethingAgain", "jhump.protoreflect.desc.Bar", "testprotos.AnotherTestMessage"},
		{"DoSomethingForever", "testprotos.TestRequest", "testprotos.TestResponse"},
	}

	testutil.Eq(t, len(cases), len(sd.GetMethods()))

	for i, c := range cases {
		md := sd.GetMethods()[i]
		testutil.Eq(t, c.method, md.GetName())
		testutil.Eq(t, c.request, md.GetInputType().GetFullyQualifiedName())
		testutil.Eq(t, c.response, md.GetOutputType().GetFullyQualifiedName())
	}
}
