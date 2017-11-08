package protoprint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/internal"
	"github.com/jhump/protoreflect/dynamic"
)

const (
	// NB: It would be nice to use constants from generated code instead of hard-coding these here.
	// But code-gen does not emit these as constants anywhere. The only places they appear in generated
	// code are struct tags on fields of the generated descriptor protos.
	file_packageTag           = 2
	file_dependencyTag        = 3
	file_messagesTag          = 4
	file_enumsTag             = 5
	file_servicesTag          = 6
	file_extensionsTag        = 7
	file_syntaxTag            = 12
	message_nameTag           = 1
	message_fieldsTag         = 2
	message_nestedMessagesTag = 3
	message_enumsTag          = 4
	message_extensionRangeTag = 5
	message_extensionsTag     = 6
	message_oneOfsTag         = 8
	message_reservedRangeTag  = 9
	message_reservedNameTag   = 10
	field_nameTag             = 1
	field_extendeeTag         = 2
	field_numberTag           = 3
	field_labelTag            = 4
	field_typeTag             = 5
	oneof_nameTag             = 1
	enum_nameTag              = 1
	enum_valuesTag            = 2
	enumVal_nameTag           = 1
	enumVal_numberTag         = 2
	service_nameTag           = 1
	service_methodsTag        = 2
	method_nameTag            = 1
	method_inputTag           = 2
	method_outputTag          = 3
)

type Printer struct {
	PreferMultiLineStyleComments bool
	SortElements                 bool
	Indent                       string
}

func (p *Printer) PrintProtoFiles(fds []*desc.FileDescriptor, open func(name string) (io.WriteCloser, error)) error {
	for _, fd := range fds {
		w, err := open(fd.GetName())
		if err != nil {
			return fmt.Errorf("failed to open %s: %v", fd.GetName(), err)
		}
		err = func() error {
			defer w.Close()
			return p.PrintProtoFile(fd, w)
		}()
		if err != nil {
			return fmt.Errorf("failed to write %s: %v", fd.GetName(), err)
		}
	}
	return nil
}

func (p *Printer) PrintProtosToFileSystem(fds []*desc.FileDescriptor, rootDir string) error {
	return p.PrintProtoFiles(fds, func(name string) (io.WriteCloser, error) {
		return os.OpenFile(filepath.Join(rootDir, name), os.O_CREATE|os.O_WRONLY, 0666)
	})
}

func (p *Printer) PrintProtoFile(fd *desc.FileDescriptor, w io.Writer) error {
	out := &printer{Writer: w}
	if p.Indent == "" {
		// default indent to two spaces
		p.Indent = "  "
	} else {
		// indent must be all spaces or tabs, so convert other chars to spaces
		ind := make([]rune, 0, len(p.Indent))
		for _, r := range p.Indent {
			if r == '\t' {
				ind = append(ind, r)
			} else {
				ind = append(ind, ' ')
			}
		}
		p.Indent = string(ind)
	}

	er := extensionsForTransitiveClosure(fd)
	mf := dynamic.NewMessageFactoryWithExtensionRegistry(er)
	fdp := fd.AsFileDescriptorProto()
	sourceInfo := createSourceInfoMap(fdp)
	path := make([]int32, 1)

	path[0] = file_syntaxTag
	si := sourceInfo.Get(path)
	p.printElement(si, out, 0, func(w *printer) {
		syn := fdp.GetSyntax()
		if syn == "" {
			syn = "proto2"
		}
		fmt.Fprintf(w, "syntax = %q;\n", syn)
	})
	fmt.Fprintln(out)

	if fdp.Package != nil {
		path[0] = file_packageTag
		si := sourceInfo.Get(path)
		p.printElement(si, out, 0, func(w *printer) {
			fmt.Fprintf(w, "package %s;\n", fdp.GetPackage())
		})
		fmt.Fprintln(out)
	}

	if len(fdp.Dependency) > 0 {
		path[0] = file_dependencyTag
		for i, dep := range fdp.Dependency {
			path := append(path, int32(i))
			si := sourceInfo.Get(path)
			p.printElement(si, out, 0, func(w *printer) {
				fmt.Fprintf(w, "import %q;\n", dep)
			})
			fmt.Fprintln(out)
		}
	}

	hadOptions := p.printOptionsLong(fd.GetOptions(), mf, out, 0)

	elements := elementAddrs{dsc: fd}
	for i := range fd.GetMessageTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: file_messagesTag, elementIndex: i})
	}
	for i := range fd.GetEnumTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: file_enumsTag, elementIndex: i})
	}
	for i := range fd.GetServices() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: file_servicesTag, elementIndex: i})
	}
	for i := range fd.GetExtensions() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: file_extensionsTag, elementIndex: i})
	}

	if p.SortElements {
		// canonical sorted order
		sort.Stable(elements)
	} else {
		// use source order (per location information in SourceCodeInfo); or
		// if that isn't present use declaration order, but grouped by type
		sort.Stable(elementSrcOrder{
			elementAddrs: elements,
			sourceInfo:   sourceInfo,
		})
	}

	pkg := fd.GetPackage()

	var ext *desc.FieldDescriptor
	var extSi *descriptor.SourceCodeInfo_Location
	for i, el := range elements.addrs {
		if i == 0 && hadOptions {
			fmt.Fprintln(out)
		}

		d := elements.at(el)
		path = []int32{el.elementType, int32(el.elementIndex)}
		if el.elementType == file_extensionsTag {
			fld := d.(*desc.FieldDescriptor)
			if ext == nil || ext.GetOwner() != fld.GetOwner() {
				// need to open a new extend block
				if ext != nil {
					// close preceding extend block
					fmt.Fprintln(out, "}")
					p.printTrailingComments(extSi, out, 0)
				}
				if i > 0 {
					fmt.Fprintln(out)
				}

				ext = fld
				extSi = sourceInfo.Get(path)
				p.printLeadingComments(extSi, out, 0)

				fmt.Fprint(out, "extend ")
				extNameSi := sourceInfo.Get(append(path, field_extendeeTag))
				p.printElementString(extNameSi, out, 0, getLocalName(pkg, pkg, fld.GetOwner().GetFullyQualifiedName()))
				fmt.Fprintln(out, "{")
			} else {
				fmt.Fprintln(out)
			}
			p.printField(fld, mf, out, sourceInfo, path, pkg, 1)
		} else {
			if ext != nil {
				// close preceding extend block
				fmt.Fprintln(out, "}")
				p.printTrailingComments(extSi, out, 0)
				ext = nil
				extSi = nil
			}

			if i > 0 {
				fmt.Fprintln(out)
			}

			switch d := d.(type) {
			case *desc.MessageDescriptor:
				p.printMessage(d, mf, out, sourceInfo, path, 0)
			case *desc.EnumDescriptor:
				p.printEnum(d, mf, out, sourceInfo, path, 0)
			case *desc.ServiceDescriptor:
				p.printService(d, mf, out, sourceInfo, path, 0)
			}
		}
	}

	if ext != nil {
		// close trailing extend block
		fmt.Fprintln(out, "}")
		p.printTrailingComments(extSi, out, 0)
	}

	return out.err
}

