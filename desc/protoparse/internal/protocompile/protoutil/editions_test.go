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

package protoutil_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/gofeaturespb"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/internal/editions"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/linker"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/protoutil"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/walk"
)

func TestResolveFeature(t *testing.T) {
	t.Parallel()
	testResolveFeature(t)
}

func TestResolveFeature_Dynamic(t *testing.T) {
	t.Parallel()
	descriptorProto := protodesc.ToFileDescriptorProto(
		(*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().ParentFile(),
	)
	// Provide our own version of descriptor.proto, so the FeatureSet
	// descriptor will be dynamically built.
	testResolveFeature(t, descriptorProto)

	// Also test with an extra field (not recognized by descriptorpb).
	t.Run("editions-new-field", func(t *testing.T) {
		t.Parallel()
		var found bool
		descriptorProto := proto.Clone(descriptorProto).(*descriptorpb.FileDescriptorProto) //nolint:errcheck
		for _, msg := range descriptorProto.MessageType {
			if msg.GetName() == "FeatureSet" {
				msg.Field = append(msg.Field, &descriptorpb.FieldDescriptorProto{
					Name:     proto.String("fubar"),
					Number:   proto.Int32(8888),
					Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
					Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
					TypeName: proto.String(".google.protobuf.FeatureSet.Fubar"),
					JsonName: proto.String("fubar"),
					Options: &descriptorpb.FieldOptions{
						Targets: []descriptorpb.FieldOptions_OptionTargetType{
							descriptorpb.FieldOptions_TARGET_TYPE_FILE,
							descriptorpb.FieldOptions_TARGET_TYPE_MESSAGE,
							descriptorpb.FieldOptions_TARGET_TYPE_SERVICE,
						},
						EditionDefaults: []*descriptorpb.FieldOptions_EditionDefault{
							{
								Edition: descriptorpb.Edition_EDITION_PROTO2.Enum(),
								Value:   proto.String("FOO"),
							},
							{
								Edition: descriptorpb.Edition_EDITION_2023.Enum(),
								Value:   proto.String("BAR"),
							},
						},
					},
				})
				msg.EnumType = append(msg.EnumType, &descriptorpb.EnumDescriptorProto{
					Name: proto.String("Fubar"),
					Value: []*descriptorpb.EnumValueDescriptorProto{
						{
							Name:   proto.String("FUBAR_UNKNOWN"),
							Number: proto.Int32(0),
						},
						{
							Name:   proto.String("FOO"),
							Number: proto.Int32(1),
						},
						{
							Name:   proto.String("BAR"),
							Number: proto.Int32(2),
						},
						{
							Name:   proto.String("BAZ"),
							Number: proto.Int32(3),
						},
					},
				})
				found = true
				break
			}
		}
		require.True(t, found)

		sourceResolver := &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				"test.proto": `
					edition = "2023";
					message Foo {
						option features.fubar = FOO;
					}
					message Bar {
						// default feature value, which is Bar
					}
					service Baz {
						option features.fubar = BAZ;
						rpc Do(Foo) returns (Bar);
					}`,
			}),
		}
		file, featureSetDescriptor := compileFile(t, "test.proto", sourceResolver, descriptorProto)

		feature := featureSetDescriptor.Fields().ByName("fubar")
		val, err := protoutil.ResolveFeature(file, feature)
		require.NoError(t, err)
		// Value is the default for edition 2023
		require.Equal(t, protoreflect.EnumNumber(2), val.Enum())

		elem := file.FindDescriptorByName("Foo")
		require.NotNil(t, elem)
		val, err = protoutil.ResolveFeature(elem, feature)
		require.NoError(t, err)
		require.Equal(t, protoreflect.EnumNumber(1), val.Enum())

		elem = file.FindDescriptorByName("Bar")
		require.NotNil(t, elem)
		val, err = protoutil.ResolveFeature(elem, feature)
		require.NoError(t, err)
		require.Equal(t, protoreflect.EnumNumber(2), val.Enum())

		elem = file.FindDescriptorByName("Baz")
		require.NotNil(t, elem)
		val, err = protoutil.ResolveFeature(elem, feature)
		require.NoError(t, err)
		require.Equal(t, protoreflect.EnumNumber(3), val.Enum())
	})
}

