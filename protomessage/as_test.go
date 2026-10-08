package protomessage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
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

func TestAsWithNestedExtensions(t *testing.T) {
	t.Parallel()
	msgDesc := (&testprotos.AnotherTestMessage{}).ProtoReflect().Descriptor()
	msg := dynamicpb.NewMessage(msgDesc)
	msg.Set(testprotos.E_Xs.TypeDescriptor(), protoreflect.ValueOfString("top-level"))
	mapField := msg.Mutable(msgDesc.Fields().ByName("map_field4")).Map()
	nested := mapField.NewValue()
	nested.Message().Set(testprotos.E_Xi.TypeDescriptor(), protoreflect.ValueOfInt32(42))
	mapField.Set(protoreflect.ValueOfString("key").MapKey(), nested)

	converted, err := As[*testprotos.AnotherTestMessage](msg)
	require.NoError(t, err)
	assert.Equal(t, "top-level", proto.GetExtension(converted, testprotos.E_Xs))
	nestedConverted := converted.GetMapField4()["key"]
	require.NotNil(t, nestedConverted)
	assert.Empty(t, nestedConverted.ProtoReflect().GetUnknown(), "nested extension should be recognized")
	assert.Equal(t, int32(42), proto.GetExtension(nestedConverted, testprotos.E_Xi))
}
