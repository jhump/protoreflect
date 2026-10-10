package protoprint

import (
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/protomessage"
)

func (p *Printer) printMessage(
	md protoreflect.MessageDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	si := sourceInfo.ByPath(path)
	p.printBlockElement(true, si, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)

		_, _ = fmt.Fprint(w, "message ")
		nameSi := sourceInfo.ByPath(append(path, internal.MessageNameTag))
		p.printElementString(nameSi, w, indent, string(md.Name()))
		_, _ = fmt.Fprintln(w, "{")
		trailer(indent+1, true)

		p.printMessageBody(md, reg, w, sourceInfo, path, indent+1)
		p.indent(w, indent)
		_, _ = fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printMessageBody(
	md protoreflect.MessageDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	opts := p.extractOptions(md, reg, md.Options())

	skip := map[any]bool{}
	maxTag := internal.GetMaxTag(isMessageSet(md))

	elements := elementAddrs{dsc: md, opts: opts}
	elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.MessageOptionsTag, -1, opts)...)
	resRanges := md.ReservedRanges()
	for i, length := 0, resRanges.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageReservedRangeTag, elementIndex: i})
	}
	resNames := md.ReservedNames()
	for i, length := 0, resNames.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageReservedNameTag, elementIndex: i})
	}
	extRanges := md.ExtensionRanges()
	for i, length := 0, extRanges.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageExtensionRangeTag, elementIndex: i})
	}
	fields := md.Fields()
	for i, length := 0, fields.Len(); i < length; i++ {
		fld := fields.Get(i)
		if fld.IsMap() || isGroup(fld) {
			// we don't emit nested messages for map types or groups since
			// they get special treatment
			skip[fld.Message()] = true
		}
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageFieldsTag, elementIndex: i})
	}
	nestedMsgs := md.Messages()
	for i, length := 0, nestedMsgs.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageNestedMessagesTag, elementIndex: i})
	}
	nestedEnums := md.Enums()
	for i, length := 0, nestedEnums.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageEnumsTag, elementIndex: i})
	}
	extensions := p.computeExtensions(sourceInfo, md.Extensions(), append(path, internal.MessageExtensionsTag))
	exts := md.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		extd := exts.Get(i)
		if isGroup(extd) {
			// we don't emit nested messages for groups since
			// they get special treatment
			skip[extd.Message()] = true
		}
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageExtensionsTag, elementIndex: i})
	}

	p.sort(elements, sourceInfo, path)

	pkg := md.ParentFile().Package()
	scope := md.FullName()

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
		case protoreflect.FieldDescriptor:
			if d.IsExtension() {
				extDecl := extensions[d]
				p.printExtensions(extDecl, extensions, elements, i, reg, w, sourceInfo, path, internal.MessageExtensionsTag, pkg, scope, indent)
				// we printed all extensions in the group, so we can skip the others
				for _, fld := range extDecl.fields {
					skip[fld] = true
				}
			} else {
				ood := d.ContainingOneof()
				if ood == nil || ood.IsSynthetic() {
					p.printField(d, reg, w, sourceInfo, childPath, scope, indent)
				} else {
					// print the one-of, including all of its fields
					p.printOneOf(ood, elements, i, reg, w, sourceInfo, path, indent, ood.Index())
					fields := ood.Fields()
					for i, length := 0, fields.Len(); i < length; i++ {
						skip[fields.Get(i)] = true
					}
				}
			}
		case protoreflect.MessageDescriptor:
			p.printMessage(d, reg, w, sourceInfo, childPath, indent)
		case protoreflect.EnumDescriptor:
			p.printEnum(d, reg, w, sourceInfo, childPath, indent)
		case extensionRange:
			// collapse ranges into a single "extensions" block
			ranges := []extensionRange{d}
			addrs := []elementAddr{el}
			for idx := i + 1; idx < len(elements.addrs); idx++ {
				elnext := elements.addrs[idx]
				if elnext.elementType != el.elementType {
					break
				}
				extr := elements.at(elnext).(extensionRange)
				if !proto.Equal(d.opts, extr.opts) {
					break
				}
				ranges = append(ranges, extr)
				addrs = append(addrs, elnext)
				skip[extr] = true
			}
			p.printExtensionRanges(md, ranges, maxTag, addrs, reg, w, sourceInfo, path, indent)
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
			p.printReservedRanges(ranges, int32(maxTag), addrs, w, sourceInfo, path, indent)
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
			p.printReservedNames(names, addrs, w, sourceInfo, path, indent, reservedShouldUseQuotes(md))
		}
	}
}

func isMessageSet(msg protoreflect.MessageDescriptor) bool {
	opts, _ := protomessage.As[*descriptorpb.MessageOptions](msg.Options())
	return opts.GetMessageSetWireFormat()
}

