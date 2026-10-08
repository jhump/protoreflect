package protoprint

import (
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/sourceloc"
)

type sourceLocations struct {
	protoreflect.SourceLocations
	extrasByPath map[string]int // index into extras
	extras       []protoreflect.SourceLocation
}

func (s *sourceLocations) Len() int {
	return s.SourceLocations.Len() + len(s.extras)
}

func (s *sourceLocations) Get(i int) protoreflect.SourceLocation {
	if i < s.SourceLocations.Len() {
		return s.SourceLocations.Get(i)
	}
	return s.extras[i-s.SourceLocations.Len()]
}

func (s *sourceLocations) ByPath(path protoreflect.SourcePath) protoreflect.SourceLocation {
	loc := s.SourceLocations.ByPath(path)
	if !sourceloc.IsZero(loc) {
		return loc
	}
	extraIndex, ok := s.extrasByPath[internal.PathKey(path)]
	if !ok {
		return protoreflect.SourceLocation{}
	}
	return s.extras[extraIndex]
}

func (s *sourceLocations) putIfAbsent(path protoreflect.SourcePath, loc protoreflect.SourceLocation) {
	if existing := s.ByPath(path); sourceloc.IsZero(existing) {
		if s.extrasByPath == nil {
			s.extrasByPath = map[string]int{}
		}
		s.extrasByPath[internal.PathKey(path)] = len(s.extras)
		s.extras = append(s.extras, loc)
	}
}
