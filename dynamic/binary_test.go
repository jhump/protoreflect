package dynamic

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/structpb"

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
	testutil.Eq(t, buf.Bytes()[baseLen:], []byte(msg.ProtoReflect().GetUnknown()))

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

func TestBinaryUnmarshalRecursionLimit(t *testing.T) {
	md := recursiveMessageDescriptor(t)

	for _, tag := range []byte{
		0x0a, // optional Recurse child = 1
		0x12, // repeated Recurse children = 2
	} {
		// the top-level message counts towards the limit
		data := nestedMessagePayload(tag, protowire.DefaultRecursionLimit-1)
		msg := NewMessage(md)
		testutil.Ok(t, msg.Unmarshal(data))
		// and we can round-trip it
		roundTripped, err := msg.Marshal()
		testutil.Ok(t, err)
		testutil.Eq(t, data, roundTripped)

		data = nestedMessagePayload(tag, protowire.DefaultRecursionLimit)
		err = NewMessage(md).Unmarshal(data)
		testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)

		// the case from the original report: this used to crash the process
		data = nestedMessagePayload(tag, 500_000)
		err = NewMessage(md).Unmarshal(data)
		testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)
	}
}

func TestBinaryUnmarshalRecursionLimitWithNonDynamicMessages(t *testing.T) {
	md := recursiveMessageDescriptor(t)

	// Recurse.value is a google.protobuf.Value, which is a generated message
	// and is recursive: Value.list_value (6) -> ListValue.values (1) -> Value
	payload := func(dynamicDepth, valueDepth int) []byte {
		tags := bytes.Repeat([]byte{0x0a}, dynamicDepth-1) // Recurse.child
		tags = append(tags, 0x22)                          // Recurse.value
		for i := 1; i < valueDepth; i++ {
			if i%2 == 1 {
				tags = append(tags, 0x32) // Value.list_value
			} else {
				tags = append(tags, 0x0a) // ListValue.values
			}
		}
		return nestedPayload(tags)
	}

	limit := protowire.DefaultRecursionLimit
	// Generated messages are decoded by the protobuf runtime, with its own
	// limit, so the worst case is both limits combined. That must still work.
	msg := NewMessage(md)
	testutil.Ok(t, msg.Unmarshal(payload(limit-1, limit)))
	// make sure the leaf really was decoded into the generated type
	for i := 1; i < limit-1; i++ {
		msg = msg.GetFieldByNumber(1).(*Message)
	}
	_, isGenerated := msg.GetFieldByNumber(4).(*structpb.Value)
	testutil.Require(t, isGenerated)

	// but the runtime still enforces its limit
	err := NewMessage(md).Unmarshal(payload(2, limit+1))
	testutil.Require(t, err != nil && strings.Contains(err.Error(), "recursion depth"), "unexpected error: %v", err)
}

func TestBinaryUnmarshalNestedGeneratedMessageMissingRequiredFields(t *testing.T) {
	md := recursiveMessageDescriptor(t)
	mf := NewMessageFactoryWithKnownTypeRegistry(NewKnownTypeRegistryWithDefaults())

	// Recurse.name_part is an empty google.protobuf.UninterpretedOption.NamePart,
	// which is a generated message with required fields
	data := []byte{0x2a, 0x00}
	err := mf.NewDynamicMessage(md).Unmarshal(data)
	// the error must be the same type as returned by the v1 proto.Unmarshal
	var reqErr *proto.RequiredNotSetError
	testutil.Require(t, errors.As(err, &reqErr), "unexpected error: %v", err)
}

func TestBinaryUnmarshalDeeplyNestedUnknownGroups(t *testing.T) {
	md := recursiveMessageDescriptor(t)

	// field 99, wire type start-group and end-group
	startGroup, endGroup := []byte{0x9b, 0x06}, []byte{0x9c, 0x06}
	nestedGroups := func(depth int) []byte {
		return append(bytes.Repeat(startGroup, depth), bytes.Repeat(endGroup, depth)...)
	}

	data := nestedGroups(10)
	msg := NewMessage(md)
	testutil.Ok(t, msg.Unmarshal(data))
	roundTripped, err := msg.Marshal()
	testutil.Ok(t, err)
	testutil.Eq(t, data, roundTripped)

	// skipping deeply nested groups used to recurse and could crash the process
	testutil.Nok(t, NewMessage(md).Unmarshal(nestedGroups(500_000)))
}

