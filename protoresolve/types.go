package protoresolve

import (
	"bytes"
	"fmt"
	"math/bits"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TypePool is a type resolver that allows for iteration over all known types.
type TypePool interface {
	TypeResolver
	RangeMessages(fn func(protoreflect.MessageType) bool)
	RangeEnums(fn func(protoreflect.EnumType) bool)
	RangeExtensions(fn func(protoreflect.ExtensionType) bool)
	RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionType) bool)
}

var _ TypePool = (*protoregistry.Types)(nil)

// TypeRegistry is a type resolver that allows the caller to add elements to
// the set of types it can resolve.
type TypeRegistry interface {
	TypePool
	RegisterMessage(protoreflect.MessageType) error
	RegisterEnum(protoreflect.EnumType) error
	RegisterExtension(protoreflect.ExtensionType) error
}

var _ TypeRegistry = (*protoregistry.Types)(nil)

// ExtensionType returns a [protoreflect.ExtensionType] for the given descriptor.
// If the given descriptor implements [protoreflect.ExtensionTypeDescriptor], then
// the corresponding type is returned. Otherwise, a dynamic extension type is
// returned (created using "google.golang.org/protobuf/types/dynamicpb").
func ExtensionType(ext protoreflect.ExtensionDescriptor) protoreflect.ExtensionType {
	if xtd, ok := ext.(protoreflect.ExtensionTypeDescriptor); ok {
		return xtd.Type()
	}
	return dynamicpb.NewExtensionType(ext)
}

// TypeNameFromURL extracts the fully-qualified type name from the given URL.
// The URL is one that could be used with a google.protobuf.Any message. The
// last path component is the fully-qualified name.
func TypeNameFromURL(url string) protoreflect.FullName {
	pos := strings.LastIndexByte(url, '/')
	return protoreflect.FullName(url[pos+1:])
}

// TypeKind represents a category of types that can be registered in a TypeRegistry.
// The value for a particular kind is a single bit, so a TypeKind value can also
// represent multiple kinds, by setting multiple bits (by combining values via
// bitwise-OR).
type TypeKind int

// The various supported TypeKind values.
const (
	TypeKindMessage = TypeKind(1 << iota)
	TypeKindEnum
	TypeKindExtension

	// TypeKindsAll is a bitmask that represents all types.
	TypeKindsAll = TypeKindMessage | TypeKindEnum | TypeKindExtension
	// TypeKindsSerialization includes the kinds of types needed for serialization
	// and de-serialization: messages (for interpreting google.protobuf.Any messages)
	// and extensions. These are the same types as supported in a SerializationResolver.
	TypeKindsSerialization = TypeKindMessage | TypeKindExtension
)

func (k TypeKind) String() string {
	switch k {
	case TypeKindMessage:
		return "message"
	case TypeKindEnum:
		return "enum"
	case TypeKindExtension:
		return "extension"
	case 0:
		return "<none>"
	default:
		i := uint(k)
		if bits.OnesCount(i) == 1 {
			return fmt.Sprintf("unknown kind (%d)", k)
		}

		var buf bytes.Buffer
		l := bits.UintSize
		for i != 0 {
			if buf.Len() > 0 {
				buf.WriteByte(',')
			}
			z := bits.LeadingZeros(i)
			if z == l {
				break
			}
			shr := l - z - 1
			elem := TypeKind(1 << shr)
			buf.WriteString(elem.String())
		}
		return buf.String()
	}
}

// RegisterTypesInFile registers all the types (with kinds that match kindMask) with
// the given registry. Only the types directly in file are registered. This will result
// in an error if any of the types in the given file are already registered as belonging
// to a different file.
//
// All types will be dynamic types, created with the "google.golang.org/protobuf/types/dynamicpb"
// package. The only exception is for extension descriptors that also implement
// [protoreflect.ExtensionTypeDescriptor], in which case the corresponding extension type is used.
func RegisterTypesInFile(file protoreflect.FileDescriptor, reg TypeRegistry, kindMask TypeKind) error {
	return registerTypes(file, reg, kindMask)
}

