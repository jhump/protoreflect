package protoresolve_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestNewNotFoundError(t *testing.T) {
	t.Parallel()
	err := protoresolve.NewNotFoundError(protoreflect.FullName("foo.Bar"))
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	assert.ErrorContains(t, err, "foo.Bar")
	err = protoresolve.NewNotFoundError("foo/bar.proto")
	assert.ErrorIs(t, err, protoresolve.ErrNotFound)
	assert.ErrorContains(t, err, "foo/bar.proto")
}

func TestNewUnexpectedTypeError(t *testing.T) {
	t.Parallel()
	enum := testprotos.File_desc_test1_proto.Enums().ByName("SomeEnum")
	err := protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindMessage, enum, "")
	assert.Equal(t, &protoresolve.ErrUnexpectedType{
		Name:       "testprotos.SomeEnum",
		Expecting:  protoresolve.DescriptorKindMessage,
		Actual:     protoresolve.DescriptorKindEnum,
		Descriptor: enum,
	}, err)
	assert.EqualError(t, err, `wrong kind of descriptor for name "testprotos.SomeEnum": expected a message, got an enum`)

	url := "type.googleapis.com/testprotos.SomeEnum"
	err = protoresolve.NewUnexpectedTypeError(protoresolve.DescriptorKindMessage, enum, url)
	assert.Equal(t, &protoresolve.ErrUnexpectedType{
		URL:        url,
		Expecting:  protoresolve.DescriptorKindMessage,
		Actual:     protoresolve.DescriptorKindEnum,
		Descriptor: enum,
	}, err)
	assert.EqualError(t, err, `wrong kind of descriptor for URL "type.googleapis.com/testprotos.SomeEnum": expected a message, got an enum`)
}

func TestErrUnexpectedTypeError(t *testing.T) {
	t.Parallel()
	articles := map[protoresolve.DescriptorKind]string{
		protoresolve.DescriptorKindUnknown:   "unknown",
		protoresolve.DescriptorKindFile:      "a file",
		protoresolve.DescriptorKindMessage:   "a message",
		protoresolve.DescriptorKindField:     "a field",
		protoresolve.DescriptorKindOneof:     "a oneof",
		protoresolve.DescriptorKindEnum:      "an enum",
		protoresolve.DescriptorKindEnumValue: "an enum value",
		protoresolve.DescriptorKindExtension: "an extension",
		protoresolve.DescriptorKindService:   "a service",
		protoresolve.DescriptorKindMethod:    "a method",
		protoresolve.DescriptorKind(100):     "unknown kind (100)",
	}
	for kind, article := range articles {
		err := &protoresolve.ErrUnexpectedType{Name: "foo.Bar", Expecting: kind, Actual: kind}
		expected := fmt.Sprintf(`wrong kind of descriptor for name "foo.Bar": expected %s, got %s`, article, article)
		assert.EqualError(t, err, expected)
	}
}
