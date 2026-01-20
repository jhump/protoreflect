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
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/internal"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/reporter"
)

func TestEmptyParse(t *testing.T) {
	t.Parallel()
	errHandler := reporter.NewHandler(nil)
	ast, err := Parse("foo.proto", bytes.NewReader(nil), errHandler)
	require.NoError(t, err)
	result, err := ResultFromAST(ast, true, errHandler, false)
	require.NoError(t, err)
	fd := result.FileDescriptorProto()
	assert.Equal(t, "foo.proto", fd.GetName())
	assert.Empty(t, fd.GetDependency())
	assert.Empty(t, fd.GetMessageType())
	assert.Empty(t, fd.GetEnumType())
	assert.Empty(t, fd.GetExtension())
	assert.Empty(t, fd.GetService())
}

func TestJunkParse(t *testing.T) {
	t.Parallel()
	// inputs that have been found in the past to cause panics by oss-fuzz
	inputs := map[string]string{
		"case-34232": `'';`,
		"case-34238": `.`,
		"issue-196-a": `syntax = "proto3";
		                message TestMessage {
		                  option (ext) = { bad_array: [1,] }
		                }`,
		"issue-196-b": `syntax = "proto3";
		                message TestMessage {
		                  option (ext) = { bad_array [ , ] }
		                }`,
	}
	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			errHandler := reporter.NewHandler(reporter.NewReporter(
				// returning nil means this will keep trying to parse after any error
				func(_ reporter.ErrorWithPos) error { return nil },
				nil, // ignore warnings
			))
			protoName := name + ".proto"
			_, err := Parse(protoName, strings.NewReader(input), errHandler)
			// we expect this to error... but we don't want it to panic
			require.Error(t, err, "junk input should have returned error")
			t.Logf("error from parse: %v", err)
		})
	}
}

type parseErrorTestCase struct {
	Error   string
	NoError string
}

func runParseErrorTestCases(t *testing.T, testCases map[string]parseErrorTestCase, expected string) {
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			protoName := name + ".proto"
			ast, err := Parse(protoName, strings.NewReader(testCase.NoError), reporter.NewHandler(nil))
			require.NoError(t, err)
			_, err = ResultFromAST(ast, true, reporter.NewHandler(nil), false)
			require.NoError(t, err)

			producedError := false
			var errs []error
			errHandler := reporter.NewHandler(reporter.NewReporter(func(err reporter.ErrorWithPos) error {
				errs = append(errs, err)
				if strings.Contains(err.Error(), expected) {
					producedError = true
				}
				return nil
			}, nil))
			file, err := Parse(protoName, strings.NewReader(testCase.Error), errHandler)
			if err == nil {
				_, _ = ResultFromAST(file, true, errHandler, false)
			}
			require.Truef(t, producedError, "expected error containing %q, got %v", expected, errs)
		})
	}
}

