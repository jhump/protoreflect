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

package fastscan

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScan(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name            string
		input           string
		expectedImports []Import
		expectedPackage string
		expectedErrors  []string
	}{
		{
			name: "simple",
			input: `syntax = "proto2";
				package abc.xyz;
				import "foo/bar/baz.proto";
				import "google/protobuf/descriptor.proto";
				import "xyz/123.proto";
				message Foo {}
				service Bar { rpc Baz(Foo) returns (Foo); }
			`,
			expectedImports: []Import{
				{Path: "foo/bar/baz.proto"},
				{Path: "google/protobuf/descriptor.proto"},
				{Path: "xyz/123.proto"},
			},
			expectedPackage: "abc.xyz",
		},
		{
			name: "out of order",
			input: `syntax = "proto2";
				import "foo/bar/baz.proto";
				message Foo {}
				import "google/protobuf/descriptor.proto";
				service Bar { rpc Baz(Foo) returns (Foo); }
				import "xyz/123.proto";
				package abc.xyz;
			`,
			expectedImports: []Import{
				{Path: "foo/bar/baz.proto"},
				{Path: "google/protobuf/descriptor.proto"},
				{Path: "xyz/123.proto"},
			},
			expectedPackage: "abc.xyz",
		},
		{
			name: "last package wins",
			input: `syntax = "proto2";
				package foo.bar;
				message Foo {}
				package abc.xyz;
			`,
			expectedImports: nil,
			expectedPackage: "abc.xyz",
		},
		{
			name: "syntax errors prevent parsing some imports",
			input: `syntax = "proto2";
				package abc.xyz;
				import "foo/bar/baz.proto":
				import "google/protobuf/descriptor.proto":
				import "xyz/123.proto";
				message Foo {}
				service Bar { rpc Baz(Foo) returns (Foo); }
			`,
			expectedImports: []Import{
				{Path: "foo/bar/baz.proto"},
			},
			expectedPackage: "abc.xyz",
			expectedErrors: []string{
				`<input>:3:59: unexpected ':'; expecting semicolon`,
			},
		},
		{
			name: "syntax errors imports",
			input: `syntax = "proto2";
				import "foo/bar/baz.proto";
				import "xyz/123.proto"
				message Foo {}
				import "google/protobuf/descriptor.proto":
				service Bar { rpc Baz(Foo) returns (Foo); }
				import foo "blah/blah/blah.proto";
			`,
			expectedImports: []Import{
				{Path: "foo/bar/baz.proto"},
				{Path: "xyz/123.proto"},
				{Path: "google/protobuf/descriptor.proto"},
			},
			expectedPackage: "",
			expectedErrors: []string{
				`<input>:4:33: unexpected identifier; expecting semicolon`,
				`<input>:5:74: unexpected ':'; expecting semicolon`,
				`<input>:7:40: unexpected identifier; expecting import path string`,
			},
		},
		{
			name: "syntax errors package",
			input: `syntax = "proto3";
				package .abc.xyz;
				package "abc.com";
				package abc.com. ;
				package foo.bar 123
				message Foo {}
				package foo . . bar;
				package foo bar;
				package foo . bar;
			`,
			expectedImports: nil,
			expectedPackage: "foo.bar",
			expectedErrors: []string{
				`<input>:2:41: package name should not begin with a period`,
				`<input>:3:41: unexpected string literal; expecting package name`,
				`<input>:4:48: package name should not end with a period`,
				`<input>:5:49: unexpected numeric literal; expecting semicolon`,
				`<input>:7:47: package name should not have two periods in a row`,
				`<input>:8:45: package name should have a period between name components`,
			},
		},
		{
			name: "nothing",
			input: `syntax = "proto2";
				message Foo {}
			`,
			expectedImports: nil,
			expectedPackage: "",
		},
		{
			name: "public and weak imports",
			input: `syntax = "proto2";
				package abc.xyz;
				import public "foo/bar/baz.proto";
				import weak "google/protobuf/descriptor.proto";
			`,
			expectedImports: []Import{
				{Path: "foo/bar/baz.proto", IsPublic: true},
				{Path: "google/protobuf/descriptor.proto", IsWeak: true},
			},
			expectedPackage: "abc.xyz",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result, err := Scan("", strings.NewReader(testCase.input))
			assert.Equal(t, testCase.expectedPackage, result.PackageName)
			assert.Equal(t, testCase.expectedImports, result.Imports)
			if len(testCase.expectedErrors) == 0 {
				require.NoError(t, err)
				return
			}
			var syntaxErr SyntaxError
			require.ErrorAs(t, err, &syntaxErr)
			assert.Len(t, syntaxErr, len(testCase.expectedErrors))
			for i := 0; i < len(testCase.expectedErrors) && i < len(syntaxErr); i++ {
				assert.ErrorContains(t, syntaxErr[i], testCase.expectedErrors[i])
			}
		})
	}
}
