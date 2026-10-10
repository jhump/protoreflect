package protoprint

import (
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/protodescs"
	"github.com/jhump/protoreflect/v2/sourceloc"
)

func (p *Printer) printFile(
	fd protoreflect.FileDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
) {
	opts := p.extractOptions(fd, reg, fd.Options())

	path := make(protoreflect.SourcePath, 1)

	path[0] = internal.FileSyntaxTag
	si := sourceInfo.ByPath(path)
	p.printElement(false, si, w, 0, func(w *writer) {
		syn := fd.Syntax()
		if syn != protoreflect.Editions {
			_, _ = fmt.Fprintf(w, "syntax = %q;", syn.String())
			return
		}
		_, _ = fmt.Fprintf(w, "edition = %q;", strings.TrimPrefix(protodescs.GetEdition(fd, nil).String(), "EDITION_"))
	})
	p.newLine(w)

	skip := map[any]bool{}

	elements := elementAddrs{dsc: fd, opts: opts}
	if fd.Package() != "" {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.FilePackageTag, elementIndex: 0, order: -3})
	}
	imps := fd.Imports()
	for i, length := 0, imps.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.FileDependencyTag, elementIndex: i, order: -2})
	}
	elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.FileOptionsTag, -1, opts)...)
	msgs := fd.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.FileMessagesTag, elementIndex: i})
	}
	enums := fd.Enums()
	for i, length := 0, enums.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.FileEnumsTag, elementIndex: i})
	}
	svcs := fd.Services()
	for i, length := 0, svcs.Len(); i < length; i++ {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.FileServicesTag, elementIndex: i})
	}
	extensions := p.computeExtensions(sourceInfo, fd.Extensions(), []int32{internal.FileExtensionsTag})
	exts := fd.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		extd := exts.Get(i)
		if isGroup(extd) {
			// we don't emit nested messages for groups since
			// they get special treatment
			skip[extd.Message()] = true
		}
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.FileExtensionsTag, elementIndex: i})
	}

	p.sort(elements, sourceInfo, nil)

	pkgName := fd.Package()

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

		path = []int32{el.elementType, int32(el.elementIndex)}

		switch d := d.(type) {
		case pkg:
			si := sourceInfo.ByPath(path)
			p.printElement(false, si, w, 0, func(w *writer) {
				_, _ = fmt.Fprintf(w, "package %s;", d)
			})
		case protoreflect.FileImport:
			si := sourceInfo.ByPath(path)
			var modifier string
			if d.IsPublic {
				modifier = "public "
				//lint:ignore SA1019 not using weak import functionality, but must inspect this flag even though it's deprecated
			} else if d.IsWeak {
				modifier = "weak "

			}
			p.printElement(false, si, w, 0, func(w *writer) {
				_, _ = fmt.Fprintf(w, "import %s%q;", modifier, d.Path())
			})
		case []option:
			p.printOptionsLong(d, reg, w, sourceInfo, path, 0)
		case protoreflect.MessageDescriptor:
			p.printMessage(d, reg, w, sourceInfo, path, 0)
		case protoreflect.EnumDescriptor:
			p.printEnum(d, reg, w, sourceInfo, path, 0)
		case protoreflect.ServiceDescriptor:
			p.printService(d, reg, w, sourceInfo, path, 0)
		case protoreflect.FieldDescriptor:
			extDecl := extensions[d]
			p.printExtensions(extDecl, extensions, elements, i, reg, w, sourceInfo, nil, internal.FileExtensionsTag, pkgName, pkgName, 0)
			// we printed all extensions in the group, so we can skip the others
			for _, fld := range extDecl.fields {
				skip[fld] = true
			}
		}
	}
}

func findExtSi(locs protoreflect.SourceLocations, fieldSi, extSi protoreflect.SourceLocation) protoreflect.SourceLocation {
	if sourceloc.IsZero(fieldSi) {
		return protoreflect.SourceLocation{}
	}
	for {
		if isSpanWithin(fieldSi, extSi) {
			return extSi
		}
		if extSi.Next == 0 {
			break
		}
		extSi = locs.Get(extSi.Next)
	}
	return protoreflect.SourceLocation{}
}

