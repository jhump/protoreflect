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
	if len(res) == 0 {
		return combined(res)
	}
	allPools := true
	for _, r := range res {
		_, isPool := r.(interface {
			Resolver
			AsTypePool() TypePool
		})
		if !isPool {
			allPools = false
			break
		}
	}
	if !allPools {
		return combined(res)
	}

	pools := make([]TypePool, len(res))
	for i, r := range res {
		pools[i] = r.(interface {
			Resolver
			AsTypePool() TypePool
		}).AsTypePool()
	}
	return &combinedWithPool{combined: combined(res), pool: combinedPool(pools)}
}

// CombinePools is just like Combine, except that the returned value provides am
// AsTypePool() method that returns a TypePool that iterates through the TypePools
// of the given resolvers to find and enumerate elements. The AsTypeResolver()
// method of the returned value will also implement the broader TypePool interface.
func CombinePools(res ...interface {
	Resolver
	AsTypePool() TypePool
}) interface {
	Resolver
	AsTypePool() TypePool
} {
	pools := make([]TypePool, len(res))
	for i, r := range res {
		pools[i] = r.AsTypePool()
	}
	baseRes := make([]Resolver, len(res))
	for i, r := range res {
		baseRes[i] = r
	}
	return &combinedWithPool{combined: combined(baseRes), pool: combinedPool(pools)}
}

type combined []Resolver

func (c combined) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	return findFirst(c, func(res Resolver) (protoreflect.FileDescriptor, error) {
		return res.FindFileByPath(path)
	})
}

func (c combined) NumFiles() int {
	if len(c) == 0 {
		return 0
	}
	return c[0].NumFiles()
}

func (c combined) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	rangeDistinct(c, Resolver.RangeFiles, protoreflect.FileDescriptor.Path, f)
}

func (c combined) NumFilesByPackage(name protoreflect.FullName) int {
	if len(c) == 0 {
		return 0
	}
	return c[0].NumFilesByPackage(name)
}

func (c combined) RangeFilesByPackage(name protoreflect.FullName, f func(protoreflect.FileDescriptor) bool) {
	rangeDistinct(c, func(res Resolver, fn func(protoreflect.FileDescriptor) bool) {
		res.RangeFilesByPackage(name, fn)
	}, protoreflect.FileDescriptor.Path, f)
}

func (c combined) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	return findFirst(c, func(res Resolver) (protoreflect.Descriptor, error) {
		return res.FindDescriptorByName(name)
	})
}

func (c combined) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	return findFirst(c, func(res Resolver) (protoreflect.MessageDescriptor, error) {
		return res.FindMessageByName(name)
	})
}

func (c combined) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionDescriptor, error) {
	return findFirst(c, func(res Resolver) (protoreflect.ExtensionDescriptor, error) {
		return res.FindExtensionByName(name)
	})
}

func (c combined) FindExtensionByNumber(message protoreflect.FullName, number protoreflect.FieldNumber) (protoreflect.ExtensionDescriptor, error) {
	return findFirst(c, func(res Resolver) (protoreflect.ExtensionDescriptor, error) {
		return res.FindExtensionByNumber(message, number)
	})
}

func (c combined) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionDescriptor) bool) {
	rangeDistinct(c, func(res Resolver, fn func(protoreflect.ExtensionDescriptor) bool) {
		res.RangeExtensionsByMessage(message, fn)
	}, protoreflect.ExtensionDescriptor.Number, fn)
}

func (c combined) FindMessageByURL(url string) (protoreflect.MessageDescriptor, error) {
	return findFirst(c, func(res Resolver) (protoreflect.MessageDescriptor, error) {
		return res.FindMessageByURL(url)
	})
}

func (c combined) AsTypeResolver() TypeResolver {
	return TypesFromResolver(c)
}

type combinedPool []TypePool

func (c combinedPool) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return findFirst(c, func(res TypePool) (protoreflect.ExtensionType, error) {
		return res.FindExtensionByName(name)
	})
}

func (c combinedPool) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return findFirst(c, func(res TypePool) (protoreflect.ExtensionType, error) {
		return res.FindExtensionByNumber(message, field)
	})
}

func (c combinedPool) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageType, error) {
	return findFirst(c, func(res TypePool) (protoreflect.MessageType, error) {
		return res.FindMessageByName(name)
	})
}

func (c combinedPool) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return findFirst(c, func(res TypePool) (protoreflect.MessageType, error) {
		return res.FindMessageByURL(url)
	})
}

func (c combinedPool) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumType, error) {
	return findFirst(c, func(res TypePool) (protoreflect.EnumType, error) {
		return res.FindEnumByName(name)
	})
}

func (c combinedPool) RangeMessages(fn func(protoreflect.MessageType) bool) {
	rangeDistinct(c, TypePool.RangeMessages, func(msg protoreflect.MessageType) protoreflect.FullName {
		return msg.Descriptor().FullName()
	}, fn)
}

func (c combinedPool) RangeEnums(fn func(protoreflect.EnumType) bool) {
	rangeDistinct(c, TypePool.RangeEnums, func(en protoreflect.EnumType) protoreflect.FullName {
		return en.Descriptor().FullName()
	}, fn)
}

func (c combinedPool) RangeExtensions(fn func(protoreflect.ExtensionType) bool) {
	rangeDistinct(c, TypePool.RangeExtensions, func(ext protoreflect.ExtensionType) protoreflect.FullName {
		return ext.TypeDescriptor().FullName()
	}, fn)
}

func (c combinedPool) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionType) bool) {
	rangeDistinct(c, func(res TypePool, fn func(protoreflect.ExtensionType) bool) {
		res.RangeExtensionsByMessage(message, fn)
	}, func(ext protoreflect.ExtensionType) protoreflect.FieldNumber {
		return ext.TypeDescriptor().Number()
	}, fn)
}

type combinedWithPool struct {
	combined
	pool TypePool
}

func (c *combinedWithPool) AsTypeResolver() TypeResolver {
	return c.pool
}

func (c *combinedWithPool) AsTypePool() TypePool {
	return c.pool
}

// rangeDistinct enumerates elements from all the given resolvers, in order,
// using rangeFn to enumerate the elements of each one. Each element is passed
// to fn, except for elements with the same key as one already passed, which
// are skipped. It stops as soon as fn returns false.
func rangeDistinct[R, T any, K comparable](
	resolvers []R,
	rangeFn func(R, func(T) bool),
	key func(T) K,
	fn func(T) bool,
) {
	seen := map[K]struct{}{}
	for _, res := range resolvers {
		keepGoing := true
		rangeFn(res, func(elem T) bool {
			elemKey := key(elem)
			if _, ok := seen[elemKey]; ok {
				return true
			}
			seen[elemKey] = struct{}{}
			keepGoing = fn(elem)
			return keepGoing
		})
		if !keepGoing {
			return
		}
	}
}

// findFirst calls find with each of the given resolvers, in order, and returns
// the first result that is not a not-found error.
func findFirst[R, T any](resolvers []R, find func(R) (T, error)) (T, error) {
	for _, res := range resolvers {
		result, err := find(res)
		if errors.Is(err, protoregistry.NotFound) {
			continue
		}
		return result, err
	}
	var zero T
	return zero, protoregistry.NotFound
}