func testResolveFeature(t *testing.T, deps ...*descriptorpb.FileDescriptorProto) {
	t.Run("proto2", func(t *testing.T) {
		t.Parallel()
		file, featureSetDescriptor := compileFile(t, "desc_test1.proto", nil, deps...)

		feature := featureSetDescriptor.Fields().ByName("json_format")
		val, err := protoutil.ResolveFeature(file, feature)
		require.NoError(t, err)
		// Value is the default for proto2
		require.Equal(t, descriptorpb.FeatureSet_LEGACY_BEST_EFFORT.Number(), val.Enum())

		// Same value for a field therein
		field := file.FindDescriptorByName("testprotos.AnotherTestMessage.RockNRoll.beatles")
		require.NotNil(t, field)
		val, err = protoutil.ResolveFeature(field, feature)
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FeatureSet_LEGACY_BEST_EFFORT.Number(), val.Enum())
	})

	t.Run("proto3", func(t *testing.T) {
		t.Parallel()
		file, featureSetDescriptor := compileFile(t, "desc_test_proto3.proto", nil, deps...)

		feature := featureSetDescriptor.Fields().ByName("utf8_validation")
		val, err := protoutil.ResolveFeature(file, feature)
		require.NoError(t, err)
		// Value is the default for proto3
		require.Equal(t, descriptorpb.FeatureSet_VERIFY.Number(), val.Enum())

		// Same value for a field therein
		field := file.FindDescriptorByName("testprotos.TestRequest.FlagsEntry.value")
		require.NotNil(t, field)
		val, err = protoutil.ResolveFeature(field, feature)
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FeatureSet_VERIFY.Number(), val.Enum())
	})

	t.Run("editions-defaults", func(t *testing.T) {
		t.Parallel()
		file, featureSetDescriptor := compileFile(t, "editions/all_default_features.proto", nil, deps...)

		feature := featureSetDescriptor.Fields().ByName("repeated_field_encoding")
		val, err := protoutil.ResolveFeature(file, feature)
		require.NoError(t, err)
		// Value is the default for editions
		require.Equal(t, descriptorpb.FeatureSet_PACKED.Number(), val.Enum())

		// Same value for a field therein
		field := file.FindDescriptorByName("foo.bar.Foo.Bar.abc")
		require.NotNil(t, field)
		val, err = protoutil.ResolveFeature(field, feature)
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FeatureSet_PACKED.Number(), val.Enum())
	})

	t.Run("editions-overrides", func(t *testing.T) {
		t.Parallel()
		file, featureSetDescriptor := compileFile(t, "editions/features_with_overrides.proto", nil, deps...)

		feature := featureSetDescriptor.Fields().ByName("field_presence")
		val, err := protoutil.ResolveFeature(file, feature)
		require.NoError(t, err)
		// Value is from explicit file-wide default
		require.Equal(t, descriptorpb.FeatureSet_IMPLICIT.Number(), val.Enum())

		// Overridden value for a field therein
		field := file.FindDescriptorByName("foo.bar.baz.Bar.left")
		require.NotNil(t, field)
		val, err = protoutil.ResolveFeature(field, feature)
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FeatureSet_EXPLICIT.Number(), val.Enum())

		// Let's check another feature
		feature = featureSetDescriptor.Fields().ByName("utf8_validation")
		val, err = protoutil.ResolveFeature(file, feature)
		require.NoError(t, err)
		// Value is the default for editions
		require.Equal(t, descriptorpb.FeatureSet_VERIFY.Number(), val.Enum())

		field = file.FindDescriptorByName("foo.bar.baz.Foo.JklEntry.key")
		require.NotNil(t, field)
		val, err = protoutil.ResolveFeature(field, feature)
		require.NoError(t, err)
		require.Equal(t, descriptorpb.FeatureSet_NONE.Number(), val.Enum())
	})
}

