package protoresolve_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestRegistry(t *testing.T) {
	// TODO
	testResolver(t, &protoresolve.Registry{})
}

func TestFromFiles(t *testing.T) {
	// TODO
	reg, err := protoresolve.FromFiles(protoregistry.GlobalFiles)
	require.NoError(t, err)
	testResolver(t, reg)

	reg, err = protoresolve.FromFiles(&protoregistry.Files{})
	require.NoError(t, err)
	testResolver(t, reg)
}
