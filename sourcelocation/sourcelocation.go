package sourcelocation

import "google.golang.org/protobuf/reflect/protoreflect"

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