func TestResolveCustomFeature(t *testing.T) {
	t.Parallel()
	descriptorProto := protodesc.ToFileDescriptorProto(
		(*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().ParentFile(),
	)
	optionsSource := `
		edition = "2023";
		package test;
		import "google/protobuf/descriptor.proto";
		extend google.protobuf.FeatureSet {
			CustomFeatures custom = 9996;
		}
		message CustomFeatures {
			bool encabulate = 1 [
				targets=TARGET_TYPE_FILE,
				targets=TARGET_TYPE_FIELD,
				edition_defaults ={
					edition: EDITION_PROTO2
					value: "true"
				},
				edition_defaults = {
					edition: EDITION_2023
					value: "false"
				}
			];
			Frob nitz = 2 [
				targets=TARGET_TYPE_FILE,
				targets=TARGET_TYPE_MESSAGE,
				edition_defaults = {
					edition: EDITION_PROTO2
					value: "POWER_CYCLE"
				},
				edition_defaults = {
					edition: EDITION_PROTO3
					value: "RTFM"
				},
				edition_defaults = {
					edition: EDITION_2023
					value: "ID_10_T"
				}
			];
		}
		enum Frob {
			FROB_UNKNOWN = 0;
			POWER_CYCLE = 1;
			RTFM = 2;
			ID_10_T = 3;
		}
		`

	// We can do proto2 and proto3 in the same way since they
	// can't override feature values.
	testCases := []struct {
		syntax             string
		expectedEncabulate bool
		expectedNitz       int32
	}{
		{
			syntax:             "proto2",
			expectedEncabulate: true,
			expectedNitz:       1,
		},
		{
			syntax:             "proto3",
			expectedEncabulate: true,
			expectedNitz:       2,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.syntax, func(t *testing.T) {
			t.Parallel()
			sourceResolver := &protocompile.SourceResolver{
				Accessor: protocompile.SourceAccessorFromMap(map[string]string{
					"options.proto": optionsSource,
					"test.proto": `
						syntax = "` + testCase.syntax + `";
						import "options.proto";
						message Foo {
						}`,
				}),
			}
			file, _ := compileFile(t, "test.proto", sourceResolver, descriptorProto)
			// First we resolve the feature with the given file.
			// Then we'll do a second pass where we resolve the
			// feature, but all extensions are unrecognized. Both
			// ways should work.
			for _, clearKnownExts := range []bool{false, true} {
				if clearKnownExts {
					clearKnownExtensionsFromFile(t, protoutil.ProtoFromFileDescriptor(file))
				}

				optionsFile := file.FindImportByPath("options.proto")
				extType := dynamicpb.NewExtensionType(optionsFile.FindDescriptorByName("test.custom").(protoreflect.ExtensionDescriptor)) //nolint:errcheck
				feature := optionsFile.FindDescriptorByName("test.CustomFeatures.encabulate").(protoreflect.FieldDescriptor)              //nolint:errcheck

				val, err := protoutil.ResolveCustomFeature(file, extType, feature)
				require.NoError(t, err)
				require.Equal(t, testCase.expectedEncabulate, val.Bool())

				// Same value for an element therein
				elem := file.FindDescriptorByName("Foo")
				require.NotNil(t, elem)
				val, err = protoutil.ResolveCustomFeature(elem, extType, feature)
				require.NoError(t, err)
				require.Equal(t, testCase.expectedEncabulate, val.Bool())

				// Check the other feature field, too
				feature = optionsFile.FindDescriptorByName("test.CustomFeatures.nitz").(protoreflect.FieldDescriptor) //nolint:errcheck
				val, err = protoutil.ResolveCustomFeature(file, extType, feature)
				require.NoError(t, err)
				require.Equal(t, protoreflect.EnumNumber(testCase.expectedNitz), val.Enum())

				val, err = protoutil.ResolveCustomFeature(elem, extType, feature)
				require.NoError(t, err)
				require.Equal(t, protoreflect.EnumNumber(testCase.expectedNitz), val.Enum())
			}
		})
	}

	t.Run("editions", func(t *testing.T) {
		t.Parallel()
		sourceResolver := &protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				"options.proto": optionsSource,
				"test.proto": `
					edition = "2023";
					import "options.proto";
					message Foo {
					}
					message Bar {
						option features.(test.custom).nitz = RTFM;
						string name = 1 [
							features.(test.custom).encabulate = true
						];
						bytes extra = 2;
					}`,
			}),
		}
		file, _ := compileFile(t, "test.proto", sourceResolver, descriptorProto)
		// First we resolve the feature with the given file.
		// Then we'll do a second pass where we resolve the
		// feature, but all extensions are unrecognized. Both
		// ways should work.
		for _, clearKnownExts := range []bool{false, true} {
			if clearKnownExts {
				clearKnownExtensionsFromFile(t, protoutil.ProtoFromFileDescriptor(file))
			}

			optionsFile := file.FindImportByPath("options.proto")
			extType := dynamicpb.NewExtensionType(optionsFile.FindDescriptorByName("test.custom").(protoreflect.ExtensionDescriptor)) //nolint:errcheck
			feature := optionsFile.FindDescriptorByName("test.CustomFeatures.encabulate").(protoreflect.FieldDescriptor)              //nolint:errcheck

			val, err := protoutil.ResolveCustomFeature(file, extType, feature)
			require.NoError(t, err)
			// Default for edition
			require.False(t, val.Bool())

			// Override
			field := file.FindDescriptorByName("Bar.name")
			require.NotNil(t, field)
			val, err = protoutil.ResolveCustomFeature(field, extType, feature)
			require.NoError(t, err)
			require.True(t, val.Bool())

			// Check the other feature field, too
			feature = optionsFile.FindDescriptorByName("test.CustomFeatures.nitz").(protoreflect.FieldDescriptor) //nolint:errcheck
			val, err = protoutil.ResolveCustomFeature(file, extType, feature)
			require.NoError(t, err)
			require.Equal(t, protoreflect.EnumNumber(3), val.Enum())

			val, err = protoutil.ResolveCustomFeature(field, extType, feature)
			require.NoError(t, err)
			require.Equal(t, protoreflect.EnumNumber(2), val.Enum())
		}
	})
}

