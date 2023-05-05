package protowrap

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
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

type FileWrapper interface {
	protoreflect.FileDescriptor
	FileDescriptorProto() *descriptorpb.FileDescriptorProto
}

type MessageWrapper interface {
	protoreflect.MessageDescriptor
	MessageDescriptorProto() *descriptorpb.DescriptorProto
}

type FieldWrapper interface {
	protoreflect.FieldDescriptor
	FieldDescriptorProto() *descriptorpb.FieldDescriptorProto
}

type OneofWrapper interface {
	protoreflect.OneofDescriptor
	OneofDescriptorProto() *descriptorpb.OneofDescriptorProto
}

type EnumWrapper interface {
	protoreflect.EnumDescriptor
	EnumDescriptorProto() *descriptorpb.EnumDescriptorProto
}

type EnumValueWrapper interface {
	protoreflect.EnumValueDescriptor
	EnumValueDescriptorProto() *descriptorpb.EnumValueDescriptorProto
}

type ServiceWrapper interface {
	protoreflect.ServiceDescriptor
	ServiceDescriptorProto() *descriptorpb.ServiceDescriptorProto
}

type MethodWrapper interface {
	protoreflect.MethodDescriptor
	MethodDescriptorProto() *descriptorpb.MethodDescriptorProto
}
