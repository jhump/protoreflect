package grpcreflect

//lint:file-ignore SA1019 The refv1alpha package is deprecated, but we need it in order to adapt it to new version

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	refv1alpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	_ "google.golang.org/protobuf/types/known/apipb"
	_ "google.golang.org/protobuf/types/known/emptypb"
	_ "google.golang.org/protobuf/types/known/fieldmaskpb"
	_ "google.golang.org/protobuf/types/known/sourcecontextpb"
	_ "google.golang.org/protobuf/types/known/typepb"
	_ "google.golang.org/protobuf/types/pluginpb"

	testprotosgrpc "github.com/jhump/protoreflect/v2/internal/testdata/grpc"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

var clientv1, clientv1alpha *Client

func TestMain(m *testing.M) {
	code := 1
	defer func() {
		p := recover()
		if p != nil {
			_, _ = fmt.Fprintf(os.Stderr, "PANIC: %v\n", p)
		}
		os.Exit(code)
	}()

	svr := grpc.NewServer()
	testprotosgrpc.RegisterDummyServiceServer(svr, testService{})
	// support both v1 and v1alpha
	reflection.Register(svr)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("Failed to open server socket: %s", err.Error()))
	}
	go func() {
		_ = svr.Serve(l)
	}()
	defer svr.Stop()

	// create grpc client
	addr := l.Addr().String()
	cconn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to create grpc client: %s", err.Error()))
	}
	defer func() {
		_ = cconn.Close()
	}()

	stubv1alpha := refv1alpha.NewServerReflectionClient(cconn)
	clientv1alpha = NewClientV1Alpha(context.Background(), stubv1alpha)
	stubv1 := refv1.NewServerReflectionClient(cconn)
	clientv1 = NewClientV1(context.Background(), stubv1)

	code = m.Run()
}

func testVersions(t *testing.T, fn func(*testing.T, *Client)) {
	t.Run("v1", func(t *testing.T) {
		fn(t, clientv1)
	})
	t.Run("v1alpha", func(t *testing.T) {
		fn(t, clientv1alpha)
	})
}

func TestFileByFileName(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		fd, err := client.FileByFilename("desc_test1.proto")
		require.NoError(t, err)
		// shallow check that the descriptor appears correct and complete
		require.Equal(t, "desc_test1.proto", fd.Path())
		require.Equal(t, protoreflect.FullName("testprotos"), fd.Package())
		md := fd.Messages().Get(0)
		require.Equal(t, protoreflect.Name("TestMessage"), md.Name())
		md = md.Messages().Get(0)
		require.Equal(t, protoreflect.Name("NestedMessage"), md.Name())
		md = md.Messages().Get(0)
		require.Equal(t, protoreflect.Name("AnotherNestedMessage"), md.Name())
		md = md.Messages().Get(0)
		require.Equal(t, protoreflect.Name("YetAnotherNestedMessage"), md.Name())
		ed := md.Enums().Get(0)
		require.Equal(t, protoreflect.Name("DeeplyNestedEnum"), ed.Name())

		_, err = client.FileByFilename("does not exist")
		require.True(t, IsElementNotFoundError(err))
	})
}

func TestFileByFileNameForWellKnownProtos(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		wellKnownProtos := map[string][]protoreflect.FullName{
			"google/protobuf/any.proto":             {"google.protobuf.Any"},
			"google/protobuf/api.proto":             {"google.protobuf.Api", "google.protobuf.Method", "google.protobuf.Mixin"},
			"google/protobuf/descriptor.proto":      {"google.protobuf.FileDescriptorSet", "google.protobuf.DescriptorProto"},
			"google/protobuf/duration.proto":        {"google.protobuf.Duration"},
			"google/protobuf/empty.proto":           {"google.protobuf.Empty"},
			"google/protobuf/field_mask.proto":      {"google.protobuf.FieldMask"},
			"google/protobuf/source_context.proto":  {"google.protobuf.SourceContext"},
			"google/protobuf/struct.proto":          {"google.protobuf.Struct", "google.protobuf.Value", "google.protobuf.NullValue"},
			"google/protobuf/timestamp.proto":       {"google.protobuf.Timestamp"},
			"google/protobuf/type.proto":            {"google.protobuf.Type", "google.protobuf.Field", "google.protobuf.Syntax"},
			"google/protobuf/wrappers.proto":        {"google.protobuf.DoubleValue", "google.protobuf.Int32Value", "google.protobuf.StringValue"},
			"google/protobuf/compiler/plugin.proto": {"google.protobuf.compiler.CodeGeneratorRequest"},
		}

		for file, types := range wellKnownProtos {
			fd, err := client.FileByFilename(file)
			require.NoError(t, err)
			require.Equal(t, file, fd.Path())
			for _, typ := range types {
				d := protoresolve.FindDescriptorByNameInFile(fd, typ)
				require.NotNil(t, d)
			}
		}
	})
}

