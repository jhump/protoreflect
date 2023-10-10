package protoresolve

import (
	"bytes"
	"context"
	"fmt"
	"google.golang.org/protobuf/types/known/apipb"
	"io"
	"net/http"
	"sync"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/typepb"
)

// TypeFetcher is a value that knows how to fetch type definitions for a URL.
// The type definitions are represented by google.protobuf.Type and google.protobuf.Enum
// messages (which were originally part of the specification for google.protobuf.Any
// and how such types could be resolved at runtime).
type TypeFetcher interface {
	// FetchMessageType fetches the definition of a message type that is identified
	// by the given URL.
	FetchMessageType(ctx context.Context, url string) (*typepb.Type, error)
	// FetchEnumType fetches the definition of an enum type that is identified by
	// the given URL.
	FetchEnumType(ctx context.Context, url string) (*typepb.Enum, error)
}

// TypeFetcherFunc is a TypeFetcher implementation backed by a single function.
// The function accepts a parameter to have it switch between fetching a message
// type vs. finding an enum type.
type TypeFetcherFunc func(ctx context.Context, url string, enum bool) (proto.Message, error)

var _ TypeFetcher = TypeFetcherFunc(nil)

// FetchMessageType implements the TypeFetcher interface.
func (t TypeFetcherFunc) FetchMessageType(ctx context.Context, url string) (*typepb.Type, error) {
	msg, err := t(ctx, url, false)
	if err != nil {
		return nil, err
	}
	if typ, ok := msg.(*typepb.Type); ok {
		return typ, nil
	}
	return nil, newUnexpectedTypeError(DescriptorKindMessage, msg, url)
}

// FetchEnumType implements the TypeFetcher interface.
func (t TypeFetcherFunc) FetchEnumType(ctx context.Context, url string) (*typepb.Enum, error) {
	msg, err := t(ctx, url, true)
	if err != nil {
		return nil, err
	}
	if en, ok := msg.(*typepb.Enum); ok {
		return en, nil
	}
	return nil, newUnexpectedTypeError(DescriptorKindEnum, msg, url)
}

func newUnexpectedTypeError(expecting DescriptorKind, typ proto.Message, url string) *ErrUnexpectedType {
	var actualKind DescriptorKind
	switch typ.(type) {
	case *typepb.Type:
		actualKind = DescriptorKindMessage
	case *typepb.Field:
		actualKind = DescriptorKindField
	case *typepb.Enum:
		actualKind = DescriptorKindEnum
	case *typepb.EnumValue:
		actualKind = DescriptorKindEnumValue
	case *apipb.Api:
		actualKind = DescriptorKindService
	case *apipb.Method:
		actualKind = DescriptorKindMethod
	default:
		actualKind = DescriptorKindUnknown
	}
	return &ErrUnexpectedType{
		URL:       url,
		Expecting: expecting,
		Actual:    actualKind,
	}
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
	switch m.(type) {
	case *typepb.Type:
		if !enum {
			return m, nil
		}
	case *typepb.Enum:
		if enum {
			return m, nil
		}
	}
	var wanted DescriptorKind
	if enum {
		wanted = DescriptorKindEnum
	} else {
		wanted = DescriptorKindMessage
	}
	return nil, newUnexpectedTypeError(wanted, m, url)
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

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNonAuthoritativeInfo {
			if resp.StatusCode == http.StatusNoContent ||
				resp.StatusCode == http.StatusNotImplemented ||
				(resp.StatusCode >= 300 && resp.StatusCode <= 499) {
				// No content, unimplemented, redirect, or request error? Treat as "not found".
				return nil, fmt.Errorf("%w: HTTP request returned status code %s", ErrNotFound, resp.Status)
			}
			return nil, fmt.Errorf("HTTP request returned unsupported status code: %s", resp.Status)
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
		}
		var ret typepb.Type
		if err = proto.Unmarshal(buf.Bytes(), &ret); err != nil {
			return nil, err
		}
		return &ret, nil
	}))
}

var bufferPool = sync.Pool{New: func() interface{} {
	buf := make([]byte, 8192)
	return bytes.NewBuffer(buf)
}}
