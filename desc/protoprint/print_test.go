package protoprint

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	"github.com/jhump/protoreflect/internal/testutil"
)

const (
	// When false, test behaves normally, checking output against golden test files.
	// But when changed to true, running test will actually re-generate golden test
	// files (which assumes output is correct).
	regenerateMode = false

	testFilesDirectory = "testfiles"
)

func TestPrinter(t *testing.T) {
	prs := map[string]*Printer{
		"default":                  {},
		"multiline-style-comments": {Indent: "\t", PreferMultiLineStyleComments: true},
		"sorted":                   {Indent: "   ", SortElements: true, OmitDetachedComments: true},
		"sorted-AND-multiline-style-comments": {PreferMultiLineStyleComments: true, SortElements: true},
	}
	files := []string{
		"../../internal/testprotos/desc_test_comments.protoset",
		"../../internal/testprotos/desc_test_complex_source_info.protoset",
		"../../internal/testprotos/descriptor.protoset",
		"../../internal/testprotos/desc_test1.protoset",
	}
	for _, file := range files {
		for name, pr := range prs {
			fd, err := loadProtoset(file)
			testutil.Ok(t, err)

			baseName := filepath.Base(file)
			ext := filepath.Ext(file)
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

	goldenFile = filepath.Join(testFilesDirectory, goldenFile)

	if regenerateMode {
		err = ioutil.WriteFile(goldenFile, buf.Bytes(), 0666)
		testutil.Ok(t, err)
	}

	// verify that output matches golden test files
	b, err := ioutil.ReadFile(goldenFile)
	testutil.Ok(t, err)

	testutil.Eq(t, string(b), buf.String(), "wrong file contents for %s", goldenFile)
}

func TestParseAndPrintPreservesAsMuchAsPossible(t *testing.T) {
	pa := protoparse.Parser{ImportPaths: []string{"../../internal/testprotos"}, IncludeSourceCodeInfo: true}
	fds, err := pa.ParseFiles("desc_test_comments.proto")
	testutil.Ok(t, err)
	fd := fds[0]
	checkFile(t, &Printer{}, fd, "test-preserve-comments.proto")
}
