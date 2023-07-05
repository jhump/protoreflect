package protoprint

import (
	"github.com/jhump/protoreflect/v2/internal"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/sourcelocation"
)

type sourceLocations struct {
	protoreflect.SourceLocations
	extrasByPath map[string]*protoreflect.SourceLocation
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
	if loc.Path != nil {
		return loc
	}
	k := internal.PathKey(path)
	pLoc := s.extrasByPath[k]
	if pLoc == nil {
		return protoreflect.SourceLocation{}
	}
	return *pLoc
}

func (s *sourceLocations) putIfAbsent(path protoreflect.SourcePath, loc protoreflect.SourceLocation) {
	if existing := s.ByPath(path); sourcelocation.IsZero(existing) {
		k := internal.PathKey(path)
		s.extras = append(s.extras, loc)
		if s.extrasByPath == nil {
			s.extrasByPath = map[string]*protoreflect.SourceLocation{}
		}
		s.extrasByPath[k] = &s.extras[len(s.extras)-1]
	}
}
