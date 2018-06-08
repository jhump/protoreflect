package protoprint

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/internal"
	"github.com/jhump/protoreflect/dynamic"
)

// Printer knows how to format file descriptors as proto source code. Its fields
// provide some control over how the resulting source file is constructed and
// formatted.
type Printer struct {
	// If true, comments are rendered using "/*" style comments. Otherwise, they
	// are printed using "//" style line comments.
	PreferMultiLineStyleComments bool
	// If true, elements are sorted into a canonical order.
	//
	// The canonical order for elements in a file follows:
	//  1. Syntax
	//  2. Options (sorted by name, standard options before custom options)
	//  3. Package
	//  4. Imports (sorted lexically)
	//  5. Messages (sorted by name)
	//  6. Enums (sorted by name)
	//  7. Services (sorted by name)
	//  8. Extensions (grouped by extendee, sorted by extendee+tag)
	//
	// The canonical order of elements in a message follows:
	//  1. Options (sorted by name, standard options before custom options)
	//  2. Fields and One-Ofs (sorted by tag; one-ofs interleaved based on the
	//     minimum tag therein)
	//  3. Nested Messages (sorted by name)
	//  4. Nested Enums (sorted by name)
	//  5. Extension ranges (sorted by starting tag number)
	//  6. Nested Extensions (grouped by extendee, sorted by extendee+tag)
	//  7. Reserved ranges (sorted by starting tag number)
	//  8. Reserved names (sorted lexically)
	//
	// Methods are sorted within a service by name. Enum values are sorted
	// within an enum first by numeric value then by name.
	SortElements bool
	// The indentation used. Any characters other spaces or tabs will be
	// replaced with spaces. If unset/empty, two spaces will be used.
	Indent string
	// If true, detached comments (between elements) will be ignored.
	OmitDetachedComments bool
}

// PrintProtoFiles prints all of the given file descriptors. The given open
// function is given a file name and is responsible for creating the outputs and
// returning the corresponding writer.
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

// PrintProtosToFileSystem prints all of the given file descriptors to files in
// the given directory. If file names in the given descriptors include path
// information, they will be relative to the given root.
func (p *Printer) PrintProtosToFileSystem(fds []*desc.FileDescriptor, rootDir string) error {
	return p.PrintProtoFiles(fds, func(name string) (io.WriteCloser, error) {
		fullPath := filepath.Join(rootDir, name)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return nil, err
		}
		return os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	})
}

// pkg represents a package name
type pkg string

// imp represents an imported file name
type imp string

// ident represents an identifier
type ident string

// option represents a resolved descriptor option
type option struct {
	name string
	val  interface{}
}

// reservedRange represents a reserved range from a message or enum
type reservedRange struct {
	start, end int32
}

