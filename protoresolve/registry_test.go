package protoresolve_test

import (
	"fmt"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/resolvertest"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/internal/testprotos/pkg"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestRegistryRegisterFileErrors(t *testing.T) {
	t.Parallel()

	t.Run("duplicate file", func(t *testing.T) {
		t.Parallel()
		// The first file has extensions and the second does not. Both are
		// reported the same way.
		for _, file := range []protoreflect.FileDescriptor{testprotos.File_desc_test1_proto, pkg.File_pkg_desc_test_pkg_proto} {
			reg := newRegistry(t, []protoreflect.FileDescriptor{file})
			err := reg.RegisterFile(file)
			assert.EqualError(t, err, fmt.Sprintf("file %q already registered", file.Path()))
		}
	})
	t.Run("conflicting extension", func(t *testing.T) {
		t.Parallel()
		reg := newRegistry(t, []protoreflect.FileDescriptor{testprotos.File_desc_test1_proto})
		conflictProto := conflictingExtensionFileProto()
		conflict, err := protodesc.NewFile(conflictProto, reg)
		require.NoError(t, err)
		err = reg.RegisterFile(conflict)
		assert.ErrorContains(t, err, `extension number 100 for message "testprotos.AnotherTestMessage" already registered`)
		_, err = reg.RegisterFileProto(conflictProto)
		assert.ErrorContains(t, err, `extension number 100 for message "testprotos.AnotherTestMessage" already registered`)
		_, err = reg.FindFileByPath(conflictProto.GetName())
		assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	})
	t.Run("conflicting extensions in same file", func(t *testing.T) {
		t.Parallel()
		fileProto := fileProto("dup.proto", "dup", "google/protobuf/descriptor.proto")
		fileProto.Extension = []*descriptorpb.FieldDescriptorProto{
			int32Extension("a", 50000, "google.protobuf.MessageOptions"),
			int32Extension("b", 50000, "google.protobuf.MessageOptions"),
		}
		// protodesc permits this, but the registry can only index one of them.
		file, err := protodesc.NewFile(fileProto, protoregistry.GlobalFiles)
		require.NoError(t, err)
		reg := &protoresolve.Registry{}
		const errSubstr = `extension number 50000 for message "google.protobuf.MessageOptions" already registered`
		assert.ErrorContains(t, reg.RegisterFile(file), errSubstr)
		files := newFiles(t, []protoreflect.FileDescriptor{file})
		_, err = protoresolve.FromFiles(files)
		assert.ErrorContains(t, err, errSubstr)
	})
	t.Run("missing dependency", func(t *testing.T) {
		t.Parallel()
		reg := &protoresolve.Registry{}
		_, err := reg.RegisterFileProto(protodesc.ToFileDescriptorProto(testprotos.File_desc_test2_proto))
		assert.ErrorContains(t, err, "desc_test1.proto")
	})
}

func TestFromFilesErrors(t *testing.T) {
	t.Parallel()
	files := newFiles(t, []protoreflect.FileDescriptor{testprotos.File_desc_test1_proto})
	conflict, err := protodesc.NewFile(conflictingExtensionFileProto(), files)
	require.NoError(t, err)
	// Unlike Registry, protoregistry.Files allows conflicting extension numbers.
	require.NoError(t, files.RegisterFile(conflict))
	_, err = protoresolve.FromFiles(files)
	assert.ErrorContains(t, err, `extension number 100 for message "testprotos.AnotherTestMessage" already registered`)
}

func TestFromFilesCopiesFiles(t *testing.T) {
	t.Parallel()
	files := newFiles(t, []protoreflect.FileDescriptor{testprotos.File_desc_test1_proto})
	reg, err := protoresolve.FromFiles(files)
	require.NoError(t, err)

	// Changes to one don't affect the other.
	require.NoError(t, files.RegisterFile(pkg.File_pkg_desc_test_pkg_proto))
	_, err = reg.FindFileByPath(pkg.File_pkg_desc_test_pkg_proto.Path())
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	require.NoError(t, reg.RegisterFile(testprotos.File_desc_test_proto3_proto))
	_, err = files.FindFileByPath(testprotos.File_desc_test_proto3_proto.Path())
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)

	assert.Equal(t, 2, files.NumFiles())
	assert.Equal(t, 2, reg.NumFiles())
	var count int
	reg.RangeFiles(func(protoreflect.FileDescriptor) bool {
		count++
		return true
	})
	assert.Equal(t, 2, count)
}

func TestFromFileDescriptorSetErrors(t *testing.T) {
	t.Parallel()
	_, err := protoresolve.FromFileDescriptorSet(&descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(testprotos.File_desc_test1_proto),
			protodesc.ToFileDescriptorProto(testprotos.File_desc_test1_proto),
		},
	})
	assert.ErrorContains(t, err, `duplicate file "desc_test1.proto"`)

	_, err = protoresolve.FromFileDescriptorSet(&descriptorpb.FileDescriptorSet{
		File: []*descriptorpb.FileDescriptorProto{
			protodesc.ToFileDescriptorProto(testprotos.File_desc_test1_proto),
			conflictingExtensionFileProto(),
		},
	})
	assert.ErrorContains(t, err, `register "conflict.proto"`)
}

