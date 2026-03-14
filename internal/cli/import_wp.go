package cli

import (
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
	"github.com/google/uuid"
)

// WXR XML structs — Go xml package strips namespace prefixes when matching
// by local name using the "space localname" format.  We match the full
// namespace URI where needed via a custom decoder loop below.

type wxrRSS struct {
	XMLName xml.Name   `xml:"rss"`
	Channel wxrChannel `xml:"channel"`
}

type wxrChannel struct {
	Title string    `xml:"title"`
	Items []wxrItem `xml:"item"`
}

type wxrItem struct {
	Title    string        `xml:"title"`
	Link     string        `xml:"link"`
	PostName string        `xml:"post_name"`
	PostType string        `xml:"post_type"`
	Status   string        `xml:"status"`
	Content  string        `xml:"encoded"`
	Excerpt  string        `xml:"excerpt"`
	PostDate string        `xml:"post_date"`
	Categories []wxrCategory `xml:"category"`
}

type wxrCategory struct {
	Domain   string `xml:"domain,attr"`
	Nicename string `xml:"nicename,attr"`
	Value    string `xml:",chardata"`
}

// ImportWordPress reads a WordPress WXR export file and creates a .draft file.
func ImportWordPress(wxrPath string, outputName string) error {
	data, err := os.ReadFile(wxrPath)
	if err != nil {
		return fmt.Errorf("read WXR file: %w", err)
	}

	// Go's xml decoder requires namespace URIs for namespaced elements.
	// WordPress WXR uses wp:, content: and excerpt: prefixes. We normalise
	// by stripping namespace prefixes so the struct tags (local names) match.
	cleaned := normaliseWXR(data)

	var rss wxrRSS
	if err := xml.Unmarshal(cleaned, &rss); err != nil {
		return fmt.Errorf("parse WXR XML: %w", err)
	}

	outputFile := outputName
	if !strings.HasSuffix(outputFile, ".draft") {
		outputFile += ".draft"
	}
	if _, err := os.Stat(outputFile); err == nil {
		return fmt.Errorf("output file already exists: %s", outputFile)
	}

	store, err := draft.Open(outputFile)
	if err != nil {
		return fmt.Errorf("create draft file: %w", err)
	}
	defer store.Close()

	now := time.Now().Unix()

	// --- Entity types ---
	postType := &draft.EntityType{
		ID:        uuid.NewString(),
		Name:      "Post",
		Slug:      "post",
		Icon:      "📝",
		CreatedAt: now,
		UpdatedAt: now,
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
			{Name: "excerpt", Type: draft.FieldText},
			{Name: "content", Type: draft.FieldRichText},
			{Name: "published_at", Type: draft.FieldDateTime},
		},
		Routes: &draft.RouteConfig{List: "/blog", Detail: "/blog/:slug"},
	}
	if err := store.CreateType(postType); err != nil {
		return fmt.Errorf("create Post type: %w", err)
	}

	pageType := &draft.EntityType{
		ID:        uuid.NewString(),
		Name:      "Page",
		Slug:      "page",
		Icon:      "📄",
		CreatedAt: now,
		UpdatedAt: now,
		Fields: []draft.FieldDef{
			{Name: "title", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
			{Name: "content", Type: draft.FieldRichText},
		},
		Routes: &draft.RouteConfig{Detail: "/:slug"},
	}
	if err := store.CreateType(pageType); err != nil {
		return fmt.Errorf("create Page type: %w", err)
	}

	categoryType := &draft.EntityType{
		ID:        uuid.NewString(),
		Name:      "Category",
		Slug:      "category",
		Icon:      "🗂️",
		CreatedAt: now,
		UpdatedAt: now,
		Fields: []draft.FieldDef{
			{Name: "name", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
		},
	}
	if err := store.CreateType(categoryType); err != nil {
		return fmt.Errorf("create Category type: %w", err)
	}

	tagType := &draft.EntityType{
		ID:        uuid.NewString(),
		Name:      "Tag",
		Slug:      "tag",
		Icon:      "🏷️",
		CreatedAt: now,
		UpdatedAt: now,
		Fields: []draft.FieldDef{
			{Name: "name", Type: draft.FieldText, Required: true},
			{Name: "slug", Type: draft.FieldSlug},
		},
	}
	if err := store.CreateType(tagType); err != nil {
		return fmt.Errorf("create Tag type: %w", err)
	}

	// Track categories and tags by nicename to avoid duplicates.
	categoryIDs := map[string]string{}
	tagIDs := map[string]string{}

	for _, item := range rss.Channel.Items {
		for _, cat := range item.Categories {
			switch cat.Domain {
			case "category":
				if _, ok := categoryIDs[cat.Nicename]; !ok {
					e := &draft.Entity{
						ID:     uuid.NewString(),
						TypeID: categoryType.ID,
						Slug:   cat.Nicename,
						Status: "published",
						Data:   map[string]any{"name": cat.Value, "slug": cat.Nicename},
					}
					if err := store.CreateEntity(e); err != nil {
						return fmt.Errorf("create category %q: %w", cat.Nicename, err)
					}
					categoryIDs[cat.Nicename] = e.ID
				}
			case "post_tag":
				if _, ok := tagIDs[cat.Nicename]; !ok {
					e := &draft.Entity{
						ID:     uuid.NewString(),
						TypeID: tagType.ID,
						Slug:   cat.Nicename,
						Status: "published",
						Data:   map[string]any{"name": cat.Value, "slug": cat.Nicename},
					}
					if err := store.CreateEntity(e); err != nil {
						return fmt.Errorf("create tag %q: %w", cat.Nicename, err)
					}
					tagIDs[cat.Nicename] = e.ID
				}
			}
		}
	}

	posts, pages := 0, 0
	for _, item := range rss.Channel.Items {
		switch item.PostType {
		case "post":
			if item.Status != "publish" {
				continue
			}
			pubAt := parseWPDate(item.PostDate)
			e := &draft.Entity{
				ID:     uuid.NewString(),
				TypeID: postType.ID,
				Slug:   item.PostName,
				Status: "published",
				Data: map[string]any{
					"title":        item.Title,
					"slug":         item.PostName,
					"excerpt":      item.Excerpt,
					"published_at": pubAt,
				},
			}
			if err := store.CreateEntity(e); err != nil {
				return fmt.Errorf("create post %q: %w", item.Title, err)
			}

			blocks := htmlToBlocks(e.ID, "content", item.Content)
			if len(blocks) > 0 {
				if err := store.SetBlocks(e.ID, "content", blocks); err != nil {
					return fmt.Errorf("set blocks for %q: %w", item.Title, err)
				}
			}

			// Relations to categories and tags.
			for _, cat := range item.Categories {
				var targetID string
				var relType string
				switch cat.Domain {
				case "category":
					targetID = categoryIDs[cat.Nicename]
					relType = "category"
				case "post_tag":
					targetID = tagIDs[cat.Nicename]
					relType = "tag"
				}
				if targetID != "" {
					rel := &draft.Relation{
						SourceID:     e.ID,
						TargetID:     targetID,
						RelationType: relType,
					}
					if err := store.AddRelation(rel); err != nil {
						return fmt.Errorf("add relation: %w", err)
					}
				}
			}
			posts++

		case "page":
			if item.Status != "publish" {
				continue
			}
			e := &draft.Entity{
				ID:     uuid.NewString(),
				TypeID: pageType.ID,
				Slug:   item.PostName,
				Status: "published",
				Data: map[string]any{
					"title": item.Title,
					"slug":  item.PostName,
				},
			}
			if err := store.CreateEntity(e); err != nil {
				return fmt.Errorf("create page %q: %w", item.Title, err)
			}
			blocks := htmlToBlocks(e.ID, "content", item.Content)
			if len(blocks) > 0 {
				if err := store.SetBlocks(e.ID, "content", blocks); err != nil {
					return fmt.Errorf("set blocks for page %q: %w", item.Title, err)
				}
			}
			pages++
		}
	}

	// Design tokens.
	tokens := &draft.TokenSet{
		ID: uuid.NewString(),
		Data: draft.Tokens{
			Colors: map[string]string{
				"primary":    "#1a1a2e",
				"secondary":  "#16213e",
				"accent":     "#0f3460",
				"background": "#ffffff",
				"text":       "#333333",
			},
			Fonts: map[string]string{
				"heading": "Georgia, serif",
				"body":    "system-ui, sans-serif",
			},
			Scale: draft.ScaleTokens{
				Spacing: 1.0,
				Radius:  "4px",
				Density: "comfortable",
			},
		},
		UpdatedAt: now,
	}
	if err := store.SetTokens(tokens); err != nil {
		return fmt.Errorf("set tokens: %w", err)
	}

	if err := store.CreateAdminUser("admin", "admin"); err != nil {
		return fmt.Errorf("create admin user: %w", err)
	}

	fmt.Printf("Imported %s → %s\n", wxrPath, outputFile)
	fmt.Printf("  Posts: %d, Pages: %d, Categories: %d, Tags: %d\n",
		posts, pages, len(categoryIDs), len(tagIDs))
	fmt.Printf("  Admin: admin / admin\n")
	return nil
}

// normaliseWXR strips XML namespace prefixes so struct field tags match.
// This avoids the complexity of registering namespace URIs with Go's xml package.
func normaliseWXR(data []byte) []byte {
	s := string(data)
	// Remove namespace declarations to avoid unresolved prefix errors.
	s = regexp.MustCompile(`\s+xmlns:[a-z]+="[^"]*"`).ReplaceAllString(s, "")
	// Strip wp:, content:, excerpt: prefixes from element names.
	for _, prefix := range []string{"wp:", "content:", "excerpt:"} {
		s = strings.ReplaceAll(s, "<"+prefix, "<")
		s = strings.ReplaceAll(s, "</"+prefix, "</")
	}
	return []byte(s)
}

// parseWPDate parses WordPress date strings ("2006-01-02 15:04:05") to Unix timestamp.
func parseWPDate(s string) int64 {
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return time.Now().Unix()
	}
	return t.Unix()
}

