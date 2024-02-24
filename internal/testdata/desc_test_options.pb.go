// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.3
// source: desc_test_options.proto

package testdata

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	descriptorpb "google.golang.org/protobuf/types/descriptorpb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Test enum used by custom options
type ReallySimpleEnum int32

const (
	ReallySimpleEnum_VALUE ReallySimpleEnum = 1
)

// Enum value maps for ReallySimpleEnum.
var (
	ReallySimpleEnum_name = map[int32]string{
		1: "VALUE",
	}
	ReallySimpleEnum_value = map[string]int32{
		"VALUE": 1,
	}
)

func (x ReallySimpleEnum) Enum() *ReallySimpleEnum {
	p := new(ReallySimpleEnum)
	*p = x
	return p
}

func (x ReallySimpleEnum) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (ReallySimpleEnum) Descriptor() protoreflect.EnumDescriptor {
	return file_desc_test_options_proto_enumTypes[0].Descriptor()
}

func (ReallySimpleEnum) Type() protoreflect.EnumType {
	return &file_desc_test_options_proto_enumTypes[0]
}

func (x ReallySimpleEnum) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *ReallySimpleEnum) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = ReallySimpleEnum(num)
	return nil
}

// Deprecated: Use ReallySimpleEnum.Descriptor instead.
func (ReallySimpleEnum) EnumDescriptor() ([]byte, []int) {
	return file_desc_test_options_proto_rawDescGZIP(), []int{0}
}

// Test message used by custom options
type ReallySimpleMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id   *uint64 `protobuf:"varint,1,opt,name=id" json:"id,omitempty"`
	Name *string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
}

func (x *ReallySimpleMessage) Reset() {
	*x = ReallySimpleMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test_options_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ReallySimpleMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ReallySimpleMessage) ProtoMessage() {}

func (x *ReallySimpleMessage) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test_options_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ReallySimpleMessage.ProtoReflect.Descriptor instead.
func (*ReallySimpleMessage) Descriptor() ([]byte, []int) {
	return file_desc_test_options_proto_rawDescGZIP(), []int{0}
}

func (x *ReallySimpleMessage) GetId() uint64 {
	if x != nil && x.Id != nil {
		return *x.Id
	}
	return 0
}

func (x *ReallySimpleMessage) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

