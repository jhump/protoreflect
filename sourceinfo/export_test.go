package sourceinfo

import "google.golang.org/protobuf/reflect/protoreflect"

// exported for tests

func CanUpgrade(d protoreflect.Descriptor) bool {
	return canUpgrade(d)
}

func UpdateDescriptor[D protoreflect.Descriptor](d D) (D, error) {
	return updateDescriptor(d)
}

func UpdateField(fld protoreflect.FieldDescriptor) (protoreflect.FieldDescriptor, error) {
	return updateField(fld)
}
