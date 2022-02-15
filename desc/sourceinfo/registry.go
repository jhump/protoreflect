// Package sourceinfo provides the ability to register and query source code info
// for file descriptors that are compiled into the binary. This data is registered
// by code generated from the protoc-gen-gosrcinfo plugin.
//
// The standard descriptors bundled into the compiled binary are stripped of source
// code info, to reduce binary size and reduce runtime memory footprint. However,
// the source code info can be very handy and worth the size cost when used with
// gRPC services and the server reflection service. Without source code info, the
// descriptors that a client downloads from the reflection service have no comments.
// But the presence of comments, and the ability to show them to humans, can greatly
// improve the utility of user agents that use the reflection service.
//
// So, by using the protoc-gen-gosrcinfo plugin and this package, we can recover the
// source code info and comments that were otherwise stripped by protoc-gen-go.
//
// Also see the "github.com/jhump/protoreflect/desc/srcinfo/srcinforeflection" package
// for an implementation of the gRPC server reflection service that uses this package
// return descriptors with source code info.
package sourceinfo

import (
	"fmt"
	"sync"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// GlobalFiles is a registry of descriptors that include source code info, if the
	// file they belong to were processed with protoc-gen-gosrcinfo.
	//
	// If is mean to serve as a drop-in alternative to protoregistry.GlobalFiles that
	// can include source code info in the returned descriptors.
	GlobalFiles protodesc.Resolver = registry{}

	mu               sync.RWMutex
	sourceInfoByFile = map[string]*descriptorpb.SourceCodeInfo{}
	fileDescriptors  = map[protoreflect.FileDescriptor]protoreflect.FileDescriptor{}
)

// RegisterSourceInfo registers the given source code info for the file descriptor
// with the given path/name.
//
// This is automatically used from generated code if using the protoc-gen-gosrcinfo
// plugin.
func RegisterSourceInfo(file string, srcInfo *descriptorpb.SourceCodeInfo) {
	mu.Lock()
	defer mu.Unlock()
	sourceInfoByFile[file] = srcInfo
}

// SourceInfoForFile queries for any registered source code info for the file
// descriptor with the given path/name. It returns nil if no source code info
// was registered.
func SourceInfoForFile(file string) *descriptorpb.SourceCodeInfo {
	mu.RLock()
	defer mu.RUnlock()
	return sourceInfoByFile[file]
}

func getFile(fd protoreflect.FileDescriptor) protoreflect.FileDescriptor {
	if fd == nil {
		return nil
	}

	mu.RLock()
	result := fileDescriptors[fd]
	mu.RUnlock()

	if result != nil {
		return result
	}

	mu.Lock()
	defer mu.Unlock()
	// double-check, in case it was added to map while upgrading lock
	result = fileDescriptors[fd]
	if result != nil {
		return result
	}

	srcInfo := sourceInfoByFile[fd.Path()]
	if len(srcInfo.GetLocation()) > 0 {
		result = &fileDescriptor{
			FileDescriptor: fd,
			locs: &sourceLocations{
				orig: srcInfo.Location,
			},
		}
	} else {
		// nothing to do; don't bother wrapping
		result = fd
	}
	fileDescriptors[fd] = result
	return result
}

type registry struct{}

var _ protodesc.Resolver = &registry{}

func (r registry) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	fd, err := protoregistry.GlobalFiles.FindFileByPath(path)
	if err != nil {
		return nil, err
	}
	return getFile(fd), nil
}

func (r registry) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	switch d := d.(type) {
	case protoreflect.FileDescriptor:
		return getFile(d), nil
	case protoreflect.MessageDescriptor:
		return messageDescriptor{d}, nil
	case protoreflect.ExtensionTypeDescriptor:
		return extensionDescriptor{d}, nil
	case protoreflect.FieldDescriptor:
		return fieldDescriptor{d}, nil
	case protoreflect.OneofDescriptor:
		return oneOfDescriptor{d}, nil
	case protoreflect.EnumDescriptor:
		return enumDescriptor{d}, nil
	case protoreflect.EnumValueDescriptor:
		return enumValueDescriptor{d}, nil
	case protoreflect.ServiceDescriptor:
		return serviceDescriptor{d}, nil
	case protoreflect.MethodDescriptor:
		return methodDescriptor{d}, nil
	default:
		return nil, fmt.Errorf("unrecognized descriptor type: %T", d)
	}
}
