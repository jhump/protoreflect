package desc

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// DescriptorWrapper wraps a protoreflect.Descriptor. All of the Descriptor
// implementations in this package implement this interface. This can be
// used to recover the underlying descriptor. Each descriptor type in this
// package also provides a strongly-typed form of this method, such as the
// following method for *FileDescriptor:
//    UnwrapFile() protoreflect.FileDescriptor
type DescriptorWrapper interface {
	Unwrap() protoreflect.Descriptor
}

func WrapDescriptor(d protoreflect.Descriptor) (Descriptor, error) {
	return wrapDescriptor(d, noopCache{})
}

func wrapDescriptor(d protoreflect.Descriptor, cache descriptorCache) (Descriptor, error) {
	switch d := d.(type) {
	case protoreflect.FileDescriptor:
		return wrapFile(d, cache)
	case protoreflect.MessageDescriptor:
		return wrapMessage(d, cache)
	case protoreflect.FieldDescriptor:
		return wrapField(d, cache)
	case protoreflect.OneofDescriptor:
		return wrapOneOf(d, cache)
	case protoreflect.EnumDescriptor:
		return wrapEnum(d, cache)
	case protoreflect.EnumValueDescriptor:
		return wrapEnumValue(d, cache)
	case protoreflect.ServiceDescriptor:
		return wrapService(d, cache)
	case protoreflect.MethodDescriptor:
		return wrapMethod(d, cache)
	default:
		return nil, fmt.Errorf("unknown descriptor type: %T", d)
	}
}

func WrapFile(d protoreflect.FileDescriptor) (*FileDescriptor, error) {
	return wrapFile(d, noopCache{})
}

func wrapFile(d protoreflect.FileDescriptor, cache descriptorCache) (*FileDescriptor, error) {
	fdp := protodesc.ToFileDescriptorProto(d)
	return convertFile(d, fdp, cache)
}

func WrapMessage(d protoreflect.MessageDescriptor) (*MessageDescriptor, error) {
	return wrapMessage(d, noopCache{})
}

func wrapMessage(d protoreflect.MessageDescriptor, cache descriptorCache) (*MessageDescriptor, error) {
	parent, err := wrapDescriptor(d.Parent(), cache)
	if err != nil {
		return nil, err
	}
	switch p := parent.(type) {
	case *FileDescriptor:
		return p.messages[d.Index()], nil
	case *MessageDescriptor:
		return p.nested[d.Index()], nil
	default:
		return nil, fmt.Errorf("message has unexpected parent type: %T", parent)
	}
}

func WrapField(d protoreflect.FieldDescriptor) (*FieldDescriptor, error) {
	return wrapField(d, noopCache{})
}

func wrapField(d protoreflect.FieldDescriptor, cache descriptorCache) (*FieldDescriptor, error) {
	parent, err := wrapDescriptor(d.Parent(), cache)
	if err != nil {
		return nil, err
	}
	switch p := parent.(type) {
	case *FileDescriptor:
		return p.extensions[d.Index()], nil
	case *MessageDescriptor:
		if d.IsExtension() {
			return p.extensions[d.Index()], nil
		}
		return p.fields[d.Index()], nil
	default:
		return nil, fmt.Errorf("field has unexpected parent type: %T", parent)
	}
}

func WrapOneOf(d protoreflect.OneofDescriptor) (*OneOfDescriptor, error) {
	return wrapOneOf(d, noopCache{})
}

func wrapOneOf(d protoreflect.OneofDescriptor, cache descriptorCache) (*OneOfDescriptor, error) {
	parent, err := wrapDescriptor(d.Parent(), cache)
	if err != nil {
		return nil, err
	}
	if p, ok := parent.(*MessageDescriptor); ok {
		return p.oneOfs[d.Index()], nil
	}
	return nil, fmt.Errorf("oneof has unexpected parent type: %T", parent)
}

func WrapEnum(d protoreflect.EnumDescriptor) (*EnumDescriptor, error) {
	return wrapEnum(d, noopCache{})
}

func wrapEnum(d protoreflect.EnumDescriptor, cache descriptorCache) (*EnumDescriptor, error) {
	parent, err := wrapDescriptor(d.Parent(), cache)
	if err != nil {
		return nil, err
	}
	switch p := parent.(type) {
	case *FileDescriptor:
		return p.enums[d.Index()], nil
	case *MessageDescriptor:
		return p.enums[d.Index()], nil
	default:
		return nil, fmt.Errorf("enum has unexpected parent type: %T", parent)
	}
}

func WrapEnumValue(d protoreflect.EnumValueDescriptor) (*EnumValueDescriptor, error) {
	return wrapEnumValue(d, noopCache{})
}

func wrapEnumValue(d protoreflect.EnumValueDescriptor, cache descriptorCache) (*EnumValueDescriptor, error) {
	parent, err := wrapDescriptor(d.Parent(), cache)
	if err != nil {
		return nil, err
	}
	if p, ok := parent.(*EnumDescriptor); ok {
		return p.values[d.Index()], nil
	}
	return nil, fmt.Errorf("enum value has unexpected parent type: %T", parent)
}

func WrapService(d protoreflect.ServiceDescriptor) (*ServiceDescriptor, error) {
	return wrapService(d, noopCache{})
}

func wrapService(d protoreflect.ServiceDescriptor, cache descriptorCache) (*ServiceDescriptor, error) {
	parent, err := wrapDescriptor(d.Parent(), cache)
	if err != nil {
		return nil, err
	}
	if p, ok := parent.(*FileDescriptor); ok {
		return p.services[d.Index()], nil
	}
	return nil, fmt.Errorf("service has unexpected parent type: %T", parent)
}

func WrapMethod(d protoreflect.MethodDescriptor) (*MethodDescriptor, error) {
	return wrapMethod(d, noopCache{})
}

func wrapMethod(d protoreflect.MethodDescriptor, cache descriptorCache) (*MethodDescriptor, error) {
	parent, err := wrapDescriptor(d.Parent(), cache)
	if err != nil {
		return nil, err
	}
	if p, ok := parent.(*ServiceDescriptor); ok {
		return p.methods[d.Index()], nil
	}
	return nil, fmt.Errorf("method has unexpected parent type: %T", parent)
}