func TestFileContainingSymbol(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		fd, err := client.FileContainingSymbol("TopLevel")
		require.NoError(t, err)
		// shallow check that the descriptor appears correct and complete
		require.Equal(t, "nopkg/desc_test_nopkg_new.proto", fd.Path())
		require.Equal(t, protoreflect.FullName(""), fd.Package())
		md := fd.Messages().Get(0)
		require.Equal(t, protoreflect.Name("TopLevel"), md.Name())
		require.Equal(t, protoreflect.Name("i"), md.Fields().Get(0).Name())
		require.Equal(t, protoreflect.Name("j"), md.Fields().Get(1).Name())
		require.Equal(t, protoreflect.Name("k"), md.Fields().Get(2).Name())
		require.Equal(t, protoreflect.Name("l"), md.Fields().Get(3).Name())
		require.Equal(t, protoreflect.Name("m"), md.Fields().Get(4).Name())
		require.Equal(t, protoreflect.Name("n"), md.Fields().Get(5).Name())
		require.Equal(t, protoreflect.Name("o"), md.Fields().Get(6).Name())
		require.Equal(t, protoreflect.Name("p"), md.Fields().Get(7).Name())
		require.Equal(t, protoreflect.Name("q"), md.Fields().Get(8).Name())
		require.Equal(t, protoreflect.Name("r"), md.Fields().Get(9).Name())
		require.Equal(t, protoreflect.Name("s"), md.Fields().Get(10).Name())
		require.Equal(t, protoreflect.Name("t"), md.Fields().Get(11).Name())

		_, err = client.FileContainingSymbol("does not exist")
		require.True(t, IsElementNotFoundError(err))
	})
}

func TestFileContainingExtension(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		fd, err := client.FileContainingExtension("TopLevel", 100)
		require.NoError(t, err)
		// shallow check that the descriptor appears correct and complete
		require.Equal(t, "desc_test2.proto", fd.Path())
		require.Equal(t, protoreflect.FullName("testprotos"), fd.Package())
		require.Equal(t, 4, fd.Messages().Len())
		require.Equal(t, protoreflect.Name("Frobnitz"), fd.Messages().Get(0).Name())
		require.Equal(t, protoreflect.Name("Whatchamacallit"), fd.Messages().Get(1).Name())
		require.Equal(t, protoreflect.Name("Whatzit"), fd.Messages().Get(2).Name())
		require.Equal(t, protoreflect.Name("GroupX"), fd.Messages().Get(3).Name())

		require.Equal(t, "desc_test1.proto", fd.Imports().Get(0).Path())
		require.Equal(t, "pkg/desc_test_pkg.proto", fd.Imports().Get(1).Path())
		require.Equal(t, "nopkg/desc_test_nopkg.proto", fd.Imports().Get(2).Path())

		_, err = client.FileContainingExtension("does not exist", 100)
		require.True(t, IsElementNotFoundError(err))
		_, err = client.FileContainingExtension("TopLevel", -9)
		require.True(t, IsElementNotFoundError(err))
	})
}

