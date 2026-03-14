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
	"github.com/drafthaus/drafthaus/internal/plugins"
	"github.com/drafthaus/drafthaus/internal/render"
)

// Server wraps an http.Server with its dependencies.
type Server struct {
	httpServer *http.Server
	store      draft.Store
	pluginRT   *plugins.Runtime
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

	// Plugin runtime — non-fatal if it fails.
	ctx := context.Background()
	var pluginRT *plugins.Runtime
	var hookMgr *plugins.HookManager
	rt, err := plugins.NewRuntime(ctx)
	if err != nil {
		log.Printf("warn: plugin runtime unavailable: %v", err)
	} else {
		pluginRT = rt
		if loadErr := plugins.LoadFromStore(ctx, rt, store); loadErr != nil {
			log.Printf("warn: loading plugins from store: %v", loadErr)
		}
		hookMgr = plugins.NewHookManager(rt)
	}

	handlers := NewHandlers(store, resolver, pipeline, router, sessions, tracker, pluginRT, hookMgr)

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
		pluginRT:   pluginRT,
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

// Shutdown gracefully drains in-flight requests and closes the plugin runtime.
func (s *Server) Shutdown(ctx context.Context) error {
	if s.pluginRT != nil {
		if err := s.pluginRT.Close(ctx); err != nil {
			log.Printf("warn: close plugin runtime: %v", err)
		}
	}
	return s.httpServer.Shutdown(ctx)
}
