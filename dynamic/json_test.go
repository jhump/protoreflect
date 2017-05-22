package dynamic

import (
	"testing"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"bytes"
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
	// TODO
}

func TestMarshalJSONEnumsAsInts(t *testing.T) {
	// TODO
}

func TestMarshalJSONOrigName(t *testing.T) {
	// TODO
}

func TestMarshalJSONIndent(t *testing.T) {
	// TODO
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
