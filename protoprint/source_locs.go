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

type edgeKind int

const (
	edgeKindOption edgeKind = iota
	edgeKindFile
	edgeKindMessage
	edgeKindField
	edgeKindOneOf
	edgeKindExtensionRange
	edgeKindReservedRange
	edgeKindReservedName
	edgeKindEnum
	edgeKindEnumVal
	edgeKindService
	edgeKindMethod
)

// edges in simple state machine for matching options paths
// whose prefix should be included in source info to handle
// the way options are printed (which cannot always include
// the full path from original source)
var edges = map[edgeKind]map[int32]edgeKind{
	edgeKindFile: {
		internal.FileOptionsTag:    edgeKindOption,
		internal.FileMessagesTag:   edgeKindMessage,
		internal.FileEnumsTag:      edgeKindEnum,
		internal.FileExtensionsTag: edgeKindField,
		internal.FileServicesTag:   edgeKindService,
	},
	edgeKindMessage: {
		internal.MessageOptionsTag:        edgeKindOption,
		internal.MessageFieldsTag:         edgeKindField,
		internal.MessageOneofsTag:         edgeKindOneOf,
		internal.MessageNestedMessagesTag: edgeKindMessage,
		internal.MessageEnumsTag:          edgeKindEnum,
		internal.MessageExtensionsTag:     edgeKindField,
		internal.MessageExtensionRangeTag: edgeKindExtensionRange,
		internal.MessageReservedRangeTag:  edgeKindReservedRange,
		internal.MessageReservedNameTag:   edgeKindReservedName,
	},
	edgeKindField: {
		internal.FieldOptionsTag: edgeKindOption,
	},
	edgeKindOneOf: {
		internal.OneofOptionsTag: edgeKindOption,
	},
	edgeKindExtensionRange: {
		internal.ExtensionRangeOptionsTag: edgeKindOption,
	},
	edgeKindEnum: {
		internal.EnumOptionsTag:       edgeKindOption,
		internal.EnumValuesTag:        edgeKindEnumVal,
		internal.EnumReservedRangeTag: edgeKindReservedRange,
		internal.EnumReservedNameTag:  edgeKindReservedName,
	},
	edgeKindEnumVal: {
		internal.EnumValueOptionsTag: edgeKindOption,
	},
	edgeKindService: {
		internal.ServiceOptionsTag: edgeKindOption,
		internal.ServiceMethodsTag: edgeKindMethod,
	},
	edgeKindMethod: {
		internal.MethodOptionsTag: edgeKindOption,
	},
}

func extendOptionLocations(fd protoreflect.FileDescriptor) protoreflect.SourceLocations {
	// we iterate in the order that locations appear in descriptor
	// for determinism (if we ranged over the map, order and thus
	// potentially results are non-deterministic)
	srcLocs := sourceLocations{
		SourceLocations: fd.SourceLocations(),
	}

	for i, length := 0, srcLocs.Len(); i < length; i++ {
		loc := srcLocs.Get(i)
		allowed := edges[edgeKindFile]
		for pathIndex := 0; pathIndex+1 < len(loc.Path); pathIndex += 2 {
			nextKind, ok := allowed[loc.Path[pathIndex]]
			if !ok {
				break
			}
			if nextKind == edgeKindOption {
				// We've found an option entry. This could be arbitrarily deep
				// (for options that are nested messages) or it could end
				// abruptly (for non-repeated fields). But we need a path that
				// is exactly the path-so-far plus two: the option tag and an
				// optional index for repeated option fields (zero for
				// non-repeated option fields). This is used for querying source
				// info when printing options.
				newPath := make(protoreflect.SourcePath, pathIndex+3)
				copy(newPath, loc.Path)
				srcLocs.putIfAbsent(newPath, loc)
				// we do another path of path-so-far plus two, but with
				// explicit zero index -- just in case this actual path has
				// an extra path element, but it's not an index (e.g the
				// option field is not repeated, but the source info we are
				// looking at indicates a tag of a nested field)
				newPath[len(newPath)-1] = 0
				srcLocs.putIfAbsent(newPath, loc)
				// finally, we need the path-so-far plus one, just the option
				// tag, for sorting option groups
				newPath = newPath[:len(newPath)-1]
				srcLocs.putIfAbsent(newPath, loc)

				break
			} else {
				allowed = edges[nextKind]
			}
		}
	}

	// we also extend the package location with a synthetic zero index
	pkgPath := protoreflect.SourcePath{internal.FilePackageTag}
	pkgLoc := srcLocs.ByPath(protoreflect.SourcePath{internal.FilePackageTag})
	if pkgLoc.Path != nil {
		srcLocs.putIfAbsent(append(pkgPath, 0), pkgLoc)
	}

	if len(srcLocs.extras) == 0 {
		// no extras needed; just use original
		return srcLocs.SourceLocations
	}
	return &srcLocs
}
