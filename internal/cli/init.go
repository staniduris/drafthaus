package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/google/uuid"
)

// Init creates a new .draft file seeded with the given template.
func Init(name string, template string) error {
	if template == "" {
		template = "blog"
	}

	filename := name + ".draft"
	if _, err := os.Stat(filename); err == nil {
		return fmt.Errorf("file already exists: %s", filename)
	}

	store, err := draft.Open(filename)
	if err != nil {
		return fmt.Errorf("create draft file: %w", err)
	}
	defer store.Close()

	switch template {
	case "blank":
		if err := seedBlank(store, name); err != nil {
			return err
		}
		fmt.Printf("Created %s (blank)\n", filename)
	case "blog":
		if err := seedBlog(store, name); err != nil {
			return err
		}
		fmt.Printf("Created %s (blog)\n", filename)
	case "cafe":
		if err := seedCafe(store, name); err != nil {
			return err
		}
		fmt.Printf("Created %s (cafe)\n", filename)
	case "portfolio":
		if err := seedPortfolio(store, name); err != nil {
			return err
		}
		fmt.Printf("Created %s (portfolio)\n", filename)
	case "business":
		if err := seedBusiness(store, name); err != nil {
			return err
		}
		fmt.Printf("Created %s (business)\n", filename)
	default:
		return fmt.Errorf("unknown template: %s (valid: blank, blog, cafe, portfolio, business)", template)
	}

	if err := store.CreateAdminUser("admin", "admin"); err != nil {
		return fmt.Errorf("create default admin: %w", err)
	}
	fmt.Println("Default admin: admin/admin — change this!")

	return nil
}

// now returns the current Unix timestamp.
func now() int64 {
	return time.Now().Unix()
}

// newID generates a new UUID string.
func newID() string {
	return uuid.New().String()
}

// mustJSON marshals v to JSON, panicking on error (only for seeding known-good data).
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("mustJSON: %v", err))
	}
	return b
}

// viewTree builds a JSON string from a map[string]any component tree.
func viewTree(tree map[string]any) string {
	return string(mustJSON(tree))
}

// seedBlank seeds an empty .draft file with default tokens and a minimal homepage.
func seedBlank(store draft.Store, name string) error {
	n := now()

	homepageTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "min-h-[80vh] flex items-center justify-center text-center bg-gradient-to-br from-gray-900 via-gray-800 to-gray-700 text-white"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-3xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": name, "level": 1, "class": "text-5xl md:text-7xl font-extrabold tracking-tight leading-[1.1] mb-6"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Your site starts here.", "class": "text-xl text-gray-300 mb-10"},
							},
							map[string]any{
								"type":  "Action",
								"props": map[string]any{"label": "Get Started", "href": "/_admin", "class": "inline-block bg-white hover:bg-gray-100 text-gray-900 px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all hover:-translate-y-0.5 hover:shadow-lg"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-gray-50"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6 text-center"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Start Building", "level": 2, "class": "text-4xl font-bold mb-4 tracking-tight text-gray-900"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Add entity types, create content, and design your views in the admin panel.", "class": "text-lg text-gray-500"},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Homepage",
		Tree: viewTree(homepageTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Homepage view: %w", err)
	}

	return store.SetTokens(&draft.TokenSet{
		ID: newID(),
		Data: draft.Tokens{
			Colors: map[string]string{
				"primary":    "#2563EB",
				"surface":    "#F8FAFC",
				"text":       "#0F172A",
				"muted":      "#64748B",
				"background": "#FFFFFF",
				"border":     "#E2E8F0",
				"secondary":  "#7C3AED",
			},
			Fonts: map[string]string{
				"body":    "Inter",
				"heading": "Inter",
				"mono":    "JetBrains Mono",
			},
			Scale: draft.ScaleTokens{
				Spacing: 1,
				Radius:  "md",
				Density: "comfortable",
			},
		},
		UpdatedAt: n,
	})
}

