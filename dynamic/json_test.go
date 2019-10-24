package dynamic

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/struct"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestJSONUnaryFields(t *testing.T) {
	jsonTranslationParty(t, unaryFieldsPosMsg, false)
	jsonTranslationParty(t, unaryFieldsNegMsg, false)
	jsonTranslationParty(t, unaryFieldsPosInfMsg, false)
	jsonTranslationParty(t, unaryFieldsNegInfMsg, false)
	jsonTranslationParty(t, unaryFieldsNanMsg, true)
}

func TestJSONRepeatedFields(t *testing.T) {
	jsonTranslationParty(t, repeatedFieldsMsg, false)
	jsonTranslationParty(t, repeatedFieldsInfNanMsg, true)
}

func TestJSONMapKeyFields(t *testing.T) {
	jsonTranslationParty(t, mapKeyFieldsMsg, false)
}

func TestJSONMapValueFields(t *testing.T) {
	jsonTranslationParty(t, mapValueFieldsMsg, false)
	jsonTranslationParty(t, mapValueFieldsInfNanMsg, true)
	jsonTranslationParty(t, mapValueFieldsNilMsg, false)
	jsonTranslationParty(t, mapValueFieldsNilUnknownMsg, false)
}

func TestJSONExtensionFields(t *testing.T) {
	// TODO
}

func createTestFileDescriptor(t *testing.T, packageName string) *desc.FileDescriptor {
	// Create a new type that could only be resolved via custom resolver
	// because it does not exist in compiled form
	fdp := descriptor.FileDescriptorProto{
		Name:       proto.String(fmt.Sprintf("%s.proto", packageName)),
		Dependency: []string{"google/protobuf/any.proto"},
		Package:    proto.String(packageName),
		MessageType: []*descriptor.DescriptorProto{
			{
				Name: proto.String("MyMessage"),
				Field: []*descriptor.FieldDescriptorProto{
					{
						Name:   proto.String("abc"),
						Number: proto.Int(1),
						Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptor.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:   proto.String("def"),
						Number: proto.Int(2),
						Label:  descriptor.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:   descriptor.FieldDescriptorProto_TYPE_INT32.Enum(),
					},
					{
						Name:     proto.String("ghi"),
						Number:   proto.Int(3),
						Label:    descriptor.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptor.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".google.protobuf.Any"),
					},
				},
			},
		},
	}
	anyfd, err := desc.LoadFileDescriptor("google/protobuf/any.proto")
	testutil.Ok(t, err)
	fd, err := desc.CreateFileDescriptor(&fdp, anyfd)
	testutil.Ok(t, err)
	return fd
}

func TestJSONAnyResolver(t *testing.T) {
	fd1 := createTestFileDescriptor(t, "foobar")
	fd2 := createTestFileDescriptor(t, "snafu")
	md := fd1.FindMessage("foobar.MyMessage")
	dm := NewMessage(md)
	dm.SetFieldByNumber(1, "fubar")
	dm.SetFieldByNumber(2, int32(123))
	a1, err := ptypes.MarshalAny(dm)
	testutil.Ok(t, err)
	md = fd2.FindMessage("snafu.MyMessage")
	dm = NewMessage(md)
	dm.SetFieldByNumber(1, "snafu")
	dm.SetFieldByNumber(2, int32(456))
	a2, err := ptypes.MarshalAny(dm)
	testutil.Ok(t, err)

	msg := &testprotos.TestWellKnownTypes{Extras: []*any.Any{a1, a2}}
	resolver := AnyResolver(nil, fd1, fd2)

	jsm := jsonpb.Marshaler{AnyResolver: resolver}
	js, err := jsm.MarshalToString(msg)
	testutil.Ok(t, err)
	expected := `{"extras":[{"@type":"type.googleapis.com/foobar.MyMessage","abc":"fubar","def":123},{"@type":"type.googleapis.com/snafu.MyMessage","abc":"snafu","def":456}]}`
	testutil.Eq(t, expected, js)

	jsu := jsonpb.Unmarshaler{AnyResolver: resolver}
	msg2 := &testprotos.TestWellKnownTypes{}
	err = jsu.Unmarshal(strings.NewReader(js), msg2)
	testutil.Ok(t, err)

	testutil.Ceq(t, msg, msg2, eqpm)
}

