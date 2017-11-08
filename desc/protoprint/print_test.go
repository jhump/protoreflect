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
	"github.com/jhump/protoreflect/internal/testutil"
)

//go:generate protoc test.proto -o ./test.protoset --include_source_info --include_imports
//go:generate protoc -I ../ ../protoparse/test.proto -o ./test2.protoset --include_source_info --include_imports
//go:generate protoc -I ../../../../ ../../../../golang/protobuf/protoc-gen-go/descriptor/descriptor.proto -o ./descriptor.protoset --include_source_info --include_imports

func TestPrinter(t *testing.T) {
	prs := map[string]*Printer{
		"default":                  {},
		"multiline-style-comments": {Indent: "\t", PreferMultiLineStyleComments: true},
		"sorted":                   {Indent: "   ", SortElements: true},
		"sorted-AND-multiline-style-comments": {PreferMultiLineStyleComments: true, SortElements: true},
	}
	files := []string{
		"test.protoset",
		"test2.protoset",
		"descriptor.protoset",
		"../../internal/testprotos/desc_test1.protoset",
	}
	for _, file := range files {
		for name, pr := range prs {
			fd, err := loadProtoset(file)
			testutil.Ok(t, err)

			baseName := filepath.Base(file)
			ext := filepath.Ext(file)
			baseName = baseName[:len(baseName)-len(ext)]
			goldenFile := filepath.Join("testfiles", fmt.Sprintf("%s-%s.proto", baseName, name))

			var buf bytes.Buffer
			err = pr.PrintProtoFile(fd, &buf)
			testutil.Ok(t, err)

			// change 'true' to 'false' to re-generate golden test files
			if true {
				// verify that output matches golden test files
				b, err := ioutil.ReadFile(goldenFile)
				testutil.Ok(t, err)

				testutil.Eq(t, string(b), buf.String(), "wrong file contents for %s", goldenFile)

			} else {
				err = ioutil.WriteFile(goldenFile, buf.Bytes(), 0666)
				testutil.Ok(t, err)
			}
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
