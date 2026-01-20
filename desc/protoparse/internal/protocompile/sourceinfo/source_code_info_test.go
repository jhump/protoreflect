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

package sourceinfo_test

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/internal/prototest"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/linker"
	"github.com/jhump/protoreflect/desc/protoparse/internal/protocompile/protoutil"
)

func TestSourceCodeInfo(t *testing.T) {
	t.Parallel()
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testdata"},
		}),
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	fds, err := compiler.Compile(t.Context(), "desc_test_comments.proto", "desc_test_complex.proto")
	if pe, ok := err.(protocompile.PanicError); ok {
		t.Fatalf("panic! %v\n%v", pe, pe.Stack)
	}
	require.NoError(t, err)
	// also test that imported files have source code info
	// (desc_test_comments.proto imports desc_test_options.proto)
	importedFd := fds[0].FindImportByPath("desc_test_options.proto")
	require.NotNil(t, importedFd)

	fdset := prototest.LoadDescriptorSet(t, "../internal/testdata/source_info.protoset", linker.ResolverFromFile(fds[0]))
	actualFdset := &descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protoutil.ProtoFromFileDescriptor(importedFd),
			protoutil.ProtoFromFileDescriptor(fds[0]),
			protoutil.ProtoFromFileDescriptor(fds[1]),
		},
	}

	for _, actualFd := range actualFdset.File {
		var expectedFd *descriptorpb.FileDescriptorProto
		for _, fd := range fdset.File {
			if fd.GetName() == actualFd.GetName() {
				expectedFd = fd
				break
			}
		}
		if !assert.NotNil(t, expectedFd, "file %q not found in source_info.protoset", actualFd.GetName()) {
			continue
		}
		fixupProtocSourceCodeInfo(expectedFd.SourceCodeInfo)
		prototest.AssertMessagesEqual(t, expectedFd.SourceCodeInfo, actualFd.SourceCodeInfo, expectedFd.GetName())
	}
}

