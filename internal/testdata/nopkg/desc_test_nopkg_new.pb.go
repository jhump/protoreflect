// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.32.0
// 	protoc        v4.25.3
// source: nopkg/desc_test_nopkg_new.proto

package nopkg

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type TopLevel struct {
	state           protoimpl.MessageState
	sizeCache       protoimpl.SizeCache
	unknownFields   protoimpl.UnknownFields
	extensionFields protoimpl.ExtensionFields

	I *int32   `protobuf:"varint,1,opt,name=i" json:"i,omitempty"`
	J *int64   `protobuf:"varint,2,opt,name=j" json:"j,omitempty"`
	K *int32   `protobuf:"zigzag32,3,opt,name=k" json:"k,omitempty"`
	L *int64   `protobuf:"zigzag64,4,opt,name=l" json:"l,omitempty"`
	M *uint32  `protobuf:"varint,5,opt,name=m" json:"m,omitempty"`
	N *uint64  `protobuf:"varint,6,opt,name=n" json:"n,omitempty"`
	O *uint32  `protobuf:"fixed32,7,opt,name=o" json:"o,omitempty"`
	P *uint64  `protobuf:"fixed64,8,opt,name=p" json:"p,omitempty"`
	Q *int32   `protobuf:"fixed32,9,opt,name=q" json:"q,omitempty"`
	R *int64   `protobuf:"fixed64,10,opt,name=r" json:"r,omitempty"`
	S *float32 `protobuf:"fixed32,11,opt,name=s" json:"s,omitempty"`
	T *float64 `protobuf:"fixed64,12,opt,name=t" json:"t,omitempty"`
	U []byte   `protobuf:"bytes,13,opt,name=u" json:"u,omitempty"`
	V *string  `protobuf:"bytes,14,opt,name=v" json:"v,omitempty"`
	W *bool    `protobuf:"varint,15,opt,name=w" json:"w,omitempty"`
}

func (x *TopLevel) Reset() {
	*x = TopLevel{}
	if protoimpl.UnsafeEnabled {
		mi := &file_nopkg_desc_test_nopkg_new_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *TopLevel) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*TopLevel) ProtoMessage() {}

