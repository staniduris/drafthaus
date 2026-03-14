package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
	"github.com/drafthaus/drafthaus/internal/render"
)

// MultiSiteServer serves multiple .draft files from one process.
type MultiSiteServer struct {
	sites      map[string]*siteHandler
	host       string
	port       int
	httpServer *http.Server
}

type siteHandler struct {
	name    string
	store   draft.Store
	handler http.Handler
}

// NewMultiSiteServer creates a MultiSiteServer bound to host:port.
func NewMultiSiteServer(host string, port int) *MultiSiteServer {
	return &MultiSiteServer{
		sites: make(map[string]*siteHandler),
		host:  host,
		port:  port,
	}
}

// AddSite registers a named site with its store, wiring all layers.
func (ms *MultiSiteServer) AddSite(name string, store draft.Store) error {
	resolver := graph.NewResolver(store)
	pipeline := render.NewPipeline(store, resolver)

	router := NewRouter()
	if err := router.BuildRoutes(store); err != nil {
		return fmt.Errorf("build routes for site %q: %w", name, err)
	}

	sessions := NewSessionStore()

	handlers := NewHandlers(store, resolver, pipeline, router, sessions, nil, nil, nil)

	var h http.Handler = handlers
	h = AuthMiddleware(store, sessions)(h)
	h = CORSMiddleware(h)
	h = GzipMiddleware(h)
	h = LoggingMiddleware(h)
	h = SecurityMiddleware(h)

	ms.sites[name] = &siteHandler{
		name:    name,
		store:   store,
		handler: h,
	}
	return nil
}

// ServeHTTP routes requests to the correct site handler.
func (ms *MultiSiteServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Path-prefix routing: /<sitename>/...
	for name, sh := range ms.sites {
		prefix := "/" + name
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			stripped := strings.TrimPrefix(path, prefix)
			if stripped == "" {
				stripped = "/"
			}
			r2 := r.Clone(r.Context())
			r2.URL.Path = stripped
			sh.handler.ServeHTTP(w, r2)
			return
		}
	}

	// Subdomain routing: sitename.hostname
	host := r.Host
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.Index(host, "."); idx != -1 {
		sub := host[:idx]
		if sh, ok := ms.sites[sub]; ok {
			sh.handler.ServeHTTP(w, r)
			return
		}
	}

	// Root: show site index.
	ms.serveSiteIndex(w, r)
}

func (ms *MultiSiteServer) serveSiteIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>Drafthaus — Sites</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 640px; margin: 4rem auto; padding: 0 1rem; color: #1a1a1a; }
    h1 { font-size: 1.5rem; margin-bottom: 1.5rem; }
    ul { list-style: none; padding: 0; }
    li { margin: 0.5rem 0; }
    a { color: #2563eb; text-decoration: none; font-size: 1.1rem; }
    a:hover { text-decoration: underline; }
  </style>
</head>
<body>
  <h1>Drafthaus Sites</h1>
  <ul>
`)
	for name := range ms.sites {
		fmt.Fprintf(w, "    <li><a href=\"/%s/\">%s</a></li>\n", name, name)
	}
	fmt.Fprintf(w, `  </ul>
</body>
</html>`)
}

// Start begins listening and serving all registered sites.
func (ms *MultiSiteServer) Start() error {
	addr := fmt.Sprintf("%s:%d", ms.host, ms.port)
	ms.httpServer = &http.Server{
		Addr:    addr,
		Handler: ms,
	}
	log.Printf("Drafthaus multi-site running on http://%s (%d sites)", addr, len(ms.sites))
	for name := range ms.sites {
		log.Printf("  → /%s/", name)
	}
	if err := ms.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (ms *MultiSiteServer) Shutdown(ctx context.Context) error {
	if ms.httpServer != nil {
		return ms.httpServer.Shutdown(ctx)
	}
	return nil
}

// Close closes all registered site stores.
func (ms *MultiSiteServer) Close() error {
	var lastErr error
	for _, sh := range ms.sites {
		if err := sh.store.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
