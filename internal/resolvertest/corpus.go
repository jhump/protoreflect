package resolvertest

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// corpusIndex is an index of all elements in a corpus of files. It is
// used to compute the expected results of queries against a resolver.
type corpusIndex struct {
	files          []protoreflect.FileDescriptor
	paths          map[string]struct{}
	filesByPackage map[protoreflect.FullName][]protoreflect.FileDescriptor
	// All descriptors other than files.
	all []protoreflect.Descriptor
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
		files:               corpus,
		paths:               make(map[string]struct{}, len(corpus)),
		filesByPackage:      map[protoreflect.FullName][]protoreflect.FileDescriptor{},
		extensionsByMessage: map[protoreflect.FullName][]protoreflect.ExtensionDescriptor{},
	}
	for _, file := range corpus {
		index.paths[file.Path()] = struct{}{}
		index.filesByPackage[file.Package()] = append(index.filesByPackage[file.Package()], file)
		index.addElements(file)
		services := file.Services()
		for i, length := 0, services.Len(); i < length; i++ {
			service := services.Get(i)
			index.all = append(index.all, service)
			methods := service.Methods()
			for j, numMethods := 0, methods.Len(); j < numMethods; j++ {
				index.all = append(index.all, methods.Get(j))
			}
		}
	}
	return index
}

func (index *corpusIndex) addElements(container protoresolve.TypeContainer) {
	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		msg := msgs.Get(i)
		index.messages = append(index.messages, msg)
		index.all = append(index.all, msg)
		fields := msg.Fields()
		for j, numFields := 0, fields.Len(); j < numFields; j++ {
			index.fields = append(index.fields, fields.Get(j))
			index.all = append(index.all, fields.Get(j))
		}
		oneofs := msg.Oneofs()
		for j, numOneofs := 0, oneofs.Len(); j < numOneofs; j++ {
			index.all = append(index.all, oneofs.Get(j))
		}
		index.addElements(msg)
	}
	enums := container.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		enum := enums.Get(i)
		index.enums = append(index.enums, enum)
		index.all = append(index.all, enum)
		values := enum.Values()
		for j, numValues := 0, values.Len(); j < numValues; j++ {
			index.all = append(index.all, values.Get(j))
		}
	}
	exts := container.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		ext := exts.Get(i)
		index.extensions = append(index.extensions, ext)
		index.all = append(index.all, ext)
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