func (x *TopLevel) ProtoReflect() protoreflect.Message {
	mi := &file_nopkg_desc_test_nopkg_new_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use TopLevel.ProtoReflect.Descriptor instead.
func (*TopLevel) Descriptor() ([]byte, []int) {
	return file_nopkg_desc_test_nopkg_new_proto_rawDescGZIP(), []int{0}
}

func (x *TopLevel) GetI() int32 {
	if x != nil && x.I != nil {
		return *x.I
	}
	return 0
}

func (x *TopLevel) GetJ() int64 {
	if x != nil && x.J != nil {
		return *x.J
	}
	return 0
}

func (x *TopLevel) GetK() int32 {
	if x != nil && x.K != nil {
		return *x.K
	}
	return 0
}

func (x *TopLevel) GetL() int64 {
	if x != nil && x.L != nil {
		return *x.L
	}
	return 0
}

func (x *TopLevel) GetM() uint32 {
	if x != nil && x.M != nil {
		return *x.M
	}
	return 0
}

func (x *TopLevel) GetN() uint64 {
	if x != nil && x.N != nil {
		return *x.N
	}
	return 0
}

func (x *TopLevel) GetO() uint32 {
	if x != nil && x.O != nil {
		return *x.O
	}
	return 0
}

func (x *TopLevel) GetP() uint64 {
	if x != nil && x.P != nil {
		return *x.P
	}
	return 0
}

func (x *TopLevel) GetQ() int32 {
	if x != nil && x.Q != nil {
		return *x.Q
	}
	return 0
}

func (x *TopLevel) GetR() int64 {
	if x != nil && x.R != nil {
		return *x.R
	}
	return 0
}

func (x *TopLevel) GetS() float32 {
	if x != nil && x.S != nil {
		return *x.S
	}
	return 0
}

func (x *TopLevel) GetT() float64 {
	if x != nil && x.T != nil {
		return *x.T
	}
	return 0
}

func (x *TopLevel) GetU() []byte {
	if x != nil {
		return x.U
	}
	return nil
}

func (x *TopLevel) GetV() string {
	if x != nil && x.V != nil {
		return *x.V
	}
	return ""
}

func (x *TopLevel) GetW() bool {
	if x != nil && x.W != nil {
		return *x.W
	}
	return false
}

var File_nopkg_desc_test_nopkg_new_proto protoreflect.FileDescriptor

var file_nopkg_desc_test_nopkg_new_proto_rawDesc = []byte{
	0x0a, 0x1f, 0x6e, 0x6f, 0x70, 0x6b, 0x67, 0x2f, 0x64, 0x65, 0x73, 0x63, 0x5f, 0x74, 0x65, 0x73,
	0x74, 0x5f, 0x6e, 0x6f, 0x70, 0x6b, 0x67, 0x5f, 0x6e, 0x65, 0x77, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0xe3, 0x01, 0x0a, 0x08, 0x54, 0x6f, 0x70, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x12, 0x0c,
	0x0a, 0x01, 0x69, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x01, 0x69, 0x12, 0x0c, 0x0a, 0x01,
	0x6a, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x01, 0x6a, 0x12, 0x0c, 0x0a, 0x01, 0x6b, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x11, 0x52, 0x01, 0x6b, 0x12, 0x0c, 0x0a, 0x01, 0x6c, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x12, 0x52, 0x01, 0x6c, 0x12, 0x0c, 0x0a, 0x01, 0x6d, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x01, 0x6d, 0x12, 0x0c, 0x0a, 0x01, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x04, 0x52,
	0x01, 0x6e, 0x12, 0x0c, 0x0a, 0x01, 0x6f, 0x18, 0x07, 0x20, 0x01, 0x28, 0x07, 0x52, 0x01, 0x6f,
	0x12, 0x0c, 0x0a, 0x01, 0x70, 0x18, 0x08, 0x20, 0x01, 0x28, 0x06, 0x52, 0x01, 0x70, 0x12, 0x0c,
	0x0a, 0x01, 0x71, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0f, 0x52, 0x01, 0x71, 0x12, 0x0c, 0x0a, 0x01,
	0x72, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x10, 0x52, 0x01, 0x72, 0x12, 0x0c, 0x0a, 0x01, 0x73, 0x18,
	0x0b, 0x20, 0x01, 0x28, 0x02, 0x52, 0x01, 0x73, 0x12, 0x0c, 0x0a, 0x01, 0x74, 0x18, 0x0c, 0x20,
	0x01, 0x28, 0x01, 0x52, 0x01, 0x74, 0x12, 0x0c, 0x0a, 0x01, 0x75, 0x18, 0x0d, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x01, 0x75, 0x12, 0x0c, 0x0a, 0x01, 0x76, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x01, 0x76, 0x12, 0x0c, 0x0a, 0x01, 0x77, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x08, 0x52, 0x01, 0x77,
	0x2a, 0x05, 0x08, 0x64, 0x10, 0xe9, 0x07, 0x42, 0x40, 0x5a, 0x3e, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x6a, 0x68, 0x75, 0x6d, 0x70, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x72, 0x65, 0x66, 0x6c, 0x65, 0x63, 0x74, 0x2f, 0x76, 0x32, 0x2f, 0x69, 0x6e, 0x74, 0x65,
	0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x74, 0x65, 0x73, 0x74, 0x64, 0x61, 0x74, 0x61, 0x2f, 0x6e, 0x6f,
	0x70, 0x6b, 0x67, 0x3b, 0x6e, 0x6f, 0x70, 0x6b, 0x67,
}

var (
	file_nopkg_desc_test_nopkg_new_proto_rawDescOnce sync.Once
	file_nopkg_desc_test_nopkg_new_proto_rawDescData = file_nopkg_desc_test_nopkg_new_proto_rawDesc
)

func file_nopkg_desc_test_nopkg_new_proto_rawDescGZIP() []byte {
	file_nopkg_desc_test_nopkg_new_proto_rawDescOnce.Do(func() {
		file_nopkg_desc_test_nopkg_new_proto_rawDescData = protoimpl.X.CompressGZIP(file_nopkg_desc_test_nopkg_new_proto_rawDescData)
	})
	return file_nopkg_desc_test_nopkg_new_proto_rawDescData
}

var file_nopkg_desc_test_nopkg_new_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_nopkg_desc_test_nopkg_new_proto_goTypes = []interface{}{
	(*TopLevel)(nil), // 0: TopLevel
}
var file_nopkg_desc_test_nopkg_new_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_nopkg_desc_test_nopkg_new_proto_init() }
func file_nopkg_desc_test_nopkg_new_proto_init() {
	if File_nopkg_desc_test_nopkg_new_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_nopkg_desc_test_nopkg_new_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*TopLevel); i {
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
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_nopkg_desc_test_nopkg_new_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_nopkg_desc_test_nopkg_new_proto_goTypes,
		DependencyIndexes: file_nopkg_desc_test_nopkg_new_proto_depIdxs,
		MessageInfos:      file_nopkg_desc_test_nopkg_new_proto_msgTypes,
	}.Build()
	File_nopkg_desc_test_nopkg_new_proto = out.File
	file_nopkg_desc_test_nopkg_new_proto_rawDesc = nil
	file_nopkg_desc_test_nopkg_new_proto_goTypes = nil
	file_nopkg_desc_test_nopkg_new_proto_depIdxs = nil
}
