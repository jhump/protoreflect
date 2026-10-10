package protoresolve_test

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestRegistryProtoFromFileDescriptor(t *testing.T) {
	t.Parallel()
	file := testprotos.File_desc_test1_proto

	t.Run("registered proto", func(t *testing.T) {
		t.Parallel()
		reg := &protoresolve.Registry{}
		fileProto := protodesc.ToFileDescriptorProto(file)
		registered, err := reg.RegisterFileProto(fileProto)
		require.NoError(t, err)
		got, err := reg.ProtoFromFileDescriptor(registered)
		require.NoError(t, err)
		assert.Same(t, fileProto, got)
	})
	t.Run("registered file", func(t *testing.T) {
		t.Parallel()
		reg := newRegistry(t, resolvertest.Corpus())
		got, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		checkProtoEqual(t, protodesc.ToFileDescriptorProto(file), got)
		again, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		assert.Same(t, got, again, "proto should be memoized")
	})
	t.Run("file import", func(t *testing.T) {
		t.Parallel()
		reg := newRegistry(t, resolvertest.Corpus())
		imp := testprotos.File_desc_test2_proto.Imports().Get(0)
		require.Equal(t, file.Path(), imp.Path())
		got, err := reg.ProtoFromFileDescriptor(imp)
		require.NoError(t, err)
		want, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		assert.Same(t, want, got)
	})
	t.Run("unregistered file", func(t *testing.T) {
		t.Parallel()
		reg := &protoresolve.Registry{}
		got, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		checkProtoEqual(t, protodesc.ToFileDescriptorProto(file), got)
		registered, err := reg.FindFileByPath(file.Path())
		require.NoError(t, err, "file should be registered")
		assert.Equal(t, file, registered)
		again, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		assert.Same(t, got, again, "proto should be memoized")
	})
	t.Run("different file with same path", func(t *testing.T) {
		t.Parallel()
		reg := &protoresolve.Registry{}
		registered, err := reg.RegisterFileProto(protodesc.ToFileDescriptorProto(file))
		require.NoError(t, err)
		got, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		checkProtoEqual(t, protodesc.ToFileDescriptorProto(file), got)
		again, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		assert.NotSame(t, got, again, "proto should not be memoized")
		found, err := reg.FindFileByPath(file.Path())
		require.NoError(t, err)
		assert.Equal(t, registered, found, "registered file should be unchanged")
	})
	t.Run("file that cannot be registered", func(t *testing.T) {
		t.Parallel()
		reg := newRegistry(t, []protoreflect.FileDescriptor{file})
		// Same contents as a registered file, so its symbols conflict.
		otherProto := protodesc.ToFileDescriptorProto(file)
		otherProto.Name = new("other.proto")
		other, err := protodesc.NewFile(otherProto, nil)
		require.NoError(t, err)
		got, err := reg.ProtoFromFileDescriptor(other)
		require.NoError(t, err)
		checkProtoEqual(t, otherProto, got)
		_, err = reg.FindFileByPath(other.Path())
		assert.ErrorIs(t, err, protoresolve.ErrNotFound)
		again, err := reg.ProtoFromFileDescriptor(other)
		require.NoError(t, err)
		assert.NotSame(t, got, again, "proto should not be memoized")
	})
}

func TestNewProtoOracle(t *testing.T) {
	t.Parallel()
	corpus := resolvertest.Corpus()
	reg := newRegistry(t, corpus)
	oracle := protoresolve.NewProtoOracle(reg)

	for _, file := range corpus {
		got, err := oracle.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		want, err := reg.ProtoFromFileDescriptor(file)
		require.NoError(t, err)
		assert.Same(t, want, got)
		fromDescriptor, err := oracle.ProtoFromDescriptor(file)
		require.NoError(t, err)
		assert.Same(t, want, fromDescriptor)
	}

	for _, descriptor := range resolvertest.Descriptors(corpus) {
		got, err := oracle.ProtoFromDescriptor(descriptor)
		if !assert.NoError(t, err, "descriptor %s", descriptor.FullName()) {
			continue
		}
		var want, typed proto.Message
		var typedErr error
		switch descriptor := descriptor.(type) {
		case protoreflect.MessageDescriptor:
			want = protodesc.ToDescriptorProto(descriptor)
			typed, typedErr = oracle.ProtoFromMessageDescriptor(descriptor)
		case protoreflect.FieldDescriptor:
			want = protodesc.ToFieldDescriptorProto(descriptor)
			typed, typedErr = oracle.ProtoFromFieldDescriptor(descriptor)
		case protoreflect.OneofDescriptor:
			want = protodesc.ToOneofDescriptorProto(descriptor)
			typed, typedErr = oracle.ProtoFromOneofDescriptor(descriptor)
		case protoreflect.EnumDescriptor:
			want = protodesc.ToEnumDescriptorProto(descriptor)
			typed, typedErr = oracle.ProtoFromEnumDescriptor(descriptor)
		case protoreflect.EnumValueDescriptor:
			want = protodesc.ToEnumValueDescriptorProto(descriptor)
			typed, typedErr = oracle.ProtoFromEnumValueDescriptor(descriptor)
		case protoreflect.ServiceDescriptor:
			want = protodesc.ToServiceDescriptorProto(descriptor)
			typed, typedErr = oracle.ProtoFromServiceDescriptor(descriptor)
		case protoreflect.MethodDescriptor:
			want = protodesc.ToMethodDescriptorProto(descriptor)
			typed, typedErr = oracle.ProtoFromMethodDescriptor(descriptor)
		default:
			t.Fatalf("unexpected descriptor type %T", descriptor)
		}
		checkProtoEqual(t, want, got)
		if assert.NoError(t, typedErr, "descriptor %s", descriptor.FullName()) {
			assert.Same(t, got, typed, "descriptor %s", descriptor.FullName())
		}
	}
}

