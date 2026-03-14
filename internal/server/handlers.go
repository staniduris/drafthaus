package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/drafthaus/drafthaus/embed/admin"
	"github.com/drafthaus/drafthaus/internal/analytics"
	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
	"github.com/drafthaus/drafthaus/internal/plugins"
	"github.com/drafthaus/drafthaus/internal/projections"
	"github.com/drafthaus/drafthaus/internal/render"
)

// errNotFound signals a missing entity (should render 404, not 500).
type errNotFound struct{ err error }

func (e errNotFound) Error() string { return e.err.Error() }

// Handlers is the root HTTP handler for Drafthaus.
type Handlers struct {
	store    draft.Store
	resolver *graph.Resolver
	pipeline *render.Pipeline
	router   *Router
	sessions *SessionStore
	tracker  *analytics.Tracker
	pluginRT *plugins.Runtime
	hooks    *plugins.HookManager
}

// NewHandlers creates a Handlers wiring together all the dependencies.
func NewHandlers(store draft.Store, resolver *graph.Resolver, pipeline *render.Pipeline, router *Router, sessions *SessionStore, tracker *analytics.Tracker, pluginRT *plugins.Runtime, hooks *plugins.HookManager) *Handlers {
	return &Handlers{
		store:    store,
		resolver: resolver,
		pipeline: pipeline,
		router:   router,
		sessions: sessions,
		tracker:  tracker,
		pluginRT: pluginRT,
		hooks:    hooks,
	}
}

// ServeHTTP is the main dispatch method.
func (h *Handlers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/_health":
		HealthHandler(h.store)(w, r)
	case path == "/_admin/login":
		HandleLogin(h.store, h.sessions)(w, r)
	case path == "/_admin/logout":
		HandleLogout(h.sessions)(w, r)
	case path == "/_admin/setup":
		HandleSetup(h.store, h.sessions)(w, r)
	case path == "/_admin/analytics" || path == "/_admin/analytics/":
		h.serveAnalytics(w, r)
	case path == "/_api/analytics":
		h.serveAnalyticsAPI(w, r)
	case path == "/_admin" || path == "/_admin/":
		h.serveAdmin(w, r)
	case strings.HasPrefix(path, "/_api/"):
		h.apiDispatch(w, r)
	case strings.HasPrefix(path, "/_plugins/"):
		h.servePluginRoute(w, r)
	case strings.HasPrefix(path, "/_assets/"):
		h.serveAsset(w, r)
	case path == "/_dh/style.css":
		h.serveCSS(w, r)
	case path == "/feed.xml" || path == "/rss.xml":
		h.serveRSS(w, r)
	case path == "/sitemap.xml":
		h.serveSitemap(w, r)
	case strings.HasPrefix(path, "/api/v1/"):
		h.servePublicAPI(w, r)
	default:
		h.servePage(w, r)
	}
}

// -------------------------------------------------------------------------
// Page rendering
// -------------------------------------------------------------------------

