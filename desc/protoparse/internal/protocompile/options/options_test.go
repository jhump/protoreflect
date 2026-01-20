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

package options_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/internal/prototest"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/linker"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/options"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/parser"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/protoutil"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/reporter"
)

type ident string
type aggregate string

func TestCustomOptionsAreKnown(t *testing.T) {
	t.Parallel()
	for _, withOverride := range []bool{false, true} {
		name := "no overrides"
		if withOverride {
			name = "with override descriptor.proto"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			sources := map[string]string{
				"test.proto": `
					syntax = "proto3";
					import "other.proto";
					option (string_option) = "abc";
					`,
				"other.proto": `
					syntax = "proto3";
					import public "options.proto";
					`,
				"options.proto": `
					syntax = "proto3";
					import "google/protobuf/descriptor.proto";
					extend google.protobuf.FileOptions {
						string string_option = 10101;
					}
					`,
			}
			resolver := protocompile.Resolver(&protocompile.SourceResolver{
				Accessor: protocompile.SourceAccessorFromMap(sources),
			})
			if withOverride {
				sources["google/protobuf/descriptor.proto"] = `
					syntax = "proto2";
					package google.protobuf;
					message FileOptions {
						optional string foo = 1;
						optional bool bar = 2;
						optional int32 baz = 3;
						extensions 1000 to max;
					}
					`
			} else {
				resolver = protocompile.WithStandardImports(resolver)
			}
			compiler := &protocompile.Compiler{
				Resolver: resolver,
			}
			files, err := compiler.Compile(t.Context(), "test.proto")
			require.NoError(t, err)
			require.Len(t, files, 1)
			var knownOptionNames []string
			fileOptions := files[0].Options().ProtoReflect()
			assert.Empty(t, fileOptions.GetUnknown())
			fileOptions.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
				if fd.IsExtension() {
					knownOptionNames = append(knownOptionNames, string(fd.FullName()))
				}
				return true
			})
			sort.Strings(knownOptionNames)
			assert.Equal(t, []string{"string_option"}, knownOptionNames)
		})
	}
}

