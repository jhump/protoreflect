package dynamic

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
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
	jsonTranslationParty(t, unaryFieldsMsg)
}

func TestJSONRepeatedFields(t *testing.T) {
	jsonTranslationParty(t, repeatedFieldsMsg)
}

func TestJSONMapKeyFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	jsonTranslationParty(t, mapKeyFieldsMsg)
}

func TestJSONMapValueFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	jsonTranslationParty(t, mapValueFieldsMsg)
}

func TestJSONExtensionFields(t *testing.T) {
	// TODO
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
	testutil.Eq(t, `{"id":0,"name":""}`, string(jsDefaults))
}

func TestMarshalJSONEmitDefaultsMapKeyFields(t *testing.T) {
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

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
	testutil.Eq(t, "{\n  \"foo\": [\n    \"VALUE1\"\n  ],\n  \"bar\": \"bedazzle\"\n}", string(jsIndent))
	jsIndent, err = dm.MarshalJSONPB(&jsonpb.Marshaler{Indent: "\t"})
	testutil.Ok(t, err)
	testutil.Eq(t, "{\n\t\"foo\": [\n\t\t\"VALUE1\"\n\t],\n\t\"bar\": \"bedazzle\"\n}", string(jsIndent))
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
	testutil.Eq(t, *wkts.StartTime, *ts)

	dur, ok := dm.GetFieldByNumber(2).(*duration.Duration)
	testutil.Require(t, ok)
	testutil.Eq(t, *wkts.Elapsed, *dur)

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

func jsonTranslationParty(t *testing.T, msg proto.Message) {
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
		(*Message).MarshalJSON, (*Message).UnmarshalJSON)
}
