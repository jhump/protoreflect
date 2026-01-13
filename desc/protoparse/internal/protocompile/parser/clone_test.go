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
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/ast"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/reporter"
)

func TestClone(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("../internal/testdata/desc_test_complex.proto")
	require.NoError(t, err)
	handler := reporter.NewHandler(nil)
	fileNode, err := Parse("desc_test_complex.proto", bytes.NewReader(data), handler)
	require.NoError(t, err)
	result, err := ResultFromAST(fileNode, true, handler)
	require.NoError(t, err)

	t.Run("known result impl", func(t *testing.T) {
		t.Parallel()
		// With this result, we can clone the proto and rebuild the clone's index
		clonedResult := Clone(result)
		checkClone(t, result, clonedResult, true)
	})
	t.Run("unknown result impl", func(t *testing.T) {
		t.Parallel()
		// With this result, we have to rebuild the proto and index from scratch
		clonedResult := Clone(otherResultImpl{Result: result})
		checkClone(t, result, clonedResult, false)
	})
	t.Run("unknown result impl w/out AST", func(t *testing.T) {
		t.Parallel()
		// With this result, we can just clone the proto (no index since no AST)
		result := ResultWithoutAST(result.FileDescriptorProto())
		clonedResult := Clone(otherResultImpl{Result: result})
		checkClone(t, result, clonedResult, true)
	})
	t.Run("impl w/ custom clone impl", func(t *testing.T) {
		t.Parallel()
		// With this result, we just verify that its Clone method was invoked.
		orig := &customCloneResultImpl{Result: result}
		clonedResult := Clone(orig)
		require.Equal(t, 1, orig.cloneCalled)
		require.Equal(t, orig.clone, clonedResult)
	})
}

func checkClone(t *testing.T, orig, clone Result, isProtoClone bool) {
	t.Helper()
	require.NotSame(t, orig, clone)
	require.NotSame(t, orig.FileDescriptorProto(), clone.FileDescriptorProto())
	if !proto.Equal(orig.FileDescriptorProto(), clone.FileDescriptorProto()) {
		diff := cmp.Diff(orig.FileDescriptorProto(), clone.FileDescriptorProto(), protocmp.Transform())
		require.Empty(t, diff)
		// should not get here :P
		t.Fatal("orig and clone file descriptors are not equal but diff is empty(?!?!)")
	}
	// AST is expected to be equal since it is never mutated by compilation
	require.Same(t, orig.AST(), clone.AST())

	origRes := orig.(*result)   //nolint:errcheck
	cloneRes := clone.(*result) //nolint:errcheck

	if origRes.file == nil {
		require.Empty(t, cloneRes.nodes)
		return
	}

	// If they have ASTs, also check their node indices.
	// First create a reverse index for orig.
	origRevIndex := map[ast.Node][]proto.Message{}
	for msg, node := range origRes.nodes {
		origRevIndex[node] = append(origRevIndex[node], msg)
	}

	// clone index may be bigger due to extension range options (see below for more info)
	assert.GreaterOrEqual(t, len(cloneRes.nodes), len(origRes.nodes))
	cloneRevIndex := map[ast.Node][]proto.Message{}
	for msg, node := range cloneRes.nodes {
		cloneRevIndex[node] = append(cloneRevIndex[node], msg)
	}
	cloneSyntheticMapFields, cloneSyntheticOneofs := 0, 0
	for node, cloneMsgs := range cloneRevIndex {
		origMsgs := origRevIndex[node]
		if isProtoClone {
			// For option nodes and field reference nodes, the original may have 1
			// message but the clone may have >1. This is because when we build a
			// descriptor where the same options apply to multiple extension ranges,
			// we refer to the same uninterpreted options and name part messages in
			// each range. But proto.Clone will create a deep copy at each reference
			// site. So if 4 extension ranges share the same options, we'd see just
			// 1 message in the original and 4 messages in the clone.
			allowMismatch := false
			switch node.(type) {
			case *ast.OptionNode:
				allowMismatch = true
			case *ast.FieldReferenceNode:
				allowMismatch = true
			}
			if allowMismatch && len(origMsgs) == 1 && len(cloneMsgs) > 1 {
				continue
			}
		} else {
			// If we didn't use proto.Clone but instead rebuilt the file descriptor,
			// then we would have synthesized different values for synthetic map field
			// and oneof nodes. So we may have nodes in the clone that have no entries
			// in original.
			allowMismatch := false
			switch node.(type) {
			case *ast.SyntheticMapField:
				cloneSyntheticMapFields++
				allowMismatch = true
			case *ast.FieldReferenceNode:
				cloneSyntheticOneofs++
				allowMismatch = true
			}
			if allowMismatch && len(origMsgs) == 0 {
				continue
			}
		}
		assert.Equal(t, len(origMsgs), len(cloneMsgs), "mismatch for number of messages associated with %T (expect %+v, got %+v)", node, origMsgs, cloneMsgs)
	}
	origSyntheticMapFields, origSyntheticOneofs := 0, 0
	for node, origMsgs := range origRevIndex {
		cloneMsgs := cloneRevIndex[node]
		if !isProtoClone {
			// If we didn't use proto.Clone but instead rebuilt the file descriptor,
			// then we would have synthesized different values for synthetic map field
			// and oneof nodes. So we may have nodes in the original that have no
			// entries in clone.
			allowMismatch := false
			switch node.(type) {
			case *ast.SyntheticMapField:
				origSyntheticMapFields++
				allowMismatch = true
			case *ast.FieldReferenceNode:
				origSyntheticOneofs++
				allowMismatch = true
			}
			if allowMismatch && len(cloneMsgs) == 0 {
				continue
			}
		}
		// we already covered all cases where they are not equal above
		// except cases that are absent from cloneRevIndex
		assert.NotEmpty(t, cloneMsgs, "mismatch for number of messages associated with %T (expect %+v, got %+v)", node, origMsgs, cloneMsgs)
	}

	assert.Equal(t, origSyntheticMapFields, cloneSyntheticMapFields)
	assert.Equal(t, origSyntheticOneofs, cloneSyntheticOneofs)

	// Now we can make sure the index is a deep copy (e.g. clone
	// index does not contain pointers into original).
	for msg := range cloneRes.nodes {
		_, ok := origRes.nodes[msg]
		// messages in clone index should NOT appear in original index
		assert.False(t, ok)
	}
}

type otherResultImpl struct {
	Result
}

type customCloneResultImpl struct {
	Result
	clone       Result
	cloneCalled int
}

func (c *customCloneResultImpl) Clone() Result {
	c.cloneCalled++
	if c.clone == nil {
		c.clone = otherResultImpl{c.Result}
	}
	return c.clone
}
