package protobuilder

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FieldType represents the type of a field or extension. It can represent a
// message or enum type or any of the scalar types supported by protobufs.
//
// Message and enum types can reference a message or enum builder. A type that
// refers to a built message or enum descriptor is called an "imported" type.
//
// There are numerous factory methods for creating FieldType instances.
type FieldType struct {
	fieldType       descriptorpb.FieldDescriptorProto_Type
	foreignMsgType  protoreflect.MessageDescriptor
	localMsgType    *MessageBuilder
	foreignEnumType protoreflect.EnumDescriptor
	localEnumType   *EnumBuilder
}

// Kind returns the kind of this field type. If the kind is a message (or group)
// or enum, TypeName() provides the path of the referenced type.
func (ft *FieldType) Kind() protoreflect.Kind {
	return protoreflect.Kind(ft.fieldType)
}

// TypeName returns the fully-qualified path of the referenced message or
// enum type. It returns an empty string if this type does not represent a
// message or enum type.
func (ft *FieldType) TypeName() protoreflect.FullName {
	if ft.foreignMsgType != nil {
		return ft.foreignMsgType.FullName()
	} else if ft.foreignEnumType != nil {
		return ft.foreignEnumType.FullName()
	} else if ft.localMsgType != nil {
		return FullName(ft.localMsgType)
	} else if ft.localEnumType != nil {
		return FullName(ft.localEnumType)
	} else {
		return ""
	}
}

var scalarTypes = map[protoreflect.Kind]*FieldType{
	protoreflect.BoolKind:     {fieldType: descriptorpb.FieldDescriptorProto_TYPE_BOOL},
	protoreflect.Int32Kind:    {fieldType: descriptorpb.FieldDescriptorProto_TYPE_INT32},
	protoreflect.Int64Kind:    {fieldType: descriptorpb.FieldDescriptorProto_TYPE_INT64},
	protoreflect.Sint32Kind:   {fieldType: descriptorpb.FieldDescriptorProto_TYPE_SINT32},
	protoreflect.Sint64Kind:   {fieldType: descriptorpb.FieldDescriptorProto_TYPE_SINT64},
	protoreflect.Uint32Kind:   {fieldType: descriptorpb.FieldDescriptorProto_TYPE_UINT32},
	protoreflect.Uint64Kind:   {fieldType: descriptorpb.FieldDescriptorProto_TYPE_UINT64},
	protoreflect.Fixed32Kind:  {fieldType: descriptorpb.FieldDescriptorProto_TYPE_FIXED32},
	protoreflect.Fixed64Kind:  {fieldType: descriptorpb.FieldDescriptorProto_TYPE_FIXED64},
	protoreflect.Sfixed32Kind: {fieldType: descriptorpb.FieldDescriptorProto_TYPE_SFIXED32},
	protoreflect.Sfixed64Kind: {fieldType: descriptorpb.FieldDescriptorProto_TYPE_SFIXED64},
	protoreflect.FloatKind:    {fieldType: descriptorpb.FieldDescriptorProto_TYPE_FLOAT},
	protoreflect.DoubleKind:   {fieldType: descriptorpb.FieldDescriptorProto_TYPE_DOUBLE},
	protoreflect.StringKind:   {fieldType: descriptorpb.FieldDescriptorProto_TYPE_STRING},
	protoreflect.BytesKind:    {fieldType: descriptorpb.FieldDescriptorProto_TYPE_BYTES},
}

// FieldTypeScalar returns a FieldType for the given scalar type. If the given
// type is not scalar (e.g. it is a message, group, or enum) than this function
// will panic.
func FieldTypeScalar(k protoreflect.Kind) *FieldType {
	if ft, ok := scalarTypes[k]; ok {
		return ft
	}
	panic(fmt.Sprintf("field kind %v is not scalar", k))
}

// FieldTypeInt32 returns a FieldType for the int32 scalar type.
func FieldTypeInt32() *FieldType {
	return FieldTypeScalar(protoreflect.Int32Kind)
}

// FieldTypeUint32 returns a FieldType for the uint32 scalar type.
func FieldTypeUint32() *FieldType {
	return FieldTypeScalar(protoreflect.Uint32Kind)
}

// FieldTypeSint32 returns a FieldType for the sint32 scalar type.
func FieldTypeSint32() *FieldType {
	return FieldTypeScalar(protoreflect.Sint32Kind)
}

// FieldTypeFixed32 returns a FieldType for the fixed32 scalar type.
func FieldTypeFixed32() *FieldType {
	return FieldTypeScalar(protoreflect.Fixed32Kind)
}

// FieldTypeSfixed32 returns a FieldType for the sfixed32 scalar type.
func FieldTypeSfixed32() *FieldType {
	return FieldTypeScalar(protoreflect.Sfixed32Kind)
}

