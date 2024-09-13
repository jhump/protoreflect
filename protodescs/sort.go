package protodescs

import (
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/sort"
)

// SortFiles topologically sorts the given file descriptor protos. It returns
// an error if the given files include duplicates (more than one entry with the
// same path) or if any of the files refer to imports which are not present in
// the given files.
func SortFiles(files []*descriptorpb.FileDescriptorProto) error {
	return sort.SortFiles(files)
}
