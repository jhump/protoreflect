package dynamic

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

var wellKnownTypes = []proto.Message{
	(*wrappers.BoolValue)(nil),
	(*wrappers.BytesValue)(nil),
	(*wrappers.StringValue)(nil),
	(*wrappers.FloatValue)(nil),
	(*wrappers.DoubleValue)(nil),
	(*wrappers.Int32Value)(nil),
	(*wrappers.Int64Value)(nil),
	(*wrappers.UInt32Value)(nil),
	(*wrappers.UInt64Value)(nil),
	(*timestamp.Timestamp)(nil),
	(*duration.Duration)(nil),
	(*any.Any)(nil),
	(*empty.Empty)(nil),
	(*structpb.Struct)(nil),
	(*structpb.Value)(nil),
	(*structpb.ListValue)(nil),
}

func TestKnownTypeRegistry_AddKnownType(t *testing.T) {
	ktr := &KnownTypeRegistry{}
	dp := (*descriptor.DescriptorProto)(nil)
	ktr.AddKnownType(dp)

	checkKnownTypes(t, ktr, wellKnownTypes...)
	checkKnownTypes(t, ktr, dp)
	checkUnknownTypes(t, ktr, (*descriptor.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestKnownTypeRegistry_WithoutWellKnownTypes(t *testing.T) {
	ktr := NewKnownTypeRegistryWithoutWellKnownTypes()
	dp := (*descriptor.DescriptorProto)(nil)
	ktr.AddKnownType(dp)

	checkKnownTypes(t, ktr, dp)
	checkUnknownTypes(t, ktr, wellKnownTypes...)
	checkUnknownTypes(t, ktr, (*descriptor.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestKnownTypeRegistry_WithDefaults(t *testing.T) {
	ktr := NewKnownTypeRegistryWithDefaults()
	dp := (*descriptor.DescriptorProto)(nil)

	// they're all known
	checkKnownTypes(t, ktr, dp)
	checkKnownTypes(t, ktr, (*descriptor.DescriptorProto)(nil), (*descriptor.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func checkKnownTypes(t *testing.T, ktr *KnownTypeRegistry, knownTypes ...proto.Message) {
	for _, kt := range knownTypes {
		md, err := desc.LoadMessageDescriptorForMessage(kt)
		testutil.Ok(t, err)
		m := ktr.CreateIfKnown(md.GetFullyQualifiedName())
		testutil.Require(t, m != nil, "%v should be a known type", reflect.TypeOf(kt))
		testutil.Eq(t, reflect.TypeOf(kt), reflect.TypeOf(m))
	}
}

func checkUnknownTypes(t *testing.T, ktr *KnownTypeRegistry, unknownTypes ...proto.Message) {
	for _, kt := range unknownTypes {
		md, err := desc.LoadMessageDescriptorForMessage(kt)
		testutil.Ok(t, err)
		m := ktr.CreateIfKnown(md.GetFullyQualifiedName())
		testutil.Require(t, m == nil, "%v should not be a known type", reflect.TypeOf(kt))
	}
}

func TestMessageFactory(t *testing.T) {
	mf := &MessageFactory{}

	checkTypes(t, mf, false, wellKnownTypes...)
	checkTypes(t, mf, true, (*descriptor.DescriptorProto)(nil), (*descriptor.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestMessageFactory_WithDefaults(t *testing.T) {
	mf := NewMessageFactoryWithDefaults()

	checkTypes(t, mf, false, wellKnownTypes...)
	checkTypes(t, mf, false, (*descriptor.DescriptorProto)(nil), (*descriptor.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestMessageFactory_WithKnownTypeRegistry(t *testing.T) {
	ktr := NewKnownTypeRegistryWithoutWellKnownTypes()
	mf := NewMessageFactoryWithKnownTypeRegistry(ktr)

	checkTypes(t, mf, true, wellKnownTypes...)
	checkTypes(t, mf, true, (*descriptor.DescriptorProto)(nil), (*descriptor.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func checkTypes(t *testing.T, mf *MessageFactory, dynamic bool, types ...proto.Message) {
	for _, typ := range types {
		md, err := desc.LoadMessageDescriptorForMessage(typ)
		testutil.Ok(t, err)
		m := mf.NewMessage(md)
		if dynamic {
			testutil.Eq(t, typeOfDynamicMessage, reflect.TypeOf(m))
		} else {
			testutil.Eq(t, reflect.TypeOf(typ), reflect.TypeOf(m))
		}
	}

}
