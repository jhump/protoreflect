package protoprint

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	_ "github.com/jhump/protoreflect/internal/testprotos"
	"github.com/jhump/protoreflect/internal/testutil"
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
		"sorted":                              {Indent: "   ", SortElements: true, OmitDetachedComments: true},
		"sorted-AND-multiline-style-comments": {PreferMultiLineStyleComments: true, SortElements: true},
		"custom-sort":                         {CustomSortFunction: reverseByName},
	}

	// create descriptors to print
	files := []string{
		"../../internal/testprotos/desc_test_comments.protoset",
		"../../internal/testprotos/desc_test_complex_source_info.protoset",
		"../../internal/testprotos/descriptor.protoset",
		"../../internal/testprotos/desc_test1.protoset",
		"../../internal/testprotos/proto3_optional/desc_test_proto3_optional.protoset",
	}
	fds := make([]*desc.FileDescriptor, len(files)+1)
	for i, file := range files {
		fd, err := loadProtoset(file)
		testutil.Ok(t, err)
		fds[i] = fd
	}
	// extra descriptor that has no source info
	fd, err := desc.LoadFileDescriptor("desc_test2.proto")
	testutil.Ok(t, err)
	fds[len(files)] = fd

	for _, fd := range fds {
		for name, pr := range prs {
			baseName := filepath.Base(fd.GetName())
			ext := filepath.Ext(baseName)
			baseName = baseName[:len(baseName)-len(ext)]
			goldenFile := fmt.Sprintf("%s-%s.proto", baseName, name)

			checkFile(t, pr, fd, goldenFile)
		}
	}
}

func loadProtoset(path string) (*desc.FileDescriptor, error) {
	var fds descriptor.FileDescriptorSet
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bb, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err = proto.Unmarshal(bb, &fds); err != nil {
		return nil, err
	}
	return desc.CreateFileDescriptorFromSet(&fds)
}

func checkFile(t *testing.T, pr *Printer, fd *desc.FileDescriptor, goldenFile string) {
	var buf bytes.Buffer
	err := pr.PrintProtoFile(fd, &buf)
	testutil.Ok(t, err)

	checkContents(t, buf.String(), goldenFile)
}

func TestParseAndPrintPreservesAsMuchAsPossible(t *testing.T) {
	pa := protoparse.Parser{ImportPaths: []string{"../../internal/testprotos"}, IncludeSourceCodeInfo: true}
	fds, err := pa.ParseFiles("desc_test_comments.proto")
	testutil.Ok(t, err)
	fd := fds[0]
	checkFile(t, &Printer{}, fd, "test-preserve-comments.proto")
	checkFile(t, &Printer{OmitComments: CommentsNonDoc}, fd, "test-preserve-doc-comments.proto")
}

func TestParseAndPrintWithUnrecognizedOptions(t *testing.T) {
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

	pa := &protoparse.Parser{
		Accessor: protoparse.FileContentsFromMap(files),
	}
	fds, err := pa.ParseFiles("test.proto")
	testutil.Ok(t, err)

	checkFile(t, &Printer{}, fds[0], "test-unrecognized-options.proto")
}

func TestPrintNonFileDescriptors(t *testing.T) {
	pa := protoparse.Parser{ImportPaths: []string{"../../internal/testprotos"}, IncludeSourceCodeInfo: true}
	fds, err := pa.ParseFiles("desc_test_comments.proto")
	testutil.Ok(t, err)
	fd := fds[0]

	var buf bytes.Buffer
	crawl(t, fd, &Printer{}, &buf)
	checkContents(t, buf.String(), "test-non-files-full.txt")

	buf.Reset()
	crawl(t, fd, &Printer{OmitComments: CommentsNonDoc, Compact: true, SortElements: true, ForceFullyQualifiedNames: true}, &buf)
	checkContents(t, buf.String(), "test-non-files-compact.txt")
}

func crawl(t *testing.T, d desc.Descriptor, p *Printer, out io.Writer) {
	str, err := p.PrintProtoToString(d)
	testutil.Ok(t, err)
	fmt.Fprintf(out, "-------- %s (%T) --------\n", d.GetFullyQualifiedName(), d)
	fmt.Fprint(out, str)

	switch d := d.(type) {
	case *desc.FileDescriptor:
		for _, md := range d.GetMessageTypes() {
			crawl(t, md, p, out)
		}
		for _, ed := range d.GetEnumTypes() {
			crawl(t, ed, p, out)
		}
		for _, extd := range d.GetExtensions() {
			crawl(t, extd, p, out)
		}
		for _, sd := range d.GetServices() {
			crawl(t, sd, p, out)
		}
	case *desc.MessageDescriptor:
		for _, fd := range d.GetFields() {
			crawl(t, fd, p, out)
		}
		for _, ood := range d.GetOneOfs() {
			crawl(t, ood, p, out)
		}
		for _, md := range d.GetNestedMessageTypes() {
			crawl(t, md, p, out)
		}
		for _, ed := range d.GetNestedEnumTypes() {
			crawl(t, ed, p, out)
		}
		for _, extd := range d.GetNestedExtensions() {
			crawl(t, extd, p, out)
		}
	case *desc.EnumDescriptor:
		for _, evd := range d.GetValues() {
			crawl(t, evd, p, out)
		}
	case *desc.ServiceDescriptor:
		for _, mtd := range d.GetMethods() {
			crawl(t, mtd, p, out)
		}
	}
}

func checkContents(t *testing.T, actualContents string, goldenFileName string) {
	goldenFileName = filepath.Join(testFilesDirectory, goldenFileName)

	if regenerateMode {
		err := ioutil.WriteFile(goldenFileName, []byte(actualContents), 0666)
		testutil.Ok(t, err)
	}

	// verify that output matches golden test files
	b, err := ioutil.ReadFile(goldenFileName)
	testutil.Ok(t, err)

	testutil.Eq(t, string(b), actualContents, "wrong file contents for %s", goldenFileName)
}
