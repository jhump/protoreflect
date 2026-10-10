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
	"google.golang.org/protobuf/types/descriptorpb"

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

// WithLenientTypeErrors is like WithLenientErrors, but only applies to the
// resolver returned from the AsTypeResolver (or AsTypePool) method of the
// resolver under test. This is for resolvers that use a type resolver
// implemented outside of this module, like *protoregistry.Types.
func WithLenientTypeErrors() Option {
	return func(cfg *config) {
		cfg.lenientTypeErrors = true
	}
}

// WithInexactFileCounts indicates that the NumFiles and NumFilesByPackage
// methods of the resolver under test are not expected to be accurate, so
// they will not be checked.
func WithInexactFileCounts() Option {
	return func(cfg *config) {
		cfg.inexactFileCounts = true
	}
}

type config struct {
	allowExtraFiles   bool
	lenientErrors     bool
	lenientTypeErrors bool
	inexactFileCounts bool
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

// ProtoFileOracleFunc adapts a function to the protoresolve.ProtoFileOracle
// interface.
type ProtoFileOracleFunc func(protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error)

// ProtoFromFileDescriptor implements protoresolve.ProtoFileOracle.
func (f ProtoFileOracleFunc) ProtoFromFileDescriptor(file protoreflect.FileDescriptor) (*descriptorpb.FileDescriptorProto, error) {
	return f(file)
}

// Descriptors returns all descriptors in the given files, other than the
// file descriptors themselves.
func Descriptors(files []protoreflect.FileDescriptor) []protoreflect.Descriptor {
	return newCorpusIndex(files).all
}

// CheckResolver verifies that the given resolver can resolve all files and
// descriptors in the given corpus, and that it can enumerate all files and
// extensions in the corpus. It also verifies the errors returned for elements
// that are not found or that are the wrong kind.
//
// The resolver returned by the AsTypeResolver method is checked using
// CheckTypePool, if it implements protoresolve.TypePool, or otherwise using
// CheckTypeResolver. If the given resolver also has an AsTypePool method,
// the pool it returns is also checked.
func CheckResolver(t *testing.T, res protoresolve.Resolver, corpus []protoreflect.FileDescriptor, opts ...Option) {
	t.Helper()
	checkResolver(t, newConfig(opts), res, newCorpusIndex(corpus))
}

// CheckTypeResolver verifies that the given resolver can resolve all types
// in the given corpus. It also verifies the errors returned for elements
// that are not found or that are the wrong kind.
//
// If the given resolver also has a RangeExtensionsByMessage method, like
// protoresolve.TypePool, that method is checked, too.
func CheckTypeResolver(t *testing.T, res protoresolve.TypeResolver, corpus []protoreflect.FileDescriptor, opts ...Option) {
	t.Helper()
	checkTypeResolver(t, newConfig(opts), res, newCorpusIndex(corpus))
}

// CheckTypePool is like CheckTypeResolver, but also verifies that the methods
// for enumerating types yield all types in the given corpus.
func CheckTypePool(t *testing.T, pool protoresolve.TypePool, corpus []protoreflect.FileDescriptor, opts ...Option) {
	t.Helper()
	checkTypePool(t, newConfig(opts), pool, newCorpusIndex(corpus))
}