func TestAllExtensionNumbersForType(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		nums, err := client.AllExtensionNumbersForType("TopLevel")
		require.NoError(t, err)
		inums := make([]int, len(nums))
		for idx, v := range nums {
			inums[idx] = int(v)
		}
		sort.Ints(inums)
		require.Equal(t, []int{100, 104}, inums)

		nums, err = client.AllExtensionNumbersForType("testprotos.AnotherTestMessage")
		require.NoError(t, err)
		require.Equal(t, 5, len(nums))
		inums = make([]int, len(nums))
		for idx, v := range nums {
			inums[idx] = int(v)
		}
		sort.Ints(inums)
		require.Equal(t, []int{100, 101, 102, 103, 200}, inums)

		nums, err = client.AllExtensionNumbersForType("does not exist")
		require.NoError(t, err)
		require.Empty(t, nums)
	})
}

func TestListServices(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		s, err := client.ListServices()
		require.NoError(t, err)

		sort.Slice(s, func(i, j int) bool {
			return s[i] < s[j]
		})
		require.Equal(t, []protoreflect.FullName{
			"grpc.reflection.v1.ServerReflection",
			"grpc.reflection.v1alpha.ServerReflection",
			"testprotos.DummyService",
		}, s)
	})
}

func TestReset(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		_, err := client.ListServices()
		require.NoError(t, err)

		// save the current stream
		stream := client.stream
		// intercept cancellation
		cancel := client.cancel
		var cancelled atomic.Bool
		client.cancel = func() {
			cancelled.Store(true)
			cancel()
		}

		client.Reset()
		require.True(t, cancelled.Load())
		require.Nil(t, nil, client.stream)

		_, err = client.ListServices()
		require.NoError(t, err)

		// stream was re-created
		require.Equal(t, true, client.stream != nil && client.stream != stream)
	})
}

func TestRecover(t *testing.T) {
	testVersions(t, func(t *testing.T, client *Client) {
		_, err := client.ListServices()
		require.NoError(t, err)

		// kill the stream
		stream := client.stream
		err = client.stream.CloseSend()
		require.NoError(t, err)

		// it should auto-recover and re-create stream
		_, err = client.ListServices()
		require.NoError(t, err)
		require.Equal(t, true, client.stream != nil && client.stream != stream)
	})
}

func TestMultipleFiles(t *testing.T) {
	svr := grpc.NewServer()
	refv1alpha.RegisterServerReflectionServer(svr, testReflectionServer{})

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer cancel()
		if err := svr.Serve(l); err != nil {
			t.Logf("serve returned error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // give server a chance to start
	require.NoError(t, ctx.Err(), "failed to start server")
	defer func() {
		svr.Stop()
	}()

	cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "failed to dial %v", l.Addr().String())
	cl := refv1alpha.NewServerReflectionClient(cc)

	client := NewClientV1Alpha(ctx, cl)
	defer client.Reset()
	svcs, err := client.ListServices()
	require.NoError(t, err, "failed to list services")
	for _, svc := range svcs {
		fd, err := client.FileContainingSymbol(svc)
		require.NoError(t, err, "failed to get file for service %v", svc)
		sd := fd.Services().ByName(svc.Name())
		require.NotNil(t, sd)
		require.Equal(t, svc, sd.FullName())
	}
}

func TestAllowMissingFileDescriptors(t *testing.T) {
	svr := grpc.NewServer()
	files := createFilesWithMissingDeps(t)
	reflectionSvc := reflection.NewServer(reflection.ServerOptions{
		DescriptorResolver: files,
		ExtensionResolver:  files,
	})
	refv1alpha.RegisterServerReflectionServer(svr, reflectionSvc)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer cancel()
		if err := svr.Serve(l); err != nil {
			t.Logf("serve returned error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // give server a chance to start
	require.NoError(t, ctx.Err(), "failed to start server")
	defer func() {
		svr.Stop()
	}()

	cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "failed to dial %v", l.Addr().String())
	cl := refv1alpha.NewServerReflectionClient(cc)

	client := NewClientV1Alpha(ctx, cl)
	defer client.Reset()

	// First we try some things that should fail due to missing descriptors.
	_, err = client.FileByFilename("foo/bar/this.proto")
	require.Error(t, err)
	_, err = client.FileContainingSymbol("foo.bar.Bar")
	require.Error(t, err)
	_, err = client.FileContainingExtension("google.protobuf.MessageOptions", 10101)
	require.Error(t, err)

	client = NewClientV1Alpha(ctx, cl, WithAllowMissingFileDescriptors())
	// Now the above queries should succeed.
	file, err := client.FileByFilename("foo/bar/this.proto")
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "foo/bar/this.proto", file.Path())
	file, err = client.FileContainingSymbol("foo.bar.Bar")
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "foo/bar/this.proto", file.Path())
	file, err = client.FileContainingExtension("google.protobuf.MessageOptions", 10101)
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "test/imported.proto", file.Path())
}