func TestJSONAnyResolver_AutomaticForDynamicMessage(t *testing.T) {
	// marshaling and unmarshaling a dynamic message automatically enables
	// resolving Any messages for known types (a known type is one that is
	// in the dynamic message's file descriptor or that file's transitive
	// dependencies)
	fd := createTestFileDescriptor(t, "foobar")
	md := fd.FindMessage("foobar.MyMessage")
	dm := NewMessageFactoryWithDefaults().NewMessage(md).(*Message)
	dm.SetFieldByNumber(1, "fubar")
	dm.SetFieldByNumber(2, int32(123))
	a1, err := ptypes.MarshalAny(dm)
	testutil.Ok(t, err)
	dm.SetFieldByNumber(1, "snafu")
	dm.SetFieldByNumber(2, int32(456))
	a2, err := ptypes.MarshalAny(dm)
	testutil.Ok(t, err)
	dm.SetFieldByNumber(1, "xyz")
	dm.SetFieldByNumber(2, int32(-987))
	dm.SetFieldByNumber(3, []*any.Any{a1, a2})

	js, err := dm.MarshalJSON()
	testutil.Ok(t, err)
	expected := `{"abc":"xyz","def":-987,"ghi":[{"@type":"type.googleapis.com/foobar.MyMessage","abc":"fubar","def":123},{"@type":"type.googleapis.com/foobar.MyMessage","abc":"snafu","def":456}]}`
	testutil.Eq(t, expected, string(js))

	dm2 := NewMessageFactoryWithDefaults().NewMessage(md).(*Message)
	err = dm2.UnmarshalJSON(js)
	testutil.Ok(t, err)

	testutil.Ceq(t, dm, dm2, eqdm)
}

func TestMarshalJSONEmitDefaults(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.ReallySimpleMessage)(nil))
	testutil.Ok(t, err)
	dm := NewMessage(md)
	js, err := dm.MarshalJSON()
	testutil.Ok(t, err)
	testutil.Eq(t, `{}`, string(js))
	jsDefaults, err := dm.MarshalJSONPB(&jsonpb.Marshaler{EmitDefaults: true})
	testutil.Ok(t, err)
	testutil.Eq(t, `{"id":"0","name":""}`, string(jsDefaults))
}

func TestMarshalJSONEmitDefaultsMapKeyFields(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.MapKeyFields)(nil))
	testutil.Ok(t, err)
	dm := NewMessage(md)
	m := &jsonpb.Marshaler{EmitDefaults: true}
	jsDefaults, err := dm.MarshalJSONPB(m)
	testutil.Ok(t, err)
	testutil.Eq(t, `{"i":{},"j":{},"k":{},"l":{},"m":{},"n":{},"o":{},"p":{},"q":{},"r":{},"s":{},"t":{}}`, string(jsDefaults))

	jsDefaults2, err := m.MarshalToString(&testprotos.MapKeyFields{})
	testutil.Ok(t, err)
	testutil.Eq(t, string(jsDefaults), string(jsDefaults2))
}

func TestMarshalJSONEmitDefaultsOneOfFields(t *testing.T) {
	// we don't include default values for fields in a one-of
	// since it would not round-trip correctly
	testCases := []struct {
		msg          *testprotos.OneOfMessage
		expectedJson string
	}{
		{
			msg:          &testprotos.OneOfMessage{},
			expectedJson: `{}`,
		},
		{
			msg:          &testprotos.OneOfMessage{Value: &testprotos.OneOfMessage_IntValue{IntValue: 12345}},
			expectedJson: `{"intValue":12345}`,
		},
		{
			msg:          &testprotos.OneOfMessage{Value: &testprotos.OneOfMessage_StringValue{StringValue: "foobar"}},
			expectedJson: `{"stringValue":"foobar"}`,
		},
		{
			msg:          &testprotos.OneOfMessage{Value: &testprotos.OneOfMessage_MsgValue{MsgValue: &testprotos.OneOfMessage{}}},
			expectedJson: `{"msgValue":{}}`,
		},
		{
			msg:          &testprotos.OneOfMessage{Value: &testprotos.OneOfMessage_MsgValue{MsgValue: nil}},
			expectedJson: `{"msgValue":null}`,
		},
	}
	m := &jsonpb.Marshaler{EmitDefaults: true}
	for _, testCase := range testCases {
		dm, err := AsDynamicMessageWithMessageFactory(testCase.msg, NewMessageFactoryWithDefaults())
		testutil.Ok(t, err)
		asJson, err := dm.MarshalJSONPB(m)
		testutil.Ok(t, err)
		actualJson := string(asJson)
		testutil.Eq(t, testCase.expectedJson, actualJson)

		// round-trip
		err = jsonpb.UnmarshalString(actualJson, dm)
		testutil.Ok(t, err)
		var roundtripped testprotos.OneOfMessage
		err = dm.ConvertTo(&roundtripped)
		testutil.Ok(t, err)
		testutil.Ceq(t, testCase.msg, &roundtripped, eqpm)
	}
}

