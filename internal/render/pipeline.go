package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/embed/tailwind"
	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
)

// Pipeline ties together token loading, data binding, CSS generation, and HTML rendering.
type Pipeline struct {
	registry *Registry
	resolver *graph.Resolver
	store    draft.Store
}

// NewPipeline creates a Pipeline with all built-in primitives registered.
func NewPipeline(store draft.Store, resolver *graph.Resolver) *Pipeline {
	reg := NewRegistry()
	RegisterAll(reg)
	return &Pipeline{
		registry: reg,
		resolver: resolver,
		store:    store,
	}
}

// RenderPage renders a single entity using the given view into a complete HTML document.
func (p *Pipeline) RenderPage(entity *graph.ResolvedEntity, view *draft.View) ([]byte, error) {
	tokens, err := p.loadTokens()
	if err != nil {
		return nil, fmt.Errorf("load tokens: %w", err)
	}

	root, err := parseViewTree(view.Tree)
	if err != nil {
		return nil, fmt.Errorf("parse view tree: %w", err)
	}

	bindCtx := &BindContext{
		Entity: entity,
		Tokens: tokens,
	}
	bound, err := BindTree(root, bindCtx)
	if err != nil {
		return nil, fmt.Errorf("bind tree: %w", err)
	}

	renderCtx := &RenderContext{
		Entity:   entity,
		Tokens:   tokens,
		Registry: p.registry,
	}
	body, err := renderCtx.Render(bound)
	if err != nil {
		return nil, fmt.Errorf("render tree: %w", err)
	}

	nav, err := p.buildNav(tokens)
	if err != nil {
		nav = "" // non-fatal: degrade gracefully
	}

	css := GenerateCSS(tokens)
	meta := GenerateMeta(entity, tokens, "")

	return []byte(buildHTMLDoc(meta, css, nav+body, tokens)), nil
}

// RenderList renders a list of entities using the given view into a complete HTML document.
// The view tree handles iteration via "each" bind expressions.
func (p *Pipeline) RenderList(entities []*graph.ResolvedEntity, entityType *draft.EntityType, view *draft.View) ([]byte, error) {
	tokens, err := p.loadTokens()
	if err != nil {
		return nil, fmt.Errorf("load tokens: %w", err)
	}

	root, err := parseViewTree(view.Tree)
	if err != nil {
		return nil, fmt.Errorf("parse view tree: %w", err)
	}

	// For list views the primary entity is nil; individual entities are in Entities.
	bindCtx := &BindContext{
		Entities: entities,
		Tokens:   tokens,
	}
	if len(entities) > 0 {
		// Provide the first entity as the default Entity so top-level binds
		// (e.g. page title) can reference list-level data.
		bindCtx.Entity = entities[0]
	}

	bound, err := BindTree(root, bindCtx)
	if err != nil {
		return nil, fmt.Errorf("bind tree: %w", err)
	}

	renderCtx := &RenderContext{
		Entities: entities,
		Tokens:   tokens,
		Registry: p.registry,
	}
	if len(entities) > 0 {
		renderCtx.Entity = entities[0]
	}

	body, err := renderCtx.Render(bound)
	if err != nil {
		return nil, fmt.Errorf("render tree: %w", err)
	}

	nav, err := p.buildNav(tokens)
	if err != nil {
		nav = "" // non-fatal: degrade gracefully
	}

	css := GenerateCSS(tokens)

	// Build a synthetic resolved entity for meta generation using the type info.
	var metaEntity *graph.ResolvedEntity
	if len(entities) > 0 {
		metaEntity = entities[0]
	} else if entityType != nil {
		metaEntity = &graph.ResolvedEntity{
			Entity: &draft.Entity{
				Data: map[string]any{"title": entityType.Name},
			},
			Type:      entityType,
			Blocks:    map[string][]*draft.Block{},
			Relations: map[string][]*graph.ResolvedEntity{},
		}
	}
	meta := GenerateMeta(metaEntity, tokens, tokens.SiteName)

	return []byte(buildHTMLDoc(meta, css, nav+body, tokens)), nil
}

// buildNav constructs the site-wide navigation bar HTML.
func (p *Pipeline) buildNav(tokens *draft.Tokens) (string, error) {
	siteName := "Drafthaus"
	if tokens != nil && tokens.SiteName != "" {
		siteName = tokens.SiteName
	}

	types, err := p.store.ListTypes()
	if err != nil {
		return "", fmt.Errorf("list types for nav: %w", err)
	}

	var links strings.Builder
	// Only show entity types that have meaningful list routes (not single-segment utility types like /authors, /tags)
	for _, t := range types {
		if t.Routes == nil || t.Routes.List == "" {
			continue
		}
		// Use the list route path to derive a display label
		label := strings.TrimPrefix(t.Routes.List, "/")
		label = strings.ToUpper(label[:1]) + label[1:]
		fmt.Fprintf(&links, `<a href="%s">%s</a>`, t.Routes.List, label)
	}

	return fmt.Sprintf(
		`<nav class="dh-site-nav"><div class="dh-site-nav__inner"><a href="/" class="dh-site-nav__brand">%s</a><div class="dh-site-nav__links">%s<a href="/_admin">Admin</a></div></div></nav>`,
		siteName, links.String(),
	), nil
}

// loadTokens fetches tokens from the store, returning sensible defaults on error.
func (p *Pipeline) loadTokens() (*draft.Tokens, error) {
	ts, err := p.store.GetTokens()
	if err != nil {
		return nil, fmt.Errorf("get tokens: %w", err)
	}
	return &ts.Data, nil
}

// parseViewTree converts view.Tree (which may be a map[string]any or a JSON
// string) into a *Node.
func parseViewTree(tree any) (*Node, error) {
	if tree == nil {
		return &Node{Type: "Stack"}, nil
	}

	switch v := tree.(type) {
	case string:
		if v == "" {
			return &Node{Type: "Stack"}, nil
		}
		var n Node
		if err := json.Unmarshal([]byte(v), &n); err != nil {
			return nil, fmt.Errorf("unmarshal tree string: %w", err)
		}
		return &n, nil
	default:
		return nodeFromAny(v)
	}
}

// buildHTMLDoc wraps rendered content in a complete HTML5 document.
func buildHTMLDoc(meta, css, body string, tokens *draft.Tokens) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString(meta)
	if fontLink := GenerateFontLink(tokens); fontLink != "" {
		b.WriteString(fontLink)
		b.WriteByte('\n')
	}
	b.WriteString("<style>\n")
	b.Write(tailwind.CSS)
	b.WriteString("\n</style>\n")
	b.WriteString("<style>\n")
	b.WriteString(css)
	b.WriteString("</style>\n</head>\n<body>\n")
	b.WriteString(body)
	b.WriteString("\n<footer class=\"dh-footer\"><p>Powered by Drafthaus</p></footer>")
	b.WriteString("\n</body>\n</html>")
	return b.String()
}
