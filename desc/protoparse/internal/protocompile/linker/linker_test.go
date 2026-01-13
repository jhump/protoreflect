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

package linker_test

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/internal/messageset"
	"github.com/bufbuild/protocompile/internal/protoc"
	"github.com/bufbuild/protocompile/internal/prototest"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/bufbuild/protocompile/reporter"
)

func TestSimpleLink(t *testing.T) {
	t.Parallel()
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testdata"},
		}),
	}
	fds, err := compiler.Compile(t.Context(), "desc_test_complex.proto")
	require.NoError(t, err)

	res, ok := fds[0].(linker.Result)
	require.True(t, ok)
	fdset := prototest.LoadDescriptorSet(t, "../internal/testdata/desc_test_complex.protoset", linker.ResolverFromFile(fds[0]))
	prototest.CheckFiles(t, res, fdset, true)
}

func TestSimpleLink_Editions(t *testing.T) {
	t.Parallel()
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testdata/editions"},
		}),
	}
	fds, err := compiler.Compile(t.Context(), "all_default_features.proto", "features_with_overrides.proto")
	require.NoError(t, err)

	fdset := prototest.LoadDescriptorSet(t, "../internal/testdata/editions/all.protoset", fds.AsResolver())

	prototest.CheckFiles(t, fds[0], fdset, true)
	prototest.CheckFiles(t, fds[1], fdset, true)
}

func TestMultiFileLink(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"desc_test_defaults.proto", "desc_test_field_types.proto", "desc_test_options.proto", "desc_test_wellknowntypes.proto"} {
		compiler := protocompile.Compiler{
			Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
				ImportPaths: []string{"../internal/testdata"},
			}),
		}
		fds, err := compiler.Compile(t.Context(), name)
		require.NoError(t, err)

		res, ok := fds[0].(linker.Result)
		require.True(t, ok)
		fdset := prototest.LoadDescriptorSet(t, "../internal/testdata/all.protoset", linker.ResolverFromFile(fds[0]))
		prototest.CheckFiles(t, res, fdset, true)
	}
}

func TestProto3Optional(t *testing.T) {
	t.Parallel()
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testdata"},
		}),
	}
	fds, err := compiler.Compile(t.Context(), "desc_test_proto3_optional.proto")
	require.NoError(t, err)

	fdset := prototest.LoadDescriptorSet(t, "../internal/testdata/desc_test_proto3_optional.protoset", fds.AsResolver())

	res, ok := fds[0].(linker.Result)
	require.True(t, ok)
	prototest.CheckFiles(t, res, fdset, true)
}