func TestLenientParse_NoFieldID(t *testing.T) {
	t.Parallel()
	inputs := map[string]parseErrorTestCase{
		"field": {
			Error: `syntax = "proto3";
							message Foo {
								int32 bar;
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									int32 bar = 1;
								}`,
		},
		"field-cardinality": {
			Error: `syntax = "proto3";
							message Foo {
								repeated int32 bar;
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									repeated int32 bar = 1;
								}`,
		},
		"field-compact-options": {
			Error: `syntax = "proto3";
							message Foo {
								int32 bar [foo = 1];
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									int32 bar = 1 [foo = 1];
								}`,
		},
		"field-cardinality-compact-options": {
			Error: `syntax = "proto3";
							message Foo {
								repeated int32 bar [foo = 1];
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									repeated int32 bar = 1 [foo = 1];
								}`,
		},
		"map-field": {
			Error: `syntax = "proto3";
							message Foo {
								map<string, int32> bar;
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									map<string, int32> bar = 1;
								}`,
		},
		"map-field-compact-options": {
			Error: `syntax = "proto3";
							message Foo {
								map<string, int32> bar [foo = 1];
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									map<string, int32> bar = 1 [foo = 1];
								}`,
		},
		"oneof": {
			Error: `syntax = "proto3";
							message Foo {
								oneof bar {
									int32 baz;
								}
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									oneof bar {
										int32 baz = 1;
									}
								}`,
		},
		"oneof-compact-options": {
			Error: `syntax = "proto3";
							message Foo {
								oneof bar {
									int32 baz [foo = 1];
								}
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									oneof bar {
										int32 baz = 1 [foo = 1];
									}
								}`,
		},
		"oneof-group": {
			Error: `syntax = "proto2";
							message Foo {
								oneof bar {
									group Baz {
										optional int32 baz = 1;
									}
								}
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									oneof bar {
										group Baz = 1 {
											optional int32 baz = 1;
										}
									}
								}`,
		},
		"oneof-group-compact-options": {
			Error: `syntax = "proto2";
							message Foo {
								oneof bar {
									group Baz [foo = 1] {
										optional int32 baz = 1;
									}
								}
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									oneof bar {
										group Baz = 1 [foo = 1] {
											optional int32 baz = 1;
										}
									}
								}`,
		},
		"group-cardinality": {
			Error: `syntax = "proto2";
							message Foo {
								repeated group Bar {
									optional int32 baz = 1;
								}
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									repeated group Bar = 1 {
										optional int32 baz = 1;
									}
								}`,
		},
		"group-cardinality-compact-options": {
			Error: `syntax = "proto2";
							message Foo {
								repeated group Bar [foo = 1] {
									optional int32 baz = 1;
								}
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									repeated group Bar = 1 [foo = 1] {
										optional int32 baz = 1;
									}
								}`,
		},
	}
	runParseErrorTestCases(t, inputs, "missing field tag number")
}

func TestLenientParse_SemicolonLess(t *testing.T) {
	t.Parallel()
	inputs := map[string]parseErrorTestCase{
		"package": {
			Error: `syntax = "proto3";
							package foo
							message Foo {}`,
			NoError: `syntax = "proto3";
								package foo;
								message Foo {};`,
		},
		"import": {
			Error: `syntax = "proto3";
							import "foo.proto"
							message Foo {}`,
			NoError: `syntax = "proto3";
								import "foo.proto";;
								message Foo {};`,
		},
		"file-options": {
			Error: `syntax = "proto3";
							option (foo) = 1
							message Foo {}`,
			NoError: `syntax = "proto3";
								option (foo) = 1;;
								message Foo {};`,
		},
		"method": {
			Error: `syntax = "proto3";
							service Foo {
								;
								rpc Bar (Baz) returns (Qux)
								rpc Qux (Baz) returns (Qux);;
							}`,
			NoError: `syntax = "proto3";
									service Foo {
										;
										rpc Bar (Baz) returns (Qux);
										rpc Qux (Baz) returns (Qux);;
									}`,
		},
		"service-options": {
			Error: `syntax = "proto3";
							service Foo {
								option (foo) = { bar: 1 }
							}`,
			NoError: `syntax = "proto3";
								service Foo {
								  option (foo) = { bar: 1 };
								}`,
		},
		"method-options": {
			Error: `syntax = "proto3";
							service Foo {
								rpc Bar (Baz) returns (Qux) {
									option (foo) = { bar: 1 }
								}
							}`,
			NoError: `syntax = "proto3";
								service Foo {
									rpc Bar (Baz) returns (Qux) {
										;
										option (foo) = { bar: 1 };;
									}
								}`,
		},
		"enum-value": {
			Error: `syntax = "proto3";
							enum Foo {
								FOO = 0
							}`,
			NoError: `syntax = "proto3";
								enum Foo {
									;
									FOO = 0;;
								}`,
		},
		"enum-value-options": {
			Error: `syntax = "proto3";
							enum Foo {
								FOO = 0 [foo = 1]
							}`,
			NoError: `syntax = "proto3";
								enum Foo {
									FOO = 0 [foo = 1];
								}`,
		},
		"enum-options": {
			Error: `syntax = "proto3";
							enum Foo {
								FOO = 0;
								option (foo) = 1
							}`,
			NoError: `syntax = "proto3";
								enum Foo {
									;
									FOO = 0;
									option (foo) = 1;;
								}`,
		},
		"enum-reserved": {
			Error: `syntax = "proto3";
							enum Foo {
								BAR = 0;
								reserved "FOO"
							}`,
			NoError: `syntax = "proto3";
								enum Foo {
									;
									BAR = 0;
									reserved "FOO";;
								}`,
		},
		"oneof-options": {
			Error: `syntax = "proto3";
							message Foo {
								oneof bar {
									option (foo) = 1
									int32 baz = 1;
								}
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									oneof bar {
										option (foo) = 1;
										int32 baz = 1;
									};
								}`,
		},
		"oneof-field": {
			Error: `syntax = "proto3";
							message Foo {
								oneof bar {
									int32 baz = 1
								}
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									oneof bar {
										int32 baz = 1;
									};
								}`,
		},
		"oneof-field-options": {
			Error: `syntax = "proto3";
							message Foo {
								oneof bar {
									int32 baz = 1 [foo = 1]
								}
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									oneof bar {
										int32 baz = 1 [foo = 1];
									};
								}`,
		},
		"extension-field": {
			Error: `syntax = "proto3";
							extend Foo {
								int32 bar = 1
							}`,
			NoError: `syntax = "proto3";
								extend Foo {
									int32 bar = 1;
								}`,
		},
		"extension-field-cardinality": {
			Error: `syntax = "proto3";
							extend Foo {
								repeated int32 bar = 1
							}`,
			NoError: `syntax = "proto3";
								extend Foo {
									repeated int32 bar = 1;
								}`,
		},
		"extension-field-options": {
			Error: `syntax = "proto3";
							extend Foo {
								int32 bar = 1 [foo = 1]
							}`,
			NoError: `syntax = "proto3";
								extend Foo {
									int32 bar = 1 [foo = 1];
								}`,
		},
		"message-field": {
			Error: `syntax = "proto3";
							message Foo {
								int32 bar = 1
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									;
									int32 bar = 1;;
								}`,
		},
		"message-field-cardinality": {
			Error: `syntax = "proto3";
							message Foo {
								repeated int32 bar = 1
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									repeated int32 bar = 1;
								}`,
		},
		"message-field-options": {
			Error: `syntax = "proto3";
							message Foo {
								int32 bar = 1 [foo = 1]
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									int32 bar = 1 [foo = 1];
								}`,
		},
		"message-reserved": {
			Error: `syntax = "proto3";
							message Foo {
								reserved "FOO"
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									;
									reserved "FOO";;
								}`,
		},
		"message-options": {
			Error: `syntax = "proto3";
							message Foo {
								option (foo) = 1
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									;
									option (foo) = 1;;
								}`,
		},
		"message-map-field": {
			Error: `syntax = "proto3";
							message Foo {
								map<string, int32> bar = 1
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									;
									map<string, int32> bar = 1;;
								}`,
		},
		"message-map-field-options": {
			Error: `syntax = "proto3";
							message Foo {
								map<string, int32> bar = 1 [foo = 1]
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									;
									map<string, int32> bar = 1 [foo = 1];;
								}`,
		},
		"message-option": {
			Error: `syntax = "proto3";
							message Foo {
								option (foo) = 1
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									;
									option (foo) = 1;;
								}`,
		},
	}
	runParseErrorTestCases(t, inputs, "syntax error: expecting ';'")
}

func TestLenientParse_EmptyCompactOptions(t *testing.T) {
	t.Parallel()
	inputs := map[string]parseErrorTestCase{
		"field-options": {
			Error: `syntax = "proto2";
							message Foo {
								optional int32 bar = 1 [];
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									optional int32 bar = 1 [default=1];
								}`,
		},
		"enum-options": {
			Error: `syntax = "proto3";
							enum Foo {
								FOO = 0 [];
							}`,
			NoError: `syntax = "proto3";
								enum Foo {
									FOO = 0 [deprecated=true];
								}`,
		},
	}
	runParseErrorTestCases(t, inputs, "compact options must have at least one option")
}

func TestLenientParse_EmptyCompactValue(t *testing.T) {
	t.Parallel()
	inputs := map[string]parseErrorTestCase{
		"field-options": {
			Error: `syntax = "proto2";
							message Foo {
								optional int32 bar = 1 [deprecated=true, default];
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									optional int32 bar = 1 [deprecated=true, default=1];
								}`,
		},
		"enum-options": {
			Error: `syntax = "proto3";
							enum Foo {
								FOO = 0 [deprecated];
							}`,
			NoError: `syntax = "proto3";
								enum Foo {
									FOO = 0 [deprecated=true];
								}`,
		},
	}
	runParseErrorTestCases(t, inputs, "compact option must have a value")
}

func TestLenientParse_OptionsTrailingComma(t *testing.T) {
	t.Parallel()
	inputs := map[string]parseErrorTestCase{
		"field-options": {
			Error: `syntax = "proto2";
							message Foo {
								optional int32 bar = 1 [default=1,];
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									optional int32 bar = 1 [default=1];
								}`,
		},
	}
	runParseErrorTestCases(t, inputs, "syntax error: unexpected ','")
}

func TestLenientParse_OptionNameTrailingDot(t *testing.T) {
	t.Parallel()
	inputs := map[string]parseErrorTestCase{
		"field-options": {
			Error: `syntax = "proto2";
							message Foo {
								optional int32 bar = 1 [default.=1];
							}`,
			NoError: `syntax = "proto2";
								message Foo {
									optional int32 bar = 1 [default=1];
								}`,
		},
		"field-options-ext": {
			Error: `syntax = "proto3";
							message Foo {
								int32 bar = 1 [(foo).=1];
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									int32 bar = 1 [(foo)=1];
								}`,
		},
		"field-options-both": {
			Error: `syntax = "proto3";
							message Foo {
								int32 bar = 1 [baz.(foo).=1];
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									int32 bar = 1 [baz.(foo)=1];
								}`,
		},
		"field-options-type": {
			Error: `syntax = "proto3";
							message Foo {
								int32 bar = 1 [(type.)=1];
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									int32 bar = 1 [(type)=1];
								}`,
		},
		"message-options": {
			Error: `syntax = "proto3";
							message Foo {
								option (foo.) = 1;
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									option (foo) = 1;
								}`,
		},
		"message-option-field": {
			Error: `syntax = "proto3";
							message Foo {
								option (foo.) = {
									[type.]: <a: 1>
								};
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									option (foo) = {
										[type]: <a: 1>
									};
								}`,
		},
		"message-option-any-field": {
			Error: `syntax = "proto3";
							message Foo {
								option (foo) = {
									[url.com/type.]: <a: 1>
								};
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									option (foo) = {
										[url.com/type]: <a: 1>
									};
								}`,
		},
		"message-option-any-field-url": {
			Error: `syntax = "proto3";
							message Foo {
								option (foo) = {
									[url.com./type]: <a: 1>
								};
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									option (foo) = {
										[url.com/type]: <a: 1>
									};
								}`,
		},
		"map-value-type": {
			Error: `syntax = "proto3";
							message Foo {
								map<string, type.> bar = 1;
							}`,
			NoError: `syntax = "proto3";
								message Foo {
									map<string, type> bar = 1;
								}`,
		},
		"extension": {
			Error: `syntax = "proto3";
							extend Foo. {
								int32 bar = 1;
							}`,
			NoError: `syntax = "proto3";
								extend Foo {
									int32 bar = 1;
								}`,
		},
		"method-type": {
			Error: `syntax = "proto3";
							service Foo {
								rpc Bar(type.) returns (type);
							}`,
			NoError: `syntax = "proto3";
								service Foo {
									rpc Bar(type) returns (type);
								}`,
		},
		"method-stream-type": {
			Error: `syntax = "proto3";
							service Foo {
								rpc Bar(stream type) returns (stream A.B.);
							}`,
			NoError: `syntax = "proto3";
								service Foo {
									rpc Bar(stream type) returns (stream A.B);
								}`,
		},
	}
	runParseErrorTestCases(t, inputs, "syntax error: unexpected '.'")
}

