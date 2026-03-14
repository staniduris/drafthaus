package server

import (
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, status: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// LoggingMiddleware logs method, path, status code, and duration for each request.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)
		next.ServeHTTP(rw, r)
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rw.status, time.Since(start))
	})
}

// gzipResponseWriter intercepts writes to compress them.
type gzipResponseWriter struct {
	http.ResponseWriter
	gz     *gzip.Writer
	status int
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	g.status = code
	g.ResponseWriter.WriteHeader(code)
}

func (g *gzipResponseWriter) Write(b []byte) (int, error) {
	return g.gz.Write(b)
}

// GzipMiddleware compresses responses for text/html, text/css, application/json,
// and application/javascript when the client accepts gzip encoding.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		// We need to buffer the response to inspect the Content-Type before
		// deciding whether to compress. Use a buffering writer that defers the
		// gzip decision to the first Write/WriteHeader.
		bw := &bufferingWriter{inner: w, r: r, next: next}
		next.ServeHTTP(bw, r)
		bw.flush()
	})
}

// bufferingWriter delays the decision of whether to gzip until we know the
// Content-Type (set either via Header().Set or WriteHeader).
type bufferingWriter struct {
	http.ResponseWriter
	inner  http.ResponseWriter
	r      *http.Request
	next   http.Handler
	buf    []byte
	status int
	done   bool
}

func (b *bufferingWriter) WriteHeader(code int) {
	b.status = code
}

func (b *bufferingWriter) Write(p []byte) (int, error) {
	b.buf = append(b.buf, p...)
	return len(p), nil
}

func (b *bufferingWriter) Header() http.Header {
	return b.inner.Header()
}

func (b *bufferingWriter) flush() {
	if b.done {
		return
	}
	b.done = true

	ct := b.inner.Header().Get("Content-Type")
	shouldGzip := strings.Contains(ct, "text/html") ||
		strings.Contains(ct, "text/css") ||
		strings.Contains(ct, "application/json") ||
		strings.Contains(ct, "application/javascript")

	if !shouldGzip || len(b.buf) == 0 {
		if b.status != 0 {
			b.inner.WriteHeader(b.status)
		}
		b.inner.Write(b.buf) //nolint:errcheck
		return
	}

	b.inner.Header().Set("Content-Encoding", "gzip")
	b.inner.Header().Del("Content-Length")
	if b.status != 0 {
		b.inner.WriteHeader(b.status)
	}

	gz, err := gzip.NewWriterLevel(b.inner, gzip.BestSpeed)
	if err != nil {
		b.inner.Write(b.buf) //nolint:errcheck
		return
	}
	defer gz.Close()

	io.WriteString(gz, string(b.buf)) //nolint:errcheck
}

// CORSMiddleware sets CORS headers on /_api/ requests and handles OPTIONS preflight.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_api/") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SecurityMiddleware sets standard security headers on every response.
func SecurityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
