package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
)

// BindContext holds data available for binding.
type BindContext struct {
	Entity   *graph.ResolvedEntity
	Entities []*graph.ResolvedEntity
	Tokens   *draft.Tokens
}

// ResolveBind resolves a dot-separated bind expression against entity data.
// Returns nil, nil when the path is not found (missing data is normal).
func ResolveBind(expr string, ctx *BindContext) (any, error) {
	if ctx == nil || ctx.Entity == nil {
		return nil, nil
	}

	parts := strings.SplitN(expr, ".", 2)
	first := parts[0]

	// Check entity scalar data first.
	if val, ok := ctx.Entity.Entity.Data[first]; ok {
		if len(parts) == 1 {
			return val, nil
		}
		// Nested access into a map value.
		nested, ok := val.(map[string]any)
		if !ok {
			return nil, nil
		}
		return resolveNestedMap(nested, parts[1]), nil
	}

	// Check relations.
	if rels, ok := ctx.Entity.Relations[first]; ok {
		if len(parts) == 1 || len(rels) == 0 {
			// Return the full slice of related entity data maps.
			result := make([]any, 0, len(rels))
			for _, re := range rels {
				result = append(result, re.Entity.Data)
			}
			return result, nil
		}
		second := parts[1]
		if second == "first" {
			if len(rels) == 0 {
				return nil, nil
			}
			return rels[0].Entity.Data, nil
		}
		// Field on first related entity.
		if len(rels) > 0 {
			return rels[0].Entity.Data[second], nil
		}
		return nil, nil
	}

	// Check blocks (richtext fields).
	if blocks, ok := ctx.Entity.Blocks[first]; ok {
		result := make([]any, 0, len(blocks))
		for _, b := range blocks {
			result = append(result, b)
		}
		return result, nil
	}

	return nil, nil
}

func resolveNestedMap(m map[string]any, path string) any {
	parts := strings.SplitN(path, ".", 2)
	val, ok := m[parts[0]]
	if !ok {
		return nil
	}
	if len(parts) == 1 {
		return val
	}
	nested, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	return resolveNestedMap(nested, parts[1])
}

// BindTree deep-copies the node tree, resolving bind expressions into Props.
// For nodes with bind: {"each": "..."}, the node is expanded into multiple
// cloned nodes, one per related entity.
func BindTree(root *Node, ctx *BindContext) (*Node, error) {
	if root == nil {
		return nil, nil
	}
	return bindNode(root, ctx)
}

func bindNode(n *Node, ctx *BindContext) (*Node, error) {
	// Check for "each" binding first — expand before other binds.
	if eachExpr, ok := n.Bind["each"]; ok {
		return expandEachNode(n, eachExpr, ctx)
	}

	// Deep-copy the node.
	clone := &Node{
		Type:  n.Type,
		Props: copyProps(n.Props),
		Bind:  n.Bind,
	}

	// Resolve all non-"each" bind expressions into props.
	for propKey, expr := range n.Bind {
		val, err := ResolveBind(expr, ctx)
		if err != nil {
			return nil, fmt.Errorf("bind %q → %q: %w", propKey, expr, err)
		}
		if val != nil {
			if clone.Props == nil {
				clone.Props = make(map[string]any)
			}
			clone.Props[propKey] = val
		}
	}

	// Recurse into children.
	if len(n.Children) > 0 {
		clone.Children = make([]*Node, 0, len(n.Children))
		for _, child := range n.Children {
			bound, err := bindNode(child, ctx)
			if err != nil {
				return nil, err
			}
			if bound != nil {
				clone.Children = append(clone.Children, bound)
			}
		}
	}

	return clone, nil
}

// expandEachNode resolves an "each" binding to a slice of related entities and
// clones the node once per entity, swapping the BindContext to that entity.
// The result is a synthetic wrapper node whose children are the clones.
func expandEachNode(n *Node, eachExpr string, ctx *BindContext) (*Node, error) {
	val, err := ResolveBind(eachExpr, ctx)
	if err != nil {
		return nil, fmt.Errorf("bind each %q: %w", eachExpr, err)
	}

	// Collect entities to iterate over.
	var relatedEntities []*graph.ResolvedEntity

	// "entities" is the special keyword for the list context (e.g. homepage, list pages)
	if eachExpr == "entities" && len(ctx.Entities) > 0 {
		relatedEntities = ctx.Entities
	} else if ctx.Entity != nil {
		if rels, ok := ctx.Entity.Relations[eachExpr]; ok {
			relatedEntities = rels
		}
	}
	_ = val // val is the data maps slice; we use the full ResolvedEntity slice

	if len(relatedEntities) == 0 {
		// Nothing to iterate — return an empty wrapper.
		return &Node{Type: n.Type, Props: copyProps(n.Props)}, nil
	}

	// Build cloned children, one per entity. Each clone gets the entity
	// context swapped to the related entity.
	clones := make([]*Node, 0, len(relatedEntities))
	// Make a copy of the bind map without "each" so children don't recurse.
	templateBind := make(map[string]string)
	for k, v := range n.Bind {
		if k != "each" {
			templateBind[k] = v
		}
	}
	templateNode := &Node{
		Type:     n.Type,
		Props:    n.Props,
		Bind:     templateBind,
		Children: n.Children,
	}

	for _, re := range relatedEntities {
		childCtx := &BindContext{
			Entity:   re,
			Entities: ctx.Entities,
			Tokens:   ctx.Tokens,
		}
		clone, err := bindNode(templateNode, childCtx)
		if err != nil {
			return nil, fmt.Errorf("bind each element: %w", err)
		}
		clones = append(clones, clone)
	}

	// Return the clones wrapped in a fragment-like node. Since the render
	// tree expects a single node here, we use a "Stack" wrapper so it
	// renders as a simple flex column — callers that use Grid/Columns are
	// expected to wrap the each node themselves.
	return &Node{
		Type:     "Stack",
		Props:    map[string]any{},
		Children: clones,
	}, nil
}

// copyProps makes a shallow copy of a props map.
func copyProps(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// nodeFromAny converts an arbitrary value (e.g. parsed JSON) into a *Node.
// Used by the pipeline when view.Tree is already a map[string]any.
func nodeFromAny(v any) (*Node, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal node: %w", err)
	}
	var n Node
	if err := json.Unmarshal(b, &n); err != nil {
		return nil, fmt.Errorf("unmarshal node: %w", err)
	}
	return &n, nil
}
