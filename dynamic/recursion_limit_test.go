package dynamic

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/jhump/protoreflect/codec"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testutil"
)

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

func TestMarshalRecursionLimit(t *testing.T) {
	md := recursiveMessageDescriptor(t)

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
	formats := map[string]func(*Message) ([]byte, error){
		"binary":               (*Message).Marshal,
		"binary-deterministic": (*Message).MarshalDeterministic,
		"json":                 (*Message).MarshalJSON,
		"text":                 (*Message).MarshalText,
	}
	for linkName, link := range links {
		// the top-level message counts towards the limit
		tooDeep := nestedMessage(md, protowire.DefaultRecursionLimit+1, link)
		// a cycle must also result in an error, not a stack overflow
		cyclic := NewMessage(md)
		link(cyclic, cyclic)

		for formatName, marshal := range formats {
			t.Run(linkName+"/"+formatName, func(t *testing.T) {
				_, err := marshal(nestedMessage(md, 100, link))
				testutil.Ok(t, err)

				_, err = marshal(tooDeep)
				testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)

				_, err = marshal(cyclic)
				testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)
			})
		}
	}
}

func TestJSONUnmarshalRecursionLimit(t *testing.T) {
	md := recursiveMessageDescriptor(t)

	// steps is the number of times open and close are repeated
	nestedJSON := func(steps int, open, close string) []byte {
		return []byte(strings.Repeat(open, steps) + "{}" + strings.Repeat(close, steps))
	}
	// how many levels of JSON nesting (objects and arrays) each step adds
	jsonLevelsPerStep := map[string]int{"field": 1, "repeated": 2, "map": 2}
	for _, tc := range nestingCases(map[string][2]string{
		"field":    {`{"child":`, `}`},
		"repeated": {`{"children":[`, `]}`},
		"map":      {`{"byId":{"1":`, `}}`},
	}) {
		t.Run(tc.name, func(t *testing.T) {
			// The JSON nesting must also be within encoding/json's limit, which
			// applies in newer versions of Go. For "field" and "map", that is
			// the same, so this tests the exact limit.
			steps := min(tc.maxSteps, (maxJSONNesting-1)/jsonLevelsPerStep[tc.name])
			msg := NewMessage(md)
			testutil.Ok(t, msg.UnmarshalJSON(nestedJSON(steps, tc.open, tc.close)))

			err := NewMessage(md).UnmarshalJSON(nestedJSON(tc.maxSteps+1, tc.open, tc.close))
			testutil.Require(t, isJSONDepthError(err), "unexpected error: %v", err)

			// this used to crash the process
			err = NewMessage(md).UnmarshalJSON(nestedJSON(500_000, tc.open, tc.close))
			testutil.Require(t, isJSONDepthError(err), "unexpected error: %v", err)
		})
	}
}

func TestJSONUnmarshalDeeplyNestedUnknownField(t *testing.T) {
	md := recursiveMessageDescriptor(t)
	opts := &jsonpb.Unmarshaler{AllowUnknownFields: true}
	nestedUnknown := func(depth int) []byte {
		return []byte(`{"unknown":` + strings.Repeat(`[`, depth) + strings.Repeat(`]`, depth) + `}`)
	}

	testutil.Ok(t, NewMessage(md).UnmarshalJSONPB(opts, nestedUnknown(10)))
	// skipping deeply nested values used to recurse and could crash the process
	err := NewMessage(md).UnmarshalJSONPB(opts, nestedUnknown(500_000))
	testutil.Require(t, isJSONDepthError(err), "unexpected error: %v", err)
}

// maxJSONNesting is the maximum depth of JSON objects and arrays allowed by
// encoding/json. Starting with Go 1.27, this is enforced even when reading
// tokens with json.Decoder.Token, as the dynamic package does.
const maxJSONNesting = 10000

// isJSONDepthError returns true if err indicates that JSON input was nested
// too deeply. Starting with Go 1.27, encoding/json checks the depth of
// JSON nesting itself, which may happen before the dynamic package checks
// the depth of message nesting.
func isJSONDepthError(err error) bool {
	if errors.Is(err, codec.ErrRecursionDepth) {
		return true
	}
	var syntaxErr *json.SyntaxError
	return errors.As(err, &syntaxErr) && strings.Contains(syntaxErr.Error(), "exceeded max depth")
}

func TestJSONMarshalRecursionLimitThroughAny(t *testing.T) {
	md := recursiveMessageDescriptor(t)

	// Builds a chain of the given number of hops, where each hop is a Recurse
	// whose "any" field contains another Recurse. Marshaling an Any to JSON is
	// done by jsonpb, which calls back into the dynamic message for its value.
	anyChain := func(hops int) *Message {
		msg := NewMessage(md)
		for i := 0; i < hops; i++ {
			data, err := msg.Marshal()
			testutil.Ok(t, err)
			parent := NewMessage(md)
			parent.SetFieldByNumber(6, &anypb.Any{TypeUrl: "type.googleapis.com/test.Recurse", Value: data})
			msg = parent
		}
		return msg
	}
	const hops = 5
	setChild := func(parent, child *Message) {
		parent.SetFieldByNumber(1, child)
	}

	// The chain is attached as the child of the deepest message, and each hop
	// adds two levels: the Any and the Recurse inside it. So the last Recurse
	// is at the maximum depth.
	maxDepth := protowire.DefaultRecursionLimit - 1
	msg := nestedMessage(md, maxDepth-2*hops, setChild)
	setLeaf(msg, anyChain(hops), setChild)
	js, err := msg.MarshalJSON()
	testutil.Ok(t, err)
	testutil.Require(t, bytes.Contains(js, []byte(`"@type":"type.googleapis.com/test.Recurse"`)))

	// The depth must be carried through jsonpb: without it, each Any would
	// start counting over, and a long chain of them could crash the process.
	msg = nestedMessage(md, maxDepth-2*hops+1, setChild)
	setLeaf(msg, anyChain(hops), setChild)
	_, err = msg.MarshalJSON()
	testutil.Require(t, err != nil && strings.Contains(err.Error(), codec.ErrRecursionDepth.Error()), "unexpected error: %v", err)
}