func (p *Printer) printField(
	fld protoreflect.FieldDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	scope protoreflect.FullName,
	indent int,
) {
	var groupPath []int32
	var si protoreflect.SourceLocation

	group := isGroup(fld)

	if group {
		// compute path to group message type
		groupPath = make([]int32, len(path)-2)
		copy(groupPath, path)

		var candidates protoreflect.MessageDescriptors
		var parentTag int32
		switch parent := fld.Parent().(type) {
		case protoreflect.MessageDescriptor:
			// group in a message
			candidates = parent.Messages()
			parentTag = internal.MessageNestedMessagesTag
		case protoreflect.FileDescriptor:
			// group that is a top-level extension
			candidates = parent.Messages()
			parentTag = internal.FileMessagesTag
		}

		var groupMsgIndex int32
		for i, length := 0, candidates.Len(); i < length; i++ {
			nmd := candidates.Get(i)
			if nmd == fld.Message() {
				// found it
				groupMsgIndex = int32(i)
				break
			}
		}
		groupPath = append(groupPath, parentTag, groupMsgIndex)

		// the group message is where the field's comments and position are stored
		si = sourceInfo.ByPath(groupPath)
	} else {
		si = sourceInfo.ByPath(path)
	}

	p.printBlockElement(true, si, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)
		if shouldEmitLabel(fld) {
			locSi := sourceInfo.ByPath(append(path, internal.FieldLabelTag))
			p.printElementString(locSi, w, indent, fld.Cardinality().String())
		}

		if group {
			_, _ = fmt.Fprint(w, "group ")
		}

		var tag int32
		switch fld.Kind() {
		case protoreflect.EnumKind, protoreflect.GroupKind, protoreflect.MessageKind:
			tag = internal.FieldTypeNameTag
		default:
			tag = internal.FieldTypeTag
		}
		typeSi := sourceInfo.ByPath(append(path, tag))
		p.printElementString(typeSi, w, indent, p.typeString(fld, scope))

		if !group {
			nameSi := sourceInfo.ByPath(append(path, internal.FieldNameTag))
			p.printElementString(nameSi, w, indent, string(fld.Name()))
		}

		_, _ = fmt.Fprint(w, "= ")
		numSi := sourceInfo.ByPath(append(path, internal.FieldNumberTag))
		p.printElementString(numSi, w, indent, fmt.Sprintf("%d", fld.Number()))

		opts := p.extractOptions(fld, reg, fld.Options())

		// we use negative values for "extras" keys so they can't collide
		// with legit option tags

		if fld.HasPresence() && fld.HasDefault() {
			var defVal any
			if fld.Enum() != nil {
				defVal = ident(fld.DefaultEnumValue().Name())
			} else {
				defVal = fld.Default().Interface()
			}
			opts[-internal.FieldDefaultTag] = []option{{name: "default", val: defVal}}
		}

		jsn := fld.JSONName()
		if !fld.IsExtension() && jsn != "" && jsn != internal.JSONName(fld.Name()) {
			opts[-internal.FieldJSONNameTag] = []option{{name: "json_name", val: jsn}}
		}

		p.printOptionsShort(fld, opts, internal.FieldOptionsTag, reg, w, sourceInfo, path, indent)

		if group {
			_, _ = fmt.Fprintln(w, "{")
			trailer(indent+1, true)

			p.printMessageBody(fld.Message(), reg, w, sourceInfo, groupPath, indent+1)

			p.indent(w, indent)
			_, _ = fmt.Fprintln(w, "}")

		} else {
			_, _ = fmt.Fprint(w, ";")
			trailer(indent, false)
		}
	})
}

func isGroup(fld protoreflect.FieldDescriptor) bool {
	// Groups are a proto2 thing. If we see GroupLKind, but in editions, it
	// really just means a field with delimited message encoding.
	return fld.Kind() == protoreflect.GroupKind && fld.Syntax() != protoreflect.Editions
}

func shouldEmitLabel(fld protoreflect.FieldDescriptor) bool {
	card := fld.Cardinality()
	if card == protoreflect.Required && fld.Syntax() == protoreflect.Editions {
		// no required label in editions (it will come from a feature)
		return false
	}
	return (fld.ContainingOneof() != nil && fld.ContainingOneof().IsSynthetic()) ||
		(!fld.IsMap() && fld.ContainingOneof() == nil &&
			(card != protoreflect.Optional || fld.ParentFile().Syntax() == protoreflect.Proto2))
}