var file_desc_test_options_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*descriptorpb.MessageOptions)(nil),
		ExtensionType: (*bool)(nil),
		Field:         10101,
		Name:          "testprotos.mfubar",
		Tag:           "varint,10101,opt,name=mfubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: ([]string)(nil),
		Field:         10101,
		Name:          "testprotos.ffubar",
		Tag:           "bytes,10101,rep,name=ffubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.FieldOptions)(nil),
		ExtensionType: ([]byte)(nil),
		Field:         10102,
		Name:          "testprotos.ffubarb",
		Tag:           "bytes,10102,opt,name=ffubarb",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumOptions)(nil),
		ExtensionType: (*int32)(nil),
		Field:         10101,
		Name:          "testprotos.efubar",
		Tag:           "varint,10101,opt,name=efubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumOptions)(nil),
		ExtensionType: (*int32)(nil),
		Field:         10102,
		Name:          "testprotos.efubars",
		Tag:           "zigzag32,10102,opt,name=efubars",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumOptions)(nil),
		ExtensionType: (*int32)(nil),
		Field:         10103,
		Name:          "testprotos.efubarsf",
		Tag:           "fixed32,10103,opt,name=efubarsf",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumOptions)(nil),
		ExtensionType: (*uint32)(nil),
		Field:         10104,
		Name:          "testprotos.efubaru",
		Tag:           "varint,10104,opt,name=efubaru",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumOptions)(nil),
		ExtensionType: (*uint32)(nil),
		Field:         10105,
		Name:          "testprotos.efubaruf",
		Tag:           "fixed32,10105,opt,name=efubaruf",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumValueOptions)(nil),
		ExtensionType: (*int64)(nil),
		Field:         10101,
		Name:          "testprotos.evfubar",
		Tag:           "varint,10101,opt,name=evfubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumValueOptions)(nil),
		ExtensionType: (*int64)(nil),
		Field:         10102,
		Name:          "testprotos.evfubars",
		Tag:           "zigzag64,10102,opt,name=evfubars",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumValueOptions)(nil),
		ExtensionType: (*int64)(nil),
		Field:         10103,
		Name:          "testprotos.evfubarsf",
		Tag:           "fixed64,10103,opt,name=evfubarsf",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumValueOptions)(nil),
		ExtensionType: (*uint64)(nil),
		Field:         10104,
		Name:          "testprotos.evfubaru",
		Tag:           "varint,10104,opt,name=evfubaru",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.EnumValueOptions)(nil),
		ExtensionType: (*uint64)(nil),
		Field:         10105,
		Name:          "testprotos.evfubaruf",
		Tag:           "fixed64,10105,opt,name=evfubaruf",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.ServiceOptions)(nil),
		ExtensionType: (*ReallySimpleMessage)(nil),
		Field:         10101,
		Name:          "testprotos.sfubar",
		Tag:           "bytes,10101,opt,name=sfubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.ServiceOptions)(nil),
		ExtensionType: (*ReallySimpleEnum)(nil),
		Field:         10102,
		Name:          "testprotos.sfubare",
		Tag:           "varint,10102,opt,name=sfubare,enum=testprotos.ReallySimpleEnum",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.MethodOptions)(nil),
		ExtensionType: ([]float32)(nil),
		Field:         10101,
		Name:          "testprotos.mtfubar",
		Tag:           "fixed32,10101,rep,name=mtfubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.MethodOptions)(nil),
		ExtensionType: (*float64)(nil),
		Field:         10102,
		Name:          "testprotos.mtfubard",
		Tag:           "fixed64,10102,opt,name=mtfubard",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.ExtensionRangeOptions)(nil),
		ExtensionType: ([]string)(nil),
		Field:         10101,
		Name:          "testprotos.exfubar",
		Tag:           "bytes,10101,rep,name=exfubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.ExtensionRangeOptions)(nil),
		ExtensionType: ([]byte)(nil),
		Field:         10102,
		Name:          "testprotos.exfubarb",
		Tag:           "bytes,10102,opt,name=exfubarb",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.OneofOptions)(nil),
		ExtensionType: ([]string)(nil),
		Field:         10101,
		Name:          "testprotos.oofubar",
		Tag:           "bytes,10101,rep,name=oofubar",
		Filename:      "desc_test_options.proto",
	},
	{
		ExtendedType:  (*descriptorpb.OneofOptions)(nil),
		ExtensionType: ([]byte)(nil),
		Field:         10102,
		Name:          "testprotos.oofubarb",
		Tag:           "bytes,10102,opt,name=oofubarb",
		Filename:      "desc_test_options.proto",
	},
}

// Extension fields to descriptorpb.MessageOptions.
var (
	// optional bool mfubar = 10101;
	E_Mfubar = &file_desc_test_options_proto_extTypes[0]
)

// Extension fields to descriptorpb.FieldOptions.
var (
	// repeated string ffubar = 10101;
	E_Ffubar = &file_desc_test_options_proto_extTypes[1]
	// optional bytes ffubarb = 10102;
	E_Ffubarb = &file_desc_test_options_proto_extTypes[2]
)

// Extension fields to descriptorpb.EnumOptions.
var (
	// optional int32 efubar = 10101;
	E_Efubar = &file_desc_test_options_proto_extTypes[3]
	// optional sint32 efubars = 10102;
	E_Efubars = &file_desc_test_options_proto_extTypes[4]
	// optional sfixed32 efubarsf = 10103;
	E_Efubarsf = &file_desc_test_options_proto_extTypes[5]
	// optional uint32 efubaru = 10104;
	E_Efubaru = &file_desc_test_options_proto_extTypes[6]
	// optional fixed32 efubaruf = 10105;
	E_Efubaruf = &file_desc_test_options_proto_extTypes[7]
)

