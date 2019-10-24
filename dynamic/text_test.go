package dynamic

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestTextUnaryFields(t *testing.T) {
	textTranslationParty(t, unaryFieldsPosMsg, false)
	textTranslationParty(t, unaryFieldsNegMsg, false)
	textTranslationParty(t, unaryFieldsPosInfMsg, false)
	textTranslationParty(t, unaryFieldsNegInfMsg, false)
	textTranslationParty(t, unaryFieldsNanMsg, true)
}

func TestTextRepeatedFields(t *testing.T) {
	textTranslationParty(t, repeatedFieldsMsg, false)
	textTranslationParty(t, repeatedFieldsInfNanMsg, true)
}

func TestTextMapKeyFields(t *testing.T) {
	textTranslationParty(t, mapKeyFieldsMsg, false)
}

func TestTextMapValueFields(t *testing.T) {
	textTranslationParty(t, mapValueFieldsMsg, false)
	textTranslationParty(t, mapValueFieldsInfNanMsg, true)
	textTranslationParty(t, mapValueFieldsNilMsg, false)
	textTranslationParty(t, mapValueFieldsNilUnknownMsg, false)
}

func TestTextUnknownFields(t *testing.T) {
	// TODO
}

func TestTextExtensionFields(t *testing.T) {
	// TODO
}

func TestTextLenientParsing(t *testing.T) {
	expectedTestMsg := &testprotos.TestMessage{
		Anm: &testprotos.TestMessage_NestedMessage_AnotherNestedMessage{
			Yanm: []*testprotos.TestMessage_NestedMessage_AnotherNestedMessage_YetAnotherNestedMessage{
				{Foo: proto.String("bar"), Bar: proto.Int32(42), Baz: []byte("foo")},
			},
		},
		Ne: []testprotos.TestMessage_NestedEnum{testprotos.TestMessage_VALUE1, testprotos.TestMessage_VALUE2},
	}
	expectedAnTestMsg := &testprotos.AnotherTestMessage{
		Rocknroll: &testprotos.AnotherTestMessage_RockNRoll{
			Beatles: proto.String("abbey road"),
			Stones:  proto.String("exile on main street"),
			Doors:   proto.String("waiting for the sun"),
		},
	}
	expectedAnTestMsgExts := &testprotos.AnotherTestMessage{}
	err := proto.SetExtension(expectedAnTestMsgExts, testprotos.E_Xs, proto.String("fubar"))
	testutil.Ok(t, err)
	err = proto.SetExtension(expectedAnTestMsgExts, testprotos.E_Xi, proto.Int32(10101))
	testutil.Ok(t, err)

	extreg := NewExtensionRegistryWithDefaults()

	testCases := []struct {
		text     string
		expected proto.Message
	}{
		{
			// normal format: repeated fields are repeated, no commas
			text:     `ne: VALUE1 ne: VALUE2 anm:<yanm:<foo:"bar" bar:42 baz:"foo">>`,
			expected: expectedTestMsg,
		},
		{
			// angle bracket but no colon
			text:     `ne: VALUE1 ne: VALUE2 anm<yanm<foo:"bar" bar:42 baz:"foo">>`,
			expected: expectedTestMsg,
		},
		{
			// refer to field by tag
			text:     `4: VALUE1 4: VALUE2 2:<1:<1:"bar" 2:42 3:"foo">>`,
			expected: expectedTestMsg,
		},
		{
			// repeated fields w/ array syntax, no commas
			text:     `ne: [VALUE1 VALUE2] anm:<yanm:<foo:"bar" bar:42 baz:"foo">>`,
			expected: expectedTestMsg,
		},
		{
			// repeated fields w/ array syntax, commas
			text:     `ne: [VALUE1, VALUE2], anm:<yanm:<foo:"bar", bar:42, baz:"foo",>>`,
			expected: expectedTestMsg,
		},
		{
			// repeated fields w/ array syntax, semicolons
			text:     `ne: [VALUE1; VALUE2]; anm:<yanm:<foo:"bar"; bar:42; baz:"foo";>>`,
			expected: expectedTestMsg,
		},
		{
			// braces instead of angles for messages
			text:     `ne: VALUE1 ne: VALUE2 anm:{yanm:{foo:"bar" bar:42 baz:"foo"}}`,
			expected: expectedTestMsg,
		},
		{
			// braces and no colons for messages (group syntax)
			text:     `ne: VALUE1 ne: VALUE2 anm{yanm{foo:"bar" bar:42 baz:"foo"}}`,
			expected: expectedTestMsg,
		},
		{
			// braces and no colons for groups
			text:     `rocknroll{beatles:"abbey road" stones:"exile on main street" doors:"waiting for the sun"}`,
			expected: expectedAnTestMsg,
		},
		{
			// angles and colons for groups (message syntax)
			text:     `rocknroll:<beatles:"abbey road" stones:"exile on main street" doors:"waiting for the sun">`,
			expected: expectedAnTestMsg,
		},
		{
			// braces and colons
			text:     `rocknroll:{beatles:"abbey road" stones:"exile on main street" doors:"waiting for the sun"}`,
			expected: expectedAnTestMsg,
		},
		{
			// group name
			text:     `RockNRoll:{beatles:"abbey road" stones:"exile on main street" doors:"waiting for the sun"}`,
			expected: expectedAnTestMsg,
		},
		{
			// proper names for extension fields
			text:     `[testprotos.xs]:"fubar" [testprotos.xi]:10101`,
			expected: expectedAnTestMsgExts,
		},
		{
			// extensions with parenthesis instead of brackets
			text:     `(testprotos.xs):"fubar" (testprotos.xi):10101`,
			expected: expectedAnTestMsgExts,
		},
		{
			// extension names as if normal fields
			text:     `testprotos.xs:"fubar" testprotos.xi:10101`,
			expected: expectedAnTestMsgExts,
		},
		{
			// refer to extensions with tag numbers
			text:     `101:"fubar" 102:10101`,
			expected: expectedAnTestMsgExts,
		},
	}
	for i, testCase := range testCases {
		md, err := desc.LoadMessageDescriptorForMessage(testCase.expected)
		testutil.Ok(t, err, "case %d: failed get descriptor for %T", i+1, testCase.expected)
		dm := NewMessageWithExtensionRegistry(md, extreg)
		err = dm.UnmarshalText([]byte(testCase.text))
		testutil.Ok(t, err, "case %d: failed unmarshal text: %q", i+1, testCase.text)
		testutil.Ceq(t, testCase.expected, dm, eqm, "case %d: incorrect unmarshaled result", i+1)
	}
}

func textTranslationParty(t *testing.T, msg proto.Message, includesNaN bool) {
	doTranslationParty(t, msg,
		func(pm proto.Message) ([]byte, error) {
			return []byte(proto.CompactTextString(pm)), nil
		},
		func(b []byte, pm proto.Message) error {
			return proto.UnmarshalText(string(b), pm)
		},
		(*Message).MarshalText, (*Message).UnmarshalText, includesNaN,
		// we don't compare bytes because we can't really make the proto and dynamic
		// marshal methods work the same due to API differences in how to enable
		// indentation/pretty-printing
		false, true)
}
