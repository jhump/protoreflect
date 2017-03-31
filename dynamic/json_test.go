package dynamic

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/jsonpb"
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

func jsonTranslationParty(t *testing.T, msg proto.Message) {
	doTranslationParty(t, msg,
		func(pm proto.Message) ([]byte, error) {
			// TODO: jsonpb should handle case where given message implements json.Marshaler
			// https://github.com/golang/protobuf/pull/325
			// Remove the following three lines if/when that change is merged
			if dm, ok := pm.(*Message); ok {
				return dm.MarshalJSON()
			}
			m := jsonpb.Marshaler{}
			s, err := m.MarshalToString(pm)
			if err != nil {
				return nil, err
			} else {
				return []byte(s), nil
			}
		},
		func(b []byte, pm proto.Message) error {
			// TODO: jsonpb should handle case where given message implements json.Marshaler
			// https://github.com/golang/protobuf/pull/325
			// Remove the following three lines if/when that change is merged
			if dm, ok := pm.(*Message); ok {
				return dm.UnmarshalJSON(b)
			}
			return jsonpb.UnmarshalString(string(b), pm)
		},
		(*Message).MarshalJSON, (*Message).UnmarshalJSON)
}
