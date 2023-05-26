// This is the first detached comment for the syntax.

//
// This is a second detached comment.

// This is a third.

// Syntax comment...

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v4.23.0
// source: desc_test_comments.proto

// And now the package declaration

package testdata

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Symbols defined in public import of google/protobuf/empty.proto.

type Empty = emptypb.Empty

type Request_MarioCharacters int32

const (
	Request_MARIO     Request_MarioCharacters = 1
	Request_LUIGI     Request_MarioCharacters = 2
	Request_PEACH     Request_MarioCharacters = 3
	Request_BOWSER    Request_MarioCharacters = 4
	Request_WARIO     Request_MarioCharacters = 5
	Request_WALUIGI   Request_MarioCharacters = 6
	Request_SHY_GUY   Request_MarioCharacters = 7
	Request_HEY_HO    Request_MarioCharacters = 7
	Request_MAGIKOOPA Request_MarioCharacters = 8
	Request_KAMEK     Request_MarioCharacters = 8
	Request_SNIFIT    Request_MarioCharacters = -101
)

// Enum value maps for Request_MarioCharacters.
var (
	Request_MarioCharacters_name = map[int32]string{
		1: "MARIO",
		2: "LUIGI",
		3: "PEACH",
		4: "BOWSER",
		5: "WARIO",
		6: "WALUIGI",
		7: "SHY_GUY",
		// Duplicate value: 7: "HEY_HO",
		8: "MAGIKOOPA",
		// Duplicate value: 8: "KAMEK",
		-101: "SNIFIT",
	}
	Request_MarioCharacters_value = map[string]int32{
		"MARIO":     1,
		"LUIGI":     2,
		"PEACH":     3,
		"BOWSER":    4,
		"WARIO":     5,
		"WALUIGI":   6,
		"SHY_GUY":   7,
		"HEY_HO":    7,
		"MAGIKOOPA": 8,
		"KAMEK":     8,
		"SNIFIT":    -101,
	}
)

func (x Request_MarioCharacters) Enum() *Request_MarioCharacters {
	p := new(Request_MarioCharacters)
	*p = x
	return p
}

func (x Request_MarioCharacters) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Request_MarioCharacters) Descriptor() protoreflect.EnumDescriptor {
	return file_desc_test_comments_proto_enumTypes[0].Descriptor()
}

func (Request_MarioCharacters) Type() protoreflect.EnumType {
	return &file_desc_test_comments_proto_enumTypes[0]
}

func (x Request_MarioCharacters) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Do not use.
func (x *Request_MarioCharacters) UnmarshalJSON(b []byte) error {
	num, err := protoimpl.X.UnmarshalJSONEnum(x.Descriptor(), b)
	if err != nil {
		return err
	}
	*x = Request_MarioCharacters(num)
	return nil
}

// Deprecated: Use Request_MarioCharacters.Descriptor instead.
func (Request_MarioCharacters) EnumDescriptor() ([]byte, []int) {
	return file_desc_test_comments_proto_rawDescGZIP(), []int{0, 0}
}

