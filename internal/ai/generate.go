package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/google/uuid"
)

// SiteSpec is the AI-generated specification for a site.
type SiteSpec struct {
	SiteName    string                     `json:"site_name"`
	Description string                     `json:"description"`
	Colors      map[string]string          `json:"colors"`
	Fonts       map[string]string          `json:"fonts"`
	EntityTypes []EntityTypeSpec           `json:"entity_types"`
	Views       map[string]json.RawMessage `json:"views,omitempty"`
}

// EntityTypeSpec describes an entity type and its sample entities.
type EntityTypeSpec struct {
	Name     string           `json:"name"`
	Slug     string           `json:"slug"`
	Fields   []draft.FieldDef `json:"fields"`
	Routes   *draft.RouteConfig `json:"routes"`
	Entities []EntitySpec     `json:"entities"`
}

// EntitySpec describes a single entity instance.
type EntitySpec struct {
	Data   map[string]any `json:"data"`
	Slug   string         `json:"slug"`
	Status string         `json:"status"`
	Blocks []BlockSpec    `json:"blocks,omitempty"`
}

// BlockSpec describes a single content block.
type BlockSpec struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

const generateSystemPrompt = `You are a CMS content architect. Generate a complete site specification as a single JSON object.

CRITICAL: Output ONLY valid JSON. No markdown fences, no ` + "`" + `json` + "`" + ` prefix, no explanation text — just the raw JSON object.

Rules:
- Use semantic field types: text, richtext, number, currency, date, datetime, boolean, enum, email, url, geo, asset, relation, json, slug
- Create 2-4 entity types appropriate for the site.
- Generate 3-5 sample entities per entity type with realistic, varied content.
- Suggest brand colors (primary, secondary, background, surface, text, muted, border).
- Choose real Google Fonts for font pairings (e.g. "Playfair Display", "Inter", "Lora", "Poppins", "DM Sans", "Merriweather", "Nunito") for body, heading, mono fields.
- Slugs must be lowercase, hyphen-separated.
- Entity status should be "published" for sample data.

VIEW TREE RULES:
- Generate a "views" object containing: "Homepage", and for every entity type "<TypeName>.list" and "<TypeName>.detail".
- Every component node may have: "type", "props", "bind", "children".
- Use Tailwind CSS utility classes in the "class" prop for all styling. Add "class" inside "props".
- Available component types: Stack, Section, Container, Grid, Columns, Heading, Text, Card, Badge, Price, Action, Image, RichText, Date.
- Bind syntax: {"text": "field_name"} for Heading/Text, {"value": "field_name"} for Badge/Price, {"each": "entities"} on Card for iteration, {"blocks": "field_name"} for RichText.

HOMEPAGE PATTERN:
1. Hero Section (dark gradient bg): large Heading with site name, Text with tagline, Action CTA — use classes like "bg-gradient-to-br from-gray-900 to-gray-800 text-white py-24 px-8"
2. Content sections (alternating): first on "bg-white", second on "bg-gray-50" — each with a section Heading (h2) and a Grid of Cards
3. Cards: use "bg-white rounded-xl shadow-md hover:shadow-lg transition-shadow p-6 border border-gray-100"
4. Closing Section (dark): short CTA with "bg-gray-900 text-white py-16 px-8 text-center"

LIST VIEW PATTERN: Container → Stack with Heading (h1) → Grid (columns 2 or 3) → Card (bind each: entities) → [Badge, Heading h3, Text, Price as relevant]
DETAIL VIEW PATTERN: Container → Stack → Heading (h1, large) → optional Badge → Text or RichText for body → Price if applicable

JSON schema:
{
  "site_name": "string",
  "description": "string",
  "colors": {"primary": "#hex", "secondary": "#hex", "background": "#hex", "surface": "#hex", "text": "#hex", "muted": "#hex", "border": "#hex"},
  "fonts": {"body": "font name", "heading": "font name", "mono": "font name"},
  "entity_types": [
    {
      "name": "string",
      "slug": "string",
      "fields": [{"name": "string", "type": "string", "required": true}],
      "routes": {"list": "/path", "detail": "/path/{slug}"},
      "entities": [
        {
          "data": {"field_name": "value"},
          "slug": "string",
          "status": "published",
          "blocks": [{"type": "paragraph", "data": {"text": "..."}}]
        }
      ]
    }
  ],
  "views": {
    "Homepage": { "type": "Stack", "children": [...] },
    "TypeName.list": { "type": "Container", "children": [...] },
    "TypeName.detail": { "type": "Container", "children": [...] }
  }
}`

