package remotereg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/apipb"
	"google.golang.org/protobuf/types/known/typepb"

	"github.com/jhump/protoreflect/v2/protoresolve"
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
	return nil, newUnexpectedTypeError(protoresolve.DescriptorKindMessage, msg, url)
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
	return nil, newUnexpectedTypeError(protoresolve.DescriptorKindEnum, msg, url)
}

func newUnexpectedTypeError(expecting protoresolve.DescriptorKind, typ proto.Message, url string) *protoresolve.ErrUnexpectedType {
	var actualKind protoresolve.DescriptorKind
	switch typ.(type) {
	case *typepb.Type:
		actualKind = protoresolve.DescriptorKindMessage
	case *typepb.Field:
		actualKind = protoresolve.DescriptorKindField
	case *typepb.Enum:
		actualKind = protoresolve.DescriptorKindEnum
	case *typepb.EnumValue:
		actualKind = protoresolve.DescriptorKindEnumValue
	case *apipb.Api:
		actualKind = protoresolve.DescriptorKindService
	case *apipb.Method:
		actualKind = protoresolve.DescriptorKindMethod
	default:
		actualKind = protoresolve.DescriptorKindUnknown
	}
	return &protoresolve.ErrUnexpectedType{
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
	mu      sync.Mutex
	entries map[string]*cachingFetcherEntry
}

type cachingFetcherEntry struct {
	// Closed when msg and err are set.
	done chan struct{}
	msg  proto.Message
	err  error
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
	m, err := c.getOrLoad(ctx, url, func(ctx context.Context) (proto.Message, error) {
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
	var wanted protoresolve.DescriptorKind
	if enum {
		wanted = protoresolve.DescriptorKindEnum
	} else {
		wanted = protoresolve.DescriptorKindMessage
	}
	return nil, newUnexpectedTypeError(wanted, m, url)
}

// getOrLoad returns the cached result for the given key, waiting for it if
// it is being loaded concurrently. If there is no result, it calls loader and
// caches its result, unless the result is an error.
func (c *cachingFetcher) getOrLoad(ctx context.Context, key string, loader func(context.Context) (proto.Message, error)) (proto.Message, error) {
	for {
		c.mu.Lock()
		entry, ok := c.entries[key]
		if !ok {
			entry = &cachingFetcherEntry{done: make(chan struct{})}
			c.entries[key] = entry
			c.mu.Unlock()
			return c.load(ctx, key, entry, loader)
		}
		c.mu.Unlock()

		select {
		case <-entry.done:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if ctx.Err() == nil && (errors.Is(entry.err, context.Canceled) || errors.Is(entry.err, context.DeadlineExceeded)) {
			// The load was ended by the context of the caller that started
			// it, not by ours. So try again.
			continue
		}
		return entry.msg, entry.err
	}
}

// load calls loader and records its result in the given entry.
func (c *cachingFetcher) load(ctx context.Context, key string, entry *cachingFetcherEntry, loader func(context.Context) (proto.Message, error)) (msg proto.Message, err error) {
	completed := false
	defer func() {
		if !completed {
			// The loader panicked. Concurrent callers get an error, and the
			// panic continues.
			err = fmt.Errorf("fetching %s: type fetcher panicked", key)
		}
		entry.msg, entry.err = msg, err
		if err != nil {
			// don't leave a failed entry in the cache
			c.mu.Lock()
			delete(c.entries, key)
			c.mu.Unlock()
		}
		close(entry.done)
	}()
	msg, err = loader(ctx)
	completed = true
	return msg, err
}

// HTTPTypeFetcher returns a TypeFetcher that uses the given HTTP transport to query and
// download type definitions. The given szLimit is the maximum response size accepted. If
// used from multiple goroutines (like when a type's dependency graph is resolved in
// parallel), this resolver limits the number of parallel queries/downloads to the given
// parLimit. If parLimit is zero or negative, the number of parallel queries/downloads is
// not limited.
func HTTPTypeFetcher(transport http.RoundTripper, szLimit, parLimit int) TypeFetcher {
	var sem *semaphore.Weighted
	if parLimit > 0 {
		sem = semaphore.NewWeighted(int64(parLimit))
	}
	return CachingTypeFetcher(TypeFetcherFunc(func(ctx context.Context, typeURL string, enum bool) (proto.Message, error) {
		if sem != nil {
			if err := sem.Acquire(ctx, 1); err != nil {
				return nil, err
			}
			defer sem.Release(1)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ensureScheme(typeURL), http.NoBody)
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
				return nil, fmt.Errorf("%w: HTTP request returned status code %s", protoresolve.ErrNotFound, resp.Status)
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

var bufferPool = sync.Pool{New: func() any {
	return bytes.NewBuffer(make([]byte, 0, 8192))
}}