func TestAllowFallbackResolver(t *testing.T) {
	svr := grpc.NewServer()
	reflection.RegisterV1(svr)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer cancel()
		if err := svr.Serve(l); err != nil {
			t.Logf("serve returned error: %v", err)
		}
	}()
	time.Sleep(100 * time.Millisecond) // give server a chance to start
	require.NoError(t, ctx.Err(), "failed to start server")
	defer func() {
		svr.Stop()
	}()

	cc, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "failed to dial %v", l.Addr().String())
	cl := refv1.NewServerReflectionClient(cc)

	client := NewClientV1(ctx, cl)
	defer client.Reset()

	// First sanity-check that the well-known types are there.
	file, err := client.FileByFilename("google/protobuf/descriptor.proto")
	require.NoError(t, err)
	require.Equal(t, "google/protobuf/descriptor.proto", file.Path())
	// Now we try some things that should fail due to missing descriptors.
	_, err = client.FileByFilename("foo/bar/this.proto")
	require.Error(t, err)
	_, err = client.FileContainingSymbol("foo.bar.Bar")
	require.Error(t, err)
	_, err = client.FileContainingExtension("google.protobuf.MessageOptions", 23456)
	require.Error(t, err)
	nums, err := client.AllExtensionNumbersForType("google.protobuf.MessageOptions")
	require.NoError(t, err)
	withoutFallbackExts := len(nums)

	// Now we configure a fallback.
	fdp := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("foo/bar/this.proto"),
		Package:    proto.String("foo.bar"),
		Dependency: []string{"google/protobuf/descriptor.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Bar"),
			},
		},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Name:     proto.String("opt"),
				Extendee: proto.String(".google.protobuf.MessageOptions"),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
				TypeName: proto.String(".foo.bar.Bar"),
				Number:   proto.Int32(23456),
			},
		},
	}
	fd, err := protodesc.NewFile(fdp, protoregistry.GlobalFiles)
	require.NoError(t, err)
	var files files
	err = files.RegisterFile(fd)
	require.NoError(t, err)

	client = NewClientV1(ctx, cl, WithFallbackResolvers(&files, &files))

	// The above queries should now succeed.
	file, err = client.FileByFilename("foo/bar/this.proto")
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "foo/bar/this.proto", file.Path())
	file, err = client.FileContainingSymbol("foo.bar.Bar")
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "foo/bar/this.proto", file.Path())
	file, err = client.FileContainingExtension("google.protobuf.MessageOptions", 23456)
	require.NoError(t, err)
	require.NotNil(t, file)
	require.Equal(t, "foo/bar/this.proto", file.Path())
	nums, err = client.AllExtensionNumbersForType("google.protobuf.MessageOptions")
	require.NoError(t, err)
	// The same extensions as before, plus an extra one provided by the fallback.
	require.Len(t, nums, withoutFallbackExts+1)
}

func TestFileWithoutDeps(t *testing.T) {
	fd := &descriptorpb.FileDescriptorProto{
		Dependency: []string{
			"foo/bar.proto",
			"foo/public/bar.proto", // missing
			"foo/weak/bar.proto",
			"foo/baz.proto", // missing
			"foo/public/baz.proto",
			"foo/weak/baz.proto", // missing
			"foo/fizz.proto",
			"foo/public/fizz.proto", // missing
			"foo/weak/fizz.proto",
			"foo/buzz.proto", // missing
			"foo/public/buzz.proto",
			"foo/weak/buzz.proto", // missing
		},
		PublicDependency: []int32{1, 4, 7, 10},
		WeakDependency:   []int32{2, 5, 8, 11},
	}
	fd = fileWithoutDeps(fd, []int{1, 3, 5, 7, 9, 11})
	require.Equal(t,
		[]string{
			"foo/bar.proto",
			"foo/weak/bar.proto",
			"foo/public/baz.proto",
			"foo/fizz.proto",
			"foo/weak/fizz.proto",
			"foo/public/buzz.proto",
		},
		fd.Dependency)
	require.Equal(t, []int32{2, 5}, fd.PublicDependency)
	require.Equal(t, []int32{1, 4}, fd.WeakDependency)
}

