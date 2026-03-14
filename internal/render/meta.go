package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/drafthaus/drafthaus/internal/graph"
)

// GenerateMeta produces <head> content: title, description, OpenGraph, JSON-LD.
func GenerateMeta(entity *graph.ResolvedEntity, tokens *draft.Tokens) string {
	if entity == nil {
		return "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">"
	}

	title := extractTitle(entity)
	description := extractDescription(entity)
	ogType := mapSchemaType(entity.Type.Name) // reuse for og:type
	ogTypeValue := "article"
	if ogType == "Product" {
		ogTypeValue = "product"
	}

	var b strings.Builder

	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<link rel=\"alternate\" type=\"application/rss+xml\" title=\"RSS Feed\" href=\"/feed.xml\">\n")

	if title != "" {
		fmt.Fprintf(&b, "<title>%s</title>\n", Esc(title))
		fmt.Fprintf(&b, "<meta property=\"og:title\" content=\"%s\">\n", Esc(title))
	}

	if description != "" {
		fmt.Fprintf(&b, "<meta name=\"description\" content=\"%s\">\n", Esc(description))
		fmt.Fprintf(&b, "<meta property=\"og:description\" content=\"%s\">\n", Esc(description))
	}

	fmt.Fprintf(&b, "<meta property=\"og:type\" content=\"%s\">\n", ogTypeValue)

	// og:image from image relations.
	if imgRels, ok := entity.Relations["has_image"]; ok && len(imgRels) > 0 {
		imgData := imgRels[0].Entity.Data
		if src, ok := imgData["url"].(string); ok && src != "" {
			fmt.Fprintf(&b, "<meta property=\"og:image\" content=\"%s\">\n", Esc(src))
		} else if src, ok = imgData["src"].(string); ok && src != "" {
			fmt.Fprintf(&b, "<meta property=\"og:image\" content=\"%s\">\n", Esc(src))
		}
	}

	// JSON-LD structured data.
	jsonLD := buildJSONLD(entity, title, description)
	if jsonLD != "" {
		fmt.Fprintf(&b, "<script type=\"application/ld+json\">%s</script>\n", jsonLD)
	}

	return b.String()
}

func extractTitle(entity *graph.ResolvedEntity) string {
	if v, ok := entity.Entity.Data["title"].(string); ok && v != "" {
		return v
	}
	if v, ok := entity.Entity.Data["name"].(string); ok && v != "" {
		return v
	}
	return ""
}

func extractDescription(entity *graph.ResolvedEntity) string {
	if v, ok := entity.Entity.Data["excerpt"].(string); ok && v != "" {
		return truncate(v, 160)
	}
	if v, ok := entity.Entity.Data["description"].(string); ok && v != "" {
		return truncate(v, 160)
	}
	// Fall back to first paragraph block across all richtext fields.
	for _, blocks := range entity.Blocks {
		for _, b := range blocks {
			if b.Type == "paragraph" {
				if text, ok := b.Data["text"].(string); ok && text != "" {
					return truncate(text, 160)
				}
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "\u2026"
}

// mapSchemaType maps an entity type name to a schema.org type string.
func mapSchemaType(typeName string) string {
	switch typeName {
	case "BlogPost", "Article", "Post":
		return "Article"
	case "Product":
		return "Product"
	case "Person":
		return "Person"
	case "Organization":
		return "Organization"
	case "Event":
		return "Event"
	case "Place":
		return "Place"
	default:
		return "Thing"
	}
}

func buildJSONLD(entity *graph.ResolvedEntity, name, description string) string {
	schemaType := mapSchemaType(entity.Type.Name)

	data := map[string]any{
		"@context": "https://schema.org",
		"@type":    schemaType,
	}
	if name != "" {
		data["name"] = name
	}
	if description != "" {
		data["description"] = description
	}

	// Add URL if slug is set.
	if entity.Entity.Slug != "" && entity.Type.Routes != nil && entity.Type.Routes.Detail != "" {
		// Routes.Detail is a pattern like "/posts/:slug" — replace :slug.
		url := strings.ReplaceAll(entity.Type.Routes.Detail, ":slug", entity.Entity.Slug)
		data["url"] = url
	}

	b, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(b)
}

