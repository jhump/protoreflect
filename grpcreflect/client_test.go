package grpcreflect

import (
	"fmt"
	"net"
	"os"
	"sort"
	"sync/atomic"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/jhump/protoreflect/desc/desc_test"
	"github.com/jhump/protoreflect/testutil"
)

var client *Client

func TestMain(m *testing.M) {
	svr := grpc.NewServer()
	desc_test.RegisterTestServiceServer(svr, testService{})
	reflection.Register(svr)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open server socket: %s", err.Error())
		os.Exit(1)
	}

	go svr.Serve(l)
	defer svr.Stop()

	// wait for server to be accepting
	port := l.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	c, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open connection to server: %s", err.Error())
		os.Exit(1)
	}
	c.Close()

	// create grpc client
	cconn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create grpc client: %s", err.Error())
		os.Exit(1)
	}
	defer cconn.Close()

	stub := rpb.NewServerReflectionClient(cconn)
	client = NewClient(context.Background(), stub)
	os.Exit(m.Run())
}

func TestFileByFileName(t *testing.T) {
	fd, err := client.FileByFilename("desc_test1.proto")
	testutil.Ok(t, err)
	// shallow check that the descriptor appears correct and complete
	testutil.Eq(t, "desc_test1.proto", fd.GetName())
	testutil.Eq(t, "desc_test", fd.GetPackage())
	md := fd.GetMessageTypes()[0]
	testutil.Eq(t, "TestMessage", md.GetName())
	md = md.GetNestedMessageTypes()[0]
	testutil.Eq(t, "NestedMessage", md.GetName())
	md = md.GetNestedMessageTypes()[0]
	testutil.Eq(t, "AnotherNestedMessage", md.GetName())
	md = md.GetNestedMessageTypes()[0]
	testutil.Eq(t, "YetAnotherNestedMessage", md.GetName())
	ed := md.GetNestedEnumTypes()[0]
	testutil.Eq(t, "DeeplyNestedEnum", ed.GetName())

	_, err = client.FileByFilename("does not exist")
	testutil.Eq(t, FileOrSymbolNotFound, err)
}

func TestFileContainingSymbol(t *testing.T) {
	fd, err := client.FileContainingSymbol("TopLevel")
	testutil.Ok(t, err)
	// shallow check that the descriptor appears correct and complete
	testutil.Eq(t, "nopkg/desc_test_nopkg_new.proto", fd.GetName())
	testutil.Eq(t, "", fd.GetPackage())
	md := fd.GetMessageTypes()[0]
	testutil.Eq(t, "TopLevel", md.GetName())
	testutil.Eq(t, "i", md.GetFields()[0].GetName())
	testutil.Eq(t, "j", md.GetFields()[1].GetName())
	testutil.Eq(t, "k", md.GetFields()[2].GetName())
	testutil.Eq(t, "l", md.GetFields()[3].GetName())
	testutil.Eq(t, "m", md.GetFields()[4].GetName())
	testutil.Eq(t, "n", md.GetFields()[5].GetName())
	testutil.Eq(t, "o", md.GetFields()[6].GetName())
	testutil.Eq(t, "p", md.GetFields()[7].GetName())
	testutil.Eq(t, "q", md.GetFields()[8].GetName())
	testutil.Eq(t, "r", md.GetFields()[9].GetName())
	testutil.Eq(t, "s", md.GetFields()[10].GetName())
	testutil.Eq(t, "t", md.GetFields()[11].GetName())

	_, err = client.FileContainingSymbol("does not exist")
	testutil.Eq(t, FileOrSymbolNotFound, err)
}

func TestFileContainingExtension(t *testing.T) {
	fd, err := client.FileContainingExtension("TopLevel", 100)
	testutil.Ok(t, err)
	// shallow check that the descriptor appears correct and complete
	testutil.Eq(t, "desc_test2.proto", fd.GetName())
	testutil.Eq(t, "desc_test", fd.GetPackage())
	testutil.Eq(t, 3, len(fd.GetMessageTypes()))
	testutil.Eq(t, "Frobnitz", fd.GetMessageTypes()[0].GetName())
	testutil.Eq(t, "Whatchamacallit", fd.GetMessageTypes()[1].GetName())
	testutil.Eq(t, "Whatzit", fd.GetMessageTypes()[2].GetName())

	testutil.Eq(t, "desc_test1.proto", fd.GetDependencies()[0].GetName())
	testutil.Eq(t, "pkg/desc_test_pkg.proto", fd.GetDependencies()[1].GetName())
	testutil.Eq(t, "nopkg/desc_test_nopkg.proto", fd.GetDependencies()[2].GetName())

	_, err = client.FileContainingExtension("does not exist", 100)
	testutil.Eq(t, FileOrSymbolNotFound, err)
	_, err = client.FileContainingExtension("TopLevel", -9)
	testutil.Eq(t, FileOrSymbolNotFound, err)
}

func TestAllExtensionNumbersForType(t *testing.T) {
	nums, err := client.AllExtensionNumbersForType("TopLevel")
	testutil.Ok(t, err)
	testutil.Eq(t, 1, len(nums))
	testutil.Eq(t, 100, int(nums[0]))

	nums, err = client.AllExtensionNumbersForType("desc_test.AnotherTestMessage")
	testutil.Ok(t, err)
	testutil.Eq(t, 5, len(nums))
	inums := make([]int, len(nums))
	for idx, v := range nums {
		inums[idx] = int(v)
	}
	sort.Ints(inums)
	testutil.Eq(t, 100, inums[0])
	testutil.Eq(t, 101, inums[1])
	testutil.Eq(t, 102, inums[2])
	testutil.Eq(t, 103, inums[3])
	testutil.Eq(t, 200, inums[4])

	_, err = client.AllExtensionNumbersForType("does not exist")
	testutil.Eq(t, FileOrSymbolNotFound, err)
}

func TestListServices(t *testing.T) {
	s, err := client.ListServices()
	testutil.Ok(t, err)

	sort.Strings(s)
	testutil.Eq(t, "desc_test.TestService", s[0])
	testutil.Eq(t, "grpc.reflection.v1alpha.ServerReflection", s[1])
}

func TestReset(t *testing.T) {
	_, err := client.ListServices()
	testutil.Ok(t, err)

	// save the current stream
	stream := client.stream
	// intercept cancellation
	cancel := client.cancel
	var cancelled int32
	client.cancel = func() {
		atomic.StoreInt32(&cancelled, 1)
		cancel()
	}

	client.Reset()
	testutil.Eq(t, int32(1), atomic.LoadInt32(&cancelled))
	testutil.Eq(t, nil, client.stream)

	_, err = client.ListServices()
	testutil.Ok(t, err)

	// stream was re-created
	testutil.Eq(t, true, client.stream != nil && client.stream != stream)
}

func TestRecover(t *testing.T) {
	_, err := client.ListServices()
	testutil.Ok(t, err)

	// kill the stream
	stream := client.stream
	client.stream.CloseSend()

	// it should auto-recover and re-create stream
	_, err = client.ListServices()
	testutil.Ok(t, err)
	testutil.Eq(t, true, client.stream != nil && client.stream != stream)
}