// PrintProtoFile prints the given single file descriptor to the given writer.
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

	er := dynamic.ExtensionRegistry{}
	er.AddExtensionsFromFileRecursively(fd)
	mf := dynamic.NewMessageFactoryWithExtensionRegistry(&er)
	fdp := fd.AsFileDescriptorProto()
	sourceInfo := internal.CreateSourceInfoMap(fdp)
	extendOptionLocations(sourceInfo)
	path := make([]int32, 1)

	opts, err := extractOptions(fd.GetOptions(), mf)
	if err != nil {
		if out.err != nil {
			return out.err
		} else {
			return err
		}
	}

	path[0] = internal.File_packageTag
	sourceInfo.PutIfAbsent(append(path, 0), sourceInfo.Get(path))

	path[0] = internal.File_syntaxTag
	si := sourceInfo.Get(path)
	p.printElement(si, out, 0, func(w *printer) {
		syn := fdp.GetSyntax()
		if syn == "" {
			syn = "proto2"
		}
		fmt.Fprintf(w, "syntax = %q;\n", syn)
	})
	fmt.Fprintln(out)

	elements := elementAddrs{dsc: fd, opts: opts}
	elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.File_optionsTag, -3, opts)...)
	if fdp.Package != nil {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.File_packageTag, elementIndex: 0, order: -2})
	}
	for i := range fd.AsFileDescriptorProto().GetDependency() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.File_dependencyTag, elementIndex: i, order: -1})
	}
	for i := range fd.GetMessageTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.File_messagesTag, elementIndex: i})
	}
	for i := range fd.GetEnumTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.File_enumsTag, elementIndex: i})
	}
	for i := range fd.GetServices() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.File_servicesTag, elementIndex: i})
	}
	for i := range fd.GetExtensions() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.File_extensionsTag, elementIndex: i})
	}

	p.sort(elements, sourceInfo, nil)

	pkgName := fd.GetPackage()

	var ext *desc.FieldDescriptor
	var extSi *descriptor.SourceCodeInfo_Location
	for i, el := range elements.addrs {
		d := elements.at(el)
		path = []int32{el.elementType, int32(el.elementIndex)}
		if el.elementType == internal.File_extensionsTag {
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
				extNameSi := sourceInfo.Get(append(path, internal.Field_extendeeTag))
				p.printElementString(extNameSi, out, 0, getLocalName(pkgName, pkgName, fld.GetOwner().GetFullyQualifiedName()))
				fmt.Fprintln(out, "{")
			} else {
				fmt.Fprintln(out)
			}
			p.printField(fld, mf, out, sourceInfo, path, pkgName, 1)
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
			case pkg:
				si := sourceInfo.Get(path)
				p.printElement(si, out, 0, func(w *printer) {
					fmt.Fprintf(w, "package %s;\n", d)
				})
			case imp:
				si := sourceInfo.Get(path)
				p.printElement(si, out, 0, func(w *printer) {
					fmt.Fprintf(w, "import %q;\n", d)
				})
			case []option:
				p.printOptionsLong(d, out, sourceInfo, path, 0)
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

func (p *Printer) sort(elements elementAddrs, sourceInfo internal.SourceInfoMap, path []int32) {
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

func (p *Printer) printMessage(md *desc.MessageDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "message ")
		nameSi := sourceInfo.Get(append(path, internal.Message_nameTag))
		p.printElementString(nameSi, w, indent, md.GetName())
		fmt.Fprintln(w, "{")

		p.printMessageBody(md, mf, w, sourceInfo, path, indent+1)
		p.indent(w, indent)
		fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printMessageBody(md *desc.MessageDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	opts, err := extractOptions(md.GetOptions(), mf)
	if err != nil {
		if w.err == nil {
			w.err = err
		}
		return
	}

	skip := map[interface{}]bool{}

	elements := elementAddrs{dsc: md, opts: opts}
	elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.Message_optionsTag, -1, opts)...)
	for i := range md.AsDescriptorProto().GetReservedRange() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Message_reservedRangeTag, elementIndex: i})
	}
	for i := range md.AsDescriptorProto().GetReservedName() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Message_reservedNameTag, elementIndex: i})
	}
	for i := range md.AsDescriptorProto().GetExtensionRange() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Message_extensionRangeTag, elementIndex: i})
	}
	for i, fld := range md.GetFields() {
		if fld.IsMap() || fld.GetType() == descriptor.FieldDescriptorProto_TYPE_GROUP {
			// we don't emit nested messages for map types or groups since
			// they get special treatment
			skip[fld.GetMessageType()] = true
		}
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Message_fieldsTag, elementIndex: i})
	}
	for i := range md.GetNestedMessageTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Message_nestedMessagesTag, elementIndex: i})
	}
	for i := range md.GetNestedEnumTypes() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Message_enumsTag, elementIndex: i})
	}
	for i := range md.GetNestedExtensions() {
		elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Message_extensionsTag, elementIndex: i})
	}

	p.sort(elements, sourceInfo, path)

	pkg := md.GetFile().GetPackage()
	scope := md.GetFullyQualifiedName()

	var ext *desc.FieldDescriptor
	var extSi *descriptor.SourceCodeInfo_Location
	for i, el := range elements.addrs {
		d := elements.at(el)
		// skip[d] will panic if d is a slice (which it could be for []option),
		// so just ignore it since we don't try to skip options
		if reflect.TypeOf(d).Kind() != reflect.Slice && skip[d] {
			// skip this element
			continue
		}

		childPath := append(path, el.elementType, int32(el.elementIndex))
		if el.elementType == internal.Message_extensionsTag {
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
				extNameSi := sourceInfo.Get(append(childPath, internal.Field_extendeeTag))
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
			case []option:
				p.printOptionsLong(d, w, sourceInfo, childPath, indent)
			case *desc.FieldDescriptor:
				ood := d.GetOneOf()
				if ood == nil {
					p.printField(d, mf, w, sourceInfo, childPath, scope, indent)
				} else if !skip[ood] {
					// print the one-of, including all of its fields
					p.printOneOf(ood, elements, i, mf, w, sourceInfo, path, indent, d.AsFieldDescriptorProto().GetOneofIndex())
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
				p.printReservedRanges(ranges, false, addrs, w, sourceInfo, path, indent)
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

func (p *Printer) printField(fld *desc.FieldDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, path []int32, scope string, indent int) {
	var groupPath []int32
	var si *descriptor.SourceCodeInfo_Location
	if isGroup(fld) {
		// compute path to group message type
		groupPath = make([]int32, len(path)-2)
		copy(groupPath, path)
		var groupMsgIndex int32
		md := fld.GetParent().(*desc.MessageDescriptor)
		for i, nmd := range md.GetNestedMessageTypes() {
			if nmd == fld.GetMessageType() {
				// found it
				groupMsgIndex = int32(i)
				break
			}
		}
		groupPath = append(groupPath, internal.Message_nestedMessagesTag, groupMsgIndex)

		// the group message is where the field's comments and position are stored
		si = sourceInfo.Get(groupPath)
	} else {
		si = sourceInfo.Get(path)
	}

	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)
		if shouldEmitLabel(fld) {
			locSi := sourceInfo.Get(append(path, internal.Field_labelTag))
			p.printElementString(locSi, w, indent, labelString(fld.GetLabel()))
		}

		if isGroup(fld) {
			fmt.Fprint(w, "group ")

			typeSi := sourceInfo.Get(append(path, internal.Field_typeTag))
			p.printElementString(typeSi, w, indent, typeString(fld, scope))
			fmt.Fprint(w, "= ")

			numSi := sourceInfo.Get(append(path, internal.Field_numberTag))
			p.printElementString(numSi, w, indent, fmt.Sprintf("%d", fld.GetNumber()))

			fmt.Fprintln(w, "{")
			p.printMessageBody(fld.GetMessageType(), mf, w, sourceInfo, groupPath, indent+1)

			p.indent(w, indent)
			fmt.Fprintln(w, "}")
		} else {
			typeSi := sourceInfo.Get(append(path, internal.Field_typeTag))
			p.printElementString(typeSi, w, indent, typeString(fld, scope))

			nameSi := sourceInfo.Get(append(path, internal.Field_nameTag))
			p.printElementString(nameSi, w, indent, fld.GetName())
			fmt.Fprint(w, "= ")

			numSi := sourceInfo.Get(append(path, internal.Field_numberTag))
			p.printElementString(numSi, w, indent, fmt.Sprintf("%d", fld.GetNumber()))

			opts, err := extractOptions(fld.GetOptions(), mf)
			if err != nil {
				if w.err == nil {
					w.err = err
				}
				return
			}

			// we use negative values for "extras" keys so they can't collide
			// with legit option tags

			if !fld.GetFile().IsProto3() && fld.AsFieldDescriptorProto().DefaultValue != nil {
				defVal := fld.GetDefaultValue()
				if fld.GetEnumType() != nil {
					defVal = fld.GetEnumType().FindValueByNumber(defVal.(int32))
				}
				opts[-internal.Field_defaultTag] = []option{{name: "default", val: defVal}}
			}

			jsn := fld.AsFieldDescriptorProto().GetJsonName()
			if jsn != "" && jsn != internal.JsonName(fld.GetName()) {
				opts[-internal.Field_jsonNameTag] = []option{{name: "json_name", val: jsn}}
			}

			elements := elementAddrs{dsc: fld, opts: opts}
			elements.addrs = optionsAsElementAddrs(internal.Field_optionsTag, 0, opts)
			p.sort(elements, sourceInfo, path)
			p.printOptionElementsShort(elements, w, sourceInfo, path, indent)

			fmt.Fprintln(w, ";")
		}
	})
}

