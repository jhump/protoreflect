// Code generated by protoc-gen-gosrcinfo. DO NOT EDIT.
// source: nopkg/desc_test_nopkg.proto

package nopkg

import "github.com/jhump/protoreflect/desc/sourceinfo"
import "google.golang.org/protobuf/proto"
import "google.golang.org/protobuf/types/descriptorpb"

var srcInfo_nopkg_desc_test_nopkg = []byte{
	0x0a, 0x06, 0x12, 0x04, 0x00, 0x00, 0x04, 0x30, 0x0a, 0x08, 0x0a, 0x01, 0x0c, 0x12, 0x03, 0x00,
	0x00, 0x12, 0x0a, 0x08, 0x0a, 0x01, 0x08, 0x12, 0x03, 0x02, 0x00, 0x54, 0x0a, 0x09, 0x0a, 0x02,
	0x08, 0x0b, 0x12, 0x03, 0x02, 0x00, 0x54, 0x0a, 0x09, 0x0a, 0x02, 0x03, 0x00, 0x12, 0x03, 0x04,
	0x00, 0x30, 0x0a, 0x09, 0x0a, 0x02, 0x0a, 0x00, 0x12, 0x03, 0x04, 0x07, 0x0d,
}

func init() {
	var si descriptorpb.SourceCodeInfo
	if err := proto.Unmarshal(srcInfo_nopkg_desc_test_nopkg, &si); err != nil {
		panic(err)
	}
	sourceinfo.RegisterSourceInfo("nopkg/desc_test_nopkg.proto", &si)
}
