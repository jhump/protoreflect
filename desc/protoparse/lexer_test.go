package protoparse

import (
	"strings"
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestLexer(t *testing.T) {
	l := newLexer(strings.NewReader(`
	// comment

	/*
	 * block comment
	 */ /* inline comment */

	int32  "\032\x16\n\rfoobar\"zap"		'another\tstring\'s\t'

	// another comment
	// more and more...

	service rpc message
	.type
	.f.q.n
	name
	f.q.n

	.01
	.01e12
	.01e+5
	.033e-1

	12345
	-12345
	123.1234
	0.123
	012345
	0x2134abcdef30
	-0543
	-0xff76
	101.0102
	202.0203e1
	304.0304e-10
	3.1234e+12

	{ } + - , ;

	[option=foo]
	syntax = "proto2";

	// some strange cases
	1.543g12 /* trailing line comment */
	000.000
	0.1234.5678.
	12e12

	Random_identifier_with_numbers_0123456789_and_letters...
	`))

	var sym protoSymType
	expected := []struct {
		t int
		v interface{}
	}{
		{t: _INT32, v: "int32"},
		{t: _STRING_LIT, v: "\032\x16\n\rfoobar\"zap"},
		{t: _STRING_LIT, v: "another\tstring's\t"},
		{t: _SERVICE, v: "service"},
		{t: _RPC, v: "rpc"},
		{t: _MESSAGE, v: "message"},
		{t: _TYPENAME, v: ".type"},
		{t: _TYPENAME, v: ".f.q.n"},
		{t: _NAME, v: "name"},
		{t: _FQNAME, v: "f.q.n"},
		{t: _FLOAT_LIT, v: 0.01},
		{t: _FLOAT_LIT, v: 0.01e12},
		{t: _FLOAT_LIT, v: 0.01e5},
		{t: _FLOAT_LIT, v: 0.033e-1},
		{t: _INT_LIT, v: uint64(12345)},
		{t: '-', v: nil},
		{t: _INT_LIT, v: uint64(12345)},
		{t: _FLOAT_LIT, v: 123.1234},
		{t: _FLOAT_LIT, v: 0.123},
		{t: _INT_LIT, v: uint64(012345)},
		{t: _INT_LIT, v: uint64(0x2134abcdef30)},
		{t: '-', v: nil},
		{t: _INT_LIT, v: uint64(0543)},
		{t: '-', v: nil},
		{t: _INT_LIT, v: uint64(0xff76)},
		{t: _FLOAT_LIT, v: 101.0102},
		{t: _FLOAT_LIT, v: 202.0203e1},
		{t: _FLOAT_LIT, v: 304.0304e-10},
		{t: _FLOAT_LIT, v: 3.1234e+12},
		{t: '{', v: nil},
		{t: '}', v: nil},
		{t: '+', v: nil},
		{t: '-', v: nil},
		{t: ',', v: nil},
		{t: ';', v: nil},
		{t: '[', v: nil},
		{t: _OPTION, v: "option"},
		{t: '=', v: nil},
		{t: _NAME, v: "foo"},
		{t: ']', v: nil},
		{t: _SYNTAX, v: "syntax"},
		{t: '=', v: nil},
		{t: _STRING_LIT, v: "proto2"},
		{t: ';', v: nil},
		{t: _FLOAT_LIT, v: 1.543},
		{t: _NAME, v: "g12"},
		{t: _FLOAT_LIT, v: 0.0},
		{t: _FLOAT_LIT, v: 0.1234},
		{t: _FLOAT_LIT, v: 0.5678},
		{t: '.', v: nil},
		{t: _FLOAT_LIT, v: 12e12},
		{t: _NAME, v: "Random_identifier_with_numbers_0123456789_and_letters"},
		{t: '.', v: nil},
		{t: '.', v: nil},
		{t: '.', v: nil},
	}

	for i, exp := range expected {
		tok := l.Lex(&sym)
		if tok == 0 {
			t.Fatalf("lexer reported EOF but should have returned %v", exp)
		}
		var val interface{}
		switch tok {
		case _SYNTAX, _OPTION, _INT32, _SERVICE, _RPC, _MESSAGE, _TYPENAME, _NAME, _FQNAME:
			val = sym.id.val
		case _STRING_LIT:
			val = sym.str.val
		case _INT_LIT:
			val = sym.ui.val
		case _FLOAT_LIT:
			val = sym.f.val
		default:
			val = nil
		}
		testutil.Eq(t, exp.t, tok, "case %d: wrong token type (case %v)", i, exp.v)
		testutil.Eq(t, exp.v, val, "case %d: wrong token value", i)
	}
	if tok := l.Lex(&sym); tok != 0 {
		t.Fatalf("lexer reported symbol after what should have been EOF: %d", tok)
	}
}

func TestLexerErrors(t *testing.T) {
	testCases := []struct {
		str    string
		errMsg string
	}{
		{str: `0xffffffffffffffffffff`, errMsg: "value out of range"},
		{str: `"foobar`, errMsg: "unexpected EOF"},
		{str: `"foobar\J"`, errMsg: "invalid escape sequence"},
		{str: `"foobar\xgfoo"`, errMsg: "invalid hex escape"},
		{str: `"foobar\u09gafoo"`, errMsg: "invalid unicode escape"},
		{str: `"foobar\U0010005zfoo"`, errMsg: "invalid unicode escape"},
		{str: `"foobar\U00110000foo"`, errMsg: "unicode escape is out of range"},
		{str: "'foobar\nbaz'", errMsg: "encountered end-of-line"},
		{str: "'foobar\000baz'", errMsg: "null character ('\\0') not allowed"},
		{str: `/* foobar`, errMsg: "unexpected EOF"},
	}
	for i, tc := range testCases {
		l := newLexer(strings.NewReader(tc.str))
		var sym protoSymType
		tok := l.Lex(&sym)
		testutil.Eq(t, _ERROR, tok)
		testutil.Require(t, sym.err != nil)
		testutil.Require(t, strings.Contains(sym.err.Error(), tc.errMsg), "case %d: expected message to contain %q but does not: %q", i, tc.errMsg, sym.err.Error())
	}
}
