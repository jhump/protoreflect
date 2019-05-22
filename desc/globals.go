package desc

import (
	"reflect"
	"sync"
)

var (
	globalImportPathConf = newOptionalImportPathCache()
	globalImportPathMu   = newOptionalRWMutex()

	cacheMu       = newOptionalRWMutex()
	filesCache    = newOptionalFileDescriptorCache()
	messagesCache = newOptionalMessageDescriptorCache()
	enumCache     = newOptionalEnumDescriptorCache()
)

// TurnOffGlobals turns off usage of all global variables.
func TurnOffGlobals() {
	globalImportPathConf = nil
	globalImportPathMu = nil
	cacheMu = nil
	filesCache = nil
	messagesCache = nil
	enumCache = nil
}

type optionalRWMutex struct {
	mu sync.RWMutex
}

func newOptionalRWMutex() *optionalRWMutex {
	return &optionalRWMutex{}
}

func (r *optionalRWMutex) RLock() {
	if r == nil {
		return
	}
	r.mu.RLock()
}

func (r *optionalRWMutex) RUnlock() {
	if r == nil {
		return
	}
	r.mu.RUnlock()
}

func (r *optionalRWMutex) Lock() {
	if r == nil {
		return
	}
	r.mu.Lock()
}

func (r *optionalRWMutex) Unlock() {
	if r == nil {
		return
	}
	r.mu.Unlock()
}

type optionalImportPathCache struct {
	m map[string]string
}

func newOptionalImportPathCache() *optionalImportPathCache {
	return &optionalImportPathCache{m: make(map[string]string)}
}

func (i *optionalImportPathCache) Get(key string) (string, bool) {
	if i == nil {
		return "", false
	}
	value, ok := i.m[key]
	return value, ok
}

func (i *optionalImportPathCache) Set(key string, value string) {
	if i == nil {
		return
	}
	i.m[key] = value
}

type optionalFileDescriptorCache struct {
	m map[string]*FileDescriptor
}

func newOptionalFileDescriptorCache() *optionalFileDescriptorCache {
	return &optionalFileDescriptorCache{m: make(map[string]*FileDescriptor)}
}

func (i *optionalFileDescriptorCache) Get(key string) (*FileDescriptor, bool) {
	if i == nil {
		return nil, false
	}
	value, ok := i.m[key]
	return value, ok
}

func (i *optionalFileDescriptorCache) Set(key string, value *FileDescriptor) {
	if i == nil {
		return
	}
	i.m[key] = value
}

type optionalMessageDescriptorCache struct {
	m map[string]*MessageDescriptor
}

func newOptionalMessageDescriptorCache() *optionalMessageDescriptorCache {
	return &optionalMessageDescriptorCache{m: make(map[string]*MessageDescriptor)}
}

func (i *optionalMessageDescriptorCache) Get(key string) (*MessageDescriptor, bool) {
	if i == nil {
		return nil, false
	}
	value, ok := i.m[key]
	return value, ok
}

func (i *optionalMessageDescriptorCache) Set(key string, value *MessageDescriptor) {
	if i == nil {
		return
	}
	i.m[key] = value
}

type optionalEnumDescriptorCache struct {
	m map[reflect.Type]*EnumDescriptor
}

func newOptionalEnumDescriptorCache() *optionalEnumDescriptorCache {
	return &optionalEnumDescriptorCache{m: make(map[reflect.Type]*EnumDescriptor)}
}

func (i *optionalEnumDescriptorCache) Get(key reflect.Type) (*EnumDescriptor, bool) {
	if i == nil {
		return nil, false
	}
	value, ok := i.m[key]
	return value, ok
}

func (i *optionalEnumDescriptorCache) Set(key reflect.Type, value *EnumDescriptor) {
	if i == nil {
		return
	}
	i.m[key] = value
}
