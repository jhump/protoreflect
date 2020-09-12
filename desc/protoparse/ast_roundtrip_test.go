package protoparse

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jhump/protoreflect/desc/protoparse/ast"
	"github.com/jhump/protoreflect/internal/testutil"
)

func TestASTRoundTrips(t *testing.T) {
	err := filepath.Walk("../../internal/testprotos", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".proto" {
			t.Run(path, func(t *testing.T) {
				b, err := ioutil.ReadFile(path)
				testutil.Ok(t, err)
				data := string(b)
				filename := filepath.Base(path)
				accessor := FileContentsFromMap(map[string]string{filename: data})
				p := Parser{Accessor: accessor}
				root, err := p.ParseToAST(filepath.Base(path))
				testutil.Ok(t, err)
				testutil.Eq(t, 1, len(root))
				var buf bytes.Buffer
				err = ast.Print(&buf, root[0])
				testutil.Ok(t, err)
				// see if file survived round trip!
				testutil.Eq(t, data, buf.String())
			})
		}
		return nil
	})
	testutil.Ok(t, err)
}