func isSpanWithin(span, enclosing protoreflect.SourceLocation) bool {
	if span.StartLine < enclosing.StartLine || span.StartLine > enclosing.EndLine {
		return false
	}
	if span.StartLine == enclosing.StartLine {
		return span.StartColumn >= enclosing.StartColumn
	} else if span.StartLine == enclosing.EndLine {
		return span.StartColumn <= enclosing.EndColumn
	}
	return true
}

type extensionDecl struct {
	extendee   protoreflect.FullName
	sourceInfo protoreflect.SourceLocation
	fields     []protoreflect.FieldDescriptor
}

type extensions map[protoreflect.FieldDescriptor]*extensionDecl

type span struct {
	startLine, startCol, endLine, endCol int
}

func (p *Printer) computeExtensions(sourceInfo protoreflect.SourceLocations, exts protoreflect.ExtensionDescriptors, path []int32) extensions {
	extsMap := map[protoreflect.FullName]map[span]*extensionDecl{}
	extSis := sourceInfo.ByPath(path)
	for i, length := 0, exts.Len(); i < length; i++ {
		extd := exts.Get(i)
		name := extd.ContainingMessage().FullName()
		extSi := findExtSi(sourceInfo, sourceInfo.ByDescriptor(extd), extSis)
		extsBySpan := extsMap[name]
		if extsBySpan == nil {
			extsBySpan = map[span]*extensionDecl{}
			extsMap[name] = extsBySpan
		}
		sp := span{
			startLine: extSi.StartLine,
			startCol:  extSi.StartColumn,
			endLine:   extSi.EndLine,
			endCol:    extSi.EndColumn,
		}
		extDecl := extsBySpan[sp]
		if extDecl == nil {
			extDecl = &extensionDecl{
				sourceInfo: extSi,
				extendee:   name,
			}
			extsBySpan[sp] = extDecl
		}
		extDecl.fields = append(extDecl.fields, extd)
	}

	ret := extensions{}
	for _, extsBySi := range extsMap {
		for _, extDecl := range extsBySi {
			for _, extd := range extDecl.fields {
				ret[extd] = extDecl
			}
		}
	}
	return ret
}

func (p *Printer) printExtensions(
	exts *extensionDecl,
	allExts extensions,
	parentElements elementAddrs,
	startFieldIndex int,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	parentPath protoreflect.SourcePath,
	extTag int32, pkg,
	scope protoreflect.FullName,
	indent int,
) {
	path := append(parentPath, extTag)
	p.printLeadingComments(exts.sourceInfo, w, indent)
	p.indent(w, indent)
	_, _ = fmt.Fprint(w, "extend ")
	extNameSi := sourceInfo.ByPath(append(path, 0, internal.FieldExtendeeTag))
	p.printElementString(extNameSi, w, indent, p.qualifyTypeName(pkg, scope, exts.extendee))
	_, _ = fmt.Fprintln(w, "{")

	if p.printTrailingComments(exts.sourceInfo, w, indent+1) && !p.Compact {
		// separator line between trailing comment and next element
		_, _ = fmt.Fprintln(w)
	}

	count := len(exts.fields)
	first := true
	for idx := startFieldIndex; count > 0 && idx < len(parentElements.addrs); idx++ {
		el := parentElements.addrs[idx]
		if el.elementType != extTag {
			continue
		}
		fld := parentElements.at(el).(protoreflect.FieldDescriptor)
		if allExts[fld] == exts {
			if first {
				first = false
			} else {
				p.newLine(w)
			}
			childPath := append(path, int32(el.elementIndex))
			p.printField(fld, reg, w, sourceInfo, childPath, scope, indent+1)
			count--
		}
	}

	p.indent(w, indent)
	_, _ = fmt.Fprintln(w, "}")
}
