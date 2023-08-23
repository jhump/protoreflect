package protoresolve

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestAs(t *testing.T) {
	var msg proto.Message
	// msg needs no conversion
	msg = &anypb.Any{TypeUrl: "abc/def.xyz"}
	asAny, err := As[*anypb.Any](msg)
	require.NoError(t, err)
	require.Same(t, msg, asAny)

	// msg needs conversion from dynamic message
	msg = dynamicpb.NewMessage((&anypb.Any{}).ProtoReflect().Descriptor())
	fields := msg.ProtoReflect().Descriptor().Fields()
	msg.ProtoReflect().Set(fields.ByName("type_url"), protoreflect.ValueOfString("abc/def.xyz"))
	asAny, err = As[*anypb.Any](msg)
	require.NoError(t, err)
	// not the same instance, but equivalent data
	require.NotSame(t, msg, asAny)
	require.True(t, proto.Equal(msg, asAny))

	// msg cannot be converted: wrong type
	_, err = As[*wrapperspb.StringValue](msg)
	require.ErrorContains(t, err, `cannot return type "google.protobuf.StringValue": given message is "google.protobuf.Any"`)
}