// RegisterTypesInFileRecursive registers all the types (with kinds that match kindMask)
// with the given registry, for the given file and all of its transitive dependencies (i.e.
// its imports, and their imports, etc.). This will result in an error if any of the types in
// the given file (and its dependencies) are already registered as belonging to a different file.
//
// All types will be dynamic types, created with the "google.golang.org/protobuf/types/dynamicpb"
// package. The only exception is for extension descriptors that also implement
// [protoreflect.ExtensionTypeDescriptor], in which case the corresponding extension type is used.
func RegisterTypesInFileRecursive(file protoreflect.FileDescriptor, reg TypeRegistry, kindMask TypeKind) error {
	pathsSeen := map[string]struct{}{}
	return registerTypesInFileRecursive(file, reg, kindMask, pathsSeen)
}

// RegisterTypesInFilesRecursive registers all the types (with kinds that match kindMask)
// with the given registry, for all files in the given pool and their dependencies. This is
// essentially shorthand for this:
//
//		var err error
//		files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
//	 		err = protoresolve.RegisterTypesInFileRecursive(file, reg, kindMask)
//	 		return err == nil
//		})
//		return err
//
// However, the actual implementation is a little more efficient for cases where some files
// are imported by many other files.
//
// All types will be dynamic types, created with the "google.golang.org/protobuf/types/dynamicpb"
// package. The only exception is for extension descriptors that also implement
// [protoreflect.ExtensionTypeDescriptor], in which case the corresponding extension type is used.
func RegisterTypesInFilesRecursive(files FilePool, reg TypeRegistry, kindMask TypeKind) error {
	pathsSeen := map[string]struct{}{}
	var err error
	files.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		err = registerTypesInFileRecursive(file, reg, kindMask, pathsSeen)
		return err == nil
	})
	return err
}

func registerTypesInFileRecursive(file protoreflect.FileDescriptor, reg TypeRegistry, kindMask TypeKind, pathsSeen map[string]struct{}) error {
	if _, ok := pathsSeen[file.Path()]; ok {
		// already processed
		return nil
	}
	pathsSeen[file.Path()] = struct{}{}
	imports := file.Imports()
	for i, length := 0, imports.Len(); i < length; i++ {
		imp := imports.Get(i)
		if err := registerTypesInFileRecursive(imp.FileDescriptor, reg, kindMask, pathsSeen); err != nil {
			return err
		}
	}
	return registerTypes(file, reg, kindMask)
}

// TypeContainer is a descriptor that contains types. Both [protoreflect.FileDescriptor] and
// [protoreflect.MessageDescriptor] can contain types so both satisfy this interface.
type TypeContainer interface {
	Messages() protoreflect.MessageDescriptors
	Enums() protoreflect.EnumDescriptors
	Extensions() protoreflect.ExtensionDescriptors
}

var _ TypeContainer = (protoreflect.FileDescriptor)(nil)
var _ TypeContainer = (protoreflect.MessageDescriptor)(nil)

