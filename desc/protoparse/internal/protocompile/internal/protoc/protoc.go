// Copyright 2020-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package protoc contains some helpers for invoking protoc from tests. This
// technique is used to verify that protocompile produces equivalent outputs
// and has equivalent behavior as protoc.
package protoc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BinaryPath returns the path to an appropriate protoc executable. This
// path is created by the Makefile, so run `make test` instead of `go test ./...`
// to make sure the path is populated. You can also just create the protoc
// executable via 'make protoc'.
//
// The protoc executable is used by some tests to verify that the output of
// this repo matches the output of the reference compiler.
func BinaryPath(rootDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(rootDir, ".protoc_version"))
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(data))
	protocPath := filepath.Join(rootDir, fmt.Sprintf(".tmp/cache/protoc/%s/bin/protoc", version))
	if runtime.GOOS == "windows" {
		protocPath += ".exe"
	}
	if info, err := os.Stat(protocPath); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s does not exist; run 'make protoc' from the top-level of this repo", protocPath)
		}
		return "", err
	} else if info.IsDir() {
		return "", fmt.Errorf("%s is a directory, but should be an executable file", protocPath)
	}
	return protocPath, nil
}

// Compile compiles the given files with protoc. The fileNames parameter indicates the order of
// files that should be used in the command line for protoc (can be nil when the order does not matter).
//
// This does not return the full results of compilation, only if compilation succeeded or not. The
// returned error will be an instance of *[exec.ExitError] if protoc was successfully invoked but
// returned a non-zero status.
func Compile(files map[string]string, fileNames []string) (stdout []byte, err error) {
	if len(fileNames) != 0 {
		if len(files) != len(fileNames) {
			return nil, fmt.Errorf("fileNames has wrong number of entries: expecting %d, got %d", len(files), len(fileNames))
		}
		for _, fileName := range fileNames {
			if _, exists := files[fileName]; !exists {
				return nil, fmt.Errorf("fileNames has wrong number of entries: expecting %d, got %d", len(files), len(fileNames))
			}
		}
	}

	tempDir, err := writeFileToDisk(files)
	if err != nil {
		if tempDir != "" {
			_ = os.RemoveAll(tempDir)
		}
		return nil, err
	}
	defer func() {
		removeErr := os.RemoveAll(tempDir)
		if err == nil && removeErr != nil {
			err = removeErr
		}
	}()
	if len(fileNames) == 0 {
		fileNames = make([]string, 0, len(files))
		for fileName := range files {
			fileNames = append(fileNames, fileName)
		}
	}
	return invokeProtoc(tempDir, fileNames)
}

func writeFileToDisk(files map[string]string) (string, error) {
	tempDir, err := os.MkdirTemp("", "temp_proto_files")
	if err != nil {
		return "", err
	}

	for fileName, fileContent := range files {
		tempFileName := filepath.Join(tempDir, fileName)
		tempFileDirPart := filepath.Dir(tempFileName)
		if _, err = os.Stat(tempFileDirPart); os.IsNotExist(err) {
			if err = os.MkdirAll(tempFileDirPart, os.ModePerm); err != nil {
				return tempDir, err
			}
		}
		if err := os.WriteFile(tempFileName, []byte(fileContent), 0600); err != nil {
			return tempDir, err
		}
	}
	return tempDir, nil
}

func invokeProtoc(protoPath string, fileNames []string) (stdout []byte, err error) {
	args := []string{"-I", protoPath, "-o", os.DevNull}
	args = append(args, fileNames...)
	protocPath, err := BinaryPath("../")
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(protocPath, args...)
	return cmd.Output()
}