var (
	reTag       = regexp.MustCompile(`(?i)<(/?)([a-zA-Z][a-zA-Z0-9]*)([^>]*)>`)
	reImgSrc    = regexp.MustCompile(`(?i)src="([^"]*)"`)
	reImgAlt    = regexp.MustCompile(`(?i)alt="([^"]*)"`)
	reStripTags = regexp.MustCompile(`<[^>]+>`)
)

// htmlToBlocks performs a simple HTML-to-blocks conversion.
// It splits on block-level tags: p, h1-h6, pre, code, img, blockquote.
func htmlToBlocks(entityID, field, html string) []*draft.Block {
	if strings.TrimSpace(html) == "" {
		return nil
	}

	var blocks []*draft.Block
	pos := 0.0

	// Split by opening block-level tags.
	segments := splitByBlockTags(html)
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		if seg == "" {
			continue
		}

		block := classifySegment(entityID, field, seg, pos)
		if block != nil {
			blocks = append(blocks, block)
			pos += 1.0
		}
	}

	return blocks
}

// splitByBlockTags splits raw HTML into segments at block-level boundaries.
func splitByBlockTags(html string) []string {
	blockTags := []string{"<p", "</p>", "<h1", "<h2", "<h3", "<h4", "<h5", "<h6",
		"</h1>", "</h2>", "</h3>", "</h4>", "</h5>", "</h6>",
		"<pre", "</pre>", "<blockquote", "</blockquote>", "<img"}

	lower := strings.ToLower(html)
	var cuts []int
	for _, tag := range blockTags {
		idx := 0
		for {
			i := strings.Index(lower[idx:], tag)
			if i < 0 {
				break
			}
			pos := idx + i
			cuts = append(cuts, pos)
			idx = pos + len(tag)
		}
	}

	if len(cuts) == 0 {
		return []string{html}
	}

	// Sort cuts and split.
	sortInts(cuts)
	var segs []string
	prev := 0
	for _, c := range cuts {
		if c > prev {
			segs = append(segs, html[prev:c])
		}
		prev = c
	}
	if prev < len(html) {
		segs = append(segs, html[prev:])
	}
	return segs
}

