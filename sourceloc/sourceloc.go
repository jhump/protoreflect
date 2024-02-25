// Package sourcelocation contains some helpers for interacting with
// protoreflect.SourceLocation and protoreflect.SourcePath values.
package sourceloc

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal"
)

// IsZero returns true if loc is a zero-value SourceLocation.
func IsZero(loc protoreflect.SourceLocation) bool {
	return loc.Path == nil &&
		loc.StartLine == 0 && loc.StartColumn == 0 &&
		loc.EndLine == 0 && loc.EndColumn == 0 &&
		loc.LeadingComments == "" && loc.LeadingDetachedComments == nil &&
		loc.TrailingComments == ""
}

// PathsEqual returns true if a and b represent the same source path.
func PathsEqual(a, b protoreflect.SourcePath) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// IsSubpathOf returns true if the given candidate is a subpath of the given path.
func IsSubpathOf(candidate, path protoreflect.SourcePath) bool {
	return len(candidate) >= len(path) && PathsEqual(candidate[:len(path)], path)
}

// PathFor computes the source path for the given descriptor. It returns nil if
// the given descriptor is invalid and a path cannot be computed.
func PathFor(desc protoreflect.Descriptor) protoreflect.SourcePath {
	// we construct the path leaves up, so it is in reverse order
	// from how it needs to finally look (which is from root down)
	path := make(protoreflect.SourcePath, 0, 8)
	for {
		switch desc.(type) {
		case protoreflect.FileDescriptor:
			// Reverse the path since it was constructed in reverse.
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}
			return path
		case protoreflect.MessageDescriptor:
			path = append(path, int32(desc.Index()))
			desc = desc.Parent()
			switch desc.(type) {
			case protoreflect.FileDescriptor:
				path = append(path, int32(internal.FileMessagesTag))
			case protoreflect.MessageDescriptor:
				path = append(path, int32(internal.MessageNestedMessagesTag))
			default:
				return nil
			}
		case protoreflect.FieldDescriptor:
			isExtension := desc.(protoreflect.FieldDescriptor).IsExtension()
			path = append(path, int32(desc.Index()))
			desc = desc.Parent()
			if isExtension {
				switch desc.(type) {
				case protoreflect.FileDescriptor:
					path = append(path, int32(internal.FileExtensionsTag))
				case protoreflect.MessageDescriptor:
					path = append(path, int32(internal.MessageExtensionsTag))
				default:
					return nil
				}
			} else {
				switch desc.(type) {
				case protoreflect.MessageDescriptor:
					path = append(path, int32(internal.MessageFieldsTag))
				default:
					return nil
				}
			}
		case protoreflect.OneofDescriptor:
			path = append(path, int32(desc.Index()))
			desc = desc.Parent()
			switch desc.(type) {
			case protoreflect.MessageDescriptor:
				path = append(path, int32(internal.MessageOneofsTag))
			default:
				return nil
			}
		case protoreflect.EnumDescriptor:
			path = append(path, int32(desc.Index()))
			desc = desc.Parent()
			switch desc.(type) {
			case protoreflect.FileDescriptor:
				path = append(path, int32(internal.FileEnumsTag))
			case protoreflect.MessageDescriptor:
				path = append(path, int32(internal.MessageEnumsTag))
			default:
				return nil
			}
		case protoreflect.EnumValueDescriptor:
			path = append(path, int32(desc.Index()))
			desc = desc.Parent()
			switch desc.(type) {
			case protoreflect.EnumDescriptor:
				path = append(path, int32(internal.EnumValuesTag))
			default:
				return nil
			}
		case protoreflect.ServiceDescriptor:
			path = append(path, int32(desc.Index()))
			desc = desc.Parent()
			switch desc.(type) {
			case protoreflect.FileDescriptor:
				path = append(path, int32(internal.FileServicesTag))
			default:
				return nil
			}
		case protoreflect.MethodDescriptor:
			path = append(path, int32(desc.Index()))
			desc = desc.Parent()
			switch desc.(type) {
			case protoreflect.ServiceDescriptor:
				path = append(path, int32(internal.ServiceMethodsTag))
			default:
				return nil
			}
		default:
			return nil
		}
	}
}