// FieldTypeInt64 returns a FieldType for the int64 scalar type.
func FieldTypeInt64() *FieldType {
	return FieldTypeScalar(protoreflect.Int64Kind)
}

// FieldTypeUint64 returns a FieldType for the uint64 scalar type.
func FieldTypeUint64() *FieldType {
	return FieldTypeScalar(protoreflect.Uint64Kind)
}

// FieldTypeSint64 returns a FieldType for the sint64 scalar type.
func FieldTypeSint64() *FieldType {
	return FieldTypeScalar(protoreflect.Sint64Kind)
}

// FieldTypeFixed64 returns a FieldType for the fixed64 scalar type.
func FieldTypeFixed64() *FieldType {
	return FieldTypeScalar(protoreflect.Fixed64Kind)
}

// FieldTypeSfixed64 returns a FieldType for the sfixed64 scalar type.
func FieldTypeSfixed64() *FieldType {
	return FieldTypeScalar(protoreflect.Sfixed64Kind)
}

// FieldTypeFloat returns a FieldType for the float scalar type.
func FieldTypeFloat() *FieldType {
	return FieldTypeScalar(protoreflect.FloatKind)
}

// FieldTypeDouble returns a FieldType for the double scalar type.
func FieldTypeDouble() *FieldType {
	return FieldTypeScalar(protoreflect.DoubleKind)
}

// FieldTypeBool returns a FieldType for the bool scalar type.
func FieldTypeBool() *FieldType {
	return FieldTypeScalar(protoreflect.BoolKind)
}

// FieldTypeString returns a FieldType for the string scalar type.
func FieldTypeString() *FieldType {
	return FieldTypeScalar(protoreflect.StringKind)
}

// FieldTypeBytes returns a FieldType for the bytes scalar type.
func FieldTypeBytes() *FieldType {
	return FieldTypeScalar(protoreflect.BytesKind)
}

// FieldTypeMessage returns a FieldType for the given message type.
func FieldTypeMessage(mb *MessageBuilder) *FieldType {
	return &FieldType{
		fieldType:    descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		localMsgType: mb,
	}
}

// FieldTypeImportedMessage returns a FieldType that references the given
// message descriptor.
func FieldTypeImportedMessage(md protoreflect.MessageDescriptor) *FieldType {
	return &FieldType{
		fieldType:      descriptorpb.FieldDescriptorProto_TYPE_MESSAGE,
		foreignMsgType: md,
	}
}

// FieldTypeEnum returns a FieldType for the given enum type.
func FieldTypeEnum(eb *EnumBuilder) *FieldType {
	return &FieldType{
		fieldType:     descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		localEnumType: eb,
	}
}

// FieldTypeImportedEnum returns a FieldType that references the given enum
// descriptor.
func FieldTypeImportedEnum(ed protoreflect.EnumDescriptor) *FieldType {
	return &FieldType{
		fieldType:       descriptorpb.FieldDescriptorProto_TYPE_ENUM,
		foreignEnumType: ed,
	}
}

func fieldTypeFromDescriptor(fld protoreflect.FieldDescriptor) *FieldType {
	switch fld.Kind() {
	case protoreflect.GroupKind:
		return &FieldType{fieldType: descriptorpb.FieldDescriptorProto_TYPE_GROUP, foreignMsgType: fld.Message()}
	case protoreflect.MessageKind:
		return FieldTypeImportedMessage(fld.Message())
	case protoreflect.EnumKind:
		return FieldTypeImportedEnum(fld.Enum())
	default:
		return FieldTypeScalar(fld.Kind())
	}
}

// RpcType represents the type of an RPC request or response. The only allowed
// types are messages, but can be streams or unary messages.
//
// Message types can reference a message builder. A type that refers to a built
// message descriptor is called an "imported" type.
//
// To create an RpcType, see RpcTypeMessage and RpcTypeImportedMessage.
type RpcType struct {
	IsStream bool

	foreignType protoreflect.MessageDescriptor
	localType   *MessageBuilder
}

// RpcTypeMessage creates an RpcType that refers to the given message builder.
func RpcTypeMessage(mb *MessageBuilder, stream bool) *RpcType {
	return &RpcType{
		IsStream:  stream,
		localType: mb,
	}
}

// RpcTypeImportedMessage creates an RpcType that refers to the given message
// descriptor.
func RpcTypeImportedMessage(md protoreflect.MessageDescriptor, stream bool) *RpcType {
	return &RpcType{
		IsStream:    stream,
		foreignType: md,
	}
}

// TypeName returns the fully qualified path of the message type to which
// this RpcType refers.
func (rt *RpcType) TypeName() protoreflect.FullName {
	if rt.foreignType != nil {
		return rt.foreignType.FullName()
	} else {
		return FullName(rt.localType)
	}
}