func extensionsForTransitiveClosure(fd *desc.FileDescriptor) *dynamic.ExtensionRegistry {
	er := dynamic.NewExtensionRegistryWithDefaults()
	recursiveExtensionsFromFile(er, fd, map[string]struct{}{})
	return er
}

func recursiveExtensionsFromFile(er *dynamic.ExtensionRegistry, fd *desc.FileDescriptor, seen map[string]struct{}) {
	if _, ok := seen[fd.GetName()]; ok {
		return
	}
	seen[fd.GetName()] = struct{}{}
	er.AddExtensionsFromFile(fd)
	for _, dep := range fd.GetDependencies() {
		recursiveExtensionsFromFile(er, dep, seen)
	}
}

func getLocalName(pkg, scope, fqn string) string {
	if fqn[0] == '.' {
		fqn = fqn[1:]
	}
	if len(scope) > 0 && scope[len(scope)-1] != '.' {
		scope = scope + "."
	}
	for scope != "" {
		if strings.HasPrefix(fqn, scope) {
			return fqn[len(scope):]
		}
		if scope == pkg+"." {
			break
		}
		pos := strings.LastIndex(scope[:len(scope)-1], ".")
		scope = scope[:pos+1]
	}
	return fqn
}

func (p *Printer) printMessage(md *desc.MessageDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "message ")
		nameSi := sourceInfo.Get(append(path, message_nameTag))
		p.printElementString(nameSi, w, indent, md.GetName())
		fmt.Fprintln(w, "{")

		p.printMessageBody(md, mf, w, sourceInfo, path, indent+1)
		p.indent(w, indent)
		fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printMessageBody(md *desc.MessageDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, path []int32, indent int) {
	hadOptions := p.printOptionsLong(md.GetOptions(), mf, w, indent)

	skip := map[interface{}]bool{}

	elements := elementAddrs{dsc: md}
	for i := range md.AsDescriptorProto().GetReservedRange() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: message_reservedRangeTag, elementIndex: i})
	}
	for i := range md.AsDescriptorProto().GetReservedName() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: message_reservedNameTag, elementIndex: i})
	}
	for i := range md.AsDescriptorProto().GetExtensionRange() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: message_extensionRangeTag, elementIndex: i})
	}
	for i, fld := range md.GetFields() {
		if fld.IsMap() || fld.GetType() == descriptor.FieldDescriptorProto_TYPE_GROUP {
			// we don't emit nested messages for map types or groups since
			// they get special treatment
			skip[fld.GetMessageType()] = true
		}
		elements.addrs = append(elements.addrs, elementAddr{elementType: message_fieldsTag, elementIndex: i})
	}
	for i := range md.GetNestedMessageTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: message_nestedMessagesTag, elementIndex: i})
	}
	for i := range md.GetNestedEnumTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: message_enumsTag, elementIndex: i})
	}
	for i := range md.GetNestedExtensions() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: message_extensionsTag, elementIndex: i})
	}

	if p.SortElements {
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

	pkg := md.GetFile().GetPackage()
	scope := md.GetFullyQualifiedName()

	var ext *desc.FieldDescriptor
	var extSi *descriptor.SourceCodeInfo_Location
	for i, el := range elements.addrs {
		if i == 0 && hadOptions {
			fmt.Fprintln(w)
		}

		d := elements.at(el)
		if skip[d] {
			// skip this element
			continue
		}

		childPath := append(path, el.elementType, int32(el.elementIndex))
		if el.elementType == message_extensionsTag {
			// extension
			fld := d.(*desc.FieldDescriptor)
			if ext == nil || ext.GetOwner() != fld.GetOwner() {
				// need to open a new extend block
				if ext != nil {
					// close preceding extend block
					p.indent(w, indent)
					fmt.Fprintln(w, "}")
					p.printTrailingComments(extSi, w, indent)
				}
				if i > 0 {
					fmt.Fprintln(w)
				}

				ext = fld
				extSi = sourceInfo.Get(childPath)
				p.printLeadingComments(extSi, w, indent)

				p.indent(w, indent)
				fmt.Fprint(w, "extend ")
				extNameSi := sourceInfo.Get(append(childPath, field_extendeeTag))
				p.printElementString(extNameSi, w, indent, getLocalName(pkg, scope, fld.GetOwner().GetFullyQualifiedName()))
				fmt.Fprintln(w, "{")
			} else {
				fmt.Fprintln(w)
			}
			p.printField(fld, mf, w, sourceInfo, childPath, scope, indent+1)
		} else {
			if ext != nil {
				// close preceding extend block
				p.indent(w, indent)
				fmt.Fprintln(w, "}")
				p.printTrailingComments(extSi, w, indent)
				ext = nil
				extSi = nil
			}

			if i > 0 {
				fmt.Fprintln(w)
			}

			switch d := d.(type) {
			case *desc.FieldDescriptor:
				ood := d.GetOneOf()
				if ood == nil {
					p.printField(d, mf, w, sourceInfo, childPath, scope, indent)
				} else if !skip[ood] {
					// print the one-of, including all of its fields
					oopath := append(path, message_oneOfsTag, d.AsFieldDescriptorProto().GetOneofIndex())
					oosi := sourceInfo.Get(oopath)
					p.printElement(oosi, w, indent, func(w *printer) {

						p.indent(w, indent)
						fmt.Fprint(w, "oneof ")
						extNameSi := sourceInfo.Get(append(oopath, oneof_nameTag))
						p.printElementString(extNameSi, w, indent, ood.GetName())
						fmt.Fprintln(w, "{")

						count := len(ood.GetChoices())
						for idx := i; count > 0 && idx < len(elements.addrs); idx++ {
							if idx > i {
								fmt.Fprintln(w)
							}
							el := elements.addrs[idx]
							d := elements.at(el)
							if fld, ok := d.(*desc.FieldDescriptor); ok && !fld.IsExtension() && fld.GetOneOf() == ood {
								childPath := append(path, el.elementType, int32(el.elementIndex))
								p.printField(fld, mf, w, sourceInfo, childPath, scope, indent+1)
								count--
							}
						}

						p.indent(w, indent)
						fmt.Fprintln(w, "}")
					})
					skip[ood] = true
				}
			case *desc.MessageDescriptor:
				p.printMessage(d, mf, w, sourceInfo, childPath, indent)
			case *desc.EnumDescriptor:
				p.printEnum(d, mf, w, sourceInfo, childPath, indent)
			case *descriptor.DescriptorProto_ExtensionRange:
				// collapse ranges into a single "extensions" block
				ranges := []*descriptor.DescriptorProto_ExtensionRange{d}
				addrs := []elementAddr{el}
				for idx := i + 1; idx < len(elements.addrs); idx++ {
					elnext := elements.addrs[idx]
					if elnext.elementType != el.elementType {
						break
					}
					extr := elements.at(elnext).(*descriptor.DescriptorProto_ExtensionRange)
					if !areEqual(d.Options, extr.Options, mf) {
						break
					}
					ranges = append(ranges, extr)
					addrs = append(addrs, elnext)
					skip[extr] = true
				}
				p.printExtensionRanges(ranges, addrs, mf, w, sourceInfo, path, indent)
			case *descriptor.DescriptorProto_ReservedRange:
				// collapse reserved ranges into a single "reserved" block
				ranges := []*descriptor.DescriptorProto_ReservedRange{d}
				addrs := []elementAddr{el}
				for idx := i + 1; idx < len(elements.addrs); idx++ {
					elnext := elements.addrs[idx]
					if elnext.elementType != el.elementType {
						break
					}
					rr := elements.at(elnext).(*descriptor.DescriptorProto_ReservedRange)
					ranges = append(ranges, rr)
					addrs = append(addrs, elnext)
					skip[rr] = true
				}
				p.printReservedRanges(ranges, addrs, w, sourceInfo, path, indent)
			case string: // reserved name
				// collapse reserved names into a single "reserved" block
				names := []string{d}
				addrs := []elementAddr{el}
				for idx := i + 1; idx < len(elements.addrs); idx++ {
					elnext := elements.addrs[idx]
					if elnext.elementType != el.elementType {
						break
					}
					rn := elements.at(elnext).(string)
					names = append(names, rn)
					addrs = append(addrs, elnext)
					skip[rn] = true
				}
				p.printReservedNames(names, addrs, w, sourceInfo, path, indent)
			}
		}
	}

	if ext != nil {
		// close trailing extend block
		p.indent(w, indent)
		fmt.Fprintln(w, "}")
		p.printTrailingComments(extSi, w, 0)
	}
}

