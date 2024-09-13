package protomessage

import (
	"google.golang.org/protobuf/proto"

	"github.com/jhump/protoreflect/v2/internal/reparse"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

// ReparseUnrecognized is a helper function for re-parsing unknown fields of a message,
// resolving any extensions therein using the given resolver. This is particularly useful
// for unmarshalling FileDescriptorProto and FileDescriptorSet messages. With these messages,
// custom options may not be statically known by the unmarshalling program, but would be
// defined in the descriptor protos. So when initially unmarshalling, custom options would
// be left unrecognized. After unmarshalling, the resulting descriptor protos can be used
// to create a resolver (like using [protoresolve.FromFileDescriptorSet]). That resolver can
// in turn be supplied to this function, to re-parse the descriptor protos, thereby
// recognizing and interpreting custom options therein.
func ReparseUnrecognized(msg proto.Message, resolver protoresolve.SerializationResolver) bool {
	return reparse.ReparseUnrecognized(msg.ProtoReflect(), resolver)
}