func TestMarshalJSONEnumsAsInts(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.TestRequest)(nil))
	testutil.Ok(t, err)
	dm := NewMessage(md)
	dm.SetFieldByNumber(1, []int32{1})
	dm.SetFieldByNumber(2, "bedazzle")
	js, err := dm.MarshalJSONPB(&jsonpb.Marshaler{EnumsAsInts: true})
	testutil.Ok(t, err)
	testutil.Eq(t, `{"foo":[1],"bar":"bedazzle"}`, string(js))
}

func TestMarshalJSONOrigName(t *testing.T) {
	// TODO
}

func TestMarshalJSONIndent(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.TestRequest)(nil))
	testutil.Ok(t, err)
	dm := NewMessage(md)
	dm.SetFieldByNumber(1, []int32{1})
	dm.SetFieldByNumber(2, "bedazzle")
	js, err := dm.MarshalJSON()
	testutil.Ok(t, err)
	testutil.Eq(t, `{"foo":["VALUE1"],"bar":"bedazzle"}`, string(js))
	jsIndent, err := dm.MarshalJSONIndent()
	testutil.Ok(t, err)
	testutil.Eq(t, `{
  "foo": [
    "VALUE1"
  ],
  "bar": "bedazzle"
}`, string(jsIndent))
	jsIndent, err = dm.MarshalJSONPB(&jsonpb.Marshaler{Indent: "\t"})
	testutil.Ok(t, err)
	testutil.Eq(t, `{
	"foo": [
		"VALUE1"
	],
	"bar": "bedazzle"
}`, string(jsIndent))
}

func TestMarshalJSONIndentEmbedWellKnownTypes(t *testing.T) {
	// testing the formatting of dynamic message that embeds non-dynamic message,
	// both those w/ special/simple JSON encoding (like timestamp) and those with
	// more structure (Any).
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.TestWellKnownTypes)(nil))
	testutil.Ok(t, err)
	dm := NewMessage(md)

	ts, err := ptypes.TimestampProto(time.Date(2010, 3, 4, 5, 6, 7, 809000, time.UTC))
	testutil.Ok(t, err)
	dm.SetFieldByNumber(1, ts)

	anys := make([]*any.Any, 3)
	anys[0], err = ptypes.MarshalAny(&testprotos.TestRequest{Bar: "foo"})
	testutil.Ok(t, err)
	anys[1], err = ptypes.MarshalAny(&testprotos.TestRequest{Bar: "bar"})
	testutil.Ok(t, err)
	anys[2], err = ptypes.MarshalAny(&testprotos.TestRequest{Bar: "baz"})
	testutil.Ok(t, err)
	dm.SetFieldByNumber(13, anys)

	js, err := dm.MarshalJSON()
	testutil.Ok(t, err)
	testutil.Eq(t, `{"startTime":"2010-03-04T05:06:07.000809Z","extras":[{"@type":"type.googleapis.com/testprotos.TestRequest","bar":"foo"},{"@type":"type.googleapis.com/testprotos.TestRequest","bar":"bar"},{"@type":"type.googleapis.com/testprotos.TestRequest","bar":"baz"}]}`, string(js))
	jsIndent, err := dm.MarshalJSONIndent()
	testutil.Ok(t, err)
	testutil.Eq(t, `{
  "startTime": "2010-03-04T05:06:07.000809Z",
  "extras": [
    {
      "@type": "type.googleapis.com/testprotos.TestRequest",
      "bar": "foo"
    },
    {
      "@type": "type.googleapis.com/testprotos.TestRequest",
      "bar": "bar"
    },
    {
      "@type": "type.googleapis.com/testprotos.TestRequest",
      "bar": "baz"
    }
  ]
}`, string(jsIndent))
	jsIndent, err = dm.MarshalJSONPB(&jsonpb.Marshaler{Indent: "\t"})
	testutil.Ok(t, err)
	testutil.Eq(t, `{
	"startTime": "2010-03-04T05:06:07.000809Z",
	"extras": [
		{
			"@type": "type.googleapis.com/testprotos.TestRequest",
			"bar": "foo"
		},
		{
			"@type": "type.googleapis.com/testprotos.TestRequest",
			"bar": "bar"
		},
		{
			"@type": "type.googleapis.com/testprotos.TestRequest",
			"bar": "baz"
		}
	]
}`, string(jsIndent))
}