func shouldEmitLabel(fld *desc.FieldDescriptor) bool {
	return !fld.IsMap() && fld.GetOneOf() == nil && (fld.GetLabel() != descriptor.FieldDescriptorProto_LABEL_OPTIONAL || !fld.GetFile().IsProto3())
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

func (p *Printer) printOneOf(ood *desc.OneOfDescriptor, parentElements elementAddrs, startFieldIndex int, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, parentPath []int32, indent int, ooIndex int32) {
	oopath := append(parentPath, internal.Message_oneOfsTag, ooIndex)
	oosi := sourceInfo.Get(oopath)
	p.printElement(oosi, w, indent, func(w *printer) {
		p.indent(w, indent)
		fmt.Fprint(w, "oneof ")
		extNameSi := sourceInfo.Get(append(oopath, internal.OneOf_nameTag))
		p.printElementString(extNameSi, w, indent, ood.GetName())
		fmt.Fprintln(w, "{")

		indent++
		opts, err := extractOptions(ood.GetOptions(), mf)
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

		elements := elementAddrs{dsc: ood, opts: opts}
		elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.OneOf_optionsTag, -1, opts)...)

		count := len(ood.GetChoices())
		for idx := startFieldIndex; count > 0 && idx < len(parentElements.addrs); idx++ {
			el := parentElements.addrs[idx]
			if el.elementType != internal.Message_fieldsTag {
				continue
			}
			if parentElements.at(el).(*desc.FieldDescriptor).GetOneOf() == ood {
				// negative tag indicates that this element is actually a sibling, not a child
				elements.addrs = append(elements.addrs, elementAddr{elementType: -internal.Message_fieldsTag, elementIndex: el.elementIndex})
				count--
			}
		}

		p.sort(elements, sourceInfo, oopath)

		scope := ood.GetOwner().GetFullyQualifiedName()

		for i, el := range elements.addrs {
			if i > 0 {
				fmt.Fprintln(w)
			}

			switch d := elements.at(el).(type) {
			case []option:
				childPath := append(oopath, el.elementType, int32(el.elementIndex))
				p.printOptionsLong(d, w, sourceInfo, childPath, indent)
			case *desc.FieldDescriptor:
				childPath := append(parentPath, -el.elementType, int32(el.elementIndex))
				p.printField(d, mf, w, sourceInfo, childPath, scope, indent)
			}
		}

		p.indent(w, indent-1)
		fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printExtensionRanges(ranges []*descriptor.DescriptorProto_ExtensionRange, addrs []elementAddr, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, parentPath []int32, indent int) {
	p.indent(w, indent)
	fmt.Fprint(w, "extensions ")

	var opts *descriptor.ExtensionRangeOptions
	var elPath []int32
	first := true
	for i, extr := range ranges {
		if first {
			first = false
		} else {
			fmt.Fprint(w, ", ")
		}
		opts = extr.Options
		el := addrs[i]
		elPath = append(parentPath, el.elementType, int32(el.elementIndex))
		si := sourceInfo.Get(elPath)
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
	p.printOptionsShort(ranges[0], opts, mf, internal.ExtensionRange_optionsTag, w, sourceInfo, elPath, indent)

	fmt.Fprintln(w, ";")
}

func (p *Printer) printReservedRanges(ranges []reservedRange, isEnum bool, addrs []elementAddr, w *printer, sourceInfo internal.SourceInfoMap, parentPath []int32, indent int) {
	p.indent(w, indent)
	fmt.Fprint(w, "reserved ")

	first := true
	for i, rr := range ranges {
		if first {
			first = false
		} else {
			fmt.Fprint(w, ", ")
		}
		el := addrs[i]
		si := sourceInfo.Get(append(parentPath, el.elementType, int32(el.elementIndex)))
		p.printElement(si, w, inline(indent), func(w *printer) {
			if rr.start == rr.end {
				fmt.Fprintf(w, "%d ", rr.start)
			} else if (rr.end == internal.MaxTag && !isEnum) ||
				(rr.end == math.MaxInt32 && isEnum) {
				fmt.Fprintf(w, "%d to max ", rr.start)
			} else {
				fmt.Fprintf(w, "%d to %d ", rr.start, rr.end)
			}
		})
	}

	fmt.Fprintln(w, ";")
}

func (p *Printer) printReservedNames(names []string, addrs []elementAddr, w *printer, sourceInfo internal.SourceInfoMap, parentPath []int32, indent int) {
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

func (p *Printer) printEnum(ed *desc.EnumDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "enum ")
		nameSi := sourceInfo.Get(append(path, internal.Enum_nameTag))
		p.printElementString(nameSi, w, indent, ed.GetName())
		fmt.Fprintln(w, "{")

		indent++
		opts, err := extractOptions(ed.GetOptions(), mf)
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

		skip := map[interface{}]bool{}

		elements := elementAddrs{dsc: ed, opts: opts}
		elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.Enum_optionsTag, -1, opts)...)
		for i := range ed.GetValues() {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Enum_valuesTag, elementIndex: i})
		}
		for i := range ed.AsEnumDescriptorProto().GetReservedRange() {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Enum_reservedRangeTag, elementIndex: i})
		}
		for i := range ed.AsEnumDescriptorProto().GetReservedName() {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Enum_reservedNameTag, elementIndex: i})
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
				fmt.Fprintln(w)
			}

			childPath := append(path, el.elementType, int32(el.elementIndex))

			switch d := d.(type) {
			case []option:
				p.printOptionsLong(d, w, sourceInfo, childPath, indent)
			case *desc.EnumValueDescriptor:
				p.printEnumValue(d, mf, w, sourceInfo, childPath, indent)
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
				p.printReservedRanges(ranges, true, addrs, w, sourceInfo, path, indent)
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

		p.indent(w, indent-1)
		fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printEnumValue(evd *desc.EnumValueDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		nameSi := sourceInfo.Get(append(path, internal.EnumVal_nameTag))
		p.printElementString(nameSi, w, indent, evd.GetName())
		fmt.Fprint(w, "= ")

		numSi := sourceInfo.Get(append(path, internal.EnumVal_numberTag))
		p.printElementString(numSi, w, indent, fmt.Sprintf("%d", evd.GetNumber()))

		p.printOptionsShort(evd, evd.GetOptions(), mf, internal.EnumVal_optionsTag, w, sourceInfo, path, indent)

		fmt.Fprintln(w, ";")
	})
}

