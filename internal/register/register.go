package register

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// RegisterTypesVisibleToFile registers types (extensions and messages) that are
// visible to the given file. This includes types defined in file as well as types
// defined in the files that it imports (and any public imports thereof, etc).
func RegisterTypesVisibleToFile(file protoreflect.FileDescriptor, reg *protoregistry.Types, includeMessages bool) {
	registerTypes(file, reg, includeMessages)
	imports := file.Imports()
	for i, length := 0, imports.Len(); i < length; i++ {
		dep := imports.Get(i).FileDescriptor
		RegisterTypesInImportedFile(dep, reg, includeMessages)
	}
}

// RegisterTypesInImportedFile registers types (extensions and messages) in the
// given file as well as those in its public imports. So if another file imports
// the given file, this adds all types made visible to that importing file.
func RegisterTypesInImportedFile(file protoreflect.FileDescriptor, reg *protoregistry.Types, includeMessages bool) {
	registerTypes(file, reg, includeMessages)
	imports := file.Imports()
	for i, length := 0, imports.Len(); i < length; i++ {
		dep := file.Imports().Get(i)
		if dep.IsPublic {
			RegisterTypesInImportedFile(dep.FileDescriptor, reg, includeMessages)
		}
	}
}

type typeContainer interface {
	Messages() protoreflect.MessageDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

// NB: This is similar to the unexported function of the same name in protoresolve, but
// this version is best effort and ignores all errors, and it doesn't register enums.
func registerTypes(container typeContainer, reg *protoregistry.Types, includeMessages bool) {
	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		msg := msgs.Get(i)
		if includeMessages {
			_ = reg.RegisterMessage(dynamicpb.NewMessageType(msg))
		}
		// register nested types
		registerTypes(msg, reg, includeMessages)
	}

	exts := container.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		ext := exts.Get(i)
		_ = reg.RegisterExtension(dynamicpb.NewExtensionType(ext))
	}
}
