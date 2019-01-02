package msgregistry

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/golang/protobuf/proto"
	"google.golang.org/genproto/protobuf/ptype"
)

// TypeFetcher is a simple operation that retrieves a type definition for a given type URL.
// The returned proto message will be either a *ptype.Enum or a *ptype.Type, depending on
// whether the enum flag is true or not.
type TypeFetcher func(url string, enum bool) (proto.Message, error)

// CachingTypeFetcher adds a caching layer to the given type fetcher. Queries for
// types that have already been fetched will not result in another call to the
// underlying fetcher and instead are retrieved from the cache.
func CachingTypeFetcher(fetcher TypeFetcher) TypeFetcher {
	c := protoCache{entries: map[string]*protoCacheEntry{}}
	return func(typeUrl string, enum bool) (proto.Message, error) {
		m, err := c.getOrLoad(typeUrl, func() (proto.Message, error) {
			return fetcher(typeUrl, enum)
		})
		if err != nil {
			return nil, err
		}
		if _, isEnum := m.(*ptype.Enum); enum != isEnum {
			var want, got string
			if enum {
				want = "enum"
				got = "message"
			} else {
				want = "message"
				got = "enum"
			}
			return nil, fmt.Errorf("type for URL %v is the wrong type: wanted %s, got %s", typeUrl, want, got)
		}
		return m.(proto.Message), nil
	}
}

type protoCache struct {
	mu      sync.RWMutex
	entries map[string]*protoCacheEntry
}

type protoCacheEntry struct {
	msg proto.Message
	err error
	wg  sync.WaitGroup
}

func (c *protoCache) getOrLoad(key string, loader func() (proto.Message, error)) (m proto.Message, err error) {
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
	e := &protoCacheEntry{}
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
	sem := semaphore{count: parLimit, permits: parLimit}
	return CachingTypeFetcher(func(typeUrl string, enum bool) (proto.Message, error) {
		sem.Acquire()
		defer sem.Release()

		// build URL
		u, err := url.Parse(ensureScheme(typeUrl))
		if err != nil {
			return nil, err
		}

		resp, err := transport.RoundTrip(&http.Request{URL: u})
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("HTTP request returned non-200 status code: %s", resp.Status)
		}

		if resp.ContentLength > int64(szLimit) {
			return nil, fmt.Errorf("type definition size %d is larger than limit of %d", resp.ContentLength, szLimit)
		}

		// download the response, up to the given size limit, into a buffer
		bufptr := bufferPool.Get().(*[]byte)
		defer bufferPool.Put(bufptr)
		buf := *bufptr
		var b bytes.Buffer
		for {
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				return nil, err
			}
			if n > 0 {
				if b.Len()+n > szLimit {
					return nil, fmt.Errorf("type definition size %d+ is larger than limit of %d", b.Len()+n, szLimit)
				}
				b.Write(buf[:n])
			}
			if err == io.EOF {
				break
			}
		}

		// now we can de-serialize the type definition
		if enum {
			var ret ptype.Enum
			if err = proto.Unmarshal(b.Bytes(), &ret); err != nil {
				return nil, err
			}
			return &ret, nil
		} else {
			var ret ptype.Type
			if err = proto.Unmarshal(b.Bytes(), &ret); err != nil {
				return nil, err
			}
			return &ret, nil
		}
	})
}

var bufferPool = sync.Pool{New: func() interface{} {
	buf := make([]byte, 8192)
	return &buf
}}

type semaphore struct {
	lock    sync.Mutex
	count   int
	permits int
	cond    sync.Cond
}

func (s *semaphore) Acquire() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.cond.L == nil {
		s.cond.L = &s.lock
	}

	for s.count == 0 {
		s.cond.Wait()
	}
	s.count--
}

func (s *semaphore) Release() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.cond.L == nil {
		s.cond.L = &s.lock
	}

	if s.count == s.permits {
		panic("call to Release() without corresponding call to Acquire()")
	}
	s.count++
	s.cond.Signal()
}
