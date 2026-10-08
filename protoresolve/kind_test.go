package protoresolve_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestKindOf(t *testing.T) {
	t.Parallel()
	file := testprotos.File_desc_test1_proto
	msg := file.Messages().ByName("AnotherTestMessage")
	enum := file.Enums().ByName("SomeEnum")
	service := file.Services().ByName("SomeService")
	testCases := []struct {
		descriptor protoreflect.Descriptor
		kind       protoresolve.DescriptorKind
	}{
		{file, protoresolve.DescriptorKindFile},
		{msg, protoresolve.DescriptorKindMessage},
		{msg.Fields().Get(0), protoresolve.DescriptorKindField},
		{msg.Oneofs().Get(0), protoresolve.DescriptorKindOneof},
		{enum, protoresolve.DescriptorKindEnum},
		{enum.Values().Get(0), protoresolve.DescriptorKindEnumValue},
		{file.Extensions().Get(0), protoresolve.DescriptorKindExtension},
		{service, protoresolve.DescriptorKindService},
		{service.Methods().Get(0), protoresolve.DescriptorKindMethod},
		{nil, protoresolve.DescriptorKindUnknown},
	}
	for _, testCase := range testCases {
		assert.Equal(t, testCase.kind, protoresolve.KindOf(testCase.descriptor), "kind of %v", testCase.descriptor)
	}
}

func TestDescriptorKindString(t *testing.T) {
	t.Parallel()
	expected := map[protoresolve.DescriptorKind]string{
		protoresolve.DescriptorKindUnknown:   "unknown",
		protoresolve.DescriptorKindFile:      "file",
		protoresolve.DescriptorKindMessage:   "message",
		protoresolve.DescriptorKindField:     "field",
		protoresolve.DescriptorKindOneof:     "oneof",
		protoresolve.DescriptorKindEnum:      "enum",
		protoresolve.DescriptorKindEnumValue: "enum value",
		protoresolve.DescriptorKindExtension: "extension",
		protoresolve.DescriptorKindService:   "service",
		protoresolve.DescriptorKindMethod:    "method",
		protoresolve.DescriptorKind(100):     "unknown kind (100)",
	}
	for kind, str := range expected {
		assert.Equal(t, str, kind.String())
	}
}
