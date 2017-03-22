package dynamic

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"

	"github.com/jhump/protoreflect/desc"
)

type ExtensionRegistry struct {
	includeDefault bool
	mu             sync.RWMutex
	exts           map[string]map[int32]*desc.FieldDescriptor
}

func NewRegistryWithDefaults() *ExtensionRegistry {
	return &ExtensionRegistry{ includeDefault: true }
}

func (r *ExtensionRegistry) AddExtensionDesc(exts ...*proto.ExtensionDesc) error {
	flds := make([]*desc.FieldDescriptor, len(exts))
	for i, ext := range exts {
		fd, err := asFieldDescriptor(ext)
		if err != nil {
			return err
		}
		flds[i] = fd
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exts == nil {
		r.exts = map[string]map[int32]*desc.FieldDescriptor{}
	}
	for _, fd := range flds {
		r.putExtensionLocked(fd)
	}
	return nil
}

func asFieldDescriptor(ext *proto.ExtensionDesc) (*desc.FieldDescriptor, error) {
	file, err := desc.LoadFileDescriptor(ext.Filename)
	if err != nil {
		return nil, err
	}
	field, ok := file.FindSymbol(ext.Name).(*desc.FieldDescriptor)
	// make sure descriptor agrees with attributes of the ExtensionDesc
	if !ok || !field.IsExtension() || field.GetOwner().GetFullyQualifiedName() != proto.MessageName(ext.ExtendedType) ||
			field.GetNumber() != ext.Field {
		return nil, fmt.Errorf("File descriptor contained unexpected object with name %s:", ext.Name)
	}
	return field, nil
}

func (r *ExtensionRegistry) AddExtension(exts ...*desc.FieldDescriptor) error {
	for _, ext := range exts {
		if !ext.IsExtension() {
			return fmt.Errorf("Given field is not an extension: %s", ext.GetFullyQualifiedName())
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exts == nil {
		r.exts = map[string]map[int32]*desc.FieldDescriptor{}
	}
	for _, ext := range exts {
		r.putExtensionLocked(ext)
	}
	return nil
}

func (r *ExtensionRegistry) AddExtensionsFromFile(fd *desc.FileDescriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exts == nil {
		r.exts = map[string]map[int32]*desc.FieldDescriptor{}
	}
	for _, ext := range fd.GetExtensions() {
		r.putExtensionLocked(ext)
	}
	for _, msg := range fd.GetMessageTypes() {
		r.addExtensionsFromMessageLocked(msg)
	}
}

func (r *ExtensionRegistry) addExtensionsFromMessageLocked(md *desc.MessageDescriptor) {
	for _, ext := range md.GetNestedExtensions() {
		r.putExtensionLocked(ext)
	}
	for _, msg := range md.GetNestedMessageTypes() {
		r.addExtensionsFromMessageLocked(msg)
	}
}

func (r *ExtensionRegistry) putExtensionLocked(fd *desc.FieldDescriptor) {
	msgName := fd.GetOwner().GetFullyQualifiedName()
	m := r.exts[msgName]
	if m == nil {
		m = map[int32]*desc.FieldDescriptor{}
		r.exts[msgName] = m
	}
	m[fd.GetNumber()] = fd
}

func (r *ExtensionRegistry) FindExtension(messageName string, tagNumber int32) *desc.FieldDescriptor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	fd := r.exts[messageName][tagNumber]
	if fd == nil && r.includeDefault {
		ext := getDefaultExtensions(messageName)[tagNumber]
		if ext != nil {
			fd, _ = asFieldDescriptor(ext)
		}
	}
	return fd
}

func (r *ExtensionRegistry) FindExtensionByName(messageName string, fieldName string) *desc.FieldDescriptor {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, fd := range r.exts[messageName] {
		if fd.GetFullyQualifiedName() == fieldName {
			return fd
		}
	}
	if r.includeDefault {
		for _, ext := range getDefaultExtensions(messageName) {
			fd, _ := asFieldDescriptor(ext)
			if fd.GetFullyQualifiedName() == fieldName {
				return fd
			}
		}
	}
	return nil
}

func getDefaultExtensions(messageName string) map[int32]*proto.ExtensionDesc {
	t := proto.MessageType(messageName)
	if t != nil {
		msg := reflect.Zero(t).Interface().(proto.Message)
		return proto.RegisteredExtensions(msg)
	}
	return nil
}

func (r *ExtensionRegistry) AllExtensionsForType(messageName string) []*desc.FieldDescriptor {
	if r == nil {
		return []*desc.FieldDescriptor(nil)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	flds := r.exts[messageName]
	var ret []*desc.FieldDescriptor
	if r.includeDefault {
		exts := getDefaultExtensions(messageName)
		if len(exts) > 0 || len(flds) > 0 {
			ret = make([]*desc.FieldDescriptor, 0, len(exts) + len(flds))
		}
		for tag, ext := range exts {
			if _, ok := flds[tag]; ok {
				// skip default extension and use the one explicitly registered instead
				continue
			}
			fd, _ := asFieldDescriptor(ext)
			if fd != nil {
				ret = append(ret, fd)
			}
		}
	} else if len(flds) > 0 {
		ret = make([]*desc.FieldDescriptor, 0, len(flds))
	}

	for _, ext := range flds {
		ret = append(ret, ext)
	}
	return ret
}