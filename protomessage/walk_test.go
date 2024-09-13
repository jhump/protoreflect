package protomessage_test

import (
	"fmt"
	"testing"

	"github.com/bufbuild/protocompile/walk"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	"github.com/jhump/protoreflect/v2/protomessage"
	"github.com/jhump/protoreflect/v2/protoresolve"
	"github.com/jhump/protoreflect/v2/sourceloc"
)

func TestWalk(t *testing.T) {
	fd, err := prototesting.LoadProtoset("../internal/testprotos/desc_test_complex_source_info.protoset")
	require.NoError(t, err)

	fdProto := protodesc.ToFileDescriptorProto(fd)
	protoOracle := protoresolve.NewProtoOracle(oracleForFile{fd, fdProto})
	paths := map[proto.Message]protoreflect.SourcePath{}
	err = walk.Descriptors(fd, func(d protoreflect.Descriptor) error {
		msg, err := protoOracle.ProtoFromDescriptor(d)
		if err != nil {
			return err
		}
		paths[msg] = sourceloc.PathFor(d)
		return nil
	})
	require.NoError(t, err)

	encountered := make(map[proto.Message]struct{}, len(paths))
	protomessage.Walk(fdProto.ProtoReflect(), func(path []any, msg protoreflect.Message) bool {
		expectPath, ok := paths[msg.Interface()]
		if !ok {
			// skip
			return true
		}
		encountered[msg.Interface()] = struct{}{}
		actualPath := make(protoreflect.SourcePath, len(path))
		for i := range path {
			switch p := path[i].(type) {
			case protoreflect.FieldNumber:
				actualPath[i] = int32(p)
			case int: // index into list field
				actualPath[i] = int32(p)
			default:
				t.Fatalf("unexpected element at path[i]: %T", p)
			}
		}
		require.Equal(t, expectPath, actualPath)
		return true
	})
	require.Len(t, encountered, len(paths))
}

type oracleForFile struct {
	fd      protoreflect.FileDescriptor
	fdProto *descriptorpb.FileDescriptorProto
}

func (o oracleForFile) ProtoFromFileDescriptor(file protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error) {
	if file == o.fd {
		return o.fdProto, nil
	}
	return nil, fmt.Errorf("unexpected file: %s", file.Path())
}