var protocFixers = []struct {
	pathPatterns []*regexp.Regexp
	fixer        func(allLocs []*descriptorpb.SourceCodeInfo_Location, currentIndex int) *descriptorpb.SourceCodeInfo_Location
}{
	{
		// FieldDescriptorProto.default_value
		// https://github.com/protocolbuffers/protobuf/issues/10478
		pathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^4,\d+,(?:3,\d+,)*2,\d+,7$`), // normal fields
			regexp.MustCompile(`^7,\d+,7$`),                  // extension fields, top-level in file
			regexp.MustCompile(`^4,\d+,(?:3,\d+,)*7,\d+,7$`), // extension fields, nested in a message
		},
		fixer: func(allLocs []*descriptorpb.SourceCodeInfo_Location, currentIndex int) *descriptorpb.SourceCodeInfo_Location {
			// adjust span to include preceding "default = "
			allLocs[currentIndex].Span[1] -= 10
			return allLocs[currentIndex]
		},
	},
	{
		// FieldDescriptorProto.json_name
		// https://github.com/protocolbuffers/protobuf/issues/10478
		pathPatterns: []*regexp.Regexp{
			regexp.MustCompile(`^4,\d+,(?:3,\d+,)*2,\d+,10$`),
		},
		fixer: func(allLocs []*descriptorpb.SourceCodeInfo_Location, currentIndex int) *descriptorpb.SourceCodeInfo_Location {
			if currentIndex > 0 && pathsEqual(allLocs[currentIndex].Path, allLocs[currentIndex-1].Path) {
				// second span for json_name is not useful; remove it
				return nil
			}
			return allLocs[currentIndex]
		},
	},
}

func pathsEqual(a, b []int32) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if b[i] != a[i] {
			return false
		}
	}
	return true
}

func fixupProtocSourceCodeInfo(info *descriptorpb.SourceCodeInfo) {
	for i := 0; i < len(info.Location); i++ {
		loc := info.Location[i]

		pathStrs := make([]string, len(loc.Path))
		for j, val := range loc.Path {
			pathStrs[j] = strconv.FormatInt(int64(val), 10)
		}
		pathStr := strings.Join(pathStrs, ",")

		for _, fixerEntry := range protocFixers {
			match := false
			for _, pattern := range fixerEntry.pathPatterns {
				if pattern.MatchString(pathStr) {
					match = true
					break
				}
			}
			if !match {
				continue
			}
			newLoc := fixerEntry.fixer(info.Location, i)
			if newLoc == nil {
				// remove this entry
				info.Location = append(info.Location[:i], info.Location[i+1:]...)
				i--
			} else {
				info.Location[i] = newLoc
			}
			// only apply one fixer to each location
			break
		}
	}
}

func TestSourceCodeInfoOptions(t *testing.T) {
	t.Parallel()

	regenerateGoldenOutputFile := os.Getenv("PROTOCOMPILE_REFRESH") != ""

	generateSourceInfoText := func(t *testing.T, filename string, mode protocompile.SourceInfoMode) string {
		t.Helper()
		compiler := protocompile.Compiler{
			Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
				ImportPaths: []string{"../internal/testdata"},
			}),
			SourceInfoMode: mode,
		}
		fds, err := compiler.Compile(t.Context(), filename)
		if pe, ok := err.(protocompile.PanicError); ok {
			t.Fatalf("panic! %v\n%v", pe, pe.Stack)
		}
		require.NoError(t, err)

		file, err := linker.NewFileRecursive(fds[0])
		require.NoError(t, err)
		resolver := linker.ResolverFromFile(file)
		return describeSourceCodeInfo(file.Path(), file.SourceLocations(), resolver)
	}

	testCases := []struct {
		name     string
		filename string
		mode     protocompile.SourceInfoMode
	}{
		{
			name:     "extra_comments",
			filename: "desc_test_comments.proto",
			mode:     protocompile.SourceInfoExtraComments,
		},
		{
			name:     "extra_option_locations",
			filename: "desc_test_complex.proto",
			mode:     protocompile.SourceInfoExtraOptionLocations,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			output := generateSourceInfoText(t, testCase.filename, testCase.mode)

			baseName := strings.TrimSuffix(testCase.filename, ".proto")
			if regenerateGoldenOutputFile {
				err := os.WriteFile(fmt.Sprintf("testdata/%s.%s.txt", baseName, testCase.name), []byte(output), 0644)
				require.NoError(t, err)
				// also create a file with standard comments, as a useful demonstration of the differences
				output := generateSourceInfoText(t, testCase.filename, protocompile.SourceInfoStandard)
				err = os.WriteFile(fmt.Sprintf("testdata/%s.standard.txt", baseName), []byte(output), 0644)
				require.NoError(t, err)
				return
			}

			goldenOutput, err := os.ReadFile(fmt.Sprintf("testdata/%s.%s.txt", baseName, testCase.name))
			require.NoError(t, err)
			diff := cmp.Diff(string(goldenOutput), output)
			assert.Empty(t, diff, "source code info mismatch (-want +got):\n%v", diff)
		})
	}
}

var pathRoot = (&descriptorpb.FileDescriptorProto{}).ProtoReflect().Descriptor()

func describeSourceCodeInfo(fileName string, locs protoreflect.SourceLocations, resolver linker.Resolver) string {
	var buf bytes.Buffer
	for i := range locs.Len() {
		if i > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(fileName)
		describeLocation(&buf, locs.Get(i), resolver)
	}
	return buf.String()
}

func describeLocation(buf *bytes.Buffer, loc protoreflect.SourceLocation, resolver linker.Resolver) {
	describePath(buf, loc.Path, pathRoot, resolver)
	_, _ = fmt.Fprintf(buf, "   Span: %d:%d -> %d:%d\n",
		loc.StartLine+1, loc.StartColumn+1, loc.EndLine+1, loc.EndColumn+1)
	if len(loc.LeadingDetachedComments) > 0 {
		_, _ = fmt.Fprintf(buf, "   Detached Comments:\n")
		for i, cmt := range loc.LeadingDetachedComments {
			if i > 0 {
				buf.WriteString("\n")
			}
			cmt = strings.TrimSuffix(cmt, "\n")
			_, _ = fmt.Fprintf(buf, "%s\n", cmt)
		}
	}
	if loc.LeadingComments != "" {
		cmt := strings.TrimSuffix(loc.LeadingComments, "\n")
		_, _ = fmt.Fprintf(buf, "   Leading Comments:\n%s\n", cmt)
	}
	if loc.TrailingComments != "" {
		cmt := strings.TrimSuffix(loc.TrailingComments, "\n")
		_, _ = fmt.Fprintf(buf, "   Trailing Comments:\n%s\n", cmt)
	}
}

func describePath(buf *bytes.Buffer, path protoreflect.SourcePath, md protoreflect.MessageDescriptor, resolver linker.Resolver) {
	if len(path) == 0 {
		buf.WriteString(":\n")
		return
	}

	fieldNumber := protoreflect.FieldNumber(path[0])
	path = path[1:]
	var next protoreflect.MessageDescriptor
	fd := resolveNumber(fieldNumber, md, resolver)
	if fd == nil {
		_, _ = fmt.Fprintf(buf, " > %d?", fieldNumber)
	} else {
		if fd.IsExtension() {
			_, _ = fmt.Fprintf(buf, " > (%s)", fd.FullName())
		} else {
			_, _ = fmt.Fprintf(buf, " > %s", fd.Name())
		}
		if fd.Cardinality() == protoreflect.Repeated && len(path) > 0 {
			index := path[0]
			path = path[1:]
			_, _ = fmt.Fprintf(buf, "[%d]", index)
		}
		next = fd.Message()
	}
	describePath(buf, path, next, resolver)
}

func resolveNumber(num protoreflect.FieldNumber, md protoreflect.MessageDescriptor, resolver linker.Resolver) protoreflect.FieldDescriptor {
	if md == nil {
		return nil
	}
	fld := md.Fields().ByNumber(num)
	if fld != nil {
		return fld
	}
	xt, err := resolver.FindExtensionByNumber(md.FullName(), num)
	if err != nil {
		return nil
	}
	return xt.TypeDescriptor()
}
