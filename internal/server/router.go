package server

import (
	"sort"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
)

type routeKind int

const (
	routeList   routeKind = iota
	routeDetail
)

type route struct {
	pattern  string    // e.g. "/blog" or "/blog/{slug}"
	kind     routeKind
	typeSlug string    // entity type slug
	viewName string    // e.g. "BlogPost.list" or "BlogPost.detail"
}

// Match is the result of a successful route match.
type Match struct {
	TypeSlug   string
	EntitySlug string
	ViewName   string
	Kind       routeKind
}

// Router maps URL paths to entity types and views.
type Router struct {
	routes []route
}

// NewRouter creates an empty Router.
func NewRouter() *Router {
	return &Router{}
}

// BuildRoutes reads all entity types from the store and registers routes
// for any type that has a RouteConfig.
func (r *Router) BuildRoutes(store draft.Store) error {
	types, err := store.ListTypes()
	if err != nil {
		return err
	}

	for _, t := range types {
		if t.Routes == nil {
			continue
		}
		if t.Routes.List != "" {
			r.routes = append(r.routes, route{
				pattern:  normalizePath(t.Routes.List),
				kind:     routeList,
				typeSlug: t.Slug,
				viewName: t.Name + ".list",
			})
		}
		if t.Routes.Detail != "" {
			r.routes = append(r.routes, route{
				pattern:  normalizePath(t.Routes.Detail),
				kind:     routeDetail,
				typeSlug: t.Slug,
				viewName: t.Name + ".detail",
			})
		}
	}

	// Sort routes: exact list routes first, then detail routes with fewer segments,
	// then catch-all patterns like /{slug} last. This ensures /work matches the
	// Project list route before the Page detail route /{slug}.
	sort.SliceStable(r.routes, func(i, j int) bool {
		ri, rj := r.routes[i], r.routes[j]
		// List routes before detail routes
		if ri.kind != rj.kind {
			return ri.kind < rj.kind // routeList=0 < routeDetail=1
		}
		// Among detail routes, more specific patterns (more segments) first,
		// catch-all single-segment patterns (/{slug}) last
		si := strings.Count(ri.pattern, "/")
		sj := strings.Count(rj.pattern, "/")
		return si > sj
	})

	return nil
}

// Match returns the best match for the given path, or false if none found.
// "/" is a special case: if no route explicitly covers it, a Homepage match
// is returned.
func (r *Router) Match(path string) (*Match, bool) {
	path = normalizePath(path)

	for _, rt := range r.routes {
		switch rt.kind {
		case routeList:
			if path == rt.pattern {
				return &Match{
					TypeSlug: rt.typeSlug,
					ViewName: rt.viewName,
					Kind:     routeList,
				}, true
			}
		case routeDetail:
			if slug, ok := matchDetail(rt.pattern, path); ok {
				return &Match{
					TypeSlug:   rt.typeSlug,
					EntitySlug: slug,
					ViewName:   rt.viewName,
					Kind:       routeDetail,
				}, true
			}
		}
	}

	// Special root path fallback.
	if path == "/" {
		return &Match{
			ViewName: "Homepage",
			Kind:     routeList,
		}, true
	}

	return nil, false
}

// normalizePath strips a trailing slash unless the path is "/".
func normalizePath(p string) string {
	if p == "" {
		return "/"
	}
	if p != "/" {
		p = strings.TrimRight(p, "/")
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}

// matchDetail checks whether path matches a detail pattern that contains a
// {slug} segment, and returns the captured slug value.
func matchDetail(pattern, path string) (string, bool) {
	patternParts := strings.Split(strings.TrimPrefix(pattern, "/"), "/")
	pathParts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	if len(patternParts) != len(pathParts) {
		return "", false
	}

	slug := ""
	for i, pp := range patternParts {
		if pp == "{slug}" {
			slug = pathParts[i]
		} else if pp != pathParts[i] {
			return "", false
		}
	}

	if slug == "" {
		return "", false
	}

	return slug, true
}
