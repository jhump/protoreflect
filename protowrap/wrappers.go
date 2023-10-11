package protowrap

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/wrappers"
)

// ProtoWrapper is a protoreflect.Descriptor that wraps an underlying
// descriptor proto. It provides the same interface as Descriptor but
// with one extra operation, to efficiently query for the underlying
// descriptor proto.
//
// Descriptors that implement this should also implement another method
// whose specified return type is the concrete type returned by the
// AsProto method. The name of this method varies by the type of this
// descriptor:
//
//	 Descriptor Type        Other Method Name
//	---------------------+------------------------------------
//	 FileDescriptor      |  FileDescriptorProto()
//	 MessageDescriptor   |  MessageDescriptorProto()
//	 FieldDescriptor     |  FieldDescriptorProto()
//	 OneofDescriptor     |  OneOfDescriptorProto()
//	 EnumDescriptor      |  EnumDescriptorProto()
//	 EnumValueDescriptor |  EnumValueDescriptorProto()
//	 ServiceDescriptor   |  ServiceDescriptorProto()
//	 MethodDescriptor    |  MethodDescriptorProto()
//
// For example, a ProtoWrapper that implements FileDescriptor
// returns a *descriptorpb.FileDescriptorProto value from its AsProto
// method and also provides a method with the following signature:
//
//	FileDescriptorProto() *descriptorpb.FileDescriptorProto
type ProtoWrapper interface {
	protoreflect.Descriptor
	// AsProto returns the underlying descriptor proto. The concrete
	// type of the proto message depends on the type of this
	// descriptor:
	//    Descriptor Type        Proto Message Type
	//   ---------------------+------------------------------------
	//    FileDescriptor      |  *descriptorpb.FileDescriptorProto
	//    MessageDescriptor   |  *descriptorpb.DescriptorProto
	//    FieldDescriptor     |  *descriptorpb.FieldDescriptorProto
	//    OneofDescriptor     |  *descriptorpb.OneofDescriptorProto
	//    EnumDescriptor      |  *descriptorpb.EnumDescriptorProto
	//    EnumValueDescriptor |  *descriptorpb.EnumValueDescriptorProto
	//    ServiceDescriptor   |  *descriptorpb.ServiceDescriptorProto
	//    MethodDescriptor    |  *descriptorpb.MethodDescriptorProto
	AsProto() proto.Message
}

var _ ProtoWrapper = wrappers.ProtoWrapper(nil)
var _ wrappers.ProtoWrapper = ProtoWrapper(nil)
var _ ProtoWrapper = (*wrappers.File)(nil)
var _ ProtoWrapper = (*wrappers.Message)(nil)
var _ ProtoWrapper = (*wrappers.Field)(nil)
var _ ProtoWrapper = (*wrappers.Oneof)(nil)
var _ ProtoWrapper = (*wrappers.Extension)(nil)
var _ ProtoWrapper = (*wrappers.Enum)(nil)
var _ ProtoWrapper = (*wrappers.EnumValue)(nil)
var _ ProtoWrapper = (*wrappers.Service)(nil)
var _ ProtoWrapper = (*wrappers.Method)(nil)

// FileWrapper is a ProtoWrapper for files: it implements
// [protoreflect.FileDescriptor] and wraps a [*descriptorpb.FileDescriptorProto].
//
// Implementations of this interface should return wrappers from the methods
// used to access child elements. For example, calling file.Messages().Get(0)
// should also return a MessageWrapper, not just a plain
// [protoreflect.MessageDescriptor].
type FileWrapper interface {
	protoreflect.FileDescriptor
	FileDescriptorProto() *descriptorpb.FileDescriptorProto
}

var _ FileWrapper = (*wrappers.File)(nil)

// MessageWrapper is a ProtoWrapper for messages: it implements
// [protoreflect.MessageDescriptor] and wraps a [*descriptorpb.DescriptorProto].
//
// Implementations of this interface should return wrappers from the methods
// used to access child elements. For example, calling msg.Fields().Get(0)
// should also return a FieldWrapper, not just a plain
// [protoreflect.FieldDescriptor].
type MessageWrapper interface {
	protoreflect.MessageDescriptor
	MessageDescriptorProto() *descriptorpb.DescriptorProto
}

