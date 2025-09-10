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
	"unicode"
	"unicode/utf8"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/internal/register"
	"github.com/jhump/protoreflect/v2/protodescs"
	"github.com/jhump/protoreflect/v2/protomessage"
	"github.com/jhump/protoreflect/v2/sourceloc"
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
	//  2. Package
	//  3. Imports (sorted lexically)
	//  4. Options (sorted by name, standard options before custom options)
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
	// Methods are sorted within a service by name and appear after any service
	// options (which are sorted by name, standard options before custom ones).
	// Enum values are sorted within an enum, first by numeric value then by
	// name, and also appear after any enum options.
	//
	// Options for fields, enum values, and extension ranges are sorted by name,
	// standard options before custom ones.
	SortElements bool

	// The "less" function used to sort elements when printing. It is given two
	// elements, a and b, and should return true if a is "less than" b. In this
	// case, "less than" means that element a should appear earlier in the file
	// than element b.
	//
	// If this field is nil, no custom sorting is done and the SortElements
	// field is consulted to decide how to order the output. If this field is
	// non-nil, the SortElements field is ignored and this function is called to
	// order elements.
	CustomSortFunction func(a, b Element) bool

	// The indentation used. Any characters other than spaces or tabs will be
	// replaced with spaces. If unset/empty, two spaces will be used.
	Indent string

	// A bitmask of comment types to omit. If unset, all comments will be
	// included. Use CommentsAll to not print any comments.
	OmitComments CommentType

	// If true, trailing comments that typically appear on the same line as an
	// element (option, field, enum value, method) will be printed on a separate
	// line instead.
	//
	// So, with this set, you'll get output like so:
	//
	//    // leading comment for field
	//    repeated string names = 1;
	//    // trailing comment
	//
	// If left false, the printer will try to emit trailing comments on the
	// same line instead:
	//
	//    // leading comment for field
	//    repeated string names = 1; // trailing comment
	//
	// If the trailing comment has more than one line, it will automatically be
	// forced to the next line.
	TrailingCommentsOnSeparateLine bool

	// If true, the printed output will eschew any blank lines, which otherwise
	// appear between descriptor elements and comment blocks. Note that if
	// detached comments are being printed, this will cause them to be merged
	// into the subsequent leading comments. Similarly, any element trailing
	// comments will be merged into the subsequent leading comments.
	Compact bool

	// If true, all references to messages, extensions, and enums (such as in
	// options, field types, and method request and response types) will be
	// fully-qualified. When left unset, the referenced elements will contain
	// only as much qualifier as is required.
	//
	// For example, if a message is in the same package as the reference, the
	// simple name can be used. If a message shares some context with the
	// reference, only the unshared context needs to be included. For example:
	//
	//  message Foo {
	//    message Bar {
	//      enum Baz {
	//        ZERO = 0;
	//        ONE = 1;
	//      }
	//    }
	//
	//    // This field shares some context as the enum it references: they are
	//    // both inside of the namespace Foo:
	//    //    field is "Foo.my_baz"
	//    //     enum is "Foo.Bar.Baz"
	//    // So we only need to qualify the reference with the context that they
	//    // do NOT have in common:
	//    Bar.Baz my_baz = 1;
	//  }
	//
	// When printing fully-qualified names, they will be preceded by a dot, to
	// avoid any ambiguity that they might be relative vs. fully-qualified.
	ForceFullyQualifiedNames bool

	// The number of options that trigger short options expressions to be
	// rendered using multiple lines. Short options expressions are those
	// found on fields and enum values, that use brackets ("[" and "]") and
	// comma-separated options. If more options than this are present, they
	// will be expanded to multiple lines (one option per line).
	//
	// If unset (e.g. if zero), a default threshold of 3 is used.
	ShortOptionsExpansionThresholdCount int

	// The length of printed options that trigger short options expressions to
	// be rendered using multiple lines. If the short options contain more than
	// one option and their printed length is longer than this threshold, they
	// will be expanded to multiple lines (one option per line).
	//
	// If unset (e.g. if zero), a default threshold of 50 is used.
	ShortOptionsExpansionThresholdLength int

	// The length of a printed option value message literal that triggers the
	// message literal to be rendered using multiple lines instead of using a
	// compact single-line form. The message must include at least two fields
	// or contain a field that is a nested message to be expanded.
	//
	// This value is further used to decide when to expand individual field
	// values that are nested message literals or array literals (for repeated
	// fields).
	//
	// If unset (e.g. if zero), a default threshold of 50 is used.
	MessageLiteralExpansionThresholdLength int
}

// CommentType is a kind of comments in a proto source file. This can be used
// as a bitmask.
type CommentType int