// seedBlog seeds a full blog template.
func seedBlog(store draft.Store, name string) error {
	n := now()

	// --- Entity types ---
	authorTypeID := newID()
	authorType := &draft.EntityType{
		ID:   authorTypeID,
		Name: "Author",
		Slug: "authors",
		Fields: []draft.FieldDef{
			{Name: "name", Type: draft.FieldText, Required: true},
			{Name: "bio", Type: draft.FieldText},
			{Name: "avatar", Type: draft.FieldAsset},
		},
		Routes:    &draft.RouteConfig{List: "/authors", Detail: "/authors/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(authorType); err != nil {
		return fmt.Errorf("create author type: %w", err)
	}

	tagTypeID := newID()
	tagType := &draft.EntityType{
		ID:   tagTypeID,
		Name: "Tag",
		Slug: "tags",
		Fields: []draft.FieldDef{
			{Name: "name", Type: draft.FieldText, Required: true},
			{Name: "color", Type: draft.FieldText},
		},
		Routes:    &draft.RouteConfig{List: "/tags", Detail: "/tags/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(tagType); err != nil {
		return fmt.Errorf("create tag type: %w", err)
	}

	postTypeID := newID()
	postType := &draft.EntityType{
		ID:   postTypeID,
		Name: "BlogPost",
		Slug: "blog-posts",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
			{Name: "excerpt", Type: draft.FieldText},
			{Name: "body", Type: draft.FieldRichText},
			{Name: "published_at", Type: draft.FieldDateTime},
		},
		Routes:    &draft.RouteConfig{List: "/blog", Detail: "/blog/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(postType); err != nil {
		return fmt.Errorf("create blog post type: %w", err)
	}

	pageTypeID := newID()
	pageType := &draft.EntityType{
		ID:   pageTypeID,
		Name: "Page",
		Slug: "pages",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
			{Name: "body", Type: draft.FieldRichText},
		},
		Routes:    &draft.RouteConfig{Detail: "/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(pageType); err != nil {
		return fmt.Errorf("create page type: %w", err)
	}

	// --- Entities ---
	authorID := newID()
	author := &draft.Entity{
		ID:     authorID,
		TypeID: authorTypeID,
		Data: map[string]any{
			"name": "Draft Author",
			"bio":  "Writer and creator exploring ideas through Drafthaus.",
		},
		Slug:      "draft-author",
		Status:    "published",
		Position:  1,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(author); err != nil {
		return fmt.Errorf("create author entity: %w", err)
	}

	tag1ID := newID()
	tag1 := &draft.Entity{
		ID:     tag1ID,
		TypeID: tagTypeID,
		Data:   map[string]any{"name": "Getting Started", "color": "#2563EB"},
		Slug:   "getting-started",
		Status: "published", Position: 1,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(tag1); err != nil {
		return fmt.Errorf("create tag1: %w", err)
	}

	tag2ID := newID()
	tag2 := &draft.Entity{
		ID:     tag2ID,
		TypeID: tagTypeID,
		Data:   map[string]any{"name": "Technology", "color": "#7C3AED"},
		Slug:   "technology",
		Status: "published", Position: 2,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(tag2); err != nil {
		return fmt.Errorf("create tag2: %w", err)
	}

	postID := newID()
	post := &draft.Entity{
		ID:     postID,
		TypeID: postTypeID,
		Data: map[string]any{
			"title":        "Welcome to Drafthaus",
			"slug":         "welcome-to-drafthaus",
			"excerpt":      "Discover how Drafthaus makes building your digital presence simple.",
			"published_at": time.Now().Format(time.RFC3339),
		},
		Slug:      "welcome-to-drafthaus",
		Status:    "published",
		Position:  1,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(post); err != nil {
		return fmt.Errorf("create post entity: %w", err)
	}

	// Blocks for the blog post body
	blocks := []*draft.Block{
		{
			ID:       newID(),
			EntityID: postID,
			Field:    "body",
			Type:     "heading",
			Data:     map[string]any{"text": "Welcome to Drafthaus", "level": 1},
			Position: 1,
		},
		{
			ID:       newID(),
			EntityID: postID,
			Field:    "body",
			Type:     "paragraph",
			Data:     map[string]any{"text": "Drafthaus is a single-binary CMS that stores your entire site in one portable .draft file. No databases to manage, no servers to configure — just your content."},
			Position: 2,
		},
		{
			ID:       newID(),
			EntityID: postID,
			Field:    "body",
			Type:     "paragraph",
			Data:     map[string]any{"text": "Start by editing this post, adding new entity types, or exploring the admin interface."},
			Position: 3,
		},
		{
			ID:       newID(),
			EntityID: postID,
			Field:    "body",
			Type:     "code",
			Data:     map[string]any{"text": "drafthaus serve mysite.draft --port 3000", "lang": "bash"},
			Position: 4,
		},
	}
	if err := store.SetBlocks(postID, "body", blocks); err != nil {
		return fmt.Errorf("set post blocks: %w", err)
	}

	// Relations
	if err := store.AddRelation(&draft.Relation{
		SourceID: postID, TargetID: authorID,
		RelationType: "authored_by", Position: 1,
	}); err != nil {
		return fmt.Errorf("add authored_by relation: %w", err)
	}
	if err := store.AddRelation(&draft.Relation{
		SourceID: postID, TargetID: tag1ID,
		RelationType: "tagged_with", Position: 1,
	}); err != nil {
		return fmt.Errorf("add tagged_with tag1: %w", err)
	}
	if err := store.AddRelation(&draft.Relation{
		SourceID: postID, TargetID: tag2ID,
		RelationType: "tagged_with", Position: 2,
	}); err != nil {
		return fmt.Errorf("add tagged_with tag2: %w", err)
	}

	// Pages
	aboutID := newID()
	about := &draft.Entity{
		ID:     aboutID,
		TypeID: pageTypeID,
		Data:   map[string]any{"title": "About", "slug": "about"},
		Slug:   "about",
		Status: "published", Position: 1,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(about); err != nil {
		return fmt.Errorf("create about page: %w", err)
	}
	aboutBlocks := []*draft.Block{
		{
			ID: newID(), EntityID: aboutID, Field: "body",
			Type:     "paragraph",
			Data:     map[string]any{"text": "This site is built with Drafthaus, a single-binary CMS."},
			Position: 1,
		},
	}
	if err := store.SetBlocks(aboutID, "body", aboutBlocks); err != nil {
		return fmt.Errorf("set about blocks: %w", err)
	}

	contactID := newID()
	contact := &draft.Entity{
		ID:     contactID,
		TypeID: pageTypeID,
		Data:   map[string]any{"title": "Contact", "slug": "contact"},
		Slug:   "contact",
		Status: "published", Position: 2,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(contact); err != nil {
		return fmt.Errorf("create contact page: %w", err)
	}
	contactBlocks := []*draft.Block{
		{
			ID: newID(), EntityID: contactID, Field: "body",
			Type:     "paragraph",
			Data:     map[string]any{"text": "Get in touch with us at hello@example.com."},
			Position: 1,
		},
	}
	if err := store.SetBlocks(contactID, "body", contactBlocks); err != nil {
		return fmt.Errorf("set contact blocks: %w", err)
	}

	// --- Views ---
	postDetailTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-white"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16 bg-slate-50 border-b border-slate-200"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-2xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"bind":  map[string]any{"text": "title"},
								"props": map[string]any{"level": 1, "class": "text-4xl font-bold tracking-tight text-slate-900 mb-4"},
							},
							map[string]any{
								"type":  "Stack",
								"props": map[string]any{"class": "flex flex-row gap-4 text-sm text-slate-500"},
								"children": []any{
									map[string]any{"type": "Text", "bind": map[string]any{"text": "authored_by.name"}, "props": map[string]any{"class": "font-medium text-blue-600"}},
									map[string]any{"type": "Text", "bind": map[string]any{"text": "published_at"}, "props": map[string]any{"class": "text-slate-400"}},
								},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-12"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-2xl mx-auto px-6"},
						"children": []any{
							map[string]any{"type": "RichText", "bind": map[string]any{"blocks": "body"}, "props": map[string]any{"class": "prose prose-slate max-w-none"}},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "BlogPost.detail",
		Tree: viewTree(postDetailTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set BlogPost.detail view: %w", err)
	}

	postListTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-slate-50"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16 bg-white border-b border-slate-200"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6 text-center"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Blog", "level": 1, "class": "text-5xl font-extrabold tracking-tight text-slate-900 mb-3"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Ideas, updates, and stories.", "class": "text-lg text-slate-500"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{"type": "Text", "bind": map[string]any{"text": "published_at"}, "props": map[string]any{"class": "text-xs font-medium text-blue-600 uppercase tracking-wider mb-2"}},
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "title"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-slate-900 mb-2"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "excerpt"}, "props": map[string]any{"class": "text-slate-500 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "BlogPost.list",
		Tree: viewTree(postListTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set BlogPost.list view: %w", err)
	}

	pageDetailTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-white"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-2xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"bind":  map[string]any{"text": "title"},
								"props": map[string]any{"level": 1, "class": "text-4xl font-bold tracking-tight text-slate-900 mb-8"},
							},
							map[string]any{"type": "RichText", "bind": map[string]any{"blocks": "body"}, "props": map[string]any{"class": "prose prose-slate max-w-none"}},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Page.detail",
		Tree: viewTree(pageDetailTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Page.detail view: %w", err)
	}

	homepageTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "min-h-[80vh] flex items-center justify-center text-center bg-gradient-to-br from-slate-900 via-slate-800 to-slate-700 text-white relative"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-3xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": name, "level": 1, "class": "text-5xl md:text-7xl font-extrabold tracking-tight leading-[1.1] mb-6"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Ideas worth reading. Stories worth sharing.", "class": "text-xl text-slate-300 mb-10"},
							},
							map[string]any{
								"type":  "Action",
								"props": map[string]any{"label": "Read the Blog", "href": "/blog", "class": "inline-block bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all hover:-translate-y-0.5 hover:shadow-lg"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-slate-50"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Latest Posts", "level": 2, "class": "text-4xl font-bold text-center mb-2 tracking-tight text-slate-900"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Fresh perspectives from our writers.", "class": "text-center text-slate-500 mb-12"},
							},
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{"type": "Text", "bind": map[string]any{"text": "published_at"}, "props": map[string]any{"class": "text-xs font-medium text-blue-600 uppercase tracking-wider mb-2"}},
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "title"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-slate-900 mb-2"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "excerpt"}, "props": map[string]any{"class": "text-slate-500 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Homepage",
		Tree: viewTree(homepageTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Homepage view: %w", err)
	}

	// --- Tokens ---
	return store.SetTokens(&draft.TokenSet{
		ID: newID(),
		Data: draft.Tokens{
			Colors: map[string]string{
				"primary":    "#2563EB",
				"surface":    "#F8FAFC",
				"text":       "#0F172A",
				"muted":      "#64748B",
				"background": "#FFFFFF",
				"border":     "#E2E8F0",
				"secondary":  "#7C3AED",
			},
			Fonts: map[string]string{
				"body":    "Inter",
				"heading": "Inter",
				"mono":    "JetBrains Mono",
			},
			Scale: draft.ScaleTokens{
				Spacing: 1,
				Radius:  "md",
				Density: "comfortable",
			},
		},
		UpdatedAt: n,
	})
}

// seedCafe seeds a cafe/restaurant template.
func seedCafe(store draft.Store, name string) error {
	n := now()

	menuTypeID := newID()
	menuType := &draft.EntityType{
		ID:   menuTypeID,
		Name: "MenuItem",
		Slug: "menu-items",
		Fields: []draft.FieldDef{
			{Name: "name", Type: draft.FieldText, Required: true},
			{Name: "description", Type: draft.FieldText},
			{Name: "price", Type: draft.FieldCurrency},
			{Name: "category", Type: draft.FieldEnum, Values: []string{"coffee", "tea", "pastry", "sandwich", "other"}},
			{Name: "available", Type: draft.FieldBool},
		},
		Routes:    &draft.RouteConfig{List: "/menu", Detail: "/menu/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(menuType); err != nil {
		return fmt.Errorf("create menu item type: %w", err)
	}

	pageTypeID := newID()
	pageType := &draft.EntityType{
		ID:   pageTypeID,
		Name: "Page",
		Slug: "pages",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "body", Type: draft.FieldRichText},
		},
		Routes:    &draft.RouteConfig{Detail: "/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(pageType); err != nil {
		return fmt.Errorf("create page type: %w", err)
	}

	menuItems := []struct {
		name, slug, desc, category string
		price                      float64
	}{
		{"Espresso", "espresso", "Rich and bold single shot.", "coffee", 3.50},
		{"Cappuccino", "cappuccino", "Espresso with steamed milk foam.", "coffee", 4.50},
		{"Croissant", "croissant", "Buttery, flaky French pastry.", "pastry", 3.00},
		{"Avocado Toast", "avocado-toast", "Sourdough with fresh avocado and sea salt.", "sandwich", 8.50},
	}
	for i, item := range menuItems {
		id := newID()
		e := &draft.Entity{
			ID:     id,
			TypeID: menuTypeID,
			Data: map[string]any{
				"name":        item.name,
				"description": item.desc,
				"price":       item.price,
				"category":    item.category,
				"available":   true,
			},
			Slug:      item.slug,
			Status:    "published",
			Position:  float64(i + 1),
			CreatedAt: n, UpdatedAt: n,
		}
		if err := store.CreateEntity(e); err != nil {
			return fmt.Errorf("create menu item %s: %w", item.name, err)
		}
	}

	aboutID := newID()
	about := &draft.Entity{
		ID:     aboutID,
		TypeID: pageTypeID,
		Data:   map[string]any{"title": "About"},
		Slug:   "about",
		Status: "published", Position: 1,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(about); err != nil {
		return fmt.Errorf("create about page: %w", err)
	}
	aboutBlocks := []*draft.Block{
		{
			ID: newID(), EntityID: aboutID, Field: "body",
			Type:     "paragraph",
			Data:     map[string]any{"text": "We are a cozy neighborhood cafe serving quality coffee and food made with love. Come visit us!"},
			Position: 1,
		},
	}
	if err := store.SetBlocks(aboutID, "body", aboutBlocks); err != nil {
		return fmt.Errorf("set about blocks: %w", err)
	}

	menuListTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-amber-50"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16 bg-stone-900 text-white text-center"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-2xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Our Menu", "level": 1, "class": "text-5xl font-extrabold tracking-tight mb-3"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Crafted with care, served with love.", "class": "text-amber-200 text-lg"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{
												"type":  "Stack",
												"props": map[string]any{"class": "flex flex-row justify-between items-start mb-3"},
												"children": []any{
													map[string]any{"type": "Badge", "bind": map[string]any{"value": "category"}, "props": map[string]any{"class": "bg-amber-100 text-amber-800 text-xs font-semibold px-2 py-1 rounded-md uppercase tracking-wide"}},
													map[string]any{"type": "Price", "bind": map[string]any{"value": "price"}, "props": map[string]any{"class": "text-amber-700 font-bold text-lg"}},
												},
											},
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-stone-900 mb-1"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-stone-500 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "MenuItem.list",
		Tree: viewTree(menuListTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set MenuItem.list view: %w", err)
	}

	menuDetailTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-amber-50"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Stack",
								"props": map[string]any{"class": "flex flex-row justify-between items-center mb-4"},
								"children": []any{
									map[string]any{"type": "Badge", "bind": map[string]any{"value": "category"}, "props": map[string]any{"class": "bg-amber-100 text-amber-800 text-xs font-semibold px-2 py-1 rounded-md uppercase tracking-wide"}},
									map[string]any{"type": "Price", "bind": map[string]any{"value": "price"}, "props": map[string]any{"class": "text-amber-700 font-bold text-2xl"}},
								},
							},
							map[string]any{
								"type":  "Heading",
								"bind":  map[string]any{"text": "name"},
								"props": map[string]any{"level": 1, "class": "text-4xl font-bold tracking-tight text-stone-900 mb-4"},
							},
							map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-stone-600 text-lg leading-relaxed"}},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "MenuItem.detail",
		Tree: viewTree(menuDetailTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set MenuItem.detail view: %w", err)
	}

	homepageTree := map[string]any{
		"type": "Stack",
		"children": []any{
			// ── Hero with gradient overlay + texture ──
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "dh-hero min-h-[85vh] flex items-center justify-center text-center bg-gradient-to-br from-stone-950 via-stone-900 to-stone-800 text-white"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-3xl mx-auto px-6"},
						"children": []any{
							map[string]any{"type": "Text", "props": map[string]any{"text": "Est. 2024 · Specialty Coffee", "class": "text-xs uppercase tracking-[0.2em] text-amber-400 font-semibold mb-6"}},
							map[string]any{"type": "Heading", "props": map[string]any{"text": name, "level": 1, "class": "text-5xl md:text-7xl font-extrabold tracking-tight leading-[1.05] mb-6"}},
							map[string]any{"type": "Text", "props": map[string]any{"text": "Quality coffee. Good food. Great atmosphere. Every cup tells a story of careful sourcing and expert roasting.", "class": "text-lg text-white/70 max-w-xl mx-auto mb-10 leading-relaxed"}},
							map[string]any{
								"type":  "Stack",
								"props": map[string]any{"class": "flex-row justify-center gap-4 flex-wrap"},
								"children": []any{
									map[string]any{"type": "Action", "props": map[string]any{"label": "View Our Menu", "href": "/menu", "class": "bg-amber-700 hover:bg-amber-600 text-white px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all hover:-translate-y-0.5 hover:shadow-lg hover:shadow-amber-900/30 border-2 border-amber-700"}},
									map[string]any{"type": "Action", "props": map[string]any{"label": "Our Story", "href": "#story", "class": "bg-transparent text-white px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all border-2 border-white/30 hover:border-white hover:bg-white/5"}},
								},
							},
						},
					},
				},
			},
			// ── Features section ──
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-amber-50/50"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-5xl mx-auto px-6"},
						"children": []any{
							map[string]any{"type": "Heading", "props": map[string]any{"text": "Why Choose Us?", "level": 2, "class": "text-4xl font-bold text-center mb-2 tracking-tight text-stone-900"}},
							map[string]any{"type": "Text", "props": map[string]any{"text": "Three things we care about more than anything", "class": "text-center text-stone-500 mb-3"}},
							map[string]any{"type": "Text", "props": map[string]any{"text": "☕", "class": "dh-ornament text-center mb-12"}},
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 3, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type":  "Card",
										"props": map[string]any{"class": "text-center bg-white rounded-xl p-8 shadow-sm hover:shadow-lg hover:-translate-y-1 transition-all duration-300"},
										"children": []any{
											map[string]any{"type": "Text", "props": map[string]any{"text": "☕", "class": "dh-icon-circle"}},
											map[string]any{"type": "Heading", "props": map[string]any{"text": "Single Origin", "level": 3, "class": "text-xl font-bold mb-3 text-stone-900"}},
											map[string]any{"type": "Text", "props": map[string]any{"text": "Every bean traceable to a single farm. We visit our growers annually to ensure quality and fair trade.", "class": "text-stone-500 text-sm leading-relaxed"}},
										},
									},
									map[string]any{
										"type":  "Card",
										"props": map[string]any{"class": "text-center bg-white rounded-xl p-8 shadow-sm hover:shadow-lg hover:-translate-y-1 transition-all duration-300"},
										"children": []any{
											map[string]any{"type": "Text", "props": map[string]any{"text": "🥐", "class": "dh-icon-circle"}},
											map[string]any{"type": "Heading", "props": map[string]any{"text": "Fresh Daily", "level": 3, "class": "text-xl font-bold mb-3 text-stone-900"}},
											map[string]any{"type": "Text", "props": map[string]any{"text": "Pastries baked before dawn. Bread from local sourdough. Nothing sits on the shelf past noon.", "class": "text-stone-500 text-sm leading-relaxed"}},
										},
									},
									map[string]any{
										"type":  "Card",
										"props": map[string]any{"class": "text-center bg-white rounded-xl p-8 shadow-sm hover:shadow-lg hover:-translate-y-1 transition-all duration-300"},
										"children": []any{
											map[string]any{"type": "Text", "props": map[string]any{"text": "🌿", "class": "dh-icon-circle"}},
											map[string]any{"type": "Heading", "props": map[string]any{"text": "Zero Waste", "level": 3, "class": "text-xl font-bold mb-3 text-stone-900"}},
											map[string]any{"type": "Text", "props": map[string]any{"text": "Compostable cups, grounds to local gardens, seasonal menu to minimize food waste.", "class": "text-stone-500 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
			// ── Menu section with card image placeholders ──
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-5xl mx-auto px-6"},
						"children": []any{
							map[string]any{"type": "Heading", "props": map[string]any{"text": "Our Menu", "level": 2, "class": "text-4xl font-bold text-center mb-2 tracking-tight text-stone-900"}},
							map[string]any{"type": "Text", "props": map[string]any{"text": "Freshly roasted, carefully crafted", "class": "text-center text-stone-500 mb-3"}},
							map[string]any{"type": "Text", "props": map[string]any{"text": "✦", "class": "dh-ornament text-center mb-12"}},
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-6"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 overflow-hidden"},
										"children": []any{
											// Card image placeholder
											map[string]any{"type": "Section", "props": map[string]any{"class": "dh-card-img dh-card-img--dark"}},
											map[string]any{
												"type":  "Stack",
												"props": map[string]any{"class": "p-6 gap-2"},
												"children": []any{
													map[string]any{
														"type":  "Stack",
														"props": map[string]any{"class": "flex-row justify-between items-center"},
														"children": []any{
															map[string]any{"type": "Badge", "bind": map[string]any{"value": "category"}, "props": map[string]any{"class": "bg-amber-100 text-amber-800 text-xs font-bold px-3 py-1 rounded-full uppercase tracking-wider"}},
															map[string]any{"type": "Price", "bind": map[string]any{"value": "price"}, "props": map[string]any{"class": "text-xl font-extrabold text-amber-700"}},
														},
													},
													map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-stone-900"}},
													map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-stone-500 text-sm leading-relaxed"}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			// ── Stats bar ──
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-12 bg-amber-800 text-white"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-5xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Stack",
								"props": map[string]any{"class": "dh-stats"},
								"children": []any{
									map[string]any{"type": "Fragment", "children": []any{
										map[string]any{"type": "Text", "props": map[string]any{"text": "5+", "class": "number"}},
										map[string]any{"type": "Text", "props": map[string]any{"text": "Years Brewing", "class": "label"}},
									}},
									map[string]any{"type": "Fragment", "children": []any{
										map[string]any{"type": "Text", "props": map[string]any{"text": "12k", "class": "number"}},
										map[string]any{"type": "Text", "props": map[string]any{"text": "Cups Monthly", "class": "label"}},
									}},
									map[string]any{"type": "Fragment", "children": []any{
										map[string]any{"type": "Text", "props": map[string]any{"text": "4.8", "class": "number"}},
										map[string]any{"type": "Text", "props": map[string]any{"text": "Google Rating", "class": "label"}},
									}},
									map[string]any{"type": "Fragment", "children": []any{
										map[string]any{"type": "Text", "props": map[string]any{"text": "100%", "class": "number"}},
										map[string]any{"type": "Text", "props": map[string]any{"text": "Fair Trade", "class": "label"}},
									}},
								},
							},
						},
					},
				},
			},
			// ── Story section ──
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-stone-900 text-white"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-5xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Columns",
								"props": map[string]any{"ratio": []any{1, 1}, "class": "gap-16 items-center"},
								"children": []any{
									// Left: decorative quote block
									map[string]any{
										"type":  "Section",
										"props": map[string]any{"class": "dh-quote-bg aspect-[4/3] rounded-xl bg-gradient-to-br from-stone-800 to-stone-700 flex items-center justify-center"},
									},
									// Right: text content
									map[string]any{
										"type":  "Stack",
										"props": map[string]any{"class": "gap-6"},
										"children": []any{
											map[string]any{"type": "Heading", "props": map[string]any{"text": "A story of passion and craft", "level": 2, "class": "text-4xl font-bold tracking-tight leading-tight"}},
											map[string]any{"type": "Text", "props": map[string]any{"text": "We started in a tiny garage with a single espresso machine and a dream. Today we serve hundreds of cups daily, but the principle hasn't changed: every cup should be the best you've ever had.", "class": "text-white/70 leading-relaxed text-lg"}},
											map[string]any{"type": "Text", "props": map[string]any{"text": "We source directly from small farms in Ethiopia, Colombia, and Guatemala. We roast in small batches, twice a week.", "class": "text-white/60 leading-relaxed"}},
											map[string]any{
												"type":  "Stack",
												"props": map[string]any{"class": "dh-signature"},
												"children": []any{
													map[string]any{"type": "Text", "props": map[string]any{"text": "Marco & Elena", "class": "italic text-amber-400 text-lg"}},
													map[string]any{"type": "Text", "props": map[string]any{"text": "Founders", "class": "text-white/40 text-sm"}},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			// ── Visit section ──
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-amber-50 text-center"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-xl mx-auto px-6"},
						"children": []any{
							map[string]any{"type": "Heading", "props": map[string]any{"text": "Visit Us", "level": 2, "class": "text-4xl font-bold tracking-tight mb-4 text-stone-900"}},
							map[string]any{"type": "Text", "props": map[string]any{"text": "Open daily · 7:00 AM — 6:00 PM", "class": "text-stone-500 text-lg mb-8"}},
							map[string]any{"type": "Text", "props": map[string]any{"text": "📍 Corner of Main & Oak, Old Town", "class": "inline-flex items-center gap-2 bg-white px-6 py-3 rounded-xl shadow-sm text-stone-700 font-medium"}},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Homepage",
		Tree: viewTree(homepageTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Homepage view: %w", err)
	}

	return store.SetTokens(&draft.TokenSet{
		ID: newID(),
		Data: draft.Tokens{
			Colors: map[string]string{
				"primary":    "#92400E",
				"surface":    "#FFFBEB",
				"text":       "#1C1917",
				"muted":      "#78716C",
				"background": "#FFFBEB",
				"border":     "#D6D3D1",
				"secondary":  "#B45309",
			},
			Fonts: map[string]string{
				"body":    "Inter",
				"heading": "Playfair Display",
				"mono":    "JetBrains Mono",
			},
			Scale: draft.ScaleTokens{
				Spacing: 1,
				Radius:  "lg",
				Density: "comfortable",
			},
		},
		UpdatedAt: n,
	})
}

// seedPortfolio seeds a portfolio template.
func seedPortfolio(store draft.Store, name string) error {
	n := now()

	projectTypeID := newID()
	projectType := &draft.EntityType{
		ID:   projectTypeID,
		Name: "Project",
		Slug: "projects",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "description", Type: draft.FieldText},
			{Name: "url", Type: draft.FieldURL},
			{Name: "year", Type: draft.FieldNumber},
		},
		Routes:    &draft.RouteConfig{List: "/work", Detail: "/work/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(projectType); err != nil {
		return fmt.Errorf("create project type: %w", err)
	}

	pageTypeID := newID()
	pageType := &draft.EntityType{
		ID:   pageTypeID,
		Name: "Page",
		Slug: "pages",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
			{Name: "body", Type: draft.FieldRichText},
		},
		Routes:    &draft.RouteConfig{Detail: "/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(pageType); err != nil {
		return fmt.Errorf("create page type: %w", err)
	}

	projects := []struct {
		title, slug, desc string
		year              int
	}{
		{"Open Source Library", "open-source-library", "A widely-used utility library for modern web apps.", 2023},
		{"Brand Identity System", "brand-identity-system", "Complete visual identity for a fintech startup.", 2024},
	}
	for i, p := range projects {
		id := newID()
		e := &draft.Entity{
			ID:     id,
			TypeID: projectTypeID,
			Data: map[string]any{
				"title":       p.title,
				"description": p.desc,
				"year":        p.year,
			},
			Slug:      p.slug,
			Status:    "published",
			Position:  float64(i + 1),
			CreatedAt: n, UpdatedAt: n,
		}
		if err := store.CreateEntity(e); err != nil {
			return fmt.Errorf("create project %s: %w", p.title, err)
		}
	}

	aboutID := newID()
	about := &draft.Entity{
		ID:     aboutID,
		TypeID: pageTypeID,
		Data:   map[string]any{"title": "About", "slug": "about"},
		Slug:   "about",
		Status: "published", Position: 1,
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateEntity(about); err != nil {
		return fmt.Errorf("create about page: %w", err)
	}
	aboutBlocks := []*draft.Block{
		{
			ID: newID(), EntityID: aboutID, Field: "body",
			Type:     "paragraph",
			Data:     map[string]any{"text": "Independent designer and developer crafting thoughtful digital experiences."},
			Position: 1,
		},
	}
	if err := store.SetBlocks(aboutID, "body", aboutBlocks); err != nil {
		return fmt.Errorf("set about blocks: %w", err)
	}

	projectListTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-neutral-950 text-white"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16 border-b border-neutral-800"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6 text-center"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Work", "level": 1, "class": "text-5xl font-extrabold tracking-tight mb-3"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Selected projects and explorations.", "class": "text-neutral-400 text-lg"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-neutral-900 border border-neutral-800 rounded-xl hover:border-neutral-600 hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{
												"type":  "Stack",
												"props": map[string]any{"class": "flex flex-row justify-between items-start mb-4"},
												"children": []any{
													map[string]any{"type": "Badge", "bind": map[string]any{"value": "year"}, "props": map[string]any{"class": "bg-neutral-800 text-neutral-300 text-xs font-semibold px-2 py-1 rounded-md"}},
												},
											},
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "title"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-white mb-2"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-neutral-400 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Project.list",
		Tree: viewTree(projectListTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Project.list view: %w", err)
	}

	projectDetailTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-neutral-950 text-white"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-2xl mx-auto px-6"},
						"children": []any{
							map[string]any{"type": "Badge", "bind": map[string]any{"value": "year"}, "props": map[string]any{"class": "inline-block bg-neutral-800 text-neutral-300 text-xs font-semibold px-2 py-1 rounded-md mb-6"}},
							map[string]any{
								"type":  "Heading",
								"bind":  map[string]any{"text": "title"},
								"props": map[string]any{"level": 1, "class": "text-4xl font-bold tracking-tight mb-6"},
							},
							map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-neutral-300 text-lg leading-relaxed mb-8"}},
							map[string]any{"type": "Action", "bind": map[string]any{"href": "url"}, "props": map[string]any{"label": "View Project", "class": "inline-block border border-white text-white hover:bg-white hover:text-neutral-900 px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all"}},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Project.detail",
		Tree: viewTree(projectDetailTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Project.detail view: %w", err)
	}

	homepageTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-neutral-950 text-white"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "min-h-[80vh] flex items-center justify-center text-center bg-gradient-to-br from-neutral-950 via-neutral-900 to-neutral-800 relative"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-3xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": name, "level": 1, "class": "text-5xl md:text-7xl font-extrabold tracking-tight leading-[1.1] mb-6"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Designer. Developer. Creator.", "class": "text-xl text-neutral-400 mb-10"},
							},
							map[string]any{
								"type":  "Action",
								"props": map[string]any{"label": "View My Work", "href": "/work", "class": "inline-block border border-white text-white hover:bg-white hover:text-neutral-900 px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all hover:-translate-y-0.5 hover:shadow-lg"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-neutral-900"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Selected Work", "level": 2, "class": "text-4xl font-bold text-center mb-2 tracking-tight"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Projects that define the craft.", "class": "text-center text-neutral-400 mb-12"},
							},
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-neutral-950 border border-neutral-800 rounded-xl hover:border-neutral-600 hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{"type": "Badge", "bind": map[string]any{"value": "year"}, "props": map[string]any{"class": "inline-block bg-neutral-800 text-neutral-400 text-xs font-semibold px-2 py-1 rounded-md mb-3"}},
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "title"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-white mb-2"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-neutral-400 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Homepage",
		Tree: viewTree(homepageTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Homepage view: %w", err)
	}

	return store.SetTokens(&draft.TokenSet{
		ID: newID(),
		Data: draft.Tokens{
			Colors: map[string]string{
				"primary":    "#F8FAFC",
				"surface":    "#1E293B",
				"text":       "#F1F5F9",
				"muted":      "#94A3B8",
				"background": "#0F172A",
				"border":     "#334155",
				"secondary":  "#38BDF8",
			},
			Fonts: map[string]string{
				"body":    "Inter",
				"heading": "Inter",
				"mono":    "JetBrains Mono",
			},
			Scale: draft.ScaleTokens{
				Spacing: 1,
				Radius:  "sm",
				Density: "compact",
			},
		},
		UpdatedAt: n,
	})
}

// seedBusiness seeds a business template.
func seedBusiness(store draft.Store, name string) error {
	n := now()

	serviceTypeID := newID()
	serviceType := &draft.EntityType{
		ID:   serviceTypeID,
		Name: "Service",
		Slug: "services",
		Fields: []draft.FieldDef{
			{Name: "name", Type: draft.FieldText, Required: true},
			{Name: "description", Type: draft.FieldText},
			{Name: "price", Type: draft.FieldCurrency},
		},
		Routes:    &draft.RouteConfig{List: "/services", Detail: "/services/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(serviceType); err != nil {
		return fmt.Errorf("create service type: %w", err)
	}

	teamTypeID := newID()
	teamType := &draft.EntityType{
		ID:   teamTypeID,
		Name: "TeamMember",
		Slug: "team",
		Fields: []draft.FieldDef{
			{Name: "name", Type: draft.FieldText, Required: true},
			{Name: "role", Type: draft.FieldText},
			{Name: "bio", Type: draft.FieldText},
		},
		Routes:    &draft.RouteConfig{List: "/team", Detail: "/team/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(teamType); err != nil {
		return fmt.Errorf("create team member type: %w", err)
	}

	pageTypeID := newID()
	pageType := &draft.EntityType{
		ID:   pageTypeID,
		Name: "Page",
		Slug: "pages",
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
			{Name: "body", Type: draft.FieldRichText},
		},
		Routes:    &draft.RouteConfig{Detail: "/{slug}"},
		CreatedAt: n, UpdatedAt: n,
	}
	if err := store.CreateType(pageType); err != nil {
		return fmt.Errorf("create page type: %w", err)
	}

	services := []struct {
		name, slug, desc string
		price            float64
	}{
		{"Strategy Consulting", "strategy-consulting", "Align your business goals with a clear roadmap.", 2500},
		{"Brand Design", "brand-design", "Visual identity that communicates your values.", 1500},
		{"Web Development", "web-development", "Fast, modern websites built to last.", 3000},
	}
	for i, s := range services {
		id := newID()
		e := &draft.Entity{
			ID:     id,
			TypeID: serviceTypeID,
			Data: map[string]any{
				"name":        s.name,
				"description": s.desc,
				"price":       s.price,
			},
			Slug:      s.slug,
			Status:    "published",
			Position:  float64(i + 1),
			CreatedAt: n, UpdatedAt: n,
		}
		if err := store.CreateEntity(e); err != nil {
			return fmt.Errorf("create service %s: %w", s.name, err)
		}
	}

	members := []struct {
		name, slug, role, bio string
	}{
		{"Alex Rivera", "alex-rivera", "CEO & Founder", "Passionate about building teams and solving hard problems."},
		{"Sam Chen", "sam-chen", "Head of Design", "Crafting beautiful, functional design for over a decade."},
	}
	for i, m := range members {
		id := newID()
		e := &draft.Entity{
			ID:     id,
			TypeID: teamTypeID,
			Data: map[string]any{
				"name": m.name,
				"role": m.role,
				"bio":  m.bio,
			},
			Slug:      m.slug,
			Status:    "published",
			Position:  float64(i + 1),
			CreatedAt: n, UpdatedAt: n,
		}
		if err := store.CreateEntity(e); err != nil {
			return fmt.Errorf("create team member %s: %w", m.name, err)
		}
	}

	pagesData := []struct {
		title, slug, body string
	}{
		{"About", "about", "We are a boutique consultancy dedicated to helping businesses grow with clarity and purpose."},
		{"Contact", "contact", "Reach us at hello@example.com or call +1 (555) 000-0000."},
	}
	for i, p := range pagesData {
		pageID := newID()
		page := &draft.Entity{
			ID:     pageID,
			TypeID: pageTypeID,
			Data:   map[string]any{"title": p.title, "slug": p.slug},
			Slug:   p.slug,
			Status: "published", Position: float64(i + 1),
			CreatedAt: n, UpdatedAt: n,
		}
		if err := store.CreateEntity(page); err != nil {
			return fmt.Errorf("create page %s: %w", p.title, err)
		}
		pageBlocks := []*draft.Block{
			{
				ID: newID(), EntityID: pageID, Field: "body",
				Type: "paragraph", Data: map[string]any{"text": p.body}, Position: 1,
			},
		}
		if err := store.SetBlocks(pageID, "body", pageBlocks); err != nil {
			return fmt.Errorf("set page %s blocks: %w", p.title, err)
		}
	}

	serviceListTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-blue-50"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16 bg-white border-b border-blue-100"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6 text-center"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Services", "level": 1, "class": "text-5xl font-extrabold tracking-tight text-slate-900 mb-3"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Expert solutions tailored to your business.", "class": "text-lg text-slate-500"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-5xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 3, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-slate-900 mb-2"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-slate-500 text-sm leading-relaxed mb-4"}},
											map[string]any{"type": "Price", "bind": map[string]any{"value": "price"}, "props": map[string]any{"class": "text-blue-700 font-bold text-lg"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Service.list",
		Tree: viewTree(serviceListTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Service.list view: %w", err)
	}

	teamListTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen bg-blue-50"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16 bg-white border-b border-blue-100"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6 text-center"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Our Team", "level": 1, "class": "text-5xl font-extrabold tracking-tight text-slate-900 mb-3"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "The people behind the work.", "class": "text-lg text-slate-500"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-16"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-slate-900 mb-1"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "role"}, "props": map[string]any{"class": "text-blue-600 text-sm font-semibold mb-3 uppercase tracking-wide"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "bio"}, "props": map[string]any{"class": "text-slate-500 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "TeamMember.list",
		Tree: viewTree(teamListTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set TeamMember.list view: %w", err)
	}

	homepageTree := map[string]any{
		"type":  "Stack",
		"props": map[string]any{"class": "min-h-screen"},
		"children": []any{
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "min-h-[80vh] flex items-center justify-center text-center bg-gradient-to-br from-slate-900 via-blue-950 to-slate-800 text-white relative"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-3xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": name, "level": 1, "class": "text-5xl md:text-7xl font-extrabold tracking-tight leading-[1.1] mb-6"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Building better businesses with strategy and design.", "class": "text-xl text-blue-200 mb-10"},
							},
							map[string]any{
								"type":  "Action",
								"props": map[string]any{"label": "Our Services", "href": "/services", "class": "inline-block bg-blue-600 hover:bg-blue-500 text-white px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all hover:-translate-y-0.5 hover:shadow-lg"},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-blue-50"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-5xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "What We Do", "level": 2, "class": "text-4xl font-bold text-center mb-2 tracking-tight text-slate-900"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "Comprehensive solutions for modern businesses.", "class": "text-center text-slate-500 mb-12"},
							},
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 3, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-slate-900 mb-2"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-slate-500 text-sm leading-relaxed"}},
										},
									},
								},
							},
						},
					},
				},
			},
			map[string]any{
				"type":  "Section",
				"props": map[string]any{"class": "py-20 bg-white"},
				"children": []any{
					map[string]any{
						"type":  "Container",
						"props": map[string]any{"class": "max-w-4xl mx-auto px-6"},
						"children": []any{
							map[string]any{
								"type":  "Heading",
								"props": map[string]any{"text": "Meet the Team", "level": 2, "class": "text-4xl font-bold text-center mb-2 tracking-tight text-slate-900"},
							},
							map[string]any{
								"type":  "Text",
								"props": map[string]any{"text": "The people who make it happen.", "class": "text-center text-slate-500 mb-12"},
							},
							map[string]any{
								"type":  "Grid",
								"props": map[string]any{"columns": 2, "class": "gap-8"},
								"children": []any{
									map[string]any{
										"type": "Card",
										"bind": map[string]any{"each": "entities"},
										"props": map[string]any{"class": "bg-blue-50 rounded-xl shadow-sm hover:shadow-md hover:-translate-y-0.5 transition-all duration-300 p-6"},
										"children": []any{
											map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold text-slate-900 mb-1"}},
											map[string]any{"type": "Text", "bind": map[string]any{"text": "role"}, "props": map[string]any{"class": "text-blue-600 text-sm font-semibold uppercase tracking-wide"}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	if err := store.SetView(&draft.View{
		ID: newID(), Name: "Homepage",
		Tree: viewTree(homepageTree), Version: 1,
		CreatedAt: n, UpdatedAt: n,
	}); err != nil {
		return fmt.Errorf("set Homepage view: %w", err)
	}

	return store.SetTokens(&draft.TokenSet{
		ID: newID(),
		Data: draft.Tokens{
			Colors: map[string]string{
				"primary":    "#1D4ED8",
				"surface":    "#EFF6FF",
				"text":       "#1E3A5F",
				"muted":      "#6B7280",
				"background": "#FFFFFF",
				"border":     "#BFDBFE",
				"secondary":  "#3B82F6",
			},
			Fonts: map[string]string{
				"body":    "Inter",
				"heading": "Inter",
				"mono":    "JetBrains Mono",
			},
			Scale: draft.ScaleTokens{
				Spacing: 1,
				Radius:  "md",
				Density: "comfortable",
			},
		},
		UpdatedAt: n,
	})
}