func TestLinkerValidation(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		input map[string]string
		// The correct order of passing files to protoc in command line
		// This is set when the order of input file matters for protoc
		inputOrder []string
		// Expected error message - leave empty if input is expected to succeed
		expectedErr            string
		expectedDiffWithProtoc bool
		expectProtodescFail    bool
	}{
		"success_multi_namespace": {
			input: map[string]string{
				"foo.proto":  `syntax = "proto3"; package namespace.a; import "foo2.proto"; import "foo3.proto"; import "foo4.proto"; message Foo{ b.Bar a = 1; b.Baz b = 2; b.Buzz c = 3; }`,
				"foo2.proto": `syntax = "proto3"; package namespace.b; message Bar{}`,
				"foo3.proto": `syntax = "proto3"; package namespace.b; message Baz{}`,
				"foo4.proto": `syntax = "proto3"; package namespace.b; message Buzz{}`,
			},
		},
		"failure_missing_import": {
			input: map[string]string{
				"foo.proto": `import "foo2.proto"; message fubar{}`,
			},
			expectedErr: `foo.proto:1:8: file not found: foo2.proto`,
		},
		"failure_import_cycle": {
			input: map[string]string{
				"foo.proto":  `import "foo2.proto"; message fubar{}`,
				"foo2.proto": `import "foo.proto"; message baz{}`,
			},
			// since files are compiled concurrently, there are two possible outcomes
			expectedErr: `foo.proto:1:8: cycle found in imports: "foo.proto" -> "foo2.proto" -> "foo.proto"` +
				` || foo2.proto:1:8: cycle found in imports: "foo2.proto" -> "foo.proto" -> "foo2.proto"`,
		},
		"failure_import_lite_from_nonlite": {
			input: map[string]string{
				"foo.proto":  `option optimize_for=LITE_RUNTIME;`,
				"foo2.proto": `import "foo.proto"; message baz{}`,
			},
			expectedErr: `foo2.proto:1:8: a file that does not use optimize_for=LITE_RUNTIME may not import file "foo.proto" that does`,
		},
		"success_import_nonlite_from_lite": {
			input: map[string]string{
				"foo.proto":  `import "foo2.proto"; option optimize_for=LITE_RUNTIME;`,
				"foo2.proto": `message baz{}`,
			},
		},
		"failure_extend_nonlite_from_lite": {
			input: map[string]string{
				"foo.proto":  `import "foo2.proto"; option optimize_for=LITE_RUNTIME; extend Baz { optional string s = 1; }`,
				"foo2.proto": `message Baz { extensions 1 to 100; }`,
			},
			expectedErr: `foo.proto:1:69: extensions in a file that uses optimize_for=LITE_RUNTIME may not extend messages in file "foo2.proto" which does not`,
		},
		"failure_enum_cpp_scope": {
			input: map[string]string{
				"foo.proto": "enum foo { bar = 1; baz = 2; } enum fu { bar = 1; }",
			},
			expectedErr: `foo.proto:1:42: symbol "bar" already defined at foo.proto:1:12; protobuf uses C++ scoping rules for enum values, so they exist in the scope enclosing the enum`,
		},
		"failure_redefined_symbol": {
			input: map[string]string{
				"foo.proto": "message foo {} enum foo { V = 0; }",
			},
			expectedErr: `foo.proto:1:21: symbol "foo" already defined at foo.proto:1:9`,
		},
		"failure_duplicate_field_name": {
			input: map[string]string{
				"foo.proto": "message foo { optional string a = 1; optional string a = 2; }",
			},
			expectedErr: `foo.proto:1:54: symbol "foo.a" already defined at foo.proto:1:31`,
		},
		"failure_duplicate_symbols": {
			input: map[string]string{
				"foo.proto":  "message foo {}",
				"foo2.proto": "enum foo { V = 0; }",
			},
			// since files are compiled concurrently, there are two possible outcomes
			expectedErr: `foo.proto:1:9: symbol "foo" already defined at foo2.proto:1:6` +
				` || foo2.proto:1:6: symbol "foo" already defined at foo.proto:1:9`,
		},
		"failure_unsupported_type": {
			input: map[string]string{
				"foo.proto": "message foo { optional blah a = 1; }",
			},
			expectedErr: "foo.proto:1:24: field foo.a: unknown type blah",
		},
		"failure_invalid_method_field": {
			input: map[string]string{
				"foo.proto": "message foo { optional bar.baz a = 1; } service bar { rpc baz (foo) returns (foo); }",
			},
			expectedErr: "foo.proto:1:24: field foo.a: invalid type: bar.baz is a method, not a message or enum",
		},
		"failure_duplicate_extension": {
			input: map[string]string{
				"foo.proto": "message foo { extensions 1 to 2; } extend foo { optional string a = 1; } extend foo { optional int32 b = 1; }",
			},
			expectedErr: "foo.proto:1:106: extension with tag 1 for message foo already defined at foo.proto:1:69",
		},
		"failure_unknown_extendee": {
			input: map[string]string{
				"foo.proto": "package fu.baz; extend foobar { optional string a = 1; }",
			},
			expectedErr: "foo.proto:1:24: unknown extendee type foobar",
		},
		"failure_extend_service": {
			input: map[string]string{
				"foo.proto": "package fu.baz; service foobar{} extend foobar { optional string a = 1; }",
			},
			expectedErr: "foo.proto:1:41: extendee is invalid: fu.baz.foobar is a service, not a message",
		},
		"failure_conflict_method_message_input": {
			input: map[string]string{
				"foo.proto": "message foo{} message bar{} service foobar{ rpc foo(foo) returns (bar); }",
			},
			expectedErr: "foo.proto:1:53: method foobar.foo: invalid request type: foobar.foo is a method, not a message",
		},
		"failure_conflict_method_message_output": {
			input: map[string]string{
				"foo.proto": "message foo{} message bar{} service foobar{ rpc foo(bar) returns (foo); }",
			},
			expectedErr: "foo.proto:1:67: method foobar.foo: invalid response type: foobar.foo is a method, not a message",
		},
		"failure_invalid_extension_field": {
			input: map[string]string{
				"foo.proto": "package fu.baz; message foobar{ extensions 1; } extend foobar { optional string a = 2; }",
			},
			expectedErr: "foo.proto:1:85: extension fu.baz.a: tag 2 is not in valid range for extended type fu.baz.foobar",
		},
		"failure_unknown_type": {
			input: map[string]string{
				"foo.proto":  `package fu.baz; import public "foo2.proto"; message foobar{ optional baz a = 1; }`,
				"foo2.proto": `package fu.baz; import "foo3.proto"; message fizzle{ }`,
				"foo3.proto": "package fu.baz; message baz{ }",
			},
			expectedErr: "foo.proto:1:70: field fu.baz.foobar.a: unknown type baz; resolved to fu.baz which is not defined; consider using a leading dot",
		},
		"success_extension_types": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					package foo;
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.FileOptions           { optional string fil_foo = 12000; }
					extend google.protobuf.MessageOptions        { optional string msg_foo = 12000; }
					extend google.protobuf.FieldOptions          { optional string fld_foo = 12000 [(fld_foo) = "extension"]; }
					extend google.protobuf.OneofOptions          { optional string oof_foo = 12000; }
					extend google.protobuf.EnumOptions           { optional string enm_foo = 12000; }
					extend google.protobuf.EnumValueOptions      { optional string env_foo = 12000; }
					extend google.protobuf.ExtensionRangeOptions { optional string ext_foo = 12000; }
					extend google.protobuf.ServiceOptions        { optional string svc_foo = 12000; }
					extend google.protobuf.MethodOptions         { optional string mtd_foo = 12000; }
					option (fil_foo) = "file";
					message Foo {
						option (msg_foo) = "message";
						oneof foo {
							option (oof_foo) = "oneof";
							string bar = 1 [(fld_foo) = "field"];
						}
						extensions 100 to 200 [(ext_foo) = "extensionrange"];
					}
					enum Baz {
						option (enm_foo) = "enum";
						ZERO = 0 [(env_foo) = "enumvalue"];
					}
					service FooService {
						option (svc_foo) = "service";
						rpc Bar(Foo) returns (Foo) {
							option (mtd_foo) = "method";
						}
					}`,
			},
		},
		"failure_default_repeated": {
			input: map[string]string{
				"foo.proto": `package fu.baz; message foobar{ repeated string a = 1 [default = "abc"]; }`,
			},
			expectedErr: "foo.proto:1:56: field fu.baz.foobar.a: default value cannot be set because field is repeated",
		},
		"failure_default_message": {
			input: map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional foobar a = 1 [default = { a: {} }]; }",
			},
			expectedErr: "foo.proto:1:56: field fu.baz.foobar.a: default value cannot be set because field is a message",
		},
		"failure_default_string_message": {
			input: map[string]string{
				"foo.proto": `package fu.baz; message foobar{ optional string a = 1 [default = { a: "abc" }]; }`,
			},
			expectedErr: "foo.proto:1:66: field fu.baz.foobar.a: option default: default value cannot be a message",
		},
		"failure_string_default_double": {
			input: map[string]string{
				"foo.proto": "package fu.baz; message foobar{ optional string a = 1 [default = 1.234]; }",
			},
			expectedErr: "foo.proto:1:66: field fu.baz.foobar.a: option default: expecting string, got double",
		},
		"failure_editions_default_with_implicit_presence": {
			input: map[string]string{
				"foo.proto": `edition = "2023"; message Foo { string s = 1 [default="abc", features.field_presence=IMPLICIT]; }`,
			},
			expectedErr: "foo.proto:1:47: default value is not allowed on fields with implicit presence",
		},
		"failure_enum_default_not_found": {
			input: map[string]string{
				"foo.proto": "package fu.baz; enum abc { OK=0; NOK=1; } message foobar{ optional abc a = 1 [default = NACK]; }",
			},
			expectedErr: "foo.proto:1:89: field fu.baz.foobar.a: option default: enum fu.baz.abc has no value named NACK",
		},
		"failure_unknown_file_option": {
			input: map[string]string{
				"foo.proto": "option b = 123;",
			},
			expectedErr: "foo.proto:1:8: option b: field b of google.protobuf.FileOptions does not exist",
		},
		"failure_unknown_extension": {
			input: map[string]string{
				"foo.proto": "option (foo.bar) = 123;",
			},
			expectedErr: "foo.proto:1:8: unknown extension foo.bar",
		},
		"failure_invalid_option": {
			input: map[string]string{
				"foo.proto": "option uninterpreted_option = { };",
			},
			expectedErr: "foo.proto:1:8: invalid option 'uninterpreted_option'",
		},
		"failure_option_unknown_field": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f).b = 123;`,
			},
			expectedErr: "foo.proto:5:12: option (f).b: field b of foo does not exist",
		},
		"failure_option_wrong_type": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f).a = 123;`,
			},
			expectedErr: "foo.proto:5:16: option (f).a: expecting string, got integer",
		},
		"failure_extension_message_not_file": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (b) = 123;`,
			},
			expectedErr: "foo.proto:5:8: option (b): extension b should extend google.protobuf.FileOptions but instead extends foo",
		},
		"failure_option_message_not_extension": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (foo) = 123;`,
			},
			expectedErr: "foo.proto:5:8: invalid extension: foo is a message, not an extension",
		},
		"failure_option_field_not_extension": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (foo.a) = 123;`,
			},
			expectedErr: "foo.proto:5:8: invalid extension: foo.a is a field but not an extension",
		},
		"failure_option_not_repeated": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f) = { a: [ 123 ] };`,
			},
			expectedErr: "foo.proto:5:19: option (f): value is an array but field is not repeated",
		},
		"failure_option_repeated_string_integer": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { repeated string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f) = { a: [ "a", "b", 123 ] };`,
			},
			expectedErr: "foo.proto:5:31: option (f): expecting string, got integer",
		},
		"failure_option_non_repeated_override": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f) = { a: "a" };
					option (f) = { a: "b" };`,
			},
			expectedErr: "foo.proto:6:8: option (f): non-repeated option field (f) already set",
		},
		"failure_option_non_repeated_override2": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f) = { a: "a" };
					option (f).a = "b";`,
			},
			expectedErr: "foo.proto:6:12: option (f).a: non-repeated option field a already set",
		},
		"failure_option_int32_not_string": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { optional string a = 1; extensions 10 to 20; }
					extend foo { optional int32 b = 10; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f) = { a: "a" };
					option (f).(b) = "b";`,
			},
			expectedErr: "foo.proto:6:18: option (f).(b): expecting int32, got string",
		},
		"failure_option_required_field_unset": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { required string a = 1; required string b = 2; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f) = { a: "a" };`,
			},
			expectedErr: "foo.proto:4:1: error in file options: some required fields missing: (f).b",
		},
		"failure_option_required_field_unset2": {
			input: map[string]string{
				"foo.proto": `
					import "google/protobuf/descriptor.proto";
					message foo { required string a = 1; required string b = 2; }
					extend google.protobuf.FileOptions { optional foo f = 20000; }
					option (f).a = "a";`,
			},
			expectedErr: "foo.proto:4:1: error in file options: some required fields missing: (f).b",
		},
		"success_extensions_do_not_inherit_file_field_presence": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					option features.field_presence = IMPLICIT;
					message Foo {
					  extensions 1 to 100;
					}
					enum Enum {
					  option features.enum_type = CLOSED;
					  ZERO = 0;
					  ONE = 1;
					}
					extend Foo {
					  string s = 1 [default="abc"];
					  Enum en = 2;
					  repeated Enum ens = 3;
					}`,
			},
		},
		"failure_message_set_wire_format_scalar": {
			input: map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to 100; } extend Foo { optional int32 bar = 1; }",
			},
			expectedErr: "foo.proto:1:99: messages with message-set wire format cannot contain scalar extensions, only messages",
		},
		"success_message_set_wire_format": {
			input: map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to 100; } extend Foo { optional Foo bar = 1; }",
			},
			expectProtodescFail: !messageset.CanSupportMessageSets(),
		},
		"failure_tag_out_of_range": {
			input: map[string]string{
				"foo.proto": "message Foo { extensions 1 to max; } extend Foo { optional int32 bar = 536870912; }",
			},
			expectedErr: "foo.proto:1:72: extension bar: tag 536870912 is not in valid range for extended type Foo",
		},
		"success_tag_message_set_wire_format": {
			input: map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to max; } extend Foo { optional Foo bar = 536870912; }",
			},
			expectProtodescFail: !messageset.CanSupportMessageSets(),
		},
		"failure_message_set_wire_format_repeated": {
			input: map[string]string{
				"foo.proto": "message Foo { option message_set_wire_format = true; extensions 1 to 100; } extend Foo { repeated Foo bar = 1; }",
			},
			expectedErr: "foo.proto:1:90: messages with message-set wire format cannot contain repeated extensions, only optional",
		},
		"failure_resolve_first_part_of_name": {
			input: map[string]string{
				"foo.proto": `syntax = "proto3"; package com.google; import "google/protobuf/wrappers.proto"; message Foo { google.protobuf.StringValue str = 1; }`,
			},
			expectedErr: "foo.proto:1:95: field com.google.Foo.str: unknown type google.protobuf.StringValue; resolved to com.google.protobuf.StringValue which is not defined; consider using a leading dot",
		},
		"success_group_in_custom_option": {
			// Groups must be referred by lower-case field name in option name.
			// Hence "bar" instead of "Bar" in the option at the end.
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  optional group Bar = 1 { optional string name = 1; }
					}
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz { option (foo).bar.name = "abc"; }`,
			},
		},
		"failure_group_in_custom_option_referred_by_type_name": {
			// Trying to refer to group by the group name (which is a message name)
			// instead of lower-case field name fails.
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  optional group Bar = 1 { optional string name = 1; }
					}
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz { option (foo).Bar.name = "abc"; }`,
			},
			expectedErr: "foo.proto:7:28: message Baz: option (foo).Bar.name: field Bar of Foo does not exist",
		},
		"success_group_custom_option": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions {
					  optional group Foo = 10001 { optional string name = 1; }
					}
					message Bar { option (foo).name = "abc"; }`,
			},
		},
		"failure_group_custom_option_referred_by_type_name": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions {
					  optional group Foo = 10001 { optional string name = 1; }
					}
					message Bar { option (Foo).name = "abc"; }`,
			},
			expectedErr: "foo.proto:6:22: message Bar: invalid extension: Foo is a message, not an extension",
		},
		"success_group_in_custom_option_msg_literal_referred_by_field_name": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  optional group Bar = 1 { optional string name = 1; }
					}
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz { option (foo) = { bar< name: "abc" > }; }`,
			},
		},
		"success_group_in_custom_option_msg_literal": {
			// However, groups MAY be referred by group name (i.e. message name)
			// inside of a message literal. So, in this example, "Bar" is allowed
			// (instead of lower-case "bar") because it's inside a message literal.
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  optional group Bar = 1 { optional string name = 1; }
					}
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz { option (foo) = { Bar< name: "abc" > }; }`,
			},
		},
		"success_group_extension_in_custom_option_msg_literal": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message Foo { extensions 1 to 10; }
					extend Foo { optional group Bar = 10 { optional string name = 1; } }
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz { option (foo) = { [bar]< name: "abc" > }; }`,
			},
		},
		"failure_group_extension_in_custom_option_msg_literal_referred_by_type_name": {
			// BUT, groups may NOT be referred to by group name (i.e. message name) if
			// they are extensions. Only the lower-case field name works in this
			// context.
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message Foo { extensions 1 to 10; }
					extend Foo { optional group Bar = 10 { optional string name = 1; } }
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz { option (foo) = { [Bar]< name: "abc" > }; }`,
			},
			expectedErr: "foo.proto:6:33: message Baz: option (foo): invalid extension: Bar is a message, not an extension",
		},
		"success_looks_like_group_in_custom_option_msg_literal": {
			// Fields that "look like groups" may also be referred to by their
			// group/message name. This is for backwards-compatibility for proto2
			// groups that are migrated to editions. A field "looks like a group" if:
			// 1. It uses delimited encoding.
			// 2. The field name == lower-case(message name)
			// 3. The message is declared in the same scope as the field that
			//    references it (i.e. field and message type are siblings)
			input: map[string]string{
				"foo.proto": `
					edition = "2023";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  Bar bar = 1 [features.message_encoding=DELIMITED];
					  message Bar { string name = 1; }
					}
					extend google.protobuf.MessageOptions { Foo foo = 10001; }
					message Baz { option (foo) = { Bar< name: "abc" > }; }`,
			},
		},
		"failure_not_looks_like_group_in_custom_option_msg_literal_wrong_scope": {
			// ONLY fields that "look like groups" can use the message name. Other
			// fields of message type MUST use the field name.
			input: map[string]string{
				"foo.proto": `
					edition = "2023";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  Bar bar = 1 [features.message_encoding=DELIMITED];
					}
					message Bar { string name = 1; }
					extend google.protobuf.MessageOptions { Foo foo = 10001; }
					message Baz { option (foo) = { Bar< name: "abc" > }; }`,
			},
			expectedErr: `foo.proto:8:32: message Baz: option (foo): field Bar not found`,
		},
		"failure_not_looks_like_group_in_custom_option_msg_literal_wrong_field_name": {
			input: map[string]string{
				"foo.proto": `
					edition = "2023";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  Bar barbar = 1 [features.message_encoding=DELIMITED];
					  message Bar { string name = 1; }
					}
					extend google.protobuf.MessageOptions { Foo foo = 10001; }
					message Baz { option (foo) = { Bar< name: "abc" > }; }`,
			},
			expectedErr: `foo.proto:8:32: message Baz: option (foo): field Bar not found`,
		},
		"failure_not_looks_like_group_in_custom_option_msg_literal_not_delimited": {
			input: map[string]string{
				"foo.proto": `
					edition = "2023";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  Bar bar = 1;
					  message Bar { string name = 1; }
					}
					extend google.protobuf.MessageOptions { Foo foo = 10001; }
					message Baz { option (foo) = { Bar< name: "abc" > }; }`,
			},
			expectedErr: `foo.proto:8:32: message Baz: option (foo): field Bar not found`,
		},
		"failure_oneof_extension_already_set_msg_literal": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { oneof bar { string baz = 1; string buzz = 2; } }
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz { option (foo) = { baz: "abc" buzz: "xyz" }; }`,
			},
			expectedErr: `foo.proto:5:43: message Baz: option (foo): oneof "bar" already has field "baz" set`,
		},
		"failure_oneof_extension_already_set": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { oneof bar { string baz = 1; string buzz = 2; } }
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz {
					  option (foo).baz = "abc";
					  option (foo).buzz = "xyz";
					}`,
			},
			expectedErr:            `foo.proto:7:16: message Baz: option (foo).buzz: oneof "bar" already has field "baz" set`,
			expectedDiffWithProtoc: true,
			// TODO: This is a bug of protoc (https://github.com/protocolbuffers/protobuf/issues/9125).
			//  Difference is expected in the test before it is fixed.
		},
		"failure_oneof_extension_already_set_implied_by_destructured_option": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { oneof bar { google.protobuf.DescriptorProto baz = 1; google.protobuf.DescriptorProto buzz = 2; } }
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz {
					  option (foo).baz.name = "abc";
					  option (foo).buzz.name = "xyz";
					}`,
			},
			expectedErr:            `foo.proto:7:16: message Baz: option (foo).buzz.name: oneof "bar" already has field "baz" set`,
			expectedDiffWithProtoc: true,
			// TODO: This is a bug of protoc (https://github.com/protocolbuffers/protobuf/issues/9125).
			//  Difference is expected in the test before it is fixed.
		},
		"failure_oneof_extension_already_set_implied_by_deeply_nested_destructured_option": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { oneof bar { google.protobuf.DescriptorProto baz = 1; google.protobuf.DescriptorProto buzz = 2; } }
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz {
					  option (foo).baz.options.(foo).baz.name = "abc";
					  option (foo).baz.options.(foo).buzz.name = "xyz";
					}`,
			},
			expectedErr:            `foo.proto:7:34: message Baz: option (foo).baz.options.(foo).buzz.name: oneof "bar" already has field "baz" set`,
			expectedDiffWithProtoc: true,
			// TODO: This is a bug of protoc (https://github.com/protocolbuffers/protobuf/issues/9125).
			//  Difference is expected in the test before it is fixed.
		},
		"success_empty_array_literal_no_leading_colon_if_msg": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { repeated string strs = 1; repeated Foo foos = 2; }
					extend google.protobuf.FileOptions { optional Foo foo = 10001; }
					option (foo) = {
					  strs: []
					  foos []
					};`,
			},
		},
		"failure_empty_array_literal_require_leading_colon_if_scalar": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { repeated string strs = 1; repeated Foo foos = 2; }
					extend google.protobuf.FileOptions { optional Foo foo = 10001; }
					option (foo) = {
					  strs []
					  foos []
					};`,
			},
			expectedErr: `foo.proto:6:8: syntax error: unexpected value, expecting ':'`,
		},
		"success_array_literal": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { repeated string strs = 1; repeated Foo foos = 2; }
					extend google.protobuf.FileOptions { optional Foo foo = 10001; }
					option (foo) = {
					  strs: ['abc', 'def']
					  foos [<strs:'foo'>, <strs:'bar'>]
					};`,
			},
		},
		"failure_array_literal_require_leading_colon_if_scalar": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { repeated string strs = 1; repeated Foo foos = 2; }
					extend google.protobuf.FileOptions { optional Foo foo = 10001; }
					option (foo) = {
					  strs ['abc', 'def']
					  foos [<strs:'foo'>, <strs:'bar'>]
					};`,
			},
			expectedErr: `foo.proto:6:9: syntax error: unexpected string literal, expecting '{' or '<' or ']'`,
		},
		"failure_scoping_resolves_to_sibling_not_parent": {
			input: map[string]string{
				"foo.proto": `
					package foo.bar;
					message M {
					  enum E { M = 0; }
					  optional M F1 = 1;
					  extensions 2 to 2;
					  extend M { optional string F2 = 2; }
					}`,
			},
			expectedErr: `foo.proto:6:10: extendee is invalid: foo.bar.M.M is an enum value, not a message`,
		},
		"failure_json_name_on_extension": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions {
					  string foobar = 10001 [json_name="FooBar"];
					}`,
			},
			expectedErr: "foo.proto:4:26: field foobar: option json_name is not allowed on extensions",
		},
		"success_json_name_on_extension_ok_if_default": {
			// Unclear if this should really be valid... But it's what protoc does.
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions {
					  string foobar = 10001 [json_name="foobar"];
					}`,
			},
		},
		"failure_json_name_looks_like_extension": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string foobar = 10001 [json_name="[FooBar]"];
					}`,
			},
			expectedErr: "foo.proto:3:36: field Foo.foobar: option json_name value cannot start with '[' and end with ']'; that is reserved for representing extensions",
		},
		"success_json_name_not_quite_extension_okay": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string foobar = 10001 [json_name="[FooBar"];
					}`,
			},
		},
		"failure_synthetic_map_entry_reference": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  map<string,string> bar = 1;
					}
					message Baz {
					  Foo.BarEntry e = 1;
					}`,
			},
			expectedErr: "foo.proto:6:3: field Baz.e: Foo.BarEntry is a synthetic map entry and may not be referenced explicitly",
		},
		"failure_imported_synthetic_map_entry_reference": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/struct.proto";
					message Foo {
					  google.protobuf.Struct.FieldsEntry e = 1;
					}`,
			},
			expectedErr: "foo.proto:4:3: field Foo.e: google.protobuf.Struct.FieldsEntry is a synthetic map entry and may not be referenced explicitly",
		},
		"failure_proto3_can_only_extend_options": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					message Foo {
					  extensions 1 to 100;
					}`,
				"bar.proto": `
					syntax = "proto3";
					import "foo.proto";
					extend Foo {
					  string bar = 1;
					}`,
			},
			expectedErr: "bar.proto:3:8: extend blocks in proto3 can only be used to define custom options",
		},
		"failure_oneof_disallows_empty_statement": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  oneof bar {
					    string baz = 1;
					    uint64 buzz = 2;
					    ;
					  }
					}`,
			},
			expectedErr: "foo.proto:6:5: syntax error: unexpected ';'",
		},
		"failure_extend_disallows_empty_statement": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions {
					  string baz = 1001;
					  uint64 buzz = 1002;
					  ;
					}`,
			},
			expectedErr: "foo.proto:6:3: syntax error: unexpected ';'",
		},
		"failure_oneof_conflicts_with_contained_field": {
			input: map[string]string{
				"a.proto": `
					syntax = "proto3";
					message m{
					  oneof z{
						int64 z=1;
					  }
					}`,
			},
			expectedErr: `a.proto:4:15: symbol "m.z" already defined at a.proto:3:9`,
		},
		"failure_oneof_conflicts_with_adjacent_field": {
			input: map[string]string{
				"a.proto": `
					syntax="proto3";
					message m{
					  string z = 1;
					  oneof z{int64 b=2;}
					}`,
			},
			expectedErr: `a.proto:4:9: symbol "m.z" already defined at a.proto:3:10`,
		},
		"failure_oneof_conflicts_with_other_oneof": {
			input: map[string]string{
				"a.proto": `
					syntax="proto3";
					message m{
					  oneof z{int64 a=1;}
					  oneof z{int64 b=2;}
					}`,
			},
			expectedErr: `a.proto:4:9: symbol "m.z" already defined at a.proto:3:9`,
		},
		"success_custom_option_enums_look_like_msg_literal_keywords": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					enum Foo { option allow_alias = true; true = 0; false = 1; t = 2; f = 3; True = 0; False = 1; inf = 6; nan = 7; }
					extend google.protobuf.MessageOptions { repeated Foo foo = 10001; }
					message Baz {
					  option (foo) = true; option (foo) = false;
					  option (foo) = t; option (foo) = f;
					  option (foo) = True; option (foo) = False;
					  option (foo) = inf; option (foo) = nan;
					}`,
			},
		},
		"failure_option_boolean_names": {
			// in options, boolean values must be "true" or "false"
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions { repeated bool foo = 10001; }
					message Baz {
					  option (foo) = true; option (foo) = false;
					  option (foo) = t; option (foo) = f;
					  option (foo) = True; option (foo) = False;
					}`,
			},
			expectedErr: `foo.proto:6:18: message Baz: option (foo): expecting bool, got identifier` +
				` && foo.proto:6:36: message Baz: option (foo): expecting bool, got identifier` +
				` && foo.proto:7:18: message Baz: option (foo): expecting bool, got identifier` +
				` && foo.proto:7:39: message Baz: option (foo): expecting bool, got identifier`,
		},
		"success_message_literals_boolean_names": {
			// but inside message literals, boolean values can be
			// "true", "false", "t", "f", "True", or "False"
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo { repeated bool b = 1; }
					extend google.protobuf.MessageOptions { Foo foo = 10001; }
					message Baz {
					  option (foo) = {
						b: t     b: f
						b: true  b: false
						b: True  b: False
					  };
					}`,
			},
		},
		"failure_message_literal_leading_dot": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message Foo { extensions 1 to 10; }
					extend Foo { optional bool b = 10; }
					extend google.protobuf.MessageOptions { optional Foo foo = 10001; }
					message Baz {
					  option (foo) = {
					    [.b]: true
					  };
					}`,
			},
			expectedErr: "foo.proto:8:6: syntax error: unexpected '.'",
		},
		"success_extension_resolution_custom_options": {
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					message a { extensions 1 to 100; }
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message b {
					  message c {
						extend a { repeated int32 i = 1; repeated float f = 2; }
					  }
					  option (msga) = {
						[foo.bar.b.c.i]: 123
						[bar.b.c.i]: 234
						[b.c.i]: 345
					  };
					  option (msga).(foo.bar.b.c.f) = 1.23;
					  option (msga).(bar.b.c.f) = 2.34;
					  option (msga).(b.c.f) = 3.45;
					}`,
			},
		},
		"failure_extension_resolution_custom_options": {
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					message a { extensions 1 to 100; }
					message b { extensions 1 to 100; }
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message c {
					  extend a { optional b b = 1; }
					  extend b { repeated int32 i = 1; repeated float f = 2; }
					  option (msga) = {
						[foo.bar.c.b] {
						  [foo.bar.c.i]: 123
						  [bar.c.i]: 234
						  [c.i]: 345
						}
					  };
					  option (msga).(foo.bar.c.b).(foo.bar.c.f) = 1.23;
					  option (msga).(foo.bar.c.b).(bar.c.f) = 2.34;
					  option (msga).(foo.bar.c.b).(c.f) = 3.45;
					}`,
			},
			expectedErr: "test.proto:9:10: extendee is invalid: foo.bar.c.b is an extension, not a message",
		},
		"failure_msg_literal_scoping_rules_limited": {
			// This is due to an unfortunate way of how message literals are actually implemented
			// in protoc. It just uses the text format, so parsing the text format has different
			// (and much more limited) resolution/scoping rules for relative references than other
			// references in protobuf language.
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					message a { extensions 1 to 100; }
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message b {
					  message c {
						extend a { repeated int32 i = 1; repeated float f = 2; }
					  }
					  option (msga) = {
					    [c.i]: 456
					  };
					}`,
			},
			expectedErr: "test.proto:11:6: message foo.bar.b: option (foo.bar.msga): unknown extension c.i",
		},
		"failure_msg_literal_scoping_rules_limited2": {
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					message a { extensions 1 to 100; }
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message b {
					  message c {
					    extend a { repeated int32 i = 1; repeated float f = 2; }
					  }
					  option (msga) = {
					    [i]: 567
					  };
					}`,
			},
			expectedErr: "test.proto:11:6: message foo.bar.b: option (foo.bar.msga): unknown extension i",
		},
		"failure_option_scoping_rules_limited": {
			// This is an unfortunate side effect of having no language spec and so accidental
			// quirks in the implementation end up as part of the language :(
			// In this case, names in the option can't resolve to siblings, but must resolve
			// to a scope at least one level higher.
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					message a { extensions 1 to 100; }
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message b {
					  message c {
					    extend a { repeated int32 i = 1; repeated float f = 2; }
					  }
					  option (msga).(c.f) = 4.56;
					}`,
			},
			expectedErr: "test.proto:10:17: message foo.bar.b: unknown extension c.f",
		},
		"failure_option_scoping_rules_limited2": {
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					message a { extensions 1 to 100; }
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message b {
					  message c {
					    extend a { repeated int32 i = 1; repeated float f = 2; }
					  }
					  option (msga).(f) = 5.67;
					}`,
			},
			expectedErr: "test.proto:10:17: message foo.bar.b: unknown extension f",
		},
		"success_option_and_msg_literal_scoping_rules": {
			// This demonstrates all the ways one can successfully refer to extensions
			// in option names and in message literals.
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message a {
					  extensions 1 to 100;
					  message b {
					    message c {
					      extend a { repeated int32 i = 1; repeated float f = 2; }
					    }
					    option (msga) = {
					      [foo.bar.a.b.c.i]: 123
					      [bar.a.b.c.i]: 234
					      [a.b.c.i]: 345
					      // can't use b.c.i here
					    };
					    option (msga).(foo.bar.a.b.c.f) = 1.23;
					    option (msga).(bar.a.b.c.f) = 2.34;
					    option (msga).(a.b.c.f) = 3.45;
					    option (msga).(b.c.f) = 4.56;
					  }
					}`,
			},
		},
		"failure_msg_literal_scoping_rules_limited3": {
			input: map[string]string{
				"test.proto": `
					syntax="proto2";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MessageOptions { optional a msga = 10000; }
					message a {
					  extensions 1 to 100;
					  message b {
					    message c {
					      extend a { repeated int32 i = 1; }
					    }
					    option (msga) = {
					      [b.c.i]: 345
					    };
					  }
					}`,
			},
			expectedErr: "test.proto:12:8: message foo.bar.a.b: option (foo.bar.msga): unknown extension b.c.i",
		},
		"success_lazy": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					message Baz {
						Baz baz = 1 [lazy=true];
						repeated Baz bazzes = 2 [lazy=true];
						Baz buzz = 3 [unverified_lazy=true];
						repeated Baz buzzes = 4 [unverified_lazy=true];
						map<string,string> m1 = 5 [lazy=true];
						map<string,string> m2 = 6 [unverified_lazy=true];
					}`,
			},
		},
		"failure_lazy_not_message": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					message Baz {
						string str = 1 [lazy=true];
					}`,
			},
			expectedErr: "foo.proto:4:25: lazy option can only be used with message fields",
		},
		"failure_unverified_lazy_not_message": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					message Baz {
						string str = 1 [unverified_lazy=true];
					}`,
			},
			expectedErr: "foo.proto:4:25: unverified_lazy option can only be used with message fields",
		},
		"failure_lazy_group": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					package foo.bar;
					message Baz {
						optional group Buzz = 1 [lazy=true] {
							optional string s = 1; 
						}
					}`,
			},
			expectedErr: "foo.proto:4:34: lazy option can only be used with message fields, not groups",
		},
		"failure_lazy_editions_delimited": {
			input: map[string]string{
				"foo.proto": `
					edition = "2023";
					package foo.bar;
					message Baz {
						Baz baz = 1 [lazy=true, features.message_encoding=DELIMITED];
					}`,
			},
			expectedErr: "foo.proto:4:22: lazy option can only be used with message fields that use length-prefixed encoding",
		},
		"success_any_message_literal": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					import "google/protobuf/any.proto";
					import "google/protobuf/descriptor.proto";
					message Foo { string a = 1; int32 b = 2; }
					extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }
					message Baz {
					  option (any) = {
					    [type.googleapis.com/foo.bar.Foo] <
					      a: "abc"
					      b: 123
					    >
					  };
					}`,
			},
		},
		"failure_any_message_literal_not_any": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					import "google/protobuf/descriptor.proto";
					message Foo { string a = 1; int32 b = 2; }
					extend google.protobuf.MessageOptions { optional Foo f = 10001; }
					message Baz {
					  option (f) = {
					    [type.googleapis.com/foo.bar.Foo] <
					      a: "abc"
					      b: 123
					    >
					  };
					}`,
			},
			expectedErr: "foo.proto:8:6: message foo.bar.Baz: option (foo.bar.f): type references are only allowed for google.protobuf.Any, but this type is foo.bar.Foo",
		},
		"failure_any_message_literal_unsupported_domain": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					import "google/protobuf/any.proto";
					import "google/protobuf/descriptor.proto";
					message Foo { string a = 1; int32 b = 2; }
					extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }
					message Baz {
					  option (any) = {
					    [types.custom.io/foo.bar.Foo] <
					      a: "abc"
					      b: 123
					    >
					  };
					}`,
			},
			expectedErr: "foo.proto:9:6: message foo.bar.Baz: option (foo.bar.any): could not resolve type reference types.custom.io/foo.bar.Foo",
		},
		"failure_any_message_literal_scalar": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					import "google/protobuf/any.proto";
					import "google/protobuf/descriptor.proto";
					message Foo { string a = 1; int32 b = 2; }
					extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }
					message Baz {
					  option (any) = {
					    [type.googleapis.com/foo.bar.Foo]: 123
					  };
					}`,
			},
			expectedErr: "foo.proto:9:40: message foo.bar.Baz: option (foo.bar.any): type references for google.protobuf.Any must have message literal value",
		},
		"failure_any_message_literal_incorrect_type": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					import "google/protobuf/any.proto";
					import "google/protobuf/descriptor.proto";
					message Foo { string a = 1; int32 b = 2; }
					extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }
					message Baz {
					  option (any) = {
					    [type.googleapis.com/Foo] <
					      a: "abc"
					      b: 123
					    >
					  };
					}`,
			},
			expectedErr: "foo.proto:9:6: message foo.bar.Baz: option (foo.bar.any): could not resolve type reference type.googleapis.com/Foo",
		},
		"failure_any_message_literal_duplicate": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					import "google/protobuf/any.proto";
					import "google/protobuf/descriptor.proto";
					message Foo { string a = 1; int32 b = 2; }
					extend google.protobuf.MessageOptions { optional google.protobuf.Any any = 10001; }
					message Baz {
					  option (any) = {
					    [type.googleapis.com/foo.bar.Foo] <
					      a: "abc"
					      b: 123
					    >
					    [type.googleapis.com/foo.bar.Foo] <
					      a: "abc"
					      b: 123
					    >
					  };
					}`,
			},
			expectedErr: `foo.proto:9:6: message foo.bar.Baz: option (foo.bar.any): any type references cannot be repeated or mixed with other fields` +
				` && foo.proto:13:6: message foo.bar.Baz: option (foo.bar.any): any type references cannot be repeated or mixed with other fields`,
		},
		"failure_scope_type_name": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.foo;
					import "other.proto";
					service Foo { rpc Bar (Baz) returns (Baz); }
					message Baz {
					  foo.Foo.Bar f = 1;
					}`,
				"other.proto": `
					syntax = "proto3";
					package foo;
					message Foo {
					  enum Bar { ZED = 0; }
					}`,
			},
			expectedErr: "foo.proto:6:3: field foo.foo.Baz.f: invalid type: foo.foo.Foo.Bar is a method, not a message or enum",
		},
		"failure_scope_extension": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message Foo {
					  enum Bar { ZED = 0; }
					  message Foo {
					    extend google.protobuf.MessageOptions {
					      string Bar = 30000;
					    }
					    Foo.Bar f = 1;
					  }
					}`,
			},
			expectedErr: "foo.proto:9:5: field Foo.Foo.f: invalid type: Foo.Foo.Bar is an extension, not a message or enum",
		},
		"success_scope_extension": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.ServiceOptions {
					  string Bar = 30000;
					}
					message Empty {}
					service Foo {
					  option (Bar) = "blah";
					  rpc Bar (Empty) returns (Empty);
					}`,
			},
		},
		"failure_scope_extension2": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.MethodOptions {
					  string Bar = 30000;
					}
					message Empty {}
					service Foo {
					  rpc Bar (Empty) returns (Empty) { option (Bar) = "blah"; }
					}`,
			},
			expectedErr: "foo.proto:8:44: method Foo.Bar: invalid extension: Bar is a method, not an extension",
		},
		"success_scope_extension2": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					enum Bar { ZED = 0; }
					message Foo {
					  extend google.protobuf.MessageOptions {
					    string Bar = 30000;
					  }
					  message Foo {
					    Bar f = 1;
					  }
					}`,
			},
		},
		"success_jstype": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string foo = 1 [jstype=JS_NORMAL];
					  bool bar = 2 [jstype=JS_NORMAL];
					  uint32 u32 = 3 [jstype=JS_NORMAL];
					  // only 64-bit integer types can specify value other than normal
					  int64 a = 4 [jstype=JS_STRING];
					  uint64 b = 5 [jstype=JS_NUMBER];
					  sint64 c = 6 [jstype=JS_STRING];
					  fixed64 d = 7 [jstype=JS_NUMBER];
					  sfixed64 e = 8 [jstype=JS_STRING];
					}`,
			},
		},
		"failure_jstype_not_numeric": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string foo = 1 [jstype=JS_STRING];
					}`,
			},
			expectedErr: `foo.proto:3:19: only 64-bit integer fields (int64, uint64, sint64, fixed64, and sfixed64) can specify a jstype other than JS_NORMAL`,
		},
		"failure_jstype_not_64bit": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  uint32 foo = 1 [jstype=JS_NUMBER];
					}`,
			},
			expectedErr: `foo.proto:3:19: only 64-bit integer fields (int64, uint64, sint64, fixed64, and sfixed64) can specify a jstype other than JS_NORMAL`,
		},
		"failure_json_name_conflict_default": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string foo = 1;
					  string bar = 2 [json_name="foo"];
					}`,
			},
			expectedErr: `foo.proto:4:3: field Foo.bar: custom JSON name "foo" conflicts with default JSON name of field foo, defined at foo.proto:3:3`,
		},
		"failure_json_name_conflict_nested": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Blah {
					  message Foo {
					    string foo = 1;
					    string bar = 2 [json_name="foo"];
					  }
					}`,
			},
			expectedErr: `foo.proto:5:5: field Foo.bar: custom JSON name "foo" conflicts with default JSON name of field foo, defined at foo.proto:4:5`,
		},
		"success_json_names_case_sensitive": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string foo = 1 [json_name="foo_bar"];
					  string bar = 2 [json_name="Foo_Bar"];
					}`,
			},
		},
		"failure_json_name_conflict_default_underscore": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string fooBar = 1;
					  string foo_bar = 2;
					}`,
			},
			expectedErr: `foo.proto:4:3: field Foo.foo_bar: default JSON name "fooBar" conflicts with default JSON name of field fooBar, defined at foo.proto:3:3`,
		},
		"failure_json_name_conflict_default_override": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string fooBar = 1;
					  string foo_bar = 2 [json_name="fuber"];
					}`,
			},
			expectedErr: `foo.proto:4:3: field Foo.foo_bar: default JSON name "fooBar" conflicts with default JSON name of field fooBar, defined at foo.proto:3:3`,
		},
		"success_json_name_differs_by_case": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string fooBar = 1;
					  string FOO_BAR = 2;
					}`,
			},
		},
		"failure_json_name_conflict_leading_underscores": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Foo {
					  string _fooBar = 1;
					  string __foo_bar = 2;
					}`,
			},
			expectedErr: `foo.proto:4:3: field Foo.__foo_bar: default JSON name "FooBar" conflicts with default JSON name of field _fooBar, defined at foo.proto:3:3`,
		},
		"failure_json_name_custom_and_default_proto2": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					message Blah {
					  message Foo {
					    optional string foo = 1 [json_name="fooBar"];
					    optional string foo_bar = 2;
					  }
					}`,
			},
			expectedErr:            `foo.proto:5:5: field Foo.foo_bar: default JSON name "fooBar" conflicts with custom JSON name of field foo, defined at foo.proto:4:5`,
			expectedDiffWithProtoc: true,
			// TODO: This is a bug of protoc (https://github.com/protocolbuffers/protobuf/issues/5063).
			//  Difference is expected in the test before it is fixed.
		},
		"success_json_name_default_proto3_only": {
			// should succeed: only check default JSON names in proto3
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					message Foo {
					  optional string fooBar = 1;
					  optional string foo_bar = 2;
					}`,
			},
		},
		"failure_json_name_conflict_proto2": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					message Foo {
					  optional string fooBar = 1 [json_name="fooBar"];
					  optional string foo_bar = 2 [json_name="fooBar"];
					}`,
			},
			expectedErr:            `foo.proto:4:3: field Foo.foo_bar: custom JSON name "fooBar" conflicts with custom JSON name of field fooBar, defined at foo.proto:3:3`,
			expectedDiffWithProtoc: true,
			// TODO: This is a bug of protoc (https://github.com/protocolbuffers/protobuf/issues/5063).
			//  Difference is expected in the test before it is fixed.
		},
		"success_json_name_default_proto2": {
			// should succeed: only check default JSON names in proto3
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					message Foo {
					  optional string fooBar = 1;
					  optional string FOO_BAR = 2;
					}`,
			},
		},
		"success_json_name_default_proto2_underscore": {
			// should succeed: only check default JSON names in proto3
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					message Foo {
					  optional string fooBar = 1;
					  optional string __foo_bar = 2;
					}`,
			},
		},
		"failure_enum_name_conflict": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					enum Foo {
					  true = 0;
					  TRUE = 1;
					}`,
			},
			expectedErr: `foo.proto:4:3: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) "True" conflicts with camel-case name of enum value true, defined at foo.proto:3:3`,
		},
		"failure_nested_enum_name_conflict": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					message Blah {
					  enum Foo {
					    true = 0;
					    TRUE = 1;
					  }
					}`,
			},
			expectedErr: `foo.proto:5:5: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) "True" conflicts with camel-case name of enum value true, defined at foo.proto:4:5`,
		},
		"failure_nested_enum_scope_conflict": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					enum Foo {
					  BAR_BAZ = 0;
					  Foo_Bar_Baz = 1;
					}`,
			},
			expectedErr: `foo.proto:4:3: enum value Foo.Foo_Bar_Baz: camel-case name (with optional enum name prefix removed) "BarBaz" conflicts with camel-case name of enum value BAR_BAZ, defined at foo.proto:3:3`,
		},
		"success_enum_name_conflict_allow_alias": {
			// should succeed: not a conflict if both values have same number
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					enum Foo {
					  option allow_alias = true;
					  BAR_BAZ = 0;
					  FooBarBaz = 0;
					}`,
			},
		},
		"failure_symbol_conflicts_with_package": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					package foo.bar;
					enum baz { ZED = 0; }`,
				"bar.proto": `
					syntax = "proto3";
					package foo.bar.baz;
					message Empty { }`,
			},
			expectedErr: `foo.proto:3:6: symbol "foo.bar.baz" already defined as a package at bar.proto:2:9` +
				` || bar.proto:2:9: symbol "foo.bar.baz" already defined at foo.proto:3:6`,
		},
		"success_enum_in_msg_literal_using_number": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					enum Foo {
					  ZERO = 0;
					  ONE = 1;
					}
					message Bar {
						optional Foo foo = 1;
					}
					extend google.protobuf.FileOptions {
						optional Bar bar = 10101;
					}
					option (bar) = { foo: 1 };`,
			},
		},
		"success_enum_in_msg_literal_using_negative_number": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					enum Foo {
						ZERO = 0;
						ONE = 1;
						NEG_ONE = -1;
					}
					message Bar {
						optional Foo foo = 1;
					}
					extend google.protobuf.FileOptions {
						optional Bar bar = 10101;
					}
					option (bar) = { foo: -1 };`,
			},
		},
		"success_open_enum_in_msg_literal_using_unknown_number": {
			// enums in proto3 are "open", so unknown numbers are acceptable
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					enum Foo {
					  ZERO = 0;
					  ONE = 1;
					}
					message Bar {
						Foo foo = 1;
					}
					extend google.protobuf.FileOptions {
						Bar bar = 10101;
					}
					option (bar) = { foo: 5 };`,
			},
		},
		"failure_enum_option_using_number": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					enum Foo {
					  ZERO = 0;
					  ONE = 1;
					}
					extend google.protobuf.FileOptions {
						optional Foo foo = 10101;
					}
					option (foo) = 1;`,
			},
			expectedErr: `foo.proto:10:16: option (foo): expecting enum name, got integer`,
		},
		"failure_default_value_for_enum_using_number": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					enum Foo {
					  ZERO = 0;
					  ONE = 1;
					}
					message Bar {
						optional Foo foo = 1 [default=1];
					}`,
			},
			expectedErr: `foo.proto:7:39: field Bar.foo: option default: expecting enum name, got integer`,
		},
		"failure_closed_enum_in_msg_literal_using_unknown_number": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					enum Foo {
					  ZERO = 0;
					  ONE = 1;
					}
					message Bar {
						optional Foo foo = 1;
					}
					extend google.protobuf.FileOptions {
						optional Bar bar = 10101;
					}
					option (bar) = { foo: 5 };`,
			},
			expectedErr: `foo.proto:13:23: option (bar): closed enum Foo has no value with number 5`,
		},
		"failure_enum_in_msg_literal_using_out_of_range_number": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					enum Foo {
						ZERO = 0;
						ONE = 1;
						NEG_ONE = -1;
					}
					message Bar {
						Foo foo = 1;
					}
					extend google.protobuf.FileOptions {
						Bar bar = 10101;
					}
					option (bar) = { foo: 2147483648 };`,
			},
			expectedErr: `foo.proto:14:23: option (bar): value 2147483648 is out of range for an enum`,
		},
		"failure_enum_in_msg_literal_using_out_of_range_negative_number": {
			input: map[string]string{
				"foo.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					enum Foo {
						ZERO = 0;
						ONE = 1;
						NEG_ONE = -1;
					}
					message Bar {
						Foo foo = 1;
					}
					extend google.protobuf.FileOptions {
						Bar bar = 10101;
					}
					option (bar) = { foo: -2147483649 };`,
			},
			expectedErr: `foo.proto:14:23: option (bar): value -2147483649 is out of range for an enum`,
		},
		"success_custom_field_option": {
			input: map[string]string{
				"google/protobuf/descriptor.proto": `
					syntax = "proto2";
					package google.protobuf;
					message FieldOptions {
						optional string some_new_option = 11;
					}`,
				"bar.proto": `
					syntax = "proto3";
					package foo.bar.baz;
					message Foo {
						string bar = 1 [some_new_option="abc"];
					}`,
			},
			inputOrder: []string{"google/protobuf/descriptor.proto", "bar.proto"},
		},
		"failure_group_as_type_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { }
					message Foo { optional group Foo = 1; }
				`,
			},
			expectedErr: `test.proto:3:37: syntax error: unexpected ';', expecting '{' or '['`,
		},
		"failure_group_as_type_name_prefix": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { optional group.Bar Foo = 1; }
				`,
			},
			expectedErr: `test.proto:3:29: syntax error: unexpected '.'`,
		},
		"success_group_as_type_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { }
					message Foo { optional .group Foo = 1; }
				`,
			},
		},
		"success_group_as_type_name_prefix": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { optional .group.Bar Foo = 1; }
				`,
			},
		},
		"failure_oneof_group_as_type_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { }
					message Foo { oneof abc { group Foo = 1; } }
				`,
			},
			expectedErr: `test.proto:3:40: syntax error: unexpected ';', expecting '{' or '['`,
		},
		"failure_oneof_group_as_type_name_prefix": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { oneof abc { group.Bar Foo = 1; } }
				`,
			},
			expectedErr: `test.proto:3:32: syntax error: unexpected '.'`,
		},
		"success_oneof_group_as_type_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { }
					message Foo { oneof abc { .group Foo = 1; } }
				`,
			},
		},
		"success_oneof_group_as_type_name_prefix": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { oneof abc { .group.Bar Foo = 1; } }
				`,
			},
		},
		"failure_ext_group_as_type_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { extensions 1 to 100; }
					extend Foo { optional group Fooz = 1; }
				`,
			},
			expectedErr: `test.proto:4:37: syntax error: unexpected ';', expecting '{' or '['`,
		},
		"failure_ext_group_as_type_name_prefix": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { extensions 1 to 100; }
					extend Foo { optional group.Bar Fooz = 1; }
				`,
			},
			expectedErr: `test.proto:4:28: syntax error: unexpected '.'`,
		},
		"success_ext_group_as_type_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { extensions 1 to 100; }
					extend Foo { optional .group Fooz = 1; }
				`,
			},
		},
		"success_ext_group_as_type_name_prefix": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message group { message Bar {} }
					message Foo { extensions 1 to 100; }
					extend Foo { optional .group.Bar Fooz = 1; }
				`,
			},
		},
		"failure_group_as_type_name_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { }
					message Foo { optional group Foo = 1; }
				`,
			},
			expectedErr: `test.proto:3:37: syntax error: unexpected ';', expecting '{' or '['`,
		},
		"failure_group_as_type_name_prefix_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { message Bar {} }
					message Foo { optional group.Bar Foo = 1; }
				`,
			},
			expectedErr: `test.proto:3:29: syntax error: unexpected '.'`,
		},
		"success_group_as_type_name_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { }
					message Foo { optional .group Foo = 1; }
				`,
			},
		},
		"success_group_as_type_name_prefix_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { message Bar {} }
					message Foo { optional .group.Bar Foo = 1; }
				`,
			},
		},
		"failure_oneof_group_as_type_name_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { }
					message Foo { oneof abc { group Foo = 1; } }
				`,
			},
			expectedErr: `test.proto:3:40: syntax error: unexpected ';', expecting '{' or '['`,
		},
		"failure_oneof_group_as_type_name_prefix_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { message Bar {} }
					message Foo { oneof abc { group.Bar Foo = 1; } }
				`,
			},
			expectedErr: `test.proto:3:32: syntax error: unexpected '.'`,
		},
		"success_oneof_group_as_type_name_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { }
					message Foo { oneof abc { .group Foo = 1; } }
				`,
			},
		},
		"success_oneof_group_as_type_name_prefix_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { message Bar {} }
					message Foo { oneof abc { .group.Bar Foo = 1; } }
				`,
			},
		},
		"failure_ext_group_as_type_name_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { optional group Foo = 10101; }
				`,
			},
			expectedErr: `test.proto:4:67: syntax error: unexpected ';', expecting '{' or '['`,
		},
		"failure_ext_group_as_type_name_prefix_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { optional group.Bar Foo = 10101; }
				`,
			},
			expectedErr: `test.proto:4:55: syntax error: unexpected '.'`,
		},
		"success_ext_group_as_type_name_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { optional .group Foo = 10101; }
				`,
			},
		},
		"success_ext_group_as_type_name_prefix_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { optional .group.Bar Foo = 10101; }
				`,
			},
		},
		"failure_group_as_type_name_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { }
					message Foo { group Foo = 1; }
				`,
			},
			expectedErr: `test.proto:3:15: syntax error: unexpected "group"`,
		},
		"failure_group_as_type_name_prefix_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { message Bar {} }
					message Foo { group.Bar Foo = 1; }
				`,
			},
			expectedErr: `test.proto:3:15: syntax error: unexpected "group"`,
		},
		"success_group_as_type_name_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { }
					message Foo { .group Foo = 1; }
				`,
			},
		},
		"success_group_as_type_name_prefix_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message group { message Bar {} }
					message Foo { .group.Bar Foo = 1; }
				`,
			},
		},
		"failure_ext_group_as_type_name_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { group Foo = 10101; }
				`,
			},
			expectedErr: `test.proto:4:41: syntax error: unexpected "group"`,
		},
		"failure_ext_group_as_type_name_prefix_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { group.Bar Foo = 10101; }
				`,
			},
			expectedErr: `test.proto:4:41: syntax error: unexpected "group"`,
		},
		"success_ext_group_as_type_name_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { .group Foo = 10101; }
				`,
			},
		},
		"success_ext_group_as_type_name_prefix_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					message group { message Bar {} }
					extend google.protobuf.MessageOptions { .group.Bar Foo = 10101; }
				`,
			},
		},
		"failure_group_proto3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo { optional group Foo = 1 {} }
				`,
			},
			expectedErr: `test.proto:2:24: field Foo.foo: groups are not allowed in proto3 or editions`,
		},
		"failure_group_proto3_no_label": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo { group Foo = 1 {} }
				`,
			},
			expectedErr: `test.proto:2:15: syntax error: unexpected "group"`,
		},
		"failure_stream_looks_like_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message stream { message Foo {} }
					service FooService { rpc Do(stream.Foo) returns (stream.Foo); }
				`,
			},
			expectedErr: `test.proto:3:35: method FooService.Do: unknown request type .Foo` +
				` && test.proto:3:56: method FooService.Do: unknown response type .Foo`,
		},
		"success_stream_looks_like_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message stream { message Foo {} }
					service FooService { rpc Do(.stream.Foo) returns (.stream.Foo); }
				`,
			},
		},
		"success_editions": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					message Foo {
						string foo = 1 [features.field_presence = LEGACY_REQUIRED];
						int32 bar = 2 [features.field_presence = IMPLICIT];
					}
				`,
			},
		},
		"failure_known_not_supported_edition": {
			input: map[string]string{
				"test.proto": `
					edition = "2024";
					import option "bar.proto";
				`,
				"bar.proto": `
					edition = "2023";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.FileOptions {
						bool file_opt1 = 5000;
					}
				`,
			},
			expectedErr:            `test.proto:1:11: edition "2024" not yet fully supported; latest supported edition "2023"`,
			expectedDiffWithProtoc: true, // protoc v32.0 does support edition 2024
		},
		"failure_unknown_edition_future": {
			input: map[string]string{
				"test.proto": `
					edition = "2025";
					message Foo {
						string foo = 1 [features.field_presence = LEGACY_REQUIRED];
						int32 bar = 2 [features.field_presence = IMPLICIT];
					}
				`,
			},
			expectedErr: `test.proto:1:11: edition value "2025" not recognized; should be one of ["2023"]`,
		},
		"failure_unknown_edition_distant_future": {
			input: map[string]string{
				"test.proto": `
					edition = "99999";
					message Foo {
						string foo = 1 [features.field_presence = LEGACY_REQUIRED];
						int32 bar = 2 [features.field_presence = IMPLICIT];
					}
				`,
			},
			expectedErr: `test.proto:1:11: edition value "99999" not recognized; should be one of ["2023"]`,
		},
		"failure_unknown_edition_past": {
			input: map[string]string{
				"test.proto": `
					edition = "2022";
					message Foo {
						string foo = 1 [features.field_presence = LEGACY_REQUIRED];
						int32 bar = 2 [features.field_presence = IMPLICIT];
					}
				`,
			},
			expectedErr: `test.proto:1:11: edition value "2022" not recognized; should be one of ["2023"]`,
		},
		"success_proto2_packed": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message Foo {
					  repeated int32 i32 = 1 [packed=true];
					  repeated int64 i64 = 2 [packed=true];
					  repeated uint32 u32 = 3 [packed=true];
					  repeated uint64 u64 = 4 [packed=true];
					  repeated sint32 s32 = 5 [packed=true];
					  repeated sint64 s64 = 6 [packed=true];
					  repeated fixed32 f32 = 7 [packed=true];
					  repeated fixed64 f64 = 8 [packed=true];
					  repeated sfixed32 sf32 = 9 [packed=true];
					  repeated sfixed64 sf64 = 10 [packed=true];
					  repeated float flt = 11 [packed=true];
					  repeated double dbl = 12 [packed=true];
					  repeated bool bool = 13 [packed=true];
					  repeated En en = 14 [packed=true];
					  enum En { Z=0; A=1; B=2; }
					}
				`,
			},
		},
		"success_proto3_packed": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
					  repeated int32 i32 = 1 [packed=true];
					  repeated int64 i64 = 2 [packed=true];
					  repeated uint32 u32 = 3 [packed=true];
					  repeated uint64 u64 = 4 [packed=true];
					  repeated sint32 s32 = 5 [packed=true];
					  repeated sint64 s64 = 6 [packed=true];
					  repeated fixed32 f32 = 7 [packed=true];
					  repeated fixed64 f64 = 8 [packed=true];
					  repeated sfixed32 sf32 = 9 [packed=true];
					  repeated sfixed64 sf64 = 10 [packed=true];
					  repeated float flt = 11 [packed=true];
					  repeated double dbl = 12 [packed=true];
					  repeated bool bool = 13 [packed=true];
					  repeated En en = 14 [packed=true];
					  enum En { Z=0; A=1; B=2; }
					}
				`,
			},
		},
		"failure_proto2_packed_string": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message Foo {
					  repeated string s = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:12: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto2_packed_bytes": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message Foo {
					  repeated bytes b = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:12: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto2_packed_msg": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message Foo {
					  repeated Foo msgs = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:12: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto2_packed_group": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message Foo {
					  repeated group G = 1 [packed=true] {
					    optional string name = 1;
					  }
					}
				`,
			},
			expectedErr: `test.proto:3:12: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto2_packed_map": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message Foo {
					  map<int32,int32> m = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:3: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto2_packed_nonrepeated": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message Foo {
					  optional int32 i32 = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:3: packed option is only allowed on repeated fields`,
		},
		"failure_proto3_packed_string": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
					  repeated string s = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:12: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto3_packed_bytes": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
					  repeated bytes b = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:12: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto3_packed_msg": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
					  repeated Foo msgs = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:12: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto3_packed_map": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
					  map<int32,int32> m = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:3: packed option is only allowed on numeric, boolean, and enum fields`,
		},
		"failure_proto3_packed_nonrepeated": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto3";
					message Foo {
					  optional int32 i32 = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:3:3: packed option is only allowed on repeated fields`,
		},
		"failure_editions_packed_option_not_allowed": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						repeated uint32 ids = 1 [packed=true];
					}
				`,
			},
			expectedErr: `test.proto:4:34: field foo.A.ids: packed option is not allowed in editions; use option features.repeated_field_encoding instead`,
		},
		"failure_editions_feature_on_wrong_target_type": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					message Foo {
					  int32 i32 = 1 [features.enum_type=OPEN];
					}
				`,
			},
			expectedErr: `test.proto:3:27: field "google.protobuf.FeatureSet.enum_type" is allowed on [enum,file], not on field`,
		},
		"failure_editions_feature_on_wrong_target_type_msg_literal": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					message Foo {
					  int32 i32 = 1 [features={
					    enum_type: OPEN
					  }];
					}
				`,
			},
			expectedErr: `test.proto:4:5: field "google.protobuf.FeatureSet.enum_type" is allowed on [enum,file], not on field`,
		},
		"failure_proto3_enum_zero_value": {
			input: map[string]string{
				"test.proto": `syntax = "proto3"; enum Foo { FIRST = 1; }`,
			},
			expectedErr: `test.proto:1:39: enum Foo: proto3 requires that first value of enum have numeric value zero`,
		},
		"failure_editions_open_enum_zero_value": {
			input: map[string]string{
				"test.proto": `edition = "2023"; enum Foo { FIRST = 1; }`,
			},
			expectedErr: `test.proto:1:38: first value of open enum Foo must have numeric value zero`,
		},
		"success_extension_declarations": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					import "extendee.proto";
					package foo;
					extend A {
						// No declarations, so not validated.
						optional bool one = 1;
						optional string five = 5;
						optional Msg ten = 10;
						// But these ones must match declaration.
						optional int32 eleven = 11;
						repeated Enum twelve = 12;
						optional Msg thirteen = 13;
						optional group Fourteen = 14 { }
					}
					enum Enum { ZERO=0; }
					message Msg { }
				`,
				"extendee.proto": `
					syntax = "proto2";
					message A {
						extensions 1 to 10;
						extensions 11 to 15 [
							verification=DECLARATION,
							declaration={
								number: 11
								full_name: ".foo.eleven"
								type: "int32"
							},
							declaration={
								number: 12
								full_name: ".foo.twelve"
								type: ".foo.Enum"
								repeated: true
							},
							declaration={
								number: 13
								full_name: ".foo.thirteen"
								type: ".foo.Msg"
							},
							declaration={
								number: 14
								full_name: ".foo.fourteen"
								type: ".foo.Fourteen"
							},
							declaration={
								number: 15
								reserved: true
							}
						];
					}
				`,
			},
		},
		"success_extension_declaration_without_verification": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							declaration={
								number: 1
								full_name: ".foo.eleven"
								type: "int32"
							}
						];
					}
				`,
			},
		},
		"failure_extension_declaration_but_range_unverified": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 11 [
							verification=UNVERIFIED,
							declaration={
								number: 11
								full_name: ".foo.eleven"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:4:17: extension range cannot have declarations and have verification of UNVERIFIED`,
		},
		"failure_extension_declaration_without_number": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								full_name: ".foo.bar"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:5:17: extension declaration is missing required field number`,
		},
		"failure_extension_declaration_with_incorrect_number": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 2
								full_name: ".foo.bar"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:6:25: extension declaration has number outside the range: 2 not in [1,1]`,
		},
		"failure_extension_declaration_with_number_out_of_range": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 100 to 500 [
							verification=DECLARATION,
							declaration={
								number: 99
								full_name: ".foo.bar"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:6:25: extension declaration has number outside the range: 99 not in [100,500]`,
		},
		"failure_extension_declaration_with_number_out_of_range2": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 100 to 500, 1000 to 5000 [
							verification=DECLARATION,
							declaration={
								number: 501
								full_name: ".foo.bar"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:6:25: extension declaration has number outside the range: 501 not in [100,500]` +
				` && test.proto:6:25: extension declaration has number outside the range: 501 not in [1000,5000]`,
		},
		"failure_extension_declaration_with_number_out_of_range_multiple_ranges": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 to 3, 5 to 10 [
							verification=DECLARATION,
							declaration={
								number: 3
								full_name: ".foo.bar"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:6:25: extension declaration has number outside the range: 3 not in [5,10]; when using declarations, extends statements should indicate only a single span of field numbers`,
		},
		"failure_extension_declaration_without_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:5:17: extension declaration that is not marked reserved must have a full_name`,
		},
		"failure_extension_declaration_with_name_without_dot": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: "foo.bar"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:7:25: extension declaration full name "foo.bar" should start with a leading dot (.)`,
		},
		"failure_extension_declaration_with_invalid_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".123.5-7.Foobar"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:7:25: extension declaration full name ".123.5-7.Foobar" is not a valid qualified name`,
		},
		"failure_extension_declaration_without_type": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:5:17: extension declaration that is not marked reserved must have a type`,
		},
		"failure_extension_declaration_with_type_without_dot": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "foo.Bar"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:8:25: extension declaration type "foo.Bar" must be a builtin type or start with a leading dot (.)`,
		},
		"failure_extension_declaration_with_invalid_type": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: ".123.Foobar"
							}
						];
					}
				`,
			},
			expectedErr:            `test.proto:8:25: extension declaration type ".123.Foobar" is not a valid qualified name`,
			expectedDiffWithProtoc: true, // Oops. protoc's name validation seems incomplete
		},
		"failure_extension_declaration_with_reserved_only_name": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								reserved: true
								full_name: ".foo.bar"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:8:25: extension declarations that are reserved should specify both full_name and type or neither`,
		},
		"failure_extension_declaration_with_reserved_only_type": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								reserved: true
								type: ".foo.bar"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:8:25: extension declarations that are reserved should specify both full_name and type or neither`,
		},
		"success_extension_declaration_with_reserved_without_name_and_type": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								reserved: true
							}
						];
					}
				`,
			},
		},
		"success_extension_declaration_with_reserved_name_and_type": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								reserved: true
								full_name: ".foo.bar"
								type: ".foo.Baz"
							}
						];
					}
				`,
			},
		},
		// This exercises the code that finds the relevant node to make sure it track indexes of
		// repeated fields correctly.
		"failure_extension_declaration_report_correct_location_with_multiple_declarations": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					message A {
						extensions 1 to 3 [
							verification=DECLARATION,
							declaration={
								number: 1
								reserved: true
							},
							declaration={
								number: 2
								reserved: true
								type: ".foo.bar"
							},
							declaration={
								number: 3
								full_name: ".foo.bar"
								type: ".foo.bar"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:12:25: extension declarations that are reserved should specify both full_name and type or neither`,
		},
		"failure_extension_declarations_repeated_tags": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 to 10 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.Bar"
								type: "int32"
							},
							declaration={
								number: 1
								full_name: ".foo.Baz"
								type: "string"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:12:25: extension for tag number 1 already declared at test.proto:7:25`,
		},
		"failure_extension_declared_multiple_times": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 to 10 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.Bar"
								type: "int32"
							},
							declaration={
								number: 5
								full_name: ".foo.Bar"
								type: "string"
							}
						];
					}
				`,
			},
			expectedErr: `test.proto:13:25: extension foo.Bar already declared as extending foo.A with tag 1 at test.proto:8:25`,
		},
		"failure_extension_declared_multiple_times_across_files": {
			input: map[string]string{
				"test1.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 to 10 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.Bar"
								type: "int32"
							},
							declaration={
								number: 5
								full_name: ".foo.Baz"
								type: "int32"
							}
						];
					}
				`,
				"test2.proto": `
					syntax = "proto2";
					import "test1.proto";
					message B {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.Baz"
								type: "int32"
							}
						];
					}
				`,
			},
			expectedErr: `test2.proto:8:25: extension foo.Baz already declared as extending foo.A with tag 5 at test1.proto:13:25`,
			// protoc only validates that the same name doesn't appear again inside
			// the same extendable message
			expectedDiffWithProtoc: true,
		},
		"failure_extension_number_is_reserved": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								reserved: true
							}
						];
					}
					extend A {
						optional string s = 1;
					}
				`,
			},
			expectedErr: `test.proto:13:29: cannot use field number 1 for an extension because it is reserved in declaration at test.proto:8:25`,
		},
		"failure_extension_name_does_not_match_declaration": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
							}
						];
					}
					extend A {
						optional string s = 1;
					}
				`,
			},
			expectedErr: `test.proto:14:25: expected extension with number 1 to be named foo.bar, not foo.s, per declaration at test.proto:8:25`,
		},
		"failure_extension_name_does_not_match_declaration2": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
							}
						];
					}
					extend A {
						optional string s = 1;
					}
				`,
			},
			expectedErr: `test.proto:13:25: expected extension with number 1 to be named foo.bar, not foo.s, per declaration at test.proto:7:25`,
		},
		"failure_extension_type_does_not_match_declaration": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
							}
						];
					}
					extend A {
						optional uint32 bar = 1;
					}
				`,
			},
			expectedErr: `test.proto:14:18: expected extension with number 1 to have type string, not uint32, per declaration at test.proto:9:25`,
		},
		"failure_extension_type_does_not_match_declaration2": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: ".foo.A"
							}
						];
					}
					extend A {
						optional uint32 bar = 1;
					}
				`,
			},
			expectedErr: `test.proto:13:18: expected extension with number 1 to have type foo.A, not uint32, per declaration at test.proto:8:25`,
		},
		"failure_extension_label_does_not_match_declaration": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
								repeated: true
							}
						];
					}
					extend A {
						optional string bar = 1;
					}
				`,
			},
			expectedErr: `test.proto:15:9: expected extension with number 1 to be repeated, not optional, per declaration at test.proto:10:25`,
		},
		"failure_extension_label_does_not_match_declaration2": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
							}
						];
					}
					extend A {
						repeated string bar = 1;
					}
				`,
			},
			expectedErr: `test.proto:14:9: expected extension with number 1 to be optional, not repeated, per declaration at test.proto:6:17`,
		},
		"failure_extension_label_does_not_match_declaration3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 [
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
							}
						];
					}
					extend A {
						repeated string bar = 1;
					}
				`,
			},
			expectedErr: `test.proto:13:9: expected extension with number 1 to be optional, not repeated, per declaration at test.proto:5:17`,
		},
		"failure_extension_matches_no_declaration": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 to 3 [
							verification=DECLARATION,
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
							}
						];
					}
					extend A {
						repeated string bar = 3;
					}
				`,
			},
			expectedErr: `test.proto:14:31: expected extension with number 3 to be declared in type foo.A, but no declaration found at test.proto:5:17`,
		},
		"failure_extension_matches_no_declaration2": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 to 3 [
							declaration={
								number: 1
								full_name: ".foo.bar"
								type: "string"
							}
						];
					}
					extend A {
						repeated string bar = 3;
					}
				`,
			},
			expectedErr: `test.proto:13:31: expected extension with number 3 to be declared in type foo.A, but no declaration found at test.proto:4:9`,
		},
		"failure_extension_matches_no_declaration3": {
			input: map[string]string{
				"test.proto": `
					syntax = "proto2";
					package foo;
					message A {
						extensions 1 to 3 [
							verification=DECLARATION
						];
					}
					extend A {
						repeated string bar = 3;
					}
				`,
			},
			expectedErr: `test.proto:9:31: expected extension with number 3 to be declared in type foo.A, but no declaration found at test.proto:5:17`,
		},
		"success_field_presence": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					option features.field_presence = IMPLICIT;
					message A {
						B b = 1 [features.field_presence = LEGACY_REQUIRED];
						message B {
							string s = 1 [features.field_presence = EXPLICIT];
						}
						A a = 2 [features.field_presence = EXPLICIT];
						uint64 u = 3 [features.field_presence = IMPLICIT];
					}
				`,
			},
		},
		"failure_field_presence_on_oneof_field": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						oneof b {
							string s = 1 [features.field_presence = EXPLICIT];
							int64 n = 2;
						}
					}
				`,
			},
			expectedErr: `test.proto:5:40: oneof fields may not specify field presence`,
		},
		"failure_field_presence_on_repeated_field": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						repeated string s = 1 [features.field_presence = IMPLICIT];
					}
				`,
			},
			expectedErr: `test.proto:4:41: repeated fields may not specify field presence`,
		},
		"failure_field_presence_on_map_field": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						map<string,string> s = 1 [features.field_presence = IMPLICIT];
					}
				`,
			},
			expectedErr: `test.proto:4:44: repeated fields may not specify field presence`,
		},
		"failure_field_presence_on_message_field": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						A a = 1 [features.field_presence = IMPLICIT];
					}
				`,
			},
			expectedErr: `test.proto:4:27: message fields may not specify implicit presence`,
		},
		"failure_field_presence_on_extension_field": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						extensions 1 to 100;
					}
					extend A {
						string s = 1 [features = {
							field_presence:LEGACY_REQUIRED
						}];
					}
				`,
			},
			expectedErr: `test.proto:8:17: extension fields may not specify field presence`,
		},
		"failure_field_presence_on_file": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					option features={
						field_presence : LEGACY_REQUIRED;
					};
					message A {
						string s = 1;
					}
				`,
			},
			expectedErr: `test.proto:4:9: LEGACY_REQUIRED field presence cannot be set as the default for a file`,
		},
		"success_repeated_field_encoding": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					option features.repeated_field_encoding=EXPANDED;
					message A {
						repeated string s = 1 [features.repeated_field_encoding=EXPANDED];
						map<string,string> m = 2 [features.repeated_field_encoding=EXPANDED];
						repeated A a = 3 [features.repeated_field_encoding=EXPANDED];
						repeated bool b = 4 [features.repeated_field_encoding=PACKED];
						repeated double d = 5 [features.repeated_field_encoding=PACKED];
						repeated E e = 6 [features.repeated_field_encoding=PACKED];
						enum E { ZERO=0; }
					}
				`,
			},
		},
		"failure_repeated_field_encoding_not_repeated": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						string s = 1 [features.repeated_field_encoding=EXPANDED];
					}
				`,
			},
			expectedErr: `test.proto:4:32: only repeated fields may specify repeated field encoding`,
		},
		"failure_repeated_field_encoding_not_packable": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						repeated string s = 1 [features.repeated_field_encoding=PACKED];
					}
				`,
			},
			expectedErr: `test.proto:4:41: only repeated primitive fields may specify packed encoding`,
		},
		"failure_repeated_field_encoding_not_packable_map": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						map<string,string> s = 1 [features.repeated_field_encoding=PACKED];
					}
				`,
			},
			expectedErr: `test.proto:4:44: only repeated primitive fields may specify packed encoding`,
		},
		"success_utf8_validation": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					option features = {
						utf8_validation:NONE;
					};
					message A {
						string s = 1 [features.utf8_validation=VERIFY];
						map<string,A> ma = 2 [features.utf8_validation=VERIFY];
						map<uint32,string> mu = 3 [features.utf8_validation=VERIFY];
						map<string,string> ms = 4 [features.utf8_validation=VERIFY];
					}
				`,
			},
		},
		"failure_utf8_validation_not_string": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						bytes s = 1 [features.utf8_validation=VERIFY];
					}
				`,
			},
			expectedErr: `test.proto:4:31: only string fields may specify UTF8 validation`,
		},
		"failure_utf8_validation_not_string_map": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						map<uint32,bytes> m = 1 [features.utf8_validation=VERIFY];
					}
				`,
			},
			expectedErr: `test.proto:4:43: only string fields may specify UTF8 validation`,
		},
		"failure_utf8_validation_java_option": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					option java_string_check_utf8 = true;
					message A {
						map<uint32,bytes> m = 1;
					}
				`,
			},
			expectedErr: `test.proto:3:8: file option java_string_check_utf8 is not allowed with editions; import "google/protobuf/java_features.proto" and use (pb.java).utf8_validation instead`,
		},
		"success_message_encoding": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					option features.message_encoding = DELIMITED;
					message A {
						A a = 1 [features={
							message_encoding : LENGTH_PREFIXED,
						}];
						repeated A as = 2 [features.message_encoding=DELIMITED];
					}
				`,
			},
		},
		"failure_message_encoding_not_message": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						repeated string s = 1 [features={
							message_encoding : LENGTH_PREFIXED,
						}];
					}
				`,
			},
			expectedErr: `test.proto:5:17: only message fields may specify message encoding`,
		},
		"failure_message_encoding_map": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						map<string,string> m = 1 [features={
							message_encoding : LENGTH_PREFIXED,
						}];
					}
				`,
			},
			expectedErr: `test.proto:5:17: only message fields may specify message encoding`,
		},
		"success_editions_ctype": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					package foo;
					message A {
						string s = 1 [ctype=CORD];
						bytes b = 2 [ctype=STRING_PIECE];
						repeated string r = 3 [ctype=STRING];
						extensions 10 to 100;
					}
					extend A {
						string ext = 10 [ctype=STRING_PIECE];
					}
				`,
			},
		},
		"success_feature_within_lifetime": {
			input: map[string]string{
				"feature.proto": `
					syntax = "proto2";
					package google.protobuf;
					message FileOptions {
						optional FeatureSet features = 50 [
							// Ignored on "features" itself, only
							// validated on fields therein.
							feature_support = {
								edition_introduced: EDITION_2024
							}
						];
					}
					message FeatureSet {
						optional bool flag = 20 [
							feature_support = {
								edition_introduced: EDITION_PROTO2
								edition_deprecated: EDITION_2024
								edition_removed: EDITION_99997_TEST_ONLY
							}
						];
						optional bool other = 21 [
							feature_support = {
								edition_introduced: EDITION_PROTO2
								edition_deprecated: EDITION_2024
								edition_removed: EDITION_99997_TEST_ONLY
							}
						];
					}
					`,
				"test.proto": `
					edition = "2023";
					import "feature.proto";
					package foo;
					option features = { flag: true };
					option features.other = true;
					`,
			},
		},
		"success_feature_deprecated": {
			input: map[string]string{
				"feature.proto": `
					syntax = "proto2";
					package google.protobuf;
					message FileOptions {
						optional FeatureSet features = 50 [
							// Ignored on "features" itself, only
							// validated on fields therein.
							feature_support = {
								edition_introduced: EDITION_PROTO2
								edition_removed: EDITION_2023
							}
						];
					}
					message FeatureSet {
						optional bool flag = 20 [
							feature_support = {
								edition_introduced: EDITION_PROTO2
								edition_deprecated: EDITION_PROTO3
								deprecation_warning: "foo"
							}
						];
						optional bool other = 21 [
							feature_support = {
								edition_introduced: EDITION_PROTO2
								edition_deprecated: EDITION_PROTO3
								deprecation_warning: "foo"
							}
						];
					}
					`,
				"test.proto": `
					edition = "2023";
					import "feature.proto";
					package foo;
					option features = { flag: true };
					option features.other = true;
					`,
			},
		},
		"failure_feature_not_yet_introduced": {
			input: map[string]string{
				"feature.proto": `
					syntax = "proto2";
					package google.protobuf;
					message FileOptions {
						optional FeatureSet features = 50;
					}
					message FeatureSet {
						optional bool flag = 20 [
							feature_support = {
								edition_introduced: EDITION_2024
							}
						];
					}
					`,
				"test.proto": `
					edition = "2023";
					import "feature.proto";
					package foo;
					option features.flag = true;
					`,
			},
			expectedErr: `test.proto:4:1: field "google.protobuf.FeatureSet.flag" was not introduced until edition 2024`,
		},
		"failure_feature_not_yet_introduced_msg_literal": {
			input: map[string]string{
				"feature.proto": `
					syntax = "proto2";
					package google.protobuf;
					message FileOptions {
						optional FeatureSet features = 50;
					}
					message FeatureSet {
						optional bool flag = 20 [
							feature_support = {
								edition_introduced: EDITION_2024
							}
						];
					}
					`,
				"test.proto": `
					edition = "2023";
					import "feature.proto";
					package foo;
					option features = { flag: true };
					`,
			},
			expectedErr: `test.proto:4:21: field "google.protobuf.FeatureSet.flag" was not introduced until edition 2024`,
		},
		"failure_feature_removed": {
			input: map[string]string{
				"feature.proto": `
					syntax = "proto2";
					package google.protobuf;
					message FileOptions {
						optional FeatureSet features = 50;
					}
					message FeatureSet {
						optional bool flag = 20 [
							feature_support = {
								edition_removed: EDITION_PROTO3
							}
						];
					}
					`,
				"test.proto": `
					edition = "2023";
					import "feature.proto";
					package foo;
					option features.flag = true;
					`,
			},
			expectedErr: `test.proto:4:1: field "google.protobuf.FeatureSet.flag" was removed in edition proto3`,
		},
		"failure_feature_removed_msg_literal": {
			input: map[string]string{
				"feature.proto": `
					syntax = "proto2";
					package google.protobuf;
					message FileOptions {
						optional FeatureSet features = 50;
					}
					message FeatureSet {
						optional bool flag = 20 [
							feature_support = {
								edition_removed: EDITION_PROTO3
							}
						];
					}
					`,
				"test.proto": `
					edition = "2023";
					import "feature.proto";
					package foo;
					option features = { flag: true };
					`,
			},
			expectedErr: `test.proto:4:21: field "google.protobuf.FeatureSet.flag" was removed in edition proto3`,
		},
		"success_custom_feature_within_lifetime": {
			input: map[string]string{
				"feature.proto": `
					edition = "2023";
					package foo;
					import "google/protobuf/descriptor.proto";
					message CustomFeatures {
						bool flag = 1 [
							feature_support = {
								edition_introduced: EDITION_2023
								edition_deprecated: EDITION_2024
								edition_removed: EDITION_2024
							}
						];
					}
					extend google.protobuf.FeatureSet {
						CustomFeatures custom = 9995;
					}
					`,
				"test.proto": `
					edition = "2023";
					package foo;
					import "feature.proto";
					option features.(custom).flag = true;
					`,
			},
		},
		"failure_custom_feature_in_same_file": {
			input: map[string]string{
				"test.proto": `
					edition = "2023";
					import "google/protobuf/descriptor.proto";
					package foo;
					message CustomFeatures {
						bool flag = 1 [
							feature_support = {
								edition_introduced: EDITION_2023
								edition_deprecated: EDITION_2023
								edition_removed: EDITION_2024
							}
						];
					}
					extend google.protobuf.FeatureSet {
						CustomFeatures custom = 9995;
					}
					option features.(custom).flag = true;
					`,
			},
			expectedErr: `test.proto:16:1: custom feature (foo.custom) cannot be used from the same file in which it is defined`,
		},
		"success_custom_feature_deprecated": {
			input: map[string]string{
				"feature.proto": `
					edition = "2023";
					package foo;
					import "google/protobuf/descriptor.proto";
					message CustomFeatures {
						bool flag = 1 [
							feature_support = {
								edition_introduced: EDITION_2023
								edition_deprecated: EDITION_2023
								edition_removed: EDITION_2024
							}
						];
					}
					extend google.protobuf.FeatureSet {
						CustomFeatures custom = 9995;
					}
					`,
				"test.proto": `
					edition = "2023";
					package foo;
					import "feature.proto";
					option features.(custom).flag = true;
					`,
			},
		},
		"failure_custom_feature_not_yet_introduced": {
			input: map[string]string{
				"feature.proto": `
					edition = "2023";
					package foo;
					import "google/protobuf/descriptor.proto";
					message CustomFeatures {
						bool flag = 1 [
							feature_support = {
								edition_introduced: EDITION_2024
								edition_deprecated: EDITION_2024
							}
						];
					}
					extend google.protobuf.FeatureSet {
						CustomFeatures custom = 9995;
					}
					`,
				"test.proto": `
					edition = "2023";
					package foo;
					import "feature.proto";
					option features.(custom).flag = true;
					`,
			},
			expectedErr: `test.proto:4:1: field "foo.CustomFeatures.flag" was not introduced until edition 2024`,
		},
		"failure_custom_feature_not_yet_introduced_msg_literal": {
			input: map[string]string{
				"feature.proto": `
					edition = "2023";
					package foo;
					import "google/protobuf/descriptor.proto";
					message CustomFeatures {
						bool flag = 1 [
							feature_support = {
								edition_introduced: EDITION_2024
								edition_deprecated: EDITION_2024
							}
						];
					}
					extend google.protobuf.FeatureSet {
						CustomFeatures custom = 9995;
					}
					`,
				"test.proto": `
					edition = "2023";
					package foo;
					import "feature.proto";
					option features.(custom) = { flag: true };
					`,
			},
			expectedErr: `test.proto:4:30: field "foo.CustomFeatures.flag" was not introduced until edition 2024`,
		},
		"failure_custom_feature_not_yet_introduced_msg_literal2": {
			input: map[string]string{
				"feature.proto": `
					edition = "2023";
					package foo;
					import "google/protobuf/descriptor.proto";
					message CustomFeatures {
						bool flag = 1 [
							feature_support = {
								edition_introduced: EDITION_2024
								edition_deprecated: EDITION_2024
							}
						];
					}
					extend google.protobuf.FeatureSet {
						CustomFeatures custom = 9995;
					}
					`,
				"test.proto": `
					edition = "2023";
					package foo;
					import "feature.proto";
					option features = { [foo.custom]: { flag: true } };
					`,
			},
			expectedErr: `test.proto:4:37: field "foo.CustomFeatures.flag" was not introduced until edition 2024`,
		},
		"failure_custom_feature_removed": {
			input: map[string]string{
				"feature.proto": `
					edition = "2023";
					package foo;
					import "google/protobuf/descriptor.proto";
					message CustomFeatures {
						bool flag = 1 [
							feature_support = {
								edition_introduced: EDITION_PROTO2
								edition_removed: EDITION_2023
							}
						];
					}
					extend google.protobuf.FeatureSet {
						CustomFeatures custom = 9995;
					}
					`,
				"test.proto": `
					edition = "2023";
					package foo;
					import "feature.proto";
					option features.(custom).flag = true;
					`,
			},
			expectedErr: `test.proto:4:1: field "foo.CustomFeatures.flag" was removed in edition 2023`,
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
			for filename, data := range tc.input {
				tc.input[filename] = removePrefixIndent(data)
			}
			files, errs := compile(t, tc.input)

			actualErrs := make([]string, len(errs))
			for i := range errs {
				actualErrs[i] = errs[i].Error()
			}
			expectedErrs := strings.Split(tc.expectedErr, "&&")
			for i, expectedErr := range expectedErrs {
				expectedErrs[i] = strings.TrimSpace(expectedErr)
			}

			switch {
			case tc.expectedErr == "":
				assert.Empty(t, errs, "expecting no error; instead got error:\n%s", strings.Join(actualErrs, "\n"))
			case len(errs) == 0:
				t.Errorf("expecting validation error %q; instead got no error", tc.expectedErr)
			default:
				if len(expectedErrs) == 1 && len(errs) > 1 && len(errs) == strings.Count(expectedErrs[0], "||")+1 {
					// We were expecting one or another error, but got all of them. This can
					// happen since the multiple errors are triggered concurrently and
					// non-deterministically. When it happens, we sort both lists, since we
					// otherwise don't know in what order they could arrive.
					expectedErrs = strings.Split(expectedErrs[0], "||")
					for i := range expectedErrs {
						expectedErrs[i] = strings.TrimSpace(expectedErrs[i])
					}
					sort.Strings(expectedErrs)
					sort.Slice(errs, func(i, j int) bool {
						return errs[i].Error() < errs[j].Error()
					})
				}
				assert.Len(t, errs, len(expectedErrs), "wrong number of errors reported")
				limit := len(expectedErrs)
				if limit > len(errs) {
					limit = len(errs)
				}
				for i := range limit {
					err := errs[i]
					var panicErr protocompile.PanicError
					if errors.As(err, &panicErr) {
						t.Logf("panic! %v\n%s", panicErr.Value, panicErr.Stack)
					}
					expectedErr := expectedErrs[i]
					msgs := strings.Split(expectedErr, "||")
					found := false
					for _, errMsg := range msgs {
						if err.Error() == strings.TrimSpace(errMsg) {
							found = true
							break
						}
					}
					var errNum string
					if len(errs) > 1 {
						errNum = fmt.Sprintf("#%d", i+1)
					}
					assert.True(t, found, "expecting validation error%s %q; instead got: %q", errNum, expectedErr, err)
				}
			}

			// Make sure protobuf-go can handle resulting files
			if len(errs) == 0 && len(files) > 0 {
				err := convertToProtoreflectDescriptors(files)
				if tc.expectProtodescFail {
					// This is a known case where it cannot handle the file.
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			}

			// parse with protoc
			passProtoc := testByProtoc(t, tc.input, tc.inputOrder)
			if tc.expectedErr == "" {
				if tc.expectedDiffWithProtoc {
					// We can explicitly check different result is produced by protoc. When the bug is fixed,
					// we can change the tc.expectedDiffWithProtoc field to false and delete the comment.
					require.False(t, passProtoc, "expected protoc to disallow the case, but it allows it")
				} else {
					// if the test case passes protocompile, it should also pass protoc.
					require.True(t, passProtoc, "protoc should allow the case")
				}
			} else {
				if tc.expectedDiffWithProtoc {
					require.True(t, passProtoc, "expected protoc to allow the case, but it disallows it")
				} else {
					// if the test case fails protocompile, it should also fail protoc.
					require.False(t, passProtoc, "protoc should disallow the case")
				}
			}
		})
	}
}