func areEqual(a, b proto.Message, mf *dynamic.MessageFactory) bool {
	// proto.Equal doesn't handle unknown extensions very well :(
	// so we convert to a dynamic message (which should know about all extensions via
	// extension registry) and then compare
	return dynamic.MessagesEqual(asDynamicIfPossible(a, mf), asDynamicIfPossible(b, mf))
}

func asDynamicIfPossible(msg proto.Message, mf *dynamic.MessageFactory) proto.Message {
	if dm, ok := msg.(*dynamic.Message); ok {
		return dm
	} else {
		md, err := desc.LoadMessageDescriptorForMessage(msg)
		if err == nil {
			dm := mf.NewDynamicMessage(md)
			if dm.ConvertFrom(msg) == nil {
				return dm
			}
		}
	}
	return msg
}

func (p *Printer) printField(fld *desc.FieldDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, path []int32, scope string, indent int) {
	var groupPath []int32
	var si *descriptor.SourceCodeInfo_Location
	if isGroup(fld) {
		// compute path to group message type
		groupPath = append([]int32(nil), path[:len(path)-2]...)
		var groupMsgIndex int32
		md := fld.GetParent().(*desc.MessageDescriptor)
		for i, nmd := range md.GetNestedMessageTypes() {
			if nmd == fld.GetMessageType() {
				// found it
				groupMsgIndex = int32(i)
				break
			}
		}
		groupPath = append(groupPath, message_nestedMessagesTag, groupMsgIndex)

		// the group message is where the field's comments and position are stored
		si = sourceInfo.Get(groupPath)
	} else {
		si = sourceInfo.Get(path)
	}

	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)
		if shouldEmitLabel(fld) {
			locSi := sourceInfo.Get(append(path, field_labelTag))
			p.printElementString(locSi, w, indent, labelString(fld.GetLabel()))
		}

		if isGroup(fld) {
			fmt.Fprint(w, "group ")

			typeSi := sourceInfo.Get(append(path, field_typeTag))
			p.printElementString(typeSi, w, indent, typeString(fld, scope))
			fmt.Fprint(w, "= ")

			numSi := sourceInfo.Get(append(path, field_numberTag))
			p.printElementString(numSi, w, indent, fmt.Sprintf("%d", fld.GetNumber()))

			fmt.Fprintln(w, "{")
			p.printMessageBody(fld.GetMessageType(), mf, w, sourceInfo, groupPath, indent+1)

			p.indent(w, indent)
			fmt.Fprintln(w, "}")
		} else {
			typeSi := sourceInfo.Get(append(path, field_typeTag))
			p.printElementString(typeSi, w, indent, typeString(fld, scope))

			nameSi := sourceInfo.Get(append(path, field_nameTag))
			p.printElementString(nameSi, w, indent, fld.GetName())
			fmt.Fprint(w, "= ")

			numSi := sourceInfo.Get(append(path, field_numberTag))
			p.printElementString(numSi, w, indent, fmt.Sprintf("%d", fld.GetNumber()))

			var extraOptions []string
			if fld.AsFieldDescriptorProto().DefaultValue != nil {
				defVal := fld.AsFieldDescriptorProto().GetDefaultValue()
				if fld.GetType() == descriptor.FieldDescriptorProto_TYPE_STRING {
					defVal = quotedString(defVal)
				} else if fld.GetType() == descriptor.FieldDescriptorProto_TYPE_BYTES {
					// bytes are a weird hybrid: they do not have enclosing quotes
					// but they are already escaped
					defVal = fmt.Sprintf(`"%s"`, defVal)
				}
				extraOptions = append(extraOptions, "default", defVal)
			}

			jsn := fld.AsFieldDescriptorProto().GetJsonName()
			if jsn != "" && jsn != internal.JsonName(fld.GetName()) {
				extraOptions = append(extraOptions, "json_name", quotedString(jsn))
			}

			p.printOptionsShortWithExtras(fld.GetOptions(), mf, w, indent, extraOptions)

			fmt.Fprintln(w, ";")
		}
	})
}

