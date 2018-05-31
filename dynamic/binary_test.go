package dynamic

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestBinaryUnaryFields(t *testing.T) {
	binaryTranslationParty(t, unaryFieldsPosMsg)
	binaryTranslationParty(t, unaryFieldsNegMsg)
}

func TestBinaryRepeatedFields(t *testing.T) {
	binaryTranslationParty(t, repeatedFieldsMsg)
}

func TestBinaryPackedRepeatedFields(t *testing.T) {
	binaryTranslationParty(t, repeatedPackedFieldsMsg)
}

func TestBinaryMapKeyFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	defaultDeterminism = true
	defer func() {
		defaultDeterminism = false
	}()

	binaryTranslationParty(t, mapKeyFieldsMsg)
}

func TestMarshalMapValueFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	defaultDeterminism = true
	defer func() {
		defaultDeterminism = false
	}()

	binaryTranslationParty(t, mapValueFieldsMsg)
}

func TestBinaryExtensionFields(t *testing.T) {
	// TODO
}

func TestBinaryUnknownFields(t *testing.T) {
	// create a buffer with both known fields:
	b, err := proto.Marshal(&testprotos.TestMessage{
		Nm: &testprotos.TestMessage_NestedMessage{
			Anm: &testprotos.TestMessage_NestedMessage_AnotherNestedMessage{
				Yanm: []*testprotos.TestMessage_NestedMessage_AnotherNestedMessage_YetAnotherNestedMessage{
					{Foo: proto.String("foo"), Bar: proto.Int32(100), Baz: []byte{1, 2, 3, 4}},
				},
			}},
		Ne: []testprotos.TestMessage_NestedEnum{testprotos.TestMessage_VALUE1, testprotos.TestMessage_VALUE1},
	})
	baseLen := len(b)
	testutil.Ok(t, err)
	buf := newCodedBuffer(b)

	// and unknown fields:
	//   varint encoded field
	buf.encodeTagAndWireType(1234, proto.WireVarint)
	buf.encodeVarint(987654)
	//   fixed 64
	buf.encodeTagAndWireType(2345, proto.WireFixed64)
	buf.encodeFixed64(123456789)
	//   fixed 32, also repeated
	buf.encodeTagAndWireType(3456, proto.WireFixed32)
	buf.encodeFixed32(123456)
	buf.encodeTagAndWireType(3456, proto.WireFixed32)
	buf.encodeFixed32(123457)
	buf.encodeTagAndWireType(3456, proto.WireFixed32)
	buf.encodeFixed32(123458)
	buf.encodeTagAndWireType(3456, proto.WireFixed32)
	buf.encodeFixed32(123459)
	//   length-encoded
	buf.encodeTagAndWireType(4567, proto.WireBytes)
	buf.encodeRawBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	//   and... group!
	buf.encodeTagAndWireType(5678, proto.WireStartGroup)
	{
		buf.encodeTagAndWireType(1, proto.WireVarint)
		buf.encodeVarint(1)
		buf.encodeTagAndWireType(2, proto.WireFixed32)
		buf.encodeFixed32(2)
		buf.encodeTagAndWireType(3, proto.WireFixed64)
		buf.encodeFixed64(3)
		buf.encodeTagAndWireType(4, proto.WireBytes)
		buf.encodeRawBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
		// nested group
		buf.encodeTagAndWireType(5, proto.WireStartGroup)
		{
			buf.encodeTagAndWireType(1, proto.WireVarint)
			buf.encodeVarint(1)
			buf.encodeTagAndWireType(1, proto.WireVarint)
			buf.encodeVarint(2)
			buf.encodeTagAndWireType(1, proto.WireVarint)
			buf.encodeVarint(3)
			buf.encodeTagAndWireType(2, proto.WireBytes)
			buf.encodeRawBytes([]byte("lorem ipsum"))
		}
		buf.encodeTagAndWireType(5, proto.WireEndGroup)
	}
	buf.encodeTagAndWireType(5678, proto.WireEndGroup)
	testutil.Require(t, len(buf.buf) > baseLen) // sanity check

	var msg testprotos.TestMessage
	err = proto.Unmarshal(buf.buf, &msg)
	testutil.Ok(t, err)
	// make sure unrecognized fields parsed correctly
	testutil.Eq(t, buf.buf[baseLen:], msg.XXX_unrecognized)

	// make sure dynamic message's round trip generates same bytes
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.TestMessage)(nil))
	testutil.Ok(t, err)
	dm := NewMessage(md)
	err = dm.Unmarshal(buf.buf)
	testutil.Ok(t, err)
	bb, err := dm.Marshal()
	testutil.Ok(t, err)
	testutil.Eq(t, buf.buf, bb)

	// now try a full translation party to ensure unknown bits remain correct throughout
	binaryTranslationParty(t, &msg)
}

func binaryTranslationParty(t *testing.T, msg proto.Message) {
	doTranslationParty(t, msg, proto.Marshal, proto.Unmarshal, (*Message).Marshal, (*Message).Unmarshal)
}
