package resolvertest

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

const (
	unknownName = protoreflect.FullName("does.not.Exist")
	// The maximum field number, which no element in the corpus uses.
	unusedFieldNumber = protoreflect.FieldNumber(536870911)
)

func typeURL(name protoreflect.FullName) string {
	return "type.googleapis.com/" + string(name)
}

// descriptorKey returns a key that uniquely identifies the given descriptor:
// its path if it is a file, otherwise its full name.
func descriptorKey(descriptor protoreflect.Descriptor) string {
	if file, ok := descriptor.(protoreflect.FileDescriptor); ok {
		return file.Path()
	}
	return string(descriptor.FullName())
}

func keys[D protoreflect.Descriptor](descriptors []D) []string {
	result := make([]string, len(descriptors))
	for i, descriptor := range descriptors {
		result[i] = descriptorKey(descriptor)
	}
	return result
}

// checkUnexpectedType verifies that err indicates that a query found an
// element of the wrong kind. Only one of name or url should be non-empty,
// depending on whether the query was by name or by URL.
func checkUnexpectedType(
	t *testing.T,
	lenientErrors bool,
	err error,
	expecting protoresolve.DescriptorKind,
	actual protoresolve.DescriptorKind,
	name protoreflect.FullName,
	url string,
) {
	t.Helper()
	query := string(name) + url
	if !assert.Error(t, err, "query for %s %q should fail", expecting, query) {
		return
	}
	if lenientErrors {
		return
	}
	var unexpectedTypeErr *protoresolve.ErrUnexpectedType
	if !assert.ErrorAs(t, err, &unexpectedTypeErr, "query for %s %q", expecting, query) {
		return
	}
	assert.Equal(t, expecting.String(), unexpectedTypeErr.Expecting.String(), "query for %s %q", expecting, query)
	assert.Equal(t, actual.String(), unexpectedTypeErr.Actual.String(), "query for %s %q", expecting, query)
	assert.Equal(t, name, unexpectedTypeErr.Name, "query for %s %q", expecting, query)
	assert.Equal(t, url, unexpectedTypeErr.URL, "query for %s %q", expecting, query)
	if unexpectedTypeErr.Descriptor != nil {
		assert.Equal(t, actual.String(), protoresolve.KindOf(unexpectedTypeErr.Descriptor).String(), "query for %s %q", expecting, query)
	}
}

// checkNotFound verifies that err is exactly protoresolve.ErrNotFound, which is
// protoregistry.NotFound. An error that merely wraps it is not enough: the
// protobuf runtime, which uses resolvers when unmarshalling and in protodesc,
// requires resolvers to return that exact value (see the doc comment for
// protoregistry.NotFound) and compares errors to it with ==.
func checkNotFound(t *testing.T, err error, query string) {
	t.Helper()
	assert.Same(t, protoresolve.ErrNotFound, err, "query for %q should return exactly ErrNotFound", query)
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

// checkRange verifies that the given range function yields exactly the
// expected elements, each one only once, and that it stops when the callback
// returns false. Elements are identified using descriptorKey. Map entry
// messages are ignored, so the expected elements need not include them,
// regardless of whether the range function yields them.
//
// If cfg.allowExtraFiles is set, elements that are not in the corpus are also
// ignored.
func checkRange[T any](
	t *testing.T,
	cfg *config,
	index *corpusIndex,
	rangeFn func(func(T) bool),
	descriptorOf func(T) protoreflect.Descriptor,
	expected []string,
) {
	t.Helper()
	counts := map[string]int{}
	rangeFn(func(elem T) bool {
		descriptor := descriptorOf(elem)
		if msg, ok := descriptor.(protoreflect.MessageDescriptor); ok && msg.IsMapEntry() {
			return true
		}
		if cfg.allowExtraFiles && !index.inCorpus(descriptor) {
			return true
		}
		counts[descriptorKey(descriptor)]++
		return true
	})
	for key, count := range counts {
		assert.Equal(t, 1, count, "%s should be yielded exactly once", key)
	}
	assert.ElementsMatch(t, expected, slices.Collect(maps.Keys(counts)))

	if len(expected) == 0 {
		return
	}
	var numCalls int
	rangeFn(func(T) bool {
		numCalls++
		return false
	})
	assert.Equal(t, 1, numCalls, "iteration should stop when callback returns false")
}

// checkRangeExtensionsByMessage verifies a RangeExtensionsByMessage method,
// which yields elements of type T, which may be extension descriptors or
// extension types.
func checkRangeExtensionsByMessage[T any](
	t *testing.T,
	cfg *config,
	index *corpusIndex,
	rangeByMessage func(protoreflect.FullName, func(T) bool),
	extensionOf func(T) protoreflect.ExtensionDescriptor,
) {
	checkExtendee := func(t *testing.T, extendee protoreflect.FullName, expected []string) {
		rangeFn := func(fn func(T) bool) {
			rangeByMessage(extendee, func(elem T) bool {
				assert.Equal(t, extendee, extensionOf(elem).ContainingMessage().FullName())
				return fn(elem)
			})
		}
		checkRange(t, cfg, index, rangeFn,
			func(elem T) protoreflect.Descriptor { return extensionOf(elem) },
			expected)
	}
	for extendee, exts := range index.extensionsByMessage {
		t.Run(string(extendee), func(t *testing.T) {
			checkExtendee(t, extendee, keys(exts))
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
