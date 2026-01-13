package protoresolve_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestRegistry(t *testing.T) {
	reg := &protoresolve.Registry{}
	// Register the file locally (safe, as 'reg' is a new variable)
	require.NoError(t, reg.RegisterFile(testprotos.File_desc_test1_proto))

	testResolver(t, reg)
}

func TestFromFilaes(t *testing.T) {
	// 1. Setup GlobalFiles (SAFE VERSION)
	// We check if the file is already there. If NOT found, we register it.
	// This prevents the "already registered" panic.
	if _, err := protoregistry.GlobalFiles.FindFileByPath("desc_test1.proto"); err != nil {
		_ = protoregistry.GlobalFiles.RegisterFile(testprotos.File_desc_test1_proto)
	}

	reg, err := protoresolve.FromFiles(protoregistry.GlobalFiles)
	require.NoError(t, err)
	testResolver(t, reg)

	// 2. Setup local Files
	var files protoregistry.Files
	require.NoError(t, files.RegisterFile(testprotos.File_desc_test1_proto))

	reg, err = protoresolve.FromFiles(&files)
	require.NoError(t, err)
	testResolver(t, reg)
}
