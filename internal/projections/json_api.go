package projections

import (
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
)

// EntityToJSON converts a ResolvedEntity to a clean, public-facing JSON map.
func EntityToJSON(re *graph.ResolvedEntity) map[string]any {
	e := re.Entity

	// Build blocks map: field → []{"type":..., "data":...}
	blocksOut := make(map[string][]map[string]any, len(re.Blocks))
	for field, blocks := range re.Blocks {
		items := make([]map[string]any, 0, len(blocks))
		for _, b := range blocks {
			items = append(items, map[string]any{
				"type": b.Type,
				"data": b.Data,
			})
		}
		blocksOut[field] = items
	}

	// Build relations map: relType → []EntityToJSON (shallow)
	relOut := make(map[string][]map[string]any, len(re.Relations))
	for relType, related := range re.Relations {
		items := make([]map[string]any, 0, len(related))
		for _, rel := range related {
			items = append(items, shallowEntityJSON(rel))
		}
		relOut[relType] = items
	}

	typeName := ""
	if re.Type != nil {
		typeName = re.Type.Name
	}

	return map[string]any{
		"id":         e.ID,
		"type":       typeName,
		"slug":       e.Slug,
		"status":     e.Status,
		"data":       e.Data,
		"blocks":     blocksOut,
		"relations":  relOut,
		"created_at": time.Unix(e.CreatedAt, 0).UTC().Format(time.RFC3339),
		"updated_at": time.Unix(e.UpdatedAt, 0).UTC().Format(time.RFC3339),
	}
}

// shallowEntityJSON returns a minimal JSON representation without relations.
func shallowEntityJSON(re *graph.ResolvedEntity) map[string]any {
	e := re.Entity
	typeName := ""
	if re.Type != nil {
		typeName = re.Type.Name
	}

	// Filter data to exclude richtext fields (blocks handle those)
	filteredData := make(map[string]any, len(e.Data))
	richtextFields := richtextFieldSet(re.Type)
	for k, v := range e.Data {
		if !richtextFields[k] {
			filteredData[k] = v
		}
	}

	return map[string]any{
		"id":   e.ID,
		"type": typeName,
		"slug": e.Slug,
		"data": filteredData,
	}
}

// richtextFieldSet returns a set of field names that are richtext type.
func richtextFieldSet(et *draft.EntityType) map[string]bool {
	if et == nil {
		return map[string]bool{}
	}
	out := make(map[string]bool, len(et.Fields))
	for _, f := range et.Fields {
		if f.Type == draft.FieldRichText {
			out[f.Name] = true
		}
	}
	return out
}

// EntityListToJSON wraps a list of entities with pagination metadata.
func EntityListToJSON(entities []*graph.ResolvedEntity, total int) map[string]any {
	data := make([]map[string]any, 0, len(entities))
	for _, re := range entities {
		data = append(data, EntityToJSON(re))
	}
	return map[string]any{
		"data": data,
		"meta": map[string]any{
			"total": total,
			"count": len(entities),
		},
	}
}