func TestOptionsInUnlinkedFiles(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name             string
		contents         string
		uninterpreted    map[string]any
		checkInterpreted func(*testing.T, *descriptorpb.FileDescriptorProto)
	}{
		{
			name:     "file options",
			contents: `option go_package = "foo.bar"; option (must.link) = "FOO";`,
			uninterpreted: map[string]any{
				"test.proto:(must.link)": "FOO",
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.Equal(t, "foo.bar", fd.GetOptions().GetGoPackage())
			},
		},
		{
			name:     "file options, not custom",
			contents: `option go_package = "foo.bar"; option must_link = "FOO";`,
			uninterpreted: map[string]any{
				"test.proto:must_link": "FOO",
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.Equal(t, "foo.bar", fd.GetOptions().GetGoPackage())
			},
		},
		{
			name:     "message options",
			contents: `message Test { option (must.link) = 1.234; option deprecated = true; }`,
			uninterpreted: map[string]any{
				"Test:(must.link)": 1.234,
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.True(t, fd.GetMessageType()[0].GetOptions().GetDeprecated())
			},
		},
		{
			name:     "field options",
			contents: `message Test { optional string uid = 1 [(must.link) = 10101, (must.link) = 20202, default = "fubar", json_name = "UID", deprecated = true]; }`,
			uninterpreted: map[string]any{
				"Test.uid:(must.link)":   10101,
				"Test.uid:(must.link)#1": 20202,
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.Equal(t, "fubar", fd.GetMessageType()[0].GetField()[0].GetDefaultValue())
				assert.Equal(t, "UID", fd.GetMessageType()[0].GetField()[0].GetJsonName())
				assert.True(t, fd.GetMessageType()[0].GetField()[0].GetOptions().GetDeprecated())
			},
		},
		{
			name:     "field options, default uninterpretable",
			contents: `enum TestEnum{ ZERO = 0; ONE = 1; } message Test { optional TestEnum uid = 1 [(must.link) = {foo: bar}, default = ONE, json_name = "UID", deprecated = true]; }`,
			uninterpreted: map[string]any{
				"Test.uid:(must.link)": aggregate("foo : bar"),
				"Test.uid:default":     ident("ONE"),
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.Equal(t, "UID", fd.GetMessageType()[0].GetField()[0].GetJsonName())
				assert.True(t, fd.GetMessageType()[0].GetField()[0].GetOptions().GetDeprecated())
			},
		},
		{
			name:     "oneof options",
			contents: `message Test { oneof x { option (must.link) = true; option deprecated = true; string uid = 1; uint64 nnn = 2; } }`,
			uninterpreted: map[string]any{
				"Test.x:(must.link)": ident("true"),
				"Test.x:deprecated":  ident("true"), // one-ofs do not have deprecated option :/
			},
		},
		{
			name:     "extension range options",
			contents: `message Test { extensions 100 to 200 [(must.link) = "foo", deprecated = true]; }`,
			uninterpreted: map[string]any{
				"Test.100-200:(must.link)": "foo",
				"Test.100-200:deprecated":  ident("true"), // extension ranges do not have deprecated option :/
			},
		},
		{
			name:     "enum options",
			contents: `enum Test { option allow_alias = true; option deprecated = true; option (must.link) = 123.456; ZERO = 0; ZILCH = 0; }`,
			uninterpreted: map[string]any{
				"Test:(must.link)": 123.456,
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.True(t, fd.GetEnumType()[0].GetOptions().GetDeprecated())
				assert.True(t, fd.GetEnumType()[0].GetOptions().GetAllowAlias())
			},
		},
		{
			name:     "enum value options",
			contents: `enum Test { ZERO = 0 [deprecated = true, (must.link) = -222]; }`,
			uninterpreted: map[string]any{
				"Test.ZERO:(must.link)": -222,
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.True(t, fd.GetEnumType()[0].GetValue()[0].GetOptions().GetDeprecated())
			},
		},
		{
			name:     "service options",
			contents: `service Test { option deprecated = true; option (must.link) = {foo:1, foo:2, bar:3}; }`,
			uninterpreted: map[string]any{
				"Test:(must.link)": aggregate("foo : 1 , foo : 2 , bar : 3"),
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.True(t, fd.GetService()[0].GetOptions().GetDeprecated())
			},
		},
		{
			name:     "method options",
			contents: `import "google/protobuf/empty.proto"; service Test { rpc Foo (google.protobuf.Empty) returns (google.protobuf.Empty) { option deprecated = true; option (must.link) = FOO; } }`,
			uninterpreted: map[string]any{
				"Test.Foo:(must.link)": ident("FOO"),
			},
			checkInterpreted: func(t *testing.T, fd *descriptorpb.FileDescriptorProto) {
				assert.True(t, fd.GetService()[0].GetMethod()[0].GetOptions().GetDeprecated())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := reporter.NewHandler(nil)
			ast, err := parser.Parse("test.proto", strings.NewReader(tc.contents), h)
			require.NoError(t, err, "failed to parse")
			res, err := parser.ResultFromAST(ast, true, h, false)
			require.NoError(t, err, "failed to produce descriptor proto")
			_, err = options.InterpretUnlinkedOptions(res)
			require.NoError(t, err, "failed to interpret options")
			actual := map[string]any{}
			buildUninterpretedMapForFile(res.FileDescriptorProto(), actual)
			assert.Equal(t, tc.uninterpreted, actual, "resulted in wrong uninterpreted options")
			if tc.checkInterpreted != nil {
				tc.checkInterpreted(t, res.FileDescriptorProto())
			}
		})
	}
}

func TestOptionsInUnlinkedFileInvalid(t *testing.T) {
	t.Parallel()
	h := reporter.NewHandler(nil)
	ast, err := parser.Parse(
		"test.proto",
		strings.NewReader(
			`syntax = "proto2";
			    package foo;
			    option malformed_non_existent = true;
			    option features.utf8_validation = NONE;`,
		), h)
	require.NoError(t, err, "failed to parse")
	res, err := parser.ResultFromAST(ast, false, h, false)
	require.NoError(t, err, "failed to produce descriptor proto")
	_, err = options.InterpretUnlinkedOptions(res)
	require.ErrorContains(t, err,
		`test.proto:4:29: field "google.protobuf.FeatureSet.utf8_validation" was not introduced until edition 2023`)
}

