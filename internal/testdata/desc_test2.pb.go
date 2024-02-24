// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.3
// source: desc_test2.proto

package testdata

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"

	nopkg "github.com/jhump/protoreflect/v2/internal/testdata/nopkg"
	pkg "github.com/jhump/protoreflect/v2/internal/testdata/pkg"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Frobnitz struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	A *TestMessage        `protobuf:"bytes,1,opt,name=a" json:"a,omitempty"`
	B *AnotherTestMessage `protobuf:"bytes,2,opt,name=b" json:"b,omitempty"`
	// Types that are assignable to Abc:
	//
	//	*Frobnitz_C1
	//	*Frobnitz_C2
	Abc isFrobnitz_Abc             `protobuf_oneof:"abc"`
	D   *TestMessage_NestedMessage `protobuf:"bytes,5,opt,name=d" json:"d,omitempty"`
	E   *TestMessage_NestedEnum    `protobuf:"varint,6,opt,name=e,enum=testprotos.TestMessage_NestedEnum,def=2" json:"e,omitempty"`
	// Deprecated: Marked as deprecated in desc_test2.proto.
	F []string `protobuf:"bytes,7,rep,name=f" json:"f,omitempty"`
	// Types that are assignable to Def:
	//
	//	*Frobnitz_G1
	//	*Frobnitz_G2
	//	*Frobnitz_G3
	Def isFrobnitz_Def `protobuf_oneof:"def"`
}

// Default values for Frobnitz fields.
const (
	Default_Frobnitz_E = TestMessage_VALUE2
)

func (x *Frobnitz) Reset() {
	*x = Frobnitz{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test2_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Frobnitz) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Frobnitz) ProtoMessage() {}

func (x *Frobnitz) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test2_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Frobnitz.ProtoReflect.Descriptor instead.
func (*Frobnitz) Descriptor() ([]byte, []int) {
	return file_desc_test2_proto_rawDescGZIP(), []int{0}
}

func (x *Frobnitz) GetA() *TestMessage {
	if x != nil {
		return x.A
	}
	return nil
}

func (x *Frobnitz) GetB() *AnotherTestMessage {
	if x != nil {
		return x.B
	}
	return nil
}

func (m *Frobnitz) GetAbc() isFrobnitz_Abc {
	if m != nil {
		return m.Abc
	}
	return nil
}

func (x *Frobnitz) GetC1() *TestMessage_NestedMessage {
	if x, ok := x.GetAbc().(*Frobnitz_C1); ok {
		return x.C1
	}
	return nil
}

func (x *Frobnitz) GetC2() TestMessage_NestedEnum {
	if x, ok := x.GetAbc().(*Frobnitz_C2); ok {
		return x.C2
	}
	return TestMessage_VALUE1
}

func (x *Frobnitz) GetD() *TestMessage_NestedMessage {
	if x != nil {
		return x.D
	}
	return nil
}

func (x *Frobnitz) GetE() TestMessage_NestedEnum {
	if x != nil && x.E != nil {
		return *x.E
	}
	return Default_Frobnitz_E
}

// Deprecated: Marked as deprecated in desc_test2.proto.
func (x *Frobnitz) GetF() []string {
	if x != nil {
		return x.F
	}
	return nil
}

func (m *Frobnitz) GetDef() isFrobnitz_Def {
	if m != nil {
		return m.Def
	}
	return nil
}

func (x *Frobnitz) GetG1() int32 {
	if x, ok := x.GetDef().(*Frobnitz_G1); ok {
		return x.G1
	}
	return 0
}

func (x *Frobnitz) GetG2() int32 {
	if x, ok := x.GetDef().(*Frobnitz_G2); ok {
		return x.G2
	}
	return 0
}

func (x *Frobnitz) GetG3() uint32 {
	if x, ok := x.GetDef().(*Frobnitz_G3); ok {
		return x.G3
	}
	return 0
}

type isFrobnitz_Abc interface {
	isFrobnitz_Abc()
}

