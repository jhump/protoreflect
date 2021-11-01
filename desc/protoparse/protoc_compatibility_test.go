package protoparse

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestParser_ParseFiles_ProtocCompatibility(t *testing.T) {
	protocBin := getProtocBin()
	if protocBin == "" {
		t.Skipf("can't find protoc")
	}
	corpusPath := filepath.FromSlash("testdata/protoc_compatibility")
	dir, err := os.ReadDir(corpusPath)
	testutil.Ok(t, err)
	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		t.Run(file.Name(), func(t *testing.T) {
			testdata, err := os.ReadFile(filepath.Join(corpusPath, file.Name()))
			testutil.Ok(t, err)
			files, err := txtarMap(testdata)
			testutil.Ok(t, err)
			diff, _, err := compareParseWithProtoc(protocBin, files, &compareParseWithProtocOpts{
				ignoreSourceCodeInfo: true,
				tmpDir:               t.TempDir(),
			})
			testutil.Ok(t, err)
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
