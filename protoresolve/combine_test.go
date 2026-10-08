package protoresolve_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestCombineEmpty(t *testing.T) {
	t.Parallel()
	res := protoresolve.Combine()
	assert.Zero(t, res.NumFiles())
	assert.Zero(t, res.NumFilesByPackage("testprotos"))
	res.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		t.Errorf("unexpected file %s", file.Path())
		return true
	})
	_, err := res.FindFileByPath("desc_test1.proto")
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	_, err = res.FindDescriptorByName("testprotos.TestMessage")
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	_, err = res.FindMessageByName("testprotos.TestMessage")
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	_, err = res.FindMessageByURL("type.googleapis.com/testprotos.TestMessage")
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	_, err = res.FindExtensionByName("testprotos.xtm")
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	_, err = res.FindExtensionByNumber("testprotos.AnotherTestMessage", 100)
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	_, err = res.AsTypeResolver().FindMessageByName("testprotos.TestMessage")
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
}

func TestCombinePrefersFirst(t *testing.T) {
	t.Parallel()
	corpus := resolvertest.Corpus()
	first := newRegistry(t, corpus[:len(corpus)/2])
	second := &protoresolve.Registry{}
	for _, fileProto := range fileProtos(corpus) {
		_, err := second.RegisterFileProto(fileProto)
		require.NoError(t, err)
	}
	for name, res := range map[string]protoresolve.Resolver{
		"Combine":      protoresolve.Combine(first, second),
		"CombinePools": protoresolve.CombinePools(first, second),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Counts only include the first resolver.
			assert.Equal(t, first.NumFiles(), res.NumFiles())
			assert.Equal(t, first.NumFilesByPackage("testprotos"), res.NumFilesByPackage("testprotos"))

			// Files in both resolvers come from the first.
			want := map[string]protoreflect.FileDescriptor{}
			for _, file := range corpus {
				found, err := first.FindFileByPath(file.Path())
				if err != nil {
					found, err = second.FindFileByPath(file.Path())
					require.NoError(t, err)
				}
				want[file.Path()] = found
			}
			for path, wantFile := range want {
				found, err := res.FindFileByPath(path)
				require.NoError(t, err)
				assert.Equal(t, wantFile, found, "file %s", path)
			}
			got := map[string]protoreflect.FileDescriptor{}
			res.RangeFiles(func(file protoreflect.FileDescriptor) bool {
				got[file.Path()] = file
				return true
			})
			assert.Equal(t, want, got)
		})
	}
}

func TestCombineWrongKind(t *testing.T) {
	t.Parallel()
	// When a resolver finds an element of the wrong kind, the combined
	// resolver should return that error instead of falling back to later
	// resolvers, which might have an element of the right kind.
	first := newRegistry(t, []protoreflect.FileDescriptor{testprotos.File_desc_test1_proto})
	otherProto := fileProto("other.proto", "testprotos")
	otherProto.MessageType = []*descriptorpb.DescriptorProto{{Name: proto.String("SomeEnum")}}
	other, err := protodesc.NewFile(otherProto, nil)
	require.NoError(t, err)
	second := newRegistry(t, []protoreflect.FileDescriptor{other})
	res := protoresolve.Combine(first, second)
	_, err = res.FindMessageByName("testprotos.SomeEnum")
	var unexpectedTypeErr *protoresolve.ErrUnexpectedType
	assert.ErrorAs(t, err, &unexpectedTypeErr)
	_, err = res.AsTypeResolver().FindMessageByName("testprotos.SomeEnum")
	assert.ErrorAs(t, err, &unexpectedTypeErr)
}
