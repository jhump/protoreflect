package protoparse

import (
	"testing"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestStdImports(t *testing.T) {
	// make sure we can successfully parse all standard imports
	var p Parser
	for name, proto := range standardImports {
		fds, err := p.ParseFiles(name)
		testutil.Ok(t, err)
		testutil.Eq(t, 1, len(fds))
		testutil.Eq(t, proto, fds[0].AsFileDescriptorProto())
	}
}