func (h *Handlers) servePage(w http.ResponseWriter, r *http.Request) {
	m, ok := h.router.Match(r.URL.Path)
	if !ok {
		h.serve404(w)
		return
	}

	var (
		out []byte
		err error
	)

	switch m.Kind {
	case routeDetail:
		out, err = h.renderDetail(m)
	default:
		out, err = h.renderList(m)
	}

	if err != nil {
		if _, ok := err.(errNotFound); ok {
			h.serve404(w)
		} else {
			h.serveError(w, err)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(out) //nolint:errcheck
}

func (h *Handlers) renderDetail(m *Match) ([]byte, error) {
	entity, err := h.resolver.ResolveBySlug(m.TypeSlug, m.EntitySlug)
	if err != nil {
		return nil, errNotFound{err: err}
	}

	view, err := h.viewOrAuto(m.ViewName, entity.Type)
	if err != nil {
		return nil, err
	}

	return h.pipeline.RenderPage(entity, view)
}

func (h *Handlers) renderList(m *Match) ([]byte, error) {
	if m.ViewName == "Homepage" {
		view, err := h.viewOrAuto("Homepage", nil)
		if err != nil {
			return nil, err
		}
		// Load published entities for the homepage "each" binds.
		// Use the FIRST entity type with a list route — matches the Homepage view
		// which also picks the first type with a list route for its featured section.
		var homeEntities []*graph.ResolvedEntity
		types, _ := h.store.ListTypes()
		for _, t := range types {
			if t.Routes == nil || t.Routes.List == "" {
				continue
			}
			resolved, _, _ := h.resolver.ResolveList(t.Slug, graph.ListOpts{Status: "published", Limit: 10})
			if len(resolved) > 0 {
				homeEntities = resolved
				break
			}
		}
		return h.pipeline.RenderList(homeEntities, nil, view)
	}

	entities, _, err := h.resolver.ResolveList(m.TypeSlug, graph.ListOpts{Status: "published", Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("resolve list: %w", err)
	}

	var entityType *draft.EntityType
	if len(entities) > 0 {
		entityType = entities[0].Type
	} else {
		entityType, err = h.store.GetTypeBySlug(m.TypeSlug)
		if err != nil {
			return nil, fmt.Errorf("get type: %w", err)
		}
	}

	view, err := h.viewOrAuto(m.ViewName, entityType)
	if err != nil {
		return nil, err
	}

	return h.pipeline.RenderList(entities, entityType, view)
}

// viewOrAuto loads a view by name from the store, or generates a simple
// fallback view that lists all entity fields.
func (h *Handlers) viewOrAuto(name string, entityType *draft.EntityType) (*draft.View, error) {
	v, err := h.store.GetView(name)
	if err == nil {
		return v, nil
	}

	// Auto-generate a minimal view tree.
	tree := autoView(entityType)
	return &draft.View{
		Name:    name,
		Tree:    tree,
		Version: 1,
	}, nil
}

// autoView builds a minimal JSON view tree for an entity type.
func autoView(entityType *draft.EntityType) string {
	if entityType == nil {
		return `{"type":"Stack","props":{"gap":"md"},"children":[{"type":"Heading","props":{"level":1,"text":"Welcome"}}]}`
	}

	var children strings.Builder
	children.WriteString(`{"type":"Heading","props":{"level":1},"bind":{"text":"entity.data.title"}}`)
	for _, f := range entityType.Fields {
		if f.Name == "title" {
			continue
		}
		children.WriteString(fmt.Sprintf(
			`,{"type":"Text","bind":{"text":"entity.data.%s"}}`,
			f.Name,
		))
	}

	return fmt.Sprintf(
		`{"type":"Stack","props":{"gap":"md"},"children":[%s]}`,
		children.String(),
	)
}

func (h *Handlers) serveAsset(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/_assets/")
	if id == "" {
		http.NotFound(w, r)
		return
	}

	asset, err := h.store.GetAsset(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", asset.Mime)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Write(asset.Data) //nolint:errcheck
}

func (h *Handlers) serveCSS(w http.ResponseWriter, r *http.Request) {
	ts, err := h.store.GetTokens()
	var tokens *draft.Tokens
	if err == nil {
		tokens = &ts.Data
	}

	css := render.GenerateCSS(tokens)
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, css)
}

func (h *Handlers) serveAdmin(w http.ResponseWriter, r *http.Request) {
	data, err := admin.FS.ReadFile("index.html")
	if err != nil {
		h.serveError(w, fmt.Errorf("admin UI not found: %w", err))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(data) //nolint:errcheck
}

func (h *Handlers) serve404(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, page404HTML)
}

func (h *Handlers) serveError(w http.ResponseWriter, err error) {
	log.Printf("ERROR: %v", err)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, page500HTML)
}

const page404HTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Not Found</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb}
.box{text-align:center;padding:2rem}h1{font-size:4rem;margin:0;color:#e5e7eb}p{color:#6b7280}</style>
</head><body><div class="box"><h1>404</h1><p>Page not found.</p></div></body></html>`

const page500HTML = `<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>Error</title>
<style>body{font-family:sans-serif;display:flex;align-items:center;justify-content:center;min-height:100vh;margin:0;background:#f9fafb}
.box{text-align:center;padding:2rem}h1{font-size:2rem;margin:0;color:#dc2626}p{color:#6b7280}</style>
</head><body><div class="box"><h1>Internal Server Error</h1><p>Something went wrong. Please try again later.</p></div></body></html>`

// -------------------------------------------------------------------------
// Projection handlers (RSS, sitemap, public API)
// -------------------------------------------------------------------------

func (h *Handlers) serveRSS(w http.ResponseWriter, r *http.Request) {
	types, err := h.store.ListTypes()
	if err != nil {
		h.serveError(w, fmt.Errorf("list types: %w", err))
		return
	}

	// Find best type: prefer one with published_at, then any type with a detail route.
	var chosen *draft.EntityType
	for _, t := range types {
		if t.Routes == nil || t.Routes.Detail == "" {
			continue
		}
		for _, f := range t.Fields {
			if f.Name == "published_at" {
				chosen = t
				break
			}
		}
		if chosen != nil {
			break
		}
	}
	if chosen == nil {
		for _, t := range types {
			if t.Routes != nil && t.Routes.Detail != "" {
				chosen = t
				break
			}
		}
	}
	if chosen == nil {
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		w.Write([]byte("<?xml version=\"1.0\" encoding=\"utf-8\"?><rss version=\"2.0\"><channel><title>Feed</title></channel></rss>")) //nolint:errcheck
		return
	}

	entities, _, err := h.resolver.ResolveList(chosen.Slug, graph.ListOpts{
		Status:  "published",
		Limit:   20,
		OrderBy: "position",
		Order:   "asc",
	})
	if err != nil {
		h.serveError(w, fmt.Errorf("resolve list: %w", err))
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + r.Host

	siteName := ""
	if ts, tsErr := h.store.GetTokens(); tsErr == nil {
		siteName = ts.Data.SiteName
	}

	out, err := projections.GenerateRSS(entities, chosen, baseURL, siteName)
	if err != nil {
		h.serveError(w, fmt.Errorf("generate rss: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Write(out) //nolint:errcheck
}

func (h *Handlers) serveSitemap(w http.ResponseWriter, r *http.Request) {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	baseURL := scheme + "://" + r.Host

	out, err := projections.GenerateSitemap(h.store, h.resolver, baseURL)
	if err != nil {
		h.serveError(w, fmt.Errorf("generate sitemap: %w", err))
		return
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(out) //nolint:errcheck
}

func (h *Handlers) servePublicAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "public, max-age=60")

	if r.Method == http.MethodOptions {
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if r.Method != http.MethodGet {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Strip /api/v1/content/ prefix.
	sub := strings.TrimPrefix(r.URL.Path, "/api/v1/content/")
	sub = strings.TrimRight(sub, "/")
	parts := splitPath(sub)

	if len(parts) == 0 {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	typeSlug := parts[0]

	if len(parts) == 1 {
		// List entities of given type (published only).
		entities, total, err := h.resolver.ResolveList(typeSlug, graph.ListOpts{
			Status: "published",
			Limit:  100,
		})
		if err != nil {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projections.EntityListToJSON(entities, total)) //nolint:errcheck
		return
	}

	if len(parts) == 2 {
		entitySlug := parts[1]
		re, err := h.resolver.ResolveBySlug(typeSlug, entitySlug)
		if err != nil {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		if re.Entity.Status != "published" {
			jsonError(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(projections.EntityToJSON(re)) //nolint:errcheck
		return
	}

	jsonError(w, "not found", http.StatusNotFound)
}

// -------------------------------------------------------------------------
// API dispatch
// -------------------------------------------------------------------------

func (h *Handlers) apiDispatch(w http.ResponseWriter, r *http.Request) {
	// CSRF check: non-GET/OPTIONS requests must include X-Requested-With header,
	// unless they are multipart form uploads (which cannot set custom headers).
	if r.Method != http.MethodGet && r.Method != http.MethodOptions {
		if r.Header.Get("X-Requested-With") != "XMLHttpRequest" {
			ct := r.Header.Get("Content-Type")
			if !strings.HasPrefix(ct, "multipart/form-data") {
				jsonError(w, "CSRF validation failed — include X-Requested-With: XMLHttpRequest header", http.StatusForbidden)
				return
			}
		}
	}

	// Strip "/_api/" prefix.
	sub := strings.TrimPrefix(r.URL.Path, "/_api/")
	sub = strings.TrimRight(sub, "/")
	parts := splitPath(sub)

	if len(parts) == 0 {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	switch parts[0] {
	case "types":
		h.dispatchTypes(w, r, parts[1:])
	case "entities":
		h.dispatchEntities(w, r, parts[1:])
	case "relations":
		h.dispatchRelations(w, r)
	case "assets":
		h.dispatchAssets(w, r)
	case "views":
		h.dispatchViews(w, r, parts[1:])
	case "tokens":
		h.dispatchTokens(w, r)
	case "plugins":
		h.dispatchPlugins(w, r, parts[1:])
	default:
		jsonError(w, "not found", http.StatusNotFound)
	}
}

// splitPath splits a "/" separated path, removing empty segments.
func splitPath(p string) []string {
	raw := strings.Split(p, "/")
	out := raw[:0]
	for _, s := range raw {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// -------------------------------------------------------------------------
// Type API
// -------------------------------------------------------------------------

func (h *Handlers) dispatchTypes(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 0 && r.Method == http.MethodGet:
		h.listTypesAPI(w, r)
	case len(parts) == 0 && r.Method == http.MethodPost:
		h.createTypeAPI(w, r)
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.getTypeAPI(w, r, parts[0])
	case len(parts) == 1 && r.Method == http.MethodPut:
		h.updateTypeAPI(w, r, parts[0])
	case len(parts) == 1 && r.Method == http.MethodDelete:
		h.deleteTypeAPI(w, r, parts[0])
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) listTypesAPI(w http.ResponseWriter, r *http.Request) {
	types, err := h.store.ListTypes()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, types)
}

func (h *Handlers) createTypeAPI(w http.ResponseWriter, r *http.Request) {
	var t draft.EntityType
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.CreateType(&t); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&t) //nolint:errcheck
}

func (h *Handlers) getTypeAPI(w http.ResponseWriter, r *http.Request, slug string) {
	t, err := h.store.GetTypeBySlug(slug)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	jsonOK(w, t)
}

func (h *Handlers) updateTypeAPI(w http.ResponseWriter, r *http.Request, slug string) {
	existing, err := h.store.GetTypeBySlug(slug)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}

	var t draft.EntityType
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	t.ID = existing.ID
	t.UpdatedAt = time.Now().Unix()

	if err := h.store.UpdateType(&t); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, &t)
}

func (h *Handlers) deleteTypeAPI(w http.ResponseWriter, r *http.Request, slug string) {
	t, err := h.store.GetTypeBySlug(slug)
	if err != nil {
		jsonError(w, "not found", http.StatusNotFound)
		return
	}
	if err := h.store.DeleteType(t.ID); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -------------------------------------------------------------------------
// Entity API
// -------------------------------------------------------------------------

func (h *Handlers) dispatchEntities(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		h.listEntitiesAPI(w, r, parts[0])
	case len(parts) == 1 && r.Method == http.MethodPost:
		h.createEntityAPI(w, r, parts[0])
	case len(parts) == 2 && r.Method == http.MethodGet:
		h.getEntityAPI(w, r, parts[0], parts[1])
	case len(parts) == 2 && r.Method == http.MethodPut:
		h.updateEntityAPI(w, r, parts[0], parts[1])
	case len(parts) == 2 && r.Method == http.MethodDelete:
		h.deleteEntityAPI(w, r, parts[0], parts[1])
	case len(parts) == 4 && parts[2] == "blocks" && r.Method == http.MethodGet:
		h.getBlocksAPI(w, r, parts[0], parts[1], parts[3])
	case len(parts) == 4 && parts[2] == "blocks" && r.Method == http.MethodPut:
		h.setBlocksAPI(w, r, parts[0], parts[1], parts[3])
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) listEntitiesAPI(w http.ResponseWriter, r *http.Request, typeSlug string) {
	t, err := h.store.GetTypeBySlug(typeSlug)
	if err != nil {
		jsonError(w, "type not found", http.StatusNotFound)
		return
	}
	entities, total, err := h.store.ListEntities(t.ID, draft.ListOpts{Limit: 100})
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, map[string]any{"entities": entities, "total": total})
}

func (h *Handlers) createEntityAPI(w http.ResponseWriter, r *http.Request, typeSlug string) {
	t, err := h.store.GetTypeBySlug(typeSlug)
	if err != nil {
		jsonError(w, "type not found", http.StatusNotFound)
		return
	}

	var e draft.Entity
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	e.TypeID = t.ID

	if err := h.store.CreateEntity(&e); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&e) //nolint:errcheck
}

func (h *Handlers) getEntityAPI(w http.ResponseWriter, r *http.Request, typeSlug, entitySlug string) {
	t, err := h.store.GetTypeBySlug(typeSlug)
	if err != nil {
		jsonError(w, "type not found", http.StatusNotFound)
		return
	}
	e, err := h.store.GetEntityBySlug(t.ID, entitySlug)
	if err != nil {
		jsonError(w, "entity not found", http.StatusNotFound)
		return
	}
	jsonOK(w, e)
}

func (h *Handlers) updateEntityAPI(w http.ResponseWriter, r *http.Request, typeSlug, entitySlug string) {
	t, err := h.store.GetTypeBySlug(typeSlug)
	if err != nil {
		jsonError(w, "type not found", http.StatusNotFound)
		return
	}
	existing, err := h.store.GetEntityBySlug(t.ID, entitySlug)
	if err != nil {
		jsonError(w, "entity not found", http.StatusNotFound)
		return
	}

	var e draft.Entity
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	e.ID = existing.ID
	e.TypeID = t.ID
	e.UpdatedAt = time.Now().Unix()

	if err := h.store.UpdateEntity(&e); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, &e)
}

func (h *Handlers) deleteEntityAPI(w http.ResponseWriter, r *http.Request, typeSlug, entitySlug string) {
	t, err := h.store.GetTypeBySlug(typeSlug)
	if err != nil {
		jsonError(w, "type not found", http.StatusNotFound)
		return
	}
	e, err := h.store.GetEntityBySlug(t.ID, entitySlug)
	if err != nil {
		jsonError(w, "entity not found", http.StatusNotFound)
		return
	}
	if err := h.store.DeleteEntity(e.ID); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) getBlocksAPI(w http.ResponseWriter, r *http.Request, typeSlug, entitySlug, field string) {
	t, err := h.store.GetTypeBySlug(typeSlug)
	if err != nil {
		jsonError(w, "type not found", http.StatusNotFound)
		return
	}
	e, err := h.store.GetEntityBySlug(t.ID, entitySlug)
	if err != nil {
		jsonError(w, "entity not found", http.StatusNotFound)
		return
	}
	blocks, err := h.store.GetBlocks(e.ID, field)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, blocks)
}

func (h *Handlers) setBlocksAPI(w http.ResponseWriter, r *http.Request, typeSlug, entitySlug, field string) {
	t, err := h.store.GetTypeBySlug(typeSlug)
	if err != nil {
		jsonError(w, "type not found", http.StatusNotFound)
		return
	}
	e, err := h.store.GetEntityBySlug(t.ID, entitySlug)
	if err != nil {
		jsonError(w, "entity not found", http.StatusNotFound)
		return
	}

	var blocks []*draft.Block
	if err := json.NewDecoder(r.Body).Decode(&blocks); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.SetBlocks(e.ID, field, blocks); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, blocks)
}

// -------------------------------------------------------------------------
// Relations API
// -------------------------------------------------------------------------

func (h *Handlers) dispatchRelations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		h.addRelationAPI(w, r)
	case http.MethodDelete:
		h.removeRelationAPI(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) addRelationAPI(w http.ResponseWriter, r *http.Request) {
	var rel draft.Relation
	if err := json.NewDecoder(r.Body).Decode(&rel); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.AddRelation(&rel); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(&rel) //nolint:errcheck
}

func (h *Handlers) removeRelationAPI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SourceID     string `json:"source_id"`
		TargetID     string `json:"target_id"`
		RelationType string `json:"relation_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.store.RemoveRelation(req.SourceID, req.TargetID, req.RelationType); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// -------------------------------------------------------------------------
// Assets API
// -------------------------------------------------------------------------

func (h *Handlers) dispatchAssets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.uploadAssetAPI(w, r)
}

func (h *Handlers) uploadAssetAPI(w http.ResponseWriter, r *http.Request) {
	const maxSize = 50 << 20 // 50 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	ct := r.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil || mediaType != "multipart/form-data" {
		jsonError(w, "expected multipart/form-data", http.StatusBadRequest)
		return
	}

	mr := multipart.NewReader(r.Body, extractBoundary(ct))
	part, err := mr.NextPart()
	if err != nil {
		jsonError(w, "reading multipart: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer part.Close()

	data, err := io.ReadAll(part)
	if err != nil {
		jsonError(w, "reading file: "+err.Error(), http.StatusBadRequest)
		return
	}

	filename := part.FileName()
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	asset := &draft.Asset{
		Name:      filename,
		Mime:      mimeType,
		Size:      int64(len(data)),
		Data:      data,
		CreatedAt: time.Now().Unix(),
	}

	if err := h.store.StoreAsset(asset); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(asset) //nolint:errcheck
}

// extractBoundary pulls the boundary parameter out of a Content-Type value.
func extractBoundary(ct string) string {
	_, params, err := mime.ParseMediaType(ct)
	if err != nil {
		return ""
	}
	return params["boundary"]
}

// -------------------------------------------------------------------------
// Views API
// -------------------------------------------------------------------------

func (h *Handlers) dispatchViews(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 0 && r.Method == http.MethodGet:
		h.listViewsAPI(w, r)
	case len(parts) == 1 && r.Method == http.MethodPut:
		h.setViewAPI(w, r, parts[0])
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) listViewsAPI(w http.ResponseWriter, r *http.Request) {
	views, err := h.store.ListViews()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, views)
}

func (h *Handlers) setViewAPI(w http.ResponseWriter, r *http.Request, name string) {
	var v draft.View
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	v.Name = name
	v.UpdatedAt = time.Now().Unix()

	if err := h.store.SetView(&v); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, &v)
}

// -------------------------------------------------------------------------
// Tokens API
// -------------------------------------------------------------------------

func (h *Handlers) dispatchTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.getTokensAPI(w, r)
	case http.MethodPut:
		h.setTokensAPI(w, r)
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) getTokensAPI(w http.ResponseWriter, r *http.Request) {
	ts, err := h.store.GetTokens()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, ts)
}

