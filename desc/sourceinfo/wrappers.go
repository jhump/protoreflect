package sourceinfo

import (
	"github.com/jhump/protoreflect/v2/sourceinfo"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// WrapFile wraps the given file descriptor so that it will include source
// code info that was registered with this package if the given file was
// processed with protoc-gen-gosrcinfo. Returns fd without wrapping if fd
// already contains source code info.
func WrapFile(fd protoreflect.FileDescriptor) protoreflect.FileDescriptor {
	return sourceinfo.WrapFile(fd)
}

// WrapMessage wraps the given message descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns md without wrapping if md's
// parent file already contains source code info.
func WrapMessage(md protoreflect.MessageDescriptor) protoreflect.MessageDescriptor {
	return sourceinfo.WrapMessage(md)
}

// WrapEnum wraps the given enum descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns ed without wrapping if ed's
// parent file already contains source code info.
func WrapEnum(ed protoreflect.EnumDescriptor) protoreflect.EnumDescriptor {
	return sourceinfo.WrapEnum(ed)
}

// WrapService wraps the given service descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns sd without wrapping if sd's
// parent file already contains source code info.
func WrapService(sd protoreflect.ServiceDescriptor) protoreflect.ServiceDescriptor {
	return sourceinfo.WrapService(sd)
}

// WrapExtensionType wraps the given extension type so that its associated
// descriptor will include source code info that was registered with this package
// if the file it is defined in was processed with protoc-gen-gosrcinfo. Returns
// xt without wrapping if the parent file of xt's descriptor already contains
// source code info.
func WrapExtensionType(xt protoreflect.ExtensionType) protoreflect.ExtensionType {
	return sourceinfo.WrapExtensionType(xt)
}

// WrapMessageType wraps the given message type so that its associated
// descriptor will include source code info that was registered with this package
// if the file it is defined in was processed with protoc-gen-gosrcinfo. Returns
// mt without wrapping if the parent file of mt's descriptor already contains
// source code info.
func WrapMessageType(mt protoreflect.MessageType) protoreflect.MessageType {
	return sourceinfo.WrapMessageType(mt)
}
