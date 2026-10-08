package sort

import (
	"fmt"

	"google.golang.org/protobuf/types/descriptorpb"
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
	states := make(map[string]*fileState, len(files))
	for _, file := range files {
		if _, exists := states[file.GetName()]; exists {
			return fmt.Errorf("duplicate file %q", file.GetName())
		}
		states[file.GetName()] = &fileState{file: file}
	}
	sorted := make([]*descriptorpb.FileDescriptorProto, 0, len(files))
	for _, file := range files {
		if err := addFileSorted(states[file.GetName()], states, &sorted); err != nil {
			return err
		}
	}
	copy(files, sorted)
	return nil
}

func addFileSorted(state *fileState, states map[string]*fileState, sorted *[]*descriptorpb.FileDescriptorProto) error {
	switch state.status {
	case fileStatusAdded:
		return nil
	case fileStatusVisiting:
		return fmt.Errorf("file %q is part of an import cycle", state.file.GetName())
	}
	state.status = fileStatusVisiting
	for _, dep := range state.file.GetDependency() {
		depState := states[dep]
		if depState == nil {
			return fmt.Errorf("file %q imports %q, but %q is not present", state.file.GetName(), dep, dep)
		}
		if err := addFileSorted(depState, states, sorted); err != nil {
			return err
		}
	}
	state.status = fileStatusAdded
	*sorted = append(*sorted, state.file)
	return nil
}

// fileStatus is the state of a file in the sort. The zero value means the
// file has not been visited yet.
type fileStatus int

const (
	// The file's imports are being added.
	fileStatusVisiting = fileStatus(iota + 1)
	// The file and its imports have been added.
	fileStatusAdded
)

type fileState struct {
	file   *descriptorpb.FileDescriptorProto
	status fileStatus
}
