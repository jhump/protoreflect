// Package srcinforeflection provides an implementation of server reflection
// that includes source code info, if the protoc-gen-gosrcinfo plugin was used
// for the files that contain the descriptors being served. This allows for
// sending comment information to dynamic/reflective clients.
package srcinforeflection

import (
	"io"
	"sort"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/desc/sourceinfo"
)

// NB: This was forked from the implementation in google.golang.org/grpc/reflection.
//  However, this implementation is very different (and MUCH more concise) since it
//  uses the v2 API for protobuf. In addition to this difference, it also uses
//  the sourceinfo package to load file descriptors, which will merge in source
//  code information (which contains information about element locations and comments).
type serverReflectionServer struct {
	s reflection.GRPCServer
}

// Register registers the server reflection service on the given gRPC server.
func Register(s reflection.GRPCServer) {
	rpb.RegisterServerReflectionServer(s, &serverReflectionServer{s: s})
}

func (s *serverReflectionServer) listServices() []string {
	svcInfo := s.s.GetServiceInfo()
	svcNames := make([]string, 0, len(svcInfo))
	for n := range svcInfo {
		svcNames = append(svcNames, n)
	}
	sort.Strings(svcNames)
	return svcNames
}

// fileDescWithDependencies returns a slice of serialized fileDescriptors in
// wire format ([]byte). The fileDescriptors will include fd and all the
// transitive dependencies of fd with names not in sentFileDescriptors.
func fileDescWithDependencies(file string, sentFileDescriptors map[string]struct{}) ([][]byte, error) {
	fd, err := sourceinfo.GlobalFiles.FindFileByPath(file)
	if err != nil {
		return nil, err
	}
	var r [][]byte
	queue := []protoreflect.FileDescriptor{fd}
	for len(queue) > 0 {
		currentfd := queue[0]
		queue = queue[1:]
		if _, sent := sentFileDescriptors[currentfd.Path()]; len(r) == 0 || !sent {
			sentFileDescriptors[currentfd.Path()] = struct{}{}
			fdProto := protodesc.ToFileDescriptorProto(fd)
			currentfdEncoded, err := proto.Marshal(fdProto)
			if err != nil {
				return nil, err
			}
			r = append(r, currentfdEncoded)
		}
		for i := 0; i < currentfd.Imports().Len(); i++ {
			queue = append(queue, currentfd.Imports().Get(i))
		}
	}
	return r, nil
}

// fileDescEncodingContainingSymbol finds the file descriptor containing the
// given symbol, finds all of its previously unsent transitive dependencies,
// does marshalling on them, and returns the marshalled result. The given symbol
// can be a type, a service or a method.
func fileDescEncodingContainingSymbol(name string, sentFileDescriptors map[string]struct{}) ([][]byte, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName(name))
	if err != nil {
		return nil, err
	}
	return fileDescWithDependencies(d.ParentFile().Path(), sentFileDescriptors)
}

// fileDescEncodingContainingExtension finds the file descriptor containing
// given extension, finds all of its previously unsent transitive dependencies,
// does marshalling on them, and returns the marshalled result.
func fileDescEncodingContainingExtension(typeName string, extNum int32, sentFileDescriptors map[string]struct{}) ([][]byte, error) {
	xt, err := protoregistry.GlobalTypes.FindExtensionByNumber(protoreflect.FullName(typeName), protoreflect.FieldNumber(extNum))
	if err != nil {
		return nil, err
	}
	return fileDescWithDependencies(xt.TypeDescriptor().ParentFile().Path(), sentFileDescriptors)
}

// allExtensionNumbersForTypeName returns all extension numbers for the given type.
func allExtensionNumbersForTypeName(name string) []int32 {
	var numbers []int32
	protoregistry.GlobalTypes.RangeExtensionsByMessage(protoreflect.FullName(name), func(xt protoreflect.ExtensionType) bool {
		numbers = append(numbers, int32(xt.TypeDescriptor().Number()))
		return true
	})
	sort.Slice(numbers, func(i, j int) bool {
		return numbers[i] < numbers[j]
	})
	return numbers
}

// ServerReflectionInfo is the reflection service handler.
func (s *serverReflectionServer) ServerReflectionInfo(stream rpb.ServerReflection_ServerReflectionInfoServer) error {
	sentFileDescriptors := map[string]struct{}{}
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		out := &rpb.ServerReflectionResponse{
			ValidHost:       in.Host,
			OriginalRequest: in,
		}
		switch req := in.MessageRequest.(type) {
		case *rpb.ServerReflectionRequest_FileByFilename:
			b, err := fileDescWithDependencies(req.FileByFilename, sentFileDescriptors)
			if err != nil {
				out.MessageResponse = &rpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &rpb.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: b},
				}
			}
		case *rpb.ServerReflectionRequest_FileContainingSymbol:
			b, err := fileDescEncodingContainingSymbol(req.FileContainingSymbol, sentFileDescriptors)
			if err != nil {
				out.MessageResponse = &rpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &rpb.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: b},
				}
			}
		case *rpb.ServerReflectionRequest_FileContainingExtension:
			typeName := req.FileContainingExtension.ContainingType
			extNum := req.FileContainingExtension.ExtensionNumber
			b, err := fileDescEncodingContainingExtension(typeName, extNum, sentFileDescriptors)
			if err != nil {
				out.MessageResponse = &rpb.ServerReflectionResponse_ErrorResponse{
					ErrorResponse: &rpb.ErrorResponse{
						ErrorCode:    int32(codes.NotFound),
						ErrorMessage: err.Error(),
					},
				}
			} else {
				out.MessageResponse = &rpb.ServerReflectionResponse_FileDescriptorResponse{
					FileDescriptorResponse: &rpb.FileDescriptorResponse{FileDescriptorProto: b},
				}
			}
		case *rpb.ServerReflectionRequest_AllExtensionNumbersOfType:
			extNums := allExtensionNumbersForTypeName(req.AllExtensionNumbersOfType)
			out.MessageResponse = &rpb.ServerReflectionResponse_AllExtensionNumbersResponse{
				AllExtensionNumbersResponse: &rpb.ExtensionNumberResponse{
					BaseTypeName:    req.AllExtensionNumbersOfType,
					ExtensionNumber: extNums,
				},
			}
		case *rpb.ServerReflectionRequest_ListServices:
			svcNames := s.listServices()
			serviceResponses := make([]*rpb.ServiceResponse, len(svcNames))
			for i, n := range svcNames {
				serviceResponses[i] = &rpb.ServiceResponse{
					Name: n,
				}
			}
			out.MessageResponse = &rpb.ServerReflectionResponse_ListServicesResponse{
				ListServicesResponse: &rpb.ListServiceResponse{
					Service: serviceResponses,
				},
			}
		default:
			return status.Errorf(codes.InvalidArgument, "invalid MessageRequest: %v", in.MessageRequest)
		}

		if err := stream.Send(out); err != nil {
			return err
		}
	}
}
