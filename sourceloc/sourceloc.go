// Package sourceloc contains some helpers for interacting with
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

// IsSubpathOf returns true if the given candidate is a subpath of the given path.
func IsSubpathOf(candidate, path protoreflect.SourcePath) bool {
	return len(candidate) >= len(path) && candidate[:len(path)].Equal(path)
}

// IsSubspanOf returns true if the given candidate is a subspan (i.e. contained
// within) the given loc.
func IsSubspanOf(candidate, loc protoreflect.SourceLocation) bool {
	return (candidate.StartLine > loc.StartLine ||
		candidate.StartLine == loc.StartLine && candidate.StartColumn >= loc.StartColumn) &&
		(candidate.EndLine < loc.EndLine ||
			candidate.EndLine == loc.EndLine && candidate.EndColumn <= loc.EndColumn)
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

// ForExtendBlock returns the source code location of the "extend" block that
// contains the given extension.
func ForExtendBlock(ext protoreflect.ExtensionDescriptor) protoreflect.SourceLocation {
	if !ext.IsExtension() {
		return protoreflect.SourceLocation{}
	}
	var path protoreflect.SourcePath
	switch parent := ext.Parent().(type) {
	case protoreflect.FileDescriptor:
		path = protoreflect.SourcePath{internal.FileExtensionsTag, int32(ext.Index())}
	case protoreflect.MessageDescriptor:
		path = append(PathFor(parent), internal.MessageExtensionsTag, int32(ext.Index()))
	default:
		return protoreflect.SourceLocation{}
	}
	return enclosingParent(ext.ParentFile().SourceLocations(), path)
}

// ForReservedNamesStatement returns the source code location of the "reserved"
// statement that contains the reserved name at the given index. This
// corresponds to msg.ReservedNames().Get(reservedNameIndex).
func ForReservedNamesStatement(msg protoreflect.MessageDescriptor, reservedNameIndex int) protoreflect.SourceLocation {
	if reservedNameIndex >= msg.ReservedNames().Len() {
		return protoreflect.SourceLocation{}
	}
	path := append(PathFor(msg), internal.MessageReservedNameTag, int32(reservedNameIndex))
	return enclosingParent(msg.ParentFile().SourceLocations(), path)
}

// ForReservedRangesStatement returns the source code location of the "reserved"
// statement that contains the reserved range at the given index. This
// corresponds to msg.ReservedRanges().Get(reservedRangeIndex).
func ForReservedRangesStatement(msg protoreflect.MessageDescriptor, reservedRangeIndex int) protoreflect.SourceLocation {
	if reservedRangeIndex >= msg.ReservedRanges().Len() {
		return protoreflect.SourceLocation{}
	}
	path := append(PathFor(msg), internal.MessageReservedRangeTag, int32(reservedRangeIndex))
	return enclosingParent(msg.ParentFile().SourceLocations(), path)
}

// ForExtensionsStatement returns the source code location of the "extensions"
// statement that defines the extension range at the given index. This
// corresponds to msg.ExtensionRanges().Get(extensionRangeIndex).
func ForExtensionsStatement(msg protoreflect.MessageDescriptor, extensionRangeIndex int) protoreflect.SourceLocation {
	if extensionRangeIndex >= msg.ExtensionRanges().Len() {
		return protoreflect.SourceLocation{}
	}
	path := append(PathFor(msg), internal.MessageExtensionRangeTag, int32(extensionRangeIndex))
	return enclosingParent(msg.ParentFile().SourceLocations(), path)
}

func enclosingParent(srcLocs protoreflect.SourceLocations, path protoreflect.SourcePath) protoreflect.SourceLocation {
	loc := srcLocs.ByPath(path)
	if IsZero(loc) {
		return loc
	}
	parentPath := path[:len(path)-1]
	parentLoc := srcLocs.ByPath(parentPath)
	if IsZero(parentLoc) {
		return loc
	}
	for {
		if IsSubspanOf(loc, parentLoc) {
			return parentLoc
		}
		if parentLoc.Next == 0 {
			return protoreflect.SourceLocation{}
		}
		parentLoc = srcLocs.Get(parentLoc.Next)
	}
}
