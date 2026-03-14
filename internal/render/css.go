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

/* Hero with gradient overlays and subtle texture */
.dh-hero{position:relative;overflow:hidden}
.dh-hero::before{content:'';position:absolute;inset:0;background:radial-gradient(ellipse at 20% 80%,rgba(var(--dh-color-primary-rgb,146,64,14),0.3) 0%,transparent 60%),radial-gradient(ellipse at 80% 20%,rgba(var(--dh-color-primary-rgb,146,64,14),0.15) 0%,transparent 50%);pointer-events:none}
.dh-hero::after{content:'';position:absolute;inset:0;background:url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.03'%3E%3Cpath d='M36 34v-4h-2v4h-4v2h4v4h2v-4h4v-2h-4zm0-30V0h-2v4h-4v2h4v4h2V6h4V4h-4zM6 34v-4H4v4H0v2h4v4h2v-4h4v-2H6zM6 4V0H4v4H0v2h4v4h2V6h4V4H6z'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E");pointer-events:none}
.dh-hero>*{position:relative;z-index:1}

/* Ornamental divider */
.dh-ornament{display:flex;align-items:center;gap:1rem;justify-content:center;margin-top:1.25rem;color:var(--dh-color-primary,#92400E)}
.dh-ornament::before,.dh-ornament::after{content:'';width:3rem;height:1px;background:var(--dh-color-primary,#92400E);opacity:0.4}

/* Feature icon circle */
.dh-icon-circle{width:4rem;height:4rem;margin:0 auto 1.5rem;border-radius:50%;display:flex;align-items:center;justify-content:center;font-size:1.5rem;background:linear-gradient(135deg,rgba(var(--dh-color-primary-rgb,146,64,14),0.1),rgba(var(--dh-color-primary-rgb,146,64,14),0.2));color:var(--dh-color-primary,#92400E)}

/* Card image placeholder (gradient fill when no image) */
.dh-card-img{height:12rem;border-radius:0.75rem 0.75rem 0 0;position:relative;overflow:hidden}
.dh-card-img--dark{background:linear-gradient(135deg,#44403C 0%,#292524 50%,#1C1917 100%)}
.dh-card-img--warm{background:linear-gradient(135deg,#FDE68A 0%,#F59E0B 50%,#D97706 100%)}
.dh-card-img--cool{background:linear-gradient(135deg,#93C5FD 0%,#3B82F6 50%,#1D4ED8 100%)}
.dh-card-img--green{background:linear-gradient(135deg,#86EFAC 0%,#22C55E 50%,#15803D 100%)}
.dh-card-img--neutral{background:linear-gradient(135deg,#D6D3D1 0%,#A8A29E 100%)}
.dh-card-img::after{content:'';position:absolute;inset:0;background:url("data:image/svg+xml,%3Csvg width='40' height='40' viewBox='0 0 40 40' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='%23ffffff' fill-opacity='0.08'%3E%3Ccircle cx='20' cy='20' r='3'/%3E%3C/g%3E%3C/svg%3E")}

/* Stats bar */
.dh-stats{display:grid;grid-template-columns:repeat(auto-fit,minmax(120px,1fr));gap:2rem;text-align:center}
.dh-stats .number{font-family:var(--dh-font-heading,system-ui,sans-serif);font-size:2.5rem;font-weight:800;line-height:1;margin-bottom:0.5rem}
.dh-stats .label{font-size:0.8rem;text-transform:uppercase;letter-spacing:0.1em;opacity:0.8}

/* Story quote decoration */
.dh-quote-bg{position:relative}
.dh-quote-bg::before{content:'"';position:absolute;top:50%;left:50%;transform:translate(-50%,-50%);font-family:var(--dh-font-heading,serif);font-size:12rem;color:rgba(var(--dh-color-primary-rgb,146,64,14),0.15);line-height:1;pointer-events:none}

/* Signature line */
.dh-signature{border-top:1px solid rgba(255,255,255,0.1);padding-top:1.5rem;margin-top:2rem}

@media(max-width:768px){.dh-columns,.dh-grid,.dh-sidebar{grid-template-columns:1fr !important}.dh-stats{grid-template-columns:repeat(2,1fr)}}
`)
	return b.String()
}
