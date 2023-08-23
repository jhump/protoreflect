package protoresolve

import (
	"fmt"
	"sync"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Registry implements the full Resolver interface defined in this package. It is
// thread-safe and can be used for all kinds of operations where types or descriptors
// may need to be resolved from names or numbers.
type Registry struct {
	mu    sync.RWMutex
	files protoregistry.Files
	exts  map[protoreflect.FullName]map[protoreflect.FieldNumber]protoreflect.FieldDescriptor
}

var _ Resolver = (*Registry)(nil)

// FromFiles returns a new registry that wraps the given files. After creating
// this registry, callers should not directly use files -- most especially, they
// should not register any additional descriptors with files and should instead
// use the RegisterFile method of the returned registry.
//
// This may return an error if the given files includes conflicting extension
// definitions (i.e. more than one extension for the same extended message and
// tag number).
//
// If protoregistry.GlobalFiles is supplied, a deep copy is made first. To avoid
// such a copy, use GlobalDescriptors instead.
func FromFiles(files *protoregistry.Files) (*Registry, error) {
	if files == protoregistry.GlobalFiles {
		// Don't wrap files if it's the global registry; make an effective copy
		reg := &Registry{}
		var err error
		files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			err = reg.RegisterFile(fd)
			return err == nil
		})
		if err != nil {
			return nil, err
		}
		return reg, nil
	}

	reg := &Registry{
		files: *files,
	}
	// NB: It's okay to call methods below without first acquiring
	// lock because reg is not visible to any other goroutines yet.
	var err error
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		err = reg.checkExtensionsLocked(fd)
		return err == nil
	})
	if err != nil {
		return nil, err
	}
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		reg.registerExtensionsLocked(fd)
		return true
	})
	return reg, nil
}

// RegisterFile implements part of the Resolver interface.
func (r *Registry) RegisterFile(file protoreflect.FileDescriptor) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkExtensionsLocked(file); err != nil {
		_, findFileErr := r.files.FindFileByPath(file.Path())
		if findFileErr == nil {
			return fmt.Errorf("file %q already registered", file.Path())
		}
		return err
	}
	if err := r.files.RegisterFile(file); err != nil {
		return err
	}
	r.registerExtensionsLocked(file)
	return nil
}

func (r *Registry) checkExtensionsLocked(container TypeContainer) error {
	exts := container.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		ext := exts.Get(i)
		existing := r.exts[ext.ContainingMessage().FullName()][ext.Number()]
		if existing != nil {
			if existing.FullName() == ext.FullName() {
				return fmt.Errorf("extension named %q already registered", ext.FullName())
			}
			return fmt.Errorf("extension number %d for message %q already registered (existing: %q; trying to register: %q)",
				ext.Number(), ext.ContainingMessage().FullName(), existing.FullName(), ext.FullName())
		}
	}

	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		if err := r.checkExtensionsLocked(msgs.Get(i)); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) registerExtensionsLocked(container TypeContainer) {
	exts := container.Extensions()
	for i, length := 0, exts.Len(); i < length; i++ {
		ext := exts.Get(i)
		if r.exts == nil {
			r.exts = map[protoreflect.FullName]map[protoreflect.FieldNumber]protoreflect.FieldDescriptor{}
		}
		extsForMsg := r.exts[ext.ContainingMessage().FullName()]
		if extsForMsg == nil {
			extsForMsg = map[protoreflect.FieldNumber]protoreflect.FieldDescriptor{}
			r.exts[ext.ContainingMessage().FullName()] = extsForMsg
		}
		extsForMsg[ext.Number()] = ext
	}

	msgs := container.Messages()
	for i, length := 0, msgs.Len(); i < length; i++ {
		r.registerExtensionsLocked(msgs.Get(i))
	}
}

// FindFileByPath implements part of the Resolver interface.
func (r *Registry) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.files.FindFileByPath(path)
}

// NumFiles implements part of the FilePool interface.
func (r *Registry) NumFiles() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.files.NumFiles()
}

// RangeFiles implements part of the FilePool interface.
func (r *Registry) RangeFiles(fn func(protoreflect.FileDescriptor) bool) {
	var files []protoreflect.FileDescriptor
	func() {
		r.mu.RLock()
		defer r.mu.RUnlock()
		files = make([]protoreflect.FileDescriptor, r.files.NumFiles())
		i := 0
		r.files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
			files[i] = f
			i++
			return true
		})
	}()
	for _, file := range files {
		if !fn(file) {
			return
		}
	}
}

// NumFilesByPackage implements part of the FilePool interface.
func (r *Registry) NumFilesByPackage(name protoreflect.FullName) int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.files.NumFilesByPackage(name)
}

