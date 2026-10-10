package protomessage

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// PointerMessage is a pointer type that implements [proto.Message].
type PointerMessage[T any] interface {
	*T
	proto.Message
}

// As returns the given message as type M. If the given message is type M,
// it is returned as-is. Otherwise (like if the given message is a dynamic
// message), it will be marshalled to bytes and then unmarshalled into a
// value of type M. If M and msg do not share the same message type (e.g.
// same fully qualified message name), an error is returned.
func As[M PointerMessage[T], T any](msg proto.Message) (M, error) {
	dest, ok := msg.(M)
	if ok {
		return dest, nil
	}
	if msg.ProtoReflect().Descriptor().FullName() != dest.ProtoReflect().Descriptor().FullName() {
		return nil, fmt.Errorf("cannot return type %q: given message is %q", dest.ProtoReflect().Descriptor().FullName(), msg.ProtoReflect().Descriptor().FullName())
	}
	// Collect the extensions present anywhere in the message, so they can be
	// recognized when unmarshalling.
	var exts *protoregistry.Types
	var err error
	Walk(msg.ProtoReflect(), func(_ []any, val protoreflect.Message) bool {
		val.Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			if !fd.IsExtension() {
				return true
			}
			if exts == nil {
				exts = &protoregistry.Types{}
			}
			if _, findErr := exts.FindExtensionByName(fd.FullName()); findErr == nil {
				// already registered
				return true
			}
			err = exts.RegisterExtension(protoresolve.ExtensionType(fd))
			return err == nil
		})
		return err == nil
	})
	if err != nil {
		return nil, err
	}
	dest = new(T)
	var opts proto.UnmarshalOptions
	if exts != nil {
		opts.Resolver = exts
	}
	if data, err := proto.Marshal(msg); err != nil {
		return nil, err
	} else if err = opts.Unmarshal(data, dest); err != nil {
		return nil, err
	}
	return dest, nil
}
