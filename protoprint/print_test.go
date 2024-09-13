package protoprint

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	_ "github.com/jhump/protoreflect/v2/internal/testprotos"
)

const (
	// When false, test behaves normally, checking output against golden test files.
	// But when changed to true, running test will actually re-generate golden test
	// files (which assumes output is correct).
	regenerateMode = false

	testFilesDirectory = "testfiles"
)

func reverseByName(a, b Element) bool {
	// custom sort that is practically the *reverse* of default sort
	// order, though things like fields/extensions/enum values are
	// sorted by name (descending) instead of by number

	if a.Kind() != b.Kind() {
		return a.Kind() > b.Kind()
	}
	switch a.Kind() {
	case KindExtension:
		if a.Extendee() != b.Extendee() {
			return a.Extendee() > b.Extendee()
		}
	case KindOption:
		if a.IsCustomOption() != b.IsCustomOption() {
			return a.IsCustomOption()
		}
	}
	if a.Name() != b.Name() {
		return a.Name() > b.Name()
	}
	if a.Number() != b.Number() {
		return a.Number() > b.Number()
	}
	aStart, aEnd := a.NumberRange()
	bStart, bEnd := b.NumberRange()
	if aStart != bStart {
		return aStart > bStart
	}
	return aEnd > bEnd
}

func TestPrinter(t *testing.T) {
	prs := map[string]*Printer{
		"default":                             {},
		"compact":                             {Compact: true, ShortOptionsExpansionThresholdCount: 5, ShortOptionsExpansionThresholdLength: 100, MessageLiteralExpansionThresholdLength: 80},
		"no-trailing-comments":                {OmitComments: CommentsTrailing},
		"trailing-on-next-line":               {TrailingCommentsOnSeparateLine: true},
		"only-doc-comments":                   {OmitComments: CommentsNonDoc},
		"multiline-style-comments":            {Indent: "\t", PreferMultiLineStyleComments: true},
		"sorted":                              {Indent: "   ", SortElements: true, OmitComments: CommentsDetached},
		"sorted-AND-multiline-style-comments": {PreferMultiLineStyleComments: true, SortElements: true},
		"custom-sort":                         {CustomSortFunction: reverseByName},
	}

	// create descriptors to print
	files := []string{
		"../internal/testprotos/desc_test_comments.protoset",
		"../internal/testprotos/desc_test_complex_source_info.protoset",
		"../internal/testprotos/desc_test_editions.protoset",
		"../internal/testprotos/desc_test_proto3.protoset",
		"../internal/testprotos/descriptor.protoset",
		"../internal/testprotos/desc_test1.protoset",
		"../internal/testprotos/proto3_optional/desc_test_proto3_optional.protoset",
	}
	fds := make([]protoreflect.FileDescriptor, len(files)+1)
	for i, file := range files {
		fd, err := prototesting.LoadProtoset(file)
		require.NoError(t, err)
		fds[i] = fd
	}
	// extra descriptor that has no source info
	// NB: We can't use desc.LoadFileDescriptor here because that, under the hood, will get
	//     source code info from the desc/sourceinfo package! So explicitly load the version
	//     from the underlying registry, which will NOT have source code info.
	fd, err := protoregistry.GlobalFiles.FindFileByPath("desc_test2.proto")
	require.NoError(t, err)
	fdp := protodesc.ToFileDescriptorProto(fd)
	require.Nil(t, fdp.SourceCodeInfo)
	fds[len(files)] = fd

	for _, fd := range fds {
		for name, pr := range prs {
			baseName := filepath.Base(fd.Path())
			ext := filepath.Ext(baseName)
			baseName = baseName[:len(baseName)-len(ext)]
			goldenFile := fmt.Sprintf("%s-%s.proto", baseName, name)

			checkFile(t, pr, fd, goldenFile)
		}
	}
}

func checkFile(t *testing.T, pr *Printer, fd protoreflect.FileDescriptor, goldenFile string) {
	var buf bytes.Buffer
	err := pr.PrintProtoFile(fd, &buf)
	require.NoError(t, err)

	checkContents(t, buf.String(), goldenFile)
}

func TestPrintCustomOptions(t *testing.T) {
	files := map[string]string{"test.proto": `
syntax = "proto3";

import "google/protobuf/descriptor.proto";

message Test {}

message Foo {
  repeated Bar bar = 1;

  message Bar {
    Baz baz = 1;
    string name = 2;
  }

  enum Baz {
	ZERO = 0;
	FROB = 1;
	NITZ = 2;
  }
}

extend google.protobuf.MethodOptions {
  Foo foo = 54321;
}

service TestService {
  rpc Get (Test) returns (Test) {
    option (foo).bar = { baz:FROB name:"abc" };
    option (foo).bar = { baz:NITZ name:"xyz" };
  }
}
`}

	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			Accessor: protocompile.SourceAccessorFromMap(files),
		}),
	}
	fds, err := compiler.Compile(context.Background(), "test.proto")
	require.NoError(t, err)
	// Sanity check that custom options are recognized.
	unk := fds[0].Services().ByName("TestService").Methods().ByName("Get").Options().ProtoReflect().GetUnknown()
	require.Empty(t, unk)

	checkFile(t, &Printer{}, fds[0], "test-unrecognized-options.proto")

	// Now we try again with descriptors where custom options are unrecognized.
	// We do that by marshaling and then unmarshaling (w/out a resolver) the file
	// descriptor protos.
	fdProto := protodesc.ToFileDescriptorProto(fds[0])
	fdBytes, err := proto.Marshal(fdProto)
	require.NoError(t, err)
	err = proto.Unmarshal(fdBytes, fdProto)
	require.NoError(t, err)

	fd, err := protodesc.NewFile(fdProto, protoregistry.GlobalFiles)
	require.NoError(t, err)
	// Sanity check that this resulted in unrecognized options
	unk = fd.Services().ByName("TestService").Methods().ByName("Get").Options().ProtoReflect().GetUnknown()
	require.NotEmpty(t, unk)

	checkFile(t, &Printer{}, fd, "test-unrecognized-options.proto")
}