func shouldEmitLabel(fld *desc.FieldDescriptor) bool {
	return !fld.IsMap() && (fld.GetLabel() != descriptor.FieldDescriptorProto_LABEL_OPTIONAL || !fld.GetFile().IsProto3())
}

func labelString(lbl descriptor.FieldDescriptorProto_Label) string {
	switch lbl {
	case descriptor.FieldDescriptorProto_LABEL_OPTIONAL:
		return "optional"
	case descriptor.FieldDescriptorProto_LABEL_REQUIRED:
		return "required"
	case descriptor.FieldDescriptorProto_LABEL_REPEATED:
		return "repeated"
	}
	panic(fmt.Sprintf("invalid label: %v", lbl))
}

func typeString(fld *desc.FieldDescriptor, scope string) string {
	if fld.IsMap() {
		return fmt.Sprintf("map<%s, %s>", typeString(fld.GetMapKeyType(), scope), typeString(fld.GetMapValueType(), scope))
	}
	switch fld.GetType() {
	case descriptor.FieldDescriptorProto_TYPE_INT32:
		return "int32"
	case descriptor.FieldDescriptorProto_TYPE_INT64:
		return "int64"
	case descriptor.FieldDescriptorProto_TYPE_UINT32:
		return "uint32"
	case descriptor.FieldDescriptorProto_TYPE_UINT64:
		return "uint64"
	case descriptor.FieldDescriptorProto_TYPE_SINT32:
		return "sint32"
	case descriptor.FieldDescriptorProto_TYPE_SINT64:
		return "sint64"
	case descriptor.FieldDescriptorProto_TYPE_FIXED32:
		return "fixed32"
	case descriptor.FieldDescriptorProto_TYPE_FIXED64:
		return "fixed64"
	case descriptor.FieldDescriptorProto_TYPE_SFIXED32:
		return "sfixed32"
	case descriptor.FieldDescriptorProto_TYPE_SFIXED64:
		return "sfixed64"
	case descriptor.FieldDescriptorProto_TYPE_FLOAT:
		return "float"
	case descriptor.FieldDescriptorProto_TYPE_DOUBLE:
		return "double"
	case descriptor.FieldDescriptorProto_TYPE_BOOL:
		return "bool"
	case descriptor.FieldDescriptorProto_TYPE_STRING:
		return "string"
	case descriptor.FieldDescriptorProto_TYPE_BYTES:
		return "bytes"
	case descriptor.FieldDescriptorProto_TYPE_ENUM:
		return getLocalName(fld.GetFile().GetPackage(), scope, fld.GetEnumType().GetFullyQualifiedName())
	case descriptor.FieldDescriptorProto_TYPE_MESSAGE:
		return getLocalName(fld.GetFile().GetPackage(), scope, fld.GetMessageType().GetFullyQualifiedName())
	case descriptor.FieldDescriptorProto_TYPE_GROUP:
		return fld.GetMessageType().GetName()
	}
	panic(fmt.Sprintf("invalid type: %v", fld.GetType()))
}

func isGroup(fld *desc.FieldDescriptor) bool {
	return fld.GetType() == descriptor.FieldDescriptorProto_TYPE_GROUP
}

func (p *Printer) printExtensionRanges(ranges []*descriptor.DescriptorProto_ExtensionRange, addrs []elementAddr, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, parentPath []int32, indent int) {
	p.indent(w, indent)
	fmt.Fprint(w, "extensions ")

	var opts *descriptor.ExtensionRangeOptions
	first := true
	for i, extr := range ranges {
		if first {
			first = false
		} else {
			fmt.Fprint(w, ", ")
		}
		el := addrs[i]
		opts = extr.Options
		si := sourceInfo.Get(append(parentPath, el.elementType, int32(el.elementIndex)))
		p.printElement(si, w, inline(indent), func(w *printer) {
			if extr.GetStart() == extr.GetEnd()-1 {
				fmt.Fprintf(w, "%d ", extr.GetStart())
			} else if extr.GetEnd()-1 == internal.MaxTag {
				fmt.Fprintf(w, "%d to max ", extr.GetStart())
			} else {
				fmt.Fprintf(w, "%d to %d ", extr.GetStart(), extr.GetEnd()-1)
			}
		})
	}
	p.printOptionsShort(opts, mf, w, indent)

	fmt.Fprintln(w, ";")
}

func (p *Printer) printReservedRanges(ranges []*descriptor.DescriptorProto_ReservedRange, addrs []elementAddr, w *printer, sourceInfo sourceInfoMap, parentPath []int32, indent int) {
	p.indent(w, indent)
	fmt.Fprint(w, "reserved ")

	first := true
	for i, extr := range ranges {
		if first {
			first = false
		} else {
			fmt.Fprint(w, ", ")
		}
		el := addrs[i]
		si := sourceInfo.Get(append(parentPath, el.elementType, int32(el.elementIndex)))
		p.printElement(si, w, inline(indent), func(w *printer) {
			if extr.GetStart() == extr.GetEnd()-1 {
				fmt.Fprintf(w, "%d ", extr.GetStart())
			} else if extr.GetEnd()-1 == internal.MaxTag {
				fmt.Fprintf(w, "%d to max ", extr.GetStart())
			} else {
				fmt.Fprintf(w, "%d to %d ", extr.GetStart(), extr.GetEnd()-1)
			}
		})
	}

	fmt.Fprintln(w, ";")
}

