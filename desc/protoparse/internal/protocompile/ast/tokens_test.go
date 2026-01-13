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

func TestTokens(t *testing.T) {
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
				testTokensSequence(t, path, data)
			})
		}
		return nil
	})
	assert.NoError(t, err) //nolint:testifylint // we want to continue even if err!=nil
	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		testTokensSequence(t, "empty", []byte(`
		// this file has no lexical elements, just this one comment
		`))
	})
}

func testTokensSequence(t *testing.T, path string, data []byte) {
	filename := filepath.Base(path)
	root, err := parser.Parse(filename, bytes.NewReader(data), reporter.NewHandler(nil))
	require.NoError(t, err)
	tokens := leavesAsSlice(root)
	require.NoError(t, err)
	// Make sure sequence matches the actual leaves in the tree
	seq := root.Tokens()
	// Both forwards
	token, ok := seq.First()
	require.True(t, ok)
	for _, astToken := range tokens {
		require.Equal(t, astToken, token)
		token, _ = seq.Next(token)
	}
	// And backwards
	token, ok = seq.Last()
	require.True(t, ok)
	for i := len(tokens) - 1; i >= 0; i-- {
		astToken := tokens[i]
		require.Equal(t, astToken, token)
		token, _ = seq.Previous(token)
	}
}

func leavesAsSlice(file *ast.FileNode) []ast.Token {
	var tokens []ast.Token
	_ = ast.Walk(file, &ast.SimpleVisitor{
		DoVisitTerminalNode: func(n ast.TerminalNode) error {
			tokens = append(tokens, n.Token())
			return nil
		},
	})
	return tokens
}
