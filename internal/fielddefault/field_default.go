package fielddefault

import (
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// DefaultValue returns the string representation of the default value for
// the given field. If it has no default, this returns the empty string.
// The string representation is the same as stored in the default_value
// field of a google.protobuf.FieldDescriptorProto message.
func DefaultValue(fld protoreflect.FieldDescriptor) string {
	// TODO: ideally we could maybe use a protoresolve.ProtoOracle to more
	// efficiently recover the original field descriptor proto. Or we could
	// reproduce more logic here to convert from the FieldDescriptor.DefaultValue
	// back to the original default value representation. But fields don't have
	// children, so this call shouldn't be very expensive.
	return protodesc.ToFieldDescriptorProto(fld).GetDefaultValue()
}
