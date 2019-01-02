package msgregistry

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/genproto/protobuf/ptype"
	"google.golang.org/genproto/protobuf/source_context"

	"github.com/jhump/protoreflect/internal/testutil"
)

func TestCachingTypeFetcher(t *testing.T) {
	counts := map[string]int{}
	uncached := func(url string, enum bool) (proto.Message, error) {
		counts[url] = counts[url] + 1
		return testFetcher(url, enum)
	}

	// observe the underlying type fetcher get invoked 10x
	for i := 0; i < 10; i++ {
		pm, err := uncached("blah.blah.blah/fee.fi.fo.Fum", false)
		testutil.Ok(t, err)
		typ := pm.(*ptype.Type)
		testutil.Eq(t, "fee.fi.fo.Fum", typ.Name)
	}
	for i := 0; i < 10; i++ {
		pm, err := uncached("blah.blah.blah/fee.fi.fo.Foo", true)
		testutil.Ok(t, err)
		en := pm.(*ptype.Enum)
		testutil.Eq(t, "fee.fi.fo.Foo", en.Name)
	}

	testutil.Eq(t, 10, counts["blah.blah.blah/fee.fi.fo.Fum"])
	testutil.Eq(t, 10, counts["blah.blah.blah/fee.fi.fo.Foo"])

	// now we'll see the underlying fetcher invoked just one more time,
	// after which the result is cached
	cached := CachingTypeFetcher(uncached)

	for i := 0; i < 10; i++ {
		pm, err := cached("blah.blah.blah/fee.fi.fo.Fum", false)
		testutil.Ok(t, err)
		typ := pm.(*ptype.Type)
		testutil.Eq(t, "fee.fi.fo.Fum", typ.Name)
	}

	for i := 0; i < 10; i++ {
		pm, err := cached("blah.blah.blah/fee.fi.fo.Foo", true)
		testutil.Ok(t, err)
		en := pm.(*ptype.Enum)
		testutil.Eq(t, "fee.fi.fo.Foo", en.Name)
	}

	testutil.Eq(t, 11, counts["blah.blah.blah/fee.fi.fo.Fum"])
	testutil.Eq(t, 11, counts["blah.blah.blah/fee.fi.fo.Foo"])
}

func TestCachingTypeFetcher_MismatchType(t *testing.T) {
	fetcher := CachingTypeFetcher(testFetcher)
	// get a message type
	pm, err := fetcher("blah.blah.blah/fee.fi.fo.Fum", false)
	testutil.Ok(t, err)
	typ := pm.(*ptype.Type)
	testutil.Eq(t, "fee.fi.fo.Fum", typ.Name)
	// and an enum type
	pm, err = fetcher("blah.blah.blah/fee.fi.fo.Foo", true)
	testutil.Ok(t, err)
	en := pm.(*ptype.Enum)
	testutil.Eq(t, "fee.fi.fo.Foo", en.Name)

	// now ask for same URL, but swapped types
	_, err = fetcher("blah.blah.blah/fee.fi.fo.Fum", true)
	testutil.Require(t, err != nil && strings.Contains(err.Error(), "wanted enum, got message"))
	_, err = fetcher("blah.blah.blah/fee.fi.fo.Foo", false)
	testutil.Require(t, err != nil && strings.Contains(err.Error(), "wanted message, got enum"))
}

func TestCachingTypeFetcher_Concurrency(t *testing.T) {
	// make sure we are thread safe
	var mu sync.Mutex
	counts := map[string]int{}
	tf := CachingTypeFetcher(func(url string, enum bool) (proto.Message, error) {
		mu.Lock()
		counts[url] = counts[url] + 1
		mu.Unlock()
		return testFetcher(url, enum)
	})

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
				pm, err := tf("blah.blah.blah/"+n, false)
				testutil.Ok(t, err)
				typ := pm.(*ptype.Type)
				testutil.Eq(t, n, typ.Name)
				atomic.AddInt32(&queryCount, 1)
				// enum
				pm, err = tf("blah.blah.blah.en/"+n, true)
				testutil.Ok(t, err)
				en := pm.(*ptype.Enum)
				testutil.Eq(t, n, en.Name)
				atomic.AddInt32(&queryCount, 1)
			}
		}()
	}

	time.Sleep(2 * time.Second)
	cancel()
	wg.Wait()

	// underlying fetcher invoked just once per URL
	for _, v := range counts {
		testutil.Eq(t, 1, v)
	}

	testutil.Require(t, atomic.LoadInt32(&queryCount) > int32(len(counts)))
}

