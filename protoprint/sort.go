package protoprint

import (
	"fmt"
	"sort"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/sourceloc"
)

func (p *Printer) sort(elements elementAddrs, sourceInfo protoreflect.SourceLocations, path protoreflect.SourcePath) {
	if p.CustomSortFunction != nil {
		sort.Stable(customSortOrder{elementAddrs: elements, less: p.CustomSortFunction})
	} else if p.SortElements {
		// canonical sorted order
		sort.Stable(elements)
	} else {
		// use source order (per location information in SourceCodeInfo); or
		// if that isn't present use declaration order, but grouped by type
		sort.Stable(elementSrcOrder{
			elementAddrs: elements,
			sourceInfo:   sourceInfo,
			prefix:       path,
		})
	}
}

type elementAddr struct {
	elementType  int32
	elementIndex int
	order        int
}

type elementAddrs struct {
	addrs []elementAddr
	dsc   any
	opts  map[protoreflect.FieldNumber][]option
}

func (a elementAddrs) Len() int {
	return len(a.addrs)
}

func (a elementAddrs) Less(i, j int) bool {
	// explicit order is considered first
	addri := a.addrs[i]
	addrj := a.addrs[j]
	if addri.order < addrj.order {
		return true
	} else if addri.order > addrj.order {
		return false
	}
	// if order is equal, sort by element type
	if addri.elementType < addrj.elementType {
		return true
	} else if addri.elementType > addrj.elementType {
		return false
	}

	di := a.at(addri)
	dj := a.at(addrj)

	switch vi := di.(type) {
	case protoreflect.FieldDescriptor:
		// fields are ordered by tag number
		vj := dj.(protoreflect.FieldDescriptor)
		// regular fields before extensions; extensions grouped by extendee
		if !vi.IsExtension() && vj.IsExtension() {
			return true
		} else if vi.IsExtension() && !vj.IsExtension() {
			return false
		} else if vi.IsExtension() && vj.IsExtension() {
			if vi.ContainingMessage() != vj.ContainingMessage() {
				return vi.ContainingMessage().FullName() < vj.ContainingMessage().FullName()
			}
		}
		return vi.Number() < vj.Number()

	case protoreflect.EnumValueDescriptor:
		// enum values ordered by number then name,
		// but first value number must be 0 for open enums
		vj := dj.(protoreflect.EnumValueDescriptor)
		if vi.Number() == vj.Number() {
			return vi.Name() < vj.Name()
		}
		if ed, ok := vi.Parent().(protoreflect.EnumDescriptor); ok && !ed.IsClosed() {
			if vi.Number() == 0 {
				return true
			}
			if vj.Number() == 0 {
				return false
			}
		}
		return vi.Number() < vj.Number()

	case extensionRange:
		// extension ranges ordered by tag
		return vi.start < dj.(extensionRange).start

	case reservedRange:
		// reserved ranges ordered by tag, too
		return vi.start < dj.(reservedRange).start

	case protoreflect.Name:
		// reserved names lexically sorted
		return vi < dj.(protoreflect.Name)

	case pkg:
		// reserved names lexically sorted
		return vi < dj.(pkg)

	case protoreflect.FileImport:
		// reserved names lexically sorted
		return vi.Path() < dj.(protoreflect.FileImport).Path()

	case []option:
		// options sorted by name, extensions last
		return optionLess(vi, dj.([]option))

	default:
		// all other descriptors ordered by name
		return di.(protoreflect.Descriptor).Name() < dj.(protoreflect.Descriptor).Name()
	}
}

func (a elementAddrs) Swap(i, j int) {
	a.addrs[i], a.addrs[j] = a.addrs[j], a.addrs[i]
}

