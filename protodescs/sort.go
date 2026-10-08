package protodescs

import (
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/sort"
)

// SortFiles topologically sorts the given file descriptor protos, so that
// every file appears after all of its imports. The result depends only on the
// order of the given files.
//
// It returns an error if the given files include duplicates (more than one
// entry with the same path), if any of the files refer to imports which are
// not present in the given files, or if the imports form a cycle. When it
// returns an error, the given slice is unchanged.
func SortFiles(files []*descriptorpb.FileDescriptorProto) error {
	return sort.SortFiles(files)
}
