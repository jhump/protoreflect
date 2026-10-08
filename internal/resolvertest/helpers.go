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

func names[D protoreflect.Descriptor](descriptors []D) []protoreflect.FullName {
	result := make([]protoreflect.FullName, len(descriptors))
	for i, descriptor := range descriptors {
		result[i] = descriptor.FullName()
	}
	return result
}

// checkUnexpectedType verifies that err indicates that a query found an
// element of the wrong kind. Only one of name or url should be non-empty,
// depending on whether the query was by name or by URL.
func checkUnexpectedType(
	t *testing.T,
	cfg *config,
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
	if cfg.lenientErrors {
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

func checkNotFound(t *testing.T, err error, query string) {
	t.Helper()
	assert.ErrorIs(t, err, protoresolve.ErrNotFound, "query for %q", query)
}

// checkRange verifies that the given range function yields exactly the
// expected elements, each one only once, and that it stops when the callback
// returns false. Map entry messages are ignored, so the expected elements need
// not include them, regardless of whether the range function yields them.
//
// If cfg.allowExtraFiles is set, elements that are not in the corpus are also
// ignored.
func checkRange[T any](
	t *testing.T,
	cfg *config,
	index *corpusIndex,
	rangeFn func(func(T) bool),
	descriptorOf func(T) protoreflect.Descriptor,
	expected []protoreflect.FullName,
) {
	t.Helper()
	counts := map[protoreflect.FullName]int{}
	rangeFn(func(elem T) bool {
		descriptor := descriptorOf(elem)
		if msg, ok := descriptor.(protoreflect.MessageDescriptor); ok && msg.IsMapEntry() {
			return true
		}
		if cfg.allowExtraFiles && !index.inCorpus(descriptor) {
			return true
		}
		counts[descriptor.FullName()]++
		return true
	})
	for name, count := range counts {
		assert.Equal(t, 1, count, "%s should be yielded exactly once", name)
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
