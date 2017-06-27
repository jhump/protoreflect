package main

// fileset_to_go reads a FileDescriptorSet proto from stdin (in standard proto
// binary encoding) and writes a Go source file to stdout that lives in the same
// package as would generated code for the last file in the set and exposes a
// function named GetDescriptorSet() which loads and decompresses the embedded
// descriptor set message.

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

func main() {
	fdsBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(fmt.Sprintf("Failed to read descriptor set from stdin: %s", err.Error()))
	}

	// parse to make sure it's a valid descriptor set
	var fileset descriptor.FileDescriptorSet
	if err = proto.Unmarshal(fdsBytes, &fileset); err != nil {
		panic(fmt.Sprintf("Failed to parse descriptor set from stdin: %s", err.Error()))
	}
	// and also to extract package for generated file
	fd := fileset.GetFile()[len(fileset.GetFile())-1]
	pkg := fd.GetOptions().GetGoPackage()
	if pkg == "" {
		pkg = fd.GetPackage()
	} else {
		pos := strings.LastIndex(pkg, ";")
		if pos >= 0 {
			pkg = pkg[pos+1:]
		} else {
			pkg = fd.GetPackage()
		}
	}
	if pkg == "" {
		panic(fmt.Sprintf("File %s contains no package and no Go package", fd.GetName()))
	}
	pkg = strings.Replace(pkg, ".", "_", -1)

	var buf bytes.Buffer
	gzout := gzip.NewWriter(&buf)
	_, err = gzout.Write(fdsBytes)
	gzout.Close()
	compressedBytes := buf.Bytes()

	fmt.Printf(`package %s

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// GetDescriptorSet returns the embedded file descriptor set proto
// for file %q
func GetDescriptorSet() *descriptor.FileDescriptorSet {
	in, err := gzip.NewReader(bytes.NewReader(descriptorSet))
	if err != nil {
		panic(err.Error())
	}
	fdsBytes, err := ioutil.ReadAll(in)
	if err != nil {
		panic(err.Error())
	}
	var fds descriptor.FileDescriptorSet
	if err = proto.Unmarshal(fdsBytes, &fds); err != nil {
		panic(err.Error())
	}
	return &fds
}

// compressed form of file descriptor set
var descriptorSet = []byte {
`, pkg, fd.GetName())

	i := 0
	for i < len(compressedBytes) {
		fmt.Print(`	`)
		for j := 0; j < 20; j++ {
			fmt.Printf("0x%02x,", compressedBytes[i])
			i++
			if i >= len(compressedBytes) {
				break
			}
		}
		fmt.Println()
	}
	fmt.Println("}")
}
