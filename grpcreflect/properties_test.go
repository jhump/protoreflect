package grpcreflect

import (
	"testing"
	"google.golang.org/grpc"
	"github.com/jhump/protoreflect/desc/desc_test"
)

type testService struct {
	desc_test.TestServiceServer
}

func TestLoadServiceDescriptors(t *testing.T) {
	s := grpc.NewServer()
	desc_test.RegisterTestServiceServer(s, testService{})
	sds, err := LoadServiceDescriptors(s)
	ok(t, err)
	eq(t, 1, len(sds))
	sd := sds["desc_test.TestService"]

	cases := []struct{ method, request, response string }{
		{"DoSomething", "desc_test.TestRequest", "jhump.protoreflect.desc.Bar" },
		{"DoSomethingElse", "desc_test.TestMessage", "desc_test.TestResponse" },
		{"DoSomethingAgain", "jhump.protoreflect.desc.Bar", "desc_test.AnotherTestMessage" },
		{"DoSomethingForever", "desc_test.TestRequest", "desc_test.TestResponse" },
	}

	eq(t, len(cases), len(sd.GetMethods()))

	for i, c := range cases {
		md := sd.GetMethods()[i]
		eq(t, c.method, md.GetName())
		eq(t, c.request, md.GetInputType().GetFullyQualifiedName())
		eq(t, c.response, md.GetOutputType().GetFullyQualifiedName())
	}
}


func eq(t *testing.T, expected, actual interface{}) bool {
	if expected != actual {
		t.Errorf("Expecting %v, got %v", expected, actual)
		return false
	}
	return true
}

func ok(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Unexpected error: %s", err.Error())
	}
}