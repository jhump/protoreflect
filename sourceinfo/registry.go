package sourceinfo

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"sync"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

var (
	// Files is a registry of descriptors that include source code info, if the
	// files they belong to were processed with protoc-gen-gosrcinfo.
	//
	// It is meant to serve as a drop-in alternative to protoregistry.GlobalFiles
	// that can include source code info in the returned descriptors.
	Files protoresolve.DescriptorPool = files{}

	// Types is a registry of types that include source code info, if the
	// files they belong to were processed with protoc-gen-gosrcinfo.
	//
	// It is meant to serve as a drop-in alternative to protoregistry.GlobalTypes
	// that can include source code info in the returned types.
	Types protoresolve.TypePool = types{}

	mu                   sync.RWMutex
	sourceInfoDataByFile = map[string][]byte{}
	sourceInfoByFile     = map[string]*descriptorpb.SourceCodeInfo{}
	fileDescriptors      = map[protoreflect.FileDescriptor]protoreflect.FileDescriptor{}
)

// Register registers the given source code info, which is a serialized
// and gzipped form of a google.protobuf.SourceCodeInfo message.
//
// This is automatically used from generated code if using the protoc-gen-gosrcinfo
// plugin.
func Register(file string, data []byte) {
	mu.Lock()
	defer mu.Unlock()
	sourceInfoDataByFile[file] = data
}

// ForFile queries for any registered source code info for the file
// descriptor with the given path/name. It returns nil if no source code info
// was registered.
func ForFile(file string) (*descriptorpb.SourceCodeInfo, error) {
	mu.RLock()
	srcInfo := sourceInfoByFile[file]
	var data []byte
	if srcInfo == nil {
		data = sourceInfoDataByFile[file]
	}
	mu.RUnlock()

	if srcInfo != nil {
		return srcInfo, nil
	}
	if data == nil {
		return nil, nil
	}

	zipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = zipReader.Close()
	}()
	unzipped, err := io.ReadAll(zipReader)
	if err != nil {
		return nil, err
	}
	srcInfo = &descriptorpb.SourceCodeInfo{}
	if err := proto.Unmarshal(unzipped, srcInfo); err != nil {
		return nil, err
	}

	mu.Lock()
	defer mu.Unlock()
	// check again after upgrading lock
	if existing := sourceInfoByFile[file]; existing != nil {
		srcInfo = existing
	} else {
		sourceInfoByFile[file] = srcInfo
	}
	return srcInfo, nil
}

func canWrap(d protoreflect.Descriptor) bool {
	srcInfo, _ := ForFile(d.ParentFile().Path())
	return len(srcInfo.GetLocation()) > 0
}

func getFile(fd protoreflect.FileDescriptor) protoreflect.FileDescriptor {
	if fd == nil {
		return nil
	}

	mu.RLock()
	result := fileDescriptors[fd]
	mu.RUnlock()

	if result != nil {
		return result
	}

	srcInfo, _ := ForFile(fd.Path())
	if len(srcInfo.GetLocation()) > 0 {
		result = &fileDescriptor{
			FileDescriptor: fd,
			locs: &sourceLocations{
				orig: srcInfo.Location,
			},
		}
	} else {
		// nothing to do; don't bother wrapping
		result = fd
	}

	mu.Lock()
	// double-check, in case it was already concurrently added
	if existing := fileDescriptors[fd]; existing != nil {
		result = existing
	} else {
		fileDescriptors[fd] = result
	}
	mu.Unlock()
	return result
}

type files struct{}

func (files) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	fd, err := protoregistry.GlobalFiles.FindFileByPath(path)
	if err != nil {
		return nil, err
	}
	return getFile(fd), nil
}

