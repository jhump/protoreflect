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
	GlobalFiles protodesc.Resolver = registry{}

	mu               sync.RWMutex
	sourceInfoByFile = map[string]*descriptorpb.SourceCodeInfo{}
	fileDescriptors  = map[protoreflect.FileDescriptor]protoreflect.FileDescriptor{}
)

func RegisterSourceInfo(file string, srcInfo *descriptorpb.SourceCodeInfo) {
	mu.Lock()
	defer mu.Unlock()
	sourceInfoByFile[file] = srcInfo
}

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