type testReflectionServer struct{}

func (t testReflectionServer) ServerReflectionInfo(server refv1alpha.ServerReflection_ServerReflectionInfoServer) error {
	const svcAfile = "ChdzYW5kYm94L3NlcnZpY2VfQS5wcm90bxIHc2FuZGJveCIWCghSZXF1ZXN0QRIKCgJpZBgBIAEoBSIYCglSZXNwb25zZUESCwoDc3RyGAEgASgJMj0KCVNlcnZpY2VfQRIwCgdFeGVjdXRlEhEuc2FuZGJveC5SZXF1ZXN0QRoSLnNhbmRib3guUmVzcG9uc2VBYgZwcm90bzM="
	const svcBfile = "ChdzYW5kYm94L1NlcnZpY2VfQi5wcm90bxIHc2FuZGJveCIWCghSZXF1ZXN0QhIKCgJpZBgBIAEoBSIYCglSZXNwb25zZUISCwoDc3RyGAEgASgJMj0KCVNlcnZpY2VfQhIwCgdFeGVjdXRlEhEuc2FuZGJveC5SZXF1ZXN0QhoSLnNhbmRib3guUmVzcG9uc2VCYgZwcm90bzM="

	for {
		req, err := server.Recv()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}
		var resp refv1alpha.ServerReflectionResponse
		resp.OriginalRequest = req
		switch req := req.MessageRequest.(type) {
		case *refv1alpha.ServerReflectionRequest_FileByFilename:
			switch req.FileByFilename {
			case "sandbox/service_A.proto":
				resp.MessageResponse = msgResponseForFiles(svcAfile)
			case "sandbox/service_B.proto":
				resp.MessageResponse = msgResponseForFiles(svcBfile)
			default:
				resp.MessageResponse = &refv1alpha.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &refv1alpha.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: "not found",
					},
				}
			}
		case *refv1alpha.ServerReflectionRequest_FileContainingSymbol:
			switch req.FileContainingSymbol {
			case "sandbox.Service_A":
				resp.MessageResponse = msgResponseForFiles(svcAfile)
			case "sandbox.Service_B":
				// HERE is where we return two files instead of one
				resp.MessageResponse = msgResponseForFiles(svcAfile, svcBfile)
			default:
				resp.MessageResponse = &refv1alpha.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &refv1alpha.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: "not found",
					},
				}
			}
		case *refv1alpha.ServerReflectionRequest_ListServices:
			resp.MessageResponse = &refv1alpha.ServerReflectionResponse_ListServicesResponse{
				ListServicesResponse: &refv1alpha.ListServiceResponse{
					Service: []*refv1alpha.ServiceResponse{
						{Name: "sandbox.Service_A"},
						{Name: "sandbox.Service_B"},
					},
				},
			}
		default:
			resp.MessageResponse = &refv1alpha.ServerReflectionResponse_ErrorResponse{
				ErrorResponse: &refv1alpha.ErrorResponse{
					ErrorCode:    int32(codes.NotFound),
					ErrorMessage: "not found",
				},
			}
		}
		if err := server.Send(&resp); err != nil {
			return err
		}
	}
}

func msgResponseForFiles(files ...string) *refv1alpha.ServerReflectionResponse_FileDescriptorResponse {
	descs := make([][]byte, len(files))
	for i, f := range files {
		b, err := base64.StdEncoding.DecodeString(f)
		if err != nil {
			panic(err)
		}
		descs[i] = b
	}
	return &refv1alpha.ServerReflectionResponse_FileDescriptorResponse{
		FileDescriptorResponse: &refv1alpha.FileDescriptorResponse{
			FileDescriptorProto: descs,
		},
	}
}

