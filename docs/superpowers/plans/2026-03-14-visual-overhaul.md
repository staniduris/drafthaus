# Visual Overhaul Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Drafthaus-generated sites visually rich by embedding Tailwind CSS, adding a `class` prop to all primitives, and having AI/Go compose styled view trees.

**Architecture:** Replace the 120-line hand-written CSS with embedded Tailwind. Every primitive accepts a `class` prop that merges Tailwind utilities with structural `dh-*` classes. AI prompt expanded to generate Tailwind-styled view trees. Templates updated with rich defaults.

**Tech Stack:** Go, Tailwind CSS v3 (build-time only via npx), Google Fonts

---

## Chunk 1: Foundation — Bug Fixes, SiteName Migration, Class Helper

### Task 1: Add SiteName to Tokens struct

**Files:**
- Modify: `internal/draft/types.go:123-128`

- [ ] **Step 1: Add SiteName field to Tokens struct**

In `internal/draft/types.go`, add `SiteName` to the `Tokens` struct:

```go
type Tokens struct {
	Colors   map[string]string `json:"colors"`
	Fonts    map[string]string `json:"fonts"`
	Scale    ScaleTokens       `json:"scale"`
	Mood     string            `json:"mood,omitempty"`
	SiteName string            `json:"site_name,omitempty"`
}
```

- [ ] **Step 2: Add migration in GetTokens**

In `GetTokens()` in `internal/draft/token.go:33-55`, after the `json.Unmarshal` call on line 50, add the migration:

```go
// Migrate site_name from Colors map to SiteName field.
if ts.Data.SiteName == "" {
    if sn, ok := ts.Data.Colors["site_name"]; ok && sn != "" {
        ts.Data.SiteName = sn
        delete(ts.Data.Colors, "site_name")
    }
}
```

- [ ] **Step 3: Update all writers of site_name**

Search for `site_name` references. Only two files reference it at runtime:
- `internal/ai/generate.go:148` — `colors["site_name"] = spec.SiteName` → remove this line; set `SiteName` on the Tokens struct directly (done in Task 9 Step 4)
- `internal/render/pipeline.go:145-147` — `buildNav` reads `tokens.Colors["site_name"]` → change to `tokens.SiteName`

Note: `internal/cli/init.go` does NOT reference `site_name` — it sets the site name via `ai.ApplySiteSpec` or directly in token colors with the key "site_name" only in `generate.go`. The `GenerateCSS` skip for "site_name" in `css.go` can be removed once it's no longer in the Colors map.

- [ ] **Step 4: Run existing tests**

Run: `go test ./...`
Expected: All tests pass (the struct change is backward-compatible via JSON `omitempty`).

- [ ] **Step 5: Commit**

```bash
git add internal/draft/types.go internal/draft/tokens.go internal/ai/generate.go internal/cli/init.go internal/render/pipeline.go internal/render/css.go
git commit -m "refactor: migrate site_name from Colors map to Tokens.SiteName field"
```

---

### Task 2: Fix homepage title, JSON-LD URLs, RSS title

**Files:**
- Modify: `internal/render/meta.go`
- Modify: `internal/render/pipeline.go`
- Modify: `internal/projections/rss.go`
- Modify: `internal/server/handlers.go`

- [ ] **Step 1: Fix JSON-LD URL pattern**

In `internal/render/meta.go:137`, function `buildJSONLD` currently only replaces `:slug`:

```go
url := strings.ReplaceAll(entity.Type.Routes.Detail, ":slug", entity.Entity.Slug)
```

But routes are stored with `{slug}` pattern (e.g. `/menu/{slug}`). Add `{slug}` replacement first:

```go
url := strings.ReplaceAll(entity.Type.Routes.Detail, "{slug}", entity.Entity.Slug)
url = strings.ReplaceAll(url, ":slug", entity.Entity.Slug)
```

- [ ] **Step 2: Fix homepage title extraction**

In `internal/render/meta.go:13`, change `GenerateMeta` signature to accept a `siteName` parameter:

```go
func GenerateMeta(entity *graph.ResolvedEntity, tokens *draft.Tokens, siteName string) string {
```

Then at the top of the function, use `siteName` as the title when provided:

```go
title := siteName
if title == "" {
    title = extractTitle(entity)
}
```

