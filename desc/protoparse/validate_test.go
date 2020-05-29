package protoparse

import (
	"github.com/jhump/protoreflect/internal/testutil"
	"strings"
	"testing"
)

func TestBasicValidation(t *testing.T) {
	testCases := []struct {
		contents string
		succeeds bool
		errMsg   string
	}{
		{
			contents: `message Foo { optional double bar = 1 [default = -18446744073709551615]; }`,
			succeeds: true,
		},
		{
			contents: `message Foo { optional double bar = 1 [default = 18446744073709551616]; }`,
			succeeds: true,
		},
		{
			contents: `message Foo { optional double bar = 536870912; option message_set_wire_format = true; }`,
			succeeds: true,
		},
		{
			contents: `message Foo { oneof bar { group Baz = 1 [deprecated=true] { optional int abc = 1; } } }`,
			succeeds: true,
		},
		{
			contents: `syntax = "proto1";`,
			errMsg:   `test.proto:1:10: syntax value must be "proto2" or "proto3"`,
		},
		{
			contents: `message Foo { optional string s = 5000000000; }`,
			errMsg:   `test.proto:1:35: tag number 5000000000 is higher than max allowed tag number (536870911)`,
		},
		{
			contents: `message Foo { optional string s = 19500; }`,
			errMsg:   `test.proto:1:35: tag number 19500 is in disallowed reserved range 19000-19999`,
		},
		{
			contents: `enum Foo { V = 5000000000; }`,
			errMsg:   `test.proto:1:16: value 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = -5000000000; }`,
			errMsg:   `test.proto:1:16: value -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved 5000000000; }`,
			errMsg:   `test.proto:1:28: range start 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000000; }`,
			errMsg:   `test.proto:1:28: range start -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved 5000000000 to 5000000001; }`,
			errMsg:   `test.proto:1:28: range start 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved 5 to 5000000000; }`,
			errMsg:   `test.proto:1:33: range end 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000000 to -5; }`,
			errMsg:   `test.proto:1:28: range start -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000001 to -5000000000; }`,
			errMsg:   `test.proto:1:28: range start -5000000001 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5000000000 to 5; }`,
			errMsg:   `test.proto:1:28: range start -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { V = 0; reserved -5 to 5000000000; }`,
			errMsg:   `test.proto:1:34: range end 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		{
			contents: `enum Foo { }`,
			errMsg:   `test.proto:1:1: enum Foo: enums must define at least one value`,
		},
		{
			contents: `message Foo { oneof Bar { } }`,
			errMsg:   `test.proto:1:15: oneof must contain at least one field`,
		},
		{
			contents: `message Foo { extensions 1 to max; } extend Foo { }`,
			errMsg:   `test.proto:1:38: extend sections must define at least one extension`,
		},
		{
			contents: `message Foo { option map_entry = true; }`,
			errMsg:   `test.proto:1:34: message Foo: map_entry option should not be set explicitly; use map type instead`,
		},
		{
			contents: `message Foo { option map_entry = false; }`,
			succeeds: true, // okay if explicit setting is false
		},
		{
			contents: `syntax = "proto2"; message Foo { string s = 1; }`,
			errMsg:   `test.proto:1:41: field Foo.s: field has no label, but proto2 must indicate 'optional' or 'required'`,
		},
		{
			contents: `message Foo { string s = 1; }`, // syntax defaults to proto2
			errMsg:   `test.proto:1:22: field Foo.s: field has no label, but proto2 must indicate 'optional' or 'required'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { optional string s = 1; }`,
			errMsg:   `test.proto:1:34: field Foo.s: field has label LABEL_OPTIONAL, but proto3 must omit labels other than 'repeated'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { required string s = 1; }`,
			errMsg:   `test.proto:1:34: field Foo.s: field has label LABEL_REQUIRED, but proto3 must omit labels other than 'repeated'`,
		},
		{
			contents: `message Foo { extensions 1 to max; } extend Foo { required string sss = 100; }`,
			errMsg:   `test.proto:1:51: field sss: extension fields cannot be 'required'`,
		},
		{
			contents: `syntax = "proto3"; message Foo { optional group Grp = 1 { } }`,
			errMsg:   `test.proto:1:43: field Foo.grp: groups are not allowed in proto3`,
		},
		{
			contents: `syntax = "proto3"; message Foo { extensions 1 to max; }`,
			errMsg:   `test.proto:1:45: message Foo: extension ranges are not allowed in proto3`,
		},
		{
			contents: `syntax = "proto3"; message Foo { string s = 1 [default = "abcdef"]; }`,
			errMsg:   `test.proto:1:48: field Foo.s: default values are not allowed in proto3`,
		},
		{
			contents: `enum Foo { V1 = 1; V2 = 1; }`,
			errMsg:   `test.proto:1:25: enum Foo: values V1 and V2 both have the same numeric value 1; use allow_alias option if intentional`,
		},
		{
			contents: `enum Foo { option allow_alias = true; V1 = 1; V2 = 1; }`,
			succeeds: true,
		},
		{
			contents: `syntax = "proto3"; enum Foo { V1 = 0; reserved 1 to 20; reserved "V2"; }`,
			succeeds: true,
		},
		{
			contents: `enum Foo { V1 = 1; reserved 1 to 20; reserved "V2"; }`,
			errMsg:   `test.proto:1:17: enum Foo: value V1 is using number 1 which is in reserved range 1 to 20`,
		},
		{
			contents: `enum Foo { V1 = 20; reserved 1 to 20; reserved "V2"; }`,
			errMsg:   `test.proto:1:17: enum Foo: value V1 is using number 20 which is in reserved range 1 to 20`,
		},
		{
			contents: `enum Foo { V2 = 0; reserved 1 to 20; reserved "V2"; }`,
			errMsg:   `test.proto:1:12: enum Foo: value V2 is using a reserved name`,
		},
		{
			contents: `enum Foo { V0 = 0; reserved 1 to 20; reserved 21 to 40; reserved "V2"; }`,
			succeeds: true,
		},
		{
			contents: `enum Foo { V0 = 0; reserved 1 to 20; reserved 20 to 40; reserved "V2"; }`,
			errMsg:   `test.proto:1:47: enum Foo: reserved ranges overlap: 1 to 20 and 20 to 40`,
		},
		{
			contents: `syntax = "proto3"; enum Foo { FIRST = 1; }`,
			errMsg:   `test.proto:1:39: enum Foo: proto3 requires that first value in enum have numeric value of 0`,
		},
		{
			contents: `syntax = "proto3"; message Foo { string s = 1; int32 i = 1; }`,
			errMsg:   `test.proto:1:58: message Foo: fields s and i both have the same tag 1`,
		},
		{
			contents: `message Foo { reserved 1 to 10, 10 to 12; }`,
			errMsg:   `test.proto:1:33: message Foo: reserved ranges overlap: 1 to 10 and 10 to 12`,
		},
		{
			contents: `message Foo { extensions 1 to 10, 10 to 12; }`,
			errMsg:   `test.proto:1:35: message Foo: extension ranges overlap: 1 to 10 and 10 to 12`,
		},
		{
			contents: `message Foo { reserved 1 to 10; extensions 10 to 12; }`,
			errMsg:   `test.proto:1:44: message Foo: extension range 10 to 12 overlaps reserved range 1 to 10`,
		},
		{
			contents: `message Foo { reserved 1, 2 to 10, 11 to 20; extensions 21 to 22; }`,
			succeeds: true,
		},
		{
			contents: `message Foo { reserved 10 to 1; }`,
			errMsg:   `test.proto:1:24: range, 10 to 1, is invalid: start must be <= end`,
		},
		{
			contents: `message Foo { extensions 10 to 1; }`,
			errMsg:   `test.proto:1:26: range, 10 to 1, is invalid: start must be <= end`,
		},
		{
			contents: `message Foo { reserved 1 to 5000000000; }`,
			errMsg:   `test.proto:1:29: range end 5000000000 is out of range: should be between 0 and 536870911`,
		},
		{
			contents: `message Foo { extensions 3000000000; }`,
			errMsg:   `test.proto:1:26: range start 3000000000 is out of range: should be between 0 and 536870911`,
		},
		{
			contents: `message Foo { extensions 3000000000 to 3000000001; }`,
			errMsg:   `test.proto:1:26: range start 3000000000 is out of range: should be between 0 and 536870911`,
		},
		{
			contents: `message Foo { extensions 100 to 3000000000; }`,
			errMsg:   `test.proto:1:33: range end 3000000000 is out of range: should be between 0 and 536870911`,
		},
		{
			contents: `message Foo { reserved "foo", "foo"; }`,
			errMsg:   `test.proto:1:31: name "foo" is reserved multiple times`,
		},
		{
			contents: `message Foo { reserved "foo"; reserved "foo"; }`,
			errMsg:   `test.proto:1:40: name "foo" is reserved multiple times`,
		},
		{
			contents: `message Foo { reserved "foo"; optional string foo = 1; }`,
			errMsg:   `test.proto:1:47: message Foo: field foo is using a reserved name`,
		},
		{
			contents: `message Foo { reserved 1 to 10; optional string foo = 1; }`,
			errMsg:   `test.proto:1:55: message Foo: field foo is using tag 1 which is in reserved range 1 to 10`,
		},
		{
			contents: `message Foo { extensions 1 to 10; optional string foo = 1; }`,
			errMsg:   `test.proto:1:57: message Foo: field foo is using tag 1 which is in extension range 1 to 10`,
		},
		{
			contents: `message Foo { optional group foo = 1 { } }`,
			errMsg:   `test.proto:1:30: group foo should have a name that starts with a capital letter`,
		},
		{
			contents: `message Foo { oneof foo { group bar = 1 { } } }`,
			errMsg:   `test.proto:1:33: group bar should have a name that starts with a capital letter`,
		},
	}

	for i, tc := range testCases {
		errs := newErrorHandler(nil, nil)
		_ = parseProto("test.proto", strings.NewReader(tc.contents), errs, true)
		err := errs.getError()
		if tc.succeeds {
			testutil.Ok(t, err, "case #%d should succeed", i)
		} else {
			testutil.Nok(t, err, "case #%d should fail", i)
			testutil.Eq(t, tc.errMsg, err.Error(), "case #%d bad error message", i)
		}
	}
}