func registerTypes(container TypeContainer, reg TypeRegistry, kindMask TypeKind) error {
	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		msg := msgs.Get(i)
		if kindMask&TypeKindMessage != 0 {
			var skip bool
			if existing := findType(reg, msg.FullName()); existing != nil {
				if existing.ParentFile().Path() != msg.ParentFile().Path() {
					return fmt.Errorf("type %s is defined in both %q and %q", msg.FullName(), existing.ParentFile().Path(), msg.ParentFile().Path())
				}
				skip = true
			}
			if !skip {
				if err := reg.RegisterMessage(dynamicpb.NewMessageType(msg)); err != nil {
					return err
				}
			}
		}
		// register nested types
		if err := registerTypes(msg, reg, kindMask); err != nil {
			return err
		}
	}

	if kindMask&TypeKindEnum != 0 {
		enums := container.Enums()
		for i, length := 0, enums.Len(); i < length; i++ {
			enum := enums.Get(i)
			var skip bool
			if existing := findType(reg, enum.FullName()); existing != nil {
				if existing.ParentFile().Path() != enum.ParentFile().Path() {
					return fmt.Errorf("type %s is defined in both %q and %q", enum.FullName(), existing.ParentFile().Path(), enum.ParentFile().Path())
				}
				skip = true
			}
			if !skip {
				if err := reg.RegisterEnum(dynamicpb.NewEnumType(enum)); err != nil {
					return err
				}
			}
		}
	}

	if kindMask&TypeKindExtension != 0 {
		exts := container.Extensions()
		for i, length := 0, exts.Len(); i < length; i++ {
			ext := exts.Get(i)
			var skip bool
			if existing := findType(reg, ext.FullName()); existing != nil {
				if existing.ParentFile().Path() != ext.ParentFile().Path() {
					return fmt.Errorf("type %s is defined in both %q and %q", ext.FullName(), existing.ParentFile().Path(), ext.ParentFile().Path())
				}
				skip = true
			}
			if !skip {
				// also check extendee+tag
				existing, err := reg.FindExtensionByNumber(ext.ContainingMessage().FullName(), ext.Number())
				if err == nil {
					if existing.TypeDescriptor().ParentFile().Path() != ext.ParentFile().Path() {
						return fmt.Errorf("extension number %d for %s is defined in both %q and %q", ext.Number(), ext.ContainingMessage().FullName(), existing.TypeDescriptor().ParentFile().Path(), ext.ParentFile().Path())
					}
					skip = true
				}
			}
			if !skip {
				if err := reg.RegisterExtension(ExtensionType(ext)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func findType(res TypeResolver, name protoreflect.FullName) protoreflect.Descriptor {
	msg, err := res.FindMessageByName(name)
	if err == nil {
		return msg.Descriptor()
	}
	en, err := res.FindEnumByName(name)
	if err == nil {
		return en.Descriptor()
	}
	ext, err := res.FindExtensionByName(name)
	if err == nil {
		return ext.TypeDescriptor()
	}
	return nil
}

// TypesFromResolver adapts a resolver that returns descriptors into a resolver
// that returns types. This can be used by implementations of Resolver to
// implement the [Resolver.AsTypeResolver] method.
//
// It returns all dynamic types except for extensions, in which case, if an
// extension implements [protoreflect.ExtensionTypeDescriptor], it will return
// its associated [protoreflect.ExtensionType]. (Otherwise it returns a dynamic
// extension.)
func TypesFromResolver(resolver interface {
	DescriptorResolver
	ExtensionResolver
}) TypeResolver {
	return &typesFromResolver{resolver: resolver}
}

// TypesFromDescriptorPool adapts a descriptor pool into a pool that returns
// types. This can be used by implementations of Resolver to implement the
// [Resolver.AsTypeResolver] method.
//
// If the given resolver implements ExtensionResolver, then the returned type
// pool provides an efficient implementation for the
// [ExtensionTypeResolver.FindExtensionByNumber] method. Otherwise, it will
// use an inefficient implementation that searches through all files for the
// requested extension.
func TypesFromDescriptorPool(pool DescriptorPool) TypePool {
	return &typesFromDescriptorPool{pool: pool}
}

type typesFromResolver struct {
	// The underlying resolver. It must be able to provide descriptors by name
	// and also be able to provide extension descriptors by extendee+tag number.
	resolver interface {
		DescriptorResolver
		ExtensionResolver
	}
}

func (t *typesFromResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	d, err := t.resolver.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	ext, ok := d.(protoreflect.ExtensionDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is %s, not an extension", field, descKindWithArticle(d))
	}
	if !ext.IsExtension() {
		return nil, fmt.Errorf("%s is a normal field, not an extension", field)
	}
	return ExtensionType(ext), nil
}

func (t *typesFromResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	ext, err := t.resolver.FindExtensionByNumber(message, field)
	if err != nil {
		return nil, err
	}
	return ExtensionType(ext), nil
}

func (t *typesFromResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	d, err := t.resolver.FindDescriptorByName(message)
	if err != nil {
		return nil, err
	}
	msg, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is %s, not a message", message, descKindWithArticle(d))
	}
	return dynamicpb.NewMessageType(msg), nil
}

func (t *typesFromResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return t.FindMessageByName(TypeNameFromURL(url))
}

func (t *typesFromResolver) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	d, err := t.resolver.FindDescriptorByName(enum)
	if err != nil {
		return nil, err
	}
	en, ok := d.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is %s, not an enum", enum, descKindWithArticle(d))
	}
	return dynamicpb.NewEnumType(en), nil
}

type typesFromDescriptorPool struct {
	pool DescriptorPool
}

func (t *typesFromDescriptorPool) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	d, err := t.pool.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}
	ext, ok := d.(protoreflect.ExtensionDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is %s, not an extension", field, descKindWithArticle(d))
	}
	if !ext.IsExtension() {
		return nil, fmt.Errorf("%s is a normal field, not an extension", field)
	}
	return ExtensionType(ext), nil
}