func (a elementAddrs) at(addr elementAddr) any {
	switch dsc := a.dsc.(type) {
	case protoreflect.FileDescriptor:
		switch addr.elementType {
		case internal.FilePackageTag:
			return pkg(dsc.Package())
		case internal.FileDependencyTag:
			return dsc.Imports().Get(addr.elementIndex)
		case internal.FileOptionsTag:
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		case internal.FileMessagesTag:
			return dsc.Messages().Get(addr.elementIndex)
		case internal.FileEnumsTag:
			return dsc.Enums().Get(addr.elementIndex)
		case internal.FileServicesTag:
			return dsc.Services().Get(addr.elementIndex)
		case internal.FileExtensionsTag:
			return dsc.Extensions().Get(addr.elementIndex)
		}
	case protoreflect.MessageDescriptor:
		switch addr.elementType {
		case internal.MessageOptionsTag:
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		case internal.MessageFieldsTag:
			return dsc.Fields().Get(addr.elementIndex)
		case internal.MessageNestedMessagesTag:
			return dsc.Messages().Get(addr.elementIndex)
		case internal.MessageEnumsTag:
			return dsc.Enums().Get(addr.elementIndex)
		case internal.MessageExtensionsTag:
			return dsc.Extensions().Get(addr.elementIndex)
		case internal.MessageExtensionRangeTag:
			extr := dsc.ExtensionRanges().Get(addr.elementIndex)
			return extensionRange{
				start: extr[0],
				end:   extr[1],
				opts:  dsc.ExtensionRangeOptions(addr.elementIndex),
			}
		case internal.MessageReservedRangeTag:
			rng := dsc.ReservedRanges().Get(addr.elementIndex)
			return reservedRange{start: int32(rng[0]), end: int32(rng[1]) - 1}
		case internal.MessageReservedNameTag:
			return dsc.ReservedNames().Get(addr.elementIndex)
		}
	case protoreflect.FieldDescriptor:
		if addr.elementType == internal.FieldOptionsTag {
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		}
	case protoreflect.OneofDescriptor:
		switch addr.elementType {
		case internal.OneofOptionsTag:
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		case -internal.MessageFieldsTag:
			return dsc.Parent().(protoreflect.MessageDescriptor).Fields().Get(addr.elementIndex)
		}
	case protoreflect.EnumDescriptor:
		switch addr.elementType {
		case internal.EnumOptionsTag:
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		case internal.EnumValuesTag:
			return dsc.Values().Get(addr.elementIndex)
		case internal.EnumReservedRangeTag:
			rng := dsc.ReservedRanges().Get(addr.elementIndex)
			return reservedRange{start: int32(rng[0]), end: int32(rng[1])}
		case internal.EnumReservedNameTag:
			return dsc.ReservedNames().Get(addr.elementIndex)
		}
	case protoreflect.EnumValueDescriptor:
		if addr.elementType == internal.EnumValueOptionsTag {
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		}
	case protoreflect.ServiceDescriptor:
		switch addr.elementType {
		case internal.ServiceOptionsTag:
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		case internal.ServiceMethodsTag:
			return dsc.Methods().Get(addr.elementIndex)
		}
	case protoreflect.MethodDescriptor:
		if addr.elementType == internal.MethodOptionsTag {
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		}
	case extensionRangeMarker:
		if addr.elementType == internal.ExtensionRangeOptionsTag {
			return a.opts[protoreflect.FieldNumber(addr.elementIndex)]
		}
	}

	panic(fmt.Sprintf("location for unknown field %d of %T", addr.elementType, a.dsc))
}

type extensionRangeMarker struct {
	owner protoreflect.MessageDescriptor
}

type elementSrcOrder struct {
	elementAddrs
	sourceInfo protoreflect.SourceLocations
	prefix     protoreflect.SourcePath
}

