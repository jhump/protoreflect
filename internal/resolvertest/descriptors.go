package resolvertest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

func checkResolver(t *testing.T, cfg *config, res protoresolve.Resolver, index *corpusIndex) {
	t.Run("FindFileByPath", func(t *testing.T) {
		checkFindFiles(t, res, index)
	})
	t.Run("RangeFiles", func(t *testing.T) {
		checkRangeFiles(t, cfg, res, index)
	})
	t.Run("RangeFilesByPackage", func(t *testing.T) {
		checkRangeFilesByPackage(t, cfg, res, index)
	})
	t.Run("FindDescriptorByName", func(t *testing.T) {
		checkFindDescriptors(t, res, index)
	})
	t.Run("FindMessage", func(t *testing.T) {
		checkFindMessages(t, cfg, res, index)
	})
	t.Run("FindExtension", func(t *testing.T) {
		checkFindExtensions(t, cfg, res, index)
	})
	t.Run("RangeExtensionsByMessage", func(t *testing.T) {
		checkRangeExtensionsByMessage(t, cfg, index, res.RangeExtensionsByMessage,
			func(ext protoreflect.ExtensionDescriptor) protoreflect.ExtensionDescriptor { return ext })
	})

	typeCfg := *cfg
	typeCfg.lenientErrors = cfg.lenientErrors || cfg.lenientTypeErrors
	t.Run("AsTypeResolver", func(t *testing.T) {
		typeRes := res.AsTypeResolver()
		if pool, ok := typeRes.(protoresolve.TypePool); ok {
			checkTypePool(t, &typeCfg, pool, index)
		} else {
			checkTypeResolver(t, &typeCfg, typeRes, index)
		}
	})
	if poolRes, ok := res.(interface{ AsTypePool() protoresolve.TypePool }); ok {
		t.Run("AsTypePool", func(t *testing.T) {
			checkTypePool(t, &typeCfg, poolRes.AsTypePool(), index)
		})
	}
}

func checkFindFiles(t *testing.T, res protoresolve.FileResolver, index *corpusIndex) {
	for _, file := range index.files {
		found, err := res.FindFileByPath(file.Path())
		if assert.NoError(t, err, "file %s", file.Path()) {
			checkDescriptorMatches(t, file, found)
		}
	}
	_, err := res.FindFileByPath("does/not/exist.proto")
	checkNotFound(t, err, "does/not/exist.proto")
}

func checkFileCount(t *testing.T, cfg *config, expected, actual int, query string) {
	t.Helper()
	switch {
	case cfg.inexactFileCounts:
		return
	case cfg.allowExtraFiles:
		assert.GreaterOrEqual(t, actual, expected, "number of files for %s", query)
	default:
		assert.Equal(t, expected, actual, "number of files for %s", query)
	}
}

func checkRangeFiles(t *testing.T, cfg *config, res protoresolve.FilePool, index *corpusIndex) {
	checkFileCount(t, cfg, len(index.files), res.NumFiles(), "all files")
	checkRange(t, cfg, index, res.RangeFiles,
		func(file protoreflect.FileDescriptor) protoreflect.Descriptor { return file },
		keys(index.files))
}

func checkRangeFilesByPackage(t *testing.T, cfg *config, res protoresolve.FilePool, index *corpusIndex) {
	checkPackage := func(t *testing.T, pkg protoreflect.FullName, expected []protoreflect.FileDescriptor) {
		checkFileCount(t, cfg, len(expected), res.NumFilesByPackage(pkg), string(pkg))
		rangeFn := func(fn func(protoreflect.FileDescriptor) bool) {
			res.RangeFilesByPackage(pkg, func(file protoreflect.FileDescriptor) bool {
				assert.Equal(t, pkg, file.Package())
				return fn(file)
			})
		}
		checkRange(t, cfg, index, rangeFn,
			func(file protoreflect.FileDescriptor) protoreflect.Descriptor { return file },
			keys(expected))
	}
	// Packages that enclose other packages, but contain no files themselves.
	emptyPackages := map[protoreflect.FullName]struct{}{unknownName: {}}
	for pkg, files := range index.filesByPackage {
		if pkg == "" {
			continue
		}
		t.Run(string(pkg), func(t *testing.T) {
			checkPackage(t, pkg, files)
		})
		for parent := pkg.Parent(); parent != ""; parent = parent.Parent() {
			if _, ok := index.filesByPackage[parent]; !ok {
				emptyPackages[parent] = struct{}{}
			}
		}
	}
	for pkg := range emptyPackages {
		t.Run(string(pkg), func(t *testing.T) {
			checkPackage(t, pkg, nil)
		})
	}
}

func checkFindDescriptors(t *testing.T, res protoresolve.DescriptorResolver, index *corpusIndex) {
	for _, descriptor := range index.all {
		found, err := res.FindDescriptorByName(descriptor.FullName())
		if assert.NoError(t, err, "descriptor %s", descriptor.FullName()) {
			checkDescriptorMatches(t, descriptor, found)
		}
	}
	_, err := res.FindDescriptorByName(unknownName)
	checkNotFound(t, err, string(unknownName))
}

func checkFindMessages(t *testing.T, cfg *config, res protoresolve.MessageResolver, index *corpusIndex) {
	for _, msg := range index.messages {
		name := msg.FullName()
		found, err := res.FindMessageByName(name)
		if assert.NoError(t, err, "message %s", name) {
			checkDescriptorMatches(t, msg, found)
		}
		for _, url := range []string{typeURL(name), "example.com/foo/bar/" + string(name), string(name)} {
			found, err := res.FindMessageByURL(url)
			if assert.NoError(t, err, "message URL %s", url) {
				checkDescriptorMatches(t, msg, found)
			}
		}
	}

	require.NotEmpty(t, index.enums)
	require.NotEmpty(t, index.extensions)
	require.NotEmpty(t, index.fields)
	enum, ext, field := index.enums[0], index.extensions[0], index.fields[0]
	_, err := res.FindMessageByName(enum.FullName())
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindEnum, enum.FullName(), "")
	_, err = res.FindMessageByURL(typeURL(enum.FullName()))
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindEnum, "", typeURL(enum.FullName()))
	_, err = res.FindMessageByName(ext.FullName())
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindExtension, ext.FullName(), "")
	_, err = res.FindMessageByName(field.FullName())
	checkUnexpectedType(t, cfg.lenientErrors, err, protoresolve.DescriptorKindMessage, protoresolve.DescriptorKindField, field.FullName(), "")

	_, err = res.FindMessageByName(unknownName)
	checkNotFound(t, err, string(unknownName))
	_, err = res.FindMessageByURL(typeURL(unknownName))
	checkNotFound(t, err, typeURL(unknownName))
}

func checkFindExtensions(t *testing.T, cfg *config, res protoresolve.ExtensionResolver, index *corpusIndex) {
	for _, ext := range index.extensions {
		name := ext.FullName()
		found, err := res.FindExtensionByName(name)
		if assert.NoError(t, err, "extension %s", name) {
			checkExtensionMatches(t, ext, found)
		}
		extendee := ext.ContainingMessage().FullName()
		found, err = res.FindExtensionByNumber(extendee, ext.Number())
		if assert.NoError(t, err, "extension %s:%d", extendee, ext.Number()) {
			checkExtensionMatches(t, ext, found)
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