func removePrefixIndent(s string) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= 1 || strings.TrimSpace(lines[0]) != "" {
		return s
	}
	lines = lines[1:] // skip first blank line
	// determine whitespace prefix from first line (e.g. five tabstops)
	var prefix []rune //nolint:prealloc
	for _, r := range lines[1] {
		if !unicode.IsSpace(r) {
			break
		}
		prefix = append(prefix, r)
	}
	prefixStr := string(prefix)
	for i := range lines {
		lines[i] = strings.TrimPrefix(lines[i], prefixStr)
	}
	return strings.Join(lines, "\n")
}

func compile(t *testing.T, input map[string]string) (linker.Files, []error) {
	t.Helper()
	acc := func(filename string) (io.ReadCloser, error) {
		f, ok := input[filename]
		if !ok {
			return nil, fmt.Errorf("file not found: %s", filename)
		}
		return io.NopCloser(strings.NewReader(f)), nil
	}
	names := make([]string, 0, len(input))
	for k := range input {
		names = append(names, k)
	}

	// We use a reporter that returns nil, so the compile operation
	// will always keep going
	var errsMu sync.Mutex
	var errs []error
	rep := reporter.NewReporter(
		func(err reporter.ErrorWithPos) error {
			errsMu.Lock()
			defer errsMu.Unlock()
			errs = append(errs, err)
			return nil
		},
		nil,
	)

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: acc,
		}),
		Reporter:       rep,
		SourceInfoMode: protocompile.SourceInfoExtraOptionLocations,
	}
	files, err := compiler.Compile(t.Context(), names...)
	if err != nil && len(errs) == 0 {
		t.Log("compiler.Compile returned an error but none were reported")
		return files, []error{err}
	}
	if err == nil {
		assert.Empty(t, errs, "compiler.Compile returned no error though %d errors were reported", len(errs))
	}
	return files, errs
}