func TestAutoVersion(t *testing.T) {
	t.Run("v1", func(t *testing.T) {
		testClientAuto(t,
			func(s *grpc.Server) {
				reflection.RegisterV1(s) // this one just uses v1
				testprotosgrpc.RegisterDummyServiceServer(s, testService{})
			},
			[]protoreflect.FullName{
				"grpc.reflection.v1.ServerReflection",
				"testprotos.DummyService",
			},
			[]string{
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
			})
	})

	t.Run("v1alpha", func(t *testing.T) {
		testClientAuto(t,
			func(s *grpc.Server) {
				// this one just uses v1alpha
				refv1alpha.RegisterServerReflectionServer(s, reflection.NewServer(reflection.ServerOptions{Services: s}))
				testprotosgrpc.RegisterDummyServiceServer(s, testService{})
			},
			[]protoreflect.FullName{
				"grpc.reflection.v1alpha.ServerReflection",
				"testprotos.DummyService",
			},
			[]string{
				// first one fails, so falls back to v1alpha
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
				// next two use v1alpha
				"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
				// final one retries v1
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
			})
	})

	t.Run("both", func(t *testing.T) {
		testClientAuto(t,
			func(s *grpc.Server) {
				reflection.Register(s) // this registers both
				testprotosgrpc.RegisterDummyServiceServer(s, testService{})
			},
			[]protoreflect.FullName{
				"grpc.reflection.v1.ServerReflection",
				"grpc.reflection.v1alpha.ServerReflection",
				"testprotos.DummyService",
			},
			[]string{
				// never uses v1alpha since v1 works
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
				"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
			})
	})

	t.Run("fallback-on-unavailable", testClientAutoOnUnavailable)
}

func testClientAuto(t *testing.T, register func(*grpc.Server), expectedServices []protoreflect.FullName, expectedLog []string) {
	var capture captureStreamNames
	svr := grpc.NewServer(grpc.StreamInterceptor(capture.intercept), grpc.UnknownServiceHandler(capture.handleUnknown))
	register(svr)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("Failed to open server socket: %s", err.Error()))
	}
	go func() {
		err := svr.Serve(l)
		require.NoError(t, err)
	}()
	defer svr.Stop()

	cconn, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("Failed to create grpc client: %s", err.Error()))
	}
	defer func() {
		err := cconn.Close()
		require.NoError(t, err)
	}()
	client := NewClientAuto(context.Background(), cconn)
	now := time.Now()
	client.now = func() time.Time {
		return now
	}

	svcs, err := client.ListServices()
	require.NoError(t, err)
	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i] < svcs[j]
	})
	require.Equal(t, expectedServices, svcs)
	client.Reset()

	_, err = client.FileContainingSymbol(svcs[0])
	require.NoError(t, err)
	client.Reset()

	// at the threshold, but not quite enough to retry
	now = now.Add(time.Hour)
	_, err = client.ListServices()
	require.NoError(t, err)
	client.Reset()

	// 1 ns more, and we've crossed threshold and will retry
	now = now.Add(1)
	_, err = client.ListServices()
	require.NoError(t, err)
	client.Reset()

	actualLog := capture.names()
	require.Equal(t, expectedLog, actualLog)
}

type captureStreamNames struct {
	mu  sync.Mutex
	log []string
}

func (c *captureStreamNames) names() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := make([]string, len(c.log))
	copy(ret, c.log)
	return ret
}

func (c *captureStreamNames) intercept(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	c.mu.Lock()
	c.log = append(c.log, info.FullMethod)
	c.mu.Unlock()
	return handler(srv, ss)
}

func (c *captureStreamNames) handleUnknown(_ interface{}, _ grpc.ServerStream) error {
	return status.Errorf(codes.Unimplemented, "WTF?")
}