func (p *Printer) printReservedNames(names []string, addrs []elementAddr, w *printer, sourceInfo sourceInfoMap, parentPath []int32, indent int) {
	p.indent(w, indent)
	fmt.Fprint(w, "reserved ")

	first := true
	for i, name := range names {
		if first {
			first = false
		} else {
			fmt.Fprint(w, ", ")
		}
		el := addrs[i]
		si := sourceInfo.Get(append(parentPath, el.elementType, int32(el.elementIndex)))
		p.printElementString(si, w, indent, quotedString(name))
	}

	fmt.Fprintln(w, ";")
}

func (p *Printer) printEnum(ed *desc.EnumDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "enum ")
		nameSi := sourceInfo.Get(append(path, enum_nameTag))
		p.printElementString(nameSi, w, indent, ed.GetName())
		fmt.Fprintln(w, "{")

		indent++
		hadOptions := p.printOptionsLong(ed.GetOptions(), mf, w, indent)

		elements := elementAddrs{dsc: ed}
		for i := range ed.GetValues() {
			elements.addrs = append(elements.addrs, elementAddr{elementType: enum_valuesTag, elementIndex: i})
		}

		if p.SortElements {
			// canonical sorted order
			sort.Stable(elements)
		} else {
			// use source order (per location information in SourceCodeInfo); or
			// if that isn't present use declaration order, but grouped by type
			sort.Stable(elementSrcOrder{
				elementAddrs: elements,
				sourceInfo:   sourceInfo,
			})
		}

		for i, el := range elements.addrs {
			if i > 0 || hadOptions {
				fmt.Fprintln(w)
			}

			d := elements.at(el).(*desc.EnumValueDescriptor)
			childPath := append(path, el.elementType, int32(el.elementIndex))

			p.printEnumValue(d, mf, w, sourceInfo, childPath, indent)
		}

		p.indent(w, indent-1)
		fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printEnumValue(evd *desc.EnumValueDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		nameSi := sourceInfo.Get(append(path, enumVal_nameTag))
		p.printElementString(nameSi, w, indent, evd.GetName())
		fmt.Fprint(w, "= ")

		numSi := sourceInfo.Get(append(path, enumVal_numberTag))
		p.printElementString(numSi, w, indent, fmt.Sprintf("%d", evd.GetNumber()))

		p.printOptionsShort(evd.GetOptions(), mf, w, indent)

		fmt.Fprintln(w, ";")
	})
}

func (p *Printer) printService(sd *desc.ServiceDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "service ")
		nameSi := sourceInfo.Get(append(path, service_nameTag))
		p.printElementString(nameSi, w, indent, sd.GetName())
		fmt.Fprintln(w, "{")

		indent++
		hadOptions := p.printOptionsLong(sd.GetOptions(), mf, w, indent)

		elements := elementAddrs{dsc: sd}
		for i := range sd.GetMethods() {
			elements.addrs = append(elements.addrs, elementAddr{elementType: service_methodsTag, elementIndex: i})
		}

		if p.SortElements {
			// canonical sorted order
			sort.Stable(elements)
		} else {
			// use source order (per location information in SourceCodeInfo); or
			// if that isn't present use declaration order, but grouped by type
			sort.Stable(elementSrcOrder{
				elementAddrs: elements,
				sourceInfo:   sourceInfo,
			})
		}

		for i, el := range elements.addrs {
			if i > 0 || hadOptions {
				fmt.Fprintln(w)
			}

			d := elements.at(el).(*desc.MethodDescriptor)
			childPath := append(path, el.elementType, int32(el.elementIndex))

			p.printMethod(d, mf, w, sourceInfo, childPath, indent)
		}

		p.indent(w, indent-1)
		fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printMethod(mtd *desc.MethodDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo sourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	pkg := mtd.GetFile().GetPackage()
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "rpc ")
		nameSi := sourceInfo.Get(append(path, method_nameTag))
		p.printElementString(nameSi, w, indent, mtd.GetName())

		fmt.Fprint(w, "( ")
		inSi := sourceInfo.Get(append(path, method_inputTag))
		inName := getLocalName(pkg, pkg, mtd.GetInputType().GetFullyQualifiedName())
		if mtd.IsClientStreaming() {
			inName = "stream " + inName
		}
		p.printElementString(inSi, w, indent, inName)

		fmt.Fprint(w, ") returns ( ")

		outSi := sourceInfo.Get(append(path, method_outputTag))
		outName := getLocalName(pkg, pkg, mtd.GetOutputType().GetFullyQualifiedName())
		if mtd.IsServerStreaming() {
			outName = "stream " + outName
		}
		p.printElementString(outSi, w, indent, outName)
		fmt.Fprint(w, ") ")

		if !p.printOptionsLongWrapped(mtd.GetOptions(), mf, w, indent+1) {
			fmt.Fprintln(w, ";")
		}
	})
}

func (p *Printer) printOptionsLong(opts proto.Message, mf *dynamic.MessageFactory, w *printer, indent int) bool {
	return p.printOptions(opts, mf, w, indent, func(w *printer, indent int, fld *desc.FieldDescriptor, v interface{}, first bool) {
		p.indent(w, indent)
		fmt.Fprint(w, "option ")
		p.printOption(fld, v, w, indent)
		fmt.Fprintln(w, ";")
	})
}

func (p *Printer) printOptionsLongWrapped(opts proto.Message, mf *dynamic.MessageFactory, w *printer, indent int) bool {
	hadOptions := p.printOptions(opts, mf, w, indent, func(w *printer, indent int, fld *desc.FieldDescriptor, v interface{}, first bool) {
		if first {
			fmt.Fprintln(w, "{")
		}
		p.indent(w, indent)
		fmt.Fprint(w, "option ")
		p.printOption(fld, v, w, indent)
		fmt.Fprintln(w, ";")
	})
	if hadOptions {
		p.indent(w, indent-1)
		fmt.Fprintln(w, "}")
	}
	return hadOptions
}

