package sourceinfo

import (
	"fmt"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// For back-compat reasons, these methods are named "Wrap". But they don't wrap
// but instead "upgrade", which means they rebuild the descriptors with source
// code info when possible.

// WrapFile wraps the given file descriptor so that it will include source
// code info that was registered with this package if the given file was
// processed with protoc-gen-gosrcinfo. Returns fd without wrapping if fd
// already contains source code info.
func WrapFile(fd protoreflect.FileDescriptor) protoreflect.FileDescriptor {
	result, err := getFile(fd)
	if err != nil {
		panic(err)
	}
	return result
}

// WrapMessage wraps the given message descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns md without wrapping if md's
// parent file already contains source code info.
func WrapMessage(md protoreflect.MessageDescriptor) protoreflect.MessageDescriptor {
	result, err := updateDescriptor(md)
	if err != nil {
		panic(err)
	}
	return result
}

// WrapEnum wraps the given enum descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns ed without wrapping if ed's
// parent file already contains source code info.
func WrapEnum(ed protoreflect.EnumDescriptor) protoreflect.EnumDescriptor {
	result, err := updateDescriptor(ed)
	if err != nil {
		panic(err)
	}
	return result
}

// WrapService wraps the given service descriptor so that it will include source
// code info that was registered with this package if the file it is defined in
// was processed with protoc-gen-gosrcinfo. Returns sd without wrapping if sd's
// parent file already contains source code info.
func WrapService(sd protoreflect.ServiceDescriptor) protoreflect.ServiceDescriptor {
	result, err := updateDescriptor(sd)
	if err != nil {
		panic(err)
	}
	return result
}

// WrapExtensionType wraps the given extension type so that its associated
// descriptor will include source code info that was registered with this package
// if the file it is defined in was processed with protoc-gen-gosrcinfo. Returns
// xt without wrapping if the parent file of xt's descriptor already contains
// source code info.
func WrapExtensionType(xt protoreflect.ExtensionType) protoreflect.ExtensionType {
	if genType, err := protoregistry.GlobalTypes.FindExtensionByName(xt.TypeDescriptor().FullName()); err != nil || genType != xt {
		return xt
	}
	ext, err := updateField(xt.TypeDescriptor().Descriptor())
	if err != nil {
		panic(err)
	}
	return extensionType{ExtensionType: xt, extDesc: ext}
}

// WrapMessageType wraps the given message type so that its associated
// descriptor will include source code info that was registered with this package
// if the file it is defined in was processed with protoc-gen-gosrcinfo. Returns
// mt without wrapping if the parent file of mt's descriptor already contains
// source code info.
func WrapMessageType(mt protoreflect.MessageType) protoreflect.MessageType {
	if genType, err := protoregistry.GlobalTypes.FindMessageByName(mt.Descriptor().FullName()); err != nil || genType != mt {
		return mt
	}
	msg, err := updateDescriptor(mt.Descriptor())
	if err != nil {
		panic(err)
	}
	return messageType{MessageType: mt, msgDesc: msg}
}

type extensionType struct {
	protoreflect.ExtensionType
	extDesc protoreflect.ExtensionDescriptor
}

func (xt extensionType) TypeDescriptor() protoreflect.ExtensionTypeDescriptor {
	return extensionTypeDescriptor{ExtensionDescriptor: xt.extDesc, extType: xt.ExtensionType}
}

type extensionTypeDescriptor struct {
	protoreflect.ExtensionDescriptor
	extType protoreflect.ExtensionType
}

func (xtd extensionTypeDescriptor) Type() protoreflect.ExtensionType {
	return extensionType{ExtensionType: xtd.extType, extDesc: xtd.ExtensionDescriptor}
}

func (xtd extensionTypeDescriptor) Descriptor() protoreflect.ExtensionDescriptor {
	return xtd.ExtensionDescriptor
}

type messageType struct {
	protoreflect.MessageType
	msgDesc protoreflect.MessageDescriptor
}

func (mt messageType) Descriptor() protoreflect.MessageDescriptor {
	return mt.msgDesc
}

func updateField(fd protoreflect.FieldDescriptor) (protoreflect.FieldDescriptor, error) {
	if xtd, ok := fd.(protoreflect.ExtensionTypeDescriptor); ok {
		ext, err := updateField(xtd.Descriptor())
		if err != nil {
			return nil, err
		}
		return extensionTypeDescriptor{ExtensionDescriptor: ext, extType: xtd.Type()}, nil
	}
	d, err := updateDescriptor(fd)
	if err != nil {
		return nil, err
	}
	return d.(protoreflect.FieldDescriptor), nil
}

func updateDescriptor[D protoreflect.Descriptor](d D) (D, error) {
	updatedFile, err := getFile(d.ParentFile())
	if err != nil {
		var zero D
		return zero, err
	}
	if updatedFile == d.ParentFile() {
		// no change
		return d, nil
	}
	updated := findDescriptor(updatedFile, d)
	result, ok := updated.(D)
	if !ok {
		var zero D
		return zero, fmt.Errorf("updated result is type %T which could not be converted to %T", updated, result)
	}
	return result, nil
}

func findDescriptor(fd protoreflect.FileDescriptor, d protoreflect.Descriptor) protoreflect.Descriptor {
	if d == nil {
		return nil
	}
	if _, isFile := d.(protoreflect.FileDescriptor); isFile {
		return fd
	}
	if d.Parent() == nil {
		return d
	}
	switch d := d.(type) {
	case protoreflect.MessageDescriptor:
		parent := findDescriptor(fd, d.Parent()).(messageContainer)
		return parent.Messages().Get(d.Index())
	case protoreflect.FieldDescriptor:
		if d.IsExtension() {
			parent := findDescriptor(fd, d.Parent()).(extensionContainer)
			return parent.Extensions().Get(d.Index())
		} else {
			parent := findDescriptor(fd, d.Parent()).(fieldContainer)
			return parent.Fields().Get(d.Index())
		}
	case protoreflect.OneofDescriptor:
		parent := findDescriptor(fd, d.Parent()).(oneofContainer)
		return parent.Oneofs().Get(d.Index())
	case protoreflect.EnumDescriptor:
		parent := findDescriptor(fd, d.Parent()).(enumContainer)
		return parent.Enums().Get(d.Index())
	case protoreflect.EnumValueDescriptor:
		parent := findDescriptor(fd, d.Parent()).(enumValueContainer)
		return parent.Values().Get(d.Index())
	case protoreflect.ServiceDescriptor:
		parent := findDescriptor(fd, d.Parent()).(serviceContainer)
		return parent.Services().Get(d.Index())
	case protoreflect.MethodDescriptor:
		parent := findDescriptor(fd, d.Parent()).(methodContainer)
		return parent.Methods().Get(d.Index())
	}
	return d
}

type messageContainer interface {
	Messages() protoreflect.MessageDescriptors
}

type extensionContainer interface {
	Extensions() protoreflect.ExtensionDescriptors
}

type fieldContainer interface {
	Fields() protoreflect.FieldDescriptors
}

type oneofContainer interface {
	Oneofs() protoreflect.OneofDescriptors
}

type enumContainer interface {
	Enums() protoreflect.EnumDescriptors
}

type enumValueContainer interface {
	Values() protoreflect.EnumValueDescriptors
}

type serviceContainer interface {
	Services() protoreflect.ServiceDescriptors
}

type methodContainer interface {
	Methods() protoreflect.MethodDescriptors
}
