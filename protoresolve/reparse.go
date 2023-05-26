package protoresolve

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// ReparseUnrecognized is a helper function for re-parsing unknown fields of a message,
// resolving any extensions therein using the given resolver. This is particularly useful
// for unmarshalling FileDescriptorProto and FileDescriptorSet messages. With these messages,
// custom options may not be statically known by the unmarshalling program, but would be
// defined in the descriptor protos. So when initially unmarshalling, custom options would
// be left unrecognized. After unmarshalling, the resulting descriptor protos can be used
// to create a resolver (like using [protowrap.FromFileDescriptorSet]). That resolver can
// in turn be supplied to this function, to re-parse the descriptor protos, thereby
// recognizing and interpreting custom options therein.
func ReparseUnrecognized(msg proto.Message, resolver SerializationResolver) {
	reparseUnrecognized(msg.ProtoReflect(), resolver)
}

func reparseUnrecognized(msg protoreflect.Message, resolver SerializationResolver) {
	msg.Range(func(fld protoreflect.FieldDescriptor, val protoreflect.Value) bool {
		if fld.Kind() != protoreflect.MessageKind && fld.Kind() != protoreflect.GroupKind {
			return true
		}
		if fld.IsList() {
			l := val.List()
			for i := 0; i < l.Len(); i++ {
				reparseUnrecognized(l.Get(i).Message(), resolver)
			}
		} else if fld.IsMap() {
			mapVal := fld.MapValue()
			if mapVal.Kind() != protoreflect.MessageKind && mapVal.Kind() != protoreflect.GroupKind {
				return true
			}
			m := val.Map()
			m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
				reparseUnrecognized(v.Message(), resolver)
				return true
			})
		} else {
			reparseUnrecognized(val.Message(), resolver)
		}
		return true
	})

	unk := msg.GetUnknown()
	if len(unk) > 0 {
		other := msg.New().Interface()
		if err := (proto.UnmarshalOptions{Resolver: resolver}).Unmarshal(unk, other); err == nil {
			msg.SetUnknown(nil)
			proto.Merge(msg.Interface(), other)
		}
	}
}
