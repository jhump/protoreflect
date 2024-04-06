package protoresolve

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/internal/wrappers"
)

// GetEdition returns the edition for the given file. If it cannot be determined,
// descriptorpb.Edition_EDITION_UNKNOWN is returned.
func GetEdition(fd protoreflect.FileDescriptor) descriptorpb.Edition {
	switch fd.Syntax() {
	case protoreflect.Proto2:
		return descriptorpb.Edition_EDITION_PROTO2
	case protoreflect.Proto3:
		return descriptorpb.Edition_EDITION_PROTO3
	case protoreflect.Editions:
		break
	default:
		return descriptorpb.Edition_EDITION_UNKNOWN
	}
	type hasEdition interface{ Edition() int32 }
	if ed, ok := fd.(hasEdition); ok {
		return descriptorpb.Edition(ed.Edition())
	}
	return wrappers.ProtoFromFileDescriptor(fd).GetEdition()
}