func (files) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if !canWrap(d) {
		return d, nil
	}
	if err != nil {
		return nil, err
	}
	switch d := d.(type) {
	case protoreflect.FileDescriptor:
		return getFile(d), nil
	case protoreflect.MessageDescriptor:
		return messageDescriptor{d}, nil
	case protoreflect.ExtensionTypeDescriptor:
		return extensionTypeDescriptor{d}, nil
	case protoreflect.FieldDescriptor:
		return fieldDescriptor{d}, nil
	case protoreflect.OneofDescriptor:
		return oneofDescriptor{d}, nil
	case protoreflect.EnumDescriptor:
		return enumDescriptor{d}, nil
	case protoreflect.EnumValueDescriptor:
		return enumValueDescriptor{d}, nil
	case protoreflect.ServiceDescriptor:
		return serviceDescriptor{d}, nil
	case protoreflect.MethodDescriptor:
		return methodDescriptor{d}, nil
	default:
		return nil, fmt.Errorf("unrecognized descriptor type: %T", d)
	}
}

func (files) NumFiles() int {
	return protoregistry.GlobalFiles.NumFiles()
}

func (files) RangeFiles(fn func(protoreflect.FileDescriptor) bool) {
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		return fn(getFile(file))
	})
}

func (files) NumFilesByPackage(name protoreflect.FullName) int {
	return protoregistry.GlobalFiles.NumFilesByPackage(name)
}

func (files) RangeFilesByPackage(name protoreflect.FullName, fn func(protoreflect.FileDescriptor) bool) {
	protoregistry.GlobalFiles.RangeFilesByPackage(name, func(file protoreflect.FileDescriptor) bool {
		return fn(getFile(file))
	})
}

type types struct{}

func (types) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(message)
	if err != nil {
		return nil, err
	}
	if !canWrap(mt.Descriptor()) {
		return mt, nil
	}
	return messageType{mt}, nil
}

func (types) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	mt, err := protoregistry.GlobalTypes.FindMessageByURL(url)
	if err != nil {
		return nil, err
	}
	if !canWrap(mt.Descriptor()) {
		return mt, nil
	}
	return messageType{mt}, nil
}

func (types) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	xt, err := protoregistry.GlobalTypes.FindExtensionByName(field)
	if err != nil {
		return nil, err
	}
	if !canWrap(xt.TypeDescriptor()) {
		return xt, nil
	}
	return extensionType{xt}, nil
}

func (types) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	xt, err := protoregistry.GlobalTypes.FindExtensionByNumber(message, field)
	if err != nil {
		return nil, err
	}
	if !canWrap(xt.TypeDescriptor()) {
		return xt, nil
	}
	return extensionType{xt}, nil
}

func (types) FindEnumByName(enum protoreflect.FullName) (protoreflect.EnumType, error) {
	et, err := protoregistry.GlobalTypes.FindEnumByName(enum)
	if err != nil {
		return nil, err
	}
	if !canWrap(et.Descriptor()) {
		return et, nil
	}
	return enumType{et}, nil
}

func (types) RangeMessages(fn func(protoreflect.MessageType) bool) {
	protoregistry.GlobalTypes.RangeMessages(func(mt protoreflect.MessageType) bool {
		if canWrap(mt.Descriptor()) {
			mt = messageType{mt}
		}
		return fn(mt)
	})
}

func (types) RangeEnums(fn func(protoreflect.EnumType) bool) {
	protoregistry.GlobalTypes.RangeEnums(func(et protoreflect.EnumType) bool {
		if canWrap(et.Descriptor()) {
			et = enumType{et}
		}
		return fn(et)
	})
}

func (types) RangeExtensions(fn func(protoreflect.ExtensionType) bool) {
	protoregistry.GlobalTypes.RangeExtensions(func(xt protoreflect.ExtensionType) bool {
		if canWrap(xt.TypeDescriptor()) {
			xt = extensionType{xt}
		}
		return fn(xt)
	})
}

func (types) RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionType) bool) {
	protoregistry.GlobalTypes.RangeExtensionsByMessage(message, func(xt protoreflect.ExtensionType) bool {
		if canWrap(xt.TypeDescriptor()) {
			xt = extensionType{xt}
		}
		return fn(xt)
	})
}
