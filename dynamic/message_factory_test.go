package dynamic

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

var wellKnownTypes = []proto.Message{
	(*wrapperspb.BoolValue)(nil),
	(*wrapperspb.BytesValue)(nil),
	(*wrapperspb.StringValue)(nil),
	(*wrapperspb.FloatValue)(nil),
	(*wrapperspb.DoubleValue)(nil),
	(*wrapperspb.Int32Value)(nil),
	(*wrapperspb.Int64Value)(nil),
	(*wrapperspb.UInt32Value)(nil),
	(*wrapperspb.UInt64Value)(nil),
	(*timestamppb.Timestamp)(nil),
	(*durationpb.Duration)(nil),
	(*anypb.Any)(nil),
	(*emptypb.Empty)(nil),
	(*structpb.Struct)(nil),
	(*structpb.Value)(nil),
	(*structpb.ListValue)(nil),
}

func TestKnownTypeRegistry_AddKnownType(t *testing.T) {
	ktr := &KnownTypeRegistry{}
	dp := (*descriptorpb.DescriptorProto)(nil)
	ktr.AddKnownType(dp)

	checkKnownTypes(t, ktr, wellKnownTypes...)
	checkKnownTypes(t, ktr, dp)
	checkUnknownTypes(t, ktr, (*descriptorpb.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestKnownTypeRegistry_WithoutWellKnownTypes(t *testing.T) {
	ktr := NewKnownTypeRegistryWithoutWellKnownTypes()
	dp := (*descriptorpb.DescriptorProto)(nil)
	ktr.AddKnownType(dp)

	checkKnownTypes(t, ktr, dp)
	checkUnknownTypes(t, ktr, wellKnownTypes...)
	checkUnknownTypes(t, ktr, (*descriptorpb.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestKnownTypeRegistry_WithDefaults(t *testing.T) {
	ktr := NewKnownTypeRegistryWithDefaults()
	dp := (*descriptorpb.DescriptorProto)(nil)

	// they're all known
	checkKnownTypes(t, ktr, dp)
	checkKnownTypes(t, ktr, (*descriptorpb.DescriptorProto)(nil), (*descriptorpb.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestKnownTypeRegistry_WithDefaults_MapEntry(t *testing.T) {
	ktr := NewKnownTypeRegistryWithDefaults()
	msgType := ktr.GetKnownType("testprotos.MapKeyFields.SEntry")
	testutil.Require(t, msgType == nil, "should not be a known type for map entry but got %v", msgType)
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
	checkTypes(t, mf, true, (*descriptorpb.DescriptorProto)(nil), (*descriptorpb.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestMessageFactory_WithDefaults(t *testing.T) {
	mf := NewMessageFactoryWithDefaults()

	checkTypes(t, mf, false, wellKnownTypes...)
	checkTypes(t, mf, false, (*descriptorpb.DescriptorProto)(nil), (*descriptorpb.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
}

func TestMessageFactory_WithKnownTypeRegistry(t *testing.T) {
	ktr := NewKnownTypeRegistryWithoutWellKnownTypes()
	mf := NewMessageFactoryWithKnownTypeRegistry(ktr)

	checkTypes(t, mf, true, wellKnownTypes...)
	checkTypes(t, mf, true, (*descriptorpb.DescriptorProto)(nil), (*descriptorpb.FileDescriptorProto)(nil), (*testprotos.TestMessage)(nil))
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