const (
	// CommentsDetached refers to comments that are not "attached" to any
	// source element. They are attributed to the subsequent element in the
	// file as "detached" comments.
	CommentsDetached CommentType = 1 << iota
	// CommentsTrailing refers to a comment block immediately following an
	// element in the source file. If another element immediately follows
	// the trailing comment, it is instead considered a leading comment for
	// that subsequent element.
	CommentsTrailing
	// CommentsLeading refers to a comment block immediately preceding an
	// element in the source file. For high-level elements (those that have
	// their own descriptor), these are used as doc comments for that element.
	CommentsLeading
	// CommentsTokens refers to any comments (leading, trailing, or detached)
	// on low-level elements in the file. "High-level" elements have their own
	// descriptors, e.g. messages, enums, fields, services, and methods. But
	// comments can appear anywhere (such as around identifiers and keywords,
	// sprinkled inside the declarations of a high-level element). This class
	// of comments are for those extra comments sprinkled into the file.
	CommentsTokens

	// CommentsNonDoc refers to comments that are *not* doc comments. This is a
	// bitwise union of everything other than CommentsLeading. If you configure
	// a printer to omit this, only doc comments on descriptor elements will be
	// included in the printed output.
	CommentsNonDoc = CommentsDetached | CommentsTrailing | CommentsTokens
	// CommentsAll indicates all kinds of comments. If you configure a printer
	// to omit this, no comments will appear in the printed output, even if the
	// input descriptors had source info and comments.
	CommentsAll = -1
)

// PrintProtoFiles prints all the given file descriptors. The given open
// function is given a file name and is responsible for creating the outputs and
// returning the corresponding writer.
func (p *Printer) PrintProtoFiles(fds []protoreflect.FileDescriptor, open func(name string) (io.WriteCloser, error)) error {
	for _, fd := range fds {
		w, err := open(fd.Path())
		if err != nil {
			return fmt.Errorf("failed to open %s: %v", fd.Path(), err)
		}
		err = func() error {
			defer func() {
				_ = w.Close()
			}()
			return p.PrintProtoFile(fd, w)
		}()
		if err != nil {
			return fmt.Errorf("failed to write %s: %v", fd.Path(), err)
		}
	}
	return nil
}