func TestUnmarshalJSONAllowUnknownFields(t *testing.T) {
	md, err := desc.LoadMessageDescriptorForMessage((*testprotos.TestRequest)(nil))
	testutil.Ok(t, err)
	js := []byte(`{"foo":["VALUE1"],"bar":"bedazzle","xxx": 1}`)
	dm := NewMessage(md)
	err = dm.UnmarshalJSON(js)
	testutil.Nok(t, err)
	unmarshaler := &jsonpb.Unmarshaler{AllowUnknownFields: true}
	err = dm.UnmarshalJSONPB(unmarshaler, js)
	testutil.Ok(t, err)
	foo := dm.GetFieldByNumber(1)
	bar := dm.GetFieldByNumber(2)
	testutil.Eq(t, []int32{1}, foo)
	testutil.Eq(t, "bedazzle", bar)
}

func TestJSONWellKnownType(t *testing.T) {
	any1, err := ptypes.MarshalAny(&testprotos.TestRequest{
		Foo: []testprotos.Proto3Enum{testprotos.Proto3Enum_VALUE1, testprotos.Proto3Enum_VALUE2},
		Bar: "bar",
		Baz: &testprotos.TestMessage{Ne: []testprotos.TestMessage_NestedEnum{testprotos.TestMessage_VALUE1}},
	})
	testutil.Ok(t, err)
	any2, err := ptypes.MarshalAny(ptypes.TimestampNow())
	testutil.Ok(t, err)

	wkts := &testprotos.TestWellKnownTypes{
		StartTime: &timestamp.Timestamp{Seconds: 1010101, Nanos: 20202},
		Elapsed:   &duration.Duration{Seconds: 30303, Nanos: 40404},
		Dbl:       &wrappers.DoubleValue{Value: 3.14159},
		Flt:       &wrappers.FloatValue{Value: -1.0101010},
		Bl:        &wrappers.BoolValue{Value: true},
		I32:       &wrappers.Int32Value{Value: -42},
		I64:       &wrappers.Int64Value{Value: -9090909090},
		U32:       &wrappers.UInt32Value{Value: 42},
		U64:       &wrappers.UInt64Value{Value: 9090909090},
		Str:       &wrappers.StringValue{Value: "foobar"},
		Byt:       &wrappers.BytesValue{Value: []byte("snafu")},
		Json: []*structpb.Value{
			{Kind: &structpb.Value_BoolValue{BoolValue: true}},
			{Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: []*structpb.Value{
				{Kind: &structpb.Value_NullValue{}},
				{Kind: &structpb.Value_StringValue{StringValue: "fubar"}},
				{Kind: &structpb.Value_NumberValue{NumberValue: 10101.20202}},
			}}}},
			{Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: map[string]*structpb.Value{
				"foo": {Kind: &structpb.Value_NullValue{}},
				"bar": {Kind: &structpb.Value_StringValue{StringValue: "snafu"}},
				"baz": {Kind: &structpb.Value_NumberValue{NumberValue: 30303.40404}},
			}}}},
		},
		Extras: []*any.Any{any1, any2},
	}

	jsm := jsonpb.Marshaler{}
	js, err := jsm.MarshalToString(wkts)
	testutil.Ok(t, err)

	md, err := desc.LoadMessageDescriptorForMessage(wkts)
	testutil.Ok(t, err)
	dm := NewMessage(md)
	err = dm.UnmarshalJSON([]byte(js))
	testutil.Ok(t, err)

	// check that the unmarshalled fields were constructed correctly with the
	// right value and type (e.g. generated well-known-type, not dynamic message)
	ts, ok := dm.GetFieldByNumber(1).(*timestamp.Timestamp)
	testutil.Require(t, ok)
	testutil.Ceq(t, wkts.StartTime, ts, eqpm)

	dur, ok := dm.GetFieldByNumber(2).(*duration.Duration)
	testutil.Require(t, ok)
	testutil.Ceq(t, wkts.Elapsed, dur, eqpm)

	dbl, ok := dm.GetFieldByNumber(3).(*wrappers.DoubleValue)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.Dbl.Value, dbl.Value)

	flt, ok := dm.GetFieldByNumber(4).(*wrappers.FloatValue)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.Flt.Value, flt.Value)

	bl, ok := dm.GetFieldByNumber(5).(*wrappers.BoolValue)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.Bl.Value, bl.Value)

	i32, ok := dm.GetFieldByNumber(6).(*wrappers.Int32Value)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.I32.Value, i32.Value)

	i64, ok := dm.GetFieldByNumber(7).(*wrappers.Int64Value)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.I64.Value, i64.Value)

	u32, ok := dm.GetFieldByNumber(8).(*wrappers.UInt32Value)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.U32.Value, u32.Value)

	u64, ok := dm.GetFieldByNumber(9).(*wrappers.UInt64Value)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.U64.Value, u64.Value)

	str, ok := dm.GetFieldByNumber(10).(*wrappers.StringValue)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.Str.Value, str.Value)

	byt, ok := dm.GetFieldByNumber(11).(*wrappers.BytesValue)
	testutil.Require(t, ok)
	testutil.Eq(t, wkts.Byt.Value, byt.Value)

	vals, ok := dm.GetFieldByNumber(12).([]interface{})
	testutil.Require(t, ok)
	testutil.Eq(t, len(wkts.Json), len(vals))
	for i := range vals {
		v, ok := vals[i].(*structpb.Value)
		testutil.Require(t, ok)
		testutil.Ceq(t, wkts.Json[i], v, eqpm)
	}

	extras, ok := dm.GetFieldByNumber(13).([]interface{})
	testutil.Require(t, ok)
	testutil.Eq(t, len(wkts.Extras), len(extras))
	for i := range extras {
		v, ok := extras[i].(*any.Any)
		testutil.Require(t, ok)
		testutil.Eq(t, wkts.Extras[i].TypeUrl, v.TypeUrl)
		testutil.Eq(t, wkts.Extras[i].Value, v.Value)
	}
}

