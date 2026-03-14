package render

import (
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// GenerateCSS produces token CSS custom properties and structural dh-* classes.
// Tailwind utility classes are loaded separately via the tailwind embed package.
func GenerateCSS(tokens *draft.Tokens) string {
	if tokens == nil {
		tokens = &draft.Tokens{Colors: map[string]string{}, Fonts: map[string]string{}}
	}

	var b strings.Builder

	b.WriteString(":root {\n")
	for name, val := range tokens.Colors {
		fmt.Fprintf(&b, "  --dh-color-%s: %s;\n", name, val)
	}
	for name, val := range tokens.Fonts {
		fmt.Fprintf(&b, "  --dh-font-%s: %s;\n", name, val)
	}
	b.WriteString("}\n\n")

	b.WriteString(`body{font-family:var(--dh-font-body,system-ui,sans-serif);color:var(--dh-color-text,#1a1a1a);background:var(--dh-color-background,#fff);line-height:1.6;margin:0}
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
.dh-site-nav{background:var(--dh-color-text,#1a1a1a);color:#fff;padding:0 2rem}
.dh-site-nav__inner{max-width:72rem;margin:0 auto;display:flex;align-items:center;justify-content:space-between;height:3.5rem}
.dh-site-nav__brand{font-family:var(--dh-font-heading,system-ui,sans-serif);font-weight:700;color:inherit;text-decoration:none;font-size:1.125rem}
.dh-site-nav__links{display:flex;gap:1.5rem}
.dh-site-nav__links a{color:inherit;text-decoration:none;opacity:0.8;font-size:0.875rem;transition:opacity 0.2s}
.dh-site-nav__links a:hover{opacity:1}
.dh-footer{text-align:center;padding:2.5rem 1rem;background:var(--dh-color-text,#1a1a1a);color:rgba(255,255,255,0.4);font-size:0.8125rem}
@media(max-width:768px){.dh-columns,.dh-grid,.dh-sidebar{grid-template-columns:1fr !important}}
`)
	return b.String()
}