func TestRegistryRegisterFileProtoCustomOptions(t *testing.T) {
	t.Parallel()
	// Round-trip the proto through the binary format without knowledge of
	// the file's custom options, so they are all unrecognized fields.
	data, err := proto.Marshal(protodesc.ToFileDescriptorProto(testprotos.File_desc_test_complex_proto))
	require.NoError(t, err)
	var fileProto descriptorpb.FileDescriptorProto
	require.NoError(t, proto.UnmarshalOptions{Resolver: &protoregistry.Types{}}.Unmarshal(data, &fileProto))
	require.True(t, hasUnknownFields(fileProto.ProtoReflect()), "test proto should have unrecognized options")

	reg := newRegistry(t, []protoreflect.FileDescriptor{descriptorpb.File_google_protobuf_descriptor_proto})
	file, err := reg.RegisterFileProto(&fileProto)
	require.NoError(t, err)
	assert.False(t, hasUnknownFields(fileProto.ProtoReflect()), "custom options should be recognized")

	msg := protoresolve.FindDescriptorByNameInFile(file, "foo.bar.Test.Nested._NestedNested")
	require.NotNil(t, msg)
	var hasOption bool
	msg.Options().ProtoReflect().Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		hasOption = hasOption || field.FullName() == "foo.bar.Test.Nested.fooblez"
		return true
	})
	assert.True(t, hasOption, "message options of %s should include custom option", msg.FullName())

	// If registration fails, the given proto is not modified.
	var duplicateProto descriptorpb.FileDescriptorProto
	require.NoError(t, proto.UnmarshalOptions{Resolver: &protoregistry.Types{}}.Unmarshal(data, &duplicateProto))
	_, err = reg.RegisterFileProto(&duplicateProto)
	require.ErrorContains(t, err, "already registered")
	assert.True(t, hasUnknownFields(duplicateProto.ProtoReflect()), "proto should not be modified")
}

func TestRegistryRegisterFileProtoUnknownOptions(t *testing.T) {
	t.Parallel()
	// Round-trip the proto through the binary format without knowledge of
	// the file's custom options, so they are all unrecognized fields.
	data, err := proto.Marshal(protodesc.ToFileDescriptorProto(testprotos.File_desc_test_complex_proto))
	require.NoError(t, err)
	var fileProto descriptorpb.FileDescriptorProto
	require.NoError(t, proto.UnmarshalOptions{Resolver: &protoregistry.Types{}}.Unmarshal(data, &fileProto))
	// Add an option that no file defines next to the custom options of a
	// nested message.
	var msgProto *descriptorpb.DescriptorProto
	for _, name := range []string{"Test", "Nested", "_NestedNested"} {
		candidates := fileProto.MessageType
		if msgProto != nil {
			candidates = msgProto.NestedType
		}
		idx := slices.IndexFunc(candidates, func(candidate *descriptorpb.DescriptorProto) bool {
			return candidate.GetName() == name
		})
		require.NotEqual(t, -1, idx, "message %s", name)
		msgProto = candidates[idx]
	}
	opts := msgProto.GetOptions().ProtoReflect()
	opts.SetUnknown(protowire.AppendVarint(protowire.AppendTag(opts.GetUnknown(), 59999, protowire.VarintType), 1))

	reg := newRegistry(t, []protoreflect.FileDescriptor{descriptorpb.File_google_protobuf_descriptor_proto})
	file, err := reg.RegisterFileProto(&fileProto)
	require.NoError(t, err)

	// The custom options are still recognized, despite the unknown one.
	msg := protoresolve.FindDescriptorByNameInFile(file, "foo.bar.Test.Nested._NestedNested")
	require.NotNil(t, msg)
	var recognized []protoreflect.FullName
	msg.Options().ProtoReflect().Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		recognized = append(recognized, field.FullName())
		return true
	})
	assert.ElementsMatch(t, []protoreflect.FullName{"foo.bar.Test.Nested.fooblez", "foo.bar.rept"}, recognized)
	assert.NotEmpty(t, msg.Options().ProtoReflect().GetUnknown(), "unknown option should remain unknown")
}

func TestRegistryConcurrency(t *testing.T) {
	t.Parallel()
	corpus := resolvertest.Corpus()
	reg := &protoresolve.Registry{}
	var waitGroup sync.WaitGroup
	for _, file := range corpus {
		waitGroup.Add(2)
		go func() {
			defer waitGroup.Done()
			if err := reg.RegisterFile(file); err != nil {
				// ProtoFromFileDescriptor, called concurrently below, may have
				// registered it first.
				assert.ErrorContains(t, err, "already registered")
			}
		}()
		go func() {
			defer waitGroup.Done()
			// Results depend on timing; this is only checking for data races.
			_, _ = reg.FindFileByPath(file.Path())
			_, _ = reg.FindDescriptorByName("testprotos.TestMessage")
			_, _ = reg.FindExtensionByNumber("testprotos.AnotherTestMessage", 100)
			reg.RangeFiles(func(protoreflect.FileDescriptor) bool { return true })
			reg.RangeFilesByPackage("testprotos", func(protoreflect.FileDescriptor) bool { return true })
			reg.RangeExtensionsByMessage("testprotos.AnotherTestMessage", func(protoreflect.ExtensionDescriptor) bool { return true })
			_, _ = reg.ProtoFromFileDescriptor(file)
		}()
	}
	waitGroup.Wait()
	resolvertest.CheckResolver(t, reg, corpus)
}

func hasUnknownFields(msg protoreflect.Message) bool {
	if len(msg.GetUnknown()) > 0 {
		return true
	}
	var found bool
	msg.Range(func(field protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		switch {
		case field.IsMap():
			if field.MapValue().Message() != nil {
				val.Map().Range(func(_ protoreflect.MapKey, entry protoreflect.Value) bool {
					found = hasUnknownFields(entry.Message())
					return !found
				})
			}
		case field.Message() == nil:
		case field.IsList():
			list := val.List()
			for i := 0; i < list.Len() && !found; i++ {
				found = hasUnknownFields(list.Get(i).Message())
			}
		default:
			found = hasUnknownFields(val.Message())
		}
		return !found
	})
	return found
}
