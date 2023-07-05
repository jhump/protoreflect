package protoresolve_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal/testdata"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestReparse(t *testing.T) {
	fileDescriptor := protodesc.ToFileDescriptorProto(testdata.File_desc_test_complex_proto)
	// serialize to bytes and back, but use empty resolver when
	// de-serializing so that custom options are unrecognized
	data, err := proto.Marshal(fileDescriptor)
	require.NoError(t, err)
	opts := proto.UnmarshalOptions{Resolver: (&protoresolve.Registry{}).AsTypeResolver()}
	err = opts.Unmarshal(data, fileDescriptor)
	require.NoError(t, err)

	msgDescriptor := protodesc.ToDescriptorProto((&testdata.Another{}).ProtoReflect().Descriptor())
	// same thing for this message descriptor
	data, err = proto.Marshal(msgDescriptor)
	require.NoError(t, err)
	err = opts.Unmarshal(data, msgDescriptor)
	require.NoError(t, err)

	// Now the above messages have unrecognized fields for custom options.
	require.True(t, hasUnrecognized(fileDescriptor.ProtoReflect()))
	require.True(t, hasUnrecognized(fileDescriptor.ProtoReflect()))
	require.False(t, proto.HasExtension(msgDescriptor.Options, testdata.E_Rept))

	// Unrecognized become recognized.
	protoresolve.ReparseUnrecognized(fileDescriptor, protoregistry.GlobalTypes)
	require.False(t, hasUnrecognized(fileDescriptor.ProtoReflect()))

	protoresolve.ReparseUnrecognized(msgDescriptor, protoregistry.GlobalTypes)
	require.False(t, hasUnrecognized(msgDescriptor.ProtoReflect()))
	require.True(t, proto.HasExtension(msgDescriptor.Options, testdata.E_Rept))
}

func hasUnrecognized(msg protoreflect.Message) bool {
	if len(msg.GetUnknown()) > 0 {
		return true
	}
	var foundUnrecognized bool
	msg.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		switch {
		case fd.IsList() && isMessageKind(fd.Kind()):
			l := val.List()
			for i, length := 0, l.Len(); i < length; i++ {
				if hasUnrecognized(l.Get(i).Message()) {
					foundUnrecognized = true
					return false
				}
			}
		case fd.IsMap() && isMessageKind(fd.MapValue().Kind()):
			val.Map().Range(func(_ protoreflect.MapKey, val protoreflect.Value) bool {
				if hasUnrecognized(val.Message()) {
					foundUnrecognized = true
					return false
				}
				return true
			})
			if foundUnrecognized {
				return false
			}
		case !fd.IsMap() && isMessageKind(fd.Kind()):
			if hasUnrecognized(val.Message()) {
				foundUnrecognized = true
				return false
			}
		}
		return true
	})
	return foundUnrecognized
}

func isMessageKind(k protoreflect.Kind) bool {
	return k == protoreflect.MessageKind || k == protoreflect.GroupKind
}
