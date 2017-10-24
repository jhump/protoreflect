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
	// this is a trailing comment
	// that spans multiple lines
	// over two in fact!
	/*
	 * this is a detached comment
	 * with lots of extra words and stuff...
	 */

	// this is an attached leading comment
	foo
	`))

	var prev node
	var sym protoSymType
	expected := []struct {
		t          int
		line, col  int
		span       int
		v          interface{}
		comments   []string
		trailCount int
	}{
		{t: _INT32, line: 8, col: 2, span: 5, v: "int32", comments: []string{"// comment", "/*\n\t * block comment\n\t */", "/* inline comment */"}},
		{t: _STRING_LIT, line: 8, col: 9, span: 25, v: "\032\x16\n\rfoobar\"zap"},
		{t: _STRING_LIT, line: 8, col: 36, span: 22, v: "another\tstring's\t"},
		{t: _SERVICE, line: 13, col: 2, span: 7, v: "service", comments: []string{"// another comment", "// more and more..."}},
		{t: _RPC, line: 13, col: 10, span: 3, v: "rpc"},
		{t: _MESSAGE, line: 13, col: 14, span: 7, v: "message"},
		{t: _TYPENAME, line: 14, col: 2, span: 5, v: ".type"},
		{t: _TYPENAME, line: 15, col: 2, span: 6, v: ".f.q.n"},
		{t: _NAME, line: 16, col: 2, span: 4, v: "name"},
		{t: _FQNAME, line: 17, col: 2, span: 5, v: "f.q.n"},
		{t: _FLOAT_LIT, line: 19, col: 2, span: 3, v: 0.01},
		{t: _FLOAT_LIT, line: 20, col: 2, span: 6, v: 0.01e12},
		{t: _FLOAT_LIT, line: 21, col: 2, span: 6, v: 0.01e5},
		{t: _FLOAT_LIT, line: 22, col: 2, span: 7, v: 0.033e-1},
		{t: _INT_LIT, line: 24, col: 2, span: 5, v: uint64(12345)},
		{t: '-', line: 25, col: 2, span: 1, v: nil},
		{t: _INT_LIT, line: 25, col: 3, span: 5, v: uint64(12345)},
		{t: _FLOAT_LIT, line: 26, col: 2, span: 8, v: 123.1234},
		{t: _FLOAT_LIT, line: 27, col: 2, span: 5, v: 0.123},
		{t: _INT_LIT, line: 28, col: 2, span: 6, v: uint64(012345)},
		{t: _INT_LIT, line: 29, col: 2, span: 14, v: uint64(0x2134abcdef30)},
		{t: '-', line: 30, col: 2, span: 1, v: nil},
		{t: _INT_LIT, line: 30, col: 3, span: 4, v: uint64(0543)},
		{t: '-', line: 31, col: 2, span: 1, v: nil},
		{t: _INT_LIT, line: 31, col: 3, span: 6, v: uint64(0xff76)},
		{t: _FLOAT_LIT, line: 32, col: 2, span: 8, v: 101.0102},
		{t: _FLOAT_LIT, line: 33, col: 2, span: 10, v: 202.0203e1},
		{t: _FLOAT_LIT, line: 34, col: 2, span: 12, v: 304.0304e-10},
		{t: _FLOAT_LIT, line: 35, col: 2, span: 10, v: 3.1234e+12},
		{t: '{', line: 37, col: 2, span: 1, v: nil},
		{t: '}', line: 37, col: 4, span: 1, v: nil},
		{t: '+', line: 37, col: 6, span: 1, v: nil},
		{t: '-', line: 37, col: 8, span: 1, v: nil},
		{t: ',', line: 37, col: 10, span: 1, v: nil},
		{t: ';', line: 37, col: 12, span: 1, v: nil},
		{t: '[', line: 39, col: 2, span: 1, v: nil},
		{t: _OPTION, line: 39, col: 3, span: 6, v: "option"},
		{t: '=', line: 39, col: 9, span: 1, v: nil},
		{t: _NAME, line: 39, col: 10, span: 3, v: "foo"},
		{t: ']', line: 39, col: 13, span: 1, v: nil},
		{t: _SYNTAX, line: 40, col: 2, span: 6, v: "syntax"},
		{t: '=', line: 40, col: 9, span: 1, v: nil},
		{t: _STRING_LIT, line: 40, col: 11, span: 8, v: "proto2"},
		{t: ';', line: 40, col: 19, span: 1, v: nil},
		{t: _FLOAT_LIT, line: 43, col: 2, span: 5, v: 1.543, comments: []string{"// some strange cases"}},
		{t: _NAME, line: 43, col: 7, span: 3, v: "g12"},
		{t: _FLOAT_LIT, line: 44, col: 2, span: 7, v: 0.0, comments: []string{"/* trailing line comment */"}, trailCount: 1},
		{t: _FLOAT_LIT, line: 45, col: 2, span: 6, v: 0.1234},
		{t: _FLOAT_LIT, line: 45, col: 8, span: 5, v: 0.5678},
		{t: '.', line: 45, col: 13, span: 1, v: nil},
		{t: _FLOAT_LIT, line: 46, col: 2, span: 5, v: 12e12},
		{t: _NAME, line: 48, col: 2, span: 53, v: "Random_identifier_with_numbers_0123456789_and_letters"},
		{t: '.', line: 48, col: 55, span: 1, v: nil},
		{t: '.', line: 48, col: 56, span: 1, v: nil},
		{t: '.', line: 48, col: 57, span: 1, v: nil},
		{t: _NAME, line: 58, col: 2, span: 3, v: "foo", comments: []string{"// this is a trailing comment", "// that spans multiple lines", "// over two in fact!", "/*\n\t * this is a detached comment\n\t * with lots of extra words and stuff...\n\t */", "// this is an attached leading comment"}, trailCount: 3},
	}

	for i, exp := range expected {
		tok := l.Lex(&sym)
		if tok == 0 {
			t.Fatalf("lexer reported EOF but should have returned %v", exp)
		}
		var n node
		var val interface{}
		switch tok {
		case _SYNTAX, _OPTION, _INT32, _SERVICE, _RPC, _MESSAGE, _TYPENAME, _NAME, _FQNAME:
			n = sym.id
			val = sym.id.val
		case _STRING_LIT:
			n = sym.str
			val = sym.str.val
		case _INT_LIT:
			n = sym.ui
			val = sym.ui.val
		case _FLOAT_LIT:
			n = sym.f
			val = sym.f.val
		default:
			n = sym.b
			val = nil
		}
		testutil.Eq(t, exp.t, tok, "case %d: wrong token type (case %v)", i, exp.v)
		testutil.Eq(t, exp.v, val, "case %d: wrong token value", i)
		testutil.Eq(t, exp.line, n.start().Line, "case %d: wrong line number", i)
		testutil.Eq(t, exp.col, n.start().Col, "case %d: wrong column number", i)
		testutil.Eq(t, exp.line, n.end().Line, "case %d: wrong end line number", i)
		testutil.Eq(t, exp.col+exp.span, n.end().Col, "case %d: wrong end column number", i)
		if exp.trailCount > 0 {
			testutil.Eq(t, exp.trailCount, len(prev.trailingComment()), "case %d: wrong number of trailing comments", i)
		}
		testutil.Eq(t, len(exp.comments)-exp.trailCount, len(n.leadingComments()), "case %d: wrong number of comments", i)
		for ci := range exp.comments {
			var c *comment
			if ci < exp.trailCount {
				c = prev.trailingComment()[ci]
			} else {
				c = n.leadingComments()[ci-exp.trailCount]
			}
			testutil.Eq(t, exp.comments[ci], c.text, "case %d, comment #%d: unexpected text", i, ci+1)
		}
		prev = n
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
