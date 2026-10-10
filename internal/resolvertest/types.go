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

func checkTypeResolver(t *testing.T, cfg *config, res protoresolve.TypeResolver, index *corpusIndex) {
	t.Run("FindMessage", func(t *testing.T) {
		checkFindMessageTypes(t, cfg, res, index)
	})
	t.Run("FindEnumByName", func(t *testing.T) {
		checkFindEnumTypes(t, cfg, res, index)
	})
	t.Run("FindExtension", func(t *testing.T) {
		checkFindExtensionTypes(t, cfg, res, index)
	})
	if ranger, ok := res.(extensionTypeRanger); ok {
		t.Run("RangeExtensionsByMessage", func(t *testing.T) {
			checkRangeExtensionsByMessage(t, cfg, index, ranger.RangeExtensionsByMessage,
				func(extType protoreflect.ExtensionType) protoreflect.ExtensionDescriptor {
					return extType.TypeDescriptor()
				})
		})
	}
}

func checkTypePool(t *testing.T, cfg *config, pool protoresolve.TypePool, index *corpusIndex) {
	checkTypeResolver(t, cfg, pool, index)
	t.Run("RangeMessages", func(t *testing.T) {
		checkRange(t, cfg, index, pool.RangeMessages,
			func(msgType protoreflect.MessageType) protoreflect.Descriptor { return msgType.Descriptor() },
			keys(index.typeMessages()))
	})
	t.Run("RangeEnums", func(t *testing.T) {
		checkRange(t, cfg, index, pool.RangeEnums,
			func(enumType protoreflect.EnumType) protoreflect.Descriptor { return enumType.Descriptor() },
			keys(index.enums))
	})
	t.Run("RangeExtensions", func(t *testing.T) {
		checkRange(t, cfg, index, pool.RangeExtensions,
			func(extType protoreflect.ExtensionType) protoreflect.Descriptor { return extType.TypeDescriptor() },
			keys(index.extensions))
	})
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
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindEnum, enum.FullName(), "")
	_, err = res.FindMessageByURL(typeURL(enum.FullName()))
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindEnum, "", typeURL(enum.FullName()))
	_, err = res.FindMessageByName(ext.FullName())
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindExtension, ext.FullName(), "")

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
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindEnum, protoresolve.DescriptorKindMessage, msg.FullName(), "")
	_, err = res.FindEnumByName(enumVal.FullName())
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindEnum, protoresolve.DescriptorKindEnumValue, enumVal.FullName(), "")

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
	checkFindExtensionErrors(t, cfg, index,
		func(name protoreflect.FullName) error {
			_, err := res.FindExtensionByName(name)
			return err
		},
		func(message protoreflect.FullName, number protoreflect.FieldNumber) error {
			_, err := res.FindExtensionByNumber(message, number)
			return err
		})
}

// checkFindExtensionErrors verifies the errors returned from queries for
// extensions, which can be used with both descriptor and type resolvers.
func checkFindExtensionErrors(
	t *testing.T,
	cfg *config,
	index *corpusIndex,
	findByName func(protoreflect.FullName) error,
	findByNumber func(protoreflect.FullName, protoreflect.FieldNumber) error,
) {
	require.NotEmpty(t, index.messages)
	require.NotEmpty(t, index.fields)
	require.NotEmpty(t, index.extensions)
	msg, field, ext := index.messages[0], index.fields[0], index.extensions[0]
	err := findByName(msg.FullName())
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindExtension, protoresolve.DescriptorKindMessage, msg.FullName(), "")
	err = findByName(field.FullName())
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindExtension, protoresolve.DescriptorKindField, field.FullName(), "")

	err = findByName(unknownName)
	checkNotFound(t, err, string(unknownName))
	extendee := ext.ContainingMessage().FullName()
	err = findByNumber(extendee, unusedFieldNumber)
	checkNotFound(t, err, string(extendee))
	err = findByNumber(unknownName, 1)
	checkNotFound(t, err, string(unknownName))
	// A normal field's number is not an extension.
	err = findByNumber(field.ContainingMessage().FullName(), field.Number())
	checkNotFound(t, err, string(field.FullName()))
}
