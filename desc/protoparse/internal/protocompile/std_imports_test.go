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

package protocompile

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
)

func TestStdImports(t *testing.T) {
	t.Parallel()
	// make sure we can successfully "compile" all standard imports
	// (by regurgitating the built-in descriptors)
	c := Compiler{Resolver: WithStandardImports(&SourceResolver{})}
	ctx := t.Context()
	for name, fileProto := range standardImports {
		t.Log(name)
		fds, err := c.Compile(ctx, name)
		if err != nil {
			t.Errorf("failed to compile %q: %v", name, err)
			continue
		}
		if len(fds) != 1 {
			t.Errorf("Compile returned wrong number of descriptors: expecting 1, got %d", len(fds))
			continue
		}
		orig := protodesc.ToFileDescriptorProto(fileProto)
		actual := protodesc.ToFileDescriptorProto(fds[0])
		if !proto.Equal(orig, actual) {
			t.Errorf("result proto is incorrect:\n expecting %v\n got %v", orig, actual)
		}
	}
}
