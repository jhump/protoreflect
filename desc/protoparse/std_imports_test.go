package protoparse

import (
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestStdImports(t *testing.T) {
	// make sure we can successfully parse all standard imports
	var p Parser
	for name, fileProto := range standardImports {
		fds, err := p.ParseFiles(name)
		testutil.Ok(t, err)
		testutil.Eq(t, 1, len(fds))
		testutil.Require(t, proto.Equal(fileProto, fds[0].AsFileDescriptorProto()))
	}
}
