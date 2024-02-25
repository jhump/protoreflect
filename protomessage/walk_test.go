package protomessage_test

import (
	"testing"

	"github.com/bufbuild/protocompile/walk"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	prototesting "github.com/jhump/protoreflect/v2/internal/testing"
	"github.com/jhump/protoreflect/v2/protomessage"
	"github.com/jhump/protoreflect/v2/protowrap"
	"github.com/jhump/protoreflect/v2/sourceloc"
)

func TestWalk(t *testing.T) {
	fd, err := prototesting.LoadProtoset("../internal/testdata/desc_test_complex_source_info.protoset")
	require.NoError(t, err)

	paths := map[proto.Message]protoreflect.SourcePath{}
	err = walk.Descriptors(fd, func(d protoreflect.Descriptor) error {
		msg := protowrap.ProtoFromDescriptor(d)
		paths[msg] = sourceloc.PathFor(d)
		return nil
	})
	require.NoError(t, err)

	encountered := make(map[proto.Message]struct{}, len(paths))
	fdProto := protowrap.ProtoFromFileDescriptor(fd)
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