func TestTextUnmarshalRecursionLimit(t *testing.T) {
	md := recursiveMessageDescriptor(t)

	// steps is the number of times open and close are repeated
	nestedText := func(steps int, open, close string) []byte {
		return []byte(strings.Repeat(open, steps) + strings.Repeat(close, steps))
	}
	for _, tc := range nestingCases(map[string][2]string{
		"braces":   {`child{`, `}`},
		"angles":   {`child:<`, `>`},
		"repeated": {`children:[{`, `}]`},
		"map":      {`by_id:{key:1 value:{`, `}}`},
	}) {
		t.Run(tc.name, func(t *testing.T) {
			msg := NewMessage(md)
			testutil.Ok(t, msg.UnmarshalText(nestedText(tc.maxSteps, tc.open, tc.close)))

			err := NewMessage(md).UnmarshalText(nestedText(tc.maxSteps+1, tc.open, tc.close))
			testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)

			// this used to crash the process
			err = NewMessage(md).UnmarshalText(nestedText(500_000, tc.open, tc.close))
			testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)
		})
	}
}

func TestTextUnmarshalDeeplyNestedUnknownField(t *testing.T) {
	md := recursiveMessageDescriptor(t)
	nestedUnknown := func(depth int) []byte {
		return []byte(strings.Repeat(`99{`, depth) + strings.Repeat(`}`, depth))
	}

	testutil.Ok(t, NewMessage(md).UnmarshalText(nestedUnknown(10)))
	// skipping deeply nested values used to recurse and could crash the process
	err := NewMessage(md).UnmarshalText(nestedUnknown(500_000))
	testutil.Require(t, errors.Is(err, codec.ErrRecursionDepth), "unexpected error: %v", err)
}

type nestingCase struct {
	name        string
	open, close string
	// maxSteps is the most times that open and close can be repeated without
	// exceeding the limit
	maxSteps int
}

// nestingCases returns test cases for the given delimiters, keyed by name.
// Each repetition of the delimiters nests one more message, except for the
// "map" case, where it also nests a map entry.
func nestingCases(delims map[string][2]string) []nestingCase {
	var cases []nestingCase
	for name, d := range delims {
		// the top-level message counts towards the limit
		maxSteps := protowire.DefaultRecursionLimit - 1
		if name == "map" {
			maxSteps /= 2
		}
		cases = append(cases, nestingCase{name: name, open: d[0], close: d[1], maxSteps: maxSteps})
	}
	return cases
}

// nestedMessage returns a message with the given depth of nested messages
// (including itself), using link to make child the child of parent.
func nestedMessage(md *desc.MessageDescriptor, depth int, link func(parent, child *Message)) *Message {
	msg := NewMessage(md)
	for i := 1; i < depth; i++ {
		parent := NewMessage(md)
		link(parent, msg)
		msg = parent
	}
	return msg
}

// setLeaf follows the "child" field of msg to the deepest message and then
// uses link to make leaf its child.
func setLeaf(msg, leaf *Message, link func(parent, child *Message)) {
	for msg.HasFieldNumber(1) {
		msg = msg.GetFieldByNumber(1).(*Message)
	}
	link(msg, leaf)
}

// recursiveMessageDescriptor returns a descriptor for the following:
//
//	message Recurse {
//	  optional Recurse child = 1;
//	  repeated Recurse children = 2;
//	  map<int32, Recurse> by_id = 3;
//	  google.protobuf.Value value = 4;
//	  google.protobuf.UninterpretedOption.NamePart name_part = 5;
//	  google.protobuf.Any any = 6;
//	}
func recursiveMessageDescriptor(t *testing.T) *desc.MessageDescriptor {
	t.Helper()
	fdp := &descriptorpb.FileDescriptorProto{
		Name:    proto.String("recurse.proto"),
		Package: proto.String("test"),
		Dependency: []string{
			"google/protobuf/struct.proto",
			"google/protobuf/descriptor.proto",
			"google/protobuf/any.proto",
		},
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
				{
					Name:     proto.String("any"),
					Number:   proto.Int32(6),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".google.protobuf.Any"),
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
	anyFile, err := desc.LoadFileDescriptor("google/protobuf/any.proto")
	testutil.Ok(t, err)
	fd, err := desc.CreateFileDescriptor(fdp, structFile, descriptorFile, anyFile)
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
