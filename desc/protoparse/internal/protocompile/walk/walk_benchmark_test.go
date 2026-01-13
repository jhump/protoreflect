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

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func BenchmarkDescriptors(b *testing.B) {
	file := (*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().ParentFile()
	for range b.N {
		err := Descriptors(file, func(_ protoreflect.Descriptor) error {
			return nil
		})
		require.NoError(b, err)
	}
}

func BenchmarkDescriptorProtos(b *testing.B) {
	file := protodesc.ToFileDescriptorProto((*descriptorpb.FileDescriptorProto)(nil).ProtoReflect().Descriptor().ParentFile())
	for range b.N {
		err := DescriptorProtos(file, func(_ protoreflect.FullName, _ proto.Message) error {
			return nil
		})
		require.NoError(b, err)
	}
}