// Extension fields to descriptorpb.EnumValueOptions.
var (
	// optional int64 evfubar = 10101;
	E_Evfubar = &file_desc_test_options_proto_extTypes[8]
	// optional sint64 evfubars = 10102;
	E_Evfubars = &file_desc_test_options_proto_extTypes[9]
	// optional sfixed64 evfubarsf = 10103;
	E_Evfubarsf = &file_desc_test_options_proto_extTypes[10]
	// optional uint64 evfubaru = 10104;
	E_Evfubaru = &file_desc_test_options_proto_extTypes[11]
	// optional fixed64 evfubaruf = 10105;
	E_Evfubaruf = &file_desc_test_options_proto_extTypes[12]
)

// Extension fields to descriptorpb.ServiceOptions.
var (
	// optional testprotos.ReallySimpleMessage sfubar = 10101;
	E_Sfubar = &file_desc_test_options_proto_extTypes[13]
	// optional testprotos.ReallySimpleEnum sfubare = 10102;
	E_Sfubare = &file_desc_test_options_proto_extTypes[14]
)

// Extension fields to descriptorpb.MethodOptions.
var (
	// repeated float mtfubar = 10101;
	E_Mtfubar = &file_desc_test_options_proto_extTypes[15]
	// optional double mtfubard = 10102;
	E_Mtfubard = &file_desc_test_options_proto_extTypes[16]
)

// Extension fields to descriptorpb.ExtensionRangeOptions.
var (
	// repeated string exfubar = 10101;
	E_Exfubar = &file_desc_test_options_proto_extTypes[17]
	// optional bytes exfubarb = 10102;
	E_Exfubarb = &file_desc_test_options_proto_extTypes[18]
)

// Extension fields to descriptorpb.OneofOptions.
var (
	// repeated string oofubar = 10101;
	E_Oofubar = &file_desc_test_options_proto_extTypes[19]
	// optional bytes oofubarb = 10102;
	E_Oofubarb = &file_desc_test_options_proto_extTypes[20]
)

var File_desc_test_options_proto protoreflect.FileDescriptor

