package protoresolve

import (
	"errors"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
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
	return TypesFromResolver(c)
}
