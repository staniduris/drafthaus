package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// RegisterAll registers all 25 built-in primitives into r.
func RegisterAll(r *Registry) {
	// Layout
	r.Register("Fragment", fragmentPrimitive)
	r.Register("Page", pagePrimitive)
	r.Register("Stack", stackPrimitive)
	r.Register("Columns", columnsPrimitive)
	r.Register("Grid", gridPrimitive)
	r.Register("Section", sectionPrimitive)
	r.Register("Sidebar", sidebarPrimitive)
	r.Register("Container", containerPrimitive)
	// Content
	r.Register("Text", textPrimitive)
	r.Register("RichText", richTextPrimitive)
	r.Register("Heading", headingPrimitive)
	r.Register("Image", imagePrimitive)
	r.Register("Video", videoPrimitive)
	r.Register("Embed", embedPrimitive)
	r.Register("Code", codePrimitive)
	// Data
	r.Register("List", listPrimitive)
	r.Register("Table", tablePrimitive)
	r.Register("Card", cardPrimitive)
	r.Register("Badge", badgePrimitive)
	r.Register("Price", pricePrimitive)
	r.Register("Date", datePrimitive)
	r.Register("Map", mapPrimitive)
	// Interactive
	r.Register("Action", actionPrimitive)
	r.Register("Form", formPrimitive)
	r.Register("Input", inputPrimitive)
	r.Register("Search", searchPrimitive)
	// Navigation
	r.Register("Nav", navPrimitive)
	r.Register("Link", linkPrimitive)
	r.Register("Breadcrumb", breadcrumbPrimitive)
	r.Register("Pagination", paginationPrimitive)
}

// ---- Layout ----------------------------------------------------------------

// fragmentPrimitive renders children with no wrapper element.
func fragmentPrimitive(n *Node, ctx *RenderContext) (string, error) {
	return ctx.RenderChildren(n)
}

func pagePrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	title := PropString(n, "title")
	titleHTML := ""
	if title != "" {
		titleHTML = fmt.Sprintf("<title>%s</title>", Esc(title))
	}
	cls := mergeClasses("dh-page", PropString(n, "class"))
	return fmt.Sprintf("%s<main class=\"%s\">%s</main>", titleHTML, cls, children), nil
}

func stackPrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	style := ""
	if gap := PropFloat(n, "gap"); gap > 0 {
		style = fmt.Sprintf(" style=\"gap:%.4grem\"", gap)
	}
	cls := mergeClasses("dh-stack", PropString(n, "class"))
	return fmt.Sprintf("<div class=\"%s\"%s>%s</div>", cls, style, children), nil
}

func columnsPrimitive(n *Node, ctx *RenderContext) (string, error) {
	ratio := PropSlice(n, "ratio")
	colStyle := ""
	if len(ratio) > 0 {
		parts := make([]string, 0, len(ratio))
		for _, v := range ratio {
			switch rv := v.(type) {
			case float64:
				parts = append(parts, fmt.Sprintf("%.4gfr", rv))
			case int:
				parts = append(parts, fmt.Sprintf("%dfr", rv))
			default:
				parts = append(parts, "1fr")
			}
		}
		colStyle = fmt.Sprintf(" style=\"grid-template-columns:%s\"", strings.Join(parts, " "))
	}

	cls := mergeClasses("dh-columns", PropString(n, "class"))
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("<div class=\"%s\"%s>", cls, colStyle))
	for _, child := range n.Children {
		s, err := ctx.Render(child)
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf("<div class=\"dh-col\">%s</div>", s))
	}
	buf.WriteString("</div>")
	return buf.String(), nil
}

func gridPrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	cols := PropInt(n, "columns")
	style := ""
	if cols > 0 {
		style = fmt.Sprintf(" style=\"grid-template-columns:repeat(%d,1fr)\"", cols)
	}
	cls := mergeClasses("dh-grid", PropString(n, "class"))
	return fmt.Sprintf("<div class=\"%s\"%s>%s</div>", cls, style, children), nil
}

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

func sidebarPrimitive(n *Node, ctx *RenderContext) (string, error) {
	mainHTML := ""
	asideHTML := ""
	if len(n.Children) > 0 {
		var err error
		mainHTML, err = ctx.Render(n.Children[0])
		if err != nil {
			return "", err
		}
	}
	if len(n.Children) > 1 {
		var err error
		asideHTML, err = ctx.Render(n.Children[1])
		if err != nil {
			return "", err
		}
	}
	cls := mergeClasses("dh-sidebar", PropString(n, "class"))
	return fmt.Sprintf(
		"<div class=\"%s\" style=\"grid-template-columns:1fr var(--dh-sidebar-width,300px)\">"+
			"<div class=\"dh-sidebar__main\">%s</div>"+
			"<aside class=\"dh-sidebar__aside\">%s</aside>"+
			"</div>",
		cls, mainHTML, asideHTML,
	), nil
}

func containerPrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	cls := mergeClasses("dh-container", PropString(n, "class"))
	return fmt.Sprintf("<div class=\"%s\">%s</div>", cls, children), nil
}

// ---- Content ---------------------------------------------------------------

func textPrimitive(n *Node, _ *RenderContext) (string, error) {
	value := PropString(n, "value")
	if value == "" {
		value = PropString(n, "text")
	}
	if value == "" {
		// Try to stringify any non-string prop value
		if n.Props != nil {
			for _, key := range []string{"value", "text"} {
				if v, ok := n.Props[key]; ok && v != nil {
					value = fmt.Sprintf("%v", v)
					break
				}
			}
		}
	}
	cls := mergeClasses("dh-text", PropString(n, "class"))
	return fmt.Sprintf("<p class=\"%s\">%s</p>", cls, Esc(value)), nil
}

func richTextPrimitive(n *Node, _ *RenderContext) (string, error) {
	blocks := PropSlice(n, "content")
	if len(blocks) == 0 {
		blocks = PropSlice(n, "blocks")
	}
	richCls := mergeClasses("dh-richtext", PropString(n, "class"))
	if len(blocks) == 0 {
		return fmt.Sprintf("<div class=\"%s\"></div>", richCls), nil
	}
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("<div class=\"%s\">", richCls))
	for _, raw := range blocks {
		// Handle both map[string]any (from JSON) and *draft.Block (from resolver)
		var block map[string]any
		switch v := raw.(type) {
		case map[string]any:
			block = v
		case *draft.Block:
			block = map[string]any{"type": v.Type}
			for k, val := range v.Data {
				block[k] = val
			}
		default:
			continue
		}
		blockType, _ := block["type"].(string)
		switch blockType {
		case "paragraph":
			text, _ := block["text"].(string)
			buf.WriteString(fmt.Sprintf("<p>%s</p>", Esc(text)))
		case "heading":
			text, _ := block["text"].(string)
			level := 2
			if l, ok := block["level"].(float64); ok {
				level = int(l)
			}
			if level < 1 {
				level = 1
			}
			if level > 6 {
				level = 6
			}
			buf.WriteString(fmt.Sprintf("<h%d>%s</h%d>", level, Esc(text), level))
		case "image":
			src, _ := block["src"].(string)
			alt, _ := block["alt"].(string)
			caption, _ := block["caption"].(string)
			figcaption := ""
			if caption != "" {
				figcaption = fmt.Sprintf("<figcaption>%s</figcaption>", Esc(caption))
			}
			buf.WriteString(fmt.Sprintf("<figure><img src=\"%s\" alt=\"%s\" loading=\"lazy\">%s</figure>",
				Esc(src), Esc(alt), figcaption))
		case "code":
			lang, _ := block["lang"].(string)
			text, _ := block["text"].(string)
			langClass := ""
			if lang != "" {
				langClass = fmt.Sprintf(" class=\"language-%s\"", Esc(lang))
			}
			buf.WriteString(fmt.Sprintf("<pre><code%s>%s</code></pre>", langClass, Esc(text)))
		case "list":
			items, _ := block["items"].([]any)
			buf.WriteString("<ul>")
			for _, item := range items {
				text, _ := item.(string)
				buf.WriteString(fmt.Sprintf("<li>%s</li>", Esc(text)))
			}
			buf.WriteString("</ul>")
		case "callout":
			text, _ := block["text"].(string)
			buf.WriteString(fmt.Sprintf("<aside class=\"dh-callout\">%s</aside>", Esc(text)))
		case "embed":
			// Raw HTML in embed blocks — kept in sandbox via wrapping iframe.
			src, _ := block["src"].(string)
			title, _ := block["title"].(string)
			buf.WriteString(fmt.Sprintf(
				"<div class=\"dh-embed\"><iframe src=\"%s\" title=\"%s\" sandbox=\"allow-scripts\"></iframe></div>",
				Esc(src), Esc(title),
			))
		default:
			// Unknown block type — skip silently.
		}
	}
	buf.WriteString("</div>")
	return buf.String(), nil
}

