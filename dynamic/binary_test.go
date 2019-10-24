package dynamic

import (
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/codec"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestBinaryUnaryFields(t *testing.T) {
	binaryTranslationParty(t, unaryFieldsPosMsg, false)
	binaryTranslationParty(t, unaryFieldsNegMsg, false)
	binaryTranslationParty(t, unaryFieldsPosInfMsg, false)
	binaryTranslationParty(t, unaryFieldsNegInfMsg, false)
	binaryTranslationParty(t, unaryFieldsNanMsg, true)
}

func TestBinaryRepeatedFields(t *testing.T) {
	binaryTranslationParty(t, repeatedFieldsMsg, false)
	binaryTranslationParty(t, repeatedFieldsInfNanMsg, true)
}

func TestBinaryPackedRepeatedFields(t *testing.T) {
	binaryTranslationParty(t, repeatedPackedFieldsMsg, false)
	binaryTranslationParty(t, repeatedPackedFieldsInfNanMsg, true)
}

func TestBinaryMapKeyFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	defaultDeterminism = true
	defer func() {
		defaultDeterminism = false
	}()

	binaryTranslationParty(t, mapKeyFieldsMsg, false)
}

func TestBinaryMapValueFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	defaultDeterminism = true
	defer func() {
		defaultDeterminism = false
	}()

	binaryTranslationParty(t, mapValueFieldsMsg, false)
	binaryTranslationParty(t, mapValueFieldsInfNanMsg, true)
	binaryTranslationParty(t, mapValueFieldsNilMsg, false)
	binaryTranslationParty(t, mapValueFieldsNilUnknownMsg, false)
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
	buf := codec.NewBuffer(b)

	// and unknown fields:
	//   varint encoded field
	_ = buf.EncodeTagAndWireType(1234, proto.WireVarint)
	_ = buf.EncodeVarint(987654)
	//   fixed 64
	_ = buf.EncodeTagAndWireType(2345, proto.WireFixed64)
	_ = buf.EncodeFixed64(123456789)
	//   fixed 32, also repeated
	_ = buf.EncodeTagAndWireType(3456, proto.WireFixed32)
	_ = buf.EncodeFixed32(123456)
	_ = buf.EncodeTagAndWireType(3456, proto.WireFixed32)
	_ = buf.EncodeFixed32(123457)
	_ = buf.EncodeTagAndWireType(3456, proto.WireFixed32)
	_ = buf.EncodeFixed32(123458)
	_ = buf.EncodeTagAndWireType(3456, proto.WireFixed32)
	_ = buf.EncodeFixed32(123459)
	//   length-encoded
	_ = buf.EncodeTagAndWireType(4567, proto.WireBytes)
	_ = buf.EncodeRawBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	//   and... group!
	_ = buf.EncodeTagAndWireType(5678, proto.WireStartGroup)
	{
		_ = buf.EncodeTagAndWireType(1, proto.WireVarint)
		_ = buf.EncodeVarint(1)
		_ = buf.EncodeTagAndWireType(2, proto.WireFixed32)
		_ = buf.EncodeFixed32(2)
		_ = buf.EncodeTagAndWireType(3, proto.WireFixed64)
		_ = buf.EncodeFixed64(3)
		_ = buf.EncodeTagAndWireType(4, proto.WireBytes)
		_ = buf.EncodeRawBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
		// nested group
		_ = buf.EncodeTagAndWireType(5, proto.WireStartGroup)
		{
			_ = buf.EncodeTagAndWireType(1, proto.WireVarint)
			_ = buf.EncodeVarint(1)
			_ = buf.EncodeTagAndWireType(1, proto.WireVarint)
			_ = buf.EncodeVarint(2)
			_ = buf.EncodeTagAndWireType(1, proto.WireVarint)
			_ = buf.EncodeVarint(3)
			_ = buf.EncodeTagAndWireType(2, proto.WireBytes)
			_ = buf.EncodeRawBytes([]byte("lorem ipsum"))
		}
		_ = buf.EncodeTagAndWireType(5, proto.WireEndGroup)
	}
	_ = buf.EncodeTagAndWireType(5678, proto.WireEndGroup)
	testutil.Require(t, buf.Len() > baseLen) // sanity check

	var msg testprotos.TestMessage
	err = proto.Unmarshal(buf.Bytes(), &msg)
	testutil.Ok(t, err)
	// make sure unrecognized fields parsed correctly
	testutil.Eq(t, buf.Bytes()[baseLen:], msg.XXX_unrecognized)

	// make sure dynamic message's round trip generates same bytes
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.TestMessage)(nil))
	testutil.Ok(t, err)
	dm := NewMessage(md)
	err = dm.Unmarshal(buf.Bytes())
	testutil.Ok(t, err)
	bb, err := dm.Marshal()
	testutil.Ok(t, err)
	testutil.Eq(t, buf.Bytes(), bb)

	// now try a full translation party to ensure unknown bits remain correct throughout
	binaryTranslationParty(t, &msg, false)
}

func binaryTranslationParty(t *testing.T, msg proto.Message, includesNaN bool) {
	marshalAppendSimple := func(m *Message) ([]byte, error) {
		// Declare a function that has the same interface as (*Message.Marshal) but uses
		// MarshalAppend internally so we can reuse the translation party tests to verify
		// the behavior of MarshalAppend in addition to Marshal.
		b := make([]byte, 0, 2048)
		marshaledB, err := m.MarshalAppend(b)

		// Verify it doesn't allocate a new byte slice.
		assertByteSlicesBackedBySameData(t, b, marshaledB)
		return marshaledB, err
	}

	marshalAppendPrefix := func(m *Message) ([]byte, error) {
		// Same thing as MarshalAppendSimple, but we verify that prefix data is retained.
		prefix := "prefix"
		marshaledB, err := m.MarshalAppend([]byte(prefix))

		// Verify the prefix data is retained.
		testutil.Eq(t, prefix, string(marshaledB[:len(prefix)]))
		return marshaledB[len(prefix):], err
	}

	marshalMethods := []func(m *Message) ([]byte, error){
		(*Message).Marshal,
		marshalAppendSimple,
		marshalAppendPrefix,
	}

	protoMarshal := func(m proto.Message) ([]byte, error) {
		if defaultDeterminism {
			mm, ok := m.(interface {
				XXX_Size() int
				XXX_Marshal(b []byte, deterministic bool) ([]byte, error)
			})
			if ok {
				bb := make([]byte, 0, mm.XXX_Size())
				return mm.XXX_Marshal(bb, true)
			}

			var buf proto.Buffer
			buf.SetDeterministic(true)
			if err := buf.Marshal(m); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
		return proto.Marshal(m)
	}

	for _, marshalFn := range marshalMethods {
		doTranslationParty(t, msg, protoMarshal, proto.Unmarshal, marshalFn, (*Message).Unmarshal, includesNaN, true, false)
	}
}

// byteSlicesBackedBySameData returns a bool indicating if the raw backing bytes
// under the []byte slice point to the same memory.
func assertByteSlicesBackedBySameData(t *testing.T, a, b []byte) {
	origPtr := reflect.ValueOf(a).Pointer()
	resultPtr := reflect.ValueOf(b).Pointer()
	testutil.Eq(t, origPtr, resultPtr)
}
