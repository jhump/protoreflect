package dynamic

import (
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestUnaryFields(t *testing.T) {
	binaryTranslationParty(t, unaryFieldsMsg)
}

func TestRepeatedFields(t *testing.T) {
	binaryTranslationParty(t, repeatedFieldsMsg)
}

func TestPackedRepeatedFields(t *testing.T) {
	binaryTranslationParty(t, repeatedPackedFieldsMsg)

}

func TestMapKeyFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	binaryTranslationParty(t, mapKeyFieldsMsg)
}

func TestMapValueFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	binaryTranslationParty(t, mapValueFieldsMsg)
}

func TestUnknownFields(t *testing.T) {
	// TODO
}

func binaryTranslationParty(t *testing.T, msg proto.Message) {
	doTranslationParty(t, msg, proto.Marshal, proto.Unmarshal, (*Message).Marshal, (*Message).Unmarshal)
}