func testClientAutoOnUnavailable(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(fmt.Sprintf("Failed to open server socket: %s", err.Error()))
	}
	captureConn := &captureListener{Listener: l}

	var capture captureStreamNames
	svr := grpc.NewServer(
		grpc.StreamInterceptor(capture.intercept),
		grpc.UnknownServiceHandler(func(_ interface{}, _ grpc.ServerStream) error {
			// On unknown method, forcibly close the net.Conn, without sending
			// back any reply, which should result in an "unavailable" error.
			return captureConn.latest().Close()
		}),
	)
	impl := reflection.NewServer(reflection.ServerOptions{Services: svr})
	refv1alpha.RegisterServerReflectionServer(svr, impl)
	testprotosgrpc.RegisterDummyServiceServer(svr, testService{})

	go func() {
		err := svr.Serve(captureConn)
		require.NoError(t, err)
	}()
	defer svr.Stop()

	var captureErrs captureErrors
	cconn, err := grpc.NewClient(
		l.Addr().String(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStreamInterceptor(captureErrs.intercept),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to create grpc client: %s", err.Error()))
	}
	defer func() {
		err := cconn.Close()
		require.NoError(t, err)
	}()
	client := NewClientAuto(context.Background(), cconn)
	now := time.Now()
	client.now = func() time.Time {
		return now
	}

	svcs, err := client.ListServices()
	require.NoError(t, err)
	sort.Slice(svcs, func(i, j int) bool {
		return svcs[i] < svcs[j]
	})
	require.Equal(t, []protoreflect.FullName{
		"grpc.reflection.v1alpha.ServerReflection",
		"testprotos.DummyService",
	}, svcs)

	// It should have tried v1 first and failed then tried v1alpha.
	actualLog := capture.names()
	require.Equal(t, []string{
		"/grpc.reflection.v1.ServerReflection/ServerReflectionInfo",
		"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
	}, actualLog)

	// Make sure the error code observed by the client was unavailable and not unimplemented.
	actualCodes := captureErrs.codes()
	require.Equal(t, []codes.Code{codes.Unavailable}, actualCodes)
}

type captureListener struct {
	net.Listener
	mu   sync.Mutex
	conn net.Conn
}

func (c *captureListener) Accept() (net.Conn, error) {
	conn, err := c.Listener.Accept()
	if err == nil {
		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
	}
	return conn, err
}

func (c *captureListener) latest() net.Conn {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn
}

type captureErrors struct {
	mu       sync.Mutex
	observed []codes.Code
}

func (c *captureErrors) intercept(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	stream, err := streamer(ctx, desc, cc, method, opts...)
	if err != nil {
		c.observe(err)
		return nil, err
	}
	return &captureErrorStream{ClientStream: stream, c: c}, nil
}

func (c *captureErrors) observe(err error) {
	c.mu.Lock()
	c.observed = append(c.observed, status.Code(err))
	c.mu.Unlock()
}

func (c *captureErrors) codes() []codes.Code {
	c.mu.Lock()
	defer c.mu.Unlock()
	ret := make([]codes.Code, len(c.observed))
	copy(ret, c.observed)
	return ret
}

type captureErrorStream struct {
	grpc.ClientStream
	c    *captureErrors
	done int32
}

func (c *captureErrorStream) RecvMsg(m interface{}) error {
	err := c.ClientStream.RecvMsg(m)
	if err == nil || errors.Is(err, io.EOF) {
		return nil
	}
	// Only record one error per RPC.
	if atomic.CompareAndSwapInt32(&c.done, 0, 1) {
		c.c.observe(err)
	}
	return err
}