var _ MessageWrapper = (*wrappers.Message)(nil)

// FieldWrapper is a ProtoWrapper for fields: it implements
// [protoreflect.FieldDescriptor] and wraps a [*descriptorpb.FieldDescriptorProto].
//
// Implementations of this interface should return wrappers from the methods
// used to access related elements. For example, calling field.ContainingOneof()
// should also return a OneofWrapper, not just a plain
// [protoreflect.OneofDescriptor]. This may not always be feasible, like if the
// related element (like the field's message or enum type) is defined in another
// file that was not created as a FileWrapper.
type FieldWrapper interface {
	protoreflect.FieldDescriptor
	FieldDescriptorProto() *descriptorpb.FieldDescriptorProto
}

var _ FieldWrapper = (*wrappers.Field)(nil)
var _ FieldWrapper = (*wrappers.Extension)(nil)

// OneofWrapper is a ProtoWrapper for oneofs: it implements
// [protoreflect.OneofDescriptor] and wraps a [*descriptorpb.OneofDescriptorProto].
//
// Implementations of this interface should return wrappers from the methods
// used to access child elements. For example, calling oneof.Fields().Get(0)
// should also return a FieldWrapper, not just a plain
// [protoreflect.FieldDescriptor].
type OneofWrapper interface {
	protoreflect.OneofDescriptor
	OneofDescriptorProto() *descriptorpb.OneofDescriptorProto
}

var _ OneofWrapper = (*wrappers.Oneof)(nil)

// EnumWrapper is a ProtoWrapper for enums: it implements
// [protoreflect.EnumDescriptor] and wraps a [*descriptorpb.EnumDescriptorProto].
//
// Implementations of this interface should return wrappers from the methods
// used to access child elements. For example, calling enum.Values().Get(0)
// should also return an EnumValueWrapper, not just a plain
// [protoreflect.EnumValueDescriptor].
type EnumWrapper interface {
	protoreflect.EnumDescriptor
	EnumDescriptorProto() *descriptorpb.EnumDescriptorProto
}

var _ EnumWrapper = (*wrappers.Enum)(nil)

// EnumValueWrapper is a ProtoWrapper for enum values: it implements
// [protoreflect.EnumValueDescriptor] and wraps a [*descriptorpb.EnumValueDescriptorProto].
type EnumValueWrapper interface {
	protoreflect.EnumValueDescriptor
	EnumValueDescriptorProto() *descriptorpb.EnumValueDescriptorProto
}

var _ EnumValueWrapper = (*wrappers.EnumValue)(nil)

// ServiceWrapper is a ProtoWrapper for services: it implements
// [protoreflect.ServiceDescriptor] and wraps a [*descriptorpb.ServiceDescriptorProto].
//
// Implementations of this interface should return wrappers from the methods
// used to access child elements. For example, calling svc.Methods().Get(0)
// should also return an MethodWrapper, not just a plain
// [protoreflect.MethodDescriptor].
type ServiceWrapper interface {
	protoreflect.ServiceDescriptor
	ServiceDescriptorProto() *descriptorpb.ServiceDescriptorProto
}

var _ ServiceWrapper = (*wrappers.Service)(nil)

// MethodWrapper is a ProtoWrapper for methods: it implements
// [protoreflect.MethodDescriptor] and wraps a [*descriptorpb.MethodDescriptorProto].
//
// Implementations of this interface should return wrappers from the methods
// used to access related elements. For example, calling method.Input()
// should also return an MessageWrapper, not just a plain
// [protoreflect.MessageDescriptor]. However, this may not always be feasible,
// such as if the message type is defined in another file that was not created
// as a FileWrapper.
type MethodWrapper interface {
	protoreflect.MethodDescriptor
	MethodDescriptorProto() *descriptorpb.MethodDescriptorProto
}

var _ MethodWrapper = (*wrappers.Method)(nil)