func (p *Printer) printOptionsShort(opts proto.Message, mf *dynamic.MessageFactory, w *printer, indent int) {
	p.printOptionsShortWithExtras(opts, mf, w, indent, nil)
}

func (p *Printer) printOptionsShortWithExtras(opts proto.Message, mf *dynamic.MessageFactory, w *printer, indent int, extras []string) {
	hadOptions := p.printOptions(opts, mf, w, inline(indent), func(w *printer, indent int, fld *desc.FieldDescriptor, v interface{}, first bool) {
		if first {
			fmt.Fprint(w, "[")
			for i := 0; i < len(extras); i += 2 {
				fmt.Fprintf(w, "%s = %s", extras[i], extras[i+1])
			}
		} else {
			fmt.Fprint(w, ", ")
		}
		p.printOption(fld, v, w, indent)
	})
	if hadOptions {
		fmt.Fprint(w, "]")
	} else if len(extras) > 0 {
		fmt.Fprint(w, "[")
		for i := 0; i < len(extras); i += 2 {
			if i > 0 {
				fmt.Fprintf(w, ", ")
			}
			fmt.Fprintf(w, "%s = %s", extras[i], extras[i+1])
		}
		fmt.Fprint(w, "]")
	}
}

func inline(indent int) int {
	if indent < 0 {
		// already inlined
		return indent
	}
	// negative indent means inline; indent 2 stops further in case value wraps
	return -indent - 2
}

func (p *Printer) printOptions(opts proto.Message, mf *dynamic.MessageFactory, w *printer, indent int, fn func(w *printer, indent int, fld *desc.FieldDescriptor, v interface{}, first bool)) bool {
	md, err := desc.LoadMessageDescriptorForMessage(opts)
	if err != nil {
		if w.err != nil {
			w.err = err
		}
		return false
	}
	dm := mf.NewDynamicMessage(md)
	if err = dm.ConvertFrom(opts); err != nil {
		if w.err != nil {
			w.err = fmt.Errorf("failed convert %s to dynamic message: %v", md.GetFullyQualifiedName(), err)
		}
		return false
	}
	count := 0
	for _, fldset := range [][]*desc.FieldDescriptor{md.GetFields(), mf.GetExtensionRegistry().AllExtensionsForType(md.GetFullyQualifiedName())} {
		// make a copy so we can sort it
		fldset = append([]*desc.FieldDescriptor(nil), fldset...)
		sort.Stable(sortedFields(fldset))
		for _, fld := range fldset {
			if dm.HasField(fld) {
				val := dm.GetField(fld)
				switch val := val.(type) {
				case []interface{}:
					for _, e := range val {
						if fld.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
							ev := fld.GetEnumType().FindValueByNumber(e.(int32))
							if ev == nil {
								// have to skip unknown enum values :(
								continue
							}
							e = ev
						}
						fn(w, indent, fld, e, count == 0)
						count++
					}
				case map[interface{}]interface{}:
					for k := range sortKeys(val) {
						v := val[k]
						vf := fld.GetMessageType().FindFieldByNumber(2)
						if vf.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
							ev := vf.GetEnumType().FindValueByNumber(v.(int32))
							if ev == nil {
								// have to skip unknown enum values :(
								continue
							}
							v = ev
						}
						entry := mf.NewDynamicMessage(fld.GetMessageType())
						entry.SetFieldByNumber(1, k)
						entry.SetFieldByNumber(2, v)
						fn(w, indent, fld, entry, count == 0)
						count++
					}
				default:
					if fld.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
						ev := fld.GetEnumType().FindValueByNumber(val.(int32))
						if ev == nil {
							// have to skip unknown enum values :(
							continue
						}
						val = ev
					}
					fn(w, indent, fld, val, count == 0)
					count++
				}
			}
		}
	}

	return count > 0
}

type sortedFields []*desc.FieldDescriptor

func (f sortedFields) Len() int {
	return len(f)
}

func (f sortedFields) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func (f sortedFields) Less(i, j int) bool {
	return f[i].GetNumber() < f[j].GetNumber()
}

func sortKeys(m map[interface{}]interface{}) []interface{} {
	res := make(sortedKeys, len(m))
	i := 0
	for k := range m {
		res[i] = k
		i++
	}
	sort.Sort(res)
	return ([]interface{})(res)
}

type sortedKeys []interface{}

func (k sortedKeys) Len() int {
	return len(k)
}

func (k sortedKeys) Swap(i, j int) {
	k[i], k[j] = k[j], k[i]
}

func (k sortedKeys) Less(i, j int) bool {
	switch i := k[i].(type) {
	case int32:
		return i < k[j].(int32)
	case uint32:
		return i < k[j].(uint32)
	case int64:
		return i < k[j].(int64)
	case uint64:
		return i < k[j].(uint64)
	case string:
		return i < k[j].(string)
	case bool:
		return !i && k[j].(bool)
	default:
		panic(fmt.Sprintf("invalid type for map key: %T", i))
	}
}

func (p *Printer) printOption(optFld *desc.FieldDescriptor, optVal interface{}, w *printer, indent int) {
	if optFld.IsExtension() {
		fmt.Fprintf(w, "(%s) = ", optFld.GetFullyQualifiedName())
	} else {
		fmt.Fprintf(w, "%s = ", optFld.GetName())
	}

	switch optVal.(type) {
	case int32, uint32, int64, uint64:
		fmt.Fprintf(w, "%d", optVal)
	case float32, float64:
		fmt.Fprintf(w, "%f", optVal)
	case string:
		fmt.Fprintf(w, "%s", quotedString(optVal.(string)))
	case bool:
		fmt.Fprintf(w, "%v", optVal)
	case *desc.EnumValueDescriptor:
		fmt.Fprintf(w, "%s", optVal.(*desc.EnumValueDescriptor).GetName())
	case proto.Message:
		// TODO: if value is too long, marshal to text format with indentation to
		// make output prettier (also requires correctly indenting subsequent lines)
		fmt.Fprintf(w, "{ %s }", proto.MarshalTextString(optVal.(proto.Message)))
	default:
		panic(fmt.Sprintf("unknown type of value %T for field %s", optVal, optFld.GetFullyQualifiedName()))
	}
}

