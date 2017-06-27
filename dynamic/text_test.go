package dynamic

import (
	"testing"

	"github.com/golang/protobuf/proto"
)

func TestTextUnaryFields(t *testing.T) {
	textTranslationParty(t, unaryFieldsMsg)
}

func TestTextRepeatedFields(t *testing.T) {
	textTranslationParty(t, repeatedFieldsMsg)
}

func TestTextMapKeyFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	textTranslationParty(t, mapKeyFieldsMsg)
}

func TestTextMapValueFields(t *testing.T) {
	// translation party wants deterministic marshalling to bytes
	sort_map_keys = true
	defer func() {
		sort_map_keys = false
	}()

	textTranslationParty(t, mapValueFieldsMsg)
}

func TestTextUnknownFields(t *testing.T) {
	// TODO
}

func TestTextExtensionFields(t *testing.T) {
	// TODO
}

func TestTextLenientParsing(t *testing.T) {
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
