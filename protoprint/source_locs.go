package protoprint

import (
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
	k := pathKey(path)
	pLoc := s.extrasByPath[k]
	if pLoc == nil {
		return protoreflect.SourceLocation{}
	}
	return *pLoc
}

func (s *sourceLocations) putIfAbsent(path protoreflect.SourcePath, loc protoreflect.SourceLocation) {
	if existing := s.ByPath(path); sourcelocation.IsZero(existing) {
		k := pathKey(path)
		s.extras = append(s.extras, loc)
		if s.extrasByPath == nil {
			s.extrasByPath = map[string]*protoreflect.SourceLocation{}
		}
		s.extrasByPath[k] = &s.extras[len(s.extras)-1]
	}
}

func pathKey(path protoreflect.SourcePath) string {
	b := make([]byte, len(path)*4)
	j := 0
	for _, s := range path {
		b[j] = byte(s)
		b[j+1] = byte(s >> 8)
		b[j+2] = byte(s >> 16)
		b[j+3] = byte(s >> 24)
		j += 4
	}
	return string(b)
}