func TestResolveCustomFeature_Generated(t *testing.T) {
	t.Parallel()
	descriptorProto := protodesc.ToFileDescriptorProto(
		(*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().ParentFile(),
	)
	goFeaturesProto := protodesc.ToFileDescriptorProto(
		(*gofeaturespb.GoFeatures)(nil).ProtoReflect().Descriptor().ParentFile(),
	)

	// We can do proto2 and proto3 in the same way since they
	// can't override feature values.
	preEditionsTestCases := []struct {
		syntax        string
		expectedValue bool
	}{
		{
			syntax:        "proto2",
			expectedValue: true,
		},
		{
			syntax:        "proto3",
			expectedValue: false,
		},
	}
	for _, testCase := range preEditionsTestCases {
		t.Run(testCase.syntax, func(t *testing.T) {
			t.Parallel()
			sourceResolver := &protocompile.SourceResolver{
				Accessor: protocompile.SourceAccessorFromMap(map[string]string{
					"test.proto": `
						syntax = "` + testCase.syntax + `";
						import "google/protobuf/go_features.proto";
						enum Foo {
							ZERO = 0;
						}`,
				}),
			}
			file, _ := compileFile(t, "test.proto", sourceResolver, descriptorProto, goFeaturesProto)
			// First we resolve the feature with the given file.
			// Then we'll do a second pass where we resolve the
			// feature, but all extensions are unrecognized. Both
			// ways should work.
			for _, clearKnownExts := range []bool{false, true} {
				if clearKnownExts {
					clearKnownExtensionsFromFile(t, protoutil.ProtoFromFileDescriptor(file))
				}

				extType := gofeaturespb.E_Go
				feature := gofeaturespb.E_Go.TypeDescriptor().Message().Fields().ByName("legacy_unmarshal_json_enum")
				require.NotNil(t, feature)

				// Default for edition
				val, err := protoutil.ResolveCustomFeature(file, extType, feature)
				require.NoError(t, err)
				require.Equal(t, testCase.expectedValue, val.Bool())

				// Same value for an element therein
				elem := file.FindDescriptorByName("Foo")
				require.NotNil(t, elem)
				val, err = protoutil.ResolveCustomFeature(elem, extType, feature)
				require.NoError(t, err)
				require.Equal(t, testCase.expectedValue, val.Bool())
			}
		})
	}

	editionsTestCases := []struct {
		name           string
		source         string
		exopectedValue bool
	}{
		{
			name: "editions-2023-default",
			source: `
				edition = "2023";
				import "google/protobuf/go_features.proto";
				enum Foo {
					ZERO = 0;
				}`,
			exopectedValue: false,
		},
		{
			name: "editions-override",
			source: `
				edition = "2023";
				import "google/protobuf/go_features.proto";
				enum Foo {
					option features.(pb.go).legacy_unmarshal_json_enum = true;
					ZERO = 0;
				}`,
			exopectedValue: true,
		},
	}

	for _, testCase := range editionsTestCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			sourceResolver := &protocompile.SourceResolver{
				Accessor: protocompile.SourceAccessorFromMap(map[string]string{
					"test.proto": testCase.source,
				}),
			}
			file, _ := compileFile(t, "test.proto", sourceResolver, descriptorProto, goFeaturesProto)
			// First we resolve the feature with the given file.
			// Then we'll do a second pass where we resolve the
			// feature, but all extensions are unrecognized. Both
			// ways should work.
			for _, clearKnownExts := range []bool{false, true} {
				if clearKnownExts {
					clearKnownExtensionsFromFile(t, protoutil.ProtoFromFileDescriptor(file))
				}

				extType := gofeaturespb.E_Go
				feature := gofeaturespb.E_Go.TypeDescriptor().Message().Fields().ByName("legacy_unmarshal_json_enum")
				require.NotNil(t, feature)

				val, err := protoutil.ResolveCustomFeature(file, extType, feature)
				require.NoError(t, err)
				// Edition default is false, and can't be overridden at the file level,
				// so this should always be false.
				require.False(t, val.Bool())

				// Override
				elem := file.FindDescriptorByName("Foo")
				require.NotNil(t, elem)
				val, err = protoutil.ResolveCustomFeature(elem, extType, feature)
				require.NoError(t, err)
				require.Equal(t, testCase.exopectedValue, val.Bool())
			}
		})
	}
}

