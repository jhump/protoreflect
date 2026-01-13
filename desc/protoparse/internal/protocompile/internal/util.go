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

package internal

import (
	"bytes"
	"strings"
	"unicode"
	"unicode/utf8"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/bufbuild/protocompile/internal/cases"
)

// JSONName returns the default JSON name for a field with the given name.
// This mirrors the algorithm in protoc:
//
//	https://github.com/protocolbuffers/protobuf/blob/v21.3/src/google/protobuf/descriptor.cc#L95
func JSONName(name string) string {
	return cases.Converter{
		Case:        cases.Camel,
		NaiveSplit:  true,
		NoLowercase: true,
	}.Convert(name)
}

// MapEntry returns the map entry name for a field with the given name.
// This mirrors the algorithm in protoc:
//
//	https://github.com/protocolbuffers/protobuf/blob/v21.3/src/google/protobuf/descriptor.cc#L95
func MapEntry(name string) string {
	buf := new(strings.Builder)
	cases.Converter{
		Case:        cases.Pascal,
		NaiveSplit:  true,
		NoLowercase: true,
	}.Append(buf, name)
	_, _ = buf.WriteString("Entry")
	return buf.String()
}

// TrimPrefix is used to remove the given prefix from the given str. It does not require
// an exact match and ignores case and underscores. If the all non-underscore characters
// would be removed from str, str is returned unchanged. If str does not have the given
// prefix (even with the very lenient matching, in regard to case and underscores), then
// str is returned unchanged.
//
// The algorithm is adapted from the protoc source:
//
//	https://github.com/protocolbuffers/protobuf/blob/v21.3/src/google/protobuf/descriptor.cc#L922
func TrimPrefix(str, prefix string) string {
	j := 0
	for i, r := range str {
		if r == '_' {
			// skip underscores in the input
			continue
		}

		p, sz := utf8.DecodeRuneInString(prefix[j:])
		for p == '_' {
			j += sz // consume/skip underscore
			p, sz = utf8.DecodeRuneInString(prefix[j:])
		}

		if j == len(prefix) {
			// matched entire prefix; return rest of str
			// but skipping any leading underscores
			result := strings.TrimLeft(str[i:], "_")
			if len(result) == 0 {
				// result can't be empty string
				return str
			}
			return result
		}
		if unicode.ToLower(r) != unicode.ToLower(p) {
			// does not match prefix
			return str
		}
		j += sz // consume matched rune of prefix
	}
	return str
}

// CreatePrefixList returns a list of package prefixes to search when resolving
// a symbol name. If the given package is blank, it returns only the empty
// string. If the given package contains only one token, e.g. "foo", it returns
// that token and the empty string, e.g. ["foo", ""]. Otherwise, it returns
// successively shorter prefixes of the package and then the empty string. For
// example, for a package named "foo.bar.baz" it will return the following list:
//
//	["foo.bar.baz", "foo.bar", "foo", ""]
func CreatePrefixList(pkg string) []string {
	if pkg == "" {
		return []string{""}
	}

	numDots := 0
	// one pass to pre-allocate the returned slice
	for i := range len(pkg) {
		if pkg[i] == '.' {
			numDots++
		}
	}
	if numDots == 0 {
		return []string{pkg, ""}
	}

	prefixes := make([]string, numDots+2)
	// second pass to fill in returned slice
	for i := range len(pkg) {
		if pkg[i] == '.' {
			prefixes[numDots] = pkg[:i]
			numDots--
		}
	}
	prefixes[0] = pkg

	return prefixes
}

func WriteEscapedBytes(buf *bytes.Buffer, b []byte) {
	// This uses the same algorithm as the protoc C++ code for escaping strings.
	// The protoc C++ code in turn uses the abseil C++ library's CEscape function:
	//  https://github.com/abseil/abseil-cpp/blob/934f613818ffcb26c942dff4a80be9a4031c662c/absl/strings/escaping.cc#L406
	for _, c := range b {
		switch c {
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case '"':
			buf.WriteString("\\\"")
		case '\'':
			buf.WriteString("\\'")
		case '\\':
			buf.WriteString("\\\\")
		default:
			if c >= 0x20 && c < 0x7f {
				// simple printable characters
				buf.WriteByte(c)
			} else {
				// use octal escape for all other values
				buf.WriteRune('\\')
				buf.WriteByte('0' + ((c >> 6) & 0x7))
				buf.WriteByte('0' + ((c >> 3) & 0x7))
				buf.WriteByte('0' + (c & 0x7))
			}
		}
	}
}

