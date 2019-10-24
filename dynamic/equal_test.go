package dynamic

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func eqm(a, b interface{}) bool {
	return MessagesEqual(a.(proto.Message), b.(proto.Message))
}

func eqdm(a, b interface{}) bool {
	return Equal(a.(*Message), b.(*Message))
}

func eqpm(a, b interface{}) bool {
	return proto.Equal(a.(proto.Message), b.(proto.Message))
}

func TestEquals(t *testing.T) {
	mdProto3, err := desc.LoadMessageDescriptorForMessage((*testprotos.TestRequest)(nil))
	testutil.Ok(t, err)

	dm1 := NewMessage(mdProto3)
	dm2 := NewMessage(mdProto3)
	checkEquals(t, dm1, dm2) // sanity check

	dm1.SetFieldByName("foo", []testprotos.Proto3Enum{testprotos.Proto3Enum_VALUE1})
	dm1.SetFieldByName("bar", "barfbag")
	dm1.SetFieldByName("flags", map[string]bool{"a": true, "b": false, "c": true})

	checkNotEquals(t, dm1, dm2)

	dm2.SetFieldByName("foo", []testprotos.Proto3Enum{testprotos.Proto3Enum_VALUE1})
	dm2.SetFieldByName("bar", "barfbag")
	dm2.SetFieldByName("flags", map[string]bool{"a": true, "b": false, "c": true})

	checkEquals(t, dm1, dm2)

	// With proto3, setting fields to zero value is not distinguishable from absent fields
	dm1.Reset()
	dm2.Reset()
	dm1.SetFieldByName("foo", []testprotos.Proto3Enum{})
	dm1.SetFieldByName("bar", "")
	dm1.SetFieldByName("flags", map[string]bool{})

	checkEquals(t, dm1, dm2)

	// Now check proto2 messages
	mdProto2, err := desc.LoadMessageDescriptorForMessage((*testprotos.UnaryFields)(nil))
	testutil.Ok(t, err)

	dm1 = NewMessage(mdProto2)
	dm2 = NewMessage(mdProto2)
	checkEquals(t, dm1, dm2) // sanity check

	dm1.SetFieldByName("i", int32(123))
	dm1.SetFieldByName("v", "blueberry")

	checkNotEquals(t, dm1, dm2)

	dm2.SetFieldByName("i", int32(123))
	dm2.SetFieldByName("v", "blueberry")

	checkEquals(t, dm1, dm2)

	// In proto2, however, we can distinguish between present and zero/default values
	dm1.Reset()
	dm2.Reset()
	dm1.SetFieldByName("i", int32(0))
	dm1.SetFieldByName("v", "")

	checkNotEquals(t, dm1, dm2)

	// But, even in proto2, empty repeated and map fields are indistinguishable from absent fields
	mdProto2, err = desc.LoadMessageDescriptorForMessage((*testprotos.RepeatedFields)(nil))
	testutil.Ok(t, err)

	dm1 = NewMessage(mdProto2)
	dm2 = NewMessage(mdProto2)
	checkEquals(t, dm1, dm2) // sanity check

	dm1.SetFieldByName("i", []int32{})
	dm1.SetFieldByName("v", []string{})

	checkEquals(t, dm1, dm2)

	mdProto2, err = desc.LoadMessageDescriptorForMessage((*testprotos.MapValFields)(nil))
	testutil.Ok(t, err)

	dm1 = NewMessage(mdProto2)
	dm2 = NewMessage(mdProto2)
	checkEquals(t, dm1, dm2) // sanity check

	dm1.SetFieldByName("i", map[string]int32{})
	dm1.SetFieldByName("v", map[string]string{})

	checkEquals(t, dm1, dm2)
}

func checkEquals(t *testing.T, a, b *Message) {
	testutil.Ceq(t, a, b, eqdm)
	testutil.Ceq(t, a, b, eqm)
	testutil.Ceq(t, b, a, eqdm)
	testutil.Ceq(t, b, a, eqm)

	// and then compare generated message type to dynamic message
	msgType := proto.MessageType(a.GetMessageDescriptor().GetFullyQualifiedName())
	msg := reflect.New(msgType.Elem()).Interface().(proto.Message)
	err := a.ConvertTo(msg)
	testutil.Ok(t, err)
	testutil.Ceq(t, a, msg, eqm)
	testutil.Ceq(t, msg, a, eqm)
}

func checkNotEquals(t *testing.T, a, b *Message) {
	testutil.Cneq(t, a, b, eqdm)
	testutil.Cneq(t, a, b, eqm)
	testutil.Cneq(t, b, a, eqdm)
	testutil.Cneq(t, b, a, eqm)

	// and then compare generated message type to dynamic message
	msgType := proto.MessageType(a.GetMessageDescriptor().GetFullyQualifiedName())
	msg := reflect.New(msgType.Elem()).Interface().(proto.Message)
	err := a.ConvertTo(msg)
	testutil.Ok(t, err)
	testutil.Cneq(t, b, msg, eqm)
	testutil.Cneq(t, msg, b, eqm)
}
