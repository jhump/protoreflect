package protoprint

import (
	"fmt"

	"github.com/jhump/protoreflect/desc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

func createTypeRegistry(fd *desc.FileDescriptor) (*protoregistry.Types, error) {
	types := &protoregistry.Types{}
	if err := appendToTypeRegistry(types, fd); err != nil {
		return nil, err
	}

	return types, nil
}

func appendToTypeRegistry(registry *protoregistry.Types, fd *desc.FileDescriptor) error {
	fds := desc.ToFileDescriptorSet(fd)

	fr, err := protodesc.NewFiles(fds)
	if err != nil {
		return fmt.Errorf("creating new descriptor files, %w", err)
	}

	var fileErr error
	fr.RangeFiles(func(fileDescriptor protoreflect.FileDescriptor) bool {
		if err := visitFile(registry, fileDescriptor); err != nil {
			fileErr = fmt.Errorf("visiting file %s, %w", fileDescriptor.FullName(), err)
			return false
		}
		return true
	})
	if fileErr != nil {
		return fileErr
	}

	return nil
}

func visitFile(types *protoregistry.Types, fd protoreflect.FileDescriptor) error {

	messages := fd.Messages()
	for i := 0; i < messages.Len(); i++ {
		msg := messages.Get(i)
		if err := visitMessage(types, msg); err != nil {
			return fmt.Errorf("visiting message %s, %w", msg.FullName(), err)
		}
	}

	enums := fd.Enums()
	for i := 0; i < enums.Len(); i++ {
		enumDesc := enums.Get(i)
		if err := visitEnum(types, enumDesc); err != nil {
			return fmt.Errorf("visiting enum %s, %w", enumDesc.FullName(), err)
		}
	}

	exts := fd.Extensions()
	for i := 0; i < exts.Len(); i++ {
		extDesc := exts.Get(i)
		if err := visitExt(types, extDesc); err != nil {
			return fmt.Errorf("visiting extension %s, %w", extDesc.FullName(), err)
		}
	}

	return nil
}

func visitMessage(types *protoregistry.Types, msgDesc protoreflect.MessageDescriptor) error {
	if err := types.RegisterMessage(dynamicpb.NewMessageType(msgDesc)); err != nil {
		return fmt.Errorf("registering message %s, %w", msgDesc.FullName(), err)
	}

	nestedMessages := msgDesc.Messages()
	for i := 0; i < nestedMessages.Len(); i++ {
		nestedMessage := nestedMessages.Get(i)
		if err := visitMessage(types, nestedMessage); err != nil {
			return fmt.Errorf("visiting message %s, %w", nestedMessage.FullName(), err)
		}
	}

	nestedEnums := msgDesc.Enums()
	for i := 0; i < nestedEnums.Len(); i++ {
		nestedEnum := nestedEnums.Get(i)
		if err := visitEnum(types, nestedEnum); err != nil {
			return fmt.Errorf("visiting enum %s, %w", nestedEnum.FullName(), err)
		}
	}

	nestedExts := msgDesc.Extensions()
	for i := 0; i < nestedExts.Len(); i++ {
		nestedExt := nestedExts.Get(i)
		if err := visitExt(types, nestedExt); err != nil {
			return fmt.Errorf("visiting ext %s, %w ", nestedExt.FullName(), err)
		}
	}

	return nil
}

func visitEnum(types *protoregistry.Types, enumDesc protoreflect.EnumDescriptor) error {
	return types.RegisterEnum(dynamicpb.NewEnumType(enumDesc))
}

func visitExt(types *protoregistry.Types, extDesc protoreflect.ExtensionDescriptor) error {
	return types.RegisterExtension(dynamicpb.NewExtensionType(extDesc))
}

//type customRegistry struct {
//	messages map[protoreflect.FullName]desc.MessageDescriptor
//	enums    map[protoreflect.FullName]desc.EnumDescriptor
//	exts     map[protoreflect.FullName]protoreflect.ExtensionDescriptor
//}
//
//func createTypeRegistry2(fd *desc.FileDescriptor) (*protoregistry.Types, error) {
//	types := &protoregistry.Types{}
//	if err := appendToTypeRegistry(types, fd); err != nil {
//		return nil, err
//	}
//
//	return types, nil
//}
//
//func appendToTypeRegistry2(registry *customRegistry, fd *desc.FileDescriptor) error {
//	fds := desc.ToFileDescriptorSet(fd)
//
//	fr, err := protodesc.NewFiles(fds)
//	if err != nil {
//		return fmt.Errorf("creating new descriptor files, %w", err)
//	}
//
//	var fileErr error
//	fr.RangeFiles(func(fileDescriptor protoreflect.FileDescriptor) bool {
//		if err := visitFile2(registry, fileDescriptor); err != nil {
//			fileErr = fmt.Errorf("visiting file %s, %w", fileDescriptor.FullName(), err)
//			return false
//		}
//		return true
//	})
//	if fileErr != nil {
//		return fileErr
//	}
//
//	return nil
//}
//
//func visitFile2(registry *customRegistry, fd *desc.FileDescriptor) error {
//
//	messages := fd.Messages()
//	for _, descriptor := range fd.GetMessageTypes() {
//		if err := visitMessage2(types, descriptor); err != nil {
//			return fmt.Errorf("visiting2 message %s, %w", descriptor.GetFullyQualifiedName(), err)
//		}
//	}
//	for i := 0; i < messages.Len(); i++ {
//		msg := messages.Get(i)
//		if err := visitMessage2(types, msg); err != nil {
//			return fmt.Errorf("visiting2 message %s, %w", msg.FullName(), err)
//		}
//	}
//
//	enums := fd.Enums()
//	for i := 0; i < enums.Len(); i++ {
//		enumDesc := enums.Get(i)
//		if err := visitEnum2(types, enumDesc); err != nil {
//			return fmt.Errorf("visiting2 enum %s, %w", enumDesc.FullName(), err)
//		}
//	}
//
//	exts := fd.Extensions()
//	for i := 0; i < exts.Len(); i++ {
//		extDesc := exts.Get(i)
//		if err := visitExt2(types, extDesc); err != nil {
//			return fmt.Errorf("visiting2 extension %s, %w", extDesc.FullName(), err)
//		}
//	}
//
//	return nil
//}
//
//func visitMessage2(registry *customRegistry, msgDesc protoreflect.MessageDescriptor) error {
//	if err := types.RegisterMessage(dynamicpb.NewMessageType(msgDesc)); err != nil {
//		return fmt.Errorf("registering message %s, %w", msgDesc.FullName(), err)
//	}
//
//	nestedMessages := msgDesc.Messages()
//	for i := 0; i < nestedMessages.Len(); i++ {
//		nestedMessage := nestedMessages.Get(i)
//		if err := visitMessage2(types, nestedMessage); err != nil {
//			return fmt.Errorf("visiting2 message %s, %w", nestedMessage.FullName(), err)
//		}
//	}
//
//	nestedEnums := msgDesc.Enums()
//	for i := 0; i < nestedEnums.Len(); i++ {
//		nestedEnum := nestedEnums.Get(i)
//		if err := visitEnum2(types, nestedEnum); err != nil {
//			return fmt.Errorf("visiting2 enum %s, %w", nestedEnum.FullName(), err)
//		}
//	}
//
//	return nil
//}
//
//func visitEnum2(registry *customRegistry, enumDesc protoreflect.EnumDescriptor) error {
//	return types.RegisterEnum(dynamicpb.NewEnumType(enumDesc))
//}
//
//func visitExt2(registry *customRegistry, extDesc protoreflect.ExtensionDescriptor) error {
//	return types.RegisterExtension(dynamicpb.NewExtensionType(extDesc))
//}
