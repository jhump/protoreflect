package protoresolve

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/typepb"
)

type TypeFetcher interface {
	FetchMessageType(ctx context.Context, url string) (*typepb.Type, error)
	FetchEnumType(ctx context.Context, url string) (*typepb.Enum, error)
}

type TypeFetcherFunc func(ctx context.Context, url string, enum bool) (proto.Message, error)

var _ TypeFetcher = TypeFetcherFunc(nil)

func (t TypeFetcherFunc) FetchMessageType(ctx context.Context, url string) (*typepb.Type, error) {
	msg, err := t(ctx, url, false)
	if err != nil {
		return nil, err
	}
	if typ, ok := msg.(*typepb.Type); ok {
		return typ, nil
	}
	return nil, fmt.Errorf("fetcher returned wrong type: expecting %T, got %T", (*typepb.Type)(nil), msg)
}

func (t TypeFetcherFunc) FetchEnumType(ctx context.Context, url string) (*typepb.Enum, error) {
	msg, err := t(ctx, url, true)
	if err != nil {
		return nil, err
	}
	if en, ok := msg.(*typepb.Enum); ok {
		return en, nil
	}
	return nil, fmt.Errorf("fetcher returned wrong type: expecting %T, got %T", (*typepb.Enum)(nil), msg)
}

// CachingTypeFetcher adds a caching layer to the given type fetcher. Queries for
// types that have already been fetched will not result in another call to the
// underlying fetcher and instead are retrieved from the cache.
func CachingTypeFetcher(fetcher TypeFetcher) TypeFetcher {
	return &cachingFetcher{fetcher: fetcher, entries: map[string]*cachingFetcherEntry{}}
}

type cachingFetcher struct {
	fetcher TypeFetcher
	mu      sync.RWMutex
	entries map[string]*cachingFetcherEntry
}

type cachingFetcherEntry struct {
	msg proto.Message
	err error
	wg  sync.WaitGroup
}

func (c *cachingFetcher) FetchMessageType(ctx context.Context, url string) (*typepb.Type, error) {
	msg, err := c.fetchType(ctx, url, false)
	if err != nil {
		return nil, err
	}
	return msg.(*typepb.Type), nil
}

func (c *cachingFetcher) FetchEnumType(ctx context.Context, url string) (*typepb.Enum, error) {
	msg, err := c.fetchType(ctx, url, true)
	if err != nil {
		return nil, err
	}
	return msg.(*typepb.Enum), nil
}

func (c *cachingFetcher) fetchType(ctx context.Context, url string, enum bool) (proto.Message, error) {
	m, err := c.getOrLoad(url, func() (proto.Message, error) {
		if enum {
			return c.fetcher.FetchEnumType(ctx, url)
		}
		return c.fetcher.FetchMessageType(ctx, url)
	})
	if err != nil {
		return nil, err
	}
	if _, isEnum := m.(*typepb.Enum); enum != isEnum {
		var want, got string
		if enum {
			want = "enum"
			got = "message"
		} else {
			want = "message"
			got = "enum"
		}
		return nil, fmt.Errorf("type for URL %v is the wrong type: wanted %s, got %s", url, want, got)
	}
	return m.(proto.Message), nil
}

func (c *cachingFetcher) getOrLoad(key string, loader func() (proto.Message, error)) (m proto.Message, err error) {
	// see if it's cached
	c.mu.RLock()
	cached, ok := c.entries[key]
	c.mu.RUnlock()
	if ok {
		cached.wg.Wait()
		return cached.msg, cached.err
	}

	// must delegate and cache the result
	c.mu.Lock()
	// double-check, in case it was added concurrently while we were upgrading lock
	cached, ok = c.entries[key]
	if ok {
		c.mu.Unlock()
		cached.wg.Wait()
		return cached.msg, cached.err
	}
	e := &cachingFetcherEntry{}
	e.wg.Add(1)
	c.entries[key] = e
	c.mu.Unlock()
	defer func() {
		if err != nil {
			// don't leave broken entry in the cache
			c.mu.Lock()
			delete(c.entries, key)
			c.mu.Unlock()
		}
		e.msg, e.err = m, err
		e.wg.Done()
	}()

	return loader()
}

// HttpTypeFetcher returns a TypeFetcher that uses the given HTTP transport to query and
// download type definitions. The given szLimit is the maximum response size accepted. If
// used from multiple goroutines (like when a type's dependency graph is resolved in
// parallel), this resolver limits the number of parallel queries/downloads to the given
// parLimit.
func HttpTypeFetcher(transport http.RoundTripper, szLimit, parLimit int) TypeFetcher {
	sem := semaphore.NewWeighted(int64(parLimit))
	return CachingTypeFetcher(TypeFetcherFunc(func(ctx context.Context, typeUrl string, enum bool) (proto.Message, error) {
		if err := sem.Acquire(ctx, 1); err != nil {
			return nil, err
		}
		defer sem.Release(1)

		req, err := http.NewRequestWithContext(ctx, "GET", ensureScheme(typeUrl), http.NoBody)
		if err != nil {
			return nil, err
		}
		resp, err := transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP request returned non-200 status code: %s", resp.Status)
		}

		if resp.ContentLength > int64(szLimit) {
			return nil, fmt.Errorf("type definition size %d is larger than limit of %d", resp.ContentLength, szLimit)
		}

		// download the response, up to the given size limit, into a buffer
		buf := bufferPool.Get().(*bytes.Buffer)
		defer bufferPool.Put(buf)
		buf.Reset()
		body := io.LimitReader(resp.Body, int64(szLimit+1))
		n, err := buf.ReadFrom(body)
		if err != nil {
			return nil, err
		}
		if n > int64(szLimit) {
			return nil, fmt.Errorf("type definition size is larger than limit of %d", szLimit)
		}

		// now we can de-serialize the type definition
		if enum {
			var ret typepb.Enum
			if err = proto.Unmarshal(buf.Bytes(), &ret); err != nil {
				return nil, err
			}
			return &ret, nil
		} else {
			var ret typepb.Type
			if err = proto.Unmarshal(buf.Bytes(), &ret); err != nil {
				return nil, err
			}
			return &ret, nil
		}
	}))
}

var bufferPool = sync.Pool{New: func() interface{} {
	buf := make([]byte, 8192)
	return bytes.NewBuffer(buf)
}}
