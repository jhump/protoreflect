package protoresolve

import (
	"fmt"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// GlobalDescriptors provides a view of protoregistry.GlobalFiles and protoregistry.GlobalTypes
// as a Resolver.
var GlobalDescriptors Resolver = globalResolver{}

type globalResolver struct{}

func (g globalResolver) FindFileByPath(path string) (protoreflect.FileDescriptor, error) {
	return protoregistry.GlobalFiles.FindFileByPath(path)
}

func (g globalResolver) NumFiles() int {
	return protoregistry.GlobalFiles.NumFiles()
}

func (g globalResolver) RangeFiles(f func(protoreflect.FileDescriptor) bool) {
	protoregistry.GlobalFiles.RangeFiles(f)
}

func (g globalResolver) NumFilesByPackage(name protoreflect.FullName) int {
	return protoregistry.GlobalFiles.NumFilesByPackage(name)
}

func (g globalResolver) RangeFilesByPackage(name protoreflect.FullName, f func(protoreflect.FileDescriptor) bool) {
	protoregistry.GlobalFiles.RangeFilesByPackage(name, f)
}

func (g globalResolver) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	return protoregistry.GlobalFiles.FindDescriptorByName(name)
}

func (g globalResolver) FindMessageByName(name protoreflect.FullName) (protoreflect.MessageDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	msg, ok := d.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a message", name, descType(d))
	}
	return msg, nil
}

func (g globalResolver) FindFieldByName(name protoreflect.FullName) (protoreflect.FieldDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	fld, ok := d.(protoreflect.FieldDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a field", name, descType(d))
	}
	return fld, nil
}

func (g globalResolver) FindExtensionByName(name protoreflect.FullName) (protoreflect.ExtensionDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
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

func (g globalResolver) FindOneofByName(name protoreflect.FullName) (protoreflect.OneofDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	ood, ok := d.(protoreflect.OneofDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a oneof", name, descType(d))
	}
	return ood, nil
}

func (g globalResolver) FindEnumByName(name protoreflect.FullName) (protoreflect.EnumDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	en, ok := d.(protoreflect.EnumDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not an enum", name, descType(d))
	}
	return en, nil
}

func (g globalResolver) FindEnumValueByName(name protoreflect.FullName) (protoreflect.EnumValueDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	enVal, ok := d.(protoreflect.EnumValueDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not an enum value", name, descType(d))
	}
	return enVal, nil
}

func (g globalResolver) FindServiceByName(name protoreflect.FullName) (protoreflect.ServiceDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	svc, ok := d.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a service", name, descType(d))
	}
	return svc, nil
}

func (g globalResolver) FindMethodByName(name protoreflect.FullName) (protoreflect.MethodDescriptor, error) {
	d, err := protoregistry.GlobalFiles.FindDescriptorByName(name)
	if err != nil {
		return nil, err
	}
	mtd, ok := d.(protoreflect.MethodDescriptor)
	if !ok {
		return nil, fmt.Errorf("descriptor %q is %s, not a method", name, descType(d))
	}
	return mtd, nil
}

func (g globalResolver) FindExtensionByNumber(message protoreflect.FullName, number protoreflect.FieldNumber) (protoreflect.ExtensionDescriptor, error) {
	ext, err := protoregistry.GlobalTypes.FindExtensionByNumber(message, number)
	if err != nil {
		return nil, err
	}
	return ext.TypeDescriptor(), nil
}

func (g globalResolver) RangeExtensionsByMessage(message protoreflect.FullName, fn func(descriptor protoreflect.ExtensionDescriptor) bool) {
	protoregistry.GlobalTypes.RangeExtensionsByMessage(message, func(ext protoreflect.ExtensionType) bool {
		return fn(ext.TypeDescriptor())
	})
}

func (g globalResolver) FindMessageByURL(url string) (protoreflect.MessageDescriptor, error) {
	msg, err := protoregistry.GlobalTypes.FindMessageByURL(url)
	if err != nil {
		return nil, err
	}
	return msg.Descriptor(), nil
}

func (g globalResolver) AsTypeResolver() TypeResolver {
	return protoregistry.GlobalTypes
}
