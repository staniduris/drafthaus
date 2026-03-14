package render

import (
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// GenerateCSS produces a minimal, functional stylesheet from design tokens.
func GenerateCSS(tokens *draft.Tokens) string {
	if tokens == nil {
		tokens = &draft.Tokens{
			Colors: map[string]string{},
			Fonts:  map[string]string{},
			Scale:  draft.ScaleTokens{Spacing: 1.0, Radius: "md", Density: "comfortable"},
		}
	}

	var b strings.Builder

	// Custom properties
	b.WriteString(":root {\n")
	for name, val := range tokens.Colors {
		if name == "site_name" {
			continue
		}
		fmt.Fprintf(&b, "  --dh-color-%s: %s;\n", name, val)
	}
	for name, val := range tokens.Fonts {
		fmt.Fprintf(&b, "  --dh-font-%s: %s;\n", name, val)
	}
	b.WriteString("}\n\n")

	// Reset + base
	b.WriteString(`*,*::before,*::after{box-sizing:border-box}
body,h1,h2,h3,h4,h5,h6,p,figure,blockquote,dl,dd{margin:0}

body{
  font-family:var(--dh-font-body,system-ui,sans-serif);
  color:var(--dh-color-text,#1a1a1a);
  background:var(--dh-color-background,#fff);
  line-height:1.6;
}

h1,h2,h3,h4,h5,h6{
  font-family:var(--dh-font-heading,system-ui,sans-serif);
  line-height:1.2;
  margin-bottom:0.5rem;
}
h1{font-size:2rem}
h2{font-size:1.5rem}
h3{font-size:1.25rem}

a{color:var(--dh-color-primary,#2563eb)}

/* Layout */
.dh-page{max-width:64rem;margin:0 auto;padding:2rem 1rem}
.dh-stack{display:flex;flex-direction:column;gap:1rem}
.dh-columns{display:grid;gap:1rem}
.dh-col{min-width:0}
.dh-grid{display:grid;gap:1rem;grid-template-columns:repeat(auto-fill,minmax(16rem,1fr))}
.dh-section{padding:2rem 0}
.dh-sidebar{display:grid;gap:1rem;grid-template-columns:1fr 300px;align-items:start}
.dh-container{max-width:64rem;margin:0 auto;padding:0 1rem}

/* Content */
.dh-text{line-height:1.6}
.dh-heading{line-height:1.2}
.dh-richtext{line-height:1.7;max-width:42rem}
.dh-richtext p{margin-bottom:1em}
.dh-image img{max-width:100%;height:auto;display:block}
.dh-video video{max-width:100%;display:block}
.dh-embed iframe{width:100%;border:none}
.dh-code{background:var(--dh-color-surface,#f5f5f5);padding:1rem;overflow-x:auto;font-family:var(--dh-font-mono,monospace);font-size:0.875rem;border-radius:4px}

/* Cards */
.dh-card{border:1px solid var(--dh-color-border,#e5e5e5);border-radius:4px;padding:1rem;background:#fff}
.dh-card a{text-decoration:none;color:inherit;display:block}
.dh-card:hover{border-color:var(--dh-color-primary,#2563eb)}

/* Data */
.dh-badge{display:inline-block;padding:0.125rem 0.5rem;border-radius:9999px;font-size:0.75rem;font-weight:600;background:var(--dh-color-surface,#f5f5f5);color:var(--dh-color-muted,#666);text-transform:uppercase;letter-spacing:0.05em}
.dh-price{font-weight:700;font-variant-numeric:tabular-nums}
.dh-callout{border-left:3px solid var(--dh-color-primary,#2563eb);padding:0.5rem 1rem;background:var(--dh-color-surface,#f5f5f5)}
.dh-list{padding-left:1.5rem;line-height:1.7}
.dh-table{width:100%;border-collapse:collapse}
.dh-table th,.dh-table td{padding:0.5rem;border-bottom:1px solid var(--dh-color-border,#e5e5e5);text-align:left}
.dh-map{aspect-ratio:16/9;background:var(--dh-color-surface,#f5f5f5);display:flex;align-items:center;justify-content:center}

/* Interactive */
.dh-action{display:inline-flex;padding:0.5rem 1rem;border-radius:4px;cursor:pointer;font-weight:600;text-decoration:none;background:var(--dh-color-primary,#2563eb);color:#fff;border:none}
.dh-action--secondary{background:transparent;color:var(--dh-color-primary,#2563eb);border:1px solid var(--dh-color-primary,#2563eb)}
.dh-form{display:flex;flex-direction:column;gap:1rem}
.dh-field{display:flex;flex-direction:column;gap:0.25rem}
.dh-field label{font-size:0.875rem;font-weight:500}
.dh-field input,.dh-field textarea,.dh-field select{padding:0.5rem;border:1px solid var(--dh-color-border,#e5e5e5);border-radius:4px;font:inherit}
.dh-search{position:relative}
.dh-nav{display:flex;align-items:center;gap:1rem;flex-wrap:wrap}
.dh-link{color:var(--dh-color-primary,#2563eb);text-decoration:none}
.dh-link:hover{text-decoration:underline}
.dh-breadcrumb ol{display:flex;list-style:none;padding:0;margin:0;gap:0.5rem}
.dh-breadcrumb li+li::before{content:"/";color:#999}
.dh-pagination{display:flex;justify-content:center;gap:1rem}

/* Nav */
.dh-site-nav{background:var(--dh-color-text,#1a1a1a);color:#fff;padding:0 1rem}
.dh-site-nav__inner{max-width:64rem;margin:0 auto;display:flex;align-items:center;justify-content:space-between;height:3rem}
.dh-site-nav__brand{font-weight:700;color:inherit;text-decoration:none}
.dh-site-nav__links{display:flex;gap:1.5rem}
.dh-site-nav__links a{color:inherit;text-decoration:none;opacity:0.8;font-size:0.875rem}
.dh-site-nav__links a:hover{opacity:1}

/* Footer */
.dh-footer{text-align:center;padding:2rem 1rem;margin-top:2rem;border-top:1px solid var(--dh-color-border,#e5e5e5);color:var(--dh-color-muted,#999);font-size:0.8125rem}

/* Responsive */
@media(max-width:768px){
  .dh-columns,.dh-grid,.dh-sidebar{grid-template-columns:1fr !important}
}
`)

	return b.String()
}