// quotedString implements the text format for string literals for protocol
// buffers. This form is also acceptable for string literals in option values
// by the protocol buffer compiler, protoc.
func quotedString(s string) string {
	var b bytes.Buffer
	// use WriteByte here to get any needed indent
	b.WriteByte('"')
	// Loop over the bytes, not the runes.
	for i := 0; i < len(s); i++ {
		// Divergence from C++: we don't escape apostrophes.
		// There's no need to escape them, and the C++ parser
		// copes with a naked apostrophe.
		switch c := s[i]; c {
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '"':
			b.WriteString("\\")
		case '\\':
			b.WriteString("\\\\")
		default:
			if c >= 0x20 && c < 0x7f {
				b.WriteByte(c)
			} else {
				fmt.Fprintf(&b, "\\%03o", c)
			}
		}
	}
	b.WriteByte('"')

	return b.String()
}

type elementAddr struct {
	elementType  int32
	elementIndex int
}

type elementAddrs struct {
	addrs []elementAddr
	dsc   desc.Descriptor
}

func (a elementAddrs) Len() int {
	return len(a.addrs)
}

func (a elementAddrs) Less(i, j int) bool {
	if a.addrs[i].elementType == a.addrs[j].elementType {
		di := a.at(a.addrs[i])
		dj := a.at(a.addrs[j])

		if fi, ok := di.(*desc.FieldDescriptor); ok {
			// fields are ordered by tag number
			fj := dj.(*desc.FieldDescriptor)
			// regular fields before extensions; extensions grouped by extendee
			if !fi.IsExtension() && fj.IsExtension() {
				return true
			} else if fi.IsExtension() && !fj.IsExtension() {
				return false
			} else if fi.IsExtension() && fj.IsExtension() {
				if fi.GetOwner() != fj.GetOwner() {
					return fi.GetOwner().GetFullyQualifiedName() < fj.GetOwner().GetFullyQualifiedName()
				}
			}
			return fi.GetNumber() < fj.GetNumber()
		}

		if evi, ok := di.(*desc.EnumValueDescriptor); ok {
			// enum values ordered by number then name
			evj := dj.(*desc.EnumValueDescriptor)
			if evi.GetNumber() == evj.GetNumber() {
				return evi.GetName() < evj.GetName()
			}
			return evi.GetNumber() < evj.GetNumber()
		}
		if exr, ok := di.(*descriptor.DescriptorProto_ExtensionRange); ok {
			// extension ranges ordered by tag
			return exr.GetStart() < dj.(*descriptor.DescriptorProto_ExtensionRange).GetStart()
		}
		if rr, ok := di.(*descriptor.DescriptorProto_ReservedRange); ok {
			// reserved ranges ordered by tag, too
			return rr.GetStart() < dj.(*descriptor.DescriptorProto_ReservedRange).GetStart()
		}
		if rn, ok := di.(string); ok {
			// reserved names lexically sorted
			return rn < dj.(string)
		}

		// all other descriptors ordered by name
		return di.(desc.Descriptor).GetName() < dj.(desc.Descriptor).GetName()
	}

	return a.addrs[i].elementType < a.addrs[j].elementType
}

func (a elementAddrs) Swap(i, j int) {
	a.addrs[i], a.addrs[j] = a.addrs[j], a.addrs[i]
}

func (a elementAddrs) at(addr elementAddr) interface{} {
	switch dsc := a.dsc.(type) {
	case *desc.FileDescriptor:
		switch addr.elementType {
		case file_messagesTag:
			return dsc.GetMessageTypes()[addr.elementIndex]
		case file_enumsTag:
			return dsc.GetEnumTypes()[addr.elementIndex]
		case file_servicesTag:
			return dsc.GetServices()[addr.elementIndex]
		case file_extensionsTag:
			return dsc.GetExtensions()[addr.elementIndex]
		}
	case *desc.MessageDescriptor:
		switch addr.elementType {
		case message_fieldsTag:
			return dsc.GetFields()[addr.elementIndex]
		case message_nestedMessagesTag:
			return dsc.GetNestedMessageTypes()[addr.elementIndex]
		case message_enumsTag:
			return dsc.GetNestedEnumTypes()[addr.elementIndex]
		case message_extensionsTag:
			return dsc.GetNestedExtensions()[addr.elementIndex]
		case message_extensionRangeTag:
			return dsc.AsDescriptorProto().GetExtensionRange()[addr.elementIndex]
		case message_reservedRangeTag:
			return dsc.AsDescriptorProto().GetReservedRange()[addr.elementIndex]
		case message_reservedNameTag:
			return dsc.AsDescriptorProto().GetReservedName()[addr.elementIndex]
		}
	case *desc.EnumDescriptor:
		// TODO: reserved numbers and tags
		if addr.elementType == enum_valuesTag {
			return dsc.GetValues()[addr.elementIndex]
		}
	case *desc.ServiceDescriptor:
		if addr.elementType == service_methodsTag {
			return dsc.GetMethods()[addr.elementIndex]
		}
	}

	panic(fmt.Sprintf("location for unknown field %d of %T", addr.elementType, a.dsc))
}

type elementSrcOrder struct {
	elementAddrs
	sourceInfo sourceInfoMap
	prefix     []int32
}

func (a elementSrcOrder) Less(i, j int) bool {
	si := a.sourceInfo.Get(append(a.prefix, a.addrs[i].elementType, int32(a.addrs[i].elementIndex)))
	sj := a.sourceInfo.Get(append(a.prefix, a.addrs[j].elementType, int32(a.addrs[j].elementIndex)))
	if si != nil && sj == nil {
		// known elements before unknown ones
		return true
	} else if si == nil || sj == nil {
		// let stable sort keep unknown elements in same relative order
		return false
	}
	for idx := 0; idx < len(sj.Span); idx++ {
		if idx >= len(si.Span) {
			return true
		}
		if si.Span[idx] < sj.Span[idx] {
			return true
		}
		if si.Span[idx] > sj.Span[idx] {
			return false
		}
	}
	return false
}

func (p *Printer) printElement(si *descriptor.SourceCodeInfo_Location, w *printer, indent int, el func(*printer)) {
	if si != nil {
		p.printLeadingComments(si, w, indent)
	}
	el(w)
	if si != nil {
		p.printTrailingComments(si, w, indent)
	}
}

