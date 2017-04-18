package dynamic

import (
	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
)

// MessageFactory can be used to create new empty message objects.
type MessageFactory struct {
	er  *ExtensionRegistry
	ktr *KnownTypeRegistry
}

// NewMessageFactoryWithExtensionRegistry creates a new message factory where any
// dynamic messages produced will use the given extension registry to recognize and
// parse extension fields.
func NewMessageFactoryWithExtensionRegistry(er *ExtensionRegistry) *MessageFactory {
	return NewMessageFactoryWithRegistries(er, nil)
}

// NewMessageFactoryWithKnownTypeRegistry creates a new message factory where the
// known types, per the given registry, will be returned as normal protobuf messages
// (e.g. generated structs, instead of dynamic messages).
func NewMessageFactoryWithKnownTypeRegistry(ktr *KnownTypeRegistry) *MessageFactory {
	return NewMessageFactoryWithRegistries(nil, ktr)
}

// NewMessageFactoryWithDefaults creates a new message factory where all "default" types
// (those for which protoc-generated code is statically linked into the Go program) are
// known types. Is any dynamic messages are produced, they will recognize and parse all
// "default" extension fields. This is the equivalent of:
//   NewMessageFactoryWithRegistries(
//       NewExtensionRegistryWithDefaults(),
//       NewKnownTypeRegistryWithDefaults())
func NewMessageFactoryWithDefaults() *MessageFactory {
	return NewMessageFactoryWithRegistries(NewExtensionRegistryWithDefaults(), NewKnownTypeRegistryWithDefaults())
}

// NewMessageFactoryWithRegistries creates a new message factory with the given extension
// and known type registries.
func NewMessageFactoryWithRegistries(er *ExtensionRegistry, ktr *KnownTypeRegistry) *MessageFactory {
	return &MessageFactory{
		er:  er,
		ktr: ktr,
	}
}

// NewMessage creates a new empty message that corresponds to the given descriptor.
// If the given descriptor describes a "known type" then that type is instantiated.
// Otherwise, an empty dynamic message is returned.
func (f *MessageFactory) NewMessage(md *desc.MessageDescriptor) proto.Message {
	if f == nil {
		return NewMessage(md)
	}
	if m := f.ktr.CreateIfKnown(md.GetFullyQualifiedName()); m != nil {
		return m
	}
	return newMessageWithMessageFactory(md, f)
}
