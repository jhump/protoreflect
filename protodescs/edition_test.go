package protodescs_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protodescs"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestGetEdition(t *testing.T) {
	t.Parallel()
	editionsFile := testprotos.File_desc_test_editions_proto
	// Hides the file's Edition method, so the edition must come from elsewhere.
	hiddenEdition := struct{ protoreflect.FileDescriptor }{editionsFile}
	oracle := resolvertest.ProtoFileOracleFunc(func(protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error) {
		return &descriptorpb.FileDescriptorProto{Edition: new(descriptorpb.Edition_EDITION_2024)}, nil
	})
	failingOracle := resolvertest.ProtoFileOracleFunc(func(protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error) {
		return nil, errors.New("oracle failure")
	})

	testCases := []struct {
		name     string
		file     protoreflect.FileDescriptor
		oracle   protoresolve.ProtoFileOracle
		expected descriptorpb.Edition
	}{
		{"proto2", testprotos.File_desc_test1_proto, nil, descriptorpb.Edition_EDITION_PROTO2},
		{"proto3", testprotos.File_desc_test_proto3_proto, nil, descriptorpb.Edition_EDITION_PROTO3},
		{"editions", editionsFile, oracle, descriptorpb.Edition_EDITION_2023},
		{"editions from oracle", hiddenEdition, oracle, descriptorpb.Edition_EDITION_2024},
		// protodesc can't determine the edition either, without the Edition method.
		{"editions with failing oracle", hiddenEdition, failingOracle, descriptorpb.Edition_EDITION_UNKNOWN},
		{"editions without oracle", hiddenEdition, nil, descriptorpb.Edition_EDITION_UNKNOWN},
		{"unknown syntax", unknownSyntaxFile{editionsFile}, oracle, descriptorpb.Edition_EDITION_UNKNOWN},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, testCase.expected, protodescs.GetEdition(testCase.file, testCase.oracle))
		})
	}
}

type unknownSyntaxFile struct {
	protoreflect.FileDescriptor
}

func (unknownSyntaxFile) Syntax() protoreflect.Syntax {
	return 0
}