func TestHttpTypeFetcher(t *testing.T) {
	trt := &testRoundTripper{counts: map[string]int{}}
	fetcher := HttpTypeFetcher(trt, 65536, 10)

	for i := 0; i < 10; i++ {
		pm, err := fetcher("blah.blah.blah/fee.fi.fo.Message", false)
		testutil.Ok(t, err)
		typ := pm.(*ptype.Type)
		testutil.Eq(t, "fee.fi.fo.Message", typ.Name)
	}

	for i := 0; i < 10; i++ {
		// name must have Enum for test fetcher to return an enum type
		pm, err := fetcher("blah.blah.blah/fee.fi.fo.Enum", true)
		testutil.Ok(t, err)
		en := pm.(*ptype.Enum)
		testutil.Eq(t, "fee.fi.fo.Enum", en.Name)
	}

	// HttpTypeFetcher caches results
	testutil.Eq(t, 1, trt.counts["https://blah.blah.blah/fee.fi.fo.Message"])
	testutil.Eq(t, 1, trt.counts["https://blah.blah.blah/fee.fi.fo.Enum"])
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
			pm, err := fetcher("blah.blah.blah/"+name, false)
			testutil.Ok(t, err)
			typ := pm.(*ptype.Type)
			testutil.Eq(t, name, typ.Name)
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)

	// we should have observed exactly the maximum number of parallel downloads
	testutil.Eq(t, 10, trt.max)
	// should have taken about a second
	testutil.Require(t, elapsed >= time.Second)
}

func TestHttpTypeFetcher_SizeLimits(t *testing.T) {
	trt := &testRoundTripper{counts: map[string]int{}}
	// small size that will always get tripped
	fetcher := HttpTypeFetcher(trt, 32, 10)

	// name with "Size" causes content-length to be reported in header
	_, err := fetcher("blah.blah.blah/fee.fi.fo.FumSize", false)
	testutil.Require(t, err != nil && strings.Contains(err.Error(), "is larger than limit of 32"))

	// without size in the name, no content-length (e.g. streaming response)
	_, err = fetcher("blah.blah.blah/fee.fi.fo.Fum", false)
	testutil.Require(t, err != nil && strings.Contains(err.Error(), "is larger than limit of 32"))
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
	pm, err := testFetcher(url, strings.Contains(name, "Enum"))
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
		Body:          ioutil.NopCloser(bytes.NewReader(b)),
	}, nil
}

func testFetcher(url string, enum bool) (proto.Message, error) {
	name := url[strings.LastIndex(url, "/")+1:]
	if strings.Contains(name, "Error") {
		return nil, errors.New(name)
	} else if enum {
		return &ptype.Enum{
			Name:          name,
			SourceContext: &source_context.SourceContext{FileName: "test.proto"},
			Syntax:        ptype.Syntax_SYNTAX_PROTO3,
			Enumvalue: []*ptype.EnumValue{
				{Name: "A", Number: 0},
				{Name: "B", Number: 1},
				{Name: "C", Number: 2},
			},
		}, nil
	} else {
		return &ptype.Type{
			Name:          name,
			SourceContext: &source_context.SourceContext{FileName: "test.proto"},
			Syntax:        ptype.Syntax_SYNTAX_PROTO3,
			Fields: []*ptype.Field{
				{Name: "a", Number: 1, Cardinality: ptype.Field_CARDINALITY_OPTIONAL, Kind: ptype.Field_TYPE_INT64},
				{Name: "b", Number: 2, Cardinality: ptype.Field_CARDINALITY_OPTIONAL, Kind: ptype.Field_TYPE_STRING},
				{Name: "c1", Number: 3, OneofIndex: 1, Cardinality: ptype.Field_CARDINALITY_OPTIONAL, Kind: ptype.Field_TYPE_STRING},
				{Name: "c2", Number: 4, OneofIndex: 1, Cardinality: ptype.Field_CARDINALITY_OPTIONAL, Kind: ptype.Field_TYPE_BOOL},
				{Name: "c3", Number: 5, OneofIndex: 1, Cardinality: ptype.Field_CARDINALITY_OPTIONAL, Kind: ptype.Field_TYPE_DOUBLE},
				{Name: "d", Number: 6, Cardinality: ptype.Field_CARDINALITY_REPEATED, Kind: ptype.Field_TYPE_MESSAGE, TypeUrl: "type.googleapis.com/foo.bar.Baz"},
				{Name: "e", Number: 7, Cardinality: ptype.Field_CARDINALITY_OPTIONAL, Kind: ptype.Field_TYPE_ENUM, TypeUrl: "type.googleapis.com/foo.bar.Blah"},
			},
			Oneofs: []string{"union"},
		}, nil
	}
}