func headingPrimitive(n *Node, _ *RenderContext) (string, error) {
	level := PropInt(n, "level")
	if level < 1 || level > 6 {
		level = 2
	}
	text := PropString(n, "text")
	cls := mergeClasses("dh-heading", PropString(n, "class"))
	return fmt.Sprintf("<h%d class=\"%s\">%s</h%d>", level, cls, Esc(text), level), nil
}

func imagePrimitive(n *Node, _ *RenderContext) (string, error) {
	src := PropString(n, "src")
	alt := PropString(n, "alt")
	aspect := PropString(n, "aspect")
	style := ""
	if aspect != "" {
		// Convert "16:9" to "16/9" for CSS aspect-ratio.
		style = fmt.Sprintf(" style=\"aspect-ratio:%s\"", strings.ReplaceAll(Esc(aspect), ":", "/"))
	}
	cls := mergeClasses("dh-image", PropString(n, "class"))
	return fmt.Sprintf(
		"<figure class=\"%s\"%s><img src=\"%s\" alt=\"%s\" loading=\"lazy\"></figure>",
		cls, style, Esc(src), Esc(alt),
	), nil
}

func videoPrimitive(n *Node, _ *RenderContext) (string, error) {
	src := PropString(n, "src")
	poster := PropString(n, "poster")
	posterAttr := ""
	if poster != "" {
		posterAttr = fmt.Sprintf(" poster=\"%s\"", Esc(poster))
	}
	cls := mergeClasses("dh-video", PropString(n, "class"))
	return fmt.Sprintf(
		"<div class=\"%s\"><video src=\"%s\"%s controls></video></div>",
		cls, Esc(src), posterAttr,
	), nil
}

func embedPrimitive(n *Node, _ *RenderContext) (string, error) {
	src := PropString(n, "src")
	title := PropString(n, "title")
	cls := mergeClasses("dh-embed", PropString(n, "class"))
	return fmt.Sprintf(
		"<div class=\"%s\"><iframe src=\"%s\" title=\"%s\" sandbox=\"allow-scripts\"></iframe></div>",
		cls, Esc(src), Esc(title),
	), nil
}

func codePrimitive(n *Node, _ *RenderContext) (string, error) {
	lang := PropString(n, "lang")
	text := PropString(n, "text")
	langClass := ""
	if lang != "" {
		langClass = fmt.Sprintf(" class=\"language-%s\"", Esc(lang))
	}
	cls := mergeClasses("dh-code", PropString(n, "class"))
	return fmt.Sprintf("<pre class=\"%s\"><code%s>%s</code></pre>", cls, langClass, Esc(text)), nil
}

// ---- Data ------------------------------------------------------------------

func listPrimitive(n *Node, ctx *RenderContext) (string, error) {
	cls := mergeClasses("dh-list", PropString(n, "class"))
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("<ul class=\"%s\">", cls))
	for _, child := range n.Children {
		s, err := ctx.Render(child)
		if err != nil {
			return "", err
		}
		buf.WriteString(fmt.Sprintf("<li>%s</li>", s))
	}
	buf.WriteString("</ul>")
	return buf.String(), nil
}

func tablePrimitive(n *Node, _ *RenderContext) (string, error) {
	headers := PropSlice(n, "headers")
	rows := PropSlice(n, "rows")

	tblCls := mergeClasses("dh-table", PropString(n, "class"))
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("<table class=\"%s\">", tblCls))

	if len(headers) > 0 {
		buf.WriteString("<thead><tr>")
		for _, h := range headers {
			text, _ := h.(string)
			buf.WriteString(fmt.Sprintf("<th>%s</th>", Esc(text)))
		}
		buf.WriteString("</tr></thead>")
	}

	buf.WriteString("<tbody>")
	for _, rawRow := range rows {
		row, ok := rawRow.([]any)
		if !ok {
			continue
		}
		buf.WriteString("<tr>")
		for _, cell := range row {
			text, _ := cell.(string)
			buf.WriteString(fmt.Sprintf("<td>%s</td>", Esc(text)))
		}
		buf.WriteString("</tr>")
	}
	buf.WriteString("</tbody></table>")
	return buf.String(), nil
}

func cardPrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	href := PropString(n, "href")
	cls := mergeClasses("dh-card", PropString(n, "class"))
	if href != "" {
		return fmt.Sprintf(
			"<article class=\"%s\"><a href=\"%s\">%s</a></article>",
			cls, Esc(href), children,
		), nil
	}
	return fmt.Sprintf("<article class=\"%s\">%s</article>", cls, children), nil
}

