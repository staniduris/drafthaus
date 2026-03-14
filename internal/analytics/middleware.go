package analytics

import "net/http"

// Middleware records page views for each request.
func Middleware(tracker *Tracker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Track in background goroutine to not slow down response.
			go tracker.Track(r, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	}
}
