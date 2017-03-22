package grpcreflect

import (
	"testing"

	"google.golang.org/grpc"

	"github.com/jhump/protoreflect/desc/desc_test"
	"github.com/jhump/protoreflect/testutil"
)

type testService struct {
	desc_test.TestServiceServer
}

func TestLoadServiceDescriptors(t *testing.T) {
	s := grpc.NewServer()
	desc_test.RegisterTestServiceServer(s, testService{})
	sds, err := LoadServiceDescriptors(s)
	testutil.Ok(t, err)
	testutil.Eq(t, 1, len(sds))
	sd := sds["desc_test.TestService"]

	cases := []struct{ method, request, response string }{
		{"DoSomething", "desc_test.TestRequest", "jhump.protoreflect.desc.Bar" },
		{"DoSomethingElse", "desc_test.TestMessage", "desc_test.TestResponse" },
		{"DoSomethingAgain", "jhump.protoreflect.desc.Bar", "desc_test.AnotherTestMessage" },
		{"DoSomethingForever", "desc_test.TestRequest", "desc_test.TestResponse" },
	}

	testutil.Eq(t, len(cases), len(sd.GetMethods()))

	for i, c := range cases {
		md := sd.GetMethods()[i]
		testutil.Eq(t, c.method, md.GetName())
		testutil.Eq(t, c.request, md.GetInputType().GetFullyQualifiedName())
		testutil.Eq(t, c.response, md.GetOutputType().GetFullyQualifiedName())
	}
}
