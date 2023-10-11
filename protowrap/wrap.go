package protowrap

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/wrappers"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

// FromFileDescriptorProto is identical to [protodesc.NewFile] except that it
// returns a FileWrapper, not just a [protoreflect.FileDescriptor].
func FromFileDescriptorProto(fd *descriptorpb.FileDescriptorProto, deps protoresolve.DependencyResolver) (FileWrapper, error) {
	file, err := protodesc.NewFile(fd, deps)
	if err != nil {
		return nil, err
	}
	return wrappers.WrapFile(file, fd), nil
}

// AddToRegistry converts the given proto to a FileWrapper, using reg to resolve
// any imports, and also registers the wrapper with reg.
func AddToRegistry(fd *descriptorpb.FileDescriptorProto, reg protoresolve.DescriptorRegistry) (FileWrapper, error) {
	file, err := FromFileDescriptorProto(fd, reg)
	if err != nil {
		return nil, err
	}
	if err := reg.RegisterFile(file); err != nil {
		return nil, err
	}
	return file, nil
}

// FromFileDescriptorSet is identical to [protodesc.NewFiles] except that all
// descriptors registered with the returned resolver will be FileWrapper instances.
func FromFileDescriptorSet(files *descriptorpb.FileDescriptorSet) (protoresolve.Resolver, error) {
	protosByPath := map[string]*descriptorpb.FileDescriptorProto{}
	for _, fd := range files.File {
		if _, ok := protosByPath[fd.GetName()]; ok {
			return nil, fmt.Errorf("file %q appears in set more than once", fd.GetName())
		}
		protosByPath[fd.GetName()] = fd
	}
	reg := &protoresolve.Registry{}
	for _, fd := range files.File {
		if err := resolveFile(fd, protosByPath, reg); err != nil {
			return nil, err
		}
	}
	return reg, nil
}

func resolveFile(fd *descriptorpb.FileDescriptorProto, protosByPath map[string]*descriptorpb.FileDescriptorProto, reg *protoresolve.Registry) error {
	if _, err := reg.FindFileByPath(fd.GetName()); err == nil {
		// already resolved
		return nil
	}
	// resolve all dependencies
	for _, dep := range fd.GetDependency() {
		depFile := protosByPath[dep]
		if depFile == nil {
			return fmt.Errorf("set is missing file %q (imported by %q)", dep, fd.GetName())
		}
		if err := resolveFile(depFile, protosByPath, reg); err != nil {
			return err
		}
	}
	_, err := AddToRegistry(fd, reg)
	return err
}