// IsZeroLocation returns true if the given loc is a zero value
// (which is returned from queries that have no result).
func IsZeroLocation(loc protoreflect.SourceLocation) bool {
	return loc.Path == nil &&
		loc.StartLine == 0 &&
		loc.StartColumn == 0 &&
		loc.EndLine == 0 &&
		loc.EndColumn == 0 &&
		loc.LeadingDetachedComments == nil &&
		loc.LeadingComments == "" &&
		loc.TrailingComments == "" &&
		loc.Next == 0
}

// ComputePath computes the source location path for the given descriptor.
// The boolean value indicates whether the result is valid. If the path
// cannot be computed for d, the function returns nil, false.
func ComputePath(d protoreflect.Descriptor) (protoreflect.SourcePath, bool) {
	_, ok := d.(protoreflect.FileDescriptor)
	if ok {
		return nil, true
	}
	var path protoreflect.SourcePath
	for {
		p := d.Parent()
		switch d := d.(type) {
		case protoreflect.FileDescriptor:
			return reverse(path), true
		case protoreflect.MessageDescriptor:
			path = append(path, int32(d.Index()))
			switch p.(type) {
			case protoreflect.FileDescriptor:
				path = append(path, FileMessagesTag)
			case protoreflect.MessageDescriptor:
				path = append(path, MessageNestedMessagesTag)
			default:
				return nil, false
			}
		case protoreflect.FieldDescriptor:
			path = append(path, int32(d.Index()))
			switch p.(type) {
			case protoreflect.FileDescriptor:
				if d.IsExtension() {
					path = append(path, FileExtensionsTag)
				} else {
					return nil, false
				}
			case protoreflect.MessageDescriptor:
				if d.IsExtension() {
					path = append(path, MessageExtensionsTag)
				} else {
					path = append(path, MessageFieldsTag)
				}
			default:
				return nil, false
			}
		case protoreflect.OneofDescriptor:
			path = append(path, int32(d.Index()))
			if _, ok := p.(protoreflect.MessageDescriptor); ok {
				path = append(path, MessageOneofsTag)
			} else {
				return nil, false
			}
		case protoreflect.EnumDescriptor:
			path = append(path, int32(d.Index()))
			switch p.(type) {
			case protoreflect.FileDescriptor:
				path = append(path, FileEnumsTag)
			case protoreflect.MessageDescriptor:
				path = append(path, MessageEnumsTag)
			default:
				return nil, false
			}
		case protoreflect.EnumValueDescriptor:
			path = append(path, int32(d.Index()))
			if _, ok := p.(protoreflect.EnumDescriptor); ok {
				path = append(path, EnumValuesTag)
			} else {
				return nil, false
			}
		case protoreflect.ServiceDescriptor:
			path = append(path, int32(d.Index()))
			if _, ok := p.(protoreflect.FileDescriptor); ok {
				path = append(path, FileServicesTag)
			} else {
				return nil, false
			}
		case protoreflect.MethodDescriptor:
			path = append(path, int32(d.Index()))
			if _, ok := p.(protoreflect.ServiceDescriptor); ok {
				path = append(path, ServiceMethodsTag)
			} else {
				return nil, false
			}
		}
		d = p
	}
}

// CanPack returns true if a repeated field of the given kind
// can use packed encoding.
func CanPack(k protoreflect.Kind) bool {
	switch k {
	case protoreflect.MessageKind, protoreflect.GroupKind, protoreflect.StringKind, protoreflect.BytesKind:
		return false
	default:
		return true
	}
}

func ClonePath(path protoreflect.SourcePath) protoreflect.SourcePath {
	clone := make(protoreflect.SourcePath, len(path))
	copy(clone, path)
	return clone
}

func reverse(p protoreflect.SourcePath) protoreflect.SourcePath {
	for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
		p[i], p[j] = p[j], p[i]
	}
	return p
}
