package srcinforeflection

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestReflectionService(t *testing.T) {
	svc := grpc.NewServer()
	testprotos.RegisterRpcServiceServer(svc, &testprotos.UnimplementedRpcServiceServer{})
	Register(svc)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	testutil.Ok(t, err)
	go func() {
		if err := svc.Serve(l); err != nil {
			t.Logf("error from gRPC server: %v", err)
		}
	}()
	defer func() {
		_ = svc.Stop
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cc, err := grpc.DialContext(ctx, l.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock())
	testutil.Ok(t, err)
	defer func() {
		_ = cc.Close()
	}()

	stub := rpb.NewServerReflectionClient(cc)
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	cli := grpcreflect.NewClient(ctx, stub)
	defer cli.Reset()

	t.Run("ListServices", func(t *testing.T) {
		svcs, err := cli.ListServices()
		testutil.Ok(t, err)
		testutil.Eq(t, []string{"foo.bar.RpcService", "grpc.reflection.v1alpha.ServerReflection"}, svcs)
	})

	t.Run("FileContainingSymbol", func(t *testing.T) {
		fd, err := cli.FileContainingSymbol("foo.bar.RpcService")
		testutil.Ok(t, err)
		d := fd.FindSymbol("foo.bar.RpcService")
		testutil.Eq(t, " Service comment\n", d.GetSourceInfo().GetLeadingComments())
		md := d.(*desc.ServiceDescriptor).FindMethodByName("StreamingRpc")
		testutil.Eq(t, " Method comment\n", md.GetSourceInfo().GetLeadingComments())
		md = d.(*desc.ServiceDescriptor).FindMethodByName("UnaryRpc")
		testutil.Eq(t, " trailer for method\n", md.GetSourceInfo().GetTrailingComments())
	})

	t.Run("FileByFilename", func(t *testing.T) {
		fd, err := cli.FileByFilename("desc_test1.proto")
		testutil.Ok(t, err)
		checkFileComments(t, fd)
	})

	t.Run("FileContainingExtension", func(t *testing.T) {
		fd, err := cli.FileContainingExtension("testprotos.AnotherTestMessage", 100)
		testutil.Ok(t, err)
		testutil.Eq(t, "desc_test1.proto", fd.GetName())
		checkFileComments(t, fd)
	})

	t.Run("AllExtensionsByType", func(t *testing.T) {
		nums, err := cli.AllExtensionNumbersForType("testprotos.AnotherTestMessage")
		testutil.Ok(t, err)
		testutil.Eq(t, []int32{100, 101, 102, 103, 200}, nums)
	})
}

func checkFileComments(t *testing.T, fd *desc.FileDescriptor) {
	for _, md := range fd.GetMessageTypes() {
		checkMessageComments(t, md)
	}
	for _, ed := range fd.GetEnumTypes() {
		checkEnumComments(t, ed)
	}
	for _, exd := range fd.GetExtensions() {
		checkComment(t, exd)
	}
	for _, sd := range fd.GetServices() {
		checkComment(t, sd)
		for _, mtd := range sd.GetMethods() {
			checkComment(t, mtd)
		}
	}
}

func checkMessageComments(t *testing.T, md *desc.MessageDescriptor) {
	checkComment(t, md)

	for _, fld := range md.GetFields() {
		if fld.GetType() == descriptorpb.FieldDescriptorProto_TYPE_GROUP {
			continue // comment is attributed to group message, not field
		}
		checkComment(t, fld)
	}
	for _, od := range md.GetOneOfs() {
		checkComment(t, od)
	}

	for _, nmd := range md.GetNestedMessageTypes() {
		if nmd.IsMapEntry() {
			// synthetic map entry messages won't have comments
			continue
		}
		checkMessageComments(t, nmd)
	}
	for _, ed := range md.GetNestedEnumTypes() {
		checkEnumComments(t, ed)
	}
	for _, exd := range md.GetNestedExtensions() {
		checkComment(t, exd)
	}
}

func checkEnumComments(t *testing.T, ed *desc.EnumDescriptor) {
	checkComment(t, ed)
	for _, evd := range ed.GetValues() {
		checkComment(t, evd)
	}
}

func checkComment(t *testing.T, d desc.Descriptor) {
	cmt := fmt.Sprintf(" Comment for %s\n", d.GetName())
	testutil.Eq(t, cmt, d.GetSourceInfo().GetLeadingComments())
}
