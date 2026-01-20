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

//go:build !protolegacy

package options_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile"
)

func TestOptionWithMessageSetWireFormat(t *testing.T) {
	t.Parallel()
	// When the protobuf-go runtime doesn't support message sets, we
	// disallow using them in options, since the resulting descriptors
	// returned by the compiler would not be serializable.
	compiler := &protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(map[string]string{
				"test.proto": `
					syntax = "proto2";
					import "google/protobuf/descriptor.proto";
					message MessageSet {
						option message_set_wire_format = true;
						extensions 1 to max;
					}
					message Foo {
						extend MessageSet {
							optional Foo message_set_field = 12345;
						}
						optional string name = 1;
					}
					extend google.protobuf.FileOptions {
						optional MessageSet m = 10101;
					}
					option (m).(Foo.message_set_field).name = "abc";`,
			}),
		}),
	}
	_, err := compiler.Compile(t.Context(), "test.proto")
	require.ErrorContains(t, err, `test.proto:17:52: field "Foo.message_set_field" may not be used in an option: it uses 'message set wire format' legacy proto1 feature which is not supported`)
}
