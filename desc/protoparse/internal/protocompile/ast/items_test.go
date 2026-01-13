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

package ast_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/ast"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/parser"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/reporter"
)

func TestItems(t *testing.T) {
	t.Parallel()
	err := filepath.Walk("../internal/testdata", func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".proto" {
			t.Run(path, func(t *testing.T) {
				t.Parallel()
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				testItemsSequence(t, path, data)
			})
		}
		return nil
	})
	assert.NoError(t, err) //nolint:testifylint // we want to continue even if err!=nil
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		testItemsSequence(t, "empty", []byte(`
		// this file has no lexical elements, just this one comment
		`))
	})
}

func testItemsSequence(t *testing.T, path string, data []byte) {
	filename := filepath.Base(path)
	root, err := parser.Parse(filename, bytes.NewReader(data), reporter.NewHandler(nil))
	require.NoError(t, err)
	tokens := leavesAsSlice(root)
	require.NoError(t, err)
	// Make sure sequence matches the actual leaves in the tree
	seq := root.Items()
	// Both forwards
	item, ok := seq.First()
	require.True(t, ok)
	checkComments := func(comments ast.Comments) {
		for i := range comments.Len() {
			c := comments.Index(i)
			astItem := c.AsItem()
			require.Equal(t, astItem, item)
			infoEqual(t, c, root.ItemInfo(astItem))
			item, _ = seq.Next(item)
		}
	}
	for _, token := range tokens {
		tokInfo := root.TokenInfo(token)
		checkComments(tokInfo.LeadingComments())

		astItem := token.AsItem()
		require.Equal(t, astItem, item)
		infoEqual(t, tokInfo, root.ItemInfo(astItem))
		item, _ = seq.Next(item)

		checkComments(tokInfo.TrailingComments())
	}
	// And backwards
	item, ok = seq.Last()
	require.True(t, ok)
	checkComments = func(comments ast.Comments) {
		for i := comments.Len() - 1; i >= 0; i-- {
			c := comments.Index(i)
			astItem := c.AsItem()
			require.Equal(t, astItem, item)
			infoEqual(t, c, root.ItemInfo(astItem))
			item, _ = seq.Previous(item)
		}
	}
	for i := len(tokens) - 1; i >= 0; i-- {
		token := tokens[i]
		tokInfo := root.TokenInfo(token)
		checkComments(tokInfo.TrailingComments())

		astItem := token.AsItem()
		require.Equal(t, astItem, item)
		infoEqual(t, tokInfo, root.ItemInfo(astItem))
		item, _ = seq.Previous(item)

		checkComments(tokInfo.LeadingComments())
	}
}

func infoEqual(t *testing.T, exp, act ast.ItemInfo) {
	assert.Equal(t, act.RawText(), exp.RawText())
	assert.Equal(t, act.Start(), exp.Start(), "item %q", act.RawText())
	assert.Equal(t, act.End(), exp.End(), "item %q", act.RawText())
	assert.Equal(t, act.LeadingWhitespace(), exp.LeadingWhitespace(), "item %q", act.RawText())
}
