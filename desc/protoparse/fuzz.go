// +build gofuzz

package protoparse

import (
	"unicode/utf8"
)

// FuzzProtocCompatibility is run by go-fuzz
func FuzzProtocCompatibility(data []byte) int {
	protocBin := getProtocBin()
	if protocBin == "" {
		panic("can't find protoc")
	}
	if !utf8.Valid(data) {
		return -1
	}
	files, err := txtarMap(data)
	if err != nil {
		return -1
	}
	for filename, content := range files {
		var cleanName string
		cleanName, err = cleanProtoFilename(filename)
		if err != nil {
			return -1
		}
		delete(files, filename)
		if _, ok := files[cleanName]; ok {
			return -1
		}
		files[cleanName] = content
	}

	diff, ok, err := compareParseWithProtoc(protocBin, files, &compareParseWithProtocOpts{
		knownIssueMatchers:   knownIssueMatchers,
		ignoreSourceCodeInfo: true,
		ignoreRawFields:      true, // RawFields don't need to match exactly
	})
	if err != nil {
		return -1
	}
	if diff != "" {
		panic(diff)
	}
	if ok {
		return 1
	}
	return 0
}
