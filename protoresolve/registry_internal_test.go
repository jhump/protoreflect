package protoresolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestExtResolverForFile(t *testing.T) {
	t.Parallel()
	extension := func(name string, number int32) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{
			Name:     new(name),
			Number:   new(number),
			Label:    new(descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL),
			Type:     new(descriptorpb.FieldDescriptorProto_TYPE_INT32),
			Extendee: new(".dep.Extendee"),
		}
	}
	// The registry has a dependency of the file being registered.
	reg := &Registry{}
	dep, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:    new("dep.proto"),
		Package: new("dep"),
		MessageType: []*descriptorpb.DescriptorProto{{
			Name:           new("Extendee"),
			ExtensionRange: []*descriptorpb.DescriptorProto_ExtensionRange{{Start: new(int32(100)), End: new(int32(200))}},
		}},
		Extension: []*descriptorpb.FieldDescriptorProto{extension("dep_ext", 100)},
	}, nil)
	require.NoError(t, err)
	require.NoError(t, reg.RegisterFile(dep))
	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name:        new("file.proto"),
		Package:     new("file"),
		Dependency:  []string{"dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{{Name: new("Msg")}},
		Extension:   []*descriptorpb.FieldDescriptorProto{extension("file_ext", 101)},
	}, reg)
	require.NoError(t, err)
	res := &extResolverForFile{f: file, r: reg}

	t.Run("by name", func(t *testing.T) {
		t.Parallel()
		for _, name := range []protoreflect.FullName{"dep.dep_ext", "file.file_ext"} {
			extType, err := res.FindExtensionByName(name)
			if assert.NoError(t, err, "extension %s", name) {
				assert.Equal(t, name, extType.TypeDescriptor().FullName())
			}
		}

		_, err := res.FindExtensionByName("file.unknown")
		// Unmarshalling requires exactly ErrNotFound, not a wrapped error.
		assert.Same(t, ErrNotFound, err)

		// Elements of the wrong kind, in the registry and in the file.
		for _, name := range []protoreflect.FullName{"dep.Extendee", "file.Msg"} {
			_, err := res.FindExtensionByName(name)
			var unexpectedTypeErr *ErrUnexpectedType
			if assert.ErrorAs(t, err, &unexpectedTypeErr, "element %s", name) {
				assert.Equal(t, DescriptorKindMessage, unexpectedTypeErr.Actual)
			}
		}
	})
	t.Run("by number", func(t *testing.T) {
		t.Parallel()
		for number, name := range map[protoreflect.FieldNumber]protoreflect.FullName{100: "dep.dep_ext", 101: "file.file_ext"} {
			extType, err := res.FindExtensionByNumber("dep.Extendee", number)
			if assert.NoError(t, err, "extension %d", number) {
				assert.Equal(t, name, extType.TypeDescriptor().FullName())
			}
		}

		_, err := res.FindExtensionByNumber("dep.Extendee", 102)
		assert.Same(t, ErrNotFound, err)
	})
}
