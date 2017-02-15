package desc

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"reflect"
	"sync"

	"github.com/golang/protobuf/proto"
	dpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

var (
	cacheMu sync.RWMutex
	filesCache = map[string]*FileDescriptor{}
	messagesCache = map[string]*MessageDescriptor{}
)

// LoadFileDescriptor creates a file descriptor using the bytes returned by
// proto.FileDescriptor. Descriptors are cached so that they do not need to be
// re-processed if the same file is fetched again later.
func LoadFileDescriptor(file string) (*FileDescriptor, error) {
	f := getFileFromCache(file)
	if f != nil {
		return f, nil
	}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	return loadFileDescriptorLocked(file)
}

func loadFileDescriptorLocked(file string) (*FileDescriptor, error) {
	f := filesCache[file]
	if f != nil {
		return f, nil
	}

	fdb := proto.FileDescriptor(file)
	if fdb == nil {
		return nil, fmt.Errorf("No such file: %q", file)
	}

	fd, err := decodeFileDescriptor(file, fdb)
	if err != nil {
		return nil, err
	}

	f, err = toFileDescriptorLocked(fd)
	if err != nil {
		return nil, err
	}
	putCacheLocked(file, f)
	return f, nil
}

func toFileDescriptorLocked(fd *dpb.FileDescriptorProto) (*FileDescriptor, error) {
	deps := make([]*FileDescriptor, len(fd.GetDependency()))
	for i, dep := range(fd.GetDependency()) {
		var err error
		deps[i], err = loadFileDescriptorLocked(dep)
		if err != nil {
			return nil, err
		}
	}
	return CreateFileDescriptor(fd, deps...)
}

func decodeFileDescriptor(file string, fdb []byte) (*dpb.FileDescriptorProto, error) {
	raw, err := decompress(fdb)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress %q descriptor: %v", file, err)
	}
	fd := dpb.FileDescriptorProto{}
	if err := proto.Unmarshal(raw, &fd); err != nil {
		return nil, fmt.Errorf("bad descriptor for %q: %v", file, err)
	}
	return &fd, nil
}

func decompress(b []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	out, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	return out, nil
}

func getFileFromCache(file string) *FileDescriptor {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return filesCache[file]
}

func putCacheLocked(filename string, fd *FileDescriptor) {
	filesCache[filename] = fd
	putMessageCacheLocked(fd.messages)
}

func putMessageCacheLocked(mds []*MessageDescriptor) {
	for _, md := range(mds) {
		messagesCache[md.fqn] = md
		putMessageCacheLocked(md.nested)
	}
}

// interface implemented by generated messages, which all have a Descriptor() method in
// addition to the methods of proto.Message
type protoMessage interface {
	proto.Message
	Descriptor() ([]byte, []int)
}

// LoadMessageDescriptor loads descriptor using the encoded descriptor proto returned by
// Message.Descriptor() for the given message type.
func LoadMessageDescriptor(message string) (*MessageDescriptor, error) {
	m := getMessageFromCache(message)
	if m != nil {
		return m, nil
	}

	pt := proto.MessageType(message)
	if pt == nil {
		return nil, fmt.Errorf("unknown type: %q", message)
	}
	msg, err := messageFromType(pt)
	if err != nil {
		return nil, err
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()
	return loadMessageDescriptorForTypeLocked(message, msg)
}

// LoadMessageDescriptorForType loads descriptor using the encoded descriptor proto returned
// by Message.Descriptor() for the given message type.
func LoadMessageDescriptorForType(messageType reflect.Type) (*MessageDescriptor, error) {
	m, err := messageFromType(messageType)
	if err != nil {
		return nil, err
	}
	return LoadMessageDescriptorForMessage(m)
}

// LoadMessageDescriptorForMessage loads descriptor using the encoded descriptor proto
// returned by message.Descriptor().
func LoadMessageDescriptorForMessage(message proto.Message) (*MessageDescriptor, error) {
	name := proto.MessageName(message)
	m := getMessageFromCache(name)
	if m != nil {
		return m, nil
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()
	return loadMessageDescriptorForTypeLocked(name, message.(protoMessage))
}

func messageFromType(mt reflect.Type) (protoMessage, error) {
	if mt.Kind() != reflect.Ptr {
		mt = reflect.PtrTo(mt)
	}
	m, ok := reflect.Zero(mt).Interface().(protoMessage)
	if !ok {
		return nil, fmt.Errorf("failed to create message from type: %v", mt)
	}
	return m, nil
}

func loadMessageDescriptorForTypeLocked(name string, message protoMessage) (*MessageDescriptor, error) {
	m := messagesCache[name]
	if m != nil {
		return m, nil
	}

	fdb, _ := message.Descriptor()
	fd, err := decodeFileDescriptor(name, fdb)
	if err != nil {
		return nil, err
	}

	f, err := toFileDescriptorLocked(fd)
	if err != nil {
		return nil, err
	}
	putCacheLocked(fd.GetName(), f)
	return f.FindSymbol(name).(*MessageDescriptor), nil
}

func getMessageFromCache(message string) *MessageDescriptor {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return messagesCache[message]
}
