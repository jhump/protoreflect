package fielddefault

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"google.golang.org/protobuf/reflect/protoreflect"
)

// DefaultValue returns the string representation of the default value for
// the given field. If it has no default, this returns the empty string.
// The string representation is the same as stored in the default_value
// field of a google.protobuf.FieldDescriptorProto message.
func DefaultValue(fld protoreflect.FieldDescriptor) string {
	if !fld.HasDefault() || !fld.HasPresence() ||
		fld.Cardinality() != protoreflect.Optional || fld.Message() != nil {
		return ""
	}
	defVal := fld.Default()
	if !defVal.IsValid() {
		return ""
	}
	switch fld.Kind() {
	case protoreflect.StringKind:
		return defVal.String()
	case protoreflect.BytesKind:
		return encodeDefaultBytes(defVal.Bytes())
	case protoreflect.EnumKind:
		return string(fld.DefaultEnumValue().Name())
	case protoreflect.FloatKind:
		return FormatFloat(defVal.Float(), 32)
	case protoreflect.DoubleKind:
		return FormatFloat(defVal.Float(), 64)
	case protoreflect.BoolKind:
		return strconv.FormatBool(defVal.Bool())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(defVal.Int(), 10)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(defVal.Uint(), 10)
	default:
		// Shouldn't happen; above cases should be exhaustive...
		return fmt.Sprintf("%v", defVal.Interface())
	}
}

// FormatFloat returns the string representation of the given floating point
// value, as used in default values and in option values in proto source. The
// bitSize is 32 for float values and 64 for double values.
func FormatFloat(flt float64, bitSize int) string {
	switch {
	case math.IsInf(flt, 1):
		return "inf"
	case math.IsInf(flt, -1):
		return "-inf"
	case math.IsNaN(flt):
		return "nan"
	}
	return strconv.FormatFloat(flt, 'g', -1, bitSize)
}

func encodeDefaultBytes(data []byte) string {
	var buf bytes.Buffer
	// This uses the same algorithm as the protoc C++ code for escaping strings.
	// The protoc C++ code in turn uses the abseil C++ library's CEscape function:
	//  https://github.com/abseil/abseil-cpp/blob/934f613818ffcb26c942dff4a80be9a4031c662c/absl/strings/escaping.cc#L406
	for _, c := range data {
		switch c {
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		case '"':
			buf.WriteString("\\\"")
		case '\'':
			buf.WriteString("\\'")
		case '\\':
			buf.WriteString("\\\\")
		default:
			if c >= 0x20 && c < 0x7f {
				// simple printable characters
				buf.WriteByte(c)
			} else {
				// use octal escape for all other values
				buf.WriteRune('\\')
				buf.WriteByte('0' + ((c >> 6) & 0x7))
				buf.WriteByte('0' + ((c >> 3) & 0x7))
				buf.WriteByte('0' + (c & 0x7))
			}
		}
	}
	return buf.String()
}
