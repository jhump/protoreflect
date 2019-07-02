package protoparse

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestResolveFilenames(t *testing.T) {
	dir, err := ioutil.TempDir("", "resolve-filenames-test")
	testutil.Ok(t, err)
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	testCases := []struct {
		name        string
		importPaths []string
		fileNames   []string
		files       []string
		expectedErr string
		results     []string
	}{
		{
			name:      "no import paths",
			fileNames: []string{"test1.proto", "test2.proto", "test3.proto"},
			results:   []string{"test1.proto", "test2.proto", "test3.proto"},
		},
		{
			name:        "no import paths; absolute file name",
			fileNames:   []string{filepath.Join(dir, "test.proto")},
			expectedErr: errNoImportPathsForAbsoluteFilePath.Error(),
		},
		{
			name:        "import and file paths relative to cwd",
			importPaths: []string{"./test"},
			fileNames:   []string{"./test/a.proto", "./test/b.proto", "./test/c.proto"},
			results:     []string{"a.proto", "b.proto", "c.proto"},
		},
		{
			name:        "absolute file paths, import paths relative to cwd",
			importPaths: []string{"./test"},
			fileNames:   []string{filepath.Join(dir, "test/a.proto"), filepath.Join(dir, "test/b.proto"), filepath.Join(dir, "test/c.proto")},
			results:     []string{"a.proto", "b.proto", "c.proto"},
		},
		{
			name:        "absolute import paths, file paths relative to cwd",
			importPaths: []string{filepath.Join(dir, "test")},
			fileNames:   []string{"./test/a.proto", "./test/b.proto", "./test/c.proto"},
			results:     []string{"a.proto", "b.proto", "c.proto"},
		},
		{
			name:        "file path relative to cwd not in import path",
			importPaths: []string{filepath.Join(dir, "test")},
			fileNames:   []string{"./test/a.proto", "./test/b.proto", "./foo/c.proto"},
			expectedErr: "./foo/c.proto does not reside in any import path",
		},
		{
			name:        "absolute file path not in import path",
			importPaths: []string{filepath.Join(dir, "test")},
			fileNames:   []string{"./test/a.proto", "./test/b.proto", filepath.Join(dir, "foo/c.proto")},
			expectedErr: filepath.Join(dir, "foo/c.proto") + " does not reside in any import path",
		},
		{
			name:        "relative paths, files relative to import path",
			files:       []string{"test/a.proto", "test/b.proto", "test/c.proto"},
			importPaths: []string{"test"},
			fileNames:   []string{"a.proto", "b.proto", "c.proto"},
			results:     []string{"a.proto", "b.proto", "c.proto"},
		},
		{
			name:        "relative paths, files relative to mix",
			files:       []string{"test/a.proto", "test/b.proto", "test/c.proto"},
			importPaths: []string{"test"},
			fileNames:   []string{"test/a.proto", "b.proto", "test/c.proto"},
			results:     []string{"a.proto", "b.proto", "c.proto"},
		},
	}

	origCwd, err := os.Getwd()
	testutil.Ok(t, err)

	err = os.Chdir(dir)
	testutil.Ok(t, err)
	defer func() {
		_ = os.Chdir(origCwd)
	}()

	for _, tc := range testCases {
		// setup any test files
		for _, f := range tc.files {
			subDir := filepath.Dir(f)
			if subDir != "." {
				err := os.MkdirAll(filepath.Join(dir, subDir), os.ModePerm)
				testutil.Ok(t, err)
			}
			err := ioutil.WriteFile(filepath.Join(dir, f), nil, 0666)
			testutil.Ok(t, err)
		}

		// run the function under test
		res, err := ResolveFilenames(tc.importPaths, tc.fileNames...)
		// assert outcome
		if tc.expectedErr != "" {
			testutil.Nok(t, err, "%s", tc.name)
			testutil.Eq(t, tc.expectedErr, err.Error(), "%s", tc.name)
		} else {
			testutil.Ok(t, err, "%s", tc.name)
			testutil.Eq(t, tc.results, res, "%s", tc.name)
		}

		// remove test files
		for _, f := range tc.files {
			err := os.Remove(filepath.Join(dir, f))
			testutil.Ok(t, err)
		}
	}
}
