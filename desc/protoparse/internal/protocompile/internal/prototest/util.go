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

package prototest

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/linker"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/protoutil"
)

func LoadDescriptorSet(t *testing.T, path string, res linker.Resolver) *descriptorpb.FileDescriptorSet {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var fdset descriptorpb.FileDescriptorSet
	err = proto.UnmarshalOptions{Resolver: res}.Unmarshal(data, &fdset)
	require.NoError(t, err)
	return &fdset
}

func CheckFiles(t *testing.T, act protoreflect.FileDescriptor, expSet *descriptorpb.FileDescriptorSet, recursive bool) bool {
	t.Helper()
	return checkFiles(t, act, expSet, recursive, map[string]struct{}{})
}

func checkFiles(t *testing.T, act protoreflect.FileDescriptor, expSet *descriptorpb.FileDescriptorSet, recursive bool, checked map[string]struct{}) bool {
	if _, ok := checked[act.Path()]; ok {
		// already checked
		return true
	}
	checked[act.Path()] = struct{}{}

	expProto := findFileInSet(expSet, act.Path())
	actProto := protoutil.ProtoFromFileDescriptor(act)
	ret := AssertMessagesEqual(t, expProto, actProto, expProto.GetName())

	if recursive {
		for i := range act.Imports().Len() {
			if !checkFiles(t, act.Imports().Get(i), expSet, true, checked) {
				ret = false
			}
		}
	}

	return ret
}

func findFileInSet(fps *descriptorpb.FileDescriptorSet, name string) *descriptorpb.FileDescriptorProto {
	files := fps.File
	for _, fd := range files {
		if fd.GetName() == name {
			return fd
		}
	}
	return nil
}

func AssertMessagesEqual(t *testing.T, exp, act proto.Message, description string) bool {
	t.Helper()
	if diff := cmp.Diff(exp, act, protocmp.Transform(), cmpopts.EquateNaNs()); diff != "" {
		t.Errorf("%s: message mismatch (-want, +got):\n%s", description, diff)
		return false
	}
	return true
}
