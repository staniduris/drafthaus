package render

import (
	"fmt"
	"html"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
)

// Node represents a component in the view tree.
type Node struct {
	Type     string            `json:"type"`
	Props    map[string]any    `json:"props,omitempty"`
	Bind     map[string]string `json:"bind,omitempty"`
	Children []*Node           `json:"children,omitempty"`
}

// RenderContext carries state through the render tree walk.
type RenderContext struct {
	Entity   *graph.ResolvedEntity
	Entities []*graph.ResolvedEntity // for list views
	Tokens   *draft.Tokens
	Registry *Registry
}

// Render renders a single node using the registry.
func (ctx *RenderContext) Render(n *Node) (string, error) {
	if n == nil {
		return "", nil
	}
	p, ok := ctx.Registry.Get(n.Type)
	if !ok {
		return "", fmt.Errorf("unknown primitive: %s", n.Type)
	}
	return p(n, ctx)
}

// RenderChildren renders all children of a node and joins them.
func (ctx *RenderContext) RenderChildren(n *Node) (string, error) {
	var buf strings.Builder
	for _, child := range n.Children {
		s, err := ctx.Render(child)
		if err != nil {
			return "", err
		}
		buf.WriteString(s)
	}
	return buf.String(), nil
}

// Primitive is a function that renders a node to HTML.
type Primitive func(n *Node, ctx *RenderContext) (string, error)

// Registry maps component type names to their render functions.
type Registry struct {
	primitives map[string]Primitive
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{primitives: make(map[string]Primitive)}
}

// Register adds a primitive to the registry.
func (r *Registry) Register(name string, p Primitive) {
	r.primitives[name] = p
}

// Get retrieves a primitive by name.
func (r *Registry) Get(name string) (Primitive, bool) {
	p, ok := r.primitives[name]
	return p, ok
}

// PropString gets a string prop, returns empty if missing or wrong type.
func PropString(n *Node, key string) string {
	if n.Props == nil {
		return ""
	}
	v, ok := n.Props[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// PropFloat gets a float64 prop, returns 0 if missing.
func PropFloat(n *Node, key string) float64 {
	if n.Props == nil {
		return 0
	}
	v, ok := n.Props[key]
	if !ok {
		return 0
	}
	f, ok := v.(float64)
	if !ok {
		return 0
	}
	return f
}

// PropInt gets an int prop from float64 JSON number.
func PropInt(n *Node, key string) int {
	return int(PropFloat(n, key))
}

// PropBool gets a bool prop.
func PropBool(n *Node, key string) bool {
	if n.Props == nil {
		return false
	}
	v, ok := n.Props[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	if !ok {
		return false
	}
	return b
}

// PropSlice gets a []any prop.
func PropSlice(n *Node, key string) []any {
	if n.Props == nil {
		return nil
	}
	v, ok := n.Props[key]
	if !ok {
		return nil
	}
	s, ok := v.([]any)
	if !ok {
		return nil
	}
	return s
}

// Esc escapes HTML special characters.
func Esc(s string) string {
	return html.EscapeString(s)
}
