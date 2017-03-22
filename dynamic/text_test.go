package dynamic

import (
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestUnaryFieldsText(t *testing.T) {
	textTranslationParty(t, unaryFieldsMsg)
}

func TestRepeatedFieldsText(t *testing.T) {
	textTranslationParty(t, repeatedFieldsMsg)
}

func TestMapKeyFieldsText(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	textTranslationParty(t, mapKeyFieldsMsg)
}

func TestMapValueFieldsText(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	textTranslationParty(t, mapValueFieldsMsg)
}

func TestUnknownFieldsText(t *testing.T) {
	// TODO
}

func TestExtensionFieldsText(t *testing.T) {
	// TODO
}

func TestLenientParsingText(t *testing.T) {
	// TODO
	// include optional commas, different ways to indicate extension names, and array notation for repeated fields
}

func textTranslationParty(t *testing.T, msg proto.Message) {
	doTranslationParty(t, msg,
		func(pm proto.Message) ([]byte, error) {
			return []byte(proto.MarshalTextString(pm)), nil
		},
		func(b []byte, pm proto.Message) error {
			return proto.UnmarshalText(string(b), pm)
		},
		(*Message).MarshalText, (*Message).UnmarshalText)
}