func (t *typesFromDescriptorPool) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	var ext protoreflect.ExtensionDescriptor
	var err error
	if extRes, ok := t.pool.(ExtensionResolver); ok {
		ext, err = extRes.FindExtensionByNumber(message, field)
	} else {
		ext = FindExtensionByNumber(t.pool, message, field)
		if ext == nil {
			err = protoregistry.NotFound
		}
	}
	if err != nil {
		return nil, err
	}
	return ExtensionType(ext), nil
}

func (t *typesFromDescriptorPool) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	d, err := t.pool.FindDescriptorByName(message)
	if err != nil {
		return nil, err
	}
	msg, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is %s, not a message", message, descKindWithArticle(d))
	}
	return dynamicpb.NewMessageType(msg), nil
}

func (t *typesFromDescriptorPool) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	return t.FindMessageByName(TypeNameFromURL(url))
}

func (t *typesFromDescriptorPool) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	d, err := t.pool.FindDescriptorByName(enum)
	if err != nil {
		return nil, err
	}
	en, ok := d.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is %s, not an enum", enum, descKindWithArticle(d))
	}
	return dynamicpb.NewEnumType(en), nil
}

func (t *typesFromDescriptorPool) RangeMessages(fn func(protoreflect.MessageType) bool) {
	var rangeInContext func(container TypeContainer, fn func(protoreflect.MessageType) bool) bool
	rangeInContext = func(container TypeContainer, fn func(protoreflect.MessageType) bool) bool {
		msgs := container.Messages()
		for i, length := 0, msgs.Len(); i < length; i++ {
			msg := msgs.Get(i)
			if !fn(dynamicpb.NewMessageType(msg)) {
				return false
			}
			if !rangeInContext(msg, fn) {
				return false
			}
		}
		return true
	}
	t.pool.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return rangeInContext(file, fn)
	})
}

func (t *typesFromDescriptorPool) RangeEnums(fn func(protoreflect.EnumType) bool) {
	var rangeInContext func(container TypeContainer, fn func(protoreflect.EnumType) bool) bool
	rangeInContext = func(container TypeContainer, fn func(protoreflect.EnumType) bool) bool {
		enums := container.Enums()
		for i, length := 0, enums.Len(); i < length; i++ {
			enum := enums.Get(i)
			if !fn(dynamicpb.NewEnumType(enum)) {
				return false
			}
		}
		msgs := container.Messages()
		for i, length := 0, msgs.Len(); i < length; i++ {
			msg := msgs.Get(i)
			if !rangeInContext(msg, fn) {
				return false
			}
		}
		return true
	}
	t.pool.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return rangeInContext(file, fn)
	})
}

func (t *typesFromDescriptorPool) RangeExtensions(fn func(protoreflect.ExtensionType) bool) {
	var rangeInContext func(container TypeContainer, fn func(protoreflect.ExtensionType) bool) bool
	rangeInContext = func(container TypeContainer, fn func(protoreflect.ExtensionType) bool) bool {
		exts := container.Extensions()
		for i, length := 0, exts.Len(); i < length; i++ {
			ext := exts.Get(i)
			if !fn(ExtensionType(ext)) {
				return false
			}
		}
		msgs := container.Messages()
		for i, length := 0, msgs.Len(); i < length; i++ {
			msg := msgs.Get(i)
			if !rangeInContext(msg, fn) {
				return false
			}
		}
		return true
	}
	t.pool.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return rangeInContext(file, fn)
	})
}

func (t *typesFromDescriptorPool) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionType) bool) {
	if extPool, ok := t.pool.(ExtensionPool); ok {
		extPool.RangeExtensionsByMessage(message, func(ext protoreflect.ExtensionDescriptor) bool {
			return fn(ExtensionType(ext))
		})
		return
	}
	RangeExtensionsByMessage(t.pool, message, func(ext protoreflect.ExtensionDescriptor) bool {
		return fn(ExtensionType(ext))
	})
}