// RangeFilesByPackage implements part of the FilePool interface.
func (r *Registry) RangeFilesByPackage(name protoreflect.FullName, fn func(protoreflect.FileDescriptor) bool) {
	var files []protoreflect.FileDescriptor
	func() {
		r.mu.RLock()
		defer r.mu.RUnlock()
		files = make([]protoreflect.FileDescriptor, r.files.NumFilesByPackage(name))
		i := 0
		r.files.RangeFilesByPackage(name, func(f protoreflect.FileDescriptor) bool {
			files[i] = f
			i++
			return true
		})
	}()
	for _, file := range files {
		if !fn(file) {
			return
		}
	}
}

// FindDescriptorByName implements part of the Resolver interface.
func (r *Registry) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.files.FindDescriptorByName(name)
}

func descType(d protoreflect.Descriptor) string {
	switch d := d.(type) {
	case protoreflect.FileDescriptor:
		return "a file"
	case protoreflect.MessageDescriptor:
		return "a message"
	case protoreflect.FieldDescriptor:
		if d.IsExtension() {
			return "an extension"
		}
		return "a field"
	case protoreflect.OneofDescriptor:
		return "a oneof"
	case protoreflect.EnumDescriptor:
		return "an enum"
	case protoreflect.EnumValueDescriptor:
		return "an enum value"
	case protoreflect.ServiceDescriptor:
		return "a service"
	case protoreflect.MethodDescriptor:
		return "a method"
	default:
		return fmt.Sprintf("a %T", d)
	}
}

// FindMessageByName implements part of the Resolver interface.
func (r *Registry) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	msg, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a message", name, descType(d))
	}
	return msg, nil
}

// FindFieldByName implements part of the Resolver interface.
func (r *Registry) FindFieldByName(name protoreflect.FullName) (protoreflect.FieldDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	fld, ok := d.(protoreflect.FieldDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a field", name, descType(d))
	}
	return fld, nil
}

// FindExtensionByName implements part of the Resolver interface.
func (r *Registry) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	fld, ok := d.(protoreflect.FieldDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not an extension", name, descType(d))
	}
	if !fld.IsExtension() {
		return nil, fmt.Errorf("descriptor %q is a field, not an extension", name)
	}
	return fld, nil
}

// FindOneofByName implements part of the Resolver interface.
func (r *Registry) FindOneofByName(name protoreflect.FullName) (protoreflect.OneofDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	ood, ok := d.(protoreflect.OneofDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a oneof", name, descType(d))
	}
	return ood, nil
}

// FindEnumByName implements part of the Resolver interface.
func (r *Registry) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	en, ok := d.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not an enum", name, descType(d))
	}
	return en, nil
}

// FindEnumValueByName implements part of the Resolver interface.
func (r *Registry) FindEnumValueByName(name protoreflect.FullName) (protoreflect.EnumValueDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	enVal, ok := d.(protoreflect.EnumValueDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not an enum value", name, descType(d))
	}
	return enVal, nil
}

// FindServiceByName implements part of the Resolver interface.
func (r *Registry) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	svc, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a service", name, descType(d))
	}
	return svc, nil
}

// FindMethodByName implements part of the Resolver interface.
func (r *Registry) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	d, err := r.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	mtd, ok := d.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a method", name, descType(d))
	}
	return mtd, nil
}

// FindExtensionByNumber implements part of the Resolver interface.
func (r *Registry) FindExtensionByNumber(message protoreflect.FullName, fieldNumber protoreflect.FieldNumber) (protoreflect.ExtensionDescriptor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ext := r.exts[message][fieldNumber]
	if ext == nil {
		return nil, protoregistry.NotFound
	}
	return ext, nil
}

// FindMessageByURL implements part of the Resolver interface.
func (r *Registry) FindMessageByURL(url string) (protoreflect.MessageDescriptor, error) {
	return r.FindMessageByName(TypeNameFromURL(url))
}

// RangeExtensionsByMessage implements part of the Resolver interface.
func (r *Registry) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionDescriptor) bool) {
	var exts []protoreflect.ExtensionDescriptor
	func() {
		r.mu.RLock()
		defer r.mu.RUnlock()
		extMap := r.exts[message]
		if len(extMap) == 0 {
			return
		}
		exts = make([]protoreflect.ExtensionDescriptor, len(extMap))
		i := 0
		for _, v := range extMap {
			exts[i] = v
			i++
		}
	}()
	for _, ext := range exts {
		if !fn(ext) {
			return
		}
	}
}

// AsTypeResolver implements part of the Resolver interface.
func (r *Registry) AsTypeResolver() TypeResolver {
	return r.AsTypePool()
}

// AsTypePool returns a view of this registry as a TypePool. This offers more methods
// than AsTypeResolver, providing the ability to enumerate types.
func (r *Registry) AsTypePool() TypePool {
	return TypesFromDescriptorPool(r)
}
