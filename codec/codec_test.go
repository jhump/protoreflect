package codec_test

import (
	"github.com/jhump/protoreflect/codec"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestEncodeMessage(t *testing.T) {
	// A generated message will be encoded using its XXX_Size and XXX_Marshal
	// methods
	pm := &testprotos.Test{
		Foo:   proto.String("bar"),
		Array: []int32{0, 1, 2, 3},
		S: &testprotos.Simple{
			Name: proto.String("baz"),
			Id:   proto.Uint64(12345),
		},
		M: map[string]int32{
			"a": 1,
			"b": 2,
			"c": 3,
			"d": 4,
		},
		B: []byte{3, 2, 1, 0},
	}

	// A generated message will be encoded using its MarshalAppend and
	// MarshalAppendDeterministic methods
	md, err := desc.LoadMessageDescriptorForMessage(pm)
	testutil.Ok(t, err)
	dm := dynamic.NewMessage(md)
	err = dm.ConvertFrom(pm)
	testutil.Ok(t, err)

	// This custom message will use MarshalDeterministic method or fall back to
	// old proto.Marshal implementation for non-deterministic marshaling
	cm := (*TestMessage)(pm)

	testCases := []struct{
		Name string
		Msg  proto.Message
	}{
		{Name: "generated", Msg: pm},
		{Name: "dynamic", Msg: dm},
		{Name: "custom", Msg: cm},
	}

	var bytes []byte

	t.Run("deterministic", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				var cb codec.Buffer
				err := cb.EncodeMessageDeterministic(tc.Msg)
				testutil.Ok(t, err)
				b := cb.Bytes()
				if bytes == nil {
					bytes = b
				} else {
					// The generated proto message is the benchmark.
					// Ensure that the others match its output.
					testutil.Eq(t, bytes, b)
				}
			})
		}
	})

	t.Run("non-deterministic", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.Name, func(t *testing.T) {
				var cb codec.Buffer
				err := cb.EncodeMessage(tc.Msg)
				testutil.Ok(t, err)

				l, err := cb.DecodeVarint()
				testutil.Ok(t, err)
				b := cb.Bytes()
				testutil.Eq(t, int(l), len(b))
				// we can't compare byte slices to benchmark since the
				// message contains a map and we are not using deterministic
				// marshal method; so verify that unmarshaling the bytes
				// results in an equal message as the original
				var pm2 testprotos.Test
				err = proto.Unmarshal(b, &pm2)
				testutil.Ok(t, err)

				testutil.Require(t, proto.Equal(pm, &pm2))
			})
		}
	})
}

// NB: other field types are well-exercised by dynamic.Message serialization tests
// So we focus on serialization of groups and the various kinds of proto.Message
// implementations that can back them (similar to TestEncodeMessage above).
func TestEncodeFieldValue_Group(t *testing.T) {
	atmMd, err := desc.LoadMessageDescriptorForMessage((*testprotos.AnotherTestMessage)(nil))
	testutil.Ok(t, err)

	rrFd := atmMd.FindFieldByNumber(6) // tag 6 is the group

	// A generated message will be encoded using its XXX_Size and XXX_Marshal
	// methods
	pm := &testprotos.AnotherTestMessage_RockNRoll{
		Beatles: proto.String("Sgt. Pepper's Lonely Hearts Club Band"),
		Stones:  proto.String("Exile on Main St."),
		Doors:   proto.String("Strange Days"),
	}

	// A generated message will be encoded using its MarshalAppend and
	// MarshalAppendDeterministic methods
	md, err := desc.LoadMessageDescriptorForMessage(pm)
	testutil.Ok(t, err)
	dm := dynamic.NewMessage(md)
	err = dm.ConvertFrom(pm)
	testutil.Ok(t, err)

	// This custom message will use MarshalDeterministic method or fall back to
	// old proto.Marshal implementation for non-deterministic marshaling
	cm := (*TestGroup)(pm)

	testCases := []struct{
		Name string
		Msg  proto.Message
	}{
		{Name: "generated", Msg: pm},
		{Name: "dynamic", Msg: dm},
		{Name: "custom", Msg: cm},
	}

	methods := []struct{
		Name string
		Func func(*codec.Buffer, *desc.FieldDescriptor, interface{}) error
	}{
		{Name: "deterministic", Func: (*codec.Buffer).EncodeFieldValueDeterministic},
		{Name: "non-deterministic", Func: (*codec.Buffer).EncodeFieldValue},
	}

	var bytes []byte

	for _, mtd := range methods {
		t.Run(mtd.Name, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.Name, func(t *testing.T) {
					var cb codec.Buffer
					err := mtd.Func(&cb, rrFd, tc.Msg)
					testutil.Ok(t, err)
					b := cb.Bytes()
					if bytes == nil {
						bytes = b
						// make sure that the bytes are valid
						expected := &testprotos.AnotherTestMessage{Rocknroll: pm}
						var actual testprotos.AnotherTestMessage
						err := proto.Unmarshal(b, &actual)
						testutil.Ok(t, err)
						testutil.Require(t, proto.Equal(expected, &actual))
					} else {
						// The generated proto message is the benchmark.
						// Ensure that all others match its output.
						// (We can do this even for non-deterministic
						// method because the actual data being marshaled
						// has no map values, so will always be the same)
						testutil.Eq(t, bytes, b)
					}
				})
			}
		})
	}
}

type TestMessage testprotos.Test

func (m *TestMessage) Reset() {
	(*testprotos.Test)(m).Reset()
}

func (m *TestMessage) String() string {
	return (*testprotos.Test)(m).String()
}

func (m *TestMessage) ProtoMessage() {
}

func (m *TestMessage) MarshalDeterministic() ([]byte, error) {
	t := (*testprotos.Test)(m)
	sz := t.XXX_Size()
	b := make([]byte, 0, sz)
	return t.XXX_Marshal(b, true)
}

type TestGroup testprotos.AnotherTestMessage_RockNRoll

func (m *TestGroup) Reset() {
	(*testprotos.AnotherTestMessage_RockNRoll)(m).Reset()
}

func (m *TestGroup) String() string {
	return (*testprotos.AnotherTestMessage_RockNRoll)(m).String()
}

func (m *TestGroup) ProtoMessage() {
}

func (m *TestGroup) MarshalDeterministic() ([]byte, error) {
	t := (*testprotos.AnotherTestMessage_RockNRoll)(m)
	sz := t.XXX_Size()
	b := make([]byte, 0, sz)
	return t.XXX_Marshal(b, true)
}

func init() {
	proto.RegisterType((*TestMessage)(nil), "foo.bar.v2.TestMessage")
	proto.RegisterType((*TestGroup)(nil), "foo.bar.v2.TestGroup")
}