func buildUninterpretedMapForFile(fd *descriptorpb.FileDescriptorProto, opts map[string]any) {
	buildUninterpretedMap(fd.GetName(), fd.GetOptions().GetUninterpretedOption(), opts)
	for _, md := range fd.GetMessageType() {
		buildUninterpretedMapForMessage(fd.GetPackage(), md, opts)
	}
	for _, extd := range fd.GetExtension() {
		buildUninterpretedMap(qualify(fd.GetPackage(), extd.GetName()), extd.GetOptions().GetUninterpretedOption(), opts)
	}
	for _, ed := range fd.GetEnumType() {
		buildUninterpretedMapForEnum(fd.GetPackage(), ed, opts)
	}
	for _, sd := range fd.GetService() {
		svcFqn := qualify(fd.GetPackage(), sd.GetName())
		buildUninterpretedMap(svcFqn, sd.GetOptions().GetUninterpretedOption(), opts)
		for _, mtd := range sd.GetMethod() {
			buildUninterpretedMap(qualify(svcFqn, mtd.GetName()), mtd.GetOptions().GetUninterpretedOption(), opts)
		}
	}
}

func buildUninterpretedMapForMessage(qual string, md *descriptorpb.DescriptorProto, opts map[string]any) {
	fqn := qualify(qual, md.GetName())
	buildUninterpretedMap(fqn, md.GetOptions().GetUninterpretedOption(), opts)
	for _, fld := range md.GetField() {
		buildUninterpretedMap(qualify(fqn, fld.GetName()), fld.GetOptions().GetUninterpretedOption(), opts)
	}
	for _, ood := range md.GetOneofDecl() {
		buildUninterpretedMap(qualify(fqn, ood.GetName()), ood.GetOptions().GetUninterpretedOption(), opts)
	}
	for _, extr := range md.GetExtensionRange() {
		buildUninterpretedMap(qualify(fqn, fmt.Sprintf("%d-%d", extr.GetStart(), extr.GetEnd()-1)), extr.GetOptions().GetUninterpretedOption(), opts)
	}
	for _, nmd := range md.GetNestedType() {
		buildUninterpretedMapForMessage(fqn, nmd, opts)
	}
	for _, extd := range md.GetExtension() {
		buildUninterpretedMap(qualify(fqn, extd.GetName()), extd.GetOptions().GetUninterpretedOption(), opts)
	}
	for _, ed := range md.GetEnumType() {
		buildUninterpretedMapForEnum(fqn, ed, opts)
	}
}

func buildUninterpretedMapForEnum(qual string, ed *descriptorpb.EnumDescriptorProto, opts map[string]any) {
	fqn := qualify(qual, ed.GetName())
	buildUninterpretedMap(fqn, ed.GetOptions().GetUninterpretedOption(), opts)
	for _, evd := range ed.GetValue() {
		buildUninterpretedMap(qualify(fqn, evd.GetName()), evd.GetOptions().GetUninterpretedOption(), opts)
	}
}

func buildUninterpretedMap(prefix string, uos []*descriptorpb.UninterpretedOption, opts map[string]any) {
	for _, uo := range uos {
		parts := make([]string, len(uo.GetName()))
		for i, np := range uo.GetName() {
			if np.GetIsExtension() {
				parts[i] = fmt.Sprintf("(%s)", np.GetNamePart())
			} else {
				parts[i] = np.GetNamePart()
			}
		}
		uoName := fmt.Sprintf("%s:%s", prefix, strings.Join(parts, "."))
		key := uoName
		i := 0
		for {
			if _, ok := opts[key]; !ok {
				break
			}
			i++
			key = fmt.Sprintf("%s#%d", uoName, i)
		}
		var val any
		switch {
		case uo.AggregateValue != nil:
			val = aggregate(uo.GetAggregateValue())
		case uo.IdentifierValue != nil:
			val = ident(uo.GetIdentifierValue())
		case uo.DoubleValue != nil:
			val = uo.GetDoubleValue()
		case uo.PositiveIntValue != nil:
			val = int(uo.GetPositiveIntValue())
		case uo.NegativeIntValue != nil:
			val = int(uo.GetNegativeIntValue())
		default:
			val = string(uo.GetStringValue())
		}
		opts[key] = val
	}
}

func qualify(qualifier, name string) string {
	if qualifier == "" {
		return name
	}
	return qualifier + "." + name
}