func TestPrintUninterpretedOptions(t *testing.T) {
	fileSource := `
syntax = "proto2";
package pkg;
option go_package = "some.pkg";
import "google/protobuf/descriptor.proto";
message Options {
    optional bool some_option_value = 1;
}
extend google.protobuf.MessageOptions {
    optional Options my_some_option = 11964;
}
message SomeMessage {
    option (.pkg.my_some_option) = {some_option_value : true};
}
`

	handler := reporter.NewHandler(nil)
	ast, err := parser.Parse("test.proto", strings.NewReader(fileSource), handler)
	require.NoError(t, err)
	result, err := parser.ResultFromAST(ast, false, handler)
	require.NoError(t, err)
	fdProto := result.FileDescriptorProto()

	// Sanity check that this resulted in uninterpreted options
	unint := fdProto.MessageType[1].Options.UninterpretedOption
	require.NotEmpty(t, unint)

	fd, err := protodesc.NewFile(fdProto, protoregistry.GlobalFiles)
	require.NoError(t, err)

	checkFile(t, &Printer{}, fd, "test-uninterpreted-options.proto")
}

func TestPrintNonFileDescriptors(t *testing.T) {
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(&protocompile.SourceResolver{
			ImportPaths: []string{"../internal/testprotos"},
		}),
		SourceInfoMode: protocompile.SourceInfoExtraComments,
	}
	fds, err := compiler.Compile(context.Background(), "desc_test_comments.proto")
	require.NoError(t, err)
	fd := fds[0]

	var buf bytes.Buffer
	crawl(t, fd, &Printer{}, &buf)
	checkContents(t, buf.String(), "test-non-files-full.txt")

	buf.Reset()
	crawl(t, fd, &Printer{OmitComments: CommentsNonDoc, Compact: true, SortElements: true, ForceFullyQualifiedNames: true}, &buf)
	checkContents(t, buf.String(), "test-non-files-compact.txt")
}

func crawl(t *testing.T, d protoreflect.Descriptor, p *Printer, out io.Writer) {
	str, err := p.PrintProtoToString(d)
	require.NoError(t, err)
	_, _ = fmt.Fprintf(out, "-------- %s (%T) --------\n", d.FullName(), d)
	_, _ = fmt.Fprint(out, str)

	switch d := d.(type) {
	case protoreflect.FileDescriptor:
		msgs := d.Messages()
		for i, length := 0, msgs.Len(); i < length; i++ {
			crawl(t, msgs.Get(i), p, out)
		}
		enums := d.Enums()
		for i, length := 0, enums.Len(); i < length; i++ {
			crawl(t, enums.Get(i), p, out)
		}
		exts := d.Extensions()
		for i, length := 0, exts.Len(); i < length; i++ {
			crawl(t, exts.Get(i), p, out)
		}
		svcs := d.Services()
		for i, length := 0, svcs.Len(); i < length; i++ {
			crawl(t, svcs.Get(i), p, out)
		}
	case protoreflect.MessageDescriptor:
		fields := d.Fields()
		for i, length := 0, fields.Len(); i < length; i++ {
			crawl(t, fields.Get(i), p, out)
		}
		oneofs := d.Oneofs()
		for i, length := 0, oneofs.Len(); i < length; i++ {
			crawl(t, oneofs.Get(i), p, out)
		}
		msgs := d.Messages()
		for i, length := 0, msgs.Len(); i < length; i++ {
			crawl(t, msgs.Get(i), p, out)
		}
		enums := d.Enums()
		for i, length := 0, enums.Len(); i < length; i++ {
			crawl(t, enums.Get(i), p, out)
		}
		exts := d.Extensions()
		for i, length := 0, exts.Len(); i < length; i++ {
			crawl(t, exts.Get(i), p, out)
		}
	case protoreflect.EnumDescriptor:
		vals := d.Values()
		for i, length := 0, vals.Len(); i < length; i++ {
			crawl(t, vals.Get(i), p, out)
		}
	case protoreflect.ServiceDescriptor:
		methods := d.Methods()
		for i, length := 0, methods.Len(); i < length; i++ {
			crawl(t, methods.Get(i), p, out)
		}
	}
}

func checkContents(t *testing.T, actualContents string, goldenFileName string) {
	goldenFileName = filepath.Join(testFilesDirectory, goldenFileName)

	if regenerateMode {
		err := os.WriteFile(goldenFileName, []byte(actualContents), 0666)
		require.NoError(t, err)
	}

	// verify that output matches golden test files
	b, err := os.ReadFile(goldenFileName)
	require.NoError(t, err)

	require.Equal(t, string(b), actualContents, "wrong file contents for %s", goldenFileName)
}

func TestQuoteString(t *testing.T) {
	// other tests have examples of encountering invalid UTF8 and printable unicode
	// so this is just for testing how unprintable valid unicode characters are rendered
	s := quotedString("\x04")
	require.Equal(t, "\"\\004\"", s)
	s = quotedString("\x7F")
	require.Equal(t, "\"\\177\"", s)
	s = quotedString("\u2028")
	require.Equal(t, "\"\\u2028\"", s)
	s = quotedString("\U0010FFFF")
	require.Equal(t, "\"\\U0010FFFF\"", s)
}