func TestNewProtoOracleErrors(t *testing.T) {
	t.Parallel()
	file := testprotos.File_desc_test1_proto
	require.GreaterOrEqual(t, file.Messages().Len(), 2)
	firstMsg, secondMsg := file.Messages().Get(0), file.Messages().Get(1)

	t.Run("file oracle error", func(t *testing.T) {
		t.Parallel()
		errOracle := errors.New("oracle failure")
		oracle := protoresolve.NewProtoOracle(resolvertest.ProtoFileOracleFunc(
			func(protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error) {
				return nil, errOracle
			},
		))
		_, err := oracle.ProtoFromMessageDescriptor(firstMsg)
		assert.ErrorIs(t, err, errOracle)
	})
	t.Run("unsupported descriptor type", func(t *testing.T) {
		t.Parallel()
		oracle := protoresolve.NewProtoOracle(&protoresolve.Registry{})
		_, err := oracle.ProtoFromDescriptor(struct{ protoreflect.Descriptor }{firstMsg})
		assert.ErrorContains(t, err, "unsupported descriptor type")
	})
	t.Run("descriptor without parent", func(t *testing.T) {
		t.Parallel()
		oracle := protoresolve.NewProtoOracle(&protoresolve.Registry{})
		_, err := oracle.ProtoFromMessageDescriptor(parentlessMessage{firstMsg})
		assert.ErrorContains(t, err, "has no parent")
	})

	mismatchedOracle := func(mutate func(*descriptorpb.FileDescriptorProto)) protoresolve.ProtoOracle {
		fileProto := protodesc.ToFileDescriptorProto(file)
		mutate(fileProto)
		return protoresolve.NewProtoOracle(resolvertest.ProtoFileOracleFunc(
			func(protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error) {
				return fileProto, nil
			},
		))
	}
	t.Run("no candidates", func(t *testing.T) {
		t.Parallel()
		oracle := mismatchedOracle(func(fileProto *descriptorpb.FileDescriptorProto) {
			fileProto.MessageType = nil
		})
		_, err := oracle.ProtoFromMessageDescriptor(firstMsg)
		assert.ErrorContains(t, err, "no candidate children")
		// Errors for a parent are reported for its children, too.
		_, err = oracle.ProtoFromFieldDescriptor(firstMsg.Fields().Get(0))
		assert.ErrorContains(t, err, "no candidate children")
	})
	t.Run("too few candidates", func(t *testing.T) {
		t.Parallel()
		oracle := mismatchedOracle(func(fileProto *descriptorpb.FileDescriptorProto) {
			fileProto.MessageType = fileProto.MessageType[:1]
		})
		_, err := oracle.ProtoFromMessageDescriptor(secondMsg)
		assert.ErrorContains(t, err, "descriptor index is 1")
	})
	t.Run("wrong name", func(t *testing.T) {
		t.Parallel()
		oracle := mismatchedOracle(func(fileProto *descriptorpb.FileDescriptorProto) {
			fileProto.MessageType[0].Name = new("Foo")
		})
		_, err := oracle.ProtoFromMessageDescriptor(firstMsg)
		assert.ErrorContains(t, err, `found descriptor with name "Foo"`)
	})
}

type parentlessMessage struct {
	protoreflect.MessageDescriptor
}

func (parentlessMessage) Parent() protoreflect.Descriptor {
	return nil
}

func checkProtoEqual(t *testing.T, want, got proto.Message) {
	t.Helper()
	assert.Empty(t, cmp.Diff(want, got, protocmp.Transform()))
}