func TestJSONWellKnownTypeFromFileDescriptorSet(t *testing.T) {
	// TODO: generalize this so it tests all well-known types, not just duration

	data, err := ioutil.ReadFile("../internal/testprotos/duration.protoset")
	testutil.Ok(t, err)
	fds := &descriptor.FileDescriptorSet{}
	err = proto.Unmarshal(data, fds)
	testutil.Ok(t, err)
	fd, err := desc.CreateFileDescriptorFromSet(fds)
	testutil.Ok(t, err)
	md := fd.FindMessage("google.protobuf.Duration")
	testutil.Neq(t, nil, md)

	dur := &duration.Duration{Seconds: 30303, Nanos: 40404}

	// marshal duration to JSON
	jsm := jsonpb.Marshaler{}
	js, err := jsm.MarshalToString(dur)
	testutil.Ok(t, err)

	// make sure we can unmarshal it
	dm := NewMessage(md)
	err = dm.UnmarshalJSON([]byte(js))
	testutil.Ok(t, err)

	// and then marshal it again with same output as original
	dynJs, err := jsm.MarshalToString(dm)
	testutil.Ok(t, err)
	testutil.Eq(t, js, dynJs)
}

func jsonTranslationParty(t *testing.T, msg proto.Message, includesNaN bool) {
	doTranslationParty(t, msg,
		func(pm proto.Message) ([]byte, error) {
			m := jsonpb.Marshaler{}
			var b bytes.Buffer
			err := m.Marshal(&b, pm)
			if err != nil {
				return nil, err
			} else {
				return b.Bytes(), nil
			}
		},
		func(b []byte, pm proto.Message) error {
			return jsonpb.Unmarshal(bytes.NewReader(b), pm)
		},
		(*Message).MarshalJSON, (*Message).UnmarshalJSON, includesNaN, true, true)
}
