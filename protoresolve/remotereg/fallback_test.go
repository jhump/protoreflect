package remotereg_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoregistry"

	"github.com/jhump/protoreflect/v2/internal/testprotos"
	. "github.com/jhump/protoreflect/v2/protoresolve/remotereg"
)

func TestFetchFallsBackAndCaches(t *testing.T) {
	t.Parallel()
	var fallback protoregistry.Files
	require.NoError(t, fallback.RegisterFile(testprotos.File_desc_test1_proto))
	var numFetches atomic.Int32
	reg := &Registry{
		// The fetcher has no types, so all types come from the fallback.
		TypeFetcher: TypeFetcherFunc(func(context.Context, string, bool) (proto.Message, error) {
			numFetches.Add(1)
			return nil, protoregistry.NotFound
		}),
		Fallback: &fallback,
	}
	const url = "https://example.com/testprotos.TestMessage"
	expected := testprotos.File_desc_test1_proto.Messages().ByName("TestMessage")

	for range 3 {
		msg, err := reg.FindMessageByURL(url)
		require.NoError(t, err)
		assert.Same(t, expected, msg, "should be the fallback's descriptor")
	}
	// The result from the fallback was cached, so the fetcher was only
	// consulted once.
	assert.Equal(t, int32(1), numFetches.Load())
	assert.Equal(t, url, reg.URLForType(expected))

	enumDesc := testprotos.File_desc_test1_proto.Enums().ByName("SomeEnum")
	en, err := reg.FindEnumByURL("https://example.com/testprotos.SomeEnum")
	require.NoError(t, err)
	assert.Same(t, enumDesc, en)

	// Not found anywhere.
	_, err = reg.FindMessageByURL("https://example.com/testprotos.Unknown")
	assert.ErrorIs(t, err, protoregistry.NotFound)
}