func TestProto3Enums(t *testing.T) {
	t.Parallel()
	file1 := `syntax = "<SYNTAX>"; enum bar { A = 0; B = 1; }`
	file2 := `syntax = "<SYNTAX>"; import "f1.proto"; message foo { <LABEL> bar bar = 1; }`
	getFileContents := func(file, syntax string) string {
		contents := strings.Replace(file, "<SYNTAX>", syntax, 1)
		label := ""
		if syntax == "proto2" {
			label = "optional"
		}
		return strings.Replace(contents, "<LABEL>", label, 1)
	}

	syntaxOptions := []string{"proto2", "proto3"}
	for _, o1 := range syntaxOptions {
		fc1 := getFileContents(file1, o1)

		for _, o2 := range syntaxOptions {
			fc2 := getFileContents(file2, o2)

			// now parse the protos with protoc
			testFiles := map[string]string{
				"f1.proto": fc1,
				"f2.proto": fc2,
			}
			fileNames := []string{"f1.proto", "f2.proto"}
			passProtoc := testByProtoc(t, testFiles, fileNames)
			// parse the protos with protocompile
			acc := func(filename string) (io.ReadCloser, error) {
				var data string
				switch filename {
				case "f1.proto":
					data = fc1
				case "f2.proto":
					data = fc2
				default:
					return nil, fmt.Errorf("file not found: %s", filename)
				}
				return io.NopCloser(strings.NewReader(data)), nil
			}
			compiler := protocompile.Compiler{
				Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
					Accessor: acc,
				}),
			}
			_, err := compiler.Compile(t.Context(), "f1.proto", "f2.proto")
			if o1 != o2 && o2 == "proto3" {
				expected := "f2.proto:1:54: cannot use closed enum bar in a field with implicit presence"
				if err == nil {
					t.Errorf("expecting validation error; instead got no error")
				} else if err.Error() != expected {
					t.Errorf("expecting validation error %q; instead got: %q", expected, err)
				}
				require.False(t, passProtoc)
			} else {
				// other cases succeed (okay to for proto2 to use enum from proto3 file and
				// obviously okay for proto2 importing proto2 and proto3 importing proto3)
				require.NoError(t, err)
				require.True(t, passProtoc)
			}
		}
	}
}

