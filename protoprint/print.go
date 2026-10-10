package protoprint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/internal/register"
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
			return fmt.Errorf("failed to open %s: %w", fd.Path(), err)
		}
		err = func() error {
			defer func() {
				_ = w.Close()
			}()
			return p.PrintProtoFile(fd, w)
		}()
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", fd.Path(), err)
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
	val  any
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
	// Normalize the indent in a copy, so we don't modify the caller's Printer,
	// which may be in use concurrently.
	normalized := *p
	normalized.Indent = normalizeIndent(p.Indent)
	return normalized.printNormalizedProto(dsc, out)
}

// normalizeIndent returns the given indent, with any characters other than tabs
// converted to spaces. If the given indent is empty, it returns two spaces.
func normalizeIndent(indent string) string {
	if indent == "" {
		return "  "
	}
	ind := make([]rune, 0, len(indent))
	for _, r := range indent {
		if r == '\t' {
			ind = append(ind, r)
		} else {
			ind = append(ind, ' ')
		}
	}
	return string(ind)
}

func (p *Printer) printNormalizedProto(dsc protoreflect.Descriptor, out io.Writer) error {
	w := newWriter(out)

	fd := dsc.ParentFile()
	sourceInfo := extendOptionLocations(fd)

	var reg protoregistry.Types
	register.RegisterTypesVisibleToFile(fd, &reg, true)

	path := sourceloc.PathFor(dsc)
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
			p.printElementString(extNameSi, w, 0, p.qualifyTypeName(d.ParentFile().Package(), scope, d.ContainingMessage().FullName()))
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

func (p *Printer) newLine(w io.Writer) {
	if !p.Compact {
		_, _ = fmt.Fprintln(w)
	}
}
