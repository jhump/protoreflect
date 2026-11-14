package remotereg

import (
	"bytes"
	"maps"
	"slices"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

// fileEntry represents the contents of a single file.
type fileEntry struct {
	types  typeTrie
	deps   map[string]struct{}
	proto3 bool
	// synthetic is true if the file's name was synthesized because the types
	// in it did not indicate the name of the file that defines them.
	synthetic bool
}

// toFileDescriptor converts this file entry into a file descriptor proto. The given function
// is used to transform nodes in a typeTrie into message and/or enum descriptor protos.
func (fe *fileEntry) toFileDescriptor(name string, trieFn func(*typeTrie, string) (proto.Message, error)) (*descriptorpb.FileDescriptorProto, error) {
	pkg, elements := fe.packageAndElements()
	fd := createFileDescriptor(name, pkg, fe.proto3, fe.deps)
	// sort elements for deterministic output
	for _, elemName := range slices.Sorted(maps.Keys(elements)) {
		pm, err := trieFn(elements[elemName], elemName)
		if err != nil {
			return nil, err
		}
		if mdp, ok := pm.(*descriptorpb.DescriptorProto); ok {
			fd.MessageType = append(fd.MessageType, mdp)
		} else if edp, ok := pm.(*descriptorpb.EnumDescriptorProto); ok {
			fd.EnumType = append(fd.EnumType, edp)
		} else {
			sdp := pm.(*descriptorpb.ServiceDescriptorProto)
			fd.Service = append(fd.Service, sdp)
		}
	}
	return fd, nil
}

// packageAndElements computes the package for this file and returns it along
// with the trie nodes for the file's top-level elements, keyed by their simple
// names. Since type definitions do not distinguish a type's package from its
// enclosing messages, the package is the longest name prefix shared by all of
// the file's types. Anything in the trie below the package that is not one of
// the file's types becomes a placeholder message.
func (fe *fileEntry) packageAndElements() (string, map[string]*typeTrie) {
	var pkg bytes.Buffer
	tt := &fe.types
	first := true
	last := ""
	for tt.typ == nil {
		if last != "" {
			if first {
				first = false
			} else {
				pkg.WriteByte('.')
			}
			pkg.WriteString(last)
		}
		if len(tt.children) != 1 {
			break
		}
		for last, tt = range tt.children {
		}
	}
	if tt.typ != nil {
		return pkg.String(), map[string]*typeTrie{last: tt}
	}
	return pkg.String(), tt.children
}

// forEachMessage calls fn for every message in the file, including the
// placeholder messages that will be created for trie nodes that are not types.
// The placeholder argument to fn indicates whether the message is a
// placeholder.
func (fe *fileEntry) forEachMessage(fn func(name string, placeholder bool)) {
	pkg, elements := fe.packageAndElements()
	for elemName, elem := range elements {
		if pkg != "" {
			elemName = pkg + "." + elemName
		}
		elem.forEachMessage(elemName, fn)
	}
}