func (p *Printer) printService(sd *desc.ServiceDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "service ")
		nameSi := sourceInfo.Get(append(path, internal.Service_nameTag))
		p.printElementString(nameSi, w, indent, sd.GetName())
		fmt.Fprintln(w, "{")

		indent++

		opts, err := extractOptions(sd.GetOptions(), mf)
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

		elements := elementAddrs{dsc: sd, opts: opts}
		elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.Service_optionsTag, -1, opts)...)
		for i := range sd.GetMethods() {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.Service_methodsTag, elementIndex: i})
		}

		p.sort(elements, sourceInfo, path)

		for i, el := range elements.addrs {
			if i > 0 {
				fmt.Fprintln(w)
			}

			childPath := append(path, el.elementType, int32(el.elementIndex))

			switch d := elements.at(el).(type) {
			case []option:
				p.printOptionsLong(d, w, sourceInfo, childPath, indent)
			case *desc.MethodDescriptor:
				p.printMethod(d, mf, w, sourceInfo, childPath, indent)
			}
		}

		p.indent(w, indent-1)
		fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printMethod(mtd *desc.MethodDescriptor, mf *dynamic.MessageFactory, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	si := sourceInfo.Get(path)
	pkg := mtd.GetFile().GetPackage()
	p.printElement(si, w, indent, func(w *printer) {
		p.indent(w, indent)

		fmt.Fprint(w, "rpc ")
		nameSi := sourceInfo.Get(append(path, internal.Method_nameTag))
		p.printElementString(nameSi, w, indent, mtd.GetName())

		fmt.Fprint(w, "( ")
		inSi := sourceInfo.Get(append(path, internal.Method_inputTag))
		inName := getLocalName(pkg, pkg, mtd.GetInputType().GetFullyQualifiedName())
		if mtd.IsClientStreaming() {
			inName = "stream " + inName
		}
		p.printElementString(inSi, w, indent, inName)

		fmt.Fprint(w, ") returns ( ")

		outSi := sourceInfo.Get(append(path, internal.Method_outputTag))
		outName := getLocalName(pkg, pkg, mtd.GetOutputType().GetFullyQualifiedName())
		if mtd.IsServerStreaming() {
			outName = "stream " + outName
		}
		p.printElementString(outSi, w, indent, outName)
		fmt.Fprint(w, ") ")

		opts, err := extractOptions(mtd.GetOptions(), mf)
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

		if len(opts) > 0 {
			fmt.Fprintln(w, "{")
			indent++

			elements := elementAddrs{dsc: mtd, opts: opts}
			elements.addrs = optionsAsElementAddrs(internal.Method_optionsTag, 0, opts)
			p.sort(elements, sourceInfo, path)
			path = append(path, internal.Method_optionsTag)

			for i, addr := range elements.addrs {
				if i > 0 {
					fmt.Fprintln(w)
				}
				o := elements.at(addr).([]option)
				p.printOptionsLong(o, w, sourceInfo, path, indent)
			}

			p.indent(w, indent-1)
			fmt.Fprintln(w, "}")
		} else {
			fmt.Fprintln(w, ";")
		}
	})
}

