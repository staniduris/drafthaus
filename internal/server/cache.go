package server

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
	"time"
)

type pageCache struct {
	entries map[string]*cacheEntry
	mu      sync.RWMutex
	maxAge  time.Duration
}

type cacheEntry struct {
	body      []byte
	createdAt time.Time
}

// NewPageCache creates a page cache with the given TTL.
func NewPageCache(maxAge time.Duration) *pageCache {
	return &pageCache{
		entries: make(map[string]*cacheEntry),
		maxAge:  maxAge,
	}
}

// Get returns a cached body if present and not expired.
func (c *pageCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	if time.Since(entry.createdAt) > c.maxAge {
		return nil, false
	}
	return entry.body, true
}

// Set stores a page in the cache.
func (c *pageCache) Set(key string, body []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = &cacheEntry{
		body:      body,
		createdAt: time.Now(),
	}
}

// Invalidate removes all entries whose key starts with prefix.
func (c *pageCache) Invalidate(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
}

// cachingResponseWriter captures the response body for caching.
type cachingResponseWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (cw *cachingResponseWriter) WriteHeader(code int) {
	cw.status = code
	cw.ResponseWriter.WriteHeader(code)
}

func (cw *cachingResponseWriter) Write(b []byte) (int, error) {
	cw.buf.Write(b)
	return cw.ResponseWriter.Write(b)
}

func isCacheable(path string) bool {
	if strings.HasPrefix(path, "/_admin") ||
		strings.HasPrefix(path, "/_api") ||
		strings.HasPrefix(path, "/_plugins") {
		return false
	}
	return true
}

// CacheMiddleware caches GET responses for public pages.
func CacheMiddleware(cache *pageCache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || !isCacheable(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			key := r.URL.Path
			if body, ok := cache.Get(key); ok {
				w.Header().Set("X-Cache", "HIT")
				w.Write(body) //nolint:errcheck
				return
			}

			cw := &cachingResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(cw, r)

			if cw.status == http.StatusOK && cw.buf.Len() > 0 {
				cache.Set(key, cw.buf.Bytes())
			}
		})
	}
}