// We need a request for our RPC service below.
//
// Deprecated: Marked as deprecated in desc_test_comments.proto.
type Request struct {
	state           protoimpl.MessageState
	sizeCache       protoimpl.SizeCache
	unknownFields   protoimpl.UnknownFields
	extensionFields protoimpl.ExtensionFields

	// A field comment
	Ids []int32 `protobuf:"varint,1,rep,packed,name=ids,json=|foo|" json:"ids,omitempty"` // field trailer #1...
	// label comment
	Name   *string         `protobuf:"bytes,2,opt,name=name,def=fubar" json:"name,omitempty"`
	Extras *Request_Extras `protobuf:"group,3,opt,name=Extras,json=extras" json:"extras,omitempty"`
	// can be this or that
	//
	// Types that are assignable to Abc:
	//
	//	*Request_This
	//	*Request_That
	Abc isRequest_Abc `protobuf_oneof:"abc"`
	// can be these or those
	//
	// Types that are assignable to Xyz:
	//
	//	*Request_These
	//	*Request_Those
	Xyz isRequest_Xyz `protobuf_oneof:"xyz"`
	// map field
	Things map[string]string `protobuf:"bytes,8,rep,name=things" json:"things,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

// Default values for Request fields.
const (
	Default_Request_Name = string("fubar")
)

func (x *Request) Reset() {
	*x = Request{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test_comments_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Request) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request) ProtoMessage() {}

func (x *Request) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test_comments_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request.ProtoReflect.Descriptor instead.
func (*Request) Descriptor() ([]byte, []int) {
	return file_desc_test_comments_proto_rawDescGZIP(), []int{0}
}

func (x *Request) GetIds() []int32 {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *Request) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return Default_Request_Name
}

func (x *Request) GetExtras() *Request_Extras {
	if x != nil {
		return x.Extras
	}
	return nil
}

func (m *Request) GetAbc() isRequest_Abc {
	if m != nil {
		return m.Abc
	}
	return nil
}

func (x *Request) GetThis() string {
	if x, ok := x.GetAbc().(*Request_This); ok {
		return x.This
	}
	return ""
}

func (x *Request) GetThat() int32 {
	if x, ok := x.GetAbc().(*Request_That); ok {
		return x.That
	}
	return 0
}

func (m *Request) GetXyz() isRequest_Xyz {
	if m != nil {
		return m.Xyz
	}
	return nil
}

func (x *Request) GetThese() string {
	if x, ok := x.GetXyz().(*Request_These); ok {
		return x.These
	}
	return ""
}

func (x *Request) GetThose() int32 {
	if x, ok := x.GetXyz().(*Request_Those); ok {
		return x.Those
	}
	return 0
}

func (x *Request) GetThings() map[string]string {
	if x != nil {
		return x.Things
	}
	return nil
}

type isRequest_Abc interface {
	isRequest_Abc()
}

type Request_This struct {
	This string `protobuf:"bytes,4,opt,name=this,oneof"`
}

type Request_That struct {
	That int32 `protobuf:"varint,5,opt,name=that,oneof"`
}

func (*Request_This) isRequest_Abc() {}

func (*Request_That) isRequest_Abc() {}

type isRequest_Xyz interface {
	isRequest_Xyz()
}

type Request_These struct {
	These string `protobuf:"bytes,6,opt,name=these,oneof"`
}

type Request_Those struct {
	Those int32 `protobuf:"varint,7,opt,name=those,oneof"`
}

func (*Request_These) isRequest_Xyz() {}

func (*Request_Those) isRequest_Xyz() {}

type AnEmptyMessage struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *AnEmptyMessage) Reset() {
	*x = AnEmptyMessage{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test_comments_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AnEmptyMessage) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AnEmptyMessage) ProtoMessage() {}

func (x *AnEmptyMessage) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test_comments_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AnEmptyMessage.ProtoReflect.Descriptor instead.
func (*AnEmptyMessage) Descriptor() ([]byte, []int) {
	return file_desc_test_comments_proto_rawDescGZIP(), []int{1}
}

// Group comment with emoji 😀 😍 👻 ❤ 💯 💥 🐶 🦂 🥑 🍻 🌍 🚕 🪐
type Request_Extras struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Dbl *float64 `protobuf:"fixed64,1,opt,name=dbl" json:"dbl,omitempty"`
	Flt *float32 `protobuf:"fixed32,2,opt,name=flt" json:"flt,omitempty"`
	// Leading comment...
	Str *string `protobuf:"bytes,3,opt,name=str" json:"str,omitempty"` // Trailing comment...
}

func (x *Request_Extras) Reset() {
	*x = Request_Extras{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test_comments_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Request_Extras) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Request_Extras) ProtoMessage() {}

func (x *Request_Extras) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test_comments_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Request_Extras.ProtoReflect.Descriptor instead.
func (*Request_Extras) Descriptor() ([]byte, []int) {
	return file_desc_test_comments_proto_rawDescGZIP(), []int{0, 0}
}

func (x *Request_Extras) GetDbl() float64 {
	if x != nil && x.Dbl != nil {
		return *x.Dbl
	}
	return 0
}

func (x *Request_Extras) GetFlt() float32 {
	if x != nil && x.Flt != nil {
		return *x.Flt
	}
	return 0
}

func (x *Request_Extras) GetStr() string {
	if x != nil && x.Str != nil {
		return *x.Str
	}
	return ""
}

var file_desc_test_comments_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*Request)(nil),
		ExtensionType: (*uint64)(nil),
		Field:         123,
		Name:          "foo.bar.guid1",
		Tag:           "varint,123,opt,name=guid1",
		Filename:      "desc_test_comments.proto",
	},
	{
		ExtendedType:  (*Request)(nil),
		ExtensionType: (*uint64)(nil),
		Field:         124,
		Name:          "foo.bar.guid2",
		Tag:           "varint,124,opt,name=guid2",
		Filename:      "desc_test_comments.proto",
	},
}

// Extension fields to Request.
var (
	// comment for guid1
	//
	// optional uint64 guid1 = 123;
	E_Guid1 = &file_desc_test_comments_proto_extTypes[0]
	// ... and a comment for guid2
	//
	// optional uint64 guid2 = 124;
	E_Guid2 = &file_desc_test_comments_proto_extTypes[1]
)

var File_desc_test_comments_proto protoreflect.FileDescriptor

var file_desc_test_comments_proto_rawDesc = []byte{
	0x0a, 0x18, 0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x63, 0x6f, 0x6d, 0x6d,
	0x65, 0x6e, 0x74, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07, 0x66, 0x6f, 0x6f, 0x2e,
	0x62, 0x61, 0x72, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x17, 0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6f, 0x70, 0x74, 0x69,
	0x6f, 0x6e, 0x73, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xe8, 0x05, 0x0a, 0x07, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x24, 0x0a, 0x03, 0x69, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x05, 0x42, 0x10, 0xaa, 0xf7, 0x04, 0x03, 0x61, 0x62, 0x63, 0xb2, 0xf7, 0x04, 0x03, 0x78,
	0x79, 0x7a, 0x10, 0x01, 0x52, 0x05, 0x7c, 0x66, 0x6f, 0x6f, 0x7c, 0x12, 0x19, 0x0a, 0x04, 0x6e,
	0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x3a, 0x05, 0x66, 0x75, 0x62, 0x61, 0x72,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2f, 0x0a, 0x06, 0x65, 0x78, 0x74, 0x72, 0x61, 0x73,
	0x18, 0x03, 0x20, 0x01, 0x28, 0x0a, 0x32, 0x17, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61, 0x72,
	0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x45, 0x78, 0x74, 0x72, 0x61, 0x73, 0x52,
	0x06, 0x65, 0x78, 0x74, 0x72, 0x61, 0x73, 0x12, 0x14, 0x0a, 0x04, 0x74, 0x68, 0x69, 0x73, 0x18,
	0x04, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x04, 0x74, 0x68, 0x69, 0x73, 0x12, 0x14, 0x0a,
	0x04, 0x74, 0x68, 0x61, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05, 0x48, 0x00, 0x52, 0x04, 0x74,
	0x68, 0x61, 0x74, 0x12, 0x16, 0x0a, 0x05, 0x74, 0x68, 0x65, 0x73, 0x65, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x09, 0x48, 0x01, 0x52, 0x05, 0x74, 0x68, 0x65, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x05, 0x74,
	0x68, 0x6f, 0x73, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x05, 0x48, 0x01, 0x52, 0x05, 0x74, 0x68,
	0x6f, 0x73, 0x65, 0x12, 0x34, 0x0a, 0x06, 0x74, 0x68, 0x69, 0x6e, 0x67, 0x73, 0x18, 0x08, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61, 0x72, 0x2e, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x2e, 0x54, 0x68, 0x69, 0x6e, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x06, 0x74, 0x68, 0x69, 0x6e, 0x67, 0x73, 0x1a, 0x46, 0x0a, 0x06, 0x45, 0x78, 0x74,
	0x72, 0x61, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x64, 0x62, 0x6c, 0x18, 0x01, 0x20, 0x01, 0x28, 0x01,
	0x52, 0x03, 0x64, 0x62, 0x6c, 0x12, 0x10, 0x0a, 0x03, 0x66, 0x6c, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x02, 0x52, 0x03, 0x66, 0x6c, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x73, 0x74, 0x72, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x73, 0x74, 0x72, 0x3a, 0x06, 0xa8, 0xf7, 0x04, 0x00, 0x10,
	0x00, 0x1a, 0x39, 0x0a, 0x0b, 0x54, 0x68, 0x69, 0x6e, 0x67, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79,
	0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b,
	0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xd6, 0x01, 0x0a,
	0x0f, 0x4d, 0x61, 0x72, 0x69, 0x6f, 0x43, 0x68, 0x61, 0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x73,
	0x12, 0x15, 0x0a, 0x05, 0x4d, 0x41, 0x52, 0x49, 0x4f, 0x10, 0x01, 0x1a, 0x0a, 0xa8, 0xf7, 0x04,
	0x96, 0x02, 0xb0, 0xf7, 0x04, 0xf3, 0x04, 0x12, 0x1b, 0x0a, 0x05, 0x4c, 0x55, 0x49, 0x47, 0x49,
	0x10, 0x02, 0x1a, 0x10, 0xc0, 0xf7, 0x04, 0xc8, 0x01, 0xc9, 0xf7, 0x04, 0x64, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x12, 0x09, 0x0a, 0x05, 0x50, 0x45, 0x41, 0x43, 0x48, 0x10, 0x03, 0x12,
	0x0a, 0x0a, 0x06, 0x42, 0x4f, 0x57, 0x53, 0x45, 0x52, 0x10, 0x04, 0x12, 0x09, 0x0a, 0x05, 0x57,
	0x41, 0x52, 0x49, 0x4f, 0x10, 0x05, 0x12, 0x0b, 0x0a, 0x07, 0x57, 0x41, 0x4c, 0x55, 0x49, 0x47,
	0x49, 0x10, 0x06, 0x12, 0x18, 0x0a, 0x07, 0x53, 0x48, 0x59, 0x5f, 0x47, 0x55, 0x59, 0x10, 0x07,
	0x1a, 0x0b, 0xb9, 0xf7, 0x04, 0x75, 0x27, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x12, 0x0a, 0x0a,
	0x06, 0x48, 0x45, 0x59, 0x5f, 0x48, 0x4f, 0x10, 0x07, 0x12, 0x0d, 0x0a, 0x09, 0x4d, 0x41, 0x47,
	0x49, 0x4b, 0x4f, 0x4f, 0x50, 0x41, 0x10, 0x08, 0x12, 0x09, 0x0a, 0x05, 0x4b, 0x41, 0x4d, 0x45,
	0x4b, 0x10, 0x08, 0x12, 0x13, 0x0a, 0x06, 0x53, 0x4e, 0x49, 0x46, 0x49, 0x54, 0x10, 0x9b, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x01, 0x1a, 0x0b, 0xa8, 0xf7, 0x04, 0x7b, 0xb0, 0xf7,
	0x04, 0x81, 0x05, 0x10, 0x01, 0x2a, 0x05, 0x08, 0x64, 0x10, 0xc9, 0x01, 0x2a, 0x1e, 0x08, 0xc9,
	0x01, 0x10, 0xfb, 0x01, 0x1a, 0x16, 0xaa, 0xf7, 0x04, 0x06, 0x73, 0x70, 0x6c, 0x61, 0x74, 0x21,
	0xb2, 0xf7, 0x04, 0x08, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x3a, 0x06, 0xa8, 0xf7,
	0x04, 0x01, 0x18, 0x01, 0x42, 0x05, 0x0a, 0x03, 0x61, 0x62, 0x63, 0x42, 0x2c, 0x0a, 0x03, 0x78,
	0x79, 0x7a, 0x12, 0x25, 0xaa, 0xf7, 0x04, 0x21, 0x77, 0x68, 0x6f, 0x6f, 0x70, 0x73, 0x2c, 0x20,
	0x74, 0x68, 0x69, 0x73, 0x20, 0x68, 0x61, 0x73, 0x20, 0x69, 0x6e, 0x76, 0x61, 0x6c, 0x69, 0x64,
	0x20, 0x55, 0x54, 0x46, 0x38, 0x21, 0x20, 0xbc, 0xff, 0x4a, 0x04, 0x08, 0x0a, 0x10, 0x15, 0x4a,
	0x04, 0x08, 0x1e, 0x10, 0x33, 0x52, 0x03, 0x66, 0x6f, 0x6f, 0x52, 0x03, 0x62, 0x61, 0x72, 0x52,
	0x03, 0x62, 0x61, 0x7a, 0x22, 0x10, 0x0a, 0x0e, 0x41, 0x6e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0xa3, 0x01, 0x0a, 0x0a, 0x52, 0x70, 0x63, 0x53, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x34, 0x0a, 0x0c, 0x53, 0x74, 0x72, 0x65, 0x61, 0x6d, 0x69,
	0x6e, 0x67, 0x52, 0x70, 0x63, 0x12, 0x10, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61, 0x72, 0x2e,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x10, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61,
	0x72, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x28, 0x01, 0x12, 0x4b, 0x0a, 0x08, 0x55,
	0x6e, 0x61, 0x72, 0x79, 0x52, 0x70, 0x63, 0x12, 0x10, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61,
	0x72, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74,
	0x79, 0x22, 0x15, 0xad, 0xf7, 0x04, 0xa4, 0x70, 0x45, 0x41, 0xb1, 0xf7, 0x04, 0x77, 0xbe, 0x9f,
	0x1a, 0x2f, 0xdd, 0x5e, 0x40, 0x88, 0x02, 0x01, 0x1a, 0x12, 0xaa, 0xf7, 0x04, 0x07, 0x08, 0x64,
	0x12, 0x03, 0x62, 0x6f, 0x62, 0xb0, 0xf7, 0x04, 0x01, 0x88, 0x02, 0x00, 0x3a, 0x26, 0x0a, 0x05,
	0x67, 0x75, 0x69, 0x64, 0x31, 0x12, 0x10, 0x2e, 0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61, 0x72, 0x2e,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18, 0x7b, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x67,
	0x75, 0x69, 0x64, 0x31, 0x3a, 0x26, 0x0a, 0x05, 0x67, 0x75, 0x69, 0x64, 0x32, 0x12, 0x10, 0x2e,
	0x66, 0x6f, 0x6f, 0x2e, 0x62, 0x61, 0x72, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x18,
	0x7c, 0x20, 0x01, 0x28, 0x04, 0x52, 0x05, 0x67, 0x75, 0x69, 0x64, 0x32, 0x42, 0x34, 0x5a, 0x32,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x68, 0x75, 0x6d, 0x70,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x72, 0x65, 0x66, 0x6c, 0x65, 0x63, 0x74, 0x2f, 0x76, 0x32,
	0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61,
	0x74, 0x61, 0x50, 0x00,
}

var (
	file_desc_test_comments_proto_rawDescOnce sync.Once
	file_desc_test_comments_proto_rawDescData = file_desc_test_comments_proto_rawDesc
)

func file_desc_test_comments_proto_rawDescGZIP() []byte {
	file_desc_test_comments_proto_rawDescOnce.Do(func() {
		file_desc_test_comments_proto_rawDescData = protoimpl.X.CompressGZIP(file_desc_test_comments_proto_rawDescData)
	})
	return file_desc_test_comments_proto_rawDescData
}

var file_desc_test_comments_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_desc_test_comments_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_desc_test_comments_proto_goTypes = []interface{}{
	(Request_MarioCharacters)(0), // 0: foo.bar.Request.MarioCharacters
	(*Request)(nil),              // 1: foo.bar.Request
	(*AnEmptyMessage)(nil),       // 2: foo.bar.AnEmptyMessage
	(*Request_Extras)(nil),       // 3: foo.bar.Request.Extras
	nil,                          // 4: foo.bar.Request.ThingsEntry
	(*emptypb.Empty)(nil),        // 5: google.protobuf.Empty
}
var file_desc_test_comments_proto_depIdxs = []int32{
	3, // 0: foo.bar.Request.extras:type_name -> foo.bar.Request.Extras
	4, // 1: foo.bar.Request.things:type_name -> foo.bar.Request.ThingsEntry
	1, // 2: foo.bar.guid1:extendee -> foo.bar.Request
	1, // 3: foo.bar.guid2:extendee -> foo.bar.Request
	1, // 4: foo.bar.RpcService.StreamingRpc:input_type -> foo.bar.Request
	1, // 5: foo.bar.RpcService.UnaryRpc:input_type -> foo.bar.Request
	1, // 6: foo.bar.RpcService.StreamingRpc:output_type -> foo.bar.Request
	5, // 7: foo.bar.RpcService.UnaryRpc:output_type -> google.protobuf.Empty
	6, // [6:8] is the sub-list for method output_type
	4, // [4:6] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	2, // [2:4] is the sub-list for extension extendee
	0, // [0:2] is the sub-list for field type_name
}

func init() { file_desc_test_comments_proto_init() }
func file_desc_test_comments_proto_init() {
	if File_desc_test_comments_proto != nil {
		return
	}
	file_desc_test_options_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_desc_test_comments_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Request); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			case 3:
				return &v.extensionFields
			default:
				return nil
			}
		}
		file_desc_test_comments_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AnEmptyMessage); i {
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
		file_desc_test_comments_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Request_Extras); i {
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
	file_desc_test_comments_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Request_This)(nil),
		(*Request_That)(nil),
		(*Request_These)(nil),
		(*Request_Those)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_desc_test_comments_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   4,
			NumExtensions: 2,
			NumServices:   1,
		},
		GoTypes:           file_desc_test_comments_proto_goTypes,
		DependencyIndexes: file_desc_test_comments_proto_depIdxs,
		EnumInfos:         file_desc_test_comments_proto_enumTypes,
		MessageInfos:      file_desc_test_comments_proto_msgTypes,
		ExtensionInfos:    file_desc_test_comments_proto_extTypes,
	}.Build()
	File_desc_test_comments_proto = out.File
	file_desc_test_comments_proto_rawDesc = nil
	file_desc_test_comments_proto_goTypes = nil
	file_desc_test_comments_proto_depIdxs = nil
}
