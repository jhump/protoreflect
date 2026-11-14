package remotereg

import (
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

// remoteSubResolver is an implementation of SerializationResolver that
// uses a Registry to resolve symbols. This is used when resolving
// fetched types, when the Registry's TypeFetcher cannot fetch a
// named type. It is also used to resolve extensions when processing the
// custom options of fetched types. Unlike Registry.AsTypeResolver,
// this resolver will *not* recursively resolve types via fetching remote
// types -- it only uses locally cached types and the registry's fallback
// resolver.
type remoteSubResolver Registry

var _ protoresolve.SerializationResolver = (*remoteSubResolver)(nil)

func (r *remoteSubResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	fb := r.Fallback
	if fb == nil {
		return protoregistry.GlobalTypes.FindExtensionByName(field)
	}
	type typeRes interface {
		AsTypeResolver() protoresolve.TypeResolver
	}
	if tr, ok := fb.(typeRes); ok {
		return tr.AsTypeResolver().FindExtensionByName(field)
	}
	d, err := fb.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	fld, ok := d.(protoreflect.FieldDescriptor)
	if !ok {
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindExtension, d, "")
	}
	if !fld.IsExtension() {
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindExtension, fld, "")
	}
	return protoresolve.ExtensionType(fld), nil
}

type typeResolvable interface {
	AsTypeResolver() protoresolve.TypeResolver
}

func (r *remoteSubResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	fb := r.Fallback
	if fb == nil {
		return protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
	}
	if tr, ok := fb.(typeResolvable); ok {
		return tr.AsTypeResolver().FindExtensionByNumber(message, field)
	}
	if pool, ok := fb.(protoresolve.DescriptorPool); ok {
		ext := protoresolve.FindExtensionByNumber(pool, message, field)
		if ext != nil {
			return protoresolve.ExtensionType(ext), nil
		}
	}
	return nil, protoresolve.ErrNotFound
}

func (r *remoteSubResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	reg := (*Registry)(r)
	return r.FindMessageByURL(reg.urlForType(message, message.Parent()))
}

func (r *remoteSubResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	url = ensureScheme(url)
	r.mu.RLock()
	d := r.typeCache[url]
	r.mu.RUnlock()
	if d == nil {
		fb := r.Fallback
		if fb == nil {
			fb = protoregistry.GlobalFiles
		}
		if tr, ok := fb.(typeResolvable); ok {
			return tr.AsTypeResolver().FindMessageByURL(url)
		}
		var err error
		d, err = fb.FindDescriptorByName(protoresolve.TypeNameFromURL(url))
		if err != nil {
			return nil, err
		}
	}
	md, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindMessage, d, url)
	}
	return dynamicpb.NewMessageType(md), nil
}