func (a elementSrcOrder) Less(i, j int) bool {
	ti := a.addrs[i].elementType
	ei := a.addrs[i].elementIndex

	tj := a.addrs[j].elementType
	ej := a.addrs[j].elementIndex

	var si, sj protoreflect.SourceLocation
	if ei < 0 {
		si = a.sourceInfo.ByPath(append(a.prefix, -int32(ei)))
	} else if ti < 0 {
		p := make([]int32, len(a.prefix)-2)
		copy(p, a.prefix)
		si = a.sourceInfo.ByPath(append(p, ti, int32(ei)))
	} else {
		si = a.sourceInfo.ByPath(append(a.prefix, ti, int32(ei)))
	}
	if ej < 0 {
		sj = a.sourceInfo.ByPath(append(a.prefix, -int32(ej)))
	} else if tj < 0 {
		p := make([]int32, len(a.prefix)-2)
		copy(p, a.prefix)
		sj = a.sourceInfo.ByPath(append(p, tj, int32(ej)))
	} else {
		sj = a.sourceInfo.ByPath(append(a.prefix, tj, int32(ej)))
	}

	if sourceloc.IsZero(si) != sourceloc.IsZero(sj) {
		// generally, we put unknown elements after known ones;
		// except package, imports, and option elements go first

		// i will be unknown and j will be known
		swapped := false
		if !sourceloc.IsZero(si) {
			ti, tj = tj, ti
			swapped = true
		}
		switch a.dsc.(type) {
		case protoreflect.FileDescriptor:
			// NB: These comparisons are *trying* to get things ordered so that
			// 1) If the package element has no source info, it appears _first_.
			// 2) If any import element has no source info, it appears _after_
			//    the package element but _before_ any other element.
			// 3) If any option element has no source info, it appears _after_
			//    the package and import elements but _before_ any other element.
			// If the package, imports, and options are all missing source info,
			// this will sort them all to the top in expected order. But if they
			// are mixed (some _do_ have source info, some do not), and elements
			// with source info have spans that positions them _after_ other
			// elements in the file, then this Less function will be unstable
			// since the above dual objectives for imports and options ("before
			// this but after that") may be in conflict with one another. This
			// should not cause any problems, other than elements being possibly
			// sorted in a confusing order.
			//
			// Well-formed descriptors should instead have consistent source
			// info: either all elements have source info or none do. So this
			// should not be an issue in practice.
			if ti == internal.FilePackageTag {
				return !swapped
			}
			if ti == internal.FileDependencyTag {
				if tj == internal.FilePackageTag {
					// imports will come *after* package
					return swapped
				}
				return !swapped
			}
			if ti == internal.FileOptionsTag {
				if tj == internal.FilePackageTag || tj == internal.FileDependencyTag {
					// options will come *after* package and imports
					return swapped
				}
				return !swapped
			}
		case protoreflect.MessageDescriptor:
			if ti == internal.MessageOptionsTag {
				return !swapped
			}
		case protoreflect.EnumDescriptor:
			if ti == internal.EnumOptionsTag {
				return !swapped
			}
		case protoreflect.ServiceDescriptor:
			if ti == internal.ServiceOptionsTag {
				return !swapped
			}
		}
		return swapped

	} else if sourceloc.IsZero(si) || sourceloc.IsZero(sj) {
		// let stable sort keep unknown elements in same relative order
		return false
	}

	if si.StartLine < sj.StartLine {
		return true
	}
	if si.StartLine > sj.StartLine {
		return false
	}
	if si.StartColumn < sj.StartColumn {
		return true
	}
	if si.StartColumn > sj.StartColumn {
		return false
	}
	if si.EndLine < sj.EndLine {
		return true
	}
	if si.EndLine > sj.EndLine {
		return false
	}
	return si.EndColumn < sj.EndColumn
}

type customSortOrder struct {
	elementAddrs
	less func(a, b Element) bool
}

func (cso customSortOrder) Less(i, j int) bool {
	// Regardless of the custom sort order, for proto3 files,
	// the enum value zero MUST be first. So we override the
	// custom sort order to make sure the file will be valid
	// and can compile.
	addri := cso.addrs[i]
	addrj := cso.addrs[j]
	di := cso.at(addri)
	dj := cso.at(addrj)
	if addri.elementType == addrj.elementType {
		if vi, ok := di.(protoreflect.EnumValueDescriptor); ok {
			vj := dj.(protoreflect.EnumValueDescriptor)
			if ed, ok := vi.Parent().(protoreflect.EnumDescriptor); ok && !ed.IsClosed() {
				if vi.Number() == 0 {
					return true
				}
				if vj.Number() == 0 {
					return false
				}
			}
		}
	}

	ei := asElement(di)
	ej := asElement(dj)
	return cso.less(ei, ej)
}

type optionsByName struct {
	addrs []elementAddr
	opts  map[protoreflect.FieldNumber][]option
}

func (o optionsByName) Len() int {
	return len(o.addrs)
}

func (o optionsByName) Less(i, j int) bool {
	oi := o.opts[protoreflect.FieldNumber(o.addrs[i].elementIndex)]
	oj := o.opts[protoreflect.FieldNumber(o.addrs[j].elementIndex)]
	return optionLess(oi, oj)
}

func (o optionsByName) Swap(i, j int) {
	o.addrs[i], o.addrs[j] = o.addrs[j], o.addrs[i]
}

func optionLess(i, j []option) bool {
	ni := i[0].name
	nj := j[0].name
	if ni[0] != '(' && nj[0] == '(' {
		return true
	} else if ni[0] == '(' && nj[0] != '(' {
		return false
	}
	return ni < nj
}