func createFilesWithMissingDeps(t *testing.T) *files {
	t.Helper()
	var result files
	empty, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:   proto.String("empty.proto"),
		Syntax: proto.String("proto2"),
	}, &result)
	require.NoError(t, err)

	// These will be missing, so we create them as placeholders, so
	// the protobuf-go runtime can resolve imports for them and
	// still build a protoreflect.FileDescriptor.
	err = result.RegisterFile(&placeholder{path: "test/custom/options.proto", FileDescriptor: empty})
	require.NoError(t, err)
	err = result.RegisterFile(&placeholder{path: "test/unused.proto", FileDescriptor: empty})
	require.NoError(t, err)

	// register google/protobuf/descriptor.proto from the embedded descriptor in descriptorpb
	err = result.RegisterFile((*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().ParentFile())
	require.NoError(t, err)

	importedFile := &descriptorpb.FileDescriptorProto{
		Name:             proto.String("test/imported.proto"),
		Syntax:           proto.String("proto3"),
		Package:          proto.String("test"),
		Dependency:       []string{"google/protobuf/descriptor.proto", "test/unused.proto"},
		PublicDependency: []int32{1}, // unused is public
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Message"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("name"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
						JsonName: proto.String("name"),
					},
					{
						Name:     proto.String("tags"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_UINT64.Enum(),
						JsonName: proto.String("tags"),
					},
				},
				Extension: []*descriptorpb.FieldDescriptorProto{
					{
						Extendee: proto.String(".google.protobuf.MessageOptions"),
						Name:     proto.String("message_option"),
						Number:   proto.Int32(10101),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: proto.String("Enum"),
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{
						Name:   proto.String("VAL0"),
						Number: proto.Int32(0),
					},
					{
						Name:   proto.String("VAL1"),
						Number: proto.Int32(1),
					},
				},
			},
		},
		Extension: []*descriptorpb.FieldDescriptorProto{
			{
				Extendee: proto.String(".google.protobuf.FileOptions"),
				Name:     proto.String("file_option"),
				Number:   proto.Int32(10101),
				Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			},
		},
	}
	importedFileDesc, err := protodesc.NewFile(importedFile, &result)
	require.NoError(t, err)
	err = result.Files.RegisterFile(importedFileDesc)
	require.NoError(t, err)

	topFile := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("foo/bar/this.proto"),
		Syntax:     proto.String("proto3"),
		Package:    proto.String("foo.bar"),
		Dependency: []string{"test/imported.proto", "test/unused.proto", "test/custom/options.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Foo"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("msg"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".test.Message"),
						JsonName: proto.String("msg"),
					},
					{
						Name:     proto.String("en"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".test.Enum"),
						JsonName: proto.String("en"),
					},
				},
			},
			{
				Name: proto.String("Bar"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("foos"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".foo.bar.Foo"),
						JsonName: proto.String("foos"),
					},
				},
			},
		},
	}
	topFileDesc, err := protodesc.NewFile(topFile, &result)
	require.NoError(t, err)
	err = result.Files.RegisterFile(topFileDesc)
	require.NoError(t, err)

	return &result
}

type files struct {
	protoregistry.Files
}

func (f *files) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	d, err := f.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	fd, ok := d.(protoreflect.FieldDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a field descriptor but a %T", field, fd)
	}
	if !fd.IsExtension() {
		return nil, fmt.Errorf("%s is a normal field, not an extension", field)
	}
	return asExtensionType(fd), nil
}

func (f *files) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	var found protoreflect.ExtensionType
	f.RangeExtensionsByMessage(message, func(xt protoreflect.ExtensionType) bool {
		if xt.TypeDescriptor().Number() == field {
			found = xt
			return false
		}
		return true
	})
	if found == nil {
		return nil, protoregistry.NotFound
	}
	return found, nil
}

func (f *files) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionType) bool) {
	f.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return rangeExtensionsByMessage(file, message, fn)
	})
}

func rangeExtensionsByMessage(
	container interface {
		Messages() protoreflect.MessageDescriptors
		Extensions() protoreflect.ExtensionDescriptors
	},
	message protoreflect.FullName,
	fn func(protoreflect.ExtensionType) bool,
) bool {
	for i := 0; i < container.Extensions().Len(); i++ {
		ext := container.Extensions().Get(i)
		if ext.ContainingMessage().FullName() == message {
			if !fn(asExtensionType(ext)) {
				return false
			}
		}
	}
	for i := 0; i < container.Messages().Len(); i++ {
		if !rangeExtensionsByMessage(container.Messages().Get(i), message, fn) {
			return false
		}
	}
	return true
}

func asExtensionType(fd protoreflect.ExtensionDescriptor) protoreflect.ExtensionType {
	xtd, ok := fd.(protoreflect.ExtensionTypeDescriptor)
	if ok {
		return xtd.Type()
	}
	return dynamicpb.NewExtensionType(fd)
}

type placeholder struct {
	path string
	protoreflect.FileDescriptor
}

func (p *placeholder) IsPlaceholder() bool {
	return true
}

func (p *placeholder) Path() string {
	return p.path
}

func (p *placeholder) Syntax() protoreflect.Syntax {
	return 0
}