type Frobnitz_C1 struct {
	C1 *TestMessage_NestedMessage `protobuf:"bytes,3,opt,name=c1,oneof"`
}

type Frobnitz_C2 struct {
	C2 TestMessage_NestedEnum `protobuf:"varint,4,opt,name=c2,enum=testprotos.TestMessage_NestedEnum,oneof"`
}

func (*Frobnitz_C1) isFrobnitz_Abc() {}

func (*Frobnitz_C2) isFrobnitz_Abc() {}

type isFrobnitz_Def interface {
	isFrobnitz_Def()
}

type Frobnitz_G1 struct {
	G1 int32 `protobuf:"varint,8,opt,name=g1,oneof"`
}

type Frobnitz_G2 struct {
	G2 int32 `protobuf:"zigzag32,9,opt,name=g2,oneof"`
}

type Frobnitz_G3 struct {
	G3 uint32 `protobuf:"varint,10,opt,name=g3,oneof"`
}

func (*Frobnitz_G1) isFrobnitz_Def() {}

func (*Frobnitz_G2) isFrobnitz_Def() {}

func (*Frobnitz_G3) isFrobnitz_Def() {}

type Whatchamacallit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Foos *pkg.Foo `protobuf:"varint,1,req,name=foos,enum=jhump.protoreflect.desc.Foo" json:"foos,omitempty"`
}

func (x *Whatchamacallit) Reset() {
	*x = Whatchamacallit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test2_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Whatchamacallit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Whatchamacallit) ProtoMessage() {}

func (x *Whatchamacallit) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test2_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Whatchamacallit.ProtoReflect.Descriptor instead.
func (*Whatchamacallit) Descriptor() ([]byte, []int) {
	return file_desc_test2_proto_rawDescGZIP(), []int{1}
}

func (x *Whatchamacallit) GetFoos() pkg.Foo {
	if x != nil && x.Foos != nil {
		return *x.Foos
	}
	return pkg.Foo(0)
}

type Whatzit struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Gyzmeau []*pkg.Bar `protobuf:"bytes,1,rep,name=gyzmeau" json:"gyzmeau,omitempty"`
}

func (x *Whatzit) Reset() {
	*x = Whatzit{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test2_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Whatzit) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Whatzit) ProtoMessage() {}

func (x *Whatzit) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test2_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Whatzit.ProtoReflect.Descriptor instead.
func (*Whatzit) Descriptor() ([]byte, []int) {
	return file_desc_test2_proto_rawDescGZIP(), []int{2}
}

func (x *Whatzit) GetGyzmeau() []*pkg.Bar {
	if x != nil {
		return x.Gyzmeau
	}
	return nil
}

type GroupX struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Groupxi *int64  `protobuf:"varint,1041,opt,name=groupxi" json:"groupxi,omitempty"`
	Groupxs *string `protobuf:"bytes,1042,opt,name=groupxs" json:"groupxs,omitempty"`
}

func (x *GroupX) Reset() {
	*x = GroupX{}
	if protoimpl.UnsafeEnabled {
		mi := &file_desc_test2_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GroupX) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupX) ProtoMessage() {}

