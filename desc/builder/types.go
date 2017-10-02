package builder

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
)

import (
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

type FieldType struct {
	fieldType       dpb.FieldDescriptorProto_Type
	foreignMsgType  *desc.MessageDescriptor
	localMsgType    *MessageBuilder
	foreignEnumType *desc.EnumDescriptor
	localEnumType   *EnumBuilder
}

func (ft *FieldType) GetType() dpb.FieldDescriptorProto_Type {
	return ft.fieldType
}

func (ft *FieldType) GetTypeName() string {
	if ft.foreignMsgType != nil {
		return ft.foreignMsgType.GetFullyQualifiedName()
	} else if ft.foreignEnumType != nil {
		return ft.foreignEnumType.GetFullyQualifiedName()
	} else if ft.localMsgType != nil {
		return GetFullyQualifiedName(ft.localMsgType)
	} else if ft.localEnumType != nil {
		return GetFullyQualifiedName(ft.localEnumType)
	} else {
		return ""
	}
}

var scalarTypes = map[dpb.FieldDescriptorProto_Type]*FieldType{
	dpb.FieldDescriptorProto_TYPE_BOOL:     {fieldType: dpb.FieldDescriptorProto_TYPE_BOOL},
	dpb.FieldDescriptorProto_TYPE_INT32:    {fieldType: dpb.FieldDescriptorProto_TYPE_INT32},
	dpb.FieldDescriptorProto_TYPE_INT64:    {fieldType: dpb.FieldDescriptorProto_TYPE_INT64},
	dpb.FieldDescriptorProto_TYPE_SINT32:   {fieldType: dpb.FieldDescriptorProto_TYPE_SINT32},
	dpb.FieldDescriptorProto_TYPE_SINT64:   {fieldType: dpb.FieldDescriptorProto_TYPE_SINT64},
	dpb.FieldDescriptorProto_TYPE_UINT32:   {fieldType: dpb.FieldDescriptorProto_TYPE_UINT32},
	dpb.FieldDescriptorProto_TYPE_UINT64:   {fieldType: dpb.FieldDescriptorProto_TYPE_UINT64},
	dpb.FieldDescriptorProto_TYPE_FIXED32:  {fieldType: dpb.FieldDescriptorProto_TYPE_FIXED32},
	dpb.FieldDescriptorProto_TYPE_FIXED64:  {fieldType: dpb.FieldDescriptorProto_TYPE_FIXED64},
	dpb.FieldDescriptorProto_TYPE_SFIXED32: {fieldType: dpb.FieldDescriptorProto_TYPE_SFIXED32},
	dpb.FieldDescriptorProto_TYPE_SFIXED64: {fieldType: dpb.FieldDescriptorProto_TYPE_SFIXED64},
	dpb.FieldDescriptorProto_TYPE_FLOAT:    {fieldType: dpb.FieldDescriptorProto_TYPE_FLOAT},
	dpb.FieldDescriptorProto_TYPE_DOUBLE:   {fieldType: dpb.FieldDescriptorProto_TYPE_DOUBLE},
	dpb.FieldDescriptorProto_TYPE_STRING:   {fieldType: dpb.FieldDescriptorProto_TYPE_STRING},
	dpb.FieldDescriptorProto_TYPE_BYTES:    {fieldType: dpb.FieldDescriptorProto_TYPE_BYTES},
}

func FieldTypeScalar(t dpb.FieldDescriptorProto_Type) *FieldType {
	if ft, ok := scalarTypes[t]; ok {
		return ft
	}
	panic(fmt.Sprintf("field %v is not scalar", t))
}

func FieldTypeInt32() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_INT32)
}

func FieldTypeUInt32() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_UINT32)
}

func FieldTypeSInt32() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_SINT32)
}

func FieldTypeFixed32() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_FIXED32)
}

func FieldTypeSFixed32() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_SFIXED32)
}

func FieldTypeInt64() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_INT64)
}

func FieldTypeUInt64() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_UINT64)
}

func FieldTypeSInt64() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_SINT64)
}

func FieldTypeFixed64() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_FIXED64)
}

func FieldTypeSFixed64() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_SFIXED64)
}

func FieldTypeFloat() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_FLOAT)
}

func FieldTypeDouble() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_DOUBLE)
}

func FieldTypeBool() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_BOOL)
}

func FieldTypeString() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_STRING)
}

func FieldTypeBytes() *FieldType {
	return FieldTypeScalar(dpb.FieldDescriptorProto_TYPE_BYTES)
}

func FieldTypeMessage(mb *MessageBuilder) *FieldType {
	return &FieldType{
		fieldType:    dpb.FieldDescriptorProto_TYPE_MESSAGE,
		localMsgType: mb,
	}
}

func FieldTypeImportedMessage(md *desc.MessageDescriptor) *FieldType {
	return &FieldType{
		fieldType:      dpb.FieldDescriptorProto_TYPE_MESSAGE,
		foreignMsgType: md,
	}
}

func FieldTypeEnum(eb *EnumBuilder) *FieldType {
	return &FieldType{
		fieldType:     dpb.FieldDescriptorProto_TYPE_ENUM,
		localEnumType: eb,
	}
}

func FieldTypeImportedEnum(ed *desc.EnumDescriptor) *FieldType {
	return &FieldType{
		fieldType:       dpb.FieldDescriptorProto_TYPE_ENUM,
		foreignEnumType: ed,
	}
}

type RpcType struct {
	IsStream bool

	foreignType *desc.MessageDescriptor
	localType   *MessageBuilder
}

func RpcTypeMessage(mb *MessageBuilder, stream bool) *RpcType {
	return &RpcType{
		IsStream:  stream,
		localType: mb,
	}
}

func RpcTypeImportedMessage(md *desc.MessageDescriptor, stream bool) *RpcType {
	return &RpcType{
		IsStream:    stream,
		foreignType: md,
	}
}

func (rt *RpcType) GetTypeName() string {
	if rt.foreignType != nil {
		return rt.foreignType.GetFullyQualifiedName()
	} else {
		return GetFullyQualifiedName(rt.localType)
	}
}
