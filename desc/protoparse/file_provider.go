package protoparse

import (
	"io"

	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc"
)

type FileProvider interface {
	isFileProvider()
}

func ReadCloserFileProvider(f func(string) (io.ReadCloser, error)) FileProvider {
	return readCloserFileProvider{
		f: f,
	}
}

func FileDescriptorFileProvider(f func(string) (*desc.FileDescriptor, error)) FileProvider {
	return fileDescriptorProtoFileProvider{
		f: fileDescriptorFuncToFileDescriptorProtoFunc(f),
	}
}

func FileDescriptorProtoFileProvider(f func(string) (*dpb.FileDescriptorProto, error)) FileProvider {
	return fileDescriptorProtoFileProvider{
		f: f,
	}
}

type readCloserFileProvider struct {
	f func(string) (io.ReadCloser, error)
}

func (readCloserFileProvider) isFileProvider() {}

type fileDescriptorProtoFileProvider struct {
	f func(string) (*dpb.FileDescriptorProto, error)
}

func (fileDescriptorProtoFileProvider) isFileProvider() {}

func fileDescriptorFuncToFileDescriptorProtoFunc(
	f func(string) (*desc.FileDescriptor, error),
) func(string) (*dpb.FileDescriptorProto, error) {
	return func(path string) (*dpb.FileDescriptorProto, error) {
		fileDescriptor, err := f(path)
		if fileDescriptor != nil {
			return fileDescriptor.AsFileDescriptorProto(), err
		}
		return nil, err
	}
}