func compileFile(
	t *testing.T,
	filename string,
	sources *protocompile.SourceResolver,
	deps ...*descriptorpb.FileDescriptorProto,
) (result linker.File, featureSet protoreflect.MessageDescriptor) {
	t.Helper()
	if sources == nil {
		sources = &protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testdata"},
		}
	}
	resolver := protocompile.Resolver(sources)
	if len(deps) > 0 {
		resolver = addDepsToResolver(resolver, deps...)
	}
	compiler := &protocompile.Compiler{Resolver: resolver}
	names := make([]string, len(deps)+1)
	names[0] = filename
	for i := range deps {
		names[i+1] = deps[i].GetName()
	}
	files, err := compiler.Compile(t.Context(), names...)
	require.NoError(t, err)

	// See if compile included version of google.protobuf.FeatureSet
	var featureSetDescriptor protoreflect.MessageDescriptor
	desc, err := files.AsResolver().FindDescriptorByName(editions.FeatureSetDescriptor.FullName())
	if err != nil {
		featureSetDescriptor = editions.FeatureSetDescriptor
	} else {
		featureSetDescriptor = desc.(protoreflect.MessageDescriptor) //nolint:errcheck
	}

	return files[0], featureSetDescriptor
}

func addDepsToResolver(resolver protocompile.Resolver, deps ...*descriptorpb.FileDescriptorProto) protocompile.Resolver {
	if len(deps) == 0 {
		return resolver
	}
	depsByPath := make(map[string]*descriptorpb.FileDescriptorProto, len(deps))
	for _, dep := range deps {
		depsByPath[dep.GetName()] = dep
	}
	return protocompile.ResolverFunc(func(path string) (protocompile.SearchResult, error) {
		file := depsByPath[path]
		if file != nil {
			return protocompile.SearchResult{Proto: file}, nil
		}
		return resolver.FindFileByPath(path)
	})
}

func clearKnownExtensionsFromFile(t *testing.T, file *descriptorpb.FileDescriptorProto) {
	t.Helper()
	clearKnownExtensionsFromOptions(t, file.GetOptions())
	err := walk.DescriptorProtos(file, func(_ protoreflect.FullName, element proto.Message) error {
		switch element := element.(type) {
		case *descriptorpb.DescriptorProto:
			clearKnownExtensionsFromOptions(t, element.GetOptions())
			for _, extRange := range element.GetExtensionRange() {
				clearKnownExtensionsFromOptions(t, extRange.GetOptions())
			}
		case *descriptorpb.FieldDescriptorProto:
			clearKnownExtensionsFromOptions(t, element.GetOptions())
		case *descriptorpb.OneofDescriptorProto:
			clearKnownExtensionsFromOptions(t, element.GetOptions())
		case *descriptorpb.EnumDescriptorProto:
			clearKnownExtensionsFromOptions(t, element.GetOptions())
		case *descriptorpb.EnumValueDescriptorProto:
			clearKnownExtensionsFromOptions(t, element.GetOptions())
		case *descriptorpb.ServiceDescriptorProto:
			clearKnownExtensionsFromOptions(t, element.GetOptions())
		case *descriptorpb.MethodDescriptorProto:
			clearKnownExtensionsFromOptions(t, element.GetOptions())
		}
		return nil
	})
	require.NoError(t, err)
}

func clearKnownExtensionsFromOptions(t *testing.T, options proto.Message) {
	t.Helper()
	if options == nil || !options.ProtoReflect().IsValid() {
		return // nothing to do
	}
	data, err := proto.Marshal(options)
	require.NoError(t, err)
	// We unmarshal from bytes, with a nil resolver, so all extensions
	// will remain unrecognized.
	err = proto.UnmarshalOptions{Resolver: (*protoregistry.Types)(nil)}.Unmarshal(data, options)
	require.NoError(t, err)
}
