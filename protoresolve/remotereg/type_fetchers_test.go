package remotereg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/sourcecontextpb"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/jhump/protoreflect/v2/protoresolve"
)

func TestCachingTypeFetcher(t *testing.T) {
	counts := map[string]int{}
	uncached := TypeFetcherFunc(func(ctx context.Context, url string, enum bool) (proto.Message, error) {
		counts[url] = counts[url] + 1
		return testFetcher(ctx, url, enum)
	})

	// observe the underlying type fetcher get invoked 10x
	for i := 0; i < 10; i++ {
		typ, err := uncached.FetchMessageType(context.Background(), "blah.blah.blah/fee.fi.fo.Fum")
		require.NoError(t, err)
		require.Equal(t, "fee.fi.fo.Fum", typ.Name)
	}
	for i := 0; i < 10; i++ {
		en, err := uncached.FetchEnumType(context.Background(), "blah.blah.blah/fee.fi.fo.Foo")
		require.NoError(t, err)
		require.Equal(t, "fee.fi.fo.Foo", en.Name)
	}

	require.Equal(t, 10, counts["blah.blah.blah/fee.fi.fo.Fum"])
	require.Equal(t, 10, counts["blah.blah.blah/fee.fi.fo.Foo"])

	// now we'll see the underlying fetcher invoked just one more time,
	// after which the result is cached
	cached := CachingTypeFetcher(uncached)

	for i := 0; i < 10; i++ {
		typ, err := cached.FetchMessageType(context.Background(), "blah.blah.blah/fee.fi.fo.Fum")
		require.NoError(t, err)
		require.Equal(t, "fee.fi.fo.Fum", typ.Name)
	}

	for i := 0; i < 10; i++ {
		en, err := cached.FetchEnumType(context.Background(), "blah.blah.blah/fee.fi.fo.Foo")
		require.NoError(t, err)
		require.Equal(t, "fee.fi.fo.Foo", en.Name)
	}

	require.Equal(t, 11, counts["blah.blah.blah/fee.fi.fo.Fum"])
	require.Equal(t, 11, counts["blah.blah.blah/fee.fi.fo.Foo"])
}

func TestCachingTypeFetcher_MismatchType(t *testing.T) {
	fetcher := CachingTypeFetcher(TypeFetcherFunc(testFetcher))
	// get a message type
	typ, err := fetcher.FetchMessageType(context.Background(), "blah.blah.blah/fee.fi.fo.Fum")
	require.NoError(t, err)
	require.Equal(t, "fee.fi.fo.Fum", typ.Name)
	// and an enum type
	en, err := fetcher.FetchEnumType(context.Background(), "blah.blah.blah/fee.fi.fo.Foo")
	require.NoError(t, err)
	require.Equal(t, "fee.fi.fo.Foo", en.Name)

	// now ask for same URL, but swapped types
	_, err = fetcher.FetchEnumType(context.Background(), "blah.blah.blah/fee.fi.fo.Fum")
	var unexpectedTypeErr *protoresolve.ErrUnexpectedType
	require.ErrorAs(t, err, &unexpectedTypeErr)
	require.ErrorContains(t, err, "expected an enum, got a message")
	_, err = fetcher.FetchMessageType(context.Background(), "blah.blah.blah/fee.fi.fo.Foo")
	require.ErrorAs(t, err, &unexpectedTypeErr)
	require.ErrorContains(t, err, "expected a message, got an enum")
}

func TestCachingTypeFetcher_Concurrency(t *testing.T) {
	// make sure we are thread safe
	var mu sync.Mutex
	counts := map[string]int{}
	tf := CachingTypeFetcher(TypeFetcherFunc(func(ctx context.Context, url string, enum bool) (proto.Message, error) {
		mu.Lock()
		counts[url] = counts[url] + 1
		mu.Unlock()
		return testFetcher(ctx, url, enum)
	}))

	ctx, cancel := context.WithCancel(context.Background())
	names := []string{"Fee", "Fi", "Fo", "Fum", "I", "Smell", "Blood", "Of", "Englishman"}
	var queryCount int32
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; ctx.Err() == nil; i = (i + 1) % len(names) {
				n := "fee.fi.fo." + names[i]
				// message
				typ, err := tf.FetchMessageType(context.Background(), "blah.blah.blah/"+n)
				require.NoError(t, err)
				require.Equal(t, n, typ.Name)
				atomic.AddInt32(&queryCount, 1)
				// enum
				en, err := tf.FetchEnumType(context.Background(), "blah.blah.blah.en/"+n)
				require.NoError(t, err)
				require.Equal(t, n, en.Name)
				atomic.AddInt32(&queryCount, 1)
			}
		}()
	}

	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()

	// underlying fetcher invoked just once per URL
	for _, v := range counts {
		require.Equal(t, 1, v)
	}

	require.Greater(t, atomic.LoadInt32(&queryCount), int32(len(counts)))
}