// GenerateSite asks the AI to produce a complete SiteSpec from a text description.
func GenerateSite(ctx context.Context, provider Provider, description string) (*SiteSpec, error) {
	messages := []Message{
		{Role: "system", Content: generateSystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Generate a complete site specification for: %s", description)},
	}

	raw, err := provider.Complete(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("ai completion: %w", err)
	}

	// Strip markdown fences if present.
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "```") {
		lines := strings.SplitN(cleaned, "\n", 2)
		if len(lines) == 2 {
			cleaned = lines[1]
		}
		if idx := strings.LastIndex(cleaned, "```"); idx >= 0 {
			cleaned = cleaned[:idx]
		}
		cleaned = strings.TrimSpace(cleaned)
	}

	var spec SiteSpec
	if err := json.Unmarshal([]byte(cleaned), &spec); err != nil {
		return nil, fmt.Errorf("parse AI response as JSON: %w\nraw response (first 500 chars): %.500s", err, raw)
	}
	return &spec, nil
}

func newID() string {
	return uuid.New().String()
}

func nowUnix() int64 {
	return time.Now().Unix()
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustJSON: %v", err))
	}
	return string(b)
}

// ApplySiteSpec writes a SiteSpec into the given Store.
func ApplySiteSpec(store draft.Store, spec *SiteSpec) error {
	n := nowUnix()

	// Design tokens.
	colors := map[string]string{
		"primary":    "#2563EB",
		"secondary":  "#7C3AED",
		"background": "#FFFFFF",
		"surface":    "#F8FAFC",
		"text":       "#0F172A",
		"muted":      "#64748B",
		"border":     "#E2E8F0",
	}
	for k, v := range spec.Colors {
		colors[k] = v
	}
	fonts := map[string]string{
		"body":    "Inter",
		"heading": "Inter",
		"mono":    "JetBrains Mono",
	}
	for k, v := range spec.Fonts {
		fonts[k] = v
	}
	if err := store.SetTokens(&draft.TokenSet{
		ID: newID(),
		Data: draft.Tokens{
			Colors:   colors,
			Fonts:    fonts,
			SiteName: spec.SiteName,
			Scale: draft.ScaleTokens{
				Spacing: 1,
				Radius:  "md",
				Density: "comfortable",
			},
		},
		UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set tokens: %w", err)
	}

	// Create entity types and their entities.
	for _, etSpec := range spec.EntityTypes {
		typeID := newID()
		et := &draft.EntityType{
			ID:        typeID,
			Name:      etSpec.Name,
			Slug:      etSpec.Slug,
			Fields:    etSpec.Fields,
			Routes:    etSpec.Routes,
			CreatedAt: n,
			UpdatedAt: n,
		}
		if err := store.CreateType(et); err != nil {
			return fmt.Errorf("create entity type %s: %w", etSpec.Name, err)
		}

		for i, eSpec := range etSpec.Entities {
			entityID := newID()
			status := eSpec.Status
			if status == "" {
				status = "published"
			}
			slug := eSpec.Slug
			if slug == "" {
				slug = fmt.Sprintf("%s-%d", etSpec.Slug, i+1)
			}
			entity := &draft.Entity{
				ID:        entityID,
				TypeID:    typeID,
				Data:      eSpec.Data,
				Slug:      slug,
				Status:    status,
				Position:  float64(i + 1),
				CreatedAt: n,
				UpdatedAt: n,
			}
			if err := store.CreateEntity(entity); err != nil {
				return fmt.Errorf("create entity %s/%s: %w", etSpec.Name, slug, err)
			}

			if len(eSpec.Blocks) > 0 {
				blocks := make([]*draft.Block, len(eSpec.Blocks))
				for j, bSpec := range eSpec.Blocks {
					blocks[j] = &draft.Block{
						ID:       newID(),
						EntityID: entityID,
						Field:    "body",
						Type:     bSpec.Type,
						Data:     bSpec.Data,
						Position: float64(j + 1),
					}
				}
				if err := store.SetBlocks(entityID, "body", blocks); err != nil {
					return fmt.Errorf("set blocks for %s/%s: %w", etSpec.Name, slug, err)
				}
			}
		}

			// Skip programmatic view generation if AI provided views.
		if len(spec.Views) > 0 {
			continue
		}

		// Generate list view for types that have a list route.
		if etSpec.Routes != nil && etSpec.Routes.List != "" {
			listTitle := friendlySectionTitle(etSpec.Routes.List)
			listTree := map[string]any{
				"type": "Container",
				"children": []any{map[string]any{
					"type": "Stack",
					"children": []any{
						map[string]any{"type": "Heading", "props": map[string]any{"text": listTitle, "level": 1}},
						map[string]any{
							"type":  "Grid",
							"props": map[string]any{"columns": 2},
							"children": []any{
								map[string]any{
									"type":     "Card",
									"bind":     map[string]any{"each": "entities"},
									"children": buildListCardChildren(etSpec.Fields),
								},
							},
						},
					},
				}}}
			if err := store.SetView(&draft.View{
				ID:        newID(),
				Name:      etSpec.Name + ".list",
				Tree:      mustJSON(listTree),
				Version:   1,
				CreatedAt: n,
				UpdatedAt: n,
			}); err != nil {
				return fmt.Errorf("set %s.list view: %w", etSpec.Name, err)
			}
		}

		// Generate detail view — wrapped in a Container for proper max-width/padding.
		detailTree := map[string]any{
			"type": "Container",
			"children": []any{
				map[string]any{
					"type":     "Stack",
					"children": buildDetailChildren(etSpec.Fields),
				},
			},
		}
		if err := store.SetView(&draft.View{
			ID:        newID(),
			Name:      etSpec.Name + ".detail",
			Tree:      mustJSON(detailTree),
			Version:   1,
			CreatedAt: n,
			UpdatedAt: n,
		}); err != nil {
			return fmt.Errorf("set %s.detail view: %w", etSpec.Name, err)
		}
	}

	// Store AI-generated views if present; otherwise fall back to programmatic homepage.
	if len(spec.Views) > 0 {
		for viewName, rawTree := range spec.Views {
			if err := store.SetView(&draft.View{
				ID:        newID(),
				Name:      viewName,
				Tree:      string(rawTree),
				Version:   1,
				CreatedAt: n,
				UpdatedAt: n,
			}); err != nil {
				return fmt.Errorf("set view %s: %w", viewName, err)
			}
		}
		return nil
	}

	// Programmatic homepage fallback (no AI views).
	heroSection := map[string]any{
		"type": "Section",
		"children": []any{
			map[string]any{"type": "Heading", "props": map[string]any{"text": spec.SiteName, "level": 1}},
			map[string]any{"type": "Text", "props": map[string]any{"text": spec.Description}},
		},
	}

	var contentSections []any
	for _, etSpec := range spec.EntityTypes {
		if etSpec.Routes == nil || etSpec.Routes.List == "" {
			continue
		}
		sectionTitle := friendlySectionTitle(etSpec.Routes.List)
		section := map[string]any{
			"type": "Section",
			"children": []any{
				map[string]any{"type": "Heading", "props": map[string]any{"text": sectionTitle, "level": 2}},
				map[string]any{
					"type":  "Grid",
					"props": map[string]any{"columns": 3},
					"children": []any{
						map[string]any{
							"type":     "Card",
							"bind":     map[string]any{"each": "entities"},
							"children": buildListCardChildren(etSpec.Fields),
						},
					},
				},
			},
		}
		contentSections = append(contentSections, section)
		break
	}

	homepageTree := map[string]any{
		"type": "Stack",
		"children": []any{
			heroSection,
			map[string]any{
				"type":     "Container",
				"children": contentSections,
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID:        newID(),
		Name:      "Homepage",
		Tree:      mustJSON(homepageTree),
		Version:   1,
		CreatedAt: n,
		UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Homepage view: %w", err)
	}

	return nil
}

// friendlySectionTitle converts a route path to a display title.
func friendlySectionTitle(routePath string) string {
	path := strings.TrimPrefix(routePath, "/")
	switch path {
	case "menu":
		return "Our Menu"
	case "blog":
		return "Latest Posts"
	case "work", "projects":
		return "Our Work"
	case "services":
		return "Our Services"
	case "team":
		return "Our Team"
	case "products":
		return "Products"
	case "gallery":
		return "Gallery"
	default:
		if len(path) > 0 {
			return strings.ToUpper(path[:1]) + path[1:]
		}
		return "Featured"
	}
}

// buildListCardChildren returns card child components: category badge, heading, description, price.
func buildListCardChildren(fields []draft.FieldDef) []any {
	var children []any
	var priceField string
	var categoryField string

	// First pass: find special fields
	for _, f := range fields {
		if f.Type == draft.FieldCurrency {
			priceField = f.Name
		}
		if f.Type == draft.FieldEnum {
			categoryField = f.Name
		}
	}

	// Category badge first
	if categoryField != "" {
		children = append(children, map[string]any{
			"type": "Badge",
			"bind": map[string]any{"value": categoryField},
		})
	}

	// Heading + one text description
	count := 0
	for _, f := range fields {
		if count >= 2 {
			break
		}
		if f.Type == draft.FieldRichText || f.Type == draft.FieldAsset ||
			f.Type == draft.FieldCurrency || f.Type == draft.FieldEnum ||
			f.Type == draft.FieldBool || f.Type == draft.FieldSlug ||
			f.Type == draft.FieldDateTime || f.Type == draft.FieldDate {
			continue
		}
		nodeType := "Text"
		props := map[string]any{}
		if count == 0 {
			nodeType = "Heading"
			props["level"] = 3
		}
		node := map[string]any{
			"type": nodeType,
			"bind": map[string]any{"text": f.Name},
		}
		if len(props) > 0 {
			node["props"] = props
		}
		children = append(children, node)
		count++
	}

	// Price last
	if priceField != "" {
		children = append(children, map[string]any{
			"type": "Price",
			"bind": map[string]any{"value": priceField},
		})
	}

	if len(children) == 0 {
		children = []any{
			map[string]any{"type": "Text", "bind": map[string]any{"text": "id"}},
		}
	}
	return children
}

// buildDetailChildren returns a component list for a detail view.
func buildDetailChildren(fields []draft.FieldDef) []any {
	var children []any
	for _, f := range fields {
		switch f.Type {
		case draft.FieldRichText:
			children = append(children, map[string]any{
				"type": "RichText",
				"bind": map[string]any{"blocks": f.Name},
			})
		case draft.FieldCurrency:
			children = append(children, map[string]any{
				"type": "Price",
				"bind": map[string]any{"value": f.Name},
			})
		case draft.FieldBool, draft.FieldSlug, draft.FieldDateTime, draft.FieldDate:
			// Skip these on detail views — not user-facing
			continue
		case draft.FieldEnum:
			children = append(children, map[string]any{
				"type": "Badge",
				"bind": map[string]any{"value": f.Name},
			})
		default:
			// First text field becomes the heading
			if len(children) == 0 && (f.Type == draft.FieldText) {
				children = append(children, map[string]any{
					"type": "Heading",
					"bind": map[string]any{"text": f.Name},
					"props": map[string]any{"level": 1},
				})
			} else {
				children = append(children, map[string]any{
					"type": "Text",
					"bind": map[string]any{"text": f.Name},
				})
			}
		}
	}
	if len(children) == 0 {
		children = []any{
			map[string]any{"type": "Text", "bind": map[string]any{"text": "id"}},
		}
	}
	return children
}
