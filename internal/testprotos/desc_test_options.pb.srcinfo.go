// Code generated by protoc-gen-gosrcinfo. DO NOT EDIT.
// source: desc_test_options.proto

package testprotos

import (
	sourceinfo "github.com/jhump/protoreflect/v2/sourceinfo"
)

func init() {
	srcInfo := []byte{
		0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x7c, 0x95, 0xdb, 0x52, 0x23, 0x37,
		0x17, 0x85, 0xa5, 0x5e, 0x92, 0xad, 0x96, 0x8f, 0xbd, 0xda, 0x27, 0x8c, 0x6d, 0x8c, 0x8d, 0xb1,
		0x61, 0xc0, 0x7f, 0x0d, 0x4c, 0x0d, 0xfc, 0x35, 0x90, 0x17, 0xc8, 0x6d, 0x5e, 0x20, 0x07, 0x2a,
		0x95, 0x0b, 0x86, 0x54, 0x19, 0x2e, 0xf2, 0xf6, 0x29, 0xf5, 0xde, 0x9e, 0xea, 0xab, 0xdc, 0x2d,
		0x2d, 0xab, 0x3f, 0x2f, 0x6d, 0xed, 0xde, 0x1d, 0x1b, 0x74, 0xc6, 0xfc, 0x64, 0x63, 0x88, 0xb6,
		0x4d, 0x18, 0xc3, 0xa4, 0x02, 0x91, 0x99, 0x9f, 0x63, 0x1e, 0xb3, 0xd0, 0x12, 0x19, 0xa2, 0xcd,
		0x08, 0x67, 0xca, 0x64, 0xc2, 0x10, 0x0d, 0x73, 0x1d, 0xf3, 0x68, 0x9b, 0x74, 0xc1, 0x44, 0x9b,
		0xdc, 0xa6, 0x21, 0xf2, 0xb0, 0x89, 0x31, 0xa2, 0x69, 0x32, 0x22, 0x34, 0x55, 0xbb, 0xe4, 0xf7,
		0x45, 0x7b, 0x22, 0x2f, 0x86, 0xa2, 0x2d, 0x91, 0x8f, 0x66, 0xa2, 0x41, 0xe4, 0x67, 0x17, 0x4a,
		0x6c, 0x9b, 0x9e, 0x10, 0x2d, 0xd1, 0x09, 0xdb, 0x6a, 0x47, 0xfa, 0xfb, 0x76, 0x73, 0x2d, 0xda,
		0x25, 0x5f, 0x88, 0xd6, 0x13, 0x9d, 0x62, 0x2c, 0x3a, 0xed, 0x9f, 0x2c, 0x44, 0x83, 0xe8, 0x9c,
		0x5f, 0x56, 0x98, 0x8c, 0xe8, 0x2a, 0x26, 0xab, 0x61, 0x32, 0x97, 0x7c, 0xc1, 0x64, 0x9e, 0xe8,
		0x16, 0x23, 0xd1, 0x96, 0xe8, 0x8e, 0x05, 0x93, 0x81, 0xe8, 0x56, 0x98, 0x14, 0xac, 0x30, 0x63,
		0x09, 0x06, 0x82, 0xe1, 0xb2, 0xda, 0x81, 0x8c, 0x28, 0x9a, 0x2b, 0xd1, 0x2e, 0xf9, 0x42, 0x84,
		0x27, 0xa8, 0x44, 0x58, 0x82, 0xe3, 0xb9, 0xe8, 0xf4, 0xec, 0x72, 0x53, 0x61, 0x1c, 0x51, 0x86,
		0x5d, 0x65, 0xbb, 0x1a, 0xc6, 0x55, 0xbe, 0x60, 0x9c, 0x27, 0x4a, 0x3d, 0x9f, 0xb3, 0x44, 0x39,
		0x39, 0x13, 0x0d, 0xa2, 0x5c, 0x6d, 0x2b, 0x8c, 0x27, 0x06, 0xe1, 0x53, 0x65, 0xfb, 0x1a, 0xc6,
		0xbb, 0xe4, 0x0b, 0xc6, 0xa7, 0x3d, 0xc5, 0x89, 0x68, 0x4b, 0x0c, 0xa6, 0xba, 0x07, 0xc4, 0x60,
		0x73, 0x5d, 0x61, 0x1a, 0xc4, 0x50, 0xd3, 0x34, 0x6a, 0x98, 0x86, 0x4b, 0xbe, 0x60, 0x1a, 0x9e,
		0x18, 0x6a, 0x9a, 0x86, 0x25, 0x86, 0x9a, 0xa6, 0x01, 0x62, 0xa8, 0x69, 0x9a, 0xc4, 0x28, 0x5c,
		0x57, 0x76, 0xb3, 0x86, 0x49, 0x87, 0x1d, 0x29, 0x26, 0x25, 0x1e, 0x15, 0x13, 0xd1, 0x96, 0x18,
		0x9d, 0x9c, 0x8b, 0x06, 0x31, 0xba, 0xb8, 0xd2, 0x6a, 0x9f, 0x98, 0x33, 0xa9, 0x76, 0x20, 0xa6,
		0x7a, 0x7f, 0x21, 0x23, 0x4e, 0x9a, 0xaa, 0x5d, 0xf2, 0x85, 0x18, 0x3c, 0x31, 0xd5, 0x6a, 0x07,
		0x4b, 0x4c, 0xf5, 0xfe, 0x02, 0x88, 0xa9, 0xb6, 0x41, 0x4e, 0x9c, 0x86, 0xab, 0xca, 0xce, 0x6b,
		0x98, 0xdc, 0x25, 0x5f, 0x30, 0xb9, 0x27, 0x4e, 0xf5, 0x7c, 0xb9, 0x25, 0x4e, 0x27, 0x4b, 0xd1,
		0x20, 0x4e, 0xd7, 0xbb, 0x0a, 0x13, 0x89, 0x59, 0xb8, 0xa9, 0xec, 0x58, 0xc3, 0x44, 0x97, 0x7c,
		0xc1, 0x44, 0x4f, 0xcc, 0xb4, 0xda, 0xd1, 0x12, 0xb3, 0xa9, 0x74, 0x5c, 0x04, 0x31, 0xbb, 0xfc,
		0x54, 0x61, 0x5a, 0xc4, 0x5c, 0xd3, 0xb4, 0x6a, 0x98, 0x96, 0x4b, 0xbe, 0x60, 0x5a, 0x9e, 0x98,
		0x6b, 0x9a, 0x96, 0x25, 0xe6, 0x9a, 0xa6, 0x05, 0x62, 0xae, 0x69, 0xda, 0xc4, 0x42, 0xef, 0xbe,
		0x5d, 0xc3, 0xb4, 0x5d, 0xf2, 0x05, 0xd3, 0xf6, 0xc4, 0x42, 0xab, 0xdd, 0xb6, 0xc4, 0xe2, 0x44,
		0x6e, 0xa4, 0x0d, 0x62, 0xb1, 0x39, 0xbe, 0xc6, 0xe7, 0xe6, 0x42, 0xaa, 0xdd, 0x21, 0x56, 0xe1,
		0x4b, 0xb5, 0xa3, 0x93, 0x11, 0xe7, 0xfa, 0x1a, 0x77, 0x5c, 0xf2, 0x85, 0xd8, 0x69, 0x10, 0xab,
		0xe2, 0x42, 0xb4, 0x25, 0x56, 0x1b, 0x49, 0xd0, 0x01, 0xb1, 0xda, 0xdf, 0x57, 0x98, 0x2e, 0xb1,
		0x0e, 0x77, 0x95, 0xdd, 0xad, 0x61, 0xba, 0x2e, 0xf9, 0x82, 0xe9, 0x36, 0x88, 0x75, 0x21, 0x57,
		0xdf, 0xb5, 0xc4, 0x7a, 0x25, 0xf5, 0xe8, 0x82, 0x58, 0xdf, 0x7c, 0xd6, 0x60, 0x97, 0xe6, 0x4a,
		0x82, 0xf5, 0x88, 0xad, 0xb6, 0x41, 0x2f, 0x23, 0x2e, 0x9b, 0x12, 0xa0, 0xe7, 0x92, 0x2f, 0xc4,
		0x9e, 0x27, 0xb6, 0xda, 0x06, 0x3d, 0x4b, 0x6c, 0xb5, 0x0d, 0x7a, 0x20, 0xb6, 0xda, 0x06, 0x7d,
		0x62, 0xa7, 0x85, 0xef, 0xd7, 0x30, 0x7d, 0x97, 0x7c, 0xc1, 0xf4, 0x3d, 0xb1, 0xd3, 0xc2, 0xf7,
		0x2d, 0xb1, 0xd3, 0xc2, 0xf7, 0x41, 0xec, 0xd6, 0xbb, 0xf8, 0x39, 0x66, 0xce, 0xd0, 0xdd, 0x98,
		0xff, 0xd9, 0xe9, 0x66, 0xf9, 0xcb, 0xcb, 0xe1, 0x7d, 0xf9, 0xfa, 0x72, 0x38, 0xfc, 0xfa, 0xe7,
		0xcb, 0xf2, 0xe3, 0xf0, 0xf2, 0xc7, 0xf2, 0xb7, 0x7f, 0x96, 0xbf, 0x7f, 0x1c, 0xde, 0xdf, 0x5e,
		0x97, 0x6f, 0x7f, 0xbf, 0xff, 0xf5, 0xf6, 0xfd, 0x10, 0xd3, 0xe3, 0x2e, 0x4d, 0xbc, 0x9b, 0x70,
		0x1a, 0x5b, 0xd1, 0x39, 0x93, 0x19, 0xe2, 0x36, 0x9c, 0xc5, 0x76, 0xf4, 0x69, 0xe1, 0xd2, 0xaa,
		0x7f, 0x5c, 0x79, 0xe2, 0xb6, 0x18, 0x1f, 0x57, 0x96, 0xb8, 0x9d, 0x4c, 0x8f, 0x2b, 0x10, 0xb7,
		0xf3, 0x85, 0x42, 0x2c, 0xb1, 0x0f, 0xe7, 0xfa, 0x53, 0x9a, 0x8b, 0xfb, 0x1f, 0x90, 0x34, 0x19,
		0xf7, 0x3f, 0x20, 0x69, 0x36, 0xee, 0x27, 0xb3, 0xe3, 0x0a, 0xc4, 0xfe, 0x6c, 0x19, 0xf7, 0x31,
		0xf3, 0x86, 0xee, 0xce, 0x7c, 0xb1, 0xd3, 0x95, 0x1c, 0xe4, 0xe5, 0xfb, 0xc7, 0xeb, 0x7f, 0x9d,
		0xc2, 0xa7, 0x38, 0x77, 0x7e, 0x98, 0x02, 0xf8, 0xea, 0x14, 0xf7, 0x81, 0x09, 0xeb, 0x25, 0xe9,
		0x7d, 0xe8, 0x1c, 0x57, 0x19, 0x71, 0xdf, 0x2f, 0xf4, 0x1a, 0xbf, 0x9a, 0xff, 0xcb, 0x35, 0x16,
		0xc4, 0x83, 0x8e, 0x99, 0x22, 0x23, 0xbe, 0x36, 0xe5, 0x5d, 0x2a, 0x5c, 0xf2, 0xa5, 0xfe, 0x85,
		0x27, 0x1e, 0xb4, 0xfe, 0x85, 0x25, 0x1e, 0x74, 0xcc, 0x14, 0x20, 0x1e, 0x74, 0xcc, 0x90, 0x78,
		0x54, 0x0c, 0x6b, 0x18, 0xba, 0xe4, 0x0b, 0x86, 0x9e, 0x78, 0xd4, 0x6e, 0xa0, 0x25, 0x1e, 0xc7,
		0x82, 0x21, 0x88, 0xc7, 0x0a, 0x93, 0x82, 0x7d, 0x4b, 0x9f, 0xbf, 0x44, 0x2c, 0x89, 0x27, 0x25,
		0x96, 0x19, 0xf1, 0x4d, 0x3f, 0x13, 0xa5, 0x4b, 0xbe, 0x10, 0x4b, 0x4f, 0x3c, 0x69, 0xb0, 0xd2,
		0x12, 0x4f, 0x1a, 0xac, 0x04, 0xf1, 0xa4, 0xc1, 0x06, 0xc4, 0xb3, 0x62, 0x06, 0x35, 0xcc, 0xc0,
		0x25, 0x5f, 0x30, 0x03, 0x4f, 0x3c, 0x6b, 0xb0, 0x81, 0x25, 0x9e, 0x35, 0xd8, 0x00, 0xc4, 0xf3,
		0x6a, 0xfb, 0x6f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xf8, 0xef, 0x6c, 0xa2, 0x95, 0x07, 0x00, 0x00,
	}
	sourceinfo.Register("desc_test_options.proto", srcInfo)
}
