package resolvertest

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// corpusIndex is an index of all elements in a corpus of files. It is
// used to compute the expected results of queries against a resolver.
type corpusIndex struct {
	paths map[string]struct{}
	// All messages, including map entries.
	messages   []protoreflect.MessageDescriptor
	enums      []protoreflect.EnumDescriptor
	extensions []protoreflect.ExtensionDescriptor
	// Normal (non-extension) fields.
	fields              []protoreflect.FieldDescriptor
	extensionsByMessage map[protoreflect.FullName][]protoreflect.ExtensionDescriptor
}

func newCorpusIndex(corpus []protoreflect.FileDescriptor) *corpusIndex {
	index := &corpusIndex{
		paths:               make(map[string]struct{}, len(corpus)),
		extensionsByMessage: map[protoreflect.FullName][]protoreflect.ExtensionDescriptor{},
	}
	for _, file := range corpus {
		index.paths[file.Path()] = struct{}{}
		index.addTypes(file)
	}
	return index
}

func (index *corpusIndex) addTypes(container protoresolve.TypeContainer) {
	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		msg := msgs.Get(i)
		index.messages = append(index.messages, msg)
		fields := msg.Fields()
		for j, numFields := 0, fields.Len(); j < numFields; j++ {
			index.fields = append(index.fields, fields.Get(j))
		}
		index.addTypes(msg)
	}
	enums := container.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		index.enums = append(index.enums, enums.Get(i))
	}
	exts := container.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		ext := exts.Get(i)
		index.extensions = append(index.extensions, ext)
		extendee := ext.ContainingMessage().FullName()
		index.extensionsByMessage[extendee] = append(index.extensionsByMessage[extendee], ext)
	}
}

// typeMessages returns the messages that should be resolvable as types.
// Map entries are excluded since generated code does not register types
// for them.
func (index *corpusIndex) typeMessages() []protoreflect.MessageDescriptor {
	msgs := make([]protoreflect.MessageDescriptor, 0, len(index.messages))
	for _, msg := range index.messages {
		if !msg.IsMapEntry() {
			msgs = append(msgs, msg)
		}
	}
	return msgs
}

func (index *corpusIndex) inCorpus(d protoreflect.Descriptor) bool {
	_, ok := index.paths[d.ParentFile().Path()]
	return ok
}

func transitiveClosure(roots []protoreflect.FileDescriptor) []protoreflect.FileDescriptor {
	var files []protoreflect.FileDescriptor
	seen := map[string]struct{}{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if _, ok := seen[file.Path()]; ok {
			return
		}
		seen[file.Path()] = struct{}{}
		imports := file.Imports()
		for i, length := 0, imports.Len(); i < length; i++ {
			add(imports.Get(i).FileDescriptor)
		}
		files = append(files, file)
	}
	for _, root := range roots {
		add(root)
	}
	return files
}
