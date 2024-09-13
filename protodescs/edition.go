package protodescs

import (
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// GetEdition returns the edition for the given file. If it cannot be determined,
// descriptorpb.Edition_EDITION_UNKNOWN is returned.
//
// The given protos value is optional. If non-nil, it will be consulted, if
// necessary, to find the underlying file descriptor proto for the given fd, from
// which the edition can be queried.
func GetEdition(fd protoreflect.FileDescriptor, protos protoresolve.ProtoFileOracle) descriptorpb.Edition {
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
	if protos != nil {
		fileProto, err := protos.ProtoFromFileDescriptor(fd)
		if err == nil {
			return fileProto.GetEdition()
		}
	}
	return protodesc.ToFileDescriptorProto(fd).GetEdition()
}