func badgePrimitive(n *Node, _ *RenderContext) (string, error) {
	value := PropString(n, "value")
	variant := PropString(n, "variant")
	base := "dh-badge"
	switch variant {
	case "success", "warning", "error":
		base = fmt.Sprintf("dh-badge dh-badge--%s", variant)
	}
	cls := mergeClasses(base, PropString(n, "class"))
	return fmt.Sprintf("<span class=\"%s\">%s</span>", cls, Esc(value)), nil
}

func pricePrimitive(n *Node, _ *RenderContext) (string, error) {
	value := PropFloat(n, "value")
	currency := PropString(n, "currency")
	if currency == "" {
		currency = "$"
	}
	formatted := fmt.Sprintf("%s%.2f", Esc(currency), value)
	cls := mergeClasses("dh-price", PropString(n, "class"))
	return fmt.Sprintf("<span class=\"%s\">%s</span>", cls, formatted), nil
}

func datePrimitive(n *Node, _ *RenderContext) (string, error) {
	raw := PropString(n, "value")
	format := PropString(n, "format")
	if format == "" {
		format = "Jan 2, 2006"
	}
	cls := mergeClasses("dh-date", PropString(n, "class"))

	var t time.Time
	var err error

	// Try unix timestamp (numeric string).
	if ts, parseErr := strconv.ParseInt(raw, 10, 64); parseErr == nil {
		t = time.Unix(ts, 0).UTC()
	} else {
		// Try ISO 8601 variants.
		for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05Z", "2006-01-02"} {
			t, err = time.Parse(layout, raw)
			if err == nil {
				break
			}
		}
		if err != nil {
			// Fallback: show raw value unformatted.
			return fmt.Sprintf("<time class=\"%s\">%s</time>", cls, Esc(raw)), nil
		}
	}

	datetime := t.Format(time.RFC3339)
	display := t.Format(format)
	return fmt.Sprintf("<time class=\"%s\" datetime=\"%s\">%s</time>", cls, Esc(datetime), Esc(display)), nil
}

func mapPrimitive(n *Node, _ *RenderContext) (string, error) {
	lat := PropString(n, "lat")
	lng := PropString(n, "lng")
	label := PropString(n, "label")
	mapsURL := fmt.Sprintf("https://www.google.com/maps?q=%s,%s", Esc(lat), Esc(lng))
	labelText := label
	if labelText == "" {
		labelText = fmt.Sprintf("%s, %s", lat, lng)
	}
	cls := mergeClasses("dh-map", PropString(n, "class"))
	return fmt.Sprintf(
		"<div class=\"%s\" data-lat=\"%s\" data-lng=\"%s\">"+
			"<a href=\"%s\" target=\"_blank\" rel=\"noopener noreferrer\">%s</a>"+
			"</div>",
		cls, Esc(lat), Esc(lng), mapsURL, Esc(labelText),
	), nil
}

// ---- Interactive -----------------------------------------------------------

func actionPrimitive(n *Node, _ *RenderContext) (string, error) {
	label := PropString(n, "label")
	if label == "" {
		label = PropString(n, "text") // AI may bind text instead of label
	}
	href := PropString(n, "href")
	variant := PropString(n, "variant")
	base := "dh-action"
	switch variant {
	case "primary", "secondary":
		base = fmt.Sprintf("dh-action dh-action--%s", variant)
	}
	cls := mergeClasses(base, PropString(n, "class"))
	if href != "" {
		return fmt.Sprintf("<a class=\"%s\" href=\"%s\">%s</a>", cls, Esc(href), Esc(label)), nil
	}
	return fmt.Sprintf("<button class=\"%s\">%s</button>", cls, Esc(label)), nil
}

func formPrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	action := PropString(n, "action")
	method := PropString(n, "method")
	if method == "" {
		method = "post"
	}
	cls := mergeClasses("dh-form", PropString(n, "class"))
	return fmt.Sprintf(
		"<form class=\"%s\" method=\"%s\" action=\"%s\">%s</form>",
		cls, Esc(method), Esc(action), children,
	), nil
}