func TestLinkerSymbolCollisionNoSource(t *testing.T) {
	t.Parallel()
	fdProto := &descriptorpb.FileDescriptorProto{
		Name:       proto.String("foo.proto"),
		Dependency: []string{"google/protobuf/descriptor.proto"},
		Package:    proto.String("google.protobuf"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("DescriptorProto"),
			},
		},
	}
	resolver := protocompile.WithStandardImports(protocompile.ResolverFunc(func(s string) (protocompile.SearchResult, error) {
		if s == "foo.proto" {
			return protocompile.SearchResult{Proto: fdProto}, nil
		}
		return protocompile.SearchResult{}, protoregistry.NotFound
	}))
	compiler := &protocompile.Compiler{
		Resolver: resolver,
	}
	_, err := compiler.Compile(t.Context(), "foo.proto")
	require.ErrorContains(t, err, `foo.proto: symbol "google.protobuf.DescriptorProto" already defined at google/protobuf/descriptor.proto`)
}

func TestSyntheticMapEntryUsageNoSource(t *testing.T) {
	t.Parallel()
	baseFileDescProto := &descriptorpb.FileDescriptorProto{
		Name: proto.String("foo.proto"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Foo"),
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("BarEntry"),
						Options: &descriptorpb.MessageOptions{
							MapEntry: proto.Bool(true),
						},
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("key"),
								Number:   proto.Int32(1),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								JsonName: proto.String("key"),
							},
							{
								Name:     proto.String("value"),
								Number:   proto.Int32(2),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
								JsonName: proto.String("value"),
							},
						},
					},
				},
			},
		},
	}
	testCases := map[string]struct {
		fields      []*descriptorpb.FieldDescriptorProto
		others      []*descriptorpb.DescriptorProto
		expectedErr string
	}{
		"success_valid_map": {
			fields: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     proto.String("bar"),
					Number:   proto.Int32(1),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".Foo.BarEntry"),
					JsonName: proto.String("bar"),
				},
			},
		},
		"failure_not_repeated": {
			fields: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     proto.String("bar"),
					Number:   proto.Int32(1),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".Foo.BarEntry"),
					JsonName: proto.String("bar"),
				},
			},
			expectedErr: `foo.proto: field Foo.bar: Foo.BarEntry is a synthetic map entry and may not be referenced explicitly`,
		},
		"failure_name_mismatch": {
			fields: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     proto.String("baz"),
					Number:   proto.Int32(1),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".Foo.BarEntry"),
					JsonName: proto.String("baz"),
				},
			},
			expectedErr: `foo.proto: field Foo.baz: Foo.BarEntry is a synthetic map entry and may not be referenced explicitly`,
		},
		"failure_multiple_refs": {
			fields: []*descriptorpb.FieldDescriptorProto{
				{
					Name:     proto.String("bar"),
					Number:   proto.Int32(1),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".Foo.BarEntry"),
					JsonName: proto.String("bar"),
				},
				{
					Name:     proto.String("Bar"),
					Number:   proto.Int32(1),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
					TypeName: proto.String(".Foo.BarEntry"),
					JsonName: proto.String("Bar"),
				},
			},
			expectedErr: `foo.proto: field Foo.Bar: Foo.BarEntry is a synthetic map entry and may not be referenced explicitly`,
		},
		"failure_wrong_message": {
			others: []*descriptorpb.DescriptorProto{
				{
					Name: proto.String("Bar"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:     proto.String("bar"),
							Number:   proto.Int32(1),
							Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
							Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
							TypeName: proto.String(".Foo.BarEntry"),
							JsonName: proto.String("bar"),
						},
					},
				},
			},
			expectedErr: `foo.proto: field Bar.bar: Foo.BarEntry is a synthetic map entry and may not be referenced explicitly`,
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

			fdProto := proto.Clone(baseFileDescProto).(*descriptorpb.FileDescriptorProto) //nolint:errcheck
			fdProto.MessageType[0].Field = tc.fields
			fdProto.MessageType = append(fdProto.MessageType, tc.others...)

			resolver := protocompile.ResolverFunc(func(s string) (protocompile.SearchResult, error) {
				if s == "foo.proto" {
					return protocompile.SearchResult{Proto: fdProto}, nil
				}
				return protocompile.SearchResult{}, protoregistry.NotFound
			})
			compiler := &protocompile.Compiler{
				Resolver: resolver,
			}
			_, err := compiler.Compile(t.Context(), "foo.proto")
			if tc.expectedErr != "" {
				require.EqualError(t, err, tc.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSyntheticOneofCollisions(t *testing.T) {
	t.Parallel()
	input := map[string]string{
		"foo1.proto": `
			syntax = "proto3";
			message Foo {
			  optional string bar = 1;
			}`,
		"foo2.proto": `
			syntax = "proto3";
			message Foo {
			  optional string bar = 1;
			}`,
	}

	var errs []error
	compiler := &protocompile.Compiler{
		Reporter: reporter.NewReporter(
			func(err reporter.ErrorWithPos) error {
				errs = append(errs, err)
				// need to return nil to accumulate all errors so we can report synthetic
				// oneof collision; otherwise, the link will fail after the first collision
				// and we'll never test the synthetic oneofs
				return nil
			},
			nil,
		),
		Resolver: protocompile.ResolverFunc(func(filename string) (protocompile.SearchResult, error) {
			f, ok := input[filename]
			if !ok {
				return protocompile.SearchResult{}, fmt.Errorf("file not found: %s", filename)
			}
			return protocompile.SearchResult{Source: strings.NewReader(removePrefixIndent(f))}, nil
		}),
	}
	_, err := compiler.Compile(t.Context(), "foo1.proto", "foo2.proto")

	assert.Equal(t, reporter.ErrInvalidSource, err)

	// since files are compiled concurrently, there are two possible outcomes
	expectedFoo1FirstErrors := []string{
		`foo2.proto:2:9: symbol "Foo" already defined at foo1.proto:2:9`,
		`foo2.proto:3:19: symbol "Foo.bar" already defined at foo1.proto:3:19`,
		`foo2.proto:3:19: symbol "Foo._bar" already defined at foo1.proto:3:19`,
	}
	expectedFoo2FirstErrors := []string{
		`foo1.proto:2:9: symbol "Foo" already defined at foo2.proto:2:9`,
		`foo1.proto:3:19: symbol "Foo.bar" already defined at foo2.proto:3:19`,
		`foo1.proto:3:19: symbol "Foo._bar" already defined at foo2.proto:3:19`,
	}
	var expected []string
	require.NotEmpty(t, errs)
	actual := make([]string, len(errs))
	for i, err := range errs {
		actual[i] = err.Error()
	}
	if strings.HasPrefix(actual[0], "foo2.proto") {
		expected = expectedFoo1FirstErrors
	} else {
		expected = expectedFoo2FirstErrors
	}
	assert.Equal(t, expected, actual)

	// parse and check with protoc
	passed := testByProtoc(t, input, nil)
	require.False(t, passed)
}

func TestCustomJSONNameWarnings(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		source  string
		warning string
	}{
		{
			source: `
				syntax = "proto2";
				message Foo {
				  optional string foo_bar = 1;
				  optional string fooBar = 2;
				}`,
			warning: `test.proto:4:3: field Foo.fooBar: default JSON name "fooBar" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3`,
		},
		{
			source: `
				syntax = "proto2";
				message Foo {
				  optional string foo_bar = 1;
				  optional string fooBar = 2;
				}`,
			warning: `test.proto:4:3: field Foo.fooBar: default JSON name "fooBar" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3`,
		},
		// in nested message
		{
			source: `
				syntax = "proto2";
				message Blah { message Foo {
				  optional string foo_bar = 1;
				  optional string fooBar = 2;
				} }`,
			warning: `test.proto:4:3: field Foo.fooBar: default JSON name "fooBar" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3`,
		},
		{
			source: `
				syntax = "proto2";
				message Blah { message Foo {
				  optional string foo_bar = 1;
				  optional string fooBar = 2;
				} }`,
			warning: `test.proto:4:3: field Foo.fooBar: default JSON name "fooBar" conflicts with default JSON name of field foo_bar, defined at test.proto:3:3`,
		},
		// enum values
		{
			source: `
				syntax = "proto2";
				enum Foo {
				  true = 0;
				  TRUE = 1;
				}`,
			warning: `test.proto:4:3: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) "True" conflicts with camel-case name of enum value true, defined at test.proto:3:3`,
		},
		{
			source: `
				syntax = "proto2";
				enum Foo {
				  fooBar_Baz = 0;
				  _FOO__BAR_BAZ = 1;
				}`,
			warning: `test.proto:4:3: enum value Foo._FOO__BAR_BAZ: camel-case name (with optional enum name prefix removed) "BarBaz" conflicts with camel-case name of enum value fooBar_Baz, defined at test.proto:3:3`,
		},
		{
			source: `
				syntax = "proto2";
				enum Foo {
				  fooBar_Baz = 0;
				  FOO__BAR__BAZ__ = 1;
				}`,
			warning: `test.proto:4:3: enum value Foo.FOO__BAR__BAZ__: camel-case name (with optional enum name prefix removed) "BarBaz" conflicts with camel-case name of enum value fooBar_Baz, defined at test.proto:3:3`,
		},
		{
			source: `
				syntax = "proto2";
				enum Foo {
				  fooBarBaz = 0;
				  _FOO__BAR_BAZ = 1;
				}`,
			warning: "",
		},
		{
			source: `
				syntax = "proto2";
				enum Foo {
				  option allow_alias = true;
				  Bar_Baz = 0;
				  _BAR_BAZ_ = 0;
				  FOO_BAR_BAZ = 0;
				  foobar_baz = 0;
				}`,
			warning: "",
		},
		// in nested message
		{
			source: `
				syntax = "proto2";
				message Blah { enum Foo {
				  true = 0;
				  TRUE = 1;
				} }`,
			warning: `test.proto:4:3: enum value Foo.TRUE: camel-case name (with optional enum name prefix removed) "True" conflicts with camel-case name of enum value true, defined at test.proto:3:3`,
		},
		{
			source: `
				syntax = "proto2";
				message Blah { enum Foo {
				  fooBar_Baz = 0;
				  _FOO__BAR_BAZ = 1;
				} }`,
			warning: `test.proto:4:3: enum value Foo._FOO__BAR_BAZ: camel-case name (with optional enum name prefix removed) "BarBaz" conflicts with camel-case name of enum value fooBar_Baz, defined at test.proto:3:3`,
		},
		{
			source: `
				syntax = "proto2";
				message Blah { enum Foo {
				  option allow_alias = true;
				  Bar_Baz = 0;
				  _BAR_BAZ_ = 0;
				  FOO_BAR_BAZ = 0;
				  foobar_baz = 0;
				} }`,
			warning: "",
		},
	}
	for i, tc := range testCases {
		resolver := protocompile.ResolverFunc(func(filename string) (protocompile.SearchResult, error) {
			if filename == "test.proto" {
				return protocompile.SearchResult{Source: strings.NewReader(removePrefixIndent(tc.source))}, nil
			}
			return protocompile.SearchResult{}, fmt.Errorf("file not found: %s", filename)
		})
		var warnings []string
		warnFunc := func(err reporter.ErrorWithPos) {
			warnings = append(warnings, err.Error())
		}
		compiler := protocompile.Compiler{
			Resolver: resolver,
			Reporter: reporter.NewReporter(nil, warnFunc),
		}
		_, err := compiler.Compile(t.Context(), "test.proto")
		if err != nil {
			t.Errorf("case %d: expecting no error; instead got error %q", i, err)
		}
		if tc.warning == "" && len(warnings) > 0 {
			t.Errorf("case %d: expecting no warnings; instead got: %v", i, warnings)
		} else if tc.warning != "" {
			found := false
			for _, w := range warnings {
				if w == tc.warning {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("case %d: expecting warning %q; instead got: %v", i, tc.warning, warnings)
			}
		}
	}
	// TODO: Need to run these test cases against protoc like other test
	//  cases in this file. As of writing, the most recent version of
	//  protoc produces too many different result with protocompile. So
	//  we are focusing on other test cases first before protoc is fixed.
}

func testByProtoc(t *testing.T, files map[string]string, fileNames []string) bool {
	t.Helper()
	stdout, err := protoc.Compile(files, fileNames)
	if execErr := new(exec.ExitError); errors.As(err, &execErr) {
		t.Logf("protoc stdout:\n%s\nprotoc stderr:\n%s\n", stdout, execErr.Stderr)
		return false
	}
	require.NoError(t, err)
	return true
}

func convertToProtoreflectDescriptors(files linker.Files) error {
	allFiles := make(map[string]*descriptorpb.FileDescriptorProto, len(files))
	addFileDescriptorsToMap(files, allFiles)
	fileSlice := make([]*descriptorpb.FileDescriptorProto, 0, len(allFiles))
	for _, fileProto := range allFiles {
		fileSlice = append(fileSlice, fileProto)
	}
	_, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: fileSlice})
	return err
}

func addFileDescriptorsToMap[F protoreflect.FileDescriptor](files []F, allFiles map[string]*descriptorpb.FileDescriptorProto) {
	for _, file := range files {
		if _, exists := allFiles[file.Path()]; exists {
			continue // already added this one
		}
		allFiles[file.Path()] = protoutil.ProtoFromFileDescriptor(file)
		deps := make([]protoreflect.FileDescriptor, file.Imports().Len())
		for i := range file.Imports().Len() {
			deps[i] = file.Imports().Get(i).FileDescriptor
		}
		addFileDescriptorsToMap(deps, allFiles)
	}
}
