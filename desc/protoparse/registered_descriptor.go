package protoparse

import (
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/internal"
)

// GetRegisteredDescriptors takes a list of proto filenames which have
// already been imported as compiled Go protobufs, and returns a map
// of FileDescriptorProto objects.
func GetRegisteredDescriptors(filenames ...string) (map[string]*dpb.FileDescriptorProto, error) {
	answer := make(map[string]*dpb.FileDescriptorProto)
	for _, filename := range filenames {
		fd, err := internal.LoadFileDescriptor(filename)
		if err != nil {
			return nil, err
		}
		answer[filename] = fd
	}
	return answer, nil
}