func (x *GroupX) ProtoReflect() protoreflect.Message {
	mi := &file_desc_test2_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupX.ProtoReflect.Descriptor instead.
func (*GroupX) Descriptor() ([]byte, []int) {
	return file_desc_test2_proto_rawDescGZIP(), []int{3}
}

func (x *GroupX) GetGroupxi() int64 {
	if x != nil && x.Groupxi != nil {
		return *x.Groupxi
	}
	return 0
}

func (x *GroupX) GetGroupxs() string {
	if x != nil && x.Groupxs != nil {
		return *x.Groupxs
	}
	return ""
}

var file_desc_test2_proto_extTypes = []protoimpl.ExtensionInfo{
	{
		ExtendedType:  (*nopkg.TopLevel)(nil),
		ExtensionType: (*nopkg.TopLevel)(nil),
		Field:         100,
		Name:          "testprotos.otl",
		Tag:           "bytes,100,opt,name=otl",
		Filename:      "desc_test2.proto",
	},
	{
		ExtendedType:  (*nopkg.TopLevel)(nil),
		ExtensionType: (*GroupX)(nil),
		Field:         104,
		Name:          "testprotos.groupx",
		Tag:           "group,104,opt,name=GroupX",
		Filename:      "desc_test2.proto",
	},
}

// Extension fields to nopkg.TopLevel.
var (
	// optional TopLevel otl = 100;
	E_Otl = &file_desc_test2_proto_extTypes[0]
	// optional testprotos.GroupX groupx = 104;
	E_Groupx = &file_desc_test2_proto_extTypes[1]
)

var File_desc_test2_proto protoreflect.FileDescriptor

var file_desc_test2_proto_rawDesc = []byte{
	0x0a, 0x10, 0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x32, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x0a, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x1a, 0x10,
	0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x31, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x1a, 0x17, 0x70, 0x6b, 0x67, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f,
	0x70, 0x6b, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x1b, 0x6e, 0x6f, 0x70, 0x6b, 0x67,
	0x2f, 0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73, 0x74, 0x5f, 0x6e, 0x6f, 0x70, 0x6b, 0x67,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x93, 0x03, 0x0a, 0x08, 0x46, 0x72, 0x6f, 0x62, 0x6e,
	0x69, 0x74, 0x7a, 0x12, 0x25, 0x0a, 0x01, 0x61, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17,
	0x2e, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x54, 0x65, 0x73, 0x74,
	0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x01, 0x61, 0x12, 0x2c, 0x0a, 0x01, 0x62, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1e, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x73, 0x2e, 0x41, 0x6e, 0x6f, 0x74, 0x68, 0x65, 0x72, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x01, 0x62, 0x12, 0x37, 0x0a, 0x02, 0x63, 0x31, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x25, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x73, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4e, 0x65,
	0x73, 0x74, 0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x48, 0x00, 0x52, 0x02, 0x63,
	0x31, 0x12, 0x34, 0x0a, 0x02, 0x63, 0x32, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x22, 0x2e,
	0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x45, 0x6e, 0x75,
	0x6d, 0x48, 0x00, 0x52, 0x02, 0x63, 0x32, 0x12, 0x33, 0x0a, 0x01, 0x64, 0x18, 0x05, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x25, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x2e,
	0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x4e, 0x65, 0x73, 0x74,
	0x65, 0x64, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x52, 0x01, 0x64, 0x12, 0x38, 0x0a, 0x01,
	0x65, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x22, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x54, 0x65, 0x73, 0x74, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x2e, 0x4e, 0x65, 0x73, 0x74, 0x65, 0x64, 0x45, 0x6e, 0x75, 0x6d, 0x3a, 0x06, 0x56, 0x41, 0x4c,
	0x55, 0x45, 0x32, 0x52, 0x01, 0x65, 0x12, 0x10, 0x0a, 0x01, 0x66, 0x18, 0x07, 0x20, 0x03, 0x28,
	0x09, 0x42, 0x02, 0x18, 0x01, 0x52, 0x01, 0x66, 0x12, 0x10, 0x0a, 0x02, 0x67, 0x31, 0x18, 0x08,
	0x20, 0x01, 0x28, 0x05, 0x48, 0x01, 0x52, 0x02, 0x67, 0x31, 0x12, 0x10, 0x0a, 0x02, 0x67, 0x32,
	0x18, 0x09, 0x20, 0x01, 0x28, 0x11, 0x48, 0x01, 0x52, 0x02, 0x67, 0x32, 0x12, 0x10, 0x0a, 0x02,
	0x67, 0x33, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0d, 0x48, 0x01, 0x52, 0x02, 0x67, 0x33, 0x42, 0x05,
	0x0a, 0x03, 0x61, 0x62, 0x63, 0x42, 0x05, 0x0a, 0x03, 0x64, 0x65, 0x66, 0x22, 0x43, 0x0a, 0x0f,
	0x57, 0x68, 0x61, 0x74, 0x63, 0x68, 0x61, 0x6d, 0x61, 0x63, 0x61, 0x6c, 0x6c, 0x69, 0x74, 0x12,
	0x30, 0x0a, 0x04, 0x66, 0x6f, 0x6f, 0x73, 0x18, 0x01, 0x20, 0x02, 0x28, 0x0e, 0x32, 0x1c, 0x2e,
	0x6a, 0x68, 0x75, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x72, 0x65, 0x66, 0x6c, 0x65,
	0x63, 0x74, 0x2e, 0x64, 0x65, 0x73, 0x63, 0x2e, 0x46, 0x6f, 0x6f, 0x52, 0x04, 0x66, 0x6f, 0x6f,
	0x73, 0x22, 0x41, 0x0a, 0x07, 0x57, 0x68, 0x61, 0x74, 0x7a, 0x69, 0x74, 0x12, 0x36, 0x0a, 0x07,
	0x67, 0x79, 0x7a, 0x6d, 0x65, 0x61, 0x75, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e,
	0x6a, 0x68, 0x75, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x72, 0x65, 0x66, 0x6c, 0x65,
	0x63, 0x74, 0x2e, 0x64, 0x65, 0x73, 0x63, 0x2e, 0x42, 0x61, 0x72, 0x52, 0x07, 0x67, 0x79, 0x7a,
	0x6d, 0x65, 0x61, 0x75, 0x22, 0x3e, 0x0a, 0x06, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x58, 0x12, 0x19,
	0x0a, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x78, 0x69, 0x18, 0x91, 0x08, 0x20, 0x01, 0x28, 0x03,
	0x52, 0x07, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x78, 0x69, 0x12, 0x19, 0x0a, 0x07, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x78, 0x73, 0x18, 0x92, 0x08, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x78, 0x73, 0x3a, 0x26, 0x0a, 0x03, 0x6f, 0x74, 0x6c, 0x12, 0x09, 0x2e, 0x54, 0x6f,
	0x70, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x18, 0x64, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x54,
	0x6f, 0x70, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x52, 0x03, 0x6f, 0x74, 0x6c, 0x3a, 0x35, 0x0a, 0x06,
	0x67, 0x72, 0x6f, 0x75, 0x70, 0x78, 0x12, 0x09, 0x2e, 0x54, 0x6f, 0x70, 0x4c, 0x65, 0x76, 0x65,
	0x6c, 0x18, 0x68, 0x20, 0x01, 0x28, 0x0a, 0x32, 0x12, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x73, 0x2e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x58, 0x52, 0x06, 0x67, 0x72, 0x6f,
	0x75, 0x70, 0x78, 0x42, 0xa9, 0x01, 0x0a, 0x31, 0x63, 0x6f, 0x6d, 0x2e, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x6a, 0x68, 0x75, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x72, 0x65,
	0x66, 0x6c, 0x65, 0x63, 0x74, 0x2e, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2e, 0x74,
	0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73, 0x50, 0x01, 0x5a, 0x32, 0x67, 0x69, 0x74,
	0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x68, 0x75, 0x6d, 0x70, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x72, 0x65, 0x66, 0x6c, 0x65, 0x63, 0x74, 0x2f, 0x76, 0x32, 0x2f, 0x69, 0x6e,
	0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0xa0,
	0x01, 0x01, 0xf8, 0x01, 0x01, 0xaa, 0x02, 0x1d, 0x6a, 0x68, 0x75, 0x6d, 0x70, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x72, 0x65, 0x66, 0x6c, 0x65, 0x63, 0x74, 0x2e, 0x74, 0x65, 0x73, 0x74, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x73, 0xea, 0x02, 0x17, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x72, 0x65, 0x66,
	0x6c, 0x65, 0x63, 0x74, 0x2d, 0x74, 0x65, 0x73, 0x74, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x73,
}

var (
	file_desc_test2_proto_rawDescOnce sync.Once
	file_desc_test2_proto_rawDescData = file_desc_test2_proto_rawDesc
)

func file_desc_test2_proto_rawDescGZIP() []byte {
	file_desc_test2_proto_rawDescOnce.Do(func() {
		file_desc_test2_proto_rawDescData = protoimpl.X.CompressGZIP(file_desc_test2_proto_rawDescData)
	})
	return file_desc_test2_proto_rawDescData
}

var file_desc_test2_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_desc_test2_proto_goTypes = []interface{}{
	(*Frobnitz)(nil),                  // 0: testprotos.Frobnitz
	(*Whatchamacallit)(nil),           // 1: testprotos.Whatchamacallit
	(*Whatzit)(nil),                   // 2: testprotos.Whatzit
	(*GroupX)(nil),                    // 3: testprotos.GroupX
	(*TestMessage)(nil),               // 4: testprotos.TestMessage
	(*AnotherTestMessage)(nil),        // 5: testprotos.AnotherTestMessage
	(*TestMessage_NestedMessage)(nil), // 6: testprotos.TestMessage.NestedMessage
	(TestMessage_NestedEnum)(0),       // 7: testprotos.TestMessage.NestedEnum
	(pkg.Foo)(0),                      // 8: jhump.protoreflect.desc.Foo
	(*pkg.Bar)(nil),                   // 9: jhump.protoreflect.desc.Bar
	(*nopkg.TopLevel)(nil),            // 10: TopLevel
}
var file_desc_test2_proto_depIdxs = []int32{
	4,  // 0: testprotos.Frobnitz.a:type_name -> testprotos.TestMessage
	5,  // 1: testprotos.Frobnitz.b:type_name -> testprotos.AnotherTestMessage
	6,  // 2: testprotos.Frobnitz.c1:type_name -> testprotos.TestMessage.NestedMessage
	7,  // 3: testprotos.Frobnitz.c2:type_name -> testprotos.TestMessage.NestedEnum
	6,  // 4: testprotos.Frobnitz.d:type_name -> testprotos.TestMessage.NestedMessage
	7,  // 5: testprotos.Frobnitz.e:type_name -> testprotos.TestMessage.NestedEnum
	8,  // 6: testprotos.Whatchamacallit.foos:type_name -> jhump.protoreflect.desc.Foo
	9,  // 7: testprotos.Whatzit.gyzmeau:type_name -> jhump.protoreflect.desc.Bar
	10, // 8: testprotos.otl:extendee -> TopLevel
	10, // 9: testprotos.groupx:extendee -> TopLevel
	10, // 10: testprotos.otl:type_name -> TopLevel
	3,  // 11: testprotos.groupx:type_name -> testprotos.GroupX
	12, // [12:12] is the sub-list for method output_type
	12, // [12:12] is the sub-list for method input_type
	10, // [10:12] is the sub-list for extension type_name
	8,  // [8:10] is the sub-list for extension extendee
	0,  // [0:8] is the sub-list for field type_name
}

func init() { file_desc_test2_proto_init() }
func file_desc_test2_proto_init() {
	if File_desc_test2_proto != nil {
		return
	}
	file_desc_test1_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_desc_test2_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Frobnitz); i {
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
		file_desc_test2_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Whatchamacallit); i {
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
		file_desc_test2_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Whatzit); i {
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
		file_desc_test2_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GroupX); i {
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
	file_desc_test2_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Frobnitz_C1)(nil),
		(*Frobnitz_C2)(nil),
		(*Frobnitz_G1)(nil),
		(*Frobnitz_G2)(nil),
		(*Frobnitz_G3)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_desc_test2_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 2,
			NumServices:   0,
		},
		GoTypes:           file_desc_test2_proto_goTypes,
		DependencyIndexes: file_desc_test2_proto_depIdxs,
		MessageInfos:      file_desc_test2_proto_msgTypes,
		ExtensionInfos:    file_desc_test2_proto_extTypes,
	}.Build()
	File_desc_test2_proto = out.File
	file_desc_test2_proto_rawDesc = nil
	file_desc_test2_proto_goTypes = nil
	file_desc_test2_proto_depIdxs = nil
}
