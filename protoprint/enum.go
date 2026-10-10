package protoprint

import (
	"fmt"
	"math"
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal"
)

func (p *Printer) printEnum(
	ed protoreflect.EnumDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	si := sourceInfo.ByPath(path)
	p.printBlockElement(true, si, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)

		_, _ = fmt.Fprint(w, "enum ")
		nameSi := sourceInfo.ByPath(append(path, internal.EnumNameTag))
		p.printElementString(nameSi, w, indent, string(ed.Name()))
		_, _ = fmt.Fprintln(w, "{")
		indent++
		trailer(indent, true)

		opts := p.extractOptions(ed, reg, ed.Options())

		skip := map[any]bool{}

		elements := elementAddrs{dsc: ed, opts: opts}
		elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.EnumOptionsTag, -1, opts)...)
		vals := ed.Values()
		for i, length := 0, vals.Len(); i < length; i++ {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.EnumValuesTag, elementIndex: i})
		}
		resRanges := ed.ReservedRanges()
		for i, length := 0, resRanges.Len(); i < length; i++ {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.EnumReservedRangeTag, elementIndex: i})
		}
		resNames := ed.ReservedNames()
		for i, length := 0, resNames.Len(); i < length; i++ {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.EnumReservedNameTag, elementIndex: i})
		}

		p.sort(elements, sourceInfo, path)

		for i, el := range elements.addrs {
			d := elements.at(el)

			// skip[d] will panic if d is a slice (which it could be for []option),
			// so just ignore it since we don't try to skip options
			if reflect.TypeOf(d).Kind() != reflect.Slice && skip[d] {
				// skip this element
				continue
			}

			if i > 0 {
				p.newLine(w)
			}

			childPath := append(path, el.elementType, int32(el.elementIndex))

			switch d := d.(type) {
			case []option:
				p.printOptionsLong(d, reg, w, sourceInfo, childPath, indent)
			case protoreflect.EnumValueDescriptor:
				p.printEnumValue(d, reg, w, sourceInfo, childPath, indent)
			case reservedRange:
				// collapse reserved ranges into a single "reserved" block
				ranges := []reservedRange{d}
				addrs := []elementAddr{el}
				for idx := i + 1; idx < len(elements.addrs); idx++ {
					elnext := elements.addrs[idx]
					if elnext.elementType != el.elementType {
						break
					}
					rr := elements.at(elnext).(reservedRange)
					ranges = append(ranges, rr)
					addrs = append(addrs, elnext)
					skip[rr] = true
				}
				p.printReservedRanges(ranges, math.MaxInt32, addrs, w, sourceInfo, path, indent)
			case protoreflect.Name: // reserved name
				// collapse reserved names into a single "reserved" block
				names := []protoreflect.Name{d}
				addrs := []elementAddr{el}
				for idx := i + 1; idx < len(elements.addrs); idx++ {
					elnext := elements.addrs[idx]
					if elnext.elementType != el.elementType {
						break
					}
					rn := elements.at(elnext).(protoreflect.Name)
					names = append(names, rn)
					addrs = append(addrs, elnext)
					skip[rn] = true
				}
				p.printReservedNames(names, addrs, w, sourceInfo, path, indent, reservedShouldUseQuotes(ed))
			}
		}

		p.indent(w, indent-1)
		_, _ = fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printEnumValue(
	evd protoreflect.EnumValueDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	si := sourceInfo.ByPath(path)
	p.printElement(true, si, w, indent, func(w *writer) {
		p.indent(w, indent)

		nameSi := sourceInfo.ByPath(append(path, internal.EnumValueNameTag))
		p.printElementString(nameSi, w, indent, string(evd.Name()))
		_, _ = fmt.Fprint(w, "= ")

		numSi := sourceInfo.ByPath(append(path, internal.EnumValueNumberTag))
		p.printElementString(numSi, w, indent, fmt.Sprintf("%d", evd.Number()))

		p.extractAndPrintOptionsShort(evd, evd.Options(), reg, internal.EnumValueOptionsTag, w, sourceInfo, path, indent)

		_, _ = fmt.Fprint(w, ";")
	})
}
