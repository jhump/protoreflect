// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	//"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	//"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/internal/protoc"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/reporter"
)

func TestBasicValidation(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		contents string
		// Expected error message - leave empty if input is expected to succeed
		expectedErr            string
		expectedDiffWithProtoc bool
	}{
		"success_large_negative_integer": {
			contents: `message Foo { optional double bar = 1 [default = -18446744073709551615]; }`,
		},
		"success_large_negative_integer_bom": {
			// with byte order marker
			contents: string([]byte{0xEF, 0xBB, 0xBF}) + `message Foo { optional double bar = 1 [default = -18446744073709551615]; }`,
		},
		"success_large_positive_integer": {
			contents: `message Foo { optional double bar = 1 [default = 18446744073709551616]; }`,
		},
		"success_message_set_wire_format_w_ext": {
			contents: `message Foo { extensions 100 to max; option message_set_wire_format = true; } message Bar { } extend Foo { optional Bar bar = 536870912; }`,
		},
		"success_message_set_wire_format": {
			contents: `message Foo { option message_set_wire_format = true; extensions 1 to 100; }`,
		},
		"failure_message_set_wire_format_in_proto3": {
			contents:    `syntax = "proto3"; message Foo { option message_set_wire_format = true; extensions 1 to 100; }`,
			expectedErr: "test.proto:1:34: messages with message-set wire format are not allowed with proto3 syntax",
		},
		"success_message_set_wire_format_in_editions": {
			contents: `edition = "2023"; message Foo { option message_set_wire_format = true; extensions 1 to 100; }`,
		},
		"failure_message_set_wire_format_non_ext_field": {
			contents:    `message Foo { optional double bar = 536870912; option message_set_wire_format = true; }`,
			expectedErr: "test.proto:1:15: messages with message-set wire format cannot contain non-extension fields",
		},
		"failure_message_set_wire_format_no_extension_range": {
			contents:    `message Foo { option message_set_wire_format = true; }`,
			expectedErr: "test.proto:1:15: messages with message-set wire format must contain at least one extension range",
			// protoc allows this, ostensibly for empty messages?
			// We disallow it since the Go runtime does not support descriptors that look this way:
			//   https://github.com/protocolbuffers/protobuf-go/blob/6d0a5dbd95005b70501b4cc2c5124dab07a1f4a0/reflect/protodesc/desc_validate.go#L110-L112
			expectedDiffWithProtoc: true,
		},
		"success_oneof_w_group": {
			contents: `message Foo { oneof bar { group Baz = 1 [deprecated=true] { optional int32 abc = 1; } } }`,
		},
		"failure_bad_syntax": {
			contents:    `syntax = "proto1";`,
			expectedErr: `test.proto:1:10: syntax value must be "proto2" or "proto3"`,
		},
		"failure_field_number_out_of_range": {
			contents:    `message Foo { optional string s = 5000000000; }`,
			expectedErr: `test.proto:1:35: tag number 5000000000 is higher than max allowed tag number (536870911)`,
		},
		"failure_field_number_reserved": {
			contents:    `message Foo { optional string s = 19500; }`,
			expectedErr: `test.proto:1:35: tag number 19500 is in disallowed reserved range 19000-19999`,
		},
		"failure_enum_value_number_out_of_range": {
			contents:    `enum Foo { V = 5000000000; }`,
			expectedErr: `test.proto:1:16: value 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_value_number_out_of_range_negative": {
			contents:    `enum Foo { V = -5000000000; }`,
			expectedErr: `test.proto:1:16: value -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_start_out_of_range": {
			contents:    `enum Foo { V = 0; reserved 5000000000; }`,
			expectedErr: `test.proto:1:28: range start 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_start_out_of_range_negative": {
			contents:    `enum Foo { V = 0; reserved -5000000000; }`,
			expectedErr: `test.proto:1:28: range start -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_both_out_of_range": {
			contents:    `enum Foo { V = 0; reserved 5000000000 to 5000000001; }`,
			expectedErr: `test.proto:1:28: range start 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_end_out_of_range": {
			contents:    `enum Foo { V = 0; reserved 5 to 5000000000; }`,
			expectedErr: `test.proto:1:33: range end 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_start_out_of_range_negative2": {
			contents:    `enum Foo { V = 0; reserved -5000000000 to -5; }`,
			expectedErr: `test.proto:1:28: range start -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_both_out_of_range_negative": {
			contents:    `enum Foo { V = 0; reserved -5000000001 to -5000000000; }`,
			expectedErr: `test.proto:1:28: range start -5000000001 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_end_out_of_range_negative": {
			contents:    `enum Foo { V = 0; reserved -5000000000 to 5; }`,
			expectedErr: `test.proto:1:28: range start -5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_reserved_end_out_of_range2": {
			contents:    `enum Foo { V = 0; reserved -5 to 5000000000; }`,
			expectedErr: `test.proto:1:34: range end 5000000000 is out of range: should be between -2147483648 and 2147483647`,
		},
		"failure_enum_without_value": {
			contents:    `enum Foo { }`,
			expectedErr: `test.proto:1:1: enum Foo: enums must define at least one value`,
		},
		"failure_oneof_without_field": {
			contents:    `message Foo { oneof Bar { } }`,
			expectedErr: `test.proto:1:15: oneof must contain at least one field`,
		},
		"failure_extend_without_field": {
			contents:    `message Foo { extensions 1 to max; } extend Foo { }`,
			expectedErr: `test.proto:1:38: extend sections must define at least one extension`,
		},
		"failure_explicit_map_entry_option": {
			contents:    `message Foo { option map_entry = true; }`,
			expectedErr: `test.proto:1:22: message Foo: map_entry option should not be set explicitly; use map type instead`,
		},
		"failure_explicit_map_entry_option_false": {
			contents:    `message Foo { option map_entry = false; }`,
			expectedErr: `test.proto:1:22: message Foo: map_entry option should not be set explicitly; use map type instead`,
		},
		"failure_proto2_requires_label": {
			contents:    `syntax = "proto2"; message Foo { string s = 1; }`,
			expectedErr: `test.proto:1:41: field Foo.s: field has no label; proto2 requires explicit 'optional' label`,
		},
		"failure_proto2_requires_label2": {
			contents:    `message Foo { string s = 1; }`, // syntax defaults to proto2
			expectedErr: `test.proto:1:22: field Foo.s: field has no label; proto2 requires explicit 'optional' label`,
		},
		"success_proto3_optional": {
			contents: `syntax = "proto3"; message Foo { optional string s = 1; }`,
		},
		"success_proto3_optional_ext": {
			contents: `syntax = "proto3"; import "google/protobuf/descriptor.proto"; extend google.protobuf.MessageOptions { optional string s = 50000; }`,
		},
		"failure_proto3_required": {
			contents:    `syntax = "proto3"; message Foo { required string s = 1; }`,
			expectedErr: `test.proto:1:34: field Foo.s: label 'required' is not allowed in proto3 or editions`,
		},
		"failure_editions_required": {
			contents:    `edition = "2023"; message Foo { required string s = 1; }`,
			expectedErr: `test.proto:1:33: field Foo.s: label 'required' is not allowed in proto3 or editions`,
		},
		"failure_extension_required": {
			contents:    `message Foo { extensions 1 to max; } extend Foo { required string sss = 100; }`,
			expectedErr: `test.proto:1:51: extension sss: extension fields cannot be 'required'`,
		},
		"failure_proto3_group": {
			contents:    `syntax = "proto3"; message Foo { optional group Grp = 1 { } }`,
			expectedErr: `test.proto:1:43: field Foo.grp: groups are not allowed in proto3 or editions`,
		},
		"failure_proto3_extension_range": {
			contents:    `syntax = "proto3"; message Foo { extensions 1 to max; }`,
			expectedErr: `test.proto:1:45: message Foo: extension ranges are not allowed in proto3`,
		},
		"failure_proto3_default": {
			contents:    `syntax = "proto3"; message Foo { string s = 1 [default = "abcdef"]; }`,
			expectedErr: `test.proto:1:48: field Foo.s: default values are not allowed in proto3`,
		},
		"failure_editions_group": {
			contents:    `edition = "2023"; message Foo { optional group Grp = 1 { } }`,
			expectedErr: `test.proto:1:42: field Foo.grp: groups are not allowed in proto3 or editions`,
		},
		"success_editions_extension_range": {
			contents: `edition = "2023"; message Foo { extensions 1 to max; }`,
		},
		"success_editions_default": {
			contents: `edition = "2023"; message Foo { string s = 1 [default = "abcdef"]; }`,
		},
		"failure_editions_optional": {
			contents:    `edition = "2023"; message Foo { optional string name = 1; }`,
			expectedErr: `test.proto:1:33: field Foo.name: label 'optional' is not allowed in editions; use option features.field_presence instead`,
		},
		"failure_editions_optional_ext": {
			contents:    `edition = "2023"; import "google/protobuf/descriptor.proto"; extend google.protobuf.MessageOptions { optional string s = 50000; }`,
			expectedErr: `test.proto:1:102: extension s: label 'optional' is not allowed in editions; use option features.field_presence instead`,
		},
		"failure_enum_value_number_duplicate": {
			contents:    `enum Foo { V1 = 1; V2 = 1; }`,
			expectedErr: `test.proto:1:25: enum Foo: values V1 and V2 both have the same numeric value 1; use allow_alias option if intentional`,
		},
		"success_enum_allow_alias_true": {
			contents: `enum Foo { option allow_alias = true; V1 = 1; V2 = 1; }`,
		},
		"success_enum_allow_alias_false": {
			contents:               `enum Foo { option allow_alias = false; V1 = 1; V2 = 2; }`,
			expectedDiffWithProtoc: true, // strange that protoc disallows this;
			// TODO: update protocompile to reject explicit allow_alias=false to match protoc
		},
		"failure_enum_allow_alias": {
			contents:    `enum Foo { option allow_alias = true; V1 = 1; V2 = 2; }`,
			expectedErr: `test.proto:1:33: enum Foo: allow_alias is true but no values are aliases`,
		},
		"success_enum_reserved": {
			contents: `syntax = "proto3"; enum Foo { V1 = 0; reserved 1 to 20; reserved "V2"; }`,
		},
		"failure_enum_value_in_reserved_range": {
			contents:    `enum Foo { V1 = 1; reserved 1 to 20; reserved "V2"; }`,
			expectedErr: `test.proto:1:17: enum Foo: value V1 is using number 1 which is in reserved range 1 to 20`,
		},
		"failure_enum_value_in_reserved_range2": {
			contents:    `enum Foo { V1 = 20; reserved 1 to 20; reserved "V2"; }`,
			expectedErr: `test.proto:1:17: enum Foo: value V1 is using number 20 which is in reserved range 1 to 20`,
		},
		"failure_enum_value_w_reserved_name": {
			contents:    `enum Foo { V2 = 0; reserved 1 to 20; reserved "V2"; }`,
			expectedErr: `test.proto:1:12: enum Foo: value V2 is using a reserved name`,
		},
		"success_enum_reserved2": {
			contents: `enum Foo { V0 = 0; reserved 1 to 20; reserved 21 to 40; reserved "V2"; }`,
		},
		"failure_enum_reserved_overlap": {
			contents:    `enum Foo { V0 = 0; reserved 1 to 20; reserved 20 to 40; reserved "V2"; }`,
			expectedErr: `test.proto:1:47: enum Foo: reserved ranges overlap: 1 to 20 and 20 to 40`,
		},
		"failure_proto3_enum_zero_value": {
			contents:    `syntax = "proto3"; enum Foo { FIRST = 1; }`,
			expectedErr: `test.proto:1:39: enum Foo: proto3 requires that first value of enum have numeric value zero`,
		},
		"failure_message_number_conflict": {
			contents:    `syntax = "proto3"; message Foo { string s = 1; int32 i = 1; }`,
			expectedErr: `test.proto:1:58: message Foo: fields s and i both have the same tag 1`,
		},
		"failure_message_reserved_overlap": {
			contents:    `message Foo { reserved 1 to 10, 10 to 12; }`,
			expectedErr: `test.proto:1:33: message Foo: reserved ranges overlap: 1 to 10 and 10 to 12`,
		},
		"failure_message_extensions_overlap": {
			contents:    `message Foo { extensions 1 to 10, 10 to 12; }`,
			expectedErr: `test.proto:1:35: message Foo: extension ranges overlap: 1 to 10 and 10 to 12`,
		},
		"failure_message_reserved_extensions_overlap": {
			contents:    `message Foo { reserved 1 to 10; extensions 10 to 12; }`,
			expectedErr: `test.proto:1:44: message Foo: extension range 10 to 12 overlaps reserved range 1 to 10`,
		},
		"success_message_reserved_extensions": {
			contents: `message Foo { reserved 1, 2 to 10, 11 to 20; extensions 21 to 22; }`,
		},
		"failure_message_reserved_start_after_end": {
			contents:    `message Foo { reserved 10 to 1; }`,
			expectedErr: `test.proto:1:24: range, 10 to 1, is invalid: start must be <= end`,
		},
		"failure_message_extensions_start_after_end": {
			contents:    `message Foo { extensions 10 to 1; }`,
			expectedErr: `test.proto:1:26: range, 10 to 1, is invalid: start must be <= end`,
		},
		"failure_message_reserved_end_out_of_range": {
			contents:    `message Foo { reserved 1 to 5000000000; }`,
			expectedErr: `test.proto:1:29: range end 5000000000 is out of range: should be between 1 and 536870911`,
		},
		"failure_message_reserved_start_out_of_range": {
			contents:    `message Foo { reserved 0 to 10; }`,
			expectedErr: `test.proto:1:24: range start 0 is out of range: should be between 1 and 536870911`,
		},
		"failure_message_extensions_start_out_of_range": {
			contents:    `message Foo { extensions 3000000000; }`,
			expectedErr: `test.proto:1:26: range start 3000000000 is out of range: should be between 1 and 536870911`,
		},
		"failure_message_extensions_both_out_of_range": {
			contents:    `message Foo { extensions 3000000000 to 3000000001; }`,
			expectedErr: `test.proto:1:26: range start 3000000000 is out of range: should be between 1 and 536870911`,
		},
		"failure_message_extensions_start_out_of_range2": {
			contents:    `message Foo { extensions 0 to 10; }`,
			expectedErr: `test.proto:1:26: range start 0 is out of range: should be between 1 and 536870911`,
		},
		"failure_message_extensions_end_out_of_range": {
			contents:    `message Foo { extensions 100 to 3000000000; }`,
			expectedErr: `test.proto:1:33: range end 3000000000 is out of range: should be between 1 and 536870911`,
		},
		"failure_message_reserved_name_duplicate": {
			contents:    `message Foo { reserved "foo", "foo"; }`,
			expectedErr: `test.proto:1:31: name "foo" is already reserved at test.proto:1:24`,
		},
		"failure_message_reserved_name_duplicate2": {
			contents:    `message Foo { reserved "foo"; reserved "foo"; }`,
			expectedErr: `test.proto:1:40: name "foo" is already reserved at test.proto:1:24`,
		},
		"failure_message_field_w_reserved_name": {
			contents:    `message Foo { reserved "foo"; optional string foo = 1; }`,
			expectedErr: `test.proto:1:47: message Foo: field foo is using a reserved name`,
		},
		"failure_message_field_w_reserved_number": {
			contents:    `message Foo { reserved 1 to 10; optional string foo = 1; }`,
			expectedErr: `test.proto:1:55: message Foo: field foo is using tag 1 which is in reserved range 1 to 10`,
		},
		"failure_message_field_w_number_in_ext_range": {
			contents:    `message Foo { extensions 1 to 10; optional string foo = 1; }`,
			expectedErr: `test.proto:1:57: message Foo: field foo is using tag 1 which is in extension range 1 to 10`,
		},
		"failure_group_name": {
			contents:    `message Foo { optional group foo = 1 { } }`,
			expectedErr: `test.proto:1:30: group foo should have a name that starts with a capital letter`,
		},
		"failure_oneof_group_name": {
			contents:    `message Foo { oneof foo { group bar = 1 { } } }`,
			expectedErr: `test.proto:1:33: group bar should have a name that starts with a capital letter`,
		},
		"failure_message_decl_start_w_option": {
			contents:    `enum Foo { option = 1; }`,
			expectedErr: `test.proto:1:19: syntax error: unexpected '='`,
		},
		"failure_message_decl_start_w_reserved": {
			contents:    `enum Foo { reserved = 1; }`,
			expectedErr: `test.proto:1:21: syntax error: unexpected '='`,
		},
		"failure_message_decl_start_w_message": {
			contents:    `syntax = "proto3"; enum message { unset = 0; } message Foo { message bar = 1; }`,
			expectedErr: `test.proto:1:74: syntax error: unexpected '=', expecting '{'`,
		},
		"failure_message_decl_start_w_enum": {
			contents:    `syntax = "proto3"; enum enum { unset = 0; } message Foo { enum bar = 1; }`,
			expectedErr: `test.proto:1:68: syntax error: unexpected '=', expecting '{'`,
		},
		"failure_message_decl_start_w_reserved2": {
			contents:    `syntax = "proto3"; enum reserved { unset = 0; } message Foo { reserved bar = 1; }`,
			expectedErr: `test.proto:1:76: syntax error: expecting ';'`,
		},
		"failure_message_decl_start_w_extend": {
			contents:    `syntax = "proto3"; enum extend { unset = 0; } message Foo { extend bar = 1; }`,
			expectedErr: `test.proto:1:72: syntax error: unexpected '=', expecting '{'`,
		},
		"failure_message_decl_start_w_oneof": {
			contents:    `syntax = "proto3"; enum oneof { unset = 0; } message Foo { oneof bar = 1; }`,
			expectedErr: `test.proto:1:70: syntax error: unexpected '=', expecting '{'`,
		},
		"failure_message_decl_start_w_optional": {
			contents:    `syntax = "proto3"; enum optional { unset = 0; } message Foo { optional bar = 1; }`,
			expectedErr: `test.proto:1:76: syntax error: unexpected '='`,
		},
		"failure_message_decl_start_w_repeated": {
			contents:    `syntax = "proto3"; enum repeated { unset = 0; } message Foo { repeated bar = 1; }`,
			expectedErr: `test.proto:1:76: syntax error: unexpected '='`,
		},
		"failure_message_decl_start_w_required": {
			contents:    `syntax = "proto3"; enum required { unset = 0; } message Foo { required bar = 1; }`,
			expectedErr: `test.proto:1:76: syntax error: unexpected '='`,
		},
		"failure_extend_decl_start_w_optional": {
			contents:    `syntax = "proto3"; import "google/protobuf/descriptor.proto"; enum optional { unset = 0; } extend google.protobuf.MethodOptions { optional bar = 22222; }`,
			expectedErr: `test.proto:1:144: syntax error: unexpected '='`,
		},
		"failure_extend_decl_start_w_repeated": {
			contents:    `syntax = "proto3"; import "google/protobuf/descriptor.proto"; enum repeated { unset = 0; } extend google.protobuf.MethodOptions { repeated bar = 22222; }`,
			expectedErr: `test.proto:1:144: syntax error: unexpected '='`,
		},
		"failure_extend_decl_start_w_required": {
			contents:    `syntax = "proto3"; import "google/protobuf/descriptor.proto"; enum required { unset = 0; } extend google.protobuf.MethodOptions { required bar = 22222; }`,
			expectedErr: `test.proto:1:144: syntax error: unexpected '='`,
		},
		"failure_oneof_decl_start_w_optional": {
			contents:    `syntax = "proto3"; enum optional { unset = 0; } message Foo { oneof bar { optional bar = 1; } }`,
			expectedErr: `test.proto:1:75: syntax error: unexpected "optional"`,
		},
		"failure_oneof_decl_start_w_repeated": {
			contents:    `syntax = "proto3"; enum repeated { unset = 0; } message Foo { oneof bar { repeated bar = 1; } }`,
			expectedErr: `test.proto:1:75: syntax error: unexpected "repeated"`,
		},
		"failure_oneof_decl_start_w_required": {
			contents:    `syntax = "proto3"; enum required { unset = 0; } message Foo { oneof bar { required bar = 1; } }`,
			expectedErr: `test.proto:1:75: syntax error: unexpected "required"`,
		},
		"success_empty": {
			contents: ``,
		},
		"failure_junk_token": {
			contents:    `0`,
			expectedErr: `test.proto:1:1: syntax error: unexpected int literal`,
		},
		"failure_junk_token2": {
			contents:    `foobar`,
			expectedErr: `test.proto:1:1: syntax error: unexpected identifier`,
		},
		"failure_junk_token3": {
			contents:    `Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.`,
			expectedErr: `test.proto:1:1: syntax error: unexpected identifier`,
		},
		"failure_junk_token4": {
			contents:    `"abc"`,
			expectedErr: `test.proto:1:1: syntax error: unexpected string literal`,
		},
		"failure_junk_token5": {
			contents:    `0.0.0.0.0`,
			expectedErr: `test.proto:1:1: invalid syntax in float value: 0.0.0.0.0`,
		},
		"failure_junk_token6": {
			contents:    `0.0`,
			expectedErr: `test.proto:1:1: syntax error: unexpected float literal`,
		},
		"success_colon_before_list_literal": {
			contents: `import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions { optional Opt opt = 10101; }
					   message Opt { map<string,Opt> m = 1; }
					   option (opt) = {m: [{key: "a",value: {}}]};`,
		},
		"success_no_colon_before_list_literal": {
			contents: `import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions { optional Opt opt = 10101; }
					   message Opt { map<string,Opt> m = 1; }
					   option (opt) = {m [{key: "a",value: {}}]};`,
		},
		"success_colon_before_list_literal2": {
			contents: `import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions { optional Opt opt = 10101; }
					   message Opt { map<string,Opt> m = 1; }
					   option (opt) = {m: []};`,
		},
		"success_no_colon_before_list_literal2": {
			contents: `import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions { optional Opt opt = 10101; }
					   message Opt { map<string,Opt> m = 1; }
					   option (opt) = {m []};`,
		},
		"failure_duplicate_import": {
			contents:    `syntax = "proto3"; import "google/protobuf/descriptor.proto"; import "google/protobuf/descriptor.proto";`,
			expectedErr: `test.proto:1:63: "google/protobuf/descriptor.proto" was already imported at test.proto:1:20`,
		},
		"success_long_package_name": {
			contents: `syntax = "proto3"; package a012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789;`,
		},
		"failure_long_package_name": {
			contents:    `syntax = "proto3"; package ab012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890123456789;`,
			expectedErr: `test.proto:1:28: package name (with whitespace removed) must be less than 512 characters long`,
		},
		"success_long_package_name2": {
			contents: `syntax = "proto3"; package a .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789;`,
		},
		"failure_long_package_name2": {
			contents:    `syntax = "proto3"; package ab .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789 .  a23456789;`,
			expectedErr: `test.proto:1:28: package name (with whitespace removed) must be less than 512 characters long`,
		},
		"success_long_package_name3": {
			contents: `syntax = "proto3"; package a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1;`,
		},
		"failure_long_package_name3": {
			contents:    `syntax = "proto3"; package a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2.a3.a4.a5.a6.a7.a8.a9.a0.a1.a2;`,
			expectedErr: `test.proto:1:28: package name may not contain more than 100 periods`,
		},
		"success_deep_nesting": {
			contents: `syntax = "proto3";
					   message _01 { message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 {
					   } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }`,
		},
		"failure_deep_nesting_message1": {
			contents: `syntax = "proto3";
					   message _01 { message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 { message _32 {
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }`,
			expectedErr: `test.proto:9:86: message nesting depth must be less than 32`,
		},
		"failure_deep_nesting_message2": {
			contents: `syntax = "proto3";
					   message _01 { message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 { message _32 {
					   message _33 { }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }`,
			expectedErr: `test.proto:9:86: message nesting depth must be less than 32`,
		},
		"failure_deep_nesting_map": {
			contents: `syntax = "proto3";
					   message _01 { message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 {
					     map<string, string> m = 1;
					   } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }`,
			expectedErr: `test.proto:10:46: message nesting depth must be less than 32`,
		},
		"failure_deep_nesting_group1": {
			contents: `syntax = "proto2";
					   message _01 { message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 {
					     optional group Foo = 1 { }
					   } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }`,
			expectedErr: `test.proto:10:55: message nesting depth must be less than 32`,
		},
		"failure_deep_nesting_group2": {
			contents: `syntax = "proto2";
					   message _01 { optional group Foo = 1 {
					   message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 {
					   } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } }
					   } }`,
			expectedErr: `test.proto:10:72: message nesting depth must be less than 32`,
		},
		"failure_deep_nesting_extension_group1": {
			contents: `syntax = "proto2";
					   message Ext { extensions 1 to max; }
					   message _01 { message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 {
					     extend Ext {
					       optional group Foo = 1 { }
					     }
					   } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }`,
			expectedErr: `test.proto:12:57: message nesting depth must be less than 32`,
		},
		"failure_deep_nesting_extension_group2": {
			contents: `syntax = "proto2";
					   message Ext { extensions 1 to max; }
					   extend Ext { optional group Foo = 1 {
					   message _01 { message _02 { message _03 { message _04 {
					   message _05 { message _06 { message _07 { message _08 {
					   message _09 { message _10 { message _11 { message _12 {
					   message _13 { message _14 { message _15 { message _16 {
					   message _17 { message _18 { message _19 { message _20 {
					   message _21 { message _22 { message _23 { message _24 {
					   message _25 { message _26 { message _27 { message _28 {
					   message _29 { message _30 { message _31 {
					   } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } } } }
					   } }`,
			expectedErr: `test.proto:11:72: message nesting depth must be less than 32`,
		},
		"failure_positive_sign_not_allowed_in_default_val": {
			contents: `syntax = "proto3";
					   message Foo {
					     int32 bar = 1 [default = +123];
					   }`,
			expectedErr: `test.proto:3:71: syntax error: unexpected '+'`,
		},
		"failure_positive_sign_not_allowed_in_enum_val": {
			contents: `syntax = "proto3";
					   enum Foo {
					     BAR = +1;
					   }`,
			expectedErr: `test.proto:3:52: syntax error: unexpected '+', expecting int literal or '-'`,
		},
		"failure_positive_sign_not_allowed_in_message_literal": {
			contents: `syntax = "proto3";
					   import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions {
					     Foo foo = 10101;
					   }
					   message Foo {
					     repeated float bar = 1;
					   }
					   option (foo) = { bar: +1.01 bar: +inf };`,
			expectedErr: `test.proto:9:66: syntax error: unexpected '+'`,
		},
		"success_inf_nan_in_default_value": {
			contents: `syntax = "proto2";
					   message Foo {
					     optional float bar = 1 [default = inf];
					     optional double baz = 2 [default = nan];
					     optional double fizz = 3 [default = -inf];
					     optional float buzz = 4 [default = -nan];
					   }`,
		},
		"failure_inf_upper_in_default_value": {
			contents: `syntax = "proto2";
					   message Foo {
					     optional float bar = 1 [default = -Inf];
					   }`,
			expectedErr: `test.proto:3:81: syntax error: unexpected identifier, expecting int literal or float literal or "inf" or "nan"`,
		},
		"failure_infinity_upper_in_default_value": {
			contents: `syntax = "proto2";
					   message Foo {
					     optional float bar = 1 [default = -infinity];
					   }`,
			expectedErr: `test.proto:3:81: syntax error: unexpected identifier, expecting int literal or float literal or "inf" or "nan"`,
		},
		"failure_nan_upper_in_default_value": {
			contents: `syntax = "proto2";
					   message Foo {
					     optional float bar = 1 [default = -NaN];
					   }`,
			expectedErr: `test.proto:3:81: syntax error: unexpected identifier, expecting int literal or float literal or "inf" or "nan"`,
		},
		"success_inf_nan_in_option_value": {
			contents: `syntax = "proto3";
					   import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions {
					     repeated double foo = 10101;
					   }
					   option (foo) = inf;
					   option (foo) = -inf;
					   option (foo) = nan;
					   option (foo) = -nan;`,
		},
		"failure_inf_upper_in_option_value": {
			contents: `syntax = "proto2";
					   import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions {
					     repeated double foo = 10101;
					   }
					   option (foo) = -Inf;`,
			expectedErr: `test.proto:6:60: syntax error: unexpected identifier, expecting int literal or float literal or "inf" or "nan"`,
		},
		"failure_infinity_upper_in_option_value": {
			contents: `syntax = "proto2";
					   import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions {
					     repeated double foo = 10101;
					   }
					   option (foo) = -infinity;`,
			expectedErr: `test.proto:6:60: syntax error: unexpected identifier, expecting int literal or float literal or "inf" or "nan"`,
		},
		"failure_nan_upper_in_option_value": {
			contents: `syntax = "proto2";
					   import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions {
					     repeated double foo = 10101;
					   }
					   option (foo) = -NaN;`,
			expectedErr: `test.proto:6:60: syntax error: unexpected identifier, expecting int literal or float literal or "inf" or "nan"`,
		},
		"success_inf_nan_in_message_literal": {
			contents: `syntax = "proto3";
					   import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions {
					     Foo foo = 10101;
					   }
					   message Foo {
					     repeated float bar = 1;
					   }
					   option (foo) = {
					     bar: inf      bar: -inf      bar: INF      bar: -Inf
					     bar: infinity bar: -infinity bar: INFiniTY bar: -Infinity
					     bar: nan      bar: -nan      bar: NAN      bar: -NaN
					   };`,
		},
		"failure_invalid_signed_identifier_in_message_literal": {
			contents: `syntax = "proto3";
					   import "google/protobuf/descriptor.proto";
					   extend google.protobuf.FileOptions {
					     Foo foo = 10101;
					   }
					   message Foo {
					     repeated float bar = 1;
					   }
					   option (foo) = { bar: -Infin };`,
			expectedErr: `test.proto:9:67: only identifiers "inf", "infinity", or "nan" may appear after negative sign`,
		},
		"failure_message_invalid_reserved_name": {
			contents: `syntax = "proto3";
					   message Foo {
					     reserved "foo", "b_a_r9", " blah ";
					   }`,
			expectedErr:            `test.proto:3:72: message Foo: reserved name " blah " is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"failure_message_invalid_reserved_name2": {
			contents: `syntax = "proto3";
					   message Foo {
					     reserved "foo", "_bar123", "123";
					   }`,
			expectedErr:            `test.proto:3:73: message Foo: reserved name "123" is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"failure_message_invalid_reserved_name3": {
			contents: `syntax = "proto3";
					   message Foo {
					     reserved "foo" "_bar123" "@y!!";
					   }`,
			expectedErr:            `test.proto:3:55: message Foo: reserved name "foo_bar123@y!!" is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"failure_message_invalid_reserved_name4": {
			contents: `syntax = "proto3";
					   message Foo {
					     reserved "";
					   }`,
			expectedErr:            `test.proto:3:55: message Foo: reserved name "" is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"success_message_reserved_name_proto2": {
			contents: `syntax = "proto2";
					   message Foo {
					     reserved "foo", "_bar123", "A_B_C_1_2_3";
					   }`,
		},
		"success_message_reserved_name_proto3": {
			contents: `syntax = "proto3";
					   message Foo {
					     reserved "foo", "_bar123", "A_B_C_1_2_3";
					   }`,
		},
		"failure_message_reserved_name_editions": {
			contents: `edition = "2023";
					   message Foo {
					     reserved "foo", "_bar123", "A_B_C_1_2_3";
					   }`,
			expectedErr: `test.proto:3:55: must use identifiers, not string literals, to reserved names with editions`,
		},
		"failure_enum_invalid_reserved_name": {
			contents: `syntax = "proto3";
					   enum Foo {
					     BAR = 0;
					     reserved "foo", "b_a_r9", " blah ";
					   }`,
			expectedErr:            `test.proto:4:72: enum Foo: reserved name " blah " is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"failure_enum_invalid_reserved_name2": {
			contents: `syntax = "proto3";
					   enum Foo {
					     BAR = 0;
					     reserved "foo", "_bar123", "123";
					   }`,
			expectedErr:            `test.proto:4:73: enum Foo: reserved name "123" is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"failure_enum_invalid_reserved_name3": {
			contents: `syntax = "proto3";
					   enum Foo {
					     BAR = 0;
					     reserved "foo" "_bar123" "@y!!";
					   }`,
			expectedErr:            `test.proto:4:55: enum Foo: reserved name "foo_bar123@y!!" is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"failure_enum_invalid_reserved_name4": {
			contents: `syntax = "proto3";
					   enum Foo {
					     BAR = 0;
					     reserved "";
					   }`,
			expectedErr:            `test.proto:4:55: enum Foo: reserved name "" is not a valid identifier`,
			expectedDiffWithProtoc: true, // protoc only warns for invalid reserved names: https://github.com/protocolbuffers/protobuf/issues/6335
		},
		"success_enum_reserved_name_proto2": {
			contents: `syntax = "proto2";
					   enum Foo {
					     BAR = 0;
					     reserved "foo", "_bar123", "A_B_C_1_2_3";
					   }`,
		},
		"success_enum_reserved_name_proto3": {
			contents: `syntax = "proto3";
					   enum Foo {
					     BAR = 0;
					     reserved "foo", "_bar123", "A_B_C_1_2_3";
					   }`,
		},
		"failure_enum_reserved_name_editions": {
			contents: `edition = "2023";
					   enum Foo {
					     BAR = 0;
					     reserved "foo", "_bar123", "A_B_C_1_2_3";
					   }`,
			expectedErr: `test.proto:4:55: must use identifiers, not string literals, to reserved names with editions`,
		},
		"failure_message_reserved_ident_proto2": {
			contents: `syntax = "proto2";
					   message Foo {
					     reserved foo, _bar123, A_B_C_1_2_3;
					   }`,
			expectedErr: `test.proto:3:55: must use string literals, not identifiers, to reserved names with proto2 and proto3`,
		},
		"failure_message_reserved_ident_proto3": {
			contents: `syntax = "proto3";
					   message Foo {
					     reserved foo, _bar123, A_B_C_1_2_3;
					   }`,
			expectedErr: `test.proto:3:55: must use string literals, not identifiers, to reserved names with proto2 and proto3`,
		},
		"success_message_reserved_ident_editions": {
			contents: `edition = "2023";
					   message Foo {
					     reserved foo, _bar123, A_B_C_1_2_3;
					   }`,
		},
		"failure_enum_reserved_ident_proto2": {
			contents: `syntax = "proto2";
					   enum Foo {
					     BAR = 0;
					     reserved foo, _bar123, A_B_C_1_2_3;
					   }`,
			expectedErr: `test.proto:4:55: must use string literals, not identifiers, to reserved names with proto2 and proto3`,
		},
		"failure_enum_reserved_ident_proto3": {
			contents: `syntax = "proto3";
					   enum Foo {
					     BAR = 0;
					     reserved foo, _bar123, A_B_C_1_2_3;
					   }`,
			expectedErr: `test.proto:4:55: must use string literals, not identifiers, to reserved names with proto2 and proto3`,
		},
		"success_enum_reserved_ident_editions": {
			contents: `edition = "2023";
					   enum Foo {
					     BAR = 0;
					     reserved foo, _bar123, A_B_C_1_2_3;
					   }`,
		},
		"failure_use_of_packed_with_editions": {
			contents: `edition = "2023";
					   message Foo {
					     repeated bool foo = 1 [packed=false];
					   }`,
			expectedErr: `test.proto:3:69: field Foo.foo: packed option is not allowed in editions; use option features.repeated_field_encoding instead`,
		},
		"failure_use_of_features_without_editions_file": {
			contents: `syntax = "proto3";
					   option features.utf8_validation = VERIFY;
					   message Foo {
					     string foo = 1;
					   }`,
			expectedErr: `test.proto:2:51: file options: option 'features' may only be used with editions but file uses proto3 syntax`,
		},
		"failure_use_of_features_without_editions_message": {
			contents: `syntax = "proto3";
					   message Foo {
					     option features = {};
					     string foo = 1;
					   }`,
			expectedErr: `test.proto:3:53: message Foo: option 'features' may only be used with editions but file uses proto3 syntax`,
		},
		"failure_use_of_features_without_editions_field": {
			contents: `syntax = "proto3";
					   message Foo {
					     string foo = 1 [features.field_presence = LEGACY_REQUIRED];
					   }`,
			expectedErr: `test.proto:3:62: field Foo.foo: option 'features' may only be used with editions but file uses proto3 syntax`,
		},
		"failure_use_of_features_without_editions_oneof": {
			contents: `syntax = "proto3";
					   message Foo {
					     oneof x {
					       option features = {};
					       string foo = 1;
					     }
					   }`,
			expectedErr: `test.proto:4:55: oneof Foo.x: option 'features' may only be used with editions but file uses proto3 syntax`,
		},
		"failure_use_of_features_without_editions_ext_range": {
			contents: `syntax = "proto2";
					   message Foo {
					     extensions 1 to 100 [features={}];
					   }`,
			expectedErr: `test.proto:3:67: message Foo: option 'features' may only be used with editions but file uses proto2 syntax`,
		},
		"failure_use_of_features_without_editions_enum": {
			contents: `syntax = "proto2";
					   enum Foo {
					     option features.enum_type = CLOSED;
					     VALUE = 0;
					   }`,
			expectedErr: `test.proto:3:53: enum Foo: option 'features' may only be used with editions but file uses proto2 syntax`,
		},
		"failure_use_of_features_without_editions_enum_val": {
			contents: `syntax = "proto2";
					   enum Foo {
					     VALUE = 0 [features={}];
					   }`,
			expectedErr: `test.proto:3:57: enum value VALUE: option 'features' may only be used with editions but file uses proto2 syntax`,
		},
		"failure_use_of_features_without_editions_service": {
			contents: `syntax = "proto3";
					   message Foo {}
					   service FooService {
					     option features = {};
					     rpc Do(Foo) returns (foo);
					   }
					   `,
			expectedErr: `test.proto:4:53: service FooService: option 'features' may only be used with editions but file uses proto3 syntax`,
		},
		"failure_use_of_features_without_editions_method": {
			contents: `syntax = "proto3";
					   message Foo {}
					   service FooService {
					     rpc Do(Foo) returns (foo) {
					       option features = {};
					     }
					   }
					   `,
			expectedErr: `test.proto:5:55: method FooService.Do: option 'features' may only be used with editions but file uses proto3 syntax`,
		},
		"failure_edition_2024_import_option_not_supported": {
			contents: `edition = "2024";
					   import option "google/protobuf/descriptor.proto";
					   message Foo { string name = 1; }`,
			expectedErr:            `test.proto:1:11: edition "2024" not yet fully supported; latest supported edition "2023"`,
			expectedDiffWithProtoc: true,
		},
		"failure_edition_2024_import_option_with_regular_import_not_supported": {
			contents: `edition = "2024";
					   import "google/protobuf/empty.proto";
					   import option "google/protobuf/descriptor.proto";
					   message Foo { string name = 1; }`,
			expectedErr:            `test.proto:1:11: edition "2024" not yet fully supported; latest supported edition "2023"`,
			expectedDiffWithProtoc: true,
		},
		"failure_edition_2024_multiple_import_option_not_supported": {
			contents: `edition = "2024";
					   import option "google/protobuf/descriptor.proto";
					   import option "google/protobuf/cpp_features.proto";
					   message Foo { string name = 1; }`,
			expectedErr:            `test.proto:1:11: edition "2024" not yet fully supported; latest supported edition "2023"`,
			expectedDiffWithProtoc: true,
		},
		"failure_edition_2023_import_option": {
			contents: `edition = "2023";
					   import option "google/protobuf/descriptor.proto";
					   message Foo { string name = 1; }`,
			expectedErr: `test.proto:2:51: import option syntax is only allowed in edition 2024`,
		},
		"failure_proto2_import_option": {
			contents: `syntax = "proto2";
					   import option "google/protobuf/descriptor.proto";
					   message Foo { optional string name = 1; }`,
			expectedErr: `test.proto:2:51: import option syntax is only allowed in edition 2024`,
		},
		"failure_proto3_import_option": {
			contents: `syntax = "proto3";
					   import option "google/protobuf/descriptor.proto";
					   message Foo { string name = 1; }`,
			expectedErr: `test.proto:2:51: import option syntax is only allowed in edition 2024`,
		},
		"failure_edition_2024_export_local_not_supported": {
			contents: `edition = "2024";
					   // Top-level symbols are exported by default in Edition 2024
					   local message LocalMessage {
					     // Nested symbols are local by default in Edition 2024
					     export enum ExportedNestedEnum {
					       UNKNOWN_EXPORTED_NESTED_ENUM_VALUE = 0;
					     }
					   }`,
			expectedErr:            `test.proto:1:11: edition "2024" not yet fully supported; latest supported edition "2023"`,
			expectedDiffWithProtoc: true,
		},
		"failure_edition_2024_export_local_service_invalid": {
			contents: `edition = "2024";
					   local service LocalService {}`,
			expectedErr: `test.proto:2:50: syntax error: unexpected "service", expecting "enum" or "message"`,
		},
		"failure_edition_2023_export_message": {
			contents: `edition = "2023";
					   export message ExportedMessage {
					     string name = 1;
					   }`,
			expectedErr: `test.proto:2:44: export keyword is only allowed in edition 2024`,
		},
		"failure_edition_2023_local_message": {
			contents: `edition = "2023";
					   local message LocalMessage {
					     string name = 1;
					   }`,
			expectedErr: `test.proto:2:44: local keyword is only allowed in edition 2024`,
		},
		"failure_edition_2023_export_enum": {
			contents: `edition = "2023";
					   export enum ExportedEnum {
					     UNKNOWN_EXPORTED_ENUM_VALUE = 0;
					   }`,
			expectedErr: `test.proto:2:44: export keyword is only allowed in edition 2024`,
		},
		"failure_edition_2023_local_enum": {
			contents: `edition = "2023";
					   local enum LocalEnum {
					     UNKNOWN_LOCAL_ENUM_VALUE = 0;
					   }`,
			expectedErr: `test.proto:2:44: local keyword is only allowed in edition 2024`,
		},
		"failure_proto2_export_message": {
			contents: `syntax = "proto2";
					   export message ExportedMessage {
					     optional string name = 1;
					   }`,
			expectedErr: `test.proto:2:44: export keyword is only allowed in edition 2024`,
		},
		"failure_proto2_local_message": {
			contents: `syntax = "proto2";
					   local message LocalMessage {
					     optional string name = 1;
					   }`,
			expectedErr: `test.proto:2:44: local keyword is only allowed in edition 2024`,
		},
		"failure_proto2_export_enum": {
			contents: `syntax = "proto2";
					   export enum ExportedEnum {
					     UNKNOWN_EXPORTED_ENUM_VALUE = 0;
					   }`,
			expectedErr: `test.proto:2:44: export keyword is only allowed in edition 2024`,
		},
		"failure_proto2_local_enum": {
			contents: `syntax = "proto2";
					   local enum LocalEnum {
					     UNKNOWN_LOCAL_ENUM_VALUE = 0;
					   }`,
			expectedErr: `test.proto:2:44: local keyword is only allowed in edition 2024`,
		},
		"failure_proto3_export_message": {
			contents: `syntax = "proto3";
					   export message ExportedMessage {
					     string name = 1;
					   }`,
			expectedErr: `test.proto:2:44: export keyword is only allowed in edition 2024`,
		},
		"failure_proto3_local_message": {
			contents: `syntax = "proto3";
					   local message LocalMessage {
					     string name = 1;
					   }`,
			expectedErr: `test.proto:2:44: local keyword is only allowed in edition 2024`,
		},
		"failure_proto3_export_enum": {
			contents: `syntax = "proto3";
					   export enum ExportedEnum {
					     UNKNOWN_EXPORTED_ENUM_VALUE = 0;
					   }`,
			expectedErr: `test.proto:2:44: export keyword is only allowed in edition 2024`,
		},
		"failure_proto3_local_enum": {
			contents: `syntax = "proto3";
					   local enum LocalEnum {
					     UNKNOWN_LOCAL_ENUM_VALUE = 0;
					   }`,
			expectedErr: `test.proto:2:44: local keyword is only allowed in edition 2024`,
		},
		"failure_proto3_nested_export_enum": {
			contents: `syntax = "proto3";
					   message Container {
					     export enum ExportedNestedEnum {
					       UNKNOWN_EXPORTED_NESTED_ENUM_VALUE = 0;
					     }
					   }`,
			expectedErr: `test.proto:3:46: export keyword is only allowed in edition 2024`,
		},
		"failure_proto3_nested_local_message": {
			contents: `syntax = "proto3";
					   message Container {
					     local message LocalNestedMessage {
					       string name = 1;
					     }
					   }`,
			expectedErr: `test.proto:3:46: local keyword is only allowed in edition 2024`,
		},
		"failure_edition_2024_export_as_type": {
			contents: `edition = "2024";
					   package export;
					   message Message {
					     export.Message field = 1;
					   }`,
			// TODO: Protoc edition 2024 will error on resvered visibility keyword "export".
			// Since protocompile does not support 2024, we instead error on the edition value.
			expectedErr: `test.proto:1:11: edition "2024" not yet fully supported; latest supported edition "2023"`,
		},
		"failure_edition_2024_local_as_type": {
			contents: `edition = "2024";
					   package local;
					   message Message {
					     local.Message field = 1;
					   }`,
			// TODO: Protoc edition 2024 will error on resvered visibility keyword "export".
			// Since protocompile does not support 2024, we instead error on the edition value.
			expectedErr: `test.proto:1:11: edition "2024" not yet fully supported; latest supported edition "2023"`,
		},
		"success_proto3_export_local_as_field_names": {
			contents: `syntax = "proto3";
					   message Test {
					     string export = 1;
					     string local = 2;
					   }`,
		},
		"success_proto2_export_local_as_field_names": {
			contents: `syntax = "proto2";
					   message Test {
					     optional string export = 1;
					     optional string local = 2;
					   }`,
		},
		"success_edition_2023_export_local_as_field_names": {
			contents: `edition = "2023";
					   message Test {
					     string export = 1;
					     string local = 2;
					   }`,
		},
		"success_proto3_export_local_as_type_names": {
			contents: `syntax = "proto3";
					   package local;
					   message export {
					     local.export field = 1;
					   }`,
		},
		"success_proto2_export_local_as_type_names": {
			contents: `syntax = "proto2";
					   package export;
					   message local {
					     optional export.local field = 1;
					   }`,
		},
		"success_edition_2023_export_local_as_type_names": {
			contents: `edition = "2023";
					   package local;
					   message export {
					     local.export field = 1;
					   }`,
		},
	}

	for name, tc := range testCases {
		expectedPrefix := "success_"
		if tc.expectedErr != "" {
			expectedPrefix = "failure_"
		}
		assert.Truef(t, strings.HasPrefix(name, expectedPrefix), "expected test name %q to have %q prefix", name, expectedPrefix)

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			errs := reporter.NewHandler(nil)
			if ast, err := Parse("test.proto", strings.NewReader(tc.contents), errs); err == nil {
				_, _ = ResultFromAST(ast, true, errs, false)
			}

			err := errs.Error()
			if tc.expectedErr == "" {
				//nolint:testifylint // we want to continue even if err!=nil
				assert.NoError(t, err, "should succeed")
			} else {
				//nolint:testifylint // we want to continue even if assertion fails
				assert.EqualError(t, err, tc.expectedErr, "bad error message")
			}

			//expectSuccess := tc.expectedErr == ""
			//if tc.expectedDiffWithProtoc {
			//	expectSuccess = !expectSuccess
			//}
			//testByProtoc(t, tc.contents, expectSuccess)
		})
	}
}

// Running protoc is disabled in this fork of protocompile, mainly to avoid copying over all
// the machinery associated with downloading and managing protoc.

//func testByProtoc(t *testing.T, fileContents string, expectSuccess bool) {
//	t.Helper()
//	stdout, err := protoc.Compile(map[string]string{"test.proto": fileContents}, nil)
//	if execErr := new(exec.ExitError); errors.As(err, &execErr) {
//		t.Logf("protoc stdout:\n%s\nprotoc stderr:\n%s\n", stdout, execErr.Stderr)
//		require.False(t, expectSuccess)
//		return
//	}
//	require.NoError(t, err)
//	require.True(t, expectSuccess)
//}
