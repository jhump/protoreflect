package protoparse

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestStdImports(t *testing.T) {
	// make sure we can successfully parse all standard imports
	p := Parser{}
	for name, fileProto := range standardImports {
		fds, err := p.ParseFiles(name)
		testutil.Ok(t, err)
		testutil.Eq(t, 1, len(fds))
		sourceCodeInfo := fileProto.SourceCodeInfo
		fileProto.SourceCodeInfo = nil
		testutil.Require(t, proto.Equal(fileProto, fds[0].AsFileDescriptorProto()))
		fileProto.SourceCodeInfo = sourceCodeInfo
	}
	p = Parser{IncludeSourceCodeInfo: true}
	for name, fileProto := range standardImports {
		fds, err := p.ParseFiles(name)
		testutil.Ok(t, err)
		testutil.Eq(t, 1, len(fds))
		testutil.Require(t, proto.Equal(fileProto, fds[0].AsFileDescriptorProto()))
	}
}
