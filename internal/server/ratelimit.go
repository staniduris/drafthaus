package server

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.Mutex
	rate     int
	window   time.Duration
	reqCount int
}

type visitor struct {
	count       int
	windowStart time.Time
}

// NewRateLimiter creates a rate limiter allowing rate requests per window.
func NewRateLimiter(rate int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.reqCount++
	if rl.reqCount%100 == 0 {
		cutoff := time.Now().Add(-2 * rl.window)
		for k, v := range rl.visitors {
			if v.windowStart.Before(cutoff) {
				delete(rl.visitors, k)
			}
		}
	}

	now := time.Now()
	v, ok := rl.visitors[ip]
	if !ok || now.Sub(v.windowStart) > rl.window {
		rl.visitors[ip] = &visitor{count: 1, windowStart: now}
		return true
	}

	v.count++
	return v.count <= rl.rate
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.SplitN(fwd, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// tieredRateLimitMiddleware applies apiRL to /_api paths and pageRL to all others.
func tieredRateLimitMiddleware(pageRL, apiRL *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		pageHandler := RateLimitMiddleware(pageRL)(next)
		apiHandler := RateLimitMiddleware(apiRL)(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/_api/") {
				apiHandler.ServeHTTP(w, r)
			} else {
				pageHandler.ServeHTTP(w, r)
			}
		})
	}
}

// RateLimitMiddleware rejects requests exceeding the rate limit with 429.
func RateLimitMiddleware(rl *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r)
			if !rl.allow(ip) {
				retryAfter := int(rl.window.Seconds())
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
