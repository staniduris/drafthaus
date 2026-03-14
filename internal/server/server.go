package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/drafthaus/drafthaus/internal/analytics"
	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
	"github.com/drafthaus/drafthaus/internal/render"
)

// Server wraps an http.Server with its dependencies.
type Server struct {
	httpServer *http.Server
	store      draft.Store
}

// New creates a Server, wiring together all layers.
func New(store draft.Store, host string, port int) (*Server, error) {
	resolver := graph.NewResolver(store)
	pipeline := render.NewPipeline(store, resolver)

	router := NewRouter()
	if err := router.BuildRoutes(store); err != nil {
		return nil, fmt.Errorf("build routes: %w", err)
	}

	sessions := NewSessionStore()

	var tracker *analytics.Tracker
	if sqliteStore, ok := store.(*draft.SQLiteStore); ok {
		tracker = analytics.NewTracker(sqliteStore.DB())
	}

	handlers := NewHandlers(store, resolver, pipeline, router, sessions, tracker)

	// Middleware chain: Security → Logging → Gzip → CORS → Auth → Analytics → Handlers
	var h http.Handler = handlers
	if tracker != nil {
		h = analytics.Middleware(tracker)(h)
	}
	h = AuthMiddleware(store, sessions)(h)
	h = CORSMiddleware(h)
	h = GzipMiddleware(h)
	h = LoggingMiddleware(h)
	h = SecurityMiddleware(h)

	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Handler:      h,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: srv,
		store:      store,
	}, nil
}

// Start begins serving HTTP traffic. It returns nil only when the server
// closes gracefully; any other error is propagated.
func (s *Server) Start() error {
	log.Printf("Drafthaus running on http://%s", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully drains in-flight requests.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