func sortInts(s []int) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// classifySegment converts one HTML fragment to a Block.
func classifySegment(entityID, field, seg string, pos float64) *draft.Block {
	lower := strings.ToLower(strings.TrimSpace(seg))

	// Heading
	for level := 1; level <= 6; level++ {
		tag := fmt.Sprintf("<h%d", level)
		if strings.HasPrefix(lower, tag) {
			text := reStripTags.ReplaceAllString(seg, "")
			text = strings.TrimSpace(text)
			if text == "" {
				return nil
			}
			return &draft.Block{
				ID:       uuid.NewString(),
				EntityID: entityID,
				Field:    field,
				Type:     "heading",
				Position: pos,
				Data:     map[string]any{"text": text, "level": level},
			}
		}
	}

	// Img
	if strings.HasPrefix(lower, "<img") {
		src := ""
		alt := ""
		if m := reImgSrc.FindStringSubmatch(seg); m != nil {
			src = m[1]
		}
		if m := reImgAlt.FindStringSubmatch(seg); m != nil {
			alt = m[1]
		}
		if src == "" {
			return nil
		}
		return &draft.Block{
			ID:       uuid.NewString(),
			EntityID: entityID,
			Field:    field,
			Type:     "image",
			Position: pos,
			Data:     map[string]any{"url": src, "alt": alt},
		}
	}

	// Blockquote
	if strings.HasPrefix(lower, "<blockquote") {
		text := reStripTags.ReplaceAllString(seg, "")
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		return &draft.Block{
			ID:       uuid.NewString(),
			EntityID: entityID,
			Field:    field,
			Type:     "callout",
			Position: pos,
			Data:     map[string]any{"text": text},
		}
	}

	// Pre / code
	if strings.HasPrefix(lower, "<pre") || strings.HasPrefix(lower, "<code") {
		text := reStripTags.ReplaceAllString(seg, "")
		text = strings.TrimSpace(text)
		if text == "" {
			return nil
		}
		return &draft.Block{
			ID:       uuid.NewString(),
			EntityID: entityID,
			Field:    field,
			Type:     "code",
			Position: pos,
			Data:     map[string]any{"code": text},
		}
	}

	// Paragraph (default)
	text := reStripTags.ReplaceAllString(seg, "")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	return &draft.Block{
		ID:       uuid.NewString(),
		EntityID: entityID,
		Field:    field,
		Type:     "paragraph",
		Position: pos,
		Data:     map[string]any{"text": text},
	}
}