// PrintProtosToFileSystem prints all of the given file descriptors to files in
// the given directory. If file names in the given descriptors include path
// information, they will be relative to the given root.
func (p *Printer) PrintProtosToFileSystem(fds []protoreflect.FileDescriptor, rootDir string) error {
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

// ident represents an identifier
type ident string

// messageVal represents a message value for an option
type messageVal struct {
	// the package and scope in which the option value is defined
	pkg, scope protoreflect.FullName
	// the option value
	msg proto.Message
}

// option represents a resolved descriptor option
type option struct {
	name string
	val  interface{}
}

// reservedRange represents a reserved range from a message or enum
type reservedRange struct {
	start, end int32
}

// extensionRange represents an extension range from a message
type extensionRange struct {
	start, end protoreflect.FieldNumber
	opts       proto.Message
}

// PrintProtoFile prints the given single file descriptor to the given writer.
func (p *Printer) PrintProtoFile(fd protoreflect.FileDescriptor, out io.Writer) error {
	return p.printProto(fd, out)
}

// PrintProtoToString prints the given descriptor and returns the resulting
// string. This can be used to print proto files, but it can also be used to get
// the proto "source form" for any kind of descriptor, which can be a more
// user-friendly way to present descriptors that are intended for human
// consumption.
func (p *Printer) PrintProtoToString(dsc protoreflect.Descriptor) (string, error) {
	var buf bytes.Buffer
	if err := p.printProto(dsc, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (p *Printer) printProto(dsc protoreflect.Descriptor, out io.Writer) error {
	w := newWriter(out)

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

	fd := dsc.ParentFile()
	sourceInfo := extendOptionLocations(fd)

	var reg protoregistry.Types
	register.RegisterTypesVisibleToFile(fd, &reg, true)

	path := findElement(dsc)
	switch d := dsc.(type) {
	case protoreflect.FileDescriptor:
		p.printFile(d, &reg, w, sourceInfo)
	case protoreflect.MessageDescriptor:
		p.printMessage(d, &reg, w, sourceInfo, path, 0)
	case protoreflect.FieldDescriptor:
		var scope protoreflect.FullName
		if md, ok := d.Parent().(protoreflect.MessageDescriptor); ok {
			scope = md.FullName()
		} else {
			scope = d.ParentFile().Package()
		}
		if d.IsExtension() {
			_, _ = fmt.Fprint(w, "extend ")
			extNameSi := sourceInfo.ByPath(append(path, internal.FieldExtendeeTag))
			p.printElementString(extNameSi, w, 0, p.qualifyName(d.ParentFile().Package(), scope, d.ContainingMessage().FullName()))
			_, _ = fmt.Fprintln(w, "{")

			p.printField(d, &reg, w, sourceInfo, path, scope, 1)

			_, _ = fmt.Fprintln(w, "}")
		} else {
			p.printField(d, &reg, w, sourceInfo, path, scope, 0)
		}
	case protoreflect.OneofDescriptor:
		md := d.Parent().(protoreflect.MessageDescriptor)
		elements := elementAddrs{dsc: md}
		fields := md.Fields()
		for i, length := 0, fields.Len(); i < length; i++ {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.MessageFieldsTag, elementIndex: i})
		}
		p.printOneOf(d, elements, 0, &reg, w, sourceInfo, path[:len(path)-1], 0, int(path[len(path)-1]))
	case protoreflect.EnumDescriptor:
		p.printEnum(d, &reg, w, sourceInfo, path, 0)
	case protoreflect.EnumValueDescriptor:
		p.printEnumValue(d, &reg, w, sourceInfo, path, 0)
	case protoreflect.ServiceDescriptor:
		p.printService(d, &reg, w, sourceInfo, path, 0)
	case protoreflect.MethodDescriptor:
		p.printMethod(d, &reg, w, sourceInfo, path, 0)
	}

	return w.err
}

func findElement(dsc protoreflect.Descriptor) protoreflect.SourcePath {
	// we start with dsc (leaf) and work our way up to root,
	// which means we are building the path backwards
	var path protoreflect.SourcePath
	for dsc.Parent() != nil {
		parent := dsc.Parent()
		path = append(path, int32(dsc.Index()))
		switch d := dsc.(type) {
		case protoreflect.MessageDescriptor:
			if _, ok := parent.(protoreflect.MessageDescriptor); ok {
				path = append(path, internal.MessageNestedMessagesTag)
			} else {
				path = append(path, internal.FileMessagesTag)
			}

		case protoreflect.FieldDescriptor:
			if d.IsExtension() {
				if _, ok := parent.(protoreflect.MessageDescriptor); ok {
					path = append(path, internal.MessageExtensionsTag)
				} else {
					path = append(path, internal.FileExtensionsTag)
				}
			} else {
				path = append(path, internal.MessageFieldsTag)
			}

		case protoreflect.OneofDescriptor:
			path = append(path, internal.MessageOneofsTag)

		case protoreflect.EnumDescriptor:
			if _, ok := parent.(protoreflect.MessageDescriptor); ok {
				path = append(path, internal.MessageEnumsTag)
			} else {
				path = append(path, internal.FileEnumsTag)
			}

		case protoreflect.EnumValueDescriptor:
			path = append(path, internal.EnumValuesTag)

		case protoreflect.ServiceDescriptor:
			path = append(path, internal.FileServicesTag)

		case protoreflect.MethodDescriptor:
			path = append(path, internal.ServiceMethodsTag)

		default:
			panic(fmt.Sprintf("unexpected descriptor type: %T", dsc))
		}
		dsc = parent
	}
	// finally, we reverse the backwards path
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path
}

func (p *Printer) newLine(w io.Writer) {
	if !p.Compact {
		_, _ = fmt.Fprintln(w)
	}
}

func (p *Printer) printFile(
	fd protoreflect.FileDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
) {
	opts, err := p.extractOptions(fd, reg, fd.Options())
	if err != nil {
		return
	}

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

	skip := map[interface{}]bool{}

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

func (p *Printer) qualifyMessageOptionName(pkg, scope, fqn protoreflect.FullName) string {
	// Message options must at least include the message scope, even if the option
	// is inside that message. We do that by requiring we have at least one
	// enclosing skip in the qualified name.
	return p.qualifyElementName(pkg, scope, fqn, 1)
}

func (p *Printer) qualifyExtensionLiteralName(pkg, scope, fqn protoreflect.FullName) string {
	// In message literals, extensions can have package name omitted but may not
	// have any other scopes omitted. We signal that via negative arg.
	return p.qualifyElementName(pkg, scope, fqn, -1)
}

func (p *Printer) qualifyName(pkg, scope, fqn protoreflect.FullName) string {
	return p.qualifyElementName(pkg, scope, fqn, 0)
}

func (p *Printer) qualifyElementName(pkg, scope, fqn protoreflect.FullName, required int) string {
	if p.ForceFullyQualifiedNames {
		// forcing fully-qualified names; make sure to include preceding dot
		if fqn[0] == '.' {
			return string(fqn)
		}
		return fmt.Sprintf(".%s", fqn)
	}

	// compute relative name (so no leading dot)
	if fqn[0] == '.' {
		fqn = fqn[1:]
	}
	if required < 0 {
		scope = pkg + "."
	} else if len(scope) > 0 && scope[len(scope)-1] != '.' {
		scope = scope + "."
	}
	count := 0
	for scope != "" {
		if strings.HasPrefix(string(fqn), string(scope)) && count >= required {
			return string(fqn[len(scope):])
		}
		if scope == pkg+"." {
			break
		}
		pos := strings.LastIndex(string(scope[:len(scope)-1]), ".")
		scope = scope[:pos+1]
		count++
	}
	return string(fqn)
}

func (p *Printer) typeString(fld protoreflect.FieldDescriptor, scope protoreflect.FullName) string {
	if fld.IsMap() {
		return fmt.Sprintf("map<%s, %s>", p.typeString(fld.MapKey(), scope), p.typeString(fld.MapValue(), scope))
	}
	switch fld.Kind() {
	case protoreflect.EnumKind:
		return p.qualifyName(fld.ParentFile().Package(), scope, fld.Enum().FullName())
	case protoreflect.GroupKind:
		if isGroup(fld) {
			return string(fld.Message().Name())
		}
		fallthrough
	case protoreflect.MessageKind:
		return p.qualifyName(fld.ParentFile().Package(), scope, fld.Message().FullName())
	default:
		return fld.Kind().String()
	}
}

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
	opts, err := p.extractOptions(md, reg, md.Options())
	if err != nil {
		if w.err == nil {
			w.err = err
		}
		return
	}

	skip := map[interface{}]bool{}
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

		opts, err := p.extractOptions(fld, reg, fld.Options())
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

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
		if !fld.IsExtension() && jsn != "" && jsn != internal.JsonName(fld.Name()) {
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

		opts, err := p.extractOptions(ood, reg, ood.Options())
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

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
	p.printElementString(extNameSi, w, indent, p.qualifyName(pkg, scope, exts.extendee))
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

		opts, err := p.extractOptions(ed, reg, ed.Options())
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

		skip := map[interface{}]bool{}

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

func (p *Printer) printService(
	sd protoreflect.ServiceDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	si := sourceInfo.ByPath(path)
	p.printBlockElement(true, si, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)

		_, _ = fmt.Fprint(w, "service ")
		nameSi := sourceInfo.ByPath(append(path, internal.ServiceNameTag))
		p.printElementString(nameSi, w, indent, string(sd.Name()))
		_, _ = fmt.Fprintln(w, "{")
		indent++
		trailer(indent, true)

		opts, err := p.extractOptions(sd, reg, sd.Options())
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

		elements := elementAddrs{dsc: sd, opts: opts}
		elements.addrs = append(elements.addrs, optionsAsElementAddrs(internal.ServiceOptionsTag, -1, opts)...)
		methods := sd.Methods()
		for i, length := 0, methods.Len(); i < length; i++ {
			elements.addrs = append(elements.addrs, elementAddr{elementType: internal.ServiceMethodsTag, elementIndex: i})
		}

		p.sort(elements, sourceInfo, path)

		for i, el := range elements.addrs {
			if i > 0 {
				p.newLine(w)
			}

			childPath := append(path, el.elementType, int32(el.elementIndex))

			switch d := elements.at(el).(type) {
			case []option:
				p.printOptionsLong(d, reg, w, sourceInfo, childPath, indent)
			case protoreflect.MethodDescriptor:
				p.printMethod(d, reg, w, sourceInfo, childPath, indent)
			}
		}

		p.indent(w, indent-1)
		_, _ = fmt.Fprintln(w, "}")
	})
}

func (p *Printer) printMethod(
	mtd protoreflect.MethodDescriptor,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	si := sourceInfo.ByPath(path)
	pkg := mtd.ParentFile().Package()
	p.printBlockElement(true, si, w, indent, func(w *writer, trailer func(int, bool)) {
		p.indent(w, indent)

		_, _ = fmt.Fprint(w, "rpc ")
		nameSi := sourceInfo.ByPath(append(path, internal.MethodNameTag))
		p.printElementString(nameSi, w, indent, string(mtd.Name()))

		_, _ = fmt.Fprint(w, "( ")
		inSi := sourceInfo.ByPath(append(path, internal.MethodInputTag))
		inName := p.qualifyName(pkg, pkg, mtd.Input().FullName())
		if mtd.IsStreamingClient() {
			inName = "stream " + inName
		}
		p.printElementString(inSi, w, indent, inName)

		_, _ = fmt.Fprint(w, ") returns ( ")

		outSi := sourceInfo.ByPath(append(path, internal.MethodOutputTag))
		outName := p.qualifyName(pkg, pkg, mtd.Output().FullName())
		if mtd.IsStreamingServer() {
			outName = "stream " + outName
		}
		p.printElementString(outSi, w, indent, outName)
		_, _ = fmt.Fprint(w, ") ")

		opts, err := p.extractOptions(mtd, reg, mtd.Options())
		if err != nil {
			if w.err == nil {
				w.err = err
			}
			return
		}

		if len(opts) > 0 {
			_, _ = fmt.Fprintln(w, "{")
			indent++
			trailer(indent, true)

			elements := elementAddrs{dsc: mtd, opts: opts}
			elements.addrs = optionsAsElementAddrs(internal.MethodOptionsTag, 0, opts)
			p.sort(elements, sourceInfo, path)

			for i, el := range elements.addrs {
				if i > 0 {
					p.newLine(w)
				}
				o := elements.at(el).([]option)
				childPath := append(path, el.elementType, int32(el.elementIndex))
				p.printOptionsLong(o, reg, w, sourceInfo, childPath, indent)
			}

			p.indent(w, indent-1)
			_, _ = fmt.Fprintln(w, "}")
		} else {
			_, _ = fmt.Fprint(w, ";")
			trailer(indent, false)
		}
	})
}

func (p *Printer) printOptionsLong(
	opts []option,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	p.printOptions(opts, w, indent,
		func(i int32) protoreflect.SourceLocation {
			return sourceInfo.ByPath(append(path, i))
		},
		func(w *writer, indent int, opt option, _ bool) {
			p.indent(w, indent)
			_, _ = fmt.Fprint(w, "option ")
			p.printOption(reg, opt.name, opt.val, w, indent)
			_, _ = fmt.Fprint(w, ";")
		},
		false)
}

func (p *Printer) extractAndPrintOptionsShort(
	dsc interface{},
	optsMsg proto.Message,
	reg *protoregistry.Types,
	optsTag int32,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	d, ok := dsc.(protoreflect.Descriptor)
	if !ok {
		d = dsc.(extensionRangeMarker).owner
	}
	opts, err := p.extractOptions(d, reg, optsMsg)
	if err != nil {
		if w.err == nil {
			w.err = err
		}
		return
	}
	p.printOptionsShort(dsc, opts, optsTag, reg, w, sourceInfo, path, indent)
}

func (p *Printer) printOptionsShort(
	dsc interface{},
	opts map[protoreflect.FieldNumber][]option,
	optsTag int32,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
) {
	elements := elementAddrs{dsc: dsc, opts: opts}
	elements.addrs = optionsAsElementAddrs(optsTag, 0, opts)
	if len(elements.addrs) == 0 {
		return
	}
	p.sort(elements, sourceInfo, path)

	// we render expanded form if there are many options
	count := 0
	for _, addr := range elements.addrs {
		opts := elements.at(addr).([]option)
		count += len(opts)
	}
	threshold := p.ShortOptionsExpansionThresholdCount
	if threshold <= 0 {
		threshold = 3
	}

	if count > threshold {
		p.printOptionElementsShort(elements, reg, w, sourceInfo, path, indent, true)
	} else {
		var tmp bytes.Buffer
		tmpW := *w
		tmpW.Writer = &tmp
		p.printOptionElementsShort(elements, reg, &tmpW, sourceInfo, path, indent, false)
		threshold := p.ShortOptionsExpansionThresholdLength
		if threshold <= 0 {
			threshold = 50
		}
		// we subtract 3 so we don't consider the leading " [" and trailing "]"
		if tmp.Len()-3 > threshold {
			p.printOptionElementsShort(elements, reg, w, sourceInfo, path, indent, true)
		} else {
			// not too long: commit what we rendered
			b := tmp.Bytes()
			if w.space && len(b) > 0 && b[0] == ' ' {
				// don't write extra space
				b = b[1:]
			}
			_, _ = w.Write(b)
			w.newline = tmpW.newline
			w.space = tmpW.space
		}
	}
}

func (p *Printer) printOptionElementsShort(
	addrs elementAddrs,
	reg *protoregistry.Types,
	w *writer,
	sourceInfo protoreflect.SourceLocations,
	path protoreflect.SourcePath,
	indent int,
	expand bool,
) {
	if expand {
		_, _ = fmt.Fprintln(w, "[")
		indent++
	} else {
		_, _ = fmt.Fprint(w, "[")
	}
	for i, addr := range addrs.addrs {
		opts := addrs.at(addr).([]option)
		var childPath []int32
		if addr.elementIndex < 0 {
			// pseudo-option
			childPath = append(path, int32(-addr.elementIndex))
		} else {
			childPath = append(path, addr.elementType, int32(addr.elementIndex))
		}
		optIndent := indent
		if !expand {
			optIndent = inline(indent)
		}
		p.printOptions(opts, w, optIndent,
			func(i int32) protoreflect.SourceLocation {
				p := childPath
				if addr.elementIndex >= 0 {
					p = append(p, i)
				}
				return sourceInfo.ByPath(p)
			},
			func(w *writer, indent int, opt option, more bool) {
				if expand {
					p.indent(w, indent)
				}
				p.printOption(reg, opt.name, opt.val, w, indent)
				if more {
					if expand {
						_, _ = fmt.Fprintln(w, ",")
					} else {
						_, _ = fmt.Fprint(w, ", ")
					}
				}
			},
			i < len(addrs.addrs)-1)
	}
	if expand {
		p.indent(w, indent-1)
	}
	_, _ = fmt.Fprint(w, "] ")
}

func (p *Printer) printOptions(
	opts []option,
	w *writer,
	indent int,
	siFetch func(i int32) protoreflect.SourceLocation,
	fn func(w *writer, indent int, opt option, more bool),
	haveMore bool,
) {
	for i, opt := range opts {
		more := haveMore
		if !more {
			more = i < len(opts)-1
		}
		si := siFetch(int32(i))
		p.printElement(false, si, w, indent, func(w *writer) {
			fn(w, indent, opt, more)
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

func sortKeys(m protoreflect.Map) []protoreflect.MapKey {
	res := make([]protoreflect.MapKey, m.Len())
	i := 0
	m.Range(func(k protoreflect.MapKey, _ protoreflect.Value) bool {
		res[i] = k
		i++
		return true
	})
	sort.Slice(res, func(i, j int) bool {
		switch i := res[i].Interface().(type) {
		case int32:
			return i < int32(res[j].Int())
		case uint32:
			return i < uint32(res[j].Uint())
		case int64:
			return i < res[j].Int()
		case uint64:
			return i < res[j].Uint()
		case string:
			return i < res[j].String()
		case bool:
			return !i && res[j].Bool()
		default:
			panic(fmt.Sprintf("invalid type for map key: %T", i))
		}
	})
	return res
}

func (p *Printer) printOption(reg *protoregistry.Types, name string, optVal interface{}, w *writer, indent int) {
	_, _ = fmt.Fprintf(w, "%s = ", name)

	switch optVal := optVal.(type) {
	case int32, uint32, int64, uint64:
		_, _ = fmt.Fprintf(w, "%d", optVal)
	case float32, float64:
		_, _ = fmt.Fprintf(w, "%f", optVal)
	case string:
		_, _ = fmt.Fprintf(w, "%s", quotedString(optVal))
	case []byte:
		_, _ = fmt.Fprintf(w, "%s", quotedBytes(string(optVal)))
	case bool:
		_, _ = fmt.Fprintf(w, "%v", optVal)
	case ident:
		_, _ = fmt.Fprintf(w, "%s", optVal)
	case messageVal:
		threshold := p.MessageLiteralExpansionThresholdLength
		if threshold == 0 {
			threshold = 50
		}
		var buf bytes.Buffer
		p.printMessageLiteralToBufferMaybeCompact(&buf, optVal.msg.ProtoReflect(), reg, optVal.pkg, optVal.scope, threshold, indent)
		_, _ = w.Write(buf.Bytes())

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
		for i := 0; i+1 < len(loc.Path); i += 2 {
			nextKind, ok := allowed[loc.Path[i]]
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
				newPath := make(protoreflect.SourcePath, i+3)
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

func (p *Printer) extractOptions(dsc protoreflect.Descriptor, reg *protoregistry.Types, opts proto.Message) (map[protoreflect.FieldNumber][]option, error) {
	protomessage.ReparseUnrecognized(opts, reg)

	pkg := dsc.ParentFile().Package()
	var scope protoreflect.FullName
	isMessage := false
	if _, ok := dsc.(protoreflect.FileDescriptor); ok {
		scope = pkg
	} else {
		_, isMessage = dsc.(protoreflect.MessageDescriptor)
		scope = dsc.FullName()
	}

	ref := opts.ProtoReflect()

	options := map[protoreflect.FieldNumber][]option{}
	ref.Range(func(fld protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		var name string
		if fld.IsExtension() {
			var n string
			if isMessage {
				n = p.qualifyMessageOptionName(pkg, scope, fld.FullName())
			} else {
				n = p.qualifyName(pkg, scope, fld.FullName())
			}
			name = fmt.Sprintf("(%s)", n)
		} else {
			name = string(fld.Name())
		}
		opts := valueToOptions(fld, name, val.Interface())
		if len(opts) > 0 {
			for i := range opts {
				if msg, ok := opts[i].val.(proto.Message); ok {
					opts[i].val = messageVal{pkg: pkg, scope: scope, msg: msg}
				}
			}
			options[fld.Number()] = opts
		}
		return true
	})
	return options, nil
}

func valueToOptions(fld protoreflect.FieldDescriptor, name string, val interface{}) []option {
	switch val := val.(type) {
	case protoreflect.List:
		if fld.Number() == internal.UninterpretedOptionsTag {
			// we handle uninterpreted options differently
			uninterp := make([]*descriptorpb.UninterpretedOption, 0, val.Len())
			for i := 0; i < val.Len(); i++ {
				uo := toUninterpretedOption(val.Get(i).Message().Interface())
				if uo != nil {
					uninterp = append(uninterp, uo)
				}
			}
			return uninterpretedToOptions(uninterp)
		}
		opts := make([]option, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			elem := valueForOption(fld, val.Get(i).Interface())
			if elem != nil {
				opts = append(opts, option{name: name, val: elem})
			}
		}
		return opts
	case protoreflect.Map:
		opts := make([]option, 0, val.Len())
		for _, k := range sortKeys(val) {
			v := val.Get(k)
			vf := fld.MapValue()
			if vf.Kind() == protoreflect.EnumKind {
				if vf.Enum().Values().ByNumber(v.Enum()) == nil {
					// have to skip unknown enum values :(
					continue
				}
			}
			entry := dynamicpb.NewMessage(fld.Message())
			entry.Set(fld.Message().Fields().ByNumber(1), k.Value())
			entry.Set(fld.Message().Fields().ByNumber(2), v)
			opts = append(opts, option{name: name, val: entry})
		}
		return opts
	default:
		v := valueForOption(fld, val)
		if v == nil {
			return nil
		}
		return []option{{name: name, val: v}}
	}
}

func valueForOption(fld protoreflect.FieldDescriptor, val interface{}) interface{} {
	switch val := val.(type) {
	case protoreflect.EnumNumber:
		ev := fld.Enum().Values().ByNumber(val)
		if ev == nil {
			// if enum val is unknown, we'll return nil and have to skip it :(
			return nil
		}
		return ident(ev.Name())
	case protoreflect.Message:
		return val.Interface()
	default:
		return val
	}
}

func toUninterpretedOption(message proto.Message) *descriptorpb.UninterpretedOption {
	if uo, ok := message.(*descriptorpb.UninterpretedOption); ok {
		return uo
	}
	// marshal and unmarshal to convert; if we fail to convert, skip it
	var uo descriptorpb.UninterpretedOption
	data, err := proto.Marshal(message)
	if err != nil {
		return nil
	}
	if proto.Unmarshal(data, &uo) != nil {
		return nil
	}
	return &uo
}

func uninterpretedToOptions(uninterp []*descriptorpb.UninterpretedOption) []option {
	opts := make([]option, len(uninterp))
	for i, unint := range uninterp {
		var buf bytes.Buffer
		for ni, n := range unint.Name {
			if ni > 0 {
				buf.WriteByte('.')
			}
			if n.GetIsExtension() {
				_, _ = fmt.Fprintf(&buf, "(%s)", n.GetNamePart())
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
			v = ident("{ " + unint.GetAggregateValue() + " }")
		}

		opts[i] = option{name: buf.String(), val: v}
	}
	return opts
}

func optionsAsElementAddrs(optionsTag int32, order int, opts map[protoreflect.FieldNumber][]option) []elementAddr {
	optAddrs := make([]elementAddr, 0, len(opts))
	for tag := range opts {
		optAddrs = append(optAddrs, elementAddr{elementType: optionsTag, elementIndex: int(tag), order: order})
	}
	// We want stable output. So, if the printer can't sort these a better way,
	// they'll at least be in a deterministic order (by name).
	sort.Sort(optionsByName{addrs: optAddrs, opts: opts})
	return optAddrs
}

// quotedBytes implements the text format for string literals for protocol
// buffers. Since the underlying data is a bytes field, this encodes all
// bytes outside the 7-bit ASCII printable range. To preserve unicode strings
// without byte escapes, use quotedString.
func quotedBytes(s string) string {
	var b bytes.Buffer
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
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		default:
			if c >= 0x20 && c < 0x7f {
				b.WriteByte(c)
			} else {
				_, _ = fmt.Fprintf(&b, "\\%03o", c)
			}
		}
	}
	b.WriteByte('"')

	return b.String()
}

// quotedString implements the text format for string literals for protocol
// buffers. This form is also acceptable for string literals in option values
// by the protocol buffer compiler, protoc.
func quotedString(s string) string {
	var b bytes.Buffer
	b.WriteByte('"')
	// Loop over the bytes, not the runes.
	for {
		r, n := utf8.DecodeRuneInString(s)
		if n == 0 {
			break // end of string
		}
		if r == utf8.RuneError && n == 1 {
			// Invalid UTF8! Use an octal byte escape to encode the bad byte.
			_, _ = fmt.Fprintf(&b, "\\%03o", s[0])
			s = s[1:]
			continue
		}

		// Divergence from C++: we don't escape apostrophes.
		// There's no need to escape them, and the C++ parser
		// copes with a naked apostrophe.
		switch r {
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '"':
			b.WriteString("\\\"")
		case '\\':
			b.WriteString("\\\\")
		default:
			if unicode.IsPrint(r) {
				b.WriteRune(r)
			} else {
				// if it's not printable, use a unicode escape
				if r > 0xffff {
					_, _ = fmt.Fprintf(&b, "\\U%08X", r)
				} else if r > 0x7F {
					_, _ = fmt.Fprintf(&b, "\\u%04X", r)
				} else {
					_, _ = fmt.Fprintf(&b, "\\%03o", byte(r))
				}
			}
		}

		s = s[n:]
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

func (a elementAddrs) at(addr elementAddr) interface{} {
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

func (p *Printer) printBlockElement(
	isDecriptor bool,
	si protoreflect.SourceLocation,
	w *writer,
	indent int,
	el func(w *writer, trailer func(indent int, wantTrailingNewline bool)),
) {
	includeComments := isDecriptor || p.includeCommentType(CommentsTokens)

	if includeComments && si.Path != nil {
		p.printLeadingComments(si, w, indent)
	}
	el(w, func(indent int, wantTrailingNewline bool) {
		if includeComments && !sourceloc.IsZero(si) {
			if p.printTrailingComments(si, w, indent) && wantTrailingNewline && !p.Compact {
				// separator line between trailing comment and next element
				_, _ = fmt.Fprintln(w)
			}
		}
	})
	if indent >= 0 && !w.newline {
		// if we're not printing inline but element did not have trailing newline, add one now
		_, _ = fmt.Fprintln(w)
	}
}

func (p *Printer) printElement(isDecriptor bool, si protoreflect.SourceLocation, w *writer, indent int, el func(*writer)) {
	includeComments := isDecriptor || p.includeCommentType(CommentsTokens)

	if includeComments && !sourceloc.IsZero(si) {
		p.printLeadingComments(si, w, indent)
	}
	el(w)
	if includeComments && !sourceloc.IsZero(si) {
		p.printTrailingComments(si, w, indent)
	}
	if indent >= 0 && !w.newline {
		// if we're not printing inline but element did not have trailing newline, add one now
		_, _ = fmt.Fprintln(w)
	}
}

func (p *Printer) printElementString(si protoreflect.SourceLocation, w *writer, indent int, str string) {
	p.printElement(false, si, w, inline(indent), func(w *writer) {
		_, _ = fmt.Fprintf(w, "%s ", str)
	})
}

func (p *Printer) includeCommentType(c CommentType) bool {
	return (p.OmitComments & c) == 0
}

func (p *Printer) printLeadingComments(si protoreflect.SourceLocation, w *writer, indent int) bool {
	endsInNewLine := false

	if p.includeCommentType(CommentsDetached) {
		for _, c := range si.LeadingDetachedComments {
			if p.printComment(c, w, indent, true) {
				// if comment ended in newline, add another newline to separate
				// this comment from the next
				p.newLine(w)
				endsInNewLine = true
			} else if indent < 0 {
				// comment did not end in newline and we are trying to inline?
				// just add a space to separate this comment from what follows
				_, _ = fmt.Fprint(w, " ")
				endsInNewLine = false
			} else {
				// comment did not end in newline and we are *not* trying to inline?
				// add newline to end of comment and add another to separate this
				// comment from what follows
				_, _ = fmt.Fprintln(w) // needed to end comment, regardless of p.Compact
				p.newLine(w)
				endsInNewLine = true
			}
		}
	}

	if p.includeCommentType(CommentsLeading) && si.LeadingComments != "" {
		endsInNewLine = p.printComment(si.LeadingComments, w, indent, true)
		if !endsInNewLine {
			if indent >= 0 {
				// leading comment didn't end with newline but needs one
				// (because we're *not* inlining)
				_, _ = fmt.Fprintln(w) // needed to end comment, regardless of p.Compact
				endsInNewLine = true
			} else {
				// space between comment and following element when inlined
				_, _ = fmt.Fprint(w, " ")
			}
		}
	}

	return endsInNewLine
}

func (p *Printer) printTrailingComments(si protoreflect.SourceLocation, w *writer, indent int) bool {
	if p.includeCommentType(CommentsTrailing) && si.TrailingComments != "" {
		if !p.printComment(si.TrailingComments, w, indent, p.TrailingCommentsOnSeparateLine) && indent >= 0 {
			// trailing comment didn't end with newline but needs one
			// (because we're *not* inlining)
			_, _ = fmt.Fprintln(w) // needed to end comment, regardless of p.Compact
		} else if indent < 0 {
			_, _ = fmt.Fprint(w, " ")
		}
		return true
	}

	return false
}

func (p *Printer) printComment(comments string, w *writer, indent int, forceNextLine bool) bool {
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
	if multiLine && strings.Contains(comments, "*/") {
		// can't emit '*/' in a multi-line style comment
		multiLine = false
	}

	lines := strings.Split(comments, "\n")

	// first, remove leading and trailing blank lines
	if lines[0] == "" {
		lines = lines[1:]
	}
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return false
	}

	if indent >= 0 && !w.newline {
		// last element did not have trailing newline, so we
		// either need to tack on newline or, if comment is
		// just one line, inline it on the end
		if forceNextLine || len(lines) > 1 {
			_, _ = fmt.Fprintln(w)
		} else {
			if !w.space {
				_, _ = fmt.Fprint(w, " ")
			}
			indent = inline(indent)
		}
	}

	if len(lines) == 1 && multiLine {
		p.indent(w, indent)
		line := lines[0]
		if line[0] == ' ' && line[len(line)-1] != ' ' {
			// add trailing space for symmetry
			line += " "
		}
		_, _ = fmt.Fprintf(w, "/*%s*/", line)
		if indent >= 0 {
			_, _ = fmt.Fprintln(w)
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
		if l != "" && !strings.HasPrefix(l, " ") {
			l = " " + l
		}
		p.maybeIndent(w, indent, i > 0)
		if multiLine {
			if i == 0 {
				// first line
				_, _ = fmt.Fprintf(w, "/*%s\n", strings.TrimRight(l, " \t"))
			} else if i == len(lines)-1 {
				// last line
				if strings.TrimSpace(l) == "" {
					_, _ = fmt.Fprint(w, " */")
				} else {
					_, _ = fmt.Fprintf(w, " *%s*/", l)
				}
				if indent >= 0 {
					_, _ = fmt.Fprintln(w)
				}
			} else {
				_, _ = fmt.Fprintf(w, " *%s\n", strings.TrimRight(l, " \t"))
			}
		} else {
			_, _ = fmt.Fprintf(w, "//%s\n", strings.TrimRight(l, " \t"))
		}
	}

	// single-line comments always end in newline; multi-line comments only
	// end in newline for non-negative (e.g. non-inlined) indentation
	return !multiLine || indent >= 0
}

func (p *Printer) indent(w io.Writer, indent int) {
	for i := 0; i < indent; i++ {
		_, _ = fmt.Fprint(w, p.Indent)
	}
}

func (p *Printer) maybeIndent(w io.Writer, indent int, requireIndent bool) {
	if indent < 0 && requireIndent {
		p.indent(w, -indent)
	} else {
		p.indent(w, indent)
	}
}

type writer struct {
	io.Writer
	err     error
	space   bool
	newline bool
}

func newWriter(w io.Writer) *writer {
	return &writer{Writer: w, newline: true}
}

func (w *writer) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	w.newline = false

	if w.space {
		// skip any trailing space if the following
		// character is semicolon, comma, or close bracket
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
	if len(p) > 0 && p[len(p)-1] == '\n' {
		w.newline = true
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