var file_desc_test_options_proto_rawDesc = []byte{
	0x0a, 0x17, 0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x0a, 0x74, 0x65, 0x73, 0x74, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0x1a, 0x20, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x6f,
	0x72, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x39, 0x0a, 0x13, 0x52, 0x65, 0x61, 0x6c, 0x6c,
	0x79, 0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x0e,
	0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x12,
	0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x2a, 0x1d, 0x0a, 0x10, 0x52, 0x65, 0x61, 0x6c, 0x6c, 0x79, 0x53, 0x69, 0x6d, 0x70,
	0x6c, 0x65, 0x45, 0x6e, 0x75, 0x6d, 0x12, 0x09, 0x0a, 0x05, 0x56, 0x41, 0x4c, 0x55, 0x45, 0x10,
	0x01, 0x3a, 0x38, 0x0a, 0x06, 0x6d, 0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x1f, 0x2e, 0x67, 0x6f,
	0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5, 0x4e, 0x20,
	0x01, 0x28, 0x08, 0x52, 0x06, 0x6d, 0x66, 0x75, 0x62, 0x61, 0x72, 0x3a, 0x36, 0x0a, 0x06, 0x66,
	0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74,
	0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5, 0x4e, 0x20, 0x03, 0x28, 0x09, 0x52, 0x06, 0x66, 0x66, 0x75,
	0x62, 0x61, 0x72, 0x3a, 0x38, 0x0a, 0x07, 0x66, 0x66, 0x75, 0x62, 0x61, 0x72, 0x62, 0x12, 0x1d,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x46, 0x69, 0x65, 0x6c, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf6, 0x4e,
	0x20, 0x01, 0x28, 0x0c, 0x52, 0x07, 0x66, 0x66, 0x75, 0x62, 0x61, 0x72, 0x62, 0x3a, 0x35, 0x0a,
	0x06, 0x65, 0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5, 0x4e, 0x20, 0x01, 0x28, 0x05, 0x52, 0x06, 0x65, 0x66,
	0x75, 0x62, 0x61, 0x72, 0x3a, 0x37, 0x0a, 0x07, 0x65, 0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x12,
	0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf6, 0x4e,
	0x20, 0x01, 0x28, 0x11, 0x52, 0x07, 0x65, 0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x3a, 0x39, 0x0a,
	0x08, 0x65, 0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x66, 0x12, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d,
	0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf7, 0x4e, 0x20, 0x01, 0x28, 0x0f, 0x52, 0x08,
	0x65, 0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x66, 0x3a, 0x37, 0x0a, 0x07, 0x65, 0x66, 0x75, 0x62,
	0x61, 0x72, 0x75, 0x12, 0x1c, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e,
	0x73, 0x18, 0xf8, 0x4e, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x65, 0x66, 0x75, 0x62, 0x61, 0x72,
	0x75, 0x3a, 0x39, 0x0a, 0x08, 0x65, 0x66, 0x75, 0x62, 0x61, 0x72, 0x75, 0x66, 0x12, 0x1c, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6e, 0x75, 0x6d, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf9, 0x4e, 0x20, 0x01,
	0x28, 0x07, 0x52, 0x08, 0x65, 0x66, 0x75, 0x62, 0x61, 0x72, 0x75, 0x66, 0x3a, 0x3c, 0x0a, 0x07,
	0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x21, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5, 0x4e, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x07, 0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x3a, 0x3e, 0x0a, 0x08, 0x65, 0x76,
	0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x12, 0x21, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x56, 0x61, 0x6c,
	0x75, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf6, 0x4e, 0x20, 0x01, 0x28, 0x12,
	0x52, 0x08, 0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x3a, 0x40, 0x0a, 0x09, 0x65, 0x76,
	0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x66, 0x12, 0x21, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x56, 0x61,
	0x6c, 0x75, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf7, 0x4e, 0x20, 0x01, 0x28,
	0x10, 0x52, 0x09, 0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x73, 0x66, 0x3a, 0x3e, 0x0a, 0x08,
	0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x75, 0x12, 0x21, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d, 0x56,
	0x61, 0x6c, 0x75, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf8, 0x4e, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x08, 0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x75, 0x3a, 0x40, 0x0a, 0x09,
	0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x75, 0x66, 0x12, 0x21, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6e, 0x75, 0x6d,
	0x56, 0x61, 0x6c, 0x75, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf9, 0x4e, 0x20,
	0x01, 0x28, 0x06, 0x52, 0x09, 0x65, 0x76, 0x66, 0x75, 0x62, 0x61, 0x72, 0x75, 0x66, 0x3a, 0x59,
	0x0a, 0x06, 0x73, 0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69,
	0x63, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5, 0x4e, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x1f, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x52, 0x65,
	0x61, 0x6c, 0x6c, 0x79, 0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67,
	0x65, 0x52, 0x06, 0x73, 0x66, 0x75, 0x62, 0x61, 0x72, 0x3a, 0x58, 0x0a, 0x07, 0x73, 0x66, 0x75,
	0x62, 0x61, 0x72, 0x65, 0x12, 0x1f, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x4f, 0x70,
	0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf6, 0x4e, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x1c, 0x2e, 0x74,
	0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x52, 0x65, 0x61, 0x6c, 0x6c, 0x79,
	0x53, 0x69, 0x6d, 0x70, 0x6c, 0x65, 0x45, 0x6e, 0x75, 0x6d, 0x52, 0x07, 0x73, 0x66, 0x75, 0x62,
	0x61, 0x72, 0x65, 0x3a, 0x39, 0x0a, 0x07, 0x6d, 0x74, 0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x1e,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x4d, 0x65, 0x74, 0x68, 0x6f, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5,
	0x4e, 0x20, 0x03, 0x28, 0x02, 0x52, 0x07, 0x6d, 0x74, 0x66, 0x75, 0x62, 0x61, 0x72, 0x3a, 0x3b,
	0x0a, 0x08, 0x6d, 0x74, 0x66, 0x75, 0x62, 0x61, 0x72, 0x64, 0x12, 0x1e, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4d, 0x65, 0x74,
	0x68, 0x6f, 0x64, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf6, 0x4e, 0x20, 0x01, 0x28,
	0x01, 0x52, 0x08, 0x6d, 0x74, 0x66, 0x75, 0x62, 0x61, 0x72, 0x64, 0x3a, 0x41, 0x0a, 0x07, 0x65,
	0x78, 0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x26, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x78, 0x74, 0x65, 0x6e, 0x73, 0x69,
	0x6f, 0x6e, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5,
	0x4e, 0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x65, 0x78, 0x66, 0x75, 0x62, 0x61, 0x72, 0x3a, 0x43,
	0x0a, 0x08, 0x65, 0x78, 0x66, 0x75, 0x62, 0x61, 0x72, 0x62, 0x12, 0x26, 0x2e, 0x67, 0x6f, 0x6f,
	0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x78, 0x74,
	0x65, 0x6e, 0x73, 0x69, 0x6f, 0x6e, 0x52, 0x61, 0x6e, 0x67, 0x65, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x18, 0xf6, 0x4e, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x08, 0x65, 0x78, 0x66, 0x75, 0x62,
	0x61, 0x72, 0x62, 0x3a, 0x38, 0x0a, 0x07, 0x6f, 0x6f, 0x66, 0x75, 0x62, 0x61, 0x72, 0x12, 0x1d,
	0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66,
	0x2e, 0x4f, 0x6e, 0x65, 0x6f, 0x66, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf5, 0x4e,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x07, 0x6f, 0x6f, 0x66, 0x75, 0x62, 0x61, 0x72, 0x3a, 0x3a, 0x0a,
	0x08, 0x6f, 0x6f, 0x66, 0x75, 0x62, 0x61, 0x72, 0x62, 0x12, 0x1d, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x4f, 0x6e, 0x65, 0x6f,
	0x66, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0xf6, 0x4e, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x08, 0x6f, 0x6f, 0x66, 0x75, 0x62, 0x61, 0x72, 0x62, 0x42, 0x34, 0x5a, 0x32, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x68, 0x75, 0x6d, 0x70, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x72, 0x65, 0x66, 0x6c, 0x65, 0x63, 0x74, 0x2f, 0x76, 0x32, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61,
}