func TestBinaryMarshalRecursionLimit(t *testing.T) {
	md := recursiveMessageDescriptor(t)

	nested := func(depth int, link func(parent, child *Message)) *Message {
		msg := NewMessage(md)
		for i := 1; i < depth; i++ {
			parent := NewMessage(md)
			link(parent, msg)
			msg = parent
		}
		return msg
	}
	links := map[string]func(parent, child *Message){
		"field": func(parent, child *Message) {
			parent.SetFieldByNumber(1, child)
		},
		"repeated": func(parent, child *Message) {
			parent.AddRepeatedFieldByNumber(2, child)
		},
		"map": func(parent, child *Message) {
			parent.PutMapFieldByNumber(3, int32(1), child)
		},
	}
	for name, link := range links {
		t.Run(name, func(t *testing.T) {
			_, err := nested(100, link).Marshal()
			testutil.Ok(t, err)

			_, err = nested(protowire.DefaultRecursionLimit+1, link).Marshal()
			testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)

			// a cycle must also result in an error, not a stack overflow
			cyclic := NewMessage(md)
			link(cyclic, cyclic)
			_, err = cyclic.Marshal()
			testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)
			_, err = cyclic.MarshalDeterministic()
			testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)
		})
	}
}

// recursiveMessageDescriptor returns a descriptor for the following:
//
//	message Recurse {
//	  optional Recurse child = 1;
//	  repeated Recurse children = 2;
//	  map<int32, Recurse> by_id = 3;
//	  google.protobuf.Value value = 4;
//	  google.protobuf.UninterpretedOption.NamePart name_part = 5;
//	}
func recursiveMessageDescriptor(t *testing.T) *desc.MessageDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("recurse.proto"),
		Package:    proto.String("test"),
		Dependency: []string{"google/protobuf/struct.proto", "google/protobuf/descriptor.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{
			Name: proto.String("Recurse"),
			Field: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     proto.String("child"),
					Number:   proto.Int32(1),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".test.Recurse"),
				},
				{
					Name:     proto.String("children"),
					Number:   proto.Int32(2),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".test.Recurse"),
				},
				{
					Name:     proto.String("by_id"),
					Number:   proto.Int32(3),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".test.Recurse.ByIdEntry"),
				},
				{
					Name:     proto.String("value"),
					Number:   proto.Int32(4),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".google.protobuf.Value"),
				},
				{
					Name:     proto.String("name_part"),
					Number:   proto.Int32(5),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".google.protobuf.UninterpretedOption.NamePart"),
				},
			},
			NestedType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("ByIdEntry"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   proto.String("key"),
						Number: proto.Int32(1),
						Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
					},
					{
						Name:     proto.String("value"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".test.Recurse"),
					},
				},
				Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
			}},
		}},
	}
	structFile, err := desc.LoadFileDescriptor("google/protobuf/struct.proto")
	testutil.Ok(t, err)
	descriptorFile, err := desc.LoadFileDescriptor("google/protobuf/descriptor.proto")
	testutil.Ok(t, err)
	fd, err := desc.CreateFileDescriptor(fdp, structFile, descriptorFile)
	testutil.Ok(t, err)
	return fd.FindMessage("test.Recurse")
}

// nestedMessagePayload returns the binary encoding of a message with the given
// depth of nested messages, each in the field indicated by the given tag byte.
func nestedMessagePayload(tag byte, depth int) []byte {
	return nestedPayload(bytes.Repeat([]byte{tag}, depth))
}

// nestedPayload returns the binary encoding of nested messages, where tags[i]
// is the tag byte for the length-delimited field at the i-th level of nesting.
func nestedPayload(tags []byte) []byte {
	lengths := make([]int, len(tags))
	size := 0
	for i := len(tags) - 1; i >= 0; i-- {
		lengths[i] = size
		size += 1 + protowire.SizeVarint(uint64(size))
	}
	out := make([]byte, 0, size)
	for i, tag := range tags {
		out = append(out, tag)
		out = protowire.AppendVarint(out, uint64(lengths[i]))
	}
	return out
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
