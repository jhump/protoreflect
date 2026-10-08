package resolvertest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

type extensionTypeRanger interface {
	RangeExtensionsByMessage(message protoreflect.FullName, fn func(protoreflect.ExtensionType) bool)
}

func checkFindMessageTypes(t *testing.T, cfg *config, res protoresolve.MessageTypeResolver, index *corpusIndex) {
	for _, msg := range index.typeMessages() {
		name := msg.FullName()
		msgType, err := res.FindMessageByName(name)
		if assert.NoError(t, err, "message %s", name) {
			checkDescriptorMatches(t, msg, msgType.Descriptor())
			assert.Equal(t, name, msgType.New().Descriptor().FullName())
		}
		for _, url := range []string{typeURL(name), "example.com/foo/bar/" + string(name), string(name)} {
			msgType, err := res.FindMessageByURL(url)
			if assert.NoError(t, err, "message URL %s", url) {
				checkDescriptorMatches(t, msg, msgType.Descriptor())
			}
		}
	}

	require.NotEmpty(t, index.enums)
	require.NotEmpty(t, index.extensions)
	enum, ext := index.enums[0], index.extensions[0]
	_, err := res.FindMessageByName(enum.FullName())
	checkUnexpectedType(t, cfg, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindEnum, enum.FullName(), "")
	_, err = res.FindMessageByURL(typeURL(enum.FullName()))
	checkUnexpectedType(t, cfg, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindEnum, "", typeURL(enum.FullName()))
	_, err = res.FindMessageByName(ext.FullName())
	checkUnexpectedType(t, cfg, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindExtension, ext.FullName(), "")

	_, err = res.FindMessageByName(unknownName)
	checkNotFound(t, err, string(unknownName))
	_, err = res.FindMessageByURL(typeURL(unknownName))
	checkNotFound(t, err, typeURL(unknownName))
}

func checkFindEnumTypes(t *testing.T, cfg *config, res protoresolve.EnumTypeResolver, index *corpusIndex) {
	for _, enum := range index.enums {
		name := enum.FullName()
		enumType, err := res.FindEnumByName(name)
		if assert.NoError(t, err, "enum %s", name) {
			checkDescriptorMatches(t, enum, enumType.Descriptor())
			assert.Equal(t, name, enumType.New(enum.Values().Get(0).Number()).Descriptor().FullName())
		}
	}

	require.NotEmpty(t, index.enums)
	require.NotEmpty(t, index.messages)
	msg, enumVal := index.messages[0], index.enums[0].Values().Get(0)
	_, err := res.FindEnumByName(msg.FullName())
	checkUnexpectedType(t, cfg, err, protoresolve.DescriptorKindEnum, protoresolve.DescriptorKindMessage, msg.FullName(), "")
	_, err = res.FindEnumByName(enumVal.FullName())
	checkUnexpectedType(t, cfg, err, protoresolve.DescriptorKindEnum, protoresolve.DescriptorKindEnumValue, enumVal.FullName(), "")

	_, err = res.FindEnumByName(unknownName)
	checkNotFound(t, err, string(unknownName))
}

func checkFindExtensionTypes(t *testing.T, cfg *config, res protoresolve.ExtensionTypeResolver, index *corpusIndex) {
	for _, ext := range index.extensions {
		name := ext.FullName()
		extType, err := res.FindExtensionByName(name)
		if assert.NoError(t, err, "extension %s", name) {
			checkExtensionMatches(t, ext, extType.TypeDescriptor())
		}
		extendee := ext.ContainingMessage().FullName()
		extType, err = res.FindExtensionByNumber(extendee, ext.Number())
		if assert.NoError(t, err, "extension %s:%d", extendee, ext.Number()) {
			checkExtensionMatches(t, ext, extType.TypeDescriptor())
		}
	}

	require.NotEmpty(t, index.messages)
	require.NotEmpty(t, index.fields)
	require.NotEmpty(t, index.extensions)
	msg, field, ext := index.messages[0], index.fields[0], index.extensions[0]
	_, err := res.FindExtensionByName(msg.FullName())
	checkUnexpectedType(t, cfg, err, protoresolve.DescriptorKindExtension, protoresolve.DescriptorKindMessage, msg.FullName(), "")
	_, err = res.FindExtensionByName(field.FullName())
	checkUnexpectedType(t, cfg, err, protoresolve.DescriptorKindExtension, protoresolve.DescriptorKindField, field.FullName(), "")

	_, err = res.FindExtensionByName(unknownName)
	checkNotFound(t, err, string(unknownName))
	extendee := ext.ContainingMessage().FullName()
	_, err = res.FindExtensionByNumber(extendee, unusedFieldNumber)
	checkNotFound(t, err, string(extendee))
	_, err = res.FindExtensionByNumber(unknownName, 1)
	checkNotFound(t, err, string(unknownName))
	// A normal field's number is not an extension.
	_, err = res.FindExtensionByNumber(field.ContainingMessage().FullName(), field.Number())
	checkNotFound(t, err, string(field.FullName()))
}

func checkRangeExtensionTypesByMessage(t *testing.T, cfg *config, ranger extensionTypeRanger, index *corpusIndex) {
	checkExtendee := func(t *testing.T, extendee protoreflect.FullName, expected []protoreflect.FullName) {
		rangeFn := func(fn func(protoreflect.ExtensionType) bool) {
			ranger.RangeExtensionsByMessage(extendee, func(extType protoreflect.ExtensionType) bool {
				assert.Equal(t, extendee, extType.TypeDescriptor().ContainingMessage().FullName())
				return fn(extType)
			})
		}
		checkRange(t, cfg, index, rangeFn,
			func(extType protoreflect.ExtensionType) protoreflect.Descriptor { return extType.TypeDescriptor() },
			expected)
	}
	for extendee, exts := range index.extensionsByMessage {
		t.Run(string(extendee), func(t *testing.T) {
			checkExtendee(t, extendee, names(exts))
		})
	}
	for _, msg := range index.typeMessages() {
		if _, ok := index.extensionsByMessage[msg.FullName()]; !ok {
			t.Run("no extensions", func(t *testing.T) {
				checkExtendee(t, msg.FullName(), nil)
			})
			break
		}
	}
	t.Run("unknown message", func(t *testing.T) {
		checkExtendee(t, unknownName, nil)
	})
}

func checkDescriptorMatches(t *testing.T, expected, actual protoreflect.Descriptor) {
	t.Helper()
	assert.Equal(t, expected.FullName(), actual.FullName())
	assert.Equal(t, protoresolve.KindOf(expected).String(), protoresolve.KindOf(actual).String(), "kind of %s", expected.FullName())
	assert.Equal(t, expected.ParentFile().Path(), actual.ParentFile().Path(), "file of %s", expected.FullName())
}

func checkExtensionMatches(t *testing.T, expected, actual protoreflect.ExtensionDescriptor) {
	t.Helper()
	checkDescriptorMatches(t, expected, actual)
	assert.Equal(t, expected.Number(), actual.Number(), "number of %s", expected.FullName())
	assert.Equal(t, expected.ContainingMessage().FullName(), actual.ContainingMessage().FullName(), "extendee of %s", expected.FullName())
}