var (
	file_desc_test_options_proto_rawDescOnce sync.Once
	file_desc_test_options_proto_rawDescData = file_desc_test_options_proto_rawDesc
)

func file_desc_test_options_proto_rawDescGZIP() []byte {
	file_desc_test_options_proto_rawDescOnce.Do(func() {
		file_desc_test_options_proto_rawDescData = protoimpl.X.CompressGZIP(file_desc_test_options_proto_rawDescData)
	})
	return file_desc_test_options_proto_rawDescData
}

var file_desc_test_options_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_desc_test_options_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_desc_test_options_proto_goTypes = []interface{}{
	(ReallySimpleEnum)(0),                      // 0: testprotos.ReallySimpleEnum
	(*ReallySimpleMessage)(nil),                // 1: testprotos.ReallySimpleMessage
	(*descriptorpb.MessageOptions)(nil),        // 2: google.protobuf.MessageOptions
	(*descriptorpb.FieldOptions)(nil),          // 3: google.protobuf.FieldOptions
	(*descriptorpb.EnumOptions)(nil),           // 4: google.protobuf.EnumOptions
	(*descriptorpb.EnumValueOptions)(nil),      // 5: google.protobuf.EnumValueOptions
	(*descriptorpb.ServiceOptions)(nil),        // 6: google.protobuf.ServiceOptions
	(*descriptorpb.MethodOptions)(nil),         // 7: google.protobuf.MethodOptions
	(*descriptorpb.ExtensionRangeOptions)(nil), // 8: google.protobuf.ExtensionRangeOptions
	(*descriptorpb.OneofOptions)(nil),          // 9: google.protobuf.OneofOptions
}
var file_desc_test_options_proto_depIdxs = []int32{
	2,  // 0: testprotos.mfubar:extendee -> google.protobuf.MessageOptions
	3,  // 1: testprotos.ffubar:extendee -> google.protobuf.FieldOptions
	3,  // 2: testprotos.ffubarb:extendee -> google.protobuf.FieldOptions
	4,  // 3: testprotos.efubar:extendee -> google.protobuf.EnumOptions
	4,  // 4: testprotos.efubars:extendee -> google.protobuf.EnumOptions
	4,  // 5: testprotos.efubarsf:extendee -> google.protobuf.EnumOptions
	4,  // 6: testprotos.efubaru:extendee -> google.protobuf.EnumOptions
	4,  // 7: testprotos.efubaruf:extendee -> google.protobuf.EnumOptions
	5,  // 8: testprotos.evfubar:extendee -> google.protobuf.EnumValueOptions
	5,  // 9: testprotos.evfubars:extendee -> google.protobuf.EnumValueOptions
	5,  // 10: testprotos.evfubarsf:extendee -> google.protobuf.EnumValueOptions
	5,  // 11: testprotos.evfubaru:extendee -> google.protobuf.EnumValueOptions
	5,  // 12: testprotos.evfubaruf:extendee -> google.protobuf.EnumValueOptions
	6,  // 13: testprotos.sfubar:extendee -> google.protobuf.ServiceOptions
	6,  // 14: testprotos.sfubare:extendee -> google.protobuf.ServiceOptions
	7,  // 15: testprotos.mtfubar:extendee -> google.protobuf.MethodOptions
	7,  // 16: testprotos.mtfubard:extendee -> google.protobuf.MethodOptions
	8,  // 17: testprotos.exfubar:extendee -> google.protobuf.ExtensionRangeOptions
	8,  // 18: testprotos.exfubarb:extendee -> google.protobuf.ExtensionRangeOptions
	9,  // 19: testprotos.oofubar:extendee -> google.protobuf.OneofOptions
	9,  // 20: testprotos.oofubarb:extendee -> google.protobuf.OneofOptions
	1,  // 21: testprotos.sfubar:type_name -> testprotos.ReallySimpleMessage
	0,  // 22: testprotos.sfubare:type_name -> testprotos.ReallySimpleEnum
	23, // [23:23] is the sub-list for method output_type
	23, // [23:23] is the sub-list for method input_type
	21, // [21:23] is the sub-list for extension type_name
	0,  // [0:21] is the sub-list for extension extendee
	0,  // [0:0] is the sub-list for field type_name
}

func init() { file_desc_test_options_proto_init() }
func file_desc_test_options_proto_init() {
	if File_desc_test_options_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_desc_test_options_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ReallySimpleMessage); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_desc_test_options_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   1,
			NumExtensions: 21,
			NumServices:   0,
		},
		GoTypes:           file_desc_test_options_proto_goTypes,
		DependencyIndexes: file_desc_test_options_proto_depIdxs,
		EnumInfos:         file_desc_test_options_proto_enumTypes,
		MessageInfos:      file_desc_test_options_proto_msgTypes,
		ExtensionInfos:    file_desc_test_options_proto_extTypes,
	}.Build()
	File_desc_test_options_proto = out.File
	file_desc_test_options_proto_rawDesc = nil
	file_desc_test_options_proto_goTypes = nil
	file_desc_test_options_proto_depIdxs = nil
}
