package dynamic

import (
	"bytes"
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestUnaryFieldsJSON(t *testing.T) {
	jsonTranslationParty(t, unaryFieldsMsg)
}

func TestRepeatedFieldsJSON(t *testing.T) {
	jsonTranslationParty(t, repeatedFieldsMsg)
}

func TestMapKeyFieldsJSON(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	jsonTranslationParty(t, mapKeyFieldsMsg)
}

func TestMapValueFieldsJSON(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	jsonTranslationParty(t, mapValueFieldsMsg)
}

func TestExtensionFieldsJSON(t *testing.T) {
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
	// TODO
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