func TestSimpleParse(t *testing.T) {
	t.Parallel()
	protos := map[string]Result{}

	// Just verify that we can successfully parse the same files we use for
	// testing. We do a *very* shallow check of what was parsed because we know
	// it won't be fully correct until after linking. (So that will be tested
	// below, where we parse *and* link.)
	res, err := parseFileForTest("../internal/testdata/desc_test1.proto")
	require.NoError(t, err)
	fd := res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/desc_test1.proto", fd.GetName())
	assert.Equal(t, "testprotos", fd.GetPackage())
	assert.True(t, hasExtension(fd, "xtm"))
	assert.True(t, hasMessage(fd, "TestMessage"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/desc_test2.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/desc_test2.proto", fd.GetName())
	assert.Equal(t, "testprotos", fd.GetPackage())
	assert.True(t, hasExtension(fd, "groupx"))
	assert.True(t, hasMessage(fd, "GroupX"))
	assert.True(t, hasMessage(fd, "Frobnitz"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/desc_test_defaults.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/desc_test_defaults.proto", fd.GetName())
	assert.Equal(t, "testprotos", fd.GetPackage())
	assert.True(t, hasMessage(fd, "PrimitiveDefaults"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/desc_test_field_types.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/desc_test_field_types.proto", fd.GetName())
	assert.Equal(t, "testprotos", fd.GetPackage())
	assert.True(t, hasEnum(fd, "TestEnum"))
	assert.True(t, hasMessage(fd, "UnaryFields"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/desc_test_options.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/desc_test_options.proto", fd.GetName())
	assert.Equal(t, "testprotos", fd.GetPackage())
	assert.True(t, hasExtension(fd, "mfubar"))
	assert.True(t, hasEnum(fd, "ReallySimpleEnum"))
	assert.True(t, hasMessage(fd, "ReallySimpleMessage"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/desc_test_proto3.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/desc_test_proto3.proto", fd.GetName())
	assert.Equal(t, "testprotos", fd.GetPackage())
	assert.True(t, hasEnum(fd, "Proto3Enum"))
	assert.True(t, hasService(fd, "TestService"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/desc_test_wellknowntypes.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/desc_test_wellknowntypes.proto", fd.GetName())
	assert.Equal(t, "testprotos", fd.GetPackage())
	assert.True(t, hasMessage(fd, "TestWellKnownTypes"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/nopkg/desc_test_nopkg.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/nopkg/desc_test_nopkg.proto", fd.GetName())
	assert.Equal(t, "", fd.GetPackage())
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/nopkg/desc_test_nopkg_new.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/nopkg/desc_test_nopkg_new.proto", fd.GetName())
	assert.Equal(t, "", fd.GetPackage())
	assert.True(t, hasMessage(fd, "TopLevel"))
	protos[fd.GetName()] = res

	res, err = parseFileForTest("../internal/testdata/pkg/desc_test_pkg.proto")
	require.NoError(t, err)
	fd = res.FileDescriptorProto()
	assert.Equal(t, "../internal/testdata/pkg/desc_test_pkg.proto", fd.GetName())
	assert.Equal(t, "bufbuild.protocompile.test", fd.GetPackage())
	assert.True(t, hasEnum(fd, "Foo"))
	assert.True(t, hasMessage(fd, "Bar"))
	protos[fd.GetName()] = res
}

func TestExportLocalInIdentifiers(t *testing.T) {
	t.Parallel()
	// Verifies that for non-edition-2024 sources, we can correctly
	// handle cases where "export" or "local" appear in the type name.
	// This is done to verify that the grammar changes to support
	// decl visibility don't add back-compat issues with how these
	// keywords could be used in prior editions/syntaxes.
	source := `
		syntax = "proto3";
		message Message {
			export enum = 1;
			export.foo exp_foo = 2;
			local message = 3;
			local.bar loc_bar = 4;
        }
		message export {
			message foo {}
		}
		message local {
			message bar {}
		}`
	ast, err := Parse("test.proto", strings.NewReader(source), reporter.NewHandler(nil))
	require.NoError(t, err)
	res, err := ResultFromAST(ast, true, reporter.NewHandler(nil), false)
	require.NoError(t, err)
	msgProto := res.FileDescriptorProto().GetMessageType()[0]
	assert.Equal(t, "export", msgProto.GetField()[0].GetTypeName())
	assert.Equal(t, "enum", msgProto.GetField()[0].GetName())
	assert.Equal(t, "export.foo", msgProto.GetField()[1].GetTypeName())
	assert.Equal(t, "exp_foo", msgProto.GetField()[1].GetName())
	assert.Equal(t, "local", msgProto.GetField()[2].GetTypeName())
	assert.Equal(t, "message", msgProto.GetField()[2].GetName())
	assert.Equal(t, "local.bar", msgProto.GetField()[3].GetTypeName())
	assert.Equal(t, "loc_bar", msgProto.GetField()[3].GetName())
}

func parseFileForTest(filename string) (Result, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	errHandler := reporter.NewHandler(nil)
	res, err := Parse(filename, f, errHandler)
	if err != nil {
		return nil, err
	}
	return ResultFromAST(res, true, errHandler, false)
}

func hasExtension(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, ext := range fd.Extension {
		if ext.GetName() == name {
			return true
		}
	}
	return false
}

func hasMessage(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, md := range fd.MessageType {
		if md.GetName() == name {
			return true
		}
	}
	return false
}

func hasEnum(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, ed := range fd.EnumType {
		if ed.GetName() == name {
			return true
		}
	}
	return false
}

func hasService(fd *descriptorpb.FileDescriptorProto, name string) bool {
	for _, sd := range fd.Service {
		if sd.GetName() == name {
			return true
		}
	}
	return false
}

func TestAggregateValueInUninterpretedOptions(t *testing.T) {
	t.Parallel()
	res, err := parseFileForTest("../internal/testdata/desc_test_complex.proto")
	require.NoError(t, err)
	fd := res.FileDescriptorProto()

	// service TestTestService, method UserAuth; first option
	aggregateValue1 := *fd.Service[0].Method[0].Options.UninterpretedOption[0].AggregateValue
	assert.Equal(t, "authenticated : true permission : { action : LOGIN entity : \"client\" }", aggregateValue1)

	// service TestTestService, method Get; first option
	aggregateValue2 := *fd.Service[0].Method[1].Options.UninterpretedOption[0].AggregateValue
	assert.Equal(t, "authenticated : true permission : { action : READ entity : \"user\" }", aggregateValue2)

	// message Another; first option
	aggregateValue3 := *fd.MessageType[4].Options.UninterpretedOption[0].AggregateValue
	assert.Equal(t, "foo : \"abc\" s < name : \"foo\" , id : 123 > , array : [ 1 , 2 , 3 ] , r : [ < name : \"f\" > , { name : \"s\" } , { id : 456 } ] ,", aggregateValue3)

	// message Test.Nested._NestedNested; second option (rept)
	//  (Test.Nested is at index 1 instead of 0 because of implicit nested message from map field m)
	aggregateValue4 := *fd.MessageType[1].NestedType[1].NestedType[0].Options.UninterpretedOption[1].AggregateValue
	assert.Equal(t, "foo : \"goo\" [ foo . bar . Test . Nested . _NestedNested . _garblez ] : \"boo\"", aggregateValue4)
}

func TestBasicSuccess(t *testing.T) {
	t.Parallel()
	r := readerForTestdata(t, "largeproto.proto")
	handler := reporter.NewHandler(nil)

	fileNode, err := Parse("largeproto.proto", r, handler)
	require.NoError(t, err)

	result, err := ResultFromAST(fileNode, true, handler, false)
	require.NoError(t, err)
	require.NoError(t, handler.Error())

	assert.Equal(t, "proto3", result.AST().Syntax.Syntax.AsString())
}

func BenchmarkBasicSuccess(b *testing.B) {
	r := readerForTestdata(b, "largeproto.proto")
	bs, err := io.ReadAll(r)
	require.NoError(b, err)

	b.ResetTimer()
	for range b.N {
		b.ReportAllocs()
		byteReader := bytes.NewReader(bs)
		handler := reporter.NewHandler(nil)

		fileNode, err := Parse("largeproto.proto", byteReader, handler)
		require.NoError(b, err)

		result, err := ResultFromAST(fileNode, true, handler, false)
		require.NoError(b, err)
		require.NoError(b, handler.Error())

		assert.Equal(b, "proto3", result.AST().Syntax.Syntax.AsString())
	}
}

func readerForTestdata(t testing.TB, filename string) io.Reader {
	file, err := os.Open(filepath.Join("testdata", filename))
	require.NoError(t, err)

	return file
}

func TestPathological(t *testing.T) {
	t.Parallel()
	if internal.IsRace {
		// Note, the combo of race detector and coverage is the real death
		// knell here. But since there's not an easy way to detect if coverage
		// instrumentation is enabled, and we currently always run coverage and
		// race detection together in CI, we can just rely on the race detector.
		//
		// In CI, we do run the tests without race detection or coverage
		// instrumentation, mainly to verify that everything works with the
		// "protolegacy" build tag. So that's when these tests will actually run.
		t.Skip("skipping pathological input test cases because race detector is enabled (which slows things down too much)")
	}

	// This test verifies that the test cases found in fuzz tests have
	// adequate performance.
	//   https://oss-fuzz.com/testcase-detail/4766256800858112
	//   https://oss-fuzz.com/testcase-detail/4952577018298368
	//   https://oss-fuzz.com/testcase-detail/5539164995518464
	testCases := map[string]bool{
		"pathological.proto":  true,
		"pathological2.proto": false,
		"pathological3.proto": false,
	}
	for fileName := range testCases {
		t.Run(fileName, func(t *testing.T) {
			t.Parallel()
			// Fuzz testing complains if this loop, with 100 iterations, takes longer
			// than 60 seconds. We're only running 3 iterations, so this test isn't
			// too slow. So we can use a much tighter deadline.
			allowedDuration := 2 * time.Second
			start := time.Now()
			ctx, cancel := context.WithTimeout(t.Context(), allowedDuration)
			defer func() {
				if ctx.Err() != nil {
					t.Errorf("test took too long to execute (> %v)", allowedDuration)
				}
				t.Logf("test completed in %v", time.Since(start))
				cancel()
			}()
			for range 3 {
				if ctx.Err() != nil {
					break
				}
				r := readerForTestdata(t, fileName)
				handler := reporter.NewHandler(nil)
				fileNode, err := Parse(fileName, r, handler)
				if testCases[fileName] {
					require.NoError(t, err)
					_, err = ResultFromAST(fileNode, true, handler, false)
				}
				require.Error(t, err)
			}
		})
	}
}

func TestExportLocalIdentifier(t *testing.T) {
	t.Parallel()
	t.Run("field_names", func(t *testing.T) {
		t.Parallel()
		proto3Content := `syntax = "proto3";
						  message Test {
						    string export = 1;
						    string local = 2;
						  }`

		errHandler := reporter.NewHandler(nil)
		ast, err := Parse("test.proto", strings.NewReader(proto3Content), errHandler)
		require.NoError(t, err, "Should be able to parse export/local as field names in proto3")

		result, err := ResultFromAST(ast, true, errHandler, false)
		require.NoError(t, err, "Should be able to create result from AST")

		fd := result.FileDescriptorProto()
		assert.Equal(t, "proto3", fd.GetSyntax())
		require.Len(t, fd.GetMessageType(), 1)

		msg := fd.GetMessageType()[0]
		require.Equal(t, "Test", msg.GetName())
		require.Len(t, msg.GetField(), 2)

		fields := msg.GetField()
		require.Equal(t, "export", fields[0].GetName())
		require.Equal(t, "local", fields[1].GetName())
	})
	t.Run("type_names", func(t *testing.T) {
		t.Parallel()
		proto2Content := `syntax = "proto2";
						  message Test {
						    optional export.Message field1 = 1;
						    optional local.Message field2 = 2;
						  }`

		errHandler := reporter.NewHandler(nil)
		ast, err := Parse("test.proto", strings.NewReader(proto2Content), errHandler)
		require.NoError(t, err, "Should be able to parse export/local as type names in proto2")

		result, err := ResultFromAST(ast, true, errHandler, false)
		require.NoError(t, err, "Should be able to create result from AST")

		fd := result.FileDescriptorProto()
		require.Len(t, fd.GetMessageType(), 1)

		msg := fd.GetMessageType()[0]
		require.Equal(t, "Test", msg.GetName())
		require.Len(t, msg.GetField(), 2)

		fields := msg.GetField()
		require.Equal(t, "field1", fields[0].GetName())
		require.Equal(t, "export.Message", fields[0].GetTypeName())
		require.Equal(t, "field2", fields[1].GetName())
		require.Equal(t, "local.Message", fields[1].GetTypeName())
	})
}