Update all callers in `internal/render/pipeline.go`:
- `RenderPage` (line 67): pass `""` — detail pages use entity title
- `RenderList` (line 136): pass `tokens.SiteName` — list/homepage uses site name

- [ ] **Step 3: Fix RSS feed title**

In `internal/projections/rss.go`, update `GenerateRSS` to accept a `siteName` parameter:

```go
func GenerateRSS(entities []*graph.ResolvedEntity, entityType *draft.EntityType, baseURL string, siteName string) ([]byte, error) {
```

Then use it:

```go
feedTitle := siteName
if feedTitle == "" {
    feedTitle = entityType.Name
}
// ...
Channel: RSSChannel{
    Title: feedTitle,
```

Update the caller in `internal/server/handlers.go` `serveRSS` to pass the site name from tokens.

- [ ] **Step 4: Run tests**

Run: `go test ./...`
Expected: All pass.

- [ ] **Step 5: Commit**

```bash
git add internal/render/meta.go internal/render/pipeline.go internal/projections/rss.go internal/server/handlers.go
git commit -m "fix: homepage title, JSON-LD URL pattern, RSS feed title"
```

---

### Task 3: Create mergeClasses helper and sanitizer

**Files:**
- Create: `internal/render/classes.go`
- Create: `internal/render/classes_test.go`

- [ ] **Step 1: Write tests for mergeClasses and sanitizeClasses**

Create `internal/render/classes_test.go`:

```go
package render

import "testing"

func TestMergeClasses(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		extra    string
		expected string
	}{
		{"base only", "dh-section", "", "dh-section"},
		{"extra only", "", "bg-white p-4", "bg-white p-4"},
		{"both", "dh-card", "bg-white shadow-lg", "dh-card bg-white shadow-lg"},
		{"trims whitespace", " dh-card ", " bg-white ", "dh-card bg-white"},
		{"empty both", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeClasses(tt.base, tt.extra)
			if got != tt.expected {
				t.Errorf("mergeClasses(%q, %q) = %q, want %q", tt.base, tt.extra, got, tt.expected)
			}
		})
	}
}

func TestSanitizeClasses(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"bg-white p-4", "bg-white p-4"},
		{"text-[#fff] w-[300px]", "text-[#fff] w-[300px]"},
		{"hover:bg-red-500", "hover:bg-red-500"},
		{"\" onmouseover=\"alert(1)", " onmouseoveralert1"},
		{"bg-white\t\np-4", "bg-white p-4"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeClasses(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeClasses(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/render/ -run TestMergeClasses -v`
Expected: FAIL — functions not defined.

- [ ] **Step 3: Implement mergeClasses and sanitizeClasses**

Create `internal/render/classes.go`:

```go
package render

import (
	"regexp"
	"strings"
)

// classAllowedRe matches characters safe for HTML class attributes.
// Allows: alphanumeric, hyphens, underscores, colons (Tailwind variants),
// slashes (opacity), brackets (arbitrary values), dots, percent, parens, spaces.
var classAllowedRe = regexp.MustCompile(`[^a-zA-Z0-9\-_:/\[\]().%# ]`)

// sanitizeClasses removes any characters that could enable XSS via class attributes.
func sanitizeClasses(s string) string {
	// Replace newlines/tabs with spaces first.
	s = strings.NewReplacer("\n", " ", "\r", " ", "\t", " ").Replace(s)
	return classAllowedRe.ReplaceAllString(s, "")
}

// mergeClasses combines a base structural class with extra Tailwind classes.
// Either may be empty. Extra classes are sanitized.
func mergeClasses(base, extra string) string {
	base = strings.TrimSpace(base)
	extra = strings.TrimSpace(sanitizeClasses(extra))
	if base == "" {
		return extra
	}
	if extra == "" {
		return base
	}
	return base + " " + extra
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/render/ -run "TestMergeClasses|TestSanitizeClasses" -v`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/render/classes.go internal/render/classes_test.go
git commit -m "feat: add mergeClasses helper with XSS sanitization for Tailwind class props"
```

---

### Task 4: Add class prop to all primitives

**Files:**
- Modify: `internal/render/primitives.go`

- [ ] **Step 1: Update every primitive to use mergeClasses**

In `internal/render/primitives.go`, update each primitive that outputs a class attribute. Pattern:

For `sectionPrimitive`:
```go
func sectionPrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	titleHTML := ""
	if t := PropString(n, "title"); t != "" {
		titleHTML = fmt.Sprintf("<h2>%s</h2>", Esc(t))
	}
	cls := mergeClasses("dh-section", PropString(n, "class"))
	return fmt.Sprintf("<section class=\"%s\">%s%s</section>", cls, titleHTML, children), nil
}
```

Apply the same pattern to all primitives that output class attributes:
- `stackPrimitive` — `mergeClasses("dh-stack", ...)`
- `columnsPrimitive` — `mergeClasses("dh-columns", ...)`
- `gridPrimitive` — `mergeClasses("dh-grid", ...)`
- `sectionPrimitive` — `mergeClasses("dh-section", ...)`
- `sidebarPrimitive` — `mergeClasses("dh-sidebar", ...)`
- `containerPrimitive` — `mergeClasses("dh-container", ...)`
- `pagePrimitive` — `mergeClasses("dh-page", ...)`
- `textPrimitive` — `mergeClasses("dh-text", ...)`
- `richTextPrimitive` — `mergeClasses("dh-richtext", ...)`
- `headingPrimitive` — `mergeClasses("dh-heading", ...)`
- `imagePrimitive` — `mergeClasses("dh-image", ...)`
- `videoPrimitive` — `mergeClasses("dh-video", ...)`
- `embedPrimitive` — `mergeClasses("dh-embed", ...)`
- `codePrimitive` — `mergeClasses("dh-code", ...)`
- `listPrimitive` — `mergeClasses("dh-list", ...)`
- `tablePrimitive` — `mergeClasses("dh-table", ...)`
- `cardPrimitive` — `mergeClasses("dh-card", ...)`
- `badgePrimitive` — `mergeClasses("dh-badge", ...)`
- `pricePrimitive` — `mergeClasses("dh-price", ...)`
- `datePrimitive` — `mergeClasses("dh-date", ...)`
- `mapPrimitive` — `mergeClasses("dh-map", ...)`
- `actionPrimitive` — `mergeClasses("dh-action", ...)`
- `formPrimitive` — `mergeClasses("dh-form", ...)`
- `searchPrimitive` — `mergeClasses("dh-search", ...)`
- `navPrimitive` — `mergeClasses("dh-nav", ...)`
- `linkPrimitive` — `mergeClasses("dh-link", ...)`
- `breadcrumbPrimitive` — `mergeClasses("dh-breadcrumb", ...)`
- `paginationPrimitive` — `mergeClasses("dh-pagination", ...)`

Note: `fragmentPrimitive` has no wrapper element — skip it. `inputPrimitive` uses `"dh-field"` — update that too.

- [ ] **Step 2: Run tests**

Run: `go test ./...`
Expected: All pass — the class prop defaults to empty string, so existing behavior unchanged.

- [ ] **Step 3: Commit**

```bash
git add internal/render/primitives.go
git commit -m "feat: add class prop support to all rendering primitives"
```

---

## Chunk 2: Tailwind Embedding + CSS Overhaul

### Task 5: Set up Tailwind build pipeline

**Files:**
- Create: `embed/tailwind/embed.go`
- Create: `embed/tailwind/input.css`
- Create: `tailwind.config.js` (project root)
- Create: `package.json` (project root, minimal)

- [ ] **Step 1: Create package.json**

Create `package.json` in project root:

```json
{
  "private": true,
  "devDependencies": {
    "tailwindcss": "^3.4"
  }
}
```

- [ ] **Step 2: Install Tailwind**

Run: `cd /Users/stan/Desktop/drafthaus && npm install`

- [ ] **Step 3: Create tailwind.config.js**

Create `tailwind.config.js` in project root:

```js
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [],
  safelist: [{ pattern: /.*/ }],
  theme: {
    extend: {},
  },
  plugins: [],
}
```

- [ ] **Step 4: Create input.css**

Create `embed/tailwind/input.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

- [ ] **Step 5: Build Tailwind**

Run: `npx tailwindcss -i embed/tailwind/input.css -o embed/tailwind/dist.css --minify`
Expected: Creates `embed/tailwind/dist.css` — verify it's ~4MB (full, safelisted).

- [ ] **Step 6: Create embed.go**

Create `embed/tailwind/embed.go`:

```go
package tailwind

import (
	_ "embed"
)

//go:embed dist.css
var CSS []byte
```

Note: using `[]byte` instead of `string` because the file is ~4MB. The Go compiler handles large `[]byte` embeds more efficiently than string constants.

- [ ] **Step 7: Add to .gitignore**

Add to `.gitignore`:

```
node_modules/
embed/tailwind/dist.css
```

- [ ] **Step 8: Verify build**

Run: `go build -o drafthaus ./cmd/drafthaus/`
Expected: Builds successfully. Check binary size: `ls -lh drafthaus` — should be ~25MB.

- [ ] **Step 9: Commit**

```bash
git add package.json tailwind.config.js embed/tailwind/input.css embed/tailwind/embed.go .gitignore
git commit -m "feat: add Tailwind CSS build pipeline (full safelist, embedded in binary)"
```

---

### Task 6: Overhaul CSS generation and buildHTMLDoc

**Files:**
- Modify: `internal/render/css.go`
- Create: `internal/render/fonts.go`
- Modify: `internal/render/pipeline.go`

- [ ] **Step 1: Create font link generator**

Create `internal/render/fonts.go`:

```go
package render

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// systemFonts are font families that don't need Google Fonts loading.
var systemFonts = map[string]bool{
	"system-ui": true, "sans-serif": true, "serif": true, "monospace": true,
	"cursive": true, "fantasy": true, "ui-serif": true, "ui-sans-serif": true,
	"ui-monospace": true, "ui-rounded": true, "Georgia": true, "Times New Roman": true,
	"Arial": true, "Helvetica": true, "Courier New": true, "Verdana": true,
}

// GenerateFontLink builds a Google Fonts <link> tag from token font names.
// Returns empty string if all fonts are system fonts.
func GenerateFontLink(tokens *draft.Tokens) string {
	if tokens == nil {
		return ""
	}
	seen := make(map[string]bool)
	var families []string
	for _, fontName := range tokens.Fonts {
		fontName = strings.TrimSpace(fontName)
		if fontName == "" || systemFonts[fontName] || seen[fontName] {
			continue
		}
		seen[fontName] = true
		encoded := url.QueryEscape(fontName)
		encoded = strings.ReplaceAll(encoded, "+", "%20")
		// Request common weights including italic.
		families = append(families, fmt.Sprintf("family=%s:ital,wght@0,400;0,500;0,600;0,700;0,800;1,400;1,700", encoded))
	}
	if len(families) == 0 {
		return ""
	}
	return fmt.Sprintf(`<link href="https://fonts.googleapis.com/css2?%s&display=swap" rel="stylesheet">`, strings.Join(families, "&"))
}
```

- [ ] **Step 2: Reduce GenerateCSS to structural-only + token custom properties**

In `internal/render/css.go`, replace `GenerateCSS` with a minimal version that only outputs:
1. CSS custom properties from tokens (`:root { --color-primary: ...; }`)
2. Minimal structural `dh-*` classes (~30 lines for grid/flex defaults that Tailwind can't replace — like the nav bar)

```go
func GenerateCSS(tokens *draft.Tokens) string {
	if tokens == nil {
		tokens = &draft.Tokens{Colors: map[string]string{}, Fonts: map[string]string{}}
	}

	var b strings.Builder

	// Token custom properties.
	b.WriteString(":root {\n")
	for name, val := range tokens.Colors {
		fmt.Fprintf(&b, "  --dh-color-%s: %s;\n", name, val)
	}
	for name, val := range tokens.Fonts {
		fmt.Fprintf(&b, "  --dh-font-%s: %s;\n", name, val)
	}
	b.WriteString("}\n\n")

	// Structural defaults — only what Tailwind can't replace.
	b.WriteString(`/* Structural defaults */
body{font-family:var(--dh-font-body,system-ui,sans-serif);color:var(--dh-color-text,#1a1a1a);background:var(--dh-color-background,#fff);line-height:1.6;margin:0}
h1,h2,h3,h4,h5,h6{font-family:var(--dh-font-heading,system-ui,sans-serif)}
.dh-stack{display:flex;flex-direction:column;gap:1rem}
.dh-columns{display:grid;gap:1rem}
.dh-col{min-width:0}
.dh-grid{display:grid;gap:1rem;grid-template-columns:repeat(auto-fill,minmax(16rem,1fr))}
.dh-section{padding:2rem 0}
.dh-container{max-width:72rem;margin:0 auto;padding:0 1rem}
.dh-sidebar{display:grid;gap:1rem;grid-template-columns:1fr 300px;align-items:start}
.dh-richtext{line-height:1.7;max-width:42rem}
.dh-richtext p{margin-bottom:1em}
.dh-image img{max-width:100%;height:auto;display:block}
.dh-video video{max-width:100%;display:block}
.dh-embed iframe{width:100%;border:none}
.dh-search{position:relative}

/* Nav */
.dh-site-nav{background:var(--dh-color-text,#1a1a1a);color:#fff;padding:0 2rem}
.dh-site-nav__inner{max-width:72rem;margin:0 auto;display:flex;align-items:center;justify-content:space-between;height:3.5rem}
.dh-site-nav__brand{font-family:var(--dh-font-heading,system-ui,sans-serif);font-weight:700;color:inherit;text-decoration:none;font-size:1.125rem}
.dh-site-nav__links{display:flex;gap:1.5rem}
.dh-site-nav__links a{color:inherit;text-decoration:none;opacity:0.8;font-size:0.875rem;transition:opacity 0.2s}
.dh-site-nav__links a:hover{opacity:1}

/* Footer */
.dh-footer{text-align:center;padding:2.5rem 1rem;background:var(--dh-color-text,#1a1a1a);color:rgba(255,255,255,0.4);font-size:0.8125rem}

/* Responsive */
@media(max-width:768px){
  .dh-columns,.dh-grid,.dh-sidebar{grid-template-columns:1fr !important}
}
`)
	return b.String()
}
```

- [ ] **Step 3: Update buildHTMLDoc to embed Tailwind + font link**

In `internal/render/pipeline.go`, update `buildHTMLDoc`:

```go
func buildHTMLDoc(meta, css, body string, tokens *draft.Tokens) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n")
	b.WriteString(meta)

	// Google Fonts.
	if fontLink := GenerateFontLink(tokens); fontLink != "" {
		b.WriteString(fontLink)
		b.WriteByte('\n')
	}

	// Tailwind CSS.
	b.WriteString("<style>\n")
	b.Write(tailwind.CSS)
	b.WriteString("\n</style>\n")

	// Token custom properties + structural CSS.
	b.WriteString("<style>\n")
	b.WriteString(css)
	b.WriteString("</style>\n</head>\n<body>\n")
	b.WriteString(body)
	b.WriteString("\n<footer class=\"dh-footer\"><p>Powered by Drafthaus</p></footer>")
	b.WriteString("\n</body>\n</html>")
	return b.String()
}
```

Add import: `"github.com/drafthaus/drafthaus/embed/tailwind"`

- [ ] **Step 4: Build and test**

Run: `go build -o drafthaus ./cmd/drafthaus/ && go test ./...`
Expected: Build succeeds, all tests pass.

- [ ] **Step 5: Smoke test — serve a cafe site**

```bash
rm -f /tmp/smoketest.draft
./drafthaus init smoketest --template cafe
./drafthaus serve /tmp/smoketest.draft --port 3099 &
sleep 2
curl -s http://localhost:3099/ | head -20
# Verify: should contain <link href="https://fonts.googleapis.com/css2?...">
# Verify: should contain Tailwind CSS in first <style> block
pkill -f "drafthaus serve"
```

- [ ] **Step 6: Commit**

```bash
git add internal/render/css.go internal/render/fonts.go internal/render/pipeline.go
git commit -m "feat: embed Tailwind CSS + Google Fonts loading in HTML output"
```

---

## Chunk 3: Template View Tree Overhaul

### Task 7: Update cafe template with rich Tailwind-styled views

**Files:**
- Modify: `internal/cli/init.go` (the `seedCafe` function)

This is the highest-impact change — making `drafthaus init mysite --template cafe` produce a beautiful site.

- [ ] **Step 1: Update seedCafe homepage view tree**

Replace the homepage view tree in `seedCafe` with a Tailwind-styled version. The tree should follow the pattern: hero → features/menu → dark section → footer.

Key sections of the new homepage view:

```go
homepageTree := map[string]any{
    "type": "Stack",
    "children": []any{
        // Hero section
        map[string]any{
            "type": "Section",
            "props": map[string]any{
                "class": "min-h-[80vh] flex items-center justify-center text-center bg-gradient-to-br from-stone-900 via-stone-800 to-stone-700 text-white relative overflow-hidden",
            },
            "children": []any{
                map[string]any{
                    "type": "Container",
                    "props": map[string]any{"class": "relative z-10 max-w-3xl mx-auto px-4"},
                    "children": []any{
                        map[string]any{"type": "Text", "props": map[string]any{"class": "text-sm uppercase tracking-[0.2em] text-amber-400 font-semibold mb-6", "text": "Welcome to"}},
                        map[string]any{"type": "Heading", "props": map[string]any{"level": 1, "class": "text-5xl md:text-7xl font-extrabold tracking-tight mb-6 leading-[1.1]"}, "bind": map[string]any{"text": "site_name"}},
                        map[string]any{"type": "Text", "props": map[string]any{"class": "text-lg text-white/70 max-w-xl mx-auto mb-10", "text": "Quality coffee. Good food. Great atmosphere."}},
                        map[string]any{
                            "type": "Stack",
                            "props": map[string]any{"class": "flex-row justify-center gap-4"},
                            "children": []any{
                                map[string]any{"type": "Action", "props": map[string]any{"label": "View Our Menu", "href": "/menu", "class": "bg-amber-700 hover:bg-amber-600 text-white px-8 py-4 rounded-xl font-semibold uppercase tracking-wider text-sm transition-all hover:-translate-y-0.5 hover:shadow-lg"}},
                            },
                        },
                    },
                },
            },
        },
        // Menu section
        map[string]any{
            "type": "Section",
            "props": map[string]any{"class": "py-20 bg-amber-50"},
            "children": []any{
                map[string]any{
                    "type": "Container",
                    "children": []any{
                        map[string]any{"type": "Heading", "props": map[string]any{"level": 2, "class": "text-4xl font-bold text-center mb-2 tracking-tight", "text": "Our Menu"}},
                        map[string]any{"type": "Text", "props": map[string]any{"class": "text-center text-stone-500 mb-12", "text": "Freshly roasted, carefully crafted"}},
                        map[string]any{
                            "type":  "Grid",
                            "props": map[string]any{"columns": 2, "class": "gap-6"},
                            "children": []any{
                                map[string]any{
                                    "type": "Card",
                                    "bind": map[string]any{"each": "entities"},
                                    "props": map[string]any{"class": "bg-white rounded-xl shadow-sm hover:shadow-xl hover:-translate-y-1 transition-all duration-300 p-6 border-0"},
                                    "children": []any{
                                        map[string]any{
                                            "type": "Stack",
                                            "props": map[string]any{"class": "flex-row items-center justify-between mb-2"},
                                            "children": []any{
                                                map[string]any{"type": "Badge", "bind": map[string]any{"value": "category"}, "props": map[string]any{"class": "bg-amber-100 text-amber-800 px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider"}},
                                                map[string]any{"type": "Price", "bind": map[string]any{"value": "price"}, "props": map[string]any{"class": "text-xl font-extrabold text-amber-700"}},
                                            },
                                        },
                                        map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 3, "class": "text-lg font-bold mt-1"}},
                                        map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-stone-500 text-sm mt-1"}},
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
        // Dark closing section
        map[string]any{
            "type": "Section",
            "props": map[string]any{"class": "py-20 bg-stone-900 text-white text-center"},
            "children": []any{
                map[string]any{
                    "type": "Container",
                    "children": []any{
                        map[string]any{"type": "Heading", "props": map[string]any{"level": 2, "class": "text-4xl font-bold mb-4 tracking-tight", "text": "Visit Us"}},
                        map[string]any{"type": "Text", "props": map[string]any{"class": "text-white/60 text-lg", "text": "Open daily 7:00 AM — 6:00 PM"}},
                    },
                },
            },
        },
    },
}
```

- [ ] **Step 2: Update seedCafe list view for menu items**

Replace the menu items list view with a Tailwind-styled version matching the homepage grid style.

- [ ] **Step 3: Update seedCafe detail view**

Replace the detail view with proper Tailwind styling:

```go
detailTree := map[string]any{
    "type": "Container",
    "props": map[string]any{"class": "py-16 max-w-2xl mx-auto px-4"},
    "children": []any{
        map[string]any{
            "type": "Stack",
            "props": map[string]any{"class": "gap-4"},
            "children": []any{
                map[string]any{"type": "Badge", "bind": map[string]any{"value": "category"}, "props": map[string]any{"class": "bg-amber-100 text-amber-800 px-3 py-1 rounded-full text-xs font-bold uppercase tracking-wider w-fit"}},
                map[string]any{"type": "Heading", "bind": map[string]any{"text": "name"}, "props": map[string]any{"level": 1, "class": "text-4xl font-bold tracking-tight"}},
                map[string]any{"type": "Text", "bind": map[string]any{"text": "description"}, "props": map[string]any{"class": "text-stone-600 text-lg leading-relaxed"}},
                map[string]any{"type": "Price", "bind": map[string]any{"value": "price"}, "props": map[string]any{"class": "text-3xl font-extrabold text-amber-700"}},
            },
        },
    },
}
```

- [ ] **Step 4: Build and smoke test**

```bash
go build -o drafthaus ./cmd/drafthaus/
rm -f /tmp/cafetest.draft
./drafthaus init cafetest --template cafe
./drafthaus serve /tmp/cafetest.draft --port 3099 &
sleep 2
# Open http://localhost:3099 in browser and visually verify
pkill -f "drafthaus serve"
```

- [ ] **Step 5: Commit**

```bash
git add internal/cli/init.go
git commit -m "feat: cafe template with rich Tailwind-styled hero, cards, and sections"
```

---

### Task 8: Update remaining templates (blog, portfolio, business, blank)

**Files:**
- Modify: `internal/cli/init.go` (functions `seedBlog`, `seedPortfolio`, `seedBusiness`, `seedBlank`)

- [ ] **Step 1: Update seedBlog**

Apply same Tailwind styling pattern — hero section, article grid with shadows, blog post detail with proper typography.

- [ ] **Step 2: Update seedPortfolio**

Hero with gradient, project grid with hover effects, project detail with large images.

- [ ] **Step 3: Update seedBusiness**

Professional layout — hero, services grid, team section, CTA section.

- [ ] **Step 4: Update seedBlank**

Minimal but styled — clean hero, empty content section with instructions.

- [ ] **Step 5: Smoke test all templates**

```bash
for t in blog portfolio business blank; do
  rm -f /tmp/test-$t.draft
  ./drafthaus init test-$t --template $t
done
# Serve each and visually verify
```

- [ ] **Step 6: Commit**

```bash
git add internal/cli/init.go
git commit -m "feat: Tailwind-styled view trees for blog, portfolio, business, blank templates"
```

---

## Chunk 4: AI Prompt Overhaul

### Task 9: Expand AI SiteSpec to include styled view trees

**Files:**
- Modify: `internal/ai/generate.go`

- [ ] **Step 1: Add Views to SiteSpec struct**

```go
type SiteSpec struct {
	SiteName    string                       `json:"site_name"`
	Description string                       `json:"description"`
	Colors      map[string]string            `json:"colors"`
	Fonts       map[string]string            `json:"fonts"`
	EntityTypes []EntityTypeSpec             `json:"entity_types"`
	Views       map[string]json.RawMessage   `json:"views,omitempty"`
}
```

- [ ] **Step 2: Update generateSystemPrompt**

Expand the AI prompt to instruct the model to generate Tailwind-styled view trees. Add after the existing schema description:

```go
const generateSystemPrompt = `You are a CMS content architect and web designer. Generate a complete site specification as a single JSON object.

Rules:
- Output ONLY valid JSON, no markdown fences, no explanation text.
- Use semantic field types from: text, richtext, number, currency, date, datetime, boolean, enum, email, url, geo, asset, relation, json, slug
- Create 2-4 entity types appropriate for the site.
- Generate 3-5 sample entities per entity type with realistic, varied content.
- Suggest brand colors (primary, secondary, background, surface, text, muted, border).
- Choose appropriate Google Font pairings (body, heading, mono) — use real font names like "Playfair Display", "Inter", "Lora", "Poppins".
- Slugs must be lowercase, hyphen-separated.
- Entity status should be "published" for sample data.

IMPORTANT — View trees with Tailwind CSS:
- Generate a "views" object containing view trees for "Homepage", each entity type's list view ("<TypeName>.list"), and detail view ("<TypeName>.detail").
- Each view is a component tree using these types: Stack, Section, Container, Grid, Columns, Heading, Text, Card, Badge, Price, Action, Image, RichText.
- Use the "class" prop with Tailwind CSS utility classes for all styling. Be creative and make it look premium.
- Homepage should have: a full-height hero section with dark/gradient background and centered text, a CTA button, content sections with alternating backgrounds, and a dark closing section.
- Cards should use: shadow-sm, hover:shadow-xl, hover:-translate-y-1, transition-all, rounded-xl.
- Use the "bind" prop to reference entity data: {"text": "field_name"} for text, {"value": "field_name"} for badges/prices, {"each": "entities"} for iteration.
- For list views, Card components use bind.each="entities" to iterate.

` + // ... rest of JSON schema
```

- [ ] **Step 3: Update ApplySiteSpec to store AI-generated views**

In `ApplySiteSpec`, after creating entity types and entities, check if `spec.Views` has entries and store them:

```go
// Store AI-generated views if provided.
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
} else {
    // Fallback: generate views programmatically (existing code)
    // ... keep existing view generation as fallback
}
```

- [ ] **Step 4: Update SiteName handling**

In `ApplySiteSpec`, replace:
```go
colors["site_name"] = spec.SiteName
```
With setting `SiteName` on the `Tokens` struct directly (per Task 1).

- [ ] **Step 5: Run tests and build**

Run: `go test ./... && go build -o drafthaus ./cmd/drafthaus/`
Expected: All pass, builds.

- [ ] **Step 6: Commit**

```bash
git add internal/ai/generate.go
git commit -m "feat: AI generates Tailwind-styled view trees in SiteSpec"
```

---

## Chunk 5: Export Font Download + Final Verification

### Task 10: Update export to download fonts locally

**Files:**
- Modify: `internal/cli/export.go`
- Modify: `internal/render/fonts.go`

- [ ] **Step 1: Add GenerateFontFaces function**

In `internal/render/fonts.go`, add a function that downloads Google Fonts woff2 files and returns `@font-face` CSS rules:

```go
// DownloadFonts fetches woff2 files for the given fonts into outputDir/_fonts/.
// Returns @font-face CSS rules with local paths, or empty string on failure.
func DownloadFonts(tokens *draft.Tokens, outputDir string) string {
    // For each non-system font in tokens.Fonts:
    //   1. Fetch the Google Fonts CSS with woff2 user-agent
    //   2. Extract woff2 URLs from @font-face rules
    //   3. Download each woff2 file to outputDir/_fonts/
    //   4. Rewrite @font-face rules with local paths
    // On any error, log a warning and return "" (fallback to system fonts)
}
```

- [ ] **Step 2: Update export to use local fonts**

In `internal/cli/export.go`, when building the HTML for export:
- Call `DownloadFonts` to get local font CSS
- If successful, replace the Google Fonts `<link>` with inline `@font-face` rules
- If failed, omit the `<link>` entirely (system font fallback)

- [ ] **Step 3: Test export**

```bash
rm -rf /tmp/export-test
./drafthaus export /tmp/cafetest.draft /tmp/export-test
ls /tmp/export-test/_fonts/  # Should contain .woff2 files
head -30 /tmp/export-test/index.html  # Should contain @font-face, not <link>
```

- [ ] **Step 4: Commit**

```bash
git add internal/cli/export.go internal/render/fonts.go
git commit -m "feat: export downloads Google Fonts as local woff2 files"
```

---

### Task 11: Final verification and cleanup

**Files:**
- Modify: `CLAUDE.md` (update primitive count if needed)

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All pass.

- [ ] **Step 2: Visual verification — all templates**

```bash
for t in cafe blog portfolio business blank; do
  rm -f /tmp/final-$t.draft
  ./drafthaus init final-$t --template $t
done
./drafthaus serve /tmp/final-cafe.draft --port 3099 &
# Open browser and verify: hero section, styled cards, fonts loaded, prices formatted
pkill -f "drafthaus serve"
```

- [ ] **Step 3: Verify binary size**

Run: `go build -o drafthaus ./cmd/drafthaus/ && ls -lh drafthaus`
Expected: Under 25MB.

- [ ] **Step 4: Verify export**

```bash
rm -rf /tmp/final-export
./drafthaus export /tmp/final-cafe.draft /tmp/final-export
# Open /tmp/final-export/index.html in browser — should look the same as served version
```

- [ ] **Step 5: Update CLAUDE.md if needed**

If primitive count has changed, update the "25 primitives" reference in CLAUDE.md.

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "chore: final verification and cleanup for visual overhaul"
```
