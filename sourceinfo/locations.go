package sourceinfo

import (
	"github.com/jhump/protoreflect/v2/sourcelocation"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal"
)

// NB: forked from google.golang.org/protobuf/internal/filedesc/desc_list.go.
// Changes made:
//   * Use internal tag constants since genid is unavailable.
//   * Use internal.PathKey instead of bespoke pathKey type and newPathKey function.
//   * Use sourcelocation.PathFor in ByDescriptor to compute source path.

type sourceLocations struct {
	protoreflect.SourceLocations

	orig []*descriptorpb.SourceCodeInfo_Location
	// locs is a list of sourceLocations.
	// The SourceLocation.Next field does not need to be populated
	// as it will be lazily populated upon first need.
	locs []protoreflect.SourceLocation

	// fd is the parent file descriptor that these locations are relative to.
	// If non-nil, ByDescriptor verifies that the provided descriptor
	// is a child of this file descriptor.
	fd protoreflect.FileDescriptor

	once   sync.Once
	byPath map[string]int
}

func (p *sourceLocations) Len() int { return len(p.orig) }
func (p *sourceLocations) Get(i int) protoreflect.SourceLocation {
	return p.lazyInit().locs[i]
}
func (p *sourceLocations) byKey(k string) protoreflect.SourceLocation {
	if i, ok := p.lazyInit().byPath[k]; ok {
		return p.locs[i]
	}
	return protoreflect.SourceLocation{}
}
func (p *sourceLocations) ByPath(path protoreflect.SourcePath) protoreflect.SourceLocation {
	return p.byKey(internal.PathKey(path))
}
func (p *sourceLocations) ByDescriptor(desc protoreflect.Descriptor) protoreflect.SourceLocation {
	if p.fd != nil && desc != nil && p.fd != desc.ParentFile() {
		return protoreflect.SourceLocation{} // mismatching parent imports
	}
	path := sourcelocation.PathFor(desc)
	if path == nil {
		return protoreflect.SourceLocation{}
	}
	return p.byKey(internal.PathKey(path))
}

func (p *sourceLocations) lazyInit() *sourceLocations {
	p.once.Do(func() {
		if len(p.orig) > 0 {
			p.locs = make([]protoreflect.SourceLocation, len(p.orig))
			// Collect all the indexes for a given path.
			pathIdxs := make(map[string][]int, len(p.locs))
			for i := range p.orig {
				l := asSourceLocation(p.orig[i])
				p.locs[i] = l
				k := internal.PathKey(l.Path)
				pathIdxs[k] = append(pathIdxs[k], i)
			}

			// Update the next index for all locations.
			p.byPath = make(map[string]int, len(p.locs))
			for k, idxs := range pathIdxs {
				for i := 0; i < len(idxs)-1; i++ {
					p.locs[idxs[i]].Next = idxs[i+1]
				}
				p.locs[idxs[len(idxs)-1]].Next = 0
				p.byPath[k] = idxs[0] // record the first location for this path
			}
		}
	})
	return p
}

func asSourceLocation(l *descriptorpb.SourceCodeInfo_Location) protoreflect.SourceLocation {
	endLine := l.Span[0]
	endCol := l.Span[2]
	if len(l.Span) > 3 {
		endLine = l.Span[2]
		endCol = l.Span[3]
	}
	return protoreflect.SourceLocation{
		Path:                    l.Path,
		StartLine:               int(l.Span[0]),
		StartColumn:             int(l.Span[1]),
		EndLine:                 int(endLine),
		EndColumn:               int(endCol),
		LeadingDetachedComments: l.LeadingDetachedComments,
		LeadingComments:         l.GetLeadingComments(),
		TrailingComments:        l.GetTrailingComments(),
	}
}