func TestHttpTypeFetcher(t *testing.T) {
	trt := &testRoundTripper{counts: map[string]int{}}
	fetcher := HttpTypeFetcher(trt, 65536, 10)

	for i := 0; i < 10; i++ {
		typ, err := fetcher.FetchMessageType(context.Background(), "blah.blah.blah/fee.fi.fo.Message")
		require.NoError(t, err)
		require.Equal(t, "fee.fi.fo.Message", typ.Name)
	}

	for i := 0; i < 10; i++ {
		// name must have Enum for test fetcher to return an enum type
		en, err := fetcher.FetchEnumType(context.Background(), "blah.blah.blah/fee.fi.fo.Enum")
		require.NoError(t, err)
		require.Equal(t, "fee.fi.fo.Enum", en.Name)
	}

	// HttpTypeFetcher caches results
	require.Equal(t, 1, trt.counts["https://blah.blah.blah/fee.fi.fo.Message"])
	require.Equal(t, 1, trt.counts["https://blah.blah.blah/fee.fi.fo.Enum"])
}

func TestHttpTypeFetcher_ParallelDownloads(t *testing.T) {
	trt := &testRoundTripper{counts: map[string]int{}, delay: 100 * time.Millisecond}
	fetcher := HttpTypeFetcher(trt, 65536, 10)
	// We spin up 100 fetches in parallel, but only 10 can go at a time and each
	// one takes 100millis. So it should take about 1 second.
	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		index := i // don't capture loop variable
		go func() {
			defer wg.Done()
			name := fmt.Sprintf("fee.fi.fo.Fum%d", index)
			typ, err := fetcher.FetchMessageType(context.Background(), "blah.blah.blah/"+name)
			require.NoError(t, err)
			require.Equal(t, name, typ.Name)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	// we should have observed exactly the maximum number of parallel downloads
	require.Equal(t, 10, trt.max)
	// should have taken about a second
	require.GreaterOrEqual(t, elapsed, time.Second)
}

func TestHttpTypeFetcher_SizeLimits(t *testing.T) {
	trt := &testRoundTripper{counts: map[string]int{}}
	// small size that will always get tripped
	fetcher := HttpTypeFetcher(trt, 32, 10)

	// name with "Size" causes content-length to be reported in header
	_, err := fetcher.FetchMessageType(context.Background(), "blah.blah.blah/fee.fi.fo.FumSize")
	require.ErrorContains(t, err, "is larger than limit of 32")

	// without size in the name, no content-length (e.g. streaming response)
	_, err = fetcher.FetchMessageType(context.Background(), "blah.blah.blah/fee.fi.fo.Fum")
	require.ErrorContains(t, err, "is larger than limit of 32")
}

type testRoundTripper struct {
	// artificial delay that each fake HTTP request will take
	delay time.Duration
	mu    sync.Mutex
	// counts by requested URL
	counts map[string]int
	// total active downloads
	active int
	// max observed active downloads
	max int
}

func (t *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()

	t.mu.Lock()
	t.counts[url] = t.counts[url] + 1
	t.active++
	if t.active > t.max {
		t.max = t.active
	}
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.active--
		t.mu.Unlock()
	}()

	time.Sleep(t.delay)

	name := url[strings.LastIndex(req.URL.Path, "/")+1:]
	includeContentLength := strings.Contains(name, "Size")
	pm, err := testFetcher(req.Context(), url, strings.Contains(name, "Enum"))
	if err != nil {
		return nil, err
	}
	b, err := proto.Marshal(pm)
	if err != nil {
		return nil, err
	}
	contentLength := int64(-1)
	if includeContentLength {
		contentLength = int64(len(b))
	}
	return &http.Response{
		StatusCode:    200,
		Status:        "200 OK",
		ContentLength: contentLength,
		Body:          io.NopCloser(bytes.NewReader(b)),
	}, nil
}

func testFetcher(_ context.Context, url string, enum bool) (proto.Message, error) {
	name := url[strings.LastIndex(url, "/")+1:]
	if strings.Contains(name, "Error") {
		return nil, errors.New(name)
	} else if enum {
		return &typepb.Enum{
			Name:          name,
			SourceContext: &sourcecontextpb.SourceContext{FileName: "test.proto"},
			Syntax:        typepb.Syntax_SYNTAX_PROTO3,
			Enumvalue: []*typepb.EnumValue{
				{Name: "A", Number: 0},
				{Name: "B", Number: 1},
				{Name: "C", Number: 2},
			},
		}, nil
	} else {
		return &typepb.Type{
			Name:          name,
			SourceContext: &sourcecontextpb.SourceContext{FileName: "test.proto"},
			Syntax:        typepb.Syntax_SYNTAX_PROTO3,
			Fields: []*typepb.Field{
				{Name: "a", Number: 1, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, Kind: typepb.Field_TYPE_INT64},
				{Name: "b", Number: 2, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, Kind: typepb.Field_TYPE_STRING},
				{Name: "c1", Number: 3, OneofIndex: 1, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, Kind: typepb.Field_TYPE_STRING},
				{Name: "c2", Number: 4, OneofIndex: 1, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, Kind: typepb.Field_TYPE_BOOL},
				{Name: "c3", Number: 5, OneofIndex: 1, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, Kind: typepb.Field_TYPE_DOUBLE},
				{Name: "d", Number: 6, Cardinality: typepb.Field_CARDINALITY_REPEATED, Kind: typepb.Field_TYPE_MESSAGE, TypeUrl: "type.googleapis.com/foo.bar.Baz"},
				{Name: "e", Number: 7, Cardinality: typepb.Field_CARDINALITY_OPTIONAL, Kind: typepb.Field_TYPE_ENUM, TypeUrl: "type.googleapis.com/foo.bar.Blah"},
			},
			Oneofs: []string{"union"},
		}, nil
	}
}
