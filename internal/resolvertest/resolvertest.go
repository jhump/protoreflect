// Package resolvertest provides re-usable test suites for implementations of
// the resolver interfaces in the protoresolve package.
//
// Each suite is given a corpus of files that the resolver under test is
// expected to know about. The expected results of all queries are computed
// from that corpus, so any implementation can be checked as long as it can
// be populated with the same files. The standard corpus, from Corpus, is
// made of files generated into internal/testprotos, so it can be used even
// with resolvers backed by the global registries.
package resolvertest

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	"github.com/jhump/protoreflect/v2/protoresolve"
)

// Option customizes the behavior of the checks in this package.
type Option func(*config)

// WithExtraFiles indicates that the resolver under test may know about files
// that are not in the corpus, such as one backed by the global registries.
// Checks that enumerate elements will only verify that all elements in the
// corpus are present, ignoring any others.
func WithExtraFiles() Option {
	return func(cfg *config) {
		cfg.allowExtraFiles = true
	}
}

// WithLenientErrors indicates that the resolver under test may not return a
// *protoresolve.ErrUnexpectedType when a query resolves an element of the
// wrong kind. This is the case for resolvers implemented outside of this
// module, like *protoregistry.Types. With this option, such queries need only
// return an error.
func WithLenientErrors() Option {
	return func(cfg *config) {
		cfg.lenientErrors = true
	}
}

type config struct {
	allowExtraFiles bool
	lenientErrors   bool
}

func newConfig(opts []Option) *config {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// Corpus returns the standard corpus of files, for populating a resolver to
// test. These files are generated into internal/testprotos, so they are also
// present in protoregistry.GlobalFiles and protoregistry.GlobalTypes.
//
// The returned files are in topological order: every file appears after all
// of its imports.
func Corpus() []protoreflect.FileDescriptor {
	return transitiveClosure([]protoreflect.FileDescriptor{
		testprotos.File_desc_test1_proto,
		testprotos.File_desc_test2_proto,
		testprotos.File_desc_test_complex_proto,
		testprotos.File_desc_test_proto3_proto,
		testprotos.File_desc_test_editions_proto,
	})
}

// CheckTypeResolver verifies that the given resolver can resolve all types
// in the given corpus. It also verifies the errors returned for elements
// that are not found or that are the wrong kind.
//
// If the given resolver also has a RangeExtensionsByMessage method, like
// protoresolve.TypePool, that method is checked, too.
func CheckTypeResolver(t *testing.T, res protoresolve.TypeResolver, corpus []protoreflect.FileDescriptor, opts ...Option) {
	t.Helper()
	cfg := newConfig(opts)
	index := newCorpusIndex(corpus)
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
			checkRangeExtensionTypesByMessage(t, cfg, ranger, index)
		})
	}
}

// CheckTypePool is like CheckTypeResolver, but also verifies that the methods
// for enumerating types yield all types in the given corpus.
func CheckTypePool(t *testing.T, pool protoresolve.TypePool, corpus []protoreflect.FileDescriptor, opts ...Option) {
	t.Helper()
	CheckTypeResolver(t, pool, corpus, opts...)
	cfg := newConfig(opts)
	index := newCorpusIndex(corpus)
	t.Run("RangeMessages", func(t *testing.T) {
		checkRange(t, cfg, index, pool.RangeMessages,
			func(mt protoreflect.MessageType) protoreflect.Descriptor { return mt.Descriptor() },
			names(index.typeMessages()))
	})
	t.Run("RangeEnums", func(t *testing.T) {
		checkRange(t, cfg, index, pool.RangeEnums,
			func(et protoreflect.EnumType) protoreflect.Descriptor { return et.Descriptor() },
			names(index.enums))
	})
	t.Run("RangeExtensions", func(t *testing.T) {
		checkRange(t, cfg, index, pool.RangeExtensions,
			func(xt protoreflect.ExtensionType) protoreflect.Descriptor { return xt.TypeDescriptor() },
			names(index.extensions))
	})
}
