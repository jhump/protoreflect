package remotereg

import (
	"context"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// RemoteTypeResolver is an implementation of TypeResolver that uses
// a Registry to resolve symbols.
//
// All message and enum types returned will be dynamic types, created
// using the [dynamicpb] package, built on the descriptors resolved by
// the backing Registry.
type RemoteTypeResolver Registry

var _ protoresolve.TypeResolver = (*RemoteTypeResolver)(nil)

// FindExtensionByName implements the SerializationResolver interface.
//
// This method relies on the underlying Registry's fallback resolver.
// If the registry's fallback resolver is unconfigured or nil, then
// protoregistry.GlobalTypes will be used to find the extension. Otherwise,
// if the fallback has a method named AsTypeResolver that returns a
// TypeResolver, it will be invoked and the resulting type resolver will be
// used to find the extension type. Failing all of the above, the fallback's
// FindDescriptorByName method will be used to find the extension. In this
// case the returned extension type may be a dynamic extension type, created
// using the [dynamicpb] package.
func (r *RemoteTypeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return (*remoteSubResolver)(r).FindExtensionByName(field)
}

// FindExtensionByNumber implements the SerializationResolver interface.
//
// This method relies on the underlying Registry's fallback resolver.
// If the registry's fallback resolver is unconfigured or nil, then
// protoregistry.GlobalTypes will be used to find the extension. Otherwise,
// if the fallback has a method named AsTypeResolver that returns a
// TypeResolver, it will be invoked and the resulting type resolver will be
// used to find the extension type. Failing all of the above, the extension
// can only be found if the fallback resolver implements DescriptorPool or
// ExtensionResolver and will otherwise return ErrNotFound.
func (r *RemoteTypeResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return (*remoteSubResolver)(r).FindExtensionByNumber(message, field)
}

// FindMessageByName implements the SerializationResolver interface. Since
// finding a message may incur fetching the definition via a TypeFetcher,
// it is recommended, where possible, to instead use FindMessageByNameContext.
// Calling this version will implicitly use [context.Background]().
func (r *RemoteTypeResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	return r.FindMessageByNameContext(context.Background(), name)
}

// FindMessageByNameContext finds a message by name, using the given context
// if necessary to fetch the message type via a TypeFetcher. This uses the
// underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindMessageByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.MessageType, error) {
	md, err := (*Registry)(r).FindMessageByNameContext(ctx, name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessageType(md), nil
}

// FindMessageByURL implements the SerializationResolver interface. Since
// finding a message may incur fetching the definition via a TypeFetcher,
// it is recommended, where possible, to instead use FindMessageByURLContext.
// Calling this version will implicitly use [context.Background]().
func (r *RemoteTypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return r.FindMessageByURLContext(context.Background(), url)
}

// FindMessageByURLContext finds a message by type URL, using the given
// context if necessary to fetch the message type via a TypeFetcher. This
// uses the underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindMessageByURLContext(ctx context.Context, url string) (protoreflect.MessageType, error) {
	md, err := (*Registry)(r).FindMessageByURLContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessageType(md), nil
}

// FindEnumByName implements the method of the same name on the TypeResolver
// interface. Since finding an enum may incur fetching the definition via a
// TypeFetcher, it is recommended, where possible, to instead use
// FindEnumByNameContext. Calling this version will implicitly use
// [context.Background]().
func (r *RemoteTypeResolver) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumType, error) {
	return r.FindEnumByNameContext(context.Background(), name)
}

// FindEnumByNameContext finds an enum by name, using the given context
// if necessary to fetch the message type via a TypeFetcher. This uses the
// underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindEnumByNameContext(ctx context.Context, name protoreflect.FullName) (protoreflect.EnumType, error) {
	ed, err := (*Registry)(r).FindEnumByNameContext(ctx, name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewEnumType(ed), nil
}

// FindEnumByURL has a signature that is consistent with that of
// FindMessageByURL and is present for symmetry. Since finding an
// enum may incur fetching the definition via a TypeFetcher, it is
// recommended, where possible, to instead use FindEnumByURLContext.
// Calling this version will implicitly use [context.Background]().
func (r *RemoteTypeResolver) FindEnumByURL(url string) (protoreflect.EnumType, error) {
	return r.FindEnumByURLContext(context.Background(), url)
}

// FindEnumByURLContext finds an enum by type URL, using the given context
// if necessary to fetch the message type via a TypeFetcher. This uses the
// underlying Registry's method of the same name.
func (r *RemoteTypeResolver) FindEnumByURLContext(ctx context.Context, url string) (protoreflect.EnumType, error) {
	ed, err := (*Registry)(r).FindEnumByURLContext(ctx, url)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewEnumType(ed), nil
}
