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

package linker

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/ast"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/parser"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/reporter"
)

func TestSymbolsPackages(t *testing.T) {
	t.Parallel()

	var s Symbols
	// default/nameless package is the root
	assert.Equal(t, &s.pkgTrie, s.getPackage("", true))

	h := reporter.NewHandler(nil)
	span := ast.UnknownSpan("foo.proto")
	pkg, err := s.importPackages(span, "build.buf.foo.bar.baz", h)
	require.NoError(t, err)
	// new package has nothing in it
	assert.Empty(t, pkg.children)
	assert.Empty(t, pkg.files)
	assert.Empty(t, pkg.symbols)
	assert.Empty(t, pkg.exts)

	assert.Equal(t, pkg, s.getPackage("build.buf.foo.bar.baz", true))

	// verify that trie was created correctly:
	//   each package has just one entry, which is its immediate sub-package
	cur := &s.pkgTrie
	pkgNames := []protoreflect.FullName{"build", "build.buf", "build.buf.foo", "build.buf.foo.bar", "build.buf.foo.bar.baz"}
	for _, pkgName := range pkgNames {
		assert.Len(t, cur.children, 1)
		assert.Empty(t, cur.files)
		assert.Len(t, cur.symbols, 1)
		assert.Empty(t, cur.exts)

		entry, ok := cur.symbols[pkgName]
		require.True(t, ok)
		assert.Equal(t, span.Start(), entry.span.Start())
		assert.Equal(t, span.End(), entry.span.End())
		assert.False(t, entry.isEnumValue)
		assert.True(t, entry.isPackage)

		next, ok := cur.children[pkgName]
		require.True(t, ok)
		require.NotNil(t, next)

		cur = next
	}
	assert.Equal(t, pkg, cur)
}

func TestSymbolsImport(t *testing.T) {
	t.Parallel()

	fileAsResult := parseAndLink(t, `
		syntax = "proto2";
		import "google/protobuf/descriptor.proto";
		package foo.bar;
		message Foo {
			optional string bar = 1;
			repeated int32 baz = 2;
			extensions 10 to 20;
		}
		extend Foo {
			optional float f = 10;
			optional string s = 11;
		}
		extend google.protobuf.FieldOptions {
			optional bytes xtra = 20000;
		}
		`)

	fileAsNonResult, err := protodesc.NewFile(fileAsResult.FileDescriptorProto(), protoregistry.GlobalFiles)
	require.NoError(t, err)

	h := reporter.NewHandler(nil)
	testCases := map[string]protoreflect.FileDescriptor{
		"linker.Result":               fileAsResult,
		"protoreflect.FileDescriptor": fileAsNonResult,
	}

	for name, fd := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var s Symbols
			err := s.Import(fd, h)
			require.NoError(t, err)

			// verify contents of s

			pkg := s.getPackage("foo.bar", true)
			syms := pkg.symbols
			assert.Len(t, syms, 6)
			assert.Contains(t, syms, protoreflect.FullName("foo.bar.Foo"))
			assert.Contains(t, syms, protoreflect.FullName("foo.bar.Foo.bar"))
			assert.Contains(t, syms, protoreflect.FullName("foo.bar.Foo.baz"))
			assert.Contains(t, syms, protoreflect.FullName("foo.bar.f"))
			assert.Contains(t, syms, protoreflect.FullName("foo.bar.s"))
			assert.Contains(t, syms, protoreflect.FullName("foo.bar.xtra"))
			exts := pkg.exts
			assert.Len(t, exts, 2)
			assert.Contains(t, exts, extNumber{"foo.bar.Foo", 10})
			assert.Contains(t, exts, extNumber{"foo.bar.Foo", 11})

			pkg = s.getPackage("google.protobuf", true)
			exts = pkg.exts
			assert.Len(t, exts, 1)
			assert.Contains(t, exts, extNumber{"google.protobuf.FieldOptions", 20000})
		})
	}
}

func TestSymbolExtensions(t *testing.T) {
	t.Parallel()

	var s Symbols

	_, err := s.importPackages(ast.UnknownSpan("foo.proto"), "foo.bar", reporter.NewHandler(nil))
	require.NoError(t, err)
	_, err = s.importPackages(ast.UnknownSpan("google/protobuf/descriptor.proto"), "google.protobuf", reporter.NewHandler(nil))
	require.NoError(t, err)

	addExt := func(pkg, extendee protoreflect.FullName, num protoreflect.FieldNumber) error {
		return s.AddExtension(pkg, extendee, num, ast.UnknownSpan("foo.proto"), reporter.NewHandler(nil))
	}

	t.Run("mismatch", func(t *testing.T) {
		t.Parallel()
		err := addExt("bar.baz", "foo.bar.Foo", 11)
		require.ErrorContains(t, err, "does not match package")
	})
	t.Run("missing package", func(t *testing.T) {
		t.Parallel()
		err := addExt("bar.baz", "bar.baz.Bar", 11)
		require.ErrorContains(t, err, "missing package symbols")
	})

	err = addExt("foo.bar", "foo.bar.Foo", 11)
	require.NoError(t, err)
	err = addExt("foo.bar", "foo.bar.Foo", 12)
	require.NoError(t, err)

	err = addExt("foo.bar", "foo.bar.Foo", 11)
	require.ErrorContains(t, err, "already defined")

	err = addExt("google.protobuf", "google.protobuf.FileOptions", 10101)
	require.NoError(t, err)
	err = addExt("google.protobuf", "google.protobuf.FieldOptions", 10101)
	require.NoError(t, err)
	err = addExt("google.protobuf", "google.protobuf.MessageOptions", 10101)
	require.NoError(t, err)

	// verify contents of s

	pkg := s.getPackage("foo.bar", true)
	exts := pkg.exts
	assert.Len(t, exts, 2)
	assert.Contains(t, exts, extNumber{"foo.bar.Foo", 11})
	assert.Contains(t, exts, extNumber{"foo.bar.Foo", 12})

	pkg = s.getPackage("google.protobuf", true)
	exts = pkg.exts
	assert.Len(t, exts, 3)
	assert.Contains(t, exts, extNumber{"google.protobuf.FileOptions", 10101})
	assert.Contains(t, exts, extNumber{"google.protobuf.FieldOptions", 10101})
	assert.Contains(t, exts, extNumber{"google.protobuf.MessageOptions", 10101})
}

func parseAndLink(t *testing.T, contents string) Result {
	t.Helper()
	h := reporter.NewHandler(nil)
	fileAst, err := parser.Parse("test.proto", strings.NewReader(contents), h)
	require.NoError(t, err)
	parseResult, err := parser.ResultFromAST(fileAst, true, h)
	require.NoError(t, err)
	dep, err := protoregistry.GlobalFiles.FindFileByPath("google/protobuf/descriptor.proto")
	require.NoError(t, err)
	depAsFile, err := NewFileRecursive(dep)
	require.NoError(t, err)
	depFiles := Files{depAsFile}
	linkResult, err := Link(parseResult, depFiles, nil, h)
	require.NoError(t, err)
	return linkResult
}