func TestOptionsEncoding(t *testing.T) {
	t.Parallel()
	testCases := map[string]string{
		"proto2-opts-same-file": "options/options.proto",
		"proto2":                "options/test.proto",
		"proto3":                "options/test_proto3.proto",
		"editions":              "options/test_editions.proto",
		"defaults":              "desc_test_defaults.proto",
	}
	for testCaseName, file := range testCases {
		t.Run(testCaseName, func(t *testing.T) {
			t.Parallel()
			fileToCompile := strings.TrimPrefix(file, "options/")
			importPath := "../internal/testdata"
			if fileToCompile != file {
				importPath = "../internal/testdata/options"
			}
			compiler := protocompile.Compiler{
				Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
					ImportPaths: []string{importPath},
				}),
			}
			fds, err := compiler.Compile(t.Context(), fileToCompile)
			var panicErr protocompile.PanicError
			if errors.As(err, &panicErr) {
				t.Logf("panic! %v\n%s", panicErr.Value, panicErr.Stack)
			}
			require.NoError(t, err)

			// Make sure we can round-trip the descriptors
			res, ok := fds[0].(linker.Result)
			require.True(t, ok)
			actualFdset := &descriptorpb.FileDescriptorSet{
				File: []*descriptorpb.FileDescriptorProto{protoutil.ProtoFromFileDescriptor(res)},
			}
			data, err := proto.Marshal(actualFdset)
			require.NoError(t, err)
			resolver := linker.ResolverFromFile(fds[0])
			var roundTripped descriptorpb.FileDescriptorSet
			err = proto.UnmarshalOptions{Resolver: resolver}.Unmarshal(data, &roundTripped)
			require.NoError(t, err)
			prototest.AssertMessagesEqual(t, actualFdset, &roundTripped, "round-tripped "+file)

			// drum roll... make sure the descriptors we produce are semantically equivalent
			// to those produced by protoc
			descriptorSetFile := fmt.Sprintf("../internal/testdata/%sset", file)
			fdset := prototest.LoadDescriptorSet(t, descriptorSetFile, resolver)
			if !prototest.CheckFiles(t, res, fdset, false) {
				outputDescriptorSetFile := strings.ReplaceAll(descriptorSetFile, ".proto", ".actual.proto")
				actualData, err := proto.Marshal(actualFdset)
				require.NoError(t, err)
				err = os.WriteFile(outputDescriptorSetFile, actualData, 0644)
				if err != nil {
					t.Logf("failed to write actual to file: %v", err)
				} else {
					t.Logf("wrote actual contents to %s", outputDescriptorSetFile)
				}
			}
		})
	}
}

func TestInterpretOptionsWithoutAST(t *testing.T) {
	t.Parallel()

	// First compile from source, so we interpret options with an AST
	fileNames := []string{"desc_test_options.proto", "desc_test_comments.proto", "desc_test_complex.proto"}
	compiler := &protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testdata"},
		}),
	}
	files, err := compiler.Compile(t.Context(), fileNames...)
	require.NoError(t, err)

	// Now compile without the AST, to make sure we interpret options the same way
	compiler = &protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(protocompile.ResolverFunc(
			func(name string) (protocompile.SearchResult, error) {
				var res protocompile.SearchResult
				data, err := os.ReadFile(filepath.Join("../internal/testdata", name))
				if err != nil {
					return res, err
				}
				fileNode, err := parser.Parse(name, bytes.NewReader(data), reporter.NewHandler(nil))
				if err != nil {
					return res, err
				}
				parseResult, err := parser.ResultFromAST(fileNode, true, reporter.NewHandler(nil), false)
				if err != nil {
					return res, err
				}
				res.Proto = parseResult.FileDescriptorProto()
				return res, nil
			},
		)),
	}
	filesFromNoAST, err := compiler.Compile(t.Context(), fileNames...)
	require.NoError(t, err)

	for _, file := range files {
		fromNoAST := filesFromNoAST.FindFileByPath(file.Path())
		require.NotNil(t, fromNoAST)
		fd := file.(linker.Result).FileDescriptorProto()               //nolint:errcheck
		fdFromNoAST := fromNoAST.(linker.Result).FileDescriptorProto() //nolint:errcheck
		// final protos, with options interpreted, match
		prototest.AssertMessagesEqual(t, fd, fdFromNoAST, file.Path())
	}
}

