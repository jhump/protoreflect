package protomessage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal"
	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestReparse(t *testing.T) {
	fileDescriptor := protodesc.ToFileDescriptorProto(testprotos.File_desc_test_complex_proto)
	// serialize to bytes and back, but use empty resolver when
	// de-serializing so that custom options are unrecognized
	data, err := proto.Marshal(fileDescriptor)
	require.NoError(t, err)
	opts := proto.UnmarshalOptions{Resolver: (&protoresolve.Registry{}).AsTypeResolver()}
	err = opts.Unmarshal(data, fileDescriptor)
	require.NoError(t, err)

	msgDescriptor := protodesc.ToDescriptorProto((&testprotos.Another{}).ProtoReflect().Descriptor())
	// same thing for this message descriptor
	data, err = proto.Marshal(msgDescriptor)
	require.NoError(t, err)
	err = opts.Unmarshal(data, msgDescriptor)
	require.NoError(t, err)

	// Now the above messages have unrecognized fields for custom options.
	require.True(t, hasUnrecognized(fileDescriptor.ProtoReflect()))
	require.True(t, hasUnrecognized(fileDescriptor.ProtoReflect()))
	require.False(t, proto.HasExtension(msgDescriptor.Options, testprotos.E_Rept))

	// Unrecognized become recognized.
	require.True(t, ReparseUnrecognized(fileDescriptor, protoregistry.GlobalTypes))
	require.False(t, hasUnrecognized(fileDescriptor.ProtoReflect()))
	require.False(t, ReparseUnrecognized(fileDescriptor, protoregistry.GlobalTypes)) // no-op this time

	require.True(t, ReparseUnrecognized(msgDescriptor, protoregistry.GlobalTypes))
	require.False(t, hasUnrecognized(msgDescriptor.ProtoReflect()))
	require.True(t, proto.HasExtension(msgDescriptor.Options, testprotos.E_Rept))
}

func hasUnrecognized(msg protoreflect.Message) bool {
	if len(msg.GetUnknown()) > 0 {
		return true
	}
	var foundUnrecognized bool
	msg.Range(func(fd protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		switch {
		case fd.IsList() && internal.IsMessageKind(fd.Kind()):
			l := val.List()
			for i, length := 0, l.Len(); i < length; i++ {
				if hasUnrecognized(l.Get(i).Message()) {
					foundUnrecognized = true
					return false
				}
			}
		case fd.IsMap() && internal.IsMessageKind(fd.MapValue().Kind()):
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
		case !fd.IsMap() && internal.IsMessageKind(fd.Kind()):
			if hasUnrecognized(val.Message()) {
				foundUnrecognized = true
				return false
			}
		}
		return true
	})
	return foundUnrecognized
}

func TestReparseReportsChanges(t *testing.T) {
	t.Parallel()
	// Unknown bytes for a known extension and for an unknown field.
	known := &descriptorpb.MessageOptions{}
	proto.SetExtension(known, testprotos.E_Mfubar, true)
	data, err := proto.Marshal(known)
	require.NoError(t, err)
	data = protowire.AppendTag(data, 50000, protowire.VarintType)
	data = protowire.AppendVarint(data, 1)

	// Nothing recognized, so nothing changes.
	var opts descriptorpb.MessageOptions
	require.NoError(t, proto.UnmarshalOptions{Resolver: &protoregistry.Types{}}.Unmarshal(data, &opts))
	assert.False(t, ReparseUnrecognized(&opts, &protoregistry.Types{}))
	assert.Equal(t, []byte(data), []byte(opts.ProtoReflect().GetUnknown()))

	// The known extension is recognized; the unknown field remains unknown.
	assert.True(t, ReparseUnrecognized(&opts, protoregistry.GlobalTypes))
	assert.Equal(t, true, proto.GetExtension(&opts, testprotos.E_Mfubar))
	assert.NotEmpty(t, opts.ProtoReflect().GetUnknown())
}