func (h *Handlers) setTokensAPI(w http.ResponseWriter, r *http.Request) {
	var ts draft.TokenSet
	if err := json.NewDecoder(r.Body).Decode(&ts); err != nil {
		jsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	ts.UpdatedAt = time.Now().Unix()

	if err := h.store.SetTokens(&ts); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, &ts)
}

// -------------------------------------------------------------------------
// Analytics
// -------------------------------------------------------------------------

func (h *Handlers) serveAnalyticsAPI(w http.ResponseWriter, r *http.Request) {
	if h.tracker == nil {
		jsonError(w, "analytics not available", http.StatusServiceUnavailable)
		return
	}
	stats, err := h.tracker.Stats(30)
	if err != nil {
		jsonError(w, fmt.Sprintf("stats error: %s", err), http.StatusInternalServerError)
		return
	}
	jsonOK(w, stats)
}

func (h *Handlers) serveAnalytics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	fmt.Fprint(w, analyticsDashboardHTML)
}

const analyticsDashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Analytics - Drafthaus</title>
<style>
*,*::before,*::after{box-sizing:border-box}
body{margin:0;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#0f172a;color:#e2e8f0;min-height:100vh;display:flex}
.sidebar{width:14rem;background:#1e293b;padding:1.5rem 1rem;flex-shrink:0;display:flex;flex-direction:column;gap:0.25rem}
.brand{font-size:1.1rem;font-weight:700;letter-spacing:-0.02em;padding:0.5rem 0.75rem;margin-bottom:0.75rem;color:#f1f5f9}
.brand span{opacity:.4}
.nav-link{display:block;padding:0.5rem 0.75rem;border-radius:0.5rem;color:#94a3b8;text-decoration:none;font-size:0.875rem;transition:background .15s,color .15s}
.nav-link:hover{background:#334155;color:#f1f5f9}
.nav-link.active{background:#334155;color:#f1f5f9}
.main{flex:1;padding:2rem;overflow:auto}
h1{font-size:1.5rem;font-weight:700;margin:0 0 0.25rem;color:#f1f5f9}
.subtitle{color:#64748b;font-size:0.875rem;margin-bottom:2rem}
.metrics{display:grid;grid-template-columns:repeat(auto-fit,minmax(10rem,1fr));gap:1rem;margin-bottom:2rem}
.metric{background:#1e293b;border-radius:0.75rem;padding:1.25rem}
.metric-label{font-size:0.75rem;text-transform:uppercase;letter-spacing:.05em;color:#64748b;margin-bottom:0.5rem}
.metric-value{font-size:2rem;font-weight:700;color:#f1f5f9;line-height:1}
.section{background:#1e293b;border-radius:0.75rem;padding:1.5rem;margin-bottom:1.5rem}
.section-title{font-size:0.875rem;font-weight:600;text-transform:uppercase;letter-spacing:.05em;color:#64748b;margin:0 0 1rem}
.chart{display:flex;align-items:flex-end;gap:4px;height:8rem}
.bar-wrap{flex:1;display:flex;flex-direction:column;align-items:center;gap:4px;height:100%}
.bar{width:100%;background:#6366f1;border-radius:3px 3px 0 0;transition:opacity .15s;min-height:2px}
.bar:hover{opacity:.75}
.bar-label{font-size:0.6rem;color:#475569;transform:rotate(-45deg);white-space:nowrap}
table{width:100%;border-collapse:collapse}
th{text-align:left;font-size:0.75rem;text-transform:uppercase;letter-spacing:.05em;color:#64748b;padding:0.5rem 0;border-bottom:1px solid #334155}
td{padding:0.625rem 0;border-bottom:1px solid #1e293b;font-size:0.875rem;color:#cbd5e1}
td:last-child{text-align:right;color:#94a3b8}
.loading{color:#475569;text-align:center;padding:3rem}
.two-col{display:grid;grid-template-columns:1fr 1fr;gap:1.5rem}
@media(max-width:640px){.two-col{grid-template-columns:1fr}.sidebar{display:none}}
</style>
</head>
<body>
<nav class="sidebar">
  <div class="brand">Drafthaus <span>admin</span></div>
  <a href="/_admin" class="nav-link">Dashboard</a>
  <a href="/_admin/analytics" class="nav-link active">Analytics</a>
</nav>
<main class="main">
  <h1>Analytics</h1>
  <p class="subtitle">Last 30 days - cookieless, privacy-first</p>
  <div id="root"><p class="loading">Loading...</p></div>
</main>
<script>
(function(){
  function el(tag,attrs,children){
    var e=document.createElement(tag);
    Object.keys(attrs||{}).forEach(function(k){e[k]=attrs[k];});
    (children||[]).forEach(function(c){
      e.appendChild(typeof c==='string'?document.createTextNode(c):c);
    });
    return e;
  }
  function clearNode(n){while(n.firstChild){n.removeChild(n.firstChild);}}
  function render(data){
    var root=document.getElementById('root');
    clearNode(root);
    root.appendChild(el('div',{className:'metrics'},[
      metricCard('Total Views',data.total_views),
      metricCard('Unique Visitors',data.unique_visitors)
    ]));
    root.appendChild(buildChart(data.views_by_day||[]));
    root.appendChild(el('div',{className:'two-col'},[
      buildTable('Top Pages',['Page','Views'],data.top_pages||[],function(r){return [r.path,r.views];}),
      buildTable('Top Referrers',['Referrer','Views'],data.top_referrers||[],function(r){return [r.referrer,r.views];})
    ]));
  }
  function metricCard(label,value){
    return el('div',{className:'metric'},[
      el('div',{className:'metric-label'},[label]),
      el('div',{className:'metric-value'},[(value||0).toLocaleString()])
    ]);
  }
  function buildChart(days){
    var section=el('div',{className:'section'},[el('div',{className:'section-title'},['Views per day'])]);
    if(!days.length){section.appendChild(el('p',{className:'loading'},['No data yet']));return section;}
    var max=Math.max.apply(null,days.map(function(d){return d.views;}));
    var chart=el('div',{className:'chart'});
    days.slice(-14).forEach(function(d){
      var pct=max>0?Math.round((d.views/max)*100):0;
      var bar=el('div',{className:'bar',title:d.date+': '+d.views+' views'});
      bar.style.height=Math.max(pct,2)+'%';
      chart.appendChild(el('div',{className:'bar-wrap'},[bar,el('div',{className:'bar-label'},[d.date.slice(5)])]));
    });
    section.appendChild(chart);
    return section;
  }
  function buildTable(title,headers,rows,accessor){
    var section=el('div',{className:'section'},[el('div',{className:'section-title'},[title])]);
    if(!rows.length){section.appendChild(el('p',{className:'loading'},['No data yet']));return section;}
    var thead=el('thead',{},[el('tr',{},headers.map(function(h){return el('th',{},[h]);}))]);
    var tbody=el('tbody',{},rows.map(function(r){
      return el('tr',{},accessor(r).map(function(c){return el('td',{},[String(c)]);}));
    }));
    section.appendChild(el('table',{},[thead,tbody]));
    return section;
  }
  function showError(){
    var root=document.getElementById('root');
    clearNode(root);
    root.appendChild(el('p',{className:'loading'},['Failed to load analytics.']));
  }
  function load(){
    fetch('/_api/analytics').then(function(r){return r.json();}).then(render).catch(showError);
  }
  load();
  setInterval(load,60000);
})();
</script>
</body>
</html>`

// -------------------------------------------------------------------------
// Plugin API
// -------------------------------------------------------------------------

// dispatchPlugins routes /_api/plugins/* requests.
//
//	GET    /_api/plugins              → list plugins
//	POST   /_api/plugins              → upload plugin (multipart: name, wasm file, config JSON)
//	DELETE /_api/plugins/:name        → delete plugin
//	POST   /_api/plugins/:name/hooks  → register hook
//	GET    /_api/plugins/hooks/:type  → list hooks for type
func (h *Handlers) dispatchPlugins(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 0 && r.Method == http.MethodGet:
		h.listPluginsAPI(w, r)
	case len(parts) == 0 && r.Method == http.MethodPost:
		h.uploadPluginAPI(w, r)
	case len(parts) == 1 && r.Method == http.MethodDelete:
		h.deletePluginAPI(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "hooks" && r.Method == http.MethodPost:
		h.registerHookAPI(w, r, parts[0])
	case len(parts) == 2 && parts[0] == "hooks" && r.Method == http.MethodGet:
		h.listHooksAPI(w, r, parts[1])
	default:
		jsonError(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handlers) listPluginsAPI(w http.ResponseWriter, r *http.Request) {
	if h.pluginRT == nil {
		jsonOK(w, []*plugins.Plugin{})
		return
	}
	jsonOK(w, h.pluginRT.ListPlugins())
}

func (h *Handlers) uploadPluginAPI(w http.ResponseWriter, r *http.Request) {
	if h.pluginRT == nil {
		jsonError(w, "plugin runtime not available", http.StatusServiceUnavailable)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		jsonError(w, "parse multipart: "+err.Error(), http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	if name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}

	f, _, err := r.FormFile("wasm")
	if err != nil {
		jsonError(w, "wasm file required: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	wasmBytes, err := io.ReadAll(f)
	if err != nil {
		jsonError(w, "read wasm: "+err.Error(), http.StatusInternalServerError)
		return
	}

	configRaw := r.FormValue("config")
	var config map[string]any
	if configRaw != "" {
		if err := json.Unmarshal([]byte(configRaw), &config); err != nil {
			jsonError(w, "invalid config JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	if config == nil {
		config = map[string]any{}
	}

	p, err := h.pluginRT.LoadPlugin(r.Context(), name, wasmBytes)
	if err != nil {
		jsonError(w, "load plugin: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.store.SavePlugin(name, p.Version, wasmBytes, config); err != nil {
		jsonError(w, "save plugin: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p) //nolint:errcheck
}

func (h *Handlers) deletePluginAPI(w http.ResponseWriter, r *http.Request, name string) {
	if h.pluginRT == nil {
		jsonError(w, "plugin runtime not available", http.StatusServiceUnavailable)
		return
	}

	if err := h.store.DeletePlugin(name); err != nil {
		jsonError(w, err.Error(), http.StatusNotFound)
		return
	}

	// Unload from runtime — best effort (plugin may not be currently loaded).
	h.pluginRT.UnloadPlugin(r.Context(), name) //nolint:errcheck

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) registerHookAPI(w http.ResponseWriter, r *http.Request, pluginName string) {
	if h.hooks == nil {
		jsonError(w, "plugin runtime not available", http.StatusServiceUnavailable)
		return
	}

	var reg plugins.HookRegistration
	if err := json.NewDecoder(r.Body).Decode(&reg); err != nil {
		jsonError(w, "invalid body: "+err.Error(), http.StatusBadRequest)
		return
	}
	reg.PluginName = pluginName

	if err := h.hooks.Register(reg); err != nil {
		jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.store.SaveHook(reg.PluginName, string(reg.Hook), reg.Function, reg.Route); err != nil {
		jsonError(w, "persist hook: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reg) //nolint:errcheck
}

func (h *Handlers) listHooksAPI(w http.ResponseWriter, r *http.Request, hookType string) {
	recs, err := h.store.ListHooks(hookType)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, recs)
}

// servePluginRoute handles /_plugins/* — delegates to on_request hooks.
func (h *Handlers) servePluginRoute(w http.ResponseWriter, r *http.Request) {
	if h.hooks == nil {
		http.NotFound(w, r)
		return
	}

	input, err := json.Marshal(map[string]string{
		"method": r.Method,
		"path":   r.URL.Path,
		"query":  r.URL.RawQuery,
	})
	if err != nil {
		h.serveError(w, err)
		return
	}

	result, err := h.hooks.RunHooks(r.Context(), plugins.HookOnRequest, input)
	if err != nil {
		h.serveError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(result) //nolint:errcheck
}

// -------------------------------------------------------------------------
// Helpers
// -------------------------------------------------------------------------

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg}) //nolint:errcheck
}
