package grpcreflect

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	testprotosgrpc "github.com/jhump/protoreflect/v2/internal/testdata/grpc"
)

type testService struct {
	testprotosgrpc.DummyServiceServer
}

func TestLoadServiceDescriptors(t *testing.T) {
	s := grpc.NewServer()
	testprotosgrpc.RegisterDummyServiceServer(s, testService{})
	sds, err := LoadServiceDescriptors(s)
	require.NoError(t, err)
	require.Equal(t, 1, len(sds))
	sd := sds["testprotos.DummyService"]
	require.NotNil(t, sd)
	checkServiceDescriptor(t, sd)
}

func TestLoadServiceDescriptor(t *testing.T) {
	sd, err := LoadServiceDescriptor(&testprotosgrpc.DummyService_ServiceDesc)
	require.NoError(t, err)
	checkServiceDescriptor(t, sd)
}

func checkServiceDescriptor(t *testing.T, sd protoreflect.ServiceDescriptor) {
	t.Helper()

	cases := []struct {
		method            protoreflect.Name
		request, response protoreflect.FullName
	}{
		{"DoSomething", "testprotos.DummyRequest", "jhump.protoreflect.desc.Bar"},
		{"DoSomethingElse", "testprotos.TestMessage", "testprotos.DummyResponse"},
		{"DoSomethingAgain", "jhump.protoreflect.desc.Bar", "testprotos.AnotherTestMessage"},
		{"DoSomethingForever", "testprotos.DummyRequest", "testprotos.DummyResponse"},
	}

	require.Equal(t, len(cases), sd.Methods().Len())

	for i, c := range cases {
		md := sd.Methods().Get(i)
		require.Equal(t, c.method, md.Name())
		require.Equal(t, c.request, md.Input().FullName())
		require.Equal(t, c.response, md.Output().FullName())
	}
}