func (p *Printer) printOptionsLong(opts []option, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	p.printOptions(opts, w, indent,
		func(i int32) *descriptor.SourceCodeInfo_Location {
			return sourceInfo.Get(append(path, i))
		},
		func(w *printer, indent int, opt option) {
			p.indent(w, indent)
			fmt.Fprint(w, "option ")
			p.printOption(opt.name, opt.val, w, indent)
			fmt.Fprintln(w, ";")
		})
}

func (p *Printer) printOptionsShort(dsc interface{}, optsMsg proto.Message, mf *dynamic.MessageFactory, optsTag int32, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	opts, err := extractOptions(optsMsg, mf)
	if err != nil {
		if w.err == nil {
			w.err = err
		}
		return
	}

	elements := elementAddrs{dsc: dsc, opts: opts}
	elements.addrs = optionsAsElementAddrs(optsTag, 0, opts)
	p.sort(elements, sourceInfo, path)
	p.printOptionElementsShort(elements, w, sourceInfo, path, indent)
}

func (p *Printer) printOptionElementsShort(addrs elementAddrs, w *printer, sourceInfo internal.SourceInfoMap, path []int32, indent int) {
	if len(addrs.addrs) == 0 {
		return
	}
	first := true
	fmt.Fprint(w, "[")
	for _, addr := range addrs.addrs {
		opts := addrs.at(addr).([]option)
		var childPath []int32
		if addr.elementIndex < 0 {
			// pseudo-option
			childPath = append(path, int32(-addr.elementIndex))
		} else {
			childPath = append(path, addr.elementType, int32(addr.elementIndex))
		}
		p.printOptions(opts, w, inline(indent),
			func(i int32) *descriptor.SourceCodeInfo_Location {
				p := childPath
				if addr.elementIndex >= 0 {
					p = append(p, i)
				}
				return sourceInfo.Get(p)
			},
			func(w *printer, indent int, opt option) {
				if first {
					first = false
				} else {
					fmt.Fprint(w, ", ")
				}
				p.printOption(opt.name, opt.val, w, indent)
				fmt.Fprint(w, " ") // trailing space
			})
	}
	fmt.Fprint(w, "]")
}

