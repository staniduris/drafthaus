package render

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
)

var systemFonts = map[string]bool{
	"system-ui": true, "sans-serif": true, "serif": true, "monospace": true,
	"cursive": true, "fantasy": true, "ui-serif": true, "ui-sans-serif": true,
	"ui-monospace": true, "ui-rounded": true, "Georgia": true, "Times New Roman": true,
	"Arial": true, "Helvetica": true, "Courier New": true, "Verdana": true,
}

func GenerateFontLink(tokens *draft.Tokens) string {
	if tokens == nil {
		return ""
	}
	seen := make(map[string]bool)
	var families []string
	for _, fontName := range tokens.Fonts {
		fontName = strings.TrimSpace(fontName)
		// Strip fallback fonts like "Inter, system-ui, sans-serif" → "Inter"
		if idx := strings.IndexByte(fontName, ','); idx >= 0 {
			fontName = strings.TrimSpace(fontName[:idx])
		}
		if fontName == "" || systemFonts[fontName] || seen[fontName] {
			continue
		}
		seen[fontName] = true
		encoded := url.PathEscape(fontName)
		families = append(families, fmt.Sprintf("family=%s:ital,wght@0,400;0,500;0,600;0,700;0,800;1,400;1,700", encoded))
	}
	if len(families) == 0 {
		return ""
	}
	return fmt.Sprintf(`<link href="https://fonts.googleapis.com/css2?%s&display=swap" rel="stylesheet">`, strings.Join(families, "&"))
}
