// Package main generates a Golang file with a gzipped FileDescriptorProto for every
// Protobuf file in the current directory, including SourceCodeInfo.
//
// It is assumed that the only import path directory is the current directory.
// The Golang file is printed to stdout.
// The Golang package name is "internal".
//
// This is used to generate FileDescriptorProtos for the Well-Known Types.
// This is not meant to be a general-use tool, it is only meant to be used
// in the context of this repository.
//
// See https://github.com/jhump/protoreflect/issues/302
package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/jhump/protoreflect/desc/protoparse"
)

func main() {
	if err := run(); err != nil {
		if errString := err.Error(); errString != "" {
			fmt.Fprintln(os.Stderr, errString)
		}
		os.Exit(1)
	}
}

func run() error {
	protoFilePaths, err := getProtoFilePaths(".")
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	_, _ = buffer.WriteString(`package internal

// protoFilePathToGzippedFileDescriptorProto is a map from Protobuf file path to a gzipped FileDescriptorProto.
//
// Each FileDescriptorProto also includes SourceCodeInfo.
var protoFilePathToGzippedFileDescriptorProto = map[string][]byte{
`)

	for _, protoFilePath := range protoFilePaths {
		fileDescriptorProto, err := getFileDescriptorProto(".", protoFilePath)
		if err != nil {
			return err
		}
		data, err := proto.Marshal(fileDescriptorProto)
		if err != nil {
			return err
		}
		gzippedDataBuffer := bytes.NewBuffer(nil)
		gzipWriter, err := gzip.NewWriterLevel(gzippedDataBuffer, gzip.BestCompression)
		if err != nil {
			return err
		}
		gzipWriter.Write(data)
		gzipWriter.Close()
		gzippedData := gzippedDataBuffer.Bytes()

		_, _ = buffer.WriteString(`"`)
		_, _ = buffer.WriteString(protoFilePath)
		_, _ = buffer.WriteString(`": {
`)
		for len(gzippedData) > 0 {
			n := 16
			if n > len(gzippedData) {
				n = len(gzippedData)
			}
			accum := ""
			for _, elem := range gzippedData[:n] {
				accum += fmt.Sprintf("0x%02x,", elem)
			}
			_, _ = buffer.WriteString(accum)
			_, _ = buffer.WriteString("\n")
			gzippedData = gzippedData[n:]
		}
		_, _ = buffer.WriteString(`},
`)
	}
	_, _ = buffer.WriteString(`}`)

	golangFileData, err := format.Source(buffer.Bytes())
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(golangFileData)
	return err
}

func getProtoFilePaths(dirPath string) ([]string, error) {
	protoFilePathMap := make(map[string]struct{})
	if walkErr := filepath.Walk(
		dirPath,
		func(path string, fileInfo os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fileInfo.Mode().IsRegular() && filepath.Ext(path) == ".proto" {
				if _, ok := protoFilePathMap[path]; ok {
					return fmt.Errorf("duplicate proto file: %v", path)
				}
				protoFilePathMap[path] = struct{}{}
			}
			return nil
		},
	); walkErr != nil {
		return nil, walkErr
	}
	protoFilePaths := make([]string, 0, len(protoFilePathMap))
	for protoFilePath := range protoFilePathMap {
		protoFilePaths = append(protoFilePaths, protoFilePath)
	}
	sort.Strings(protoFilePaths)
	return protoFilePaths, nil
}

func getFileDescriptorProto(importPath string, protoFilePath string) (*descriptor.FileDescriptorProto, error) {
	parser := protoparse.Parser{
		ImportPaths:           []string{importPath},
		IncludeSourceCodeInfo: true,
	}
	fileDescriptors, err := parser.ParseFiles(protoFilePath)
	if err != nil {
		return nil, err
	}
	if len(fileDescriptors) != 1 {
		return nil, fmt.Errorf("expected 1 FileDescriptor, got %d", len(fileDescriptors))
	}
	return fileDescriptors[0].AsFileDescriptorProto(), nil
}
