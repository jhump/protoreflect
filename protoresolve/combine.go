package protoresolve

import (
	"errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Combine returns a resolver that iterates through the given resolvers to find elements.
// The first resolver given is the first one checked, so will always be the preferred resolver.
// When that returns a protoregistry.NotFound error, the next resolver will be checked, and so on.
//
// The NumFiles and NumFilesByPackage methods only return the number of files reported by the first
// resolver. (Computing an accurate number of files across all resolvers could be an expensive
// operation.) However, RangeFiles and RangeFilesByPackage do return files across all resolvers.
// They emit files for the first resolver first. If any subsequent resolver contains duplicates,
// they are suppressed such that the callback will only ever be invoked once for a given file path.
func Combine(res ...Resolver) Resolver {
	return combined(res)
}

type combined []Resolver

func (c combined) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	for _, res := range c {
		file, err := res.FindFileByPath(path)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return file, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) NumFiles() int {
	if len(c) == 0 {
		return 0
	}
	return c[0].NumFiles()
}

func (c combined) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	observed := map[string]struct{}{}
	for _, res := range c {
		res.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			if _, ok := observed[fd.Path()]; ok {
				return true
			}
			observed[fd.Path()] = struct{}{}
			return f(fd)
		})
	}
}

func (c combined) NumFilesByPackage(name protoreflect.FullName) int {
	if len(c) == 0 {
		return 0
	}
	return c[0].NumFilesByPackage(name)
}

func (c combined) RangeFilesByPackage(name protoreflect.FullName, f func(protoreflect.FileDescriptor) bool) {
	observed := map[string]struct{}{}
	for _, res := range c {
		res.RangeFilesByPackage(name, func(fd protoreflect.FileDescriptor) bool {
			if _, ok := observed[fd.Path()]; ok {
				return true
			}
			observed[fd.Path()] = struct{}{}
			return f(fd)
		})
	}
}

func (c combined) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	for _, res := range c {
		d, err := res.FindDescriptorByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return d, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	for _, res := range c {
		msg, err := res.FindMessageByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return msg, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindFieldByName(name protoreflect.FullName) (protoreflect.FieldDescriptor, error) {
	for _, res := range c {
		fld, err := res.FindFieldByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return fld, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionDescriptor, error) {
	for _, res := range c {
		ext, err := res.FindExtensionByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return ext, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindOneofByName(name protoreflect.FullName) (protoreflect.OneofDescriptor, error) {
	for _, res := range c {
		ood, err := res.FindOneofByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return ood, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	for _, res := range c {
		en, err := res.FindEnumByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return en, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindEnumValueByName(name protoreflect.FullName) (protoreflect.EnumValueDescriptor, error) {
	for _, res := range c {
		enVal, err := res.FindEnumValueByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return enVal, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	for _, res := range c {
		svc, err := res.FindServiceByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return svc, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	for _, res := range c {
		mtd, err := res.FindMethodByName(name)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return mtd, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) FindExtensionByNumber(message protoreflect.FullName, number protoreflect.FieldNumber) (protoreflect.ExtensionDescriptor, error) {
	for _, res := range c {
		ext, err := res.FindExtensionByNumber(message, number)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return ext, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionDescriptor) bool) {
	seen := map[protoreflect.FieldNumber]struct{}{}
	for _, res := range c {
		var keepGoing bool
		res.RangeExtensionsByMessage(message, func(ext protoreflect.ExtensionDescriptor) bool {
			if _, ok := seen[ext.Number()]; ok {
				return true
			}
			keepGoing = fn(ext)
			return keepGoing
		})
		if !keepGoing {
			return
		}
	}
}

func (c combined) FindMessageByURL(url string) (protoreflect.MessageDescriptor, error) {
	for _, res := range c {
		msg, err := res.FindMessageByURL(url)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return msg, err
	}
	return nil, protoregistry.NotFound
}

func (c combined) AsTypeResolver() TypeResolver {
	return dynTypeResolver{res: c}
}

type dynTypeResolver struct {
	res Resolver
}

func (d dynTypeResolver) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
	ext, err := d.res.FindExtensionByName(name)
	if err != nil {
		return nil, err
	}
	return ExtensionType(ext), nil
}

func (d dynTypeResolver) FindExtensionByNumber(message protoreflect.FullName, number protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	ext, err := d.res.FindExtensionByNumber(message, number)
	if err != nil {
		return nil, err
	}
	return ExtensionType(ext), nil
}

func (d dynTypeResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	msg, err := d.res.FindMessageByName(name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewMessageType(msg), nil
}

func (d dynTypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return d.FindMessageByName(TypeNameFromURL(url))
}

func (d dynTypeResolver) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumType, error) {
	en, err := d.res.FindEnumByName(name)
	if err != nil {
		return nil, err
	}
	return dynamicpb.NewEnumType(en), nil
}