//nolint:errcheck
func TestInterpretOptionsWithoutASTNoOp(t *testing.T) {
	t.Parallel()
	// Similar to above test, where we have descriptor protos and no AST. But this
	// time, interpreting options is a no-op because they all have options already.

	// First compile from source, so we interpret options with an AST
	fileNames := []string{"desc_test_options.proto", "desc_test_comments.proto", "desc_test_complex.proto"}
	compiler := &protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testdata"},
		}),
	}
	files, err := compiler.Compile(t.Context(), fileNames...)
	require.NoError(t, err)

	// Now compile with just the protos, with options already interpreted, to make
	// sure it's a no-op and that we don't mangle any already-interpreted options.
	compiler = &protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(protocompile.ResolverFunc(
			func(name string) (protocompile.SearchResult, error) {
				var res protocompile.SearchResult
				fd := files.FindFileByPath(name)
				if fd == nil {
					file, err := protoregistry.GlobalFiles.FindFileByPath(name)
					if err != nil {
						return res, err
					}
					res.Proto = protodesc.ToFileDescriptorProto(file)
				} else {
					res.Proto = fd.(linker.Result).FileDescriptorProto()
				}
				res.Proto = proto.Clone(res.Proto).(*descriptorpb.FileDescriptorProto)
				return res, nil
			},
		)),
	}
	filesFromNoAST, err := compiler.Compile(t.Context(), fileNames...)
	require.NoError(t, err)

	for _, file := range files {
		fromNoAST := filesFromNoAST.FindFileByPath(file.Path())
		require.NotNil(t, fromNoAST)
		fd := file.(linker.Result).FileDescriptorProto()
		fdFromNoAST := fromNoAST.(linker.Result).FileDescriptorProto()
		// final protos, with options interpreted, match
		prototest.AssertMessagesEqual(t, fd, fdFromNoAST, file.Path())
	}
}

func TestInterpretOptionsFeatureLifetimeWarnings(t *testing.T) {
	t.Parallel()
	sources := map[string]string{
		"features.proto": `
			syntax = "proto2";
			package google.protobuf;
			message FileOptions {
				optional FeatureSet features = 50;
			}
			message FeatureSet {
				extensions 1000;
				optional bool okay = 200 [
					feature_support = {
						edition_introduced: EDITION_2023
					}
				];
				optional bool removed = 201 [
					feature_support = {
						edition_introduced: EDITION_2023
						edition_removed: EDITION_2024
					}
				];
				optional bool deprecated = 202 [
					feature_support = {
						edition_introduced: EDITION_2023
						edition_deprecated: EDITION_2023
						deprecation_warning: "do not use this!"
					}
				];
				optional bool deprecated_and_removed = 203 [
					feature_support = {
						edition_introduced: EDITION_2023
						edition_deprecated: EDITION_2023
						deprecation_warning: "don't use this either!"
						edition_removed: EDITION_2024
					}
				];
			}
			`,
		"custom_features.proto": `
			edition = "2023";
			import "features.proto";
			extend google.protobuf.FeatureSet {
				Custom custom = 1000;
			}
			message Custom {
				bool okay = 200 [
					feature_support = {
						edition_introduced: EDITION_2023
					}
				];
				bool removed = 201 [
					feature_support = {
						edition_introduced: EDITION_2023
						edition_removed: EDITION_2024
					}
				];
				bool deprecated = 202 [
					feature_support = {
						edition_introduced: EDITION_2023
						edition_deprecated: EDITION_2023
						deprecation_warning: "custom feature is not to be used"
					}
				];
				bool deprecated_and_removed = 203 [
					feature_support = {
						edition_introduced: EDITION_2023
						edition_deprecated: EDITION_2023
						deprecation_warning: "other custom feature is not to be used either"
						edition_removed: EDITION_2024
					}
				];
			}
			`,
		"test.proto": `
			edition = "2023";
			import "features.proto";
			import "custom_features.proto";
			option features.okay = true;
			option features.deprecated = true;
			option features.deprecated_and_removed = true;
			option features.(custom).okay = true;
			option features.(custom).deprecated = true;
			option features.(custom).deprecated_and_removed = true;
			`,
	}
	var warnings []string
	rep := reporter.NewReporter(nil, func(err reporter.ErrorWithPos) {
		warnings = append(warnings, err.Error())
	})
	compiler := &protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(sources),
		}),
		Reporter: rep,
	}
	_, err := compiler.Compile(t.Context(), "test.proto")
	require.NoError(t, err)
	expectedWarnings := []string{
		`test.proto:10:25: field "Custom.deprecated_and_removed" is deprecated as of edition 2023: other custom feature is not to be used either`,
		`test.proto:6:25: field "google.protobuf.FeatureSet.deprecated" is deprecated as of edition 2023: do not use this!`,
		`test.proto:7:25: field "google.protobuf.FeatureSet.deprecated_and_removed" is deprecated as of edition 2023: don't use this either!`,
		`test.proto:9:25: field "Custom.deprecated" is deprecated as of edition 2023: custom feature is not to be used`,
	}
	sort.Strings(warnings)
	assert.Equal(t, expectedWarnings, warnings)
}