func inputPrimitive(n *Node, _ *RenderContext) (string, error) {
	label := PropString(n, "label")
	name := PropString(n, "name")
	inputType := PropString(n, "type")
	if inputType == "" {
		inputType = "text"
	}
	placeholder := PropString(n, "placeholder")
	required := PropBool(n, "required")

	requiredAttr := ""
	if required {
		requiredAttr = " required"
	}
	labelHTML := ""
	if label != "" {
		labelHTML = fmt.Sprintf("<label>%s</label>", Esc(label))
	}
	cls := mergeClasses("dh-field", PropString(n, "class"))
	return fmt.Sprintf(
		"<div class=\"%s\">%s<input type=\"%s\" name=\"%s\" placeholder=\"%s\"%s></div>",
		cls, labelHTML, Esc(inputType), Esc(name), Esc(placeholder), requiredAttr,
	), nil
}

func searchPrimitive(n *Node, _ *RenderContext) (string, error) {
	placeholder := PropString(n, "placeholder")
	entityType := PropString(n, "type")
	if placeholder == "" {
		placeholder = "Search..."
	}
	cls := mergeClasses("dh-search", PropString(n, "class"))
	return fmt.Sprintf(
		"<div class=\"%s\" data-dh-island=\"search\" data-entity-type=\"%s\">"+
			"<input type=\"search\" class=\"dh-search__input\" placeholder=\"%s\" autocomplete=\"off\">"+
			"<div class=\"dh-search__results\" role=\"listbox\" aria-live=\"polite\"></div>"+
			"</div>",
		cls, Esc(entityType), Esc(placeholder),
	), nil
}

// ---- Navigation ------------------------------------------------------------

func navPrimitive(n *Node, ctx *RenderContext) (string, error) {
	children, err := ctx.RenderChildren(n)
	if err != nil {
		return "", err
	}
	cls := mergeClasses("dh-nav", PropString(n, "class"))
	return fmt.Sprintf("<nav class=\"%s\">%s</nav>", cls, children), nil
}

func linkPrimitive(n *Node, ctx *RenderContext) (string, error) {
	href := PropString(n, "href")
	text := PropString(n, "text")
	cls := mergeClasses("dh-link", PropString(n, "class"))
	if len(n.Children) > 0 {
		children, err := ctx.RenderChildren(n)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("<a class=\"%s\" href=\"%s\">%s</a>", cls, Esc(href), children), nil
	}
	return fmt.Sprintf("<a class=\"%s\" href=\"%s\">%s</a>", cls, Esc(href), Esc(text)), nil
}

func breadcrumbPrimitive(n *Node, _ *RenderContext) (string, error) {
	items := PropSlice(n, "items")
	cls := mergeClasses("dh-breadcrumb", PropString(n, "class"))
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("<nav class=\"%s\" aria-label=\"Breadcrumb\"><ol>", cls))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		text, _ := item["text"].(string)
		href, _ := item["href"].(string)
		if href != "" {
			buf.WriteString(fmt.Sprintf("<li><a href=\"%s\">%s</a></li>", Esc(href), Esc(text)))
		} else {
			buf.WriteString(fmt.Sprintf("<li>%s</li>", Esc(text)))
		}
	}
	buf.WriteString("</ol></nav>")
	return buf.String(), nil
}

func paginationPrimitive(n *Node, _ *RenderContext) (string, error) {
	current := PropInt(n, "current")
	total := PropInt(n, "total")
	baseURL := PropString(n, "base_url")

	if current < 1 {
		current = 1
	}

	prevHTML := ""
	if current > 1 {
		prevURL := fmt.Sprintf("%s?page=%d", baseURL, current-1)
		prevHTML = fmt.Sprintf("<a class=\"dh-pagination__prev\" href=\"%s\" rel=\"prev\">&laquo; Prev</a>", Esc(prevURL))
	} else {
		prevHTML = "<span class=\"dh-pagination__prev dh-pagination__prev--disabled\">&laquo; Prev</span>"
	}

	nextHTML := ""
	if current < total {
		nextURL := fmt.Sprintf("%s?page=%d", baseURL, current+1)
		nextHTML = fmt.Sprintf("<a class=\"dh-pagination__next\" href=\"%s\" rel=\"next\">Next &raquo;</a>", Esc(nextURL))
	} else {
		nextHTML = "<span class=\"dh-pagination__next dh-pagination__next--disabled\">Next &raquo;</span>"
	}

	pageInfo := fmt.Sprintf("<span class=\"dh-pagination__info\">%d / %d</span>", current, total)

	cls := mergeClasses("dh-pagination", PropString(n, "class"))
	return fmt.Sprintf(
		"<nav class=\"%s\" aria-label=\"Pagination\">%s%s%s</nav>",
		cls, prevHTML, pageInfo, nextHTML,
	), nil
}