func (p *Printer) printOptions(opts []option, w *printer, indent int, siFetch func(i int32) *descriptor.SourceCodeInfo_Location, fn func(w *printer, indent int, opt option)) {
	for i, opt := range opts {
		si := siFetch(int32(i))
		p.printElement(si, w, indent, func(w *printer) {
			fn(w, indent, opt)
		})
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

func (p *Printer) printOption(name string, optVal interface{}, w *printer, indent int) {
	fmt.Fprintf(w, "%s = ", name)

	switch optVal.(type) {
	case int32, uint32, int64, uint64:
		fmt.Fprintf(w, "%d", optVal)
	case float32, float64:
		fmt.Fprintf(w, "%f", optVal)
	case string:
		fmt.Fprintf(w, "%s", quotedString(optVal.(string)))
	case []byte:
		fmt.Fprintf(w, "%s", quotedString(string(optVal.([]byte))))
	case bool:
		fmt.Fprintf(w, "%v", optVal)
	case ident:
		fmt.Fprintf(w, "%s", optVal)
	case *desc.EnumValueDescriptor:
		fmt.Fprintf(w, "%s", optVal.(*desc.EnumValueDescriptor).GetName())
	case proto.Message:
		// TODO: if value is too long, marshal to text format with indentation to
		// make output prettier (also requires correctly indenting subsequent lines)
		fmt.Fprintf(w, "{ %s }", proto.MarshalTextString(optVal.(proto.Message)))
	default:
		panic(fmt.Sprintf("unknown type of value %T for field %s", optVal, name))
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
		internal.File_optionsTag:    edgeKindOption,
		internal.File_messagesTag:   edgeKindMessage,
		internal.File_enumsTag:      edgeKindEnum,
		internal.File_extensionsTag: edgeKindField,
		internal.File_servicesTag:   edgeKindService,
	},
	edgeKindMessage: {
		internal.Message_optionsTag:        edgeKindOption,
		internal.Message_fieldsTag:         edgeKindField,
		internal.Message_oneOfsTag:         edgeKindOneOf,
		internal.Message_nestedMessagesTag: edgeKindMessage,
		internal.Message_enumsTag:          edgeKindEnum,
		internal.Message_extensionsTag:     edgeKindField,
		internal.Message_extensionRangeTag: edgeKindExtensionRange,
		// TODO: reserved range tag
	},
	edgeKindField: {
		internal.Field_optionsTag: edgeKindOption,
	},
	edgeKindOneOf: {
		internal.OneOf_optionsTag: edgeKindOption,
	},
	edgeKindExtensionRange: {
		internal.ExtensionRange_optionsTag: edgeKindOption,
	},
	edgeKindEnum: {
		internal.Enum_optionsTag: edgeKindOption,
		internal.Enum_valuesTag:  edgeKindEnumVal,
	},
	edgeKindEnumVal: {
		internal.EnumVal_optionsTag: edgeKindOption,
	},
	edgeKindService: {
		internal.Service_optionsTag: edgeKindOption,
		internal.Service_methodsTag: edgeKindMethod,
	},
	edgeKindMethod: {
		internal.Method_optionsTag: edgeKindOption,
	},
}

func extendOptionLocations(sc internal.SourceInfoMap) {
	for _, loc := range sc {
		allowed := edges[edgeKindFile]
		for i := 0; i+1 < len(loc.Path); i += 2 {
			nextKind, ok := allowed[loc.Path[i]]
			if !ok {
				break
			}
			if nextKind == edgeKindOption {
				// We've found an option entry. This could be arbitrarily
				// deep (for options that nested messages) or it could end
				// abruptly (for non-repeated fields). But we need a path
				// that is exactly the path-so-far plus two: the option tag
				// and an optional index for repeated option fields (zero
				// for non-repeated option fields). This is used for
				// querying source info when printing options.
				// for sorting elements
				newPath := make([]int32, i+3)
				copy(newPath, loc.Path)
				sc.PutIfAbsent(newPath, loc)
				// we do another path of path-so-far plus two, but with
				// explicit zero index -- just in case this actual path has
				// an extra path element, but it's not an index (e.g the
				// option field is not repeated, but the source info we are
				// looking at indicates a tag of a nested field)
				newPath[len(newPath)-1] = 0
				sc.PutIfAbsent(newPath, loc)
				// finally, we need the path-so-far plus one, just the option
				// tag, for sorting option groups
				newPath = newPath[:len(newPath)-1]
				sc.PutIfAbsent(newPath, loc)

				break
			} else {
				allowed = edges[nextKind]
			}
		}
	}
}

func extractOptions(opts proto.Message, mf *dynamic.MessageFactory) (map[int32][]option, error) {
	md, err := desc.LoadMessageDescriptorForMessage(opts)
	if err != nil {
		return nil, err
	}
	dm := mf.NewDynamicMessage(md)
	if err = dm.ConvertFrom(opts); err != nil {
		return nil, fmt.Errorf("failed convert %s to dynamic message: %v", md.GetFullyQualifiedName(), err)
	}
	options := map[int32][]option{}
	var uninterpreted []interface{}
	for _, fldset := range [][]*desc.FieldDescriptor{md.GetFields(), mf.GetExtensionRegistry().AllExtensionsForType(md.GetFullyQualifiedName())} {
		for _, fld := range fldset {
			if dm.HasField(fld) {
				val := dm.GetField(fld)
				var opts []option
				var name string
				if fld.IsExtension() {
					name = fmt.Sprintf("(%s)", fld.GetFullyQualifiedName())
				} else {
					name = fld.GetName()
				}
				switch val := val.(type) {
				case []interface{}:
					if fld.GetNumber() == internal.UninterpretedOptionsTag {
						// we handle uninterpreted options differently
						uninterpreted = val
						continue
					}

					for _, e := range val {
						if fld.GetType() == descriptor.FieldDescriptorProto_TYPE_ENUM {
							ev := fld.GetEnumType().FindValueByNumber(e.(int32))
							if ev == nil {
								// have to skip unknown enum values :(
								continue
							}
							e = ev
						}
						var name string
						if fld.IsExtension() {
							name = fmt.Sprintf("(%s)", fld.GetFullyQualifiedName())
						} else {
							name = fld.GetName()
						}
						opts = append(opts, option{name: name, val: e})
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
						opts = append(opts, option{name: name, val: entry})
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
					opts = append(opts, option{name: name, val: val})
				}
				if len(opts) > 0 {
					options[fld.GetNumber()] = opts
				}
			}
		}
	}

	// if there are uninterpreted options, add those too
	if len(uninterpreted) > 0 {
		opts := make([]option, len(uninterpreted))
		for i, u := range uninterpreted {
			var unint *descriptor.UninterpretedOption
			if un, ok := u.(*descriptor.UninterpretedOption); ok {
				unint = un
			} else {
				dm := u.(*dynamic.Message)
				unint = &descriptor.UninterpretedOption{}
				if err := dm.ConvertTo(unint); err != nil {
					return nil, err
				}
			}

			var buf bytes.Buffer
			for ni, n := range unint.Name {
				if ni > 0 {
					buf.WriteByte('.')
				}
				if n.GetIsExtension() {
					fmt.Fprintf(&buf, "(%s)", n.GetNamePart())
				} else {
					buf.WriteString(n.GetNamePart())
				}
			}

			var v interface{}
			switch {
			case unint.IdentifierValue != nil:
				v = ident(unint.GetIdentifierValue())
			case unint.StringValue != nil:
				v = string(unint.GetStringValue())
			case unint.DoubleValue != nil:
				v = unint.GetDoubleValue()
			case unint.PositiveIntValue != nil:
				v = unint.GetPositiveIntValue()
			case unint.NegativeIntValue != nil:
				v = unint.GetNegativeIntValue()
			case unint.AggregateValue != nil:
				v = ident(unint.GetAggregateValue())
			}

			opts[i] = option{name: buf.String(), val: v}
		}
		options[internal.UninterpretedOptionsTag] = opts
	}

	return options, nil
}

func optionsAsElementAddrs(optionsTag int32, order int, opts map[int32][]option) []elementAddr {
	var optAddrs []elementAddr
	for tag := range opts {
		optAddrs = append(optAddrs, elementAddr{elementType: optionsTag, elementIndex: int(tag), order: order})
	}
	sort.Sort(optionsByName{addrs: optAddrs, opts: opts})
	return optAddrs
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
	order        int
}

type elementAddrs struct {
	addrs []elementAddr
	dsc   interface{}
	opts  map[int32][]option
}

func (a elementAddrs) Len() int {
	return len(a.addrs)
}

func (a elementAddrs) Less(i, j int) bool {
	// explicit order is considered first
	if a.addrs[i].order < a.addrs[j].order {
		return true
	} else if a.addrs[i].order > a.addrs[j].order {
		return false
	}
	// if order is equal, sort by element type
	if a.addrs[i].elementType < a.addrs[j].elementType {
		return true
	} else if a.addrs[i].elementType > a.addrs[j].elementType {
		return false
	}

	di := a.at(a.addrs[i])
	dj := a.at(a.addrs[j])

	switch vi := di.(type) {
	case *desc.FieldDescriptor:
		// fields are ordered by tag number
		vj := dj.(*desc.FieldDescriptor)
		// regular fields before extensions; extensions grouped by extendee
		if !vi.IsExtension() && vj.IsExtension() {
			return true
		} else if vi.IsExtension() && !vj.IsExtension() {
			return false
		} else if vi.IsExtension() && vj.IsExtension() {
			if vi.GetOwner() != vj.GetOwner() {
				return vi.GetOwner().GetFullyQualifiedName() < vj.GetOwner().GetFullyQualifiedName()
			}
		}
		return vi.GetNumber() < vj.GetNumber()

	case *desc.EnumValueDescriptor:
		// enum values ordered by number then name
		vj := dj.(*desc.EnumValueDescriptor)
		if vi.GetNumber() == vj.GetNumber() {
			return vi.GetName() < vj.GetName()
		}
		return vi.GetNumber() < vj.GetNumber()

	case *descriptor.DescriptorProto_ExtensionRange:
		// extension ranges ordered by tag
		return vi.GetStart() < dj.(*descriptor.DescriptorProto_ExtensionRange).GetStart()

	case reservedRange:
		// reserved ranges ordered by tag, too
		return vi.start < dj.(reservedRange).start

	case string:
		// reserved names lexically sorted
		return vi < dj.(string)

	case pkg:
		// reserved names lexically sorted
		return vi < dj.(pkg)

	case imp:
		// reserved names lexically sorted
		return vi < dj.(imp)

	case []option:
		// options sorted by name, extensions last
		return optionLess(vi, dj.([]option))

	default:
		// all other descriptors ordered by name
		return di.(desc.Descriptor).GetName() < dj.(desc.Descriptor).GetName()
	}
}

func (a elementAddrs) Swap(i, j int) {
	a.addrs[i], a.addrs[j] = a.addrs[j], a.addrs[i]
}

func (a elementAddrs) at(addr elementAddr) interface{} {
	switch dsc := a.dsc.(type) {
	case *desc.FileDescriptor:
		switch addr.elementType {
		case internal.File_packageTag:
			return pkg(dsc.GetPackage())
		case internal.File_dependencyTag:
			return imp(dsc.AsFileDescriptorProto().GetDependency()[addr.elementIndex])
		case internal.File_optionsTag:
			return a.opts[int32(addr.elementIndex)]
		case internal.File_messagesTag:
			return dsc.GetMessageTypes()[addr.elementIndex]
		case internal.File_enumsTag:
			return dsc.GetEnumTypes()[addr.elementIndex]
		case internal.File_servicesTag:
			return dsc.GetServices()[addr.elementIndex]
		case internal.File_extensionsTag:
			return dsc.GetExtensions()[addr.elementIndex]
		}
	case *desc.MessageDescriptor:
		switch addr.elementType {
		case internal.Message_optionsTag:
			return a.opts[int32(addr.elementIndex)]
		case internal.Message_fieldsTag:
			return dsc.GetFields()[addr.elementIndex]
		case internal.Message_nestedMessagesTag:
			return dsc.GetNestedMessageTypes()[addr.elementIndex]
		case internal.Message_enumsTag:
			return dsc.GetNestedEnumTypes()[addr.elementIndex]
		case internal.Message_extensionsTag:
			return dsc.GetNestedExtensions()[addr.elementIndex]
		case internal.Message_extensionRangeTag:
			return dsc.AsDescriptorProto().GetExtensionRange()[addr.elementIndex]
		case internal.Message_reservedRangeTag:
			rng := dsc.AsDescriptorProto().GetReservedRange()[addr.elementIndex]
			return reservedRange{start: rng.GetStart(), end: rng.GetEnd() - 1}
		case internal.Message_reservedNameTag:
			return dsc.AsDescriptorProto().GetReservedName()[addr.elementIndex]
		}
	case *desc.FieldDescriptor:
		if addr.elementType == internal.Field_optionsTag {
			return a.opts[int32(addr.elementIndex)]
		}
	case *desc.OneOfDescriptor:
		switch addr.elementType {
		case internal.OneOf_optionsTag:
			return a.opts[int32(addr.elementIndex)]
		case -internal.Message_fieldsTag:
			return dsc.GetOwner().GetFields()[addr.elementIndex]
		}
	case *desc.EnumDescriptor:
		switch addr.elementType {
		case internal.Enum_optionsTag:
			return a.opts[int32(addr.elementIndex)]
		case internal.Enum_valuesTag:
			return dsc.GetValues()[addr.elementIndex]
		case internal.Enum_reservedRangeTag:
			rng := dsc.AsEnumDescriptorProto().GetReservedRange()[addr.elementIndex]
			return reservedRange{start: rng.GetStart(), end: rng.GetEnd()}
		case internal.Enum_reservedNameTag:
			return dsc.AsEnumDescriptorProto().GetReservedName()[addr.elementIndex]
		}
	case *desc.EnumValueDescriptor:
		if addr.elementType == internal.EnumVal_optionsTag {
			return a.opts[int32(addr.elementIndex)]
		}
	case *desc.ServiceDescriptor:
		switch addr.elementType {
		case internal.Service_optionsTag:
			return a.opts[int32(addr.elementIndex)]
		case internal.Service_methodsTag:
			return dsc.GetMethods()[addr.elementIndex]
		}
	case *desc.MethodDescriptor:
		if addr.elementType == internal.Method_optionsTag {
			return a.opts[int32(addr.elementIndex)]
		}
	case *descriptor.DescriptorProto_ExtensionRange:
		if addr.elementType == internal.ExtensionRange_optionsTag {
			return a.opts[int32(addr.elementIndex)]
		}
	}

	panic(fmt.Sprintf("location for unknown field %d of %T", addr.elementType, a.dsc))
}

type elementSrcOrder struct {
	elementAddrs
	sourceInfo internal.SourceInfoMap
	prefix     []int32
}

func (a elementSrcOrder) Less(i, j int) bool {
	ti := a.addrs[i].elementType
	ei := a.addrs[i].elementIndex

	tj := a.addrs[j].elementType
	ej := a.addrs[j].elementIndex

	var si, sj *descriptor.SourceCodeInfo_Location
	if ei < 0 {
		si = a.sourceInfo.Get(append(a.prefix, -int32(ei)))
	} else if ti < 0 {
		p := make([]int32, len(a.prefix)-2)
		copy(p, a.prefix)
		si = a.sourceInfo.Get(append(p, ti, int32(ei)))
	} else {
		si = a.sourceInfo.Get(append(a.prefix, ti, int32(ei)))
	}
	if ej < 0 {
		sj = a.sourceInfo.Get(append(a.prefix, -int32(ej)))
	} else if tj < 0 {
		p := make([]int32, len(a.prefix)-2)
		copy(p, a.prefix)
		sj = a.sourceInfo.Get(append(p, tj, int32(ej)))
	} else {
		sj = a.sourceInfo.Get(append(a.prefix, tj, int32(ej)))
	}

	if (si == nil) != (sj == nil) {
		// generally, we put unknown elements after known ones;
		// except package and option elements go first

		// i will be unknown and j will be known
		swapped := false
		if si != nil {
			si, sj = sj, si
			// no need to swap ti and tj because we don't use tj anywhere below
			ti = tj
			swapped = true
		}
		switch a.dsc.(type) {
		case *desc.FileDescriptor:
			if ti == internal.File_packageTag || ti == internal.File_optionsTag {
				return !swapped
			}
		case *desc.MessageDescriptor:
			if ti == internal.Message_optionsTag {
				return !swapped
			}
		case *desc.EnumDescriptor:
			if ti == internal.Enum_optionsTag {
				return !swapped
			}
		case *desc.ServiceDescriptor:
			if ti == internal.Service_optionsTag {
				return !swapped
			}
		}
		return swapped

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

type optionsByName struct {
	addrs []elementAddr
	opts  map[int32][]option
}

func (o optionsByName) Len() int {
	return len(o.addrs)
}

func (o optionsByName) Less(i, j int) bool {
	oi := o.opts[int32(o.addrs[i].elementIndex)]
	oj := o.opts[int32(o.addrs[j].elementIndex)]
	return optionLess(oi, oj)
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

func (o optionsByName) Swap(i, j int) {
	o.addrs[i], o.addrs[j] = o.addrs[j], o.addrs[i]
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

	if !p.OmitDetachedComments {
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
		if !endsInNewLine {
			if indent >= 0 {
				// leading comment didn't end with newline but needs one
				// (because we're *not* inlining)
				fmt.Fprintln(w)
				endsInNewLine = true
			} else {
				// space between comment and following element when inlined
				fmt.Fprint(w, " ")
			}
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
	if indent < 0 {
		// use multi-line style when inlining
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
				fmt.Fprintf(w, "/*%s\n", strings.TrimRight(l, " \t"))
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
				fmt.Fprintf(w, " *%s\n", strings.TrimRight(l, " \t"))
			}
		} else {
			fmt.Fprintf(w, "//%s\n", strings.TrimRight(l, " \t"))
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
		// character is semicolon, comma, or close bracket
		if p[0] != ';' && p[0] != ',' && p[0] != ']' {
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
