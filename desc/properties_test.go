package desc

import (
	"reflect"
	"testing"

	"github.com/jhump/protoreflect/desc/desc_test"
)

func TestLoadFileDescriptor(t *testing.T) {
	fd, err := LoadFileDescriptor("desc_test1.proto")
	ok(t, err)
	// some very shallow tests (we have more detailed ones in other test cases)
	eq(t, "desc_test1.proto", fd.GetName())
	eq(t, "desc_test1.proto", fd.GetFullyQualifiedName())
	eq(t, "desc_test", fd.GetPackage())
}

func TestLoadMessageDescriptor(t *testing.T) {
	// loading enclosed messages should return the same descriptor
	// and have a reference to the same file descriptor
	md, err := LoadMessageDescriptor("desc_test.TestMessage")
	ok(t, err)
	eq(t, "TestMessage", md.GetName())
	eq(t, "desc_test.TestMessage", md.GetFullyQualifiedName())
	fd := md.GetFile()
	eq(t, "desc_test1.proto", fd.GetName())
	eq(t, fd, md.GetParent())

	md2, err := LoadMessageDescriptorForMessage((*desc_test.TestMessage)(nil))
	ok(t, err)
	eq(t, md, md2)

	md3, err := LoadMessageDescriptorForType(reflect.TypeOf((*desc_test.TestMessage)(nil)))
	ok(t, err)
	eq(t, md, md3)
}

func TestLoadFileDescriptorWithDeps(t *testing.T) {
	// Try one with some imports
	fd, err := LoadFileDescriptor("desc_test2.proto")
	ok(t, err)
	eq(t, "desc_test2.proto", fd.GetName())
	eq(t, "desc_test2.proto", fd.GetFullyQualifiedName())
	eq(t, "desc_test", fd.GetPackage())

	deps := fd.GetDependencies()
	eq(t, 3, len(deps))
	eq(t, "desc_test1.proto", deps[0].GetName())
	eq(t, "pkg/desc_test_pkg.proto", deps[1].GetName())
	eq(t, "nopkg/desc_test_nopkg.proto", deps[2].GetName())

	// loading the dependencies yields same descriptor objects
	fd, err = LoadFileDescriptor("desc_test1.proto")
	ok(t, err)
	eq(t, deps[0], fd)
	fd, err = LoadFileDescriptor("pkg/desc_test_pkg.proto")
	ok(t, err)
	eq(t, deps[1], fd)
	fd, err = LoadFileDescriptor("nopkg/desc_test_nopkg.proto")
	ok(t, err)
	eq(t, deps[2], fd)
}
