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

package walk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestDescriptorProtosEnterAndExit(t *testing.T) {
	t.Parallel()
	file := protodesc.ToFileDescriptorProto((*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().ParentFile())
	nameStack := []string{file.GetPackage()}
	err := DescriptorProtosEnterAndExit(
		file,
		func(fullName protoreflect.FullName, message proto.Message) error {
			switch d := message.(type) {
			case *descriptorpb.DescriptorProto:
				expected := joinNames(nameStack[len(nameStack)-1], d.GetName())
				assert.Equal(t, expected, string(fullName))
			case *descriptorpb.FieldDescriptorProto:
				expected := joinNames(nameStack[len(nameStack)-1], d.GetName())
				assert.Equal(t, expected, string(fullName))
			case *descriptorpb.OneofDescriptorProto:
				expected := joinNames(nameStack[len(nameStack)-1], d.GetName())
				assert.Equal(t, expected, string(fullName))
			case *descriptorpb.EnumDescriptorProto:
				expected := joinNames(nameStack[len(nameStack)-1], d.GetName())
				assert.Equal(t, expected, string(fullName))
			case *descriptorpb.EnumValueDescriptorProto:
				// we look at the NEXT to last item on stack because enums are
				// defined not in the enum but in its enclosing scope
				expected := joinNames(nameStack[len(nameStack)-2], d.GetName())
				assert.Equal(t, expected, string(fullName))
			case *descriptorpb.ServiceDescriptorProto:
				expected := joinNames(nameStack[len(nameStack)-1], d.GetName())
				assert.Equal(t, expected, string(fullName))
			case *descriptorpb.MethodDescriptorProto:
				expected := joinNames(nameStack[len(nameStack)-1], d.GetName())
				assert.Equal(t, expected, string(fullName))
			default:
				t.Fatalf("unknown descriptor type: %T", d)
			}
			nameStack = append(nameStack, string(fullName))
			return nil
		},
		func(_ protoreflect.FullName, _ proto.Message) error {
			nameStack = nameStack[:len(nameStack)-1]
			return nil
		},
	)
	require.NoError(t, err)
}

func joinNames(prefix, name string) string {
	if len(prefix) == 0 {
		return name
	}
	return prefix + "." + name
}
