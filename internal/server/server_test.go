package server_test

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
	"github.com/drafthaus/drafthaus/internal/render"
	"github.com/drafthaus/drafthaus/internal/server"
)

func setupTestServer(t *testing.T) http.Handler {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.draft")
	store, err := draft.Open(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	// Create a blog post type + entity
	et := &draft.EntityType{
		Name: "BlogPost",
		Slug: "blog-posts",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "body", Type: draft.FieldRichText},
		},
		Routes: &draft.RouteConfig{List: "/blog", Detail: "/blog/{slug}"},
	}
	if err := store.CreateType(et); err != nil {
		t.Fatalf("create type: %v", err)
	}

	e := &draft.Entity{
		TypeID: et.ID,
		Data:   map[string]any{"title": "Hello World"},
		Slug:   "hello-world",
		Status: "published",
	}
	if err := store.CreateEntity(e); err != nil {
		t.Fatalf("create entity: %v", err)
	}

	store.SetTokens(&draft.TokenSet{
		Data: draft.Tokens{
			Colors: map[string]string{"primary": "#000", "text": "#111", "background": "#fff", "surface": "#f5f5f5", "border": "#ddd", "muted": "#888"},
			Fonts:  map[string]string{"body": "sans-serif", "heading": "sans-serif", "mono": "monospace"},
			Scale:  draft.ScaleTokens{Spacing: 1, Radius: "md", Density: "comfortable"},
		},
	})

	resolver := graph.NewResolver(store)
	pipeline := render.NewPipeline(store, resolver)
	router := server.NewRouter()
	router.BuildRoutes(store)
	sessions := server.NewSessionStore()
	handlers := server.NewHandlers(store, resolver, pipeline, router, sessions, nil)

	return handlers
}

func TestHomepage(t *testing.T) {
	h := setupTestServer(t)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "<!DOCTYPE html>") {
		t.Error("expected HTML document")
	}
	if !strings.Contains(body, "dh-site-nav") {
		t.Error("expected nav bar")
	}
}

func TestBlogList(t *testing.T) {
	h := setupTestServer(t)
	req := httptest.NewRequest("GET", "/blog", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Hello World") {
		t.Error("expected blog post title in list")
	}
}

func TestBlogDetail(t *testing.T) {
	h := setupTestServer(t)
	req := httptest.NewRequest("GET", "/blog/hello-world", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Hello World") {
		t.Error("expected blog post title in detail")
	}
}

func TestNotFound(t *testing.T) {
	h := setupTestServer(t)
	req := httptest.NewRequest("GET", "/nonexistent-page-xyz", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAPIUnauthorized(t *testing.T) {
	h := setupTestServer(t)
	req := httptest.NewRequest("GET", "/_api/types", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	// Without admin users, API should be accessible (first-run mode)
	// The test setup doesn't create admin users, so API is open
	if w.Code != 200 {
		t.Fatalf("expected 200 (no admin users = open access), got %d", w.Code)
	}
}

func TestCSSEndpoint(t *testing.T) {
	h := setupTestServer(t)
	req := httptest.NewRequest("GET", "/_dh/style.css", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/css") {
		t.Errorf("expected text/css, got %s", ct)
	}
	body := w.Body.String()
	if !strings.Contains(body, "--dh-color-primary") {
		t.Error("expected CSS custom properties")
	}
}
