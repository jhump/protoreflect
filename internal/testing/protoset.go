package testing

import (
	"io"
	"os"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// LoadProtoset loads the compiled protoset file at the given path. It returns
// the last file descriptor in the set. When generating a protoset for a single
// file, that file is always last (and its dependencies before it).
func LoadProtoset(path string) (protoreflect.FileDescriptor, error) {
	var fds descriptorpb.FileDescriptorSet
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	bb, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(bb, &fds); err != nil {
		return nil, err
	}
	res, err := protodesc.NewFiles(&fds)
	if err != nil {
		return nil, err
	}
	// return the last file in the set
	return res.FindFileByPath(fds.File[len(fds.File)-1].GetName())
}