func (p *Printer) printElementString(si *descriptor.SourceCodeInfo_Location, w *printer, indent int, str string) {
	p.printElement(si, w, inline(indent), func(w *printer) {
		fmt.Fprintf(w, "%s ", str)
	})
}

func (p *Printer) printLeadingComments(si *descriptor.SourceCodeInfo_Location, w io.Writer, indent int) bool {
	endsInNewLine := false
	// we skip detached comments if we are sorting elements since, after re-ordering elements,
	// the comments could end up in a location which makes them confusing or misleading
	if !p.SortElements {
		for _, c := range si.GetLeadingDetachedComments() {
			if p.printComment(c, w, indent) {
				// if comment ended in newline, add another newline to separate
				// this comment from the next
				fmt.Fprintln(w)
				endsInNewLine = true
			} else if indent < 0 {
				// comment did not end in newline and we are trying to inline?
				// just add a space to separate this comment from what follows
				fmt.Fprint(w, " ")
				endsInNewLine = false
			} else {
				// comment did not end in newline and we are *not* trying to inline?
				// add newline to end of comment and add another to separate this
				// comment from what follows
				fmt.Fprintln(w)
				fmt.Fprintln(w)
				endsInNewLine = true
			}
		}
	}

	if si.GetLeadingComments() != "" {
		endsInNewLine = p.printComment(si.GetLeadingComments(), w, indent)
		if !endsInNewLine && indent >= 0 {
			// leading comment didn't end with newline but needs one
			// (because we're *not* inlining)
			fmt.Fprintln(w)
			endsInNewLine = true
		}
	}

	return endsInNewLine
}

func (p *Printer) printTrailingComments(si *descriptor.SourceCodeInfo_Location, w io.Writer, indent int) {
	if si.GetTrailingComments() != "" {
		if !p.printComment(si.GetTrailingComments(), w, indent) && indent >= 0 {
			// trailing comment didn't end with newline but needs one
			// (because we're *not* inlining)
			fmt.Fprintln(w)
		} else if indent < 0 {
			fmt.Fprint(w, " ")
		}
	}
}

func (p *Printer) printComment(comments string, w io.Writer, indent int) bool {
	if comments == "" {
		return false
	}

	var multiLine bool
	if indent == -1 {
		multiLine = true
	} else {
		multiLine = p.PreferMultiLineStyleComments
	}
	if strings.Contains(comments, "*/") {
		// can't emit '*/' in a multi-line style comment
		multiLine = false
	}

	lines := strings.Split(comments, "\n")

	// first, remove leading and trailing blank lines
	if lines[0] == "" {
		lines = lines[1:]
	}
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return false
	}

	if len(lines) == 1 && multiLine {
		p.indent(w, indent)
		line := lines[0]
		if line[0] == ' ' && line[len(line)-1] != ' ' {
			// add trailing space for symmetry
			line += " "
		}
		fmt.Fprintf(w, "/*%s*/", line)
		if indent >= 0 {
			fmt.Fprintln(w)
			return true
		}
		return false
	}

	if multiLine {
		// multi-line style comments that actually span multiple lines
		// get a blank line before and after so that comment renders nicely
		lines = append(lines, "", "")
		copy(lines[1:], lines)
		lines[0] = ""
	}

	for i, l := range lines {
		p.maybeIndent(w, indent, i > 0)
		if multiLine {
			if i == 0 {
				// first line
				fmt.Fprintf(w, "/*%s\n", l)
			} else if i == len(lines)-1 {
				// last line
				if l == "" {
					fmt.Fprint(w, " */")
				} else {
					fmt.Fprintf(w, " *%s*/", l)
				}
				if indent >= 0 {
					fmt.Fprintln(w)
				}
			} else {
				fmt.Fprintf(w, " *%s\n", l)
			}
		} else {
			fmt.Fprintf(w, "//%s\n", l)
		}
	}

	// single-line comments always end in newline; multi-line comments only
	// end in newline for non-negative (e.g. non-inlined) indentation
	return !multiLine || indent >= 0
}

func (p *Printer) indent(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Fprint(w, p.Indent)
	}
}

func (p *Printer) maybeIndent(w io.Writer, indent int, requireIndent bool) {
	if indent < 0 && requireIndent {
		p.indent(w, -indent)
	} else {
		p.indent(w, indent)
	}
}

type printer struct {
	io.Writer
	err   error
	space bool
}

func (w *printer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if w.space {
		// skip any trailing space if the following
		// character is semicolon or comma
		if p[0] != ';' && p[0] != ',' {
			_, err := w.Writer.Write([]byte{' '})
			if err != nil {
				w.err = err
				return 0, err
			}
		}
		w.space = false
	}

	if p[len(p)-1] == ' ' {
		w.space = true
		p = p[:len(p)-1]
	}

	num, err := w.Writer.Write(p)
	if err != nil {
		w.err = err
	} else if w.space {
		// pretend space was written
		num++
	}
	return num, err
}

type sourceInfoMap map[interface{}]*descriptor.SourceCodeInfo_Location

func (m sourceInfoMap) Get(path []int32) *descriptor.SourceCodeInfo_Location {
	return m[asMapKey(path)]
}

func (m sourceInfoMap) Put(path []int32, loc *descriptor.SourceCodeInfo_Location) {
	m[asMapKey(path)] = loc
}

func asMapKey(slice []int32) interface{} {
	// NB: arrays should be usable as map keys, but this does not
	// work due to a bug: https://github.com/golang/go/issues/22605
	//rv := reflect.ValueOf(slice)
	//arrayType := reflect.ArrayOf(rv.Len(), rv.Type().Elem())
	//array := reflect.New(arrayType).Elem()
	//reflect.Copy(array, rv)
	//return array.Interface()

	b := make([]byte, len(slice)*4)
	for i, s := range slice {
		j := i * 4
		b[j] = byte(s)
		b[j+1] = byte(s >> 8)
		b[j+2] = byte(s >> 16)
		b[j+3] = byte(s >> 24)
	}
	return string(b)
}

func createSourceInfoMap(fd *descriptor.FileDescriptorProto) sourceInfoMap {
	res := sourceInfoMap{}
	for _, l := range fd.GetSourceCodeInfo().GetLocation() {
		res.Put(l.Path, l)
	}
	return res
}