func (p *Printer) printOneOf(
	ood protoreflect.OneofDescriptor,
	parentElements elementAddrs,
	startFieldIndex int,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	parentPath protoreflect.SourcePath,
	indent int,
	ooIndex int,
) {
	oopath := append(parentPath, internal.MessageOneofsTag, int32(ooIndex))
	oosi := sourceInfo.ByPath(oopath)
	p.printBlockElement(true, oosi, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)
		_, _ = fmt.Fprint(w, "oneof ")
		extNameSi := sourceInfo.ByPath(append(oopath, internal.OneofNameTag))
		p.printElementString(extNameSi, w, indent, string(ood.Name()))
		_, _ = fmt.Fprintln(w, "{")
		indent++
		trailer(indent, true)

		opts := p.extractOptions(ood, reg, ood.Options())

		elements := elementAddrs{dsc: ood, opts: opts}
		elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.OneofOptionsTag, -1, opts)...)

		count := ood.Fields().Len()
		for idx := startFieldIndex; count > 0 && idx < len(parentElements.addrs); idx++ {
			el := parentElements.addrs[idx]
			if el.elementType != internal.MessageFieldsTag {
				continue
			}
			if parentElements.at(el).(protoreflect.FieldDescriptor).ContainingOneof() == ood {
				// negative tag indicates that this element is actually a sibling, not a child
				elements.addrs = append(elements.addrs, elementAddr{elementType: -internal.MessageFieldsTag, elementIndex: el.elementIndex})
				count--
			}
		}

		// the fields are already sorted, but we have to re-sort in order to
		// interleave the options (in the event that we are using file location
		// order and the option locations are interleaved with the fields)
		p.sort(elements, sourceInfo, oopath)
		scope := ood.Parent().FullName()

		for i, el := range elements.addrs {
			if i > 0 {
				p.newLine(w)
			}

			switch d := elements.at(el).(type) {
			case []option:
				childPath := append(oopath, el.elementType, int32(el.elementIndex))
				p.printOptionsLong(d, reg, w, sourceInfo, childPath, indent)
			case protoreflect.FieldDescriptor:
				childPath := append(parentPath, -el.elementType, int32(el.elementIndex))
				p.printField(d, reg, w, sourceInfo, childPath, scope, indent)
			}
		}

		p.indent(w, indent-1)
		_, _ = fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printExtensionRanges(
	parent protoreflect.MessageDescriptor,
	ranges []extensionRange,
	maxTag protoreflect.FieldNumber,
	addrs []elementAddr,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	parentPath protoreflect.SourcePath,
	indent int,
) {
	p.indent(w, indent)
	_, _ = fmt.Fprint(w, "extensions ")

	var opts proto.Message
	var elPath protoreflect.SourcePath
	first := true
	for i, extr := range ranges {
		if first {
			first = false
		} else {
			_, _ = fmt.Fprint(w, ", ")
		}
		opts = extr.opts
		el := addrs[i]
		elPath = append(parentPath, el.elementType, int32(el.elementIndex))
		si := sourceInfo.ByPath(elPath)
		p.printElement(true, si, w, inline(indent), func(w *writer) {
			if extr.start == extr.end-1 {
				_, _ = fmt.Fprintf(w, "%d ", extr.start)
			} else if extr.end-1 == maxTag {
				_, _ = fmt.Fprintf(w, "%d to max ", extr.start)
			} else {
				_, _ = fmt.Fprintf(w, "%d to %d ", extr.start, extr.end-1)
			}
		})
	}
	dsc := extensionRangeMarker{owner: parent}
	p.extractAndPrintOptionsShort(dsc, opts, reg, internal.ExtensionRangeOptionsTag, w, sourceInfo, elPath, indent)

	_, _ = fmt.Fprintln(w, ";")
}

func (p *Printer) printReservedRanges(
	ranges []reservedRange,
	maxVal int32,
	addrs []elementAddr,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	parentPath protoreflect.SourcePath,
	indent int,
) {
	p.indent(w, indent)
	_, _ = fmt.Fprint(w, "reserved ")

	first := true
	for i, rr := range ranges {
		if first {
			first = false
		} else {
			_, _ = fmt.Fprint(w, ", ")
		}
		el := addrs[i]
		si := sourceInfo.ByPath(append(parentPath, el.elementType, int32(el.elementIndex)))
		p.printElement(false, si, w, inline(indent), func(w *writer) {
			if rr.start == rr.end {
				_, _ = fmt.Fprintf(w, "%d ", rr.start)
			} else if rr.end == maxVal {
				_, _ = fmt.Fprintf(w, "%d to max ", rr.start)
			} else {
				_, _ = fmt.Fprintf(w, "%d to %d ", rr.start, rr.end)
			}
		})
	}

	_, _ = fmt.Fprintln(w, ";")
}

func reservedShouldUseQuotes(d protoreflect.Descriptor) bool {
	return d.Syntax() != protoreflect.Editions
}

func (p *Printer) printReservedNames(
	names []protoreflect.Name,
	addrs []elementAddr,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	parentPath protoreflect.SourcePath,
	indent int,
	useQuotes bool,
) {
	p.indent(w, indent)
	_, _ = fmt.Fprint(w, "reserved ")

	first := true
	for i, name := range names {
		if first {
			first = false
		} else {
			_, _ = fmt.Fprint(w, ", ")
		}
		el := addrs[i]
		si := sourceInfo.ByPath(append(parentPath, el.elementType, int32(el.elementIndex)))
		reservedName := string(name)
		if useQuotes {
			reservedName = quotedString(reservedName)
		}
		p.printElementString(si, w, indent, reservedName)
	}

	_, _ = fmt.Fprintln(w, ";")
}
