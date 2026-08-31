package grpcreflect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	refv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// TestFileByFilenameCyclicImports verifies that a server which serves files
// with cyclic imports results in an error instead of infinite recursion (and
// a fatal stack overflow) in the client.
func TestFileByFilenameCyclicImports(t *testing.T) {
	t.Parallel()

	imports := map[string][]string{
		// simplest possible cycle: a file that imports itself
		"self.proto": {"self.proto"},
		// mutual recursion between two files
		"mutual1.proto": {"mutual2.proto"},
		"mutual2.proto": {"mutual1.proto"},
		// a longer cycle, entered from a file that is not part of it
		"entry.proto": {"ring1.proto"},
		"ring1.proto": {"ring2.proto"},
		"ring2.proto": {"ring3.proto"},
		"ring3.proto": {"ring1.proto"},
	}
	cconn := startFakeReflectionServer(t, fakeReflectionServer{
		fileFor: func(filename string) *descriptorpb.FileDescriptorProto {
			deps, ok := imports[filename]
			if !ok {
				return nil
			}
			return newFileProto(filename, deps...)
		},
	})

	testCases := []struct {
		name           string
		expectedSuffix string
	}{
		{
			name:           "self.proto",
			expectedSuffix: "cyclic import path: self.proto -> self.proto",
		},
		{
			name:           "mutual1.proto",
			expectedSuffix: "cyclic import path: mutual1.proto -> mutual2.proto -> mutual1.proto",
		},
		{
			// the cycle does not include the file that was requested, and the
			// reported path starts at the first file in the cycle, not at the
			// requested file
			name:           "entry.proto",
			expectedSuffix: "cyclic import path: ring1.proto -> ring2.proto -> ring3.proto -> ring1.proto",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// use a fresh Client for each case so that cached state from a
			// previous case can't mask the recursion
			refClient := NewClientV1(context.Background(), refv1.NewServerReflectionClient(cconn))
			defer refClient.Reset()

			_, err := refClient.FileByFilename(testCase.name)
			require.Error(t, err, "expected error resolving %s", testCase.name)
			require.True(t, strings.HasSuffix(err.Error(), testCase.expectedSuffix),
				"expected error to end with %q; got: %v", testCase.expectedSuffix, err)
		})
	}
}

// TestFileByFilenameImportDepthLimit verifies the client's limit on how deep an
// import graph it will crawl. Unlike the cyclic case above, the graph here is a
// well-formed (acyclic) chain, so nothing but the depth limit stops the client
// from recursing until it overflows the stack.
func TestFileByFilenameImportDepthLimit(t *testing.T) {
	t.Parallel()

	// The server synthesizes an arbitrarily long chain on demand: "1.proto"
	// imports "2.proto", which imports "3.proto", and so on. The last file in
	// the chain imports nothing.
	newClientWithChain := func(t *testing.T, chainLen int) *Client {
		cconn := startFakeReflectionServer(t, fakeReflectionServer{
			fileFor: func(filename string) *descriptorpb.FileDescriptorProto {
				num, err := strconv.Atoi(strings.TrimSuffix(filename, ".proto"))
				if err != nil || num < 1 || num > chainLen {
					return nil
				}
				if num == chainLen {
					return newFileProto(filename)
				}
				return newFileProto(filename, fmt.Sprintf("%d.proto", num+1))
			},
		})
		refClient := NewClientV1(context.Background(), refv1.NewServerReflectionClient(cconn))
		t.Cleanup(refClient.Reset)
		return refClient
	}

	t.Run("at limit", func(t *testing.T) {
		t.Parallel()
		refClient := newClientWithChain(t, maxImportDepth)
		fd, err := refClient.FileByFilename("1.proto")
		require.NoError(t, err, "failed to resolve chain of %d files", maxImportDepth)
		require.Equal(t, "1.proto", fd.Path())
		// make sure the whole chain really did get resolved
		require.Equal(t, maxImportDepth, chainLength(fd))
	})

	t.Run("past limit", func(t *testing.T) {
		t.Parallel()
		refClient := newClientWithChain(t, maxImportDepth+1)
		_, err := refClient.FileByFilename("1.proto")
		require.Error(t, err, "expected error resolving chain of %d files", maxImportDepth+1)
		expected := fmt.Sprintf("file 1.proto has import tree depth > %d", maxImportDepth)
		require.True(t, strings.HasSuffix(err.Error(), expected),
			"expected error to end with %q; got: %v", expected, err)
	})
}

// fakeReflectionServer is a reflection server whose responses are entirely
// synthesized by a callback. It is used to model a malicious (or just broken)
// server that hands back file descriptors the client must defend itself
// against. It only answers "file by filename" requests; everything else is
// reported as not found.
type fakeReflectionServer struct {
	// fileFor returns the descriptor to serve for the given filename, or nil
	// to report that the file is not found.
	fileFor func(filename string) *descriptorpb.FileDescriptorProto
	// onStreamEnd, if non-nil, is called when a stream handler returns.
	onStreamEnd func()
}

func (s fakeReflectionServer) ServerReflectionInfo(stream refv1.ServerReflection_ServerReflectionInfoServer) error {
	if s.onStreamEnd != nil {
		defer s.onStreamEnd()
	}
	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		resp := &refv1.ServerReflectionResponse{OriginalRequest: req}
		var fd *descriptorpb.FileDescriptorProto
		if fileReq, ok := req.MessageRequest.(*refv1.ServerReflectionRequest_FileByFilename); ok {
			fd = s.fileFor(fileReq.FileByFilename)
		}
		if fd == nil {
			resp.MessageResponse = &refv1.ServerReflectionResponse_ErrorResponse{
				ErrorResponse: &refv1.ErrorResponse{
					ErrorCode:    int32(codes.NotFound),
					ErrorMessage: "not found",
				},
			}
		} else {
			data, err := proto.Marshal(fd)
			if err != nil {
				return err
			}
			resp.MessageResponse = &refv1.ServerReflectionResponse_FileDescriptorResponse{
				FileDescriptorResponse: &refv1.FileDescriptorResponse{
					FileDescriptorProto: [][]byte{data},
				},
			}
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
	}
}

// startFakeReflectionServer starts svc on a local port and returns a client
// connection to it. Both are shut down when the test finishes.
func startFakeReflectionServer(t *testing.T, svc fakeReflectionServer) *grpc.ClientConn {
	t.Helper()
	svr := grpc.NewServer()
	refv1.RegisterServerReflectionServer(svr, svc)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err, "failed to listen")
	go func() {
		_ = svr.Serve(listener)
	}()
	t.Cleanup(svr.Stop)

	cconn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err, "failed to create grpc client")
	t.Cleanup(func() {
		_ = cconn.Close()
	})
	return cconn
}

// newFileProto returns a minimal file descriptor with the given name and
// imports. It intentionally declares no elements: the import graph is the only
// thing under test.
func newFileProto(name string, deps ...string) *descriptorpb.FileDescriptorProto {
	return &descriptorpb.FileDescriptorProto{
		Name:       proto.String(name),
		Syntax:     proto.String("proto3"),
		Dependency: deps,
	}
}

// chainLength returns the number of files reachable from fd, assuming each file
// has at most one import.
func chainLength(fd protoreflect.FileDescriptor) int {
	count := 0
	for fd != nil {
		count++
		imports := fd.Imports()
		if imports.Len() == 0 {
			break
		}
		fd = imports.Get(0).FileDescriptor
	}
	return count
}
