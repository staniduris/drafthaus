package render

import (
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
)

// radiusValue maps a radius token string to a CSS value.
func radiusValue(r string) string {
	switch r {
	case "none":
		return "0"
	case "sm":
		return "0.25rem"
	case "lg":
		return "1rem"
	case "full":
		return "9999px"
	default: // "md" or unrecognised
		return "0.5rem"
	}
}

// densityMultiplier returns a padding multiplier for a density token.
func densityMultiplier(d string) float64 {
	switch d {
	case "compact":
		return 0.5
	case "spacious":
		return 1.5
	default: // "comfortable"
		return 1.0
	}
}

// GenerateCSS produces a complete stylesheet from design tokens.
func GenerateCSS(tokens *draft.Tokens) string {
	if tokens == nil {
		tokens = &draft.Tokens{
			Colors: map[string]string{},
			Fonts:  map[string]string{},
			Scale:  draft.ScaleTokens{Spacing: 1.0, Radius: "md", Density: "comfortable"},
		}
	}

	radius := radiusValue(tokens.Scale.Radius)
	density := densityMultiplier(tokens.Scale.Density)
	spacing := tokens.Scale.Spacing
	if spacing <= 0 {
		spacing = 1.0
	}
	// Base gap and padding units.
	gap := fmt.Sprintf("%.4grem", 1.0*spacing)
	padSm := fmt.Sprintf("%.4grem", 0.5*spacing*density)
	padMd := fmt.Sprintf("%.4grem", 1.0*spacing*density)
	padLg := fmt.Sprintf("%.4grem", 1.5*spacing*density)
	padXl := fmt.Sprintf("%.4grem", 2.0*spacing*density)

	colorVar := func(name string) string {
		return fmt.Sprintf("var(--dh-color-%s, #000)", name)
	}
	fontVar := func(name string) string {
		return fmt.Sprintf("var(--dh-font-%s, sans-serif)", name)
	}

	var b strings.Builder

	// ---- CSS custom properties -------------------------------------------
	b.WriteString(":root {\n")
	for name, val := range tokens.Colors {
		fmt.Fprintf(&b, "  --dh-color-%s: %s;\n", name, val)
	}
	for name, val := range tokens.Fonts {
		fmt.Fprintf(&b, "  --dh-font-%s: %s;\n", name, val)
	}
	fmt.Fprintf(&b, "  --dh-spacing: %.4grem;\n", spacing)
	fmt.Fprintf(&b, "  --dh-radius: %s;\n", radius)
	fmt.Fprintf(&b, "  --dh-density: %.4g;\n", density)
	fmt.Fprintf(&b, "  --dh-sidebar-width: 300px;\n")
	b.WriteString("}\n\n")

	// ---- Reset --------------------------------------------------------------
	b.WriteString(`*,*::before,*::after{box-sizing:border-box}
body,h1,h2,h3,h4,h5,h6,p,figure,blockquote,dl,dd{margin:0}
html{line-height:1.5;-webkit-text-size-adjust:100%}
`)
	b.WriteString("\n")

	// ---- Base typography ----------------------------------------------------
	fmt.Fprintf(&b, "body{\n  font-family:%s;\n  color:%s;\n  background:%s;\n  line-height:1.6;\n}\n\n",
		fontVar("body"),
		colorVar("text"),
		colorVar("background"),
	)
	fmt.Fprintf(&b, "h1,h2,h3,h4,h5,h6{\n  font-family:%s;\n  line-height:1.2;\n  color:%s;\n}\n\n",
		fontVar("heading"),
		colorVar("text"),
	)
	fmt.Fprintf(&b, "a{color:%s;}\n\n", colorVar("primary"))

	// ---- Layout primitives --------------------------------------------------
	fmt.Fprintf(&b, ".dh-page{\n  max-width:72rem;\n  padding:%s %s;\n  margin:0 auto;\n}\n\n",
		padLg, padMd)

	fmt.Fprintf(&b, ".dh-stack{\n  display:flex;\n  flex-direction:column;\n  gap:%s;\n}\n\n", gap)

	fmt.Fprintf(&b, ".dh-columns{\n  display:grid;\n  gap:%s;\n}\n\n", gap)

	b.WriteString(".dh-col{\n  min-width:0;\n}\n\n")

	fmt.Fprintf(&b, ".dh-grid{\n  display:grid;\n  gap:%s;\n  grid-template-columns:repeat(auto-fill,minmax(16rem,1fr));\n}\n\n", gap)

	fmt.Fprintf(&b, ".dh-section{\n  padding-block:%s;\n}\n\n", padXl)

	fmt.Fprintf(&b, ".dh-sidebar{\n  display:grid;\n  gap:%s;\n  grid-template-columns:1fr var(--dh-sidebar-width,300px);\n  align-items:start;\n}\n\n", gap)

	fmt.Fprintf(&b, ".dh-container{\n  max-width:72rem;\n  margin:0 auto;\n  padding:0 %s;\n}\n\n", padMd)

	// ---- Content primitives -------------------------------------------------
	fmt.Fprintf(&b, ".dh-text{\n  line-height:1.6;\n  margin-bottom:%s;\n}\n\n", padSm)

	fmt.Fprintf(&b, ".dh-heading{\n  font-family:%s;\n  margin-bottom:%s;\n  line-height:1.2;\n}\n\n",
		fontVar("heading"), padSm)

	fmt.Fprintf(&b, ".dh-richtext{\n  line-height:1.7;\n}\n")
	fmt.Fprintf(&b, ".dh-richtext p{margin-bottom:%s}\n\n", padSm)

	fmt.Fprintf(&b, ".dh-image img{\n  max-width:100%%;\n  height:auto;\n  border-radius:%s;\n  display:block;\n}\n\n", radius)

	b.WriteString(".dh-video video{\n  max-width:100%;\n  display:block;\n}\n\n")

	b.WriteString(".dh-embed iframe{\n  width:100%;\n  border:none;\n  display:block;\n}\n\n")

	fmt.Fprintf(&b, ".dh-code{\n  background:%s;\n  padding:%s;\n  border-radius:%s;\n  overflow-x:auto;\n  font-family:%s;\n  font-size:0.875em;\n}\n\n",
		colorVar("surface"), padMd, radius, fontVar("mono"))

	// ---- Data primitives ----------------------------------------------------
	fmt.Fprintf(&b, ".dh-card{\n  border:1px solid %s;\n  border-radius:%s;\n  padding:%s;\n  background:%s;\n}\n\n",
		colorVar("border"), radius, padMd, colorVar("surface"))

	fmt.Fprintf(&b, ".dh-badge{\n  display:inline-block;\n  padding:%s %s;\n  border-radius:%s;\n  font-size:0.75rem;\n  font-weight:600;\n  background:%s;\n  color:%s;\n}\n",
		padSm, padMd, radius, colorVar("surface"), colorVar("text"))
	fmt.Fprintf(&b, ".dh-badge--success{background:#dcfce7;color:#166534;}\n")
	fmt.Fprintf(&b, ".dh-badge--warning{background:#fef9c3;color:#854d0e;}\n")
	fmt.Fprintf(&b, ".dh-badge--error{background:#fee2e2;color:#991b1b;}\n\n")

	fmt.Fprintf(&b, ".dh-price{\n  font-weight:700;\n  font-variant-numeric:tabular-nums;\n  color:%s;\n}\n\n", colorVar("text"))

	// ---- Interactive primitives ---------------------------------------------
	fmt.Fprintf(&b, ".dh-action{\n  display:inline-flex;\n  align-items:center;\n  justify-content:center;\n  padding:%s %s;\n  border-radius:%s;\n  cursor:pointer;\n  font-weight:600;\n  text-decoration:none;\n  background:%s;\n  color:#fff;\n  border:2px solid %s;\n  transition:opacity 0.15s;\n}\n",
		padSm, padMd, radius, colorVar("primary"), colorVar("primary"))
	fmt.Fprintf(&b, ".dh-action:hover{opacity:0.85;}\n")
	fmt.Fprintf(&b, ".dh-action--secondary{\n  background:transparent;\n  color:%s;\n  border-color:%s;\n}\n\n",
		colorVar("primary"), colorVar("primary"))

	fmt.Fprintf(&b, ".dh-form{\n  display:flex;\n  flex-direction:column;\n  gap:%s;\n}\n\n", gap)

	fmt.Fprintf(&b, ".dh-field{\n  display:flex;\n  flex-direction:column;\n  gap:0.25rem;\n}\n")
	fmt.Fprintf(&b, ".dh-field label{\n  font-size:0.875rem;\n  font-weight:500;\n}\n")
	fmt.Fprintf(&b, ".dh-field input,.dh-field textarea,.dh-field select{\n  padding:%s;\n  border:1px solid %s;\n  border-radius:%s;\n  font:inherit;\n  background:%s;\n  color:%s;\n}\n\n",
		padSm, colorVar("border"), radius, colorVar("background"), colorVar("text"))

	fmt.Fprintf(&b, ".dh-search{\n  position:relative;\n}\n")
	fmt.Fprintf(&b, ".dh-search__input{\n  width:100%%;\n  padding:%s;\n  border:1px solid %s;\n  border-radius:%s;\n  font:inherit;\n}\n",
		padSm, colorVar("border"), radius)
	fmt.Fprintf(&b, ".dh-search__results{\n  position:absolute;\n  top:100%%;\n  left:0;\n  right:0;\n  background:%s;\n  border:1px solid %s;\n  border-radius:%s;\n  z-index:10;\n}\n\n",
		colorVar("background"), colorVar("border"), radius)

	// ---- Navigation primitives ----------------------------------------------
	fmt.Fprintf(&b, ".dh-nav{\n  display:flex;\n  align-items:center;\n  gap:%s;\n  flex-wrap:wrap;\n}\n\n", gap)

	fmt.Fprintf(&b, ".dh-link{\n  color:%s;\n  text-decoration:none;\n}\n", colorVar("primary"))
	b.WriteString(".dh-link:hover{text-decoration:underline;}\n\n")

	b.WriteString(".dh-breadcrumb ol{\n  display:flex;\n  list-style:none;\n  padding:0;\n  margin:0;\n  gap:0.5rem;\n  align-items:center;\n}\n")
	fmt.Fprintf(&b, ".dh-breadcrumb li+li::before{\n  content:\"/\";\n  color:%s;\n}\n\n", colorVar("muted"))

	fmt.Fprintf(&b, ".dh-pagination{\n  display:flex;\n  justify-content:center;\n  align-items:center;\n  gap:%s;\n}\n\n", gap)

	// ---- Callout / List / Table ---------------------------------------------
	fmt.Fprintf(&b, ".dh-callout{\n  border-left:4px solid %s;\n  padding:%s %s;\n  background:%s;\n  border-radius:0 %s %s 0;\n}\n\n",
		colorVar("primary"), padSm, padMd, colorVar("surface"), radius, radius)

	fmt.Fprintf(&b, ".dh-list{\n  padding-left:1.5rem;\n  line-height:1.7;\n}\n\n")

	fmt.Fprintf(&b, ".dh-table{\n  width:100%%;\n  border-collapse:collapse;\n  font-size:0.9rem;\n}\n")
	fmt.Fprintf(&b, ".dh-table th,.dh-table td{\n  padding:%s;\n  border-bottom:1px solid %s;\n  text-align:left;\n}\n",
		padSm, colorVar("border"))
	fmt.Fprintf(&b, ".dh-table th{\n  font-weight:600;\n  color:%s;\n}\n\n", colorVar("secondary"))

	// ---- Map ----------------------------------------------------------------
	fmt.Fprintf(&b, ".dh-map{\n  aspect-ratio:16/9;\n  background:%s;\n  display:flex;\n  align-items:center;\n  justify-content:center;\n  border-radius:%s;\n  overflow:hidden;\n}\n\n",
		colorVar("surface"), radius)

	// ---- Site nav -----------------------------------------------------------
	b.WriteString(".dh-site-nav{\n  background:var(--dh-color-text);\n  color:var(--dh-color-background);\n  padding:0 1rem;\n}\n")
	b.WriteString(".dh-site-nav__inner{\n  max-width:72rem;\n  margin:0 auto;\n  display:flex;\n  align-items:center;\n  justify-content:space-between;\n  height:3.5rem;\n}\n")
	b.WriteString(".dh-site-nav__brand{\n  font-weight:700;\n  font-size:1.125rem;\n  color:inherit;\n  text-decoration:none;\n}\n")
	b.WriteString(".dh-site-nav__links{\n  display:flex;\n  gap:1.5rem;\n}\n")
	b.WriteString(".dh-site-nav__links a{\n  color:inherit;\n  text-decoration:none;\n  opacity:0.8;\n}\n")
	b.WriteString(".dh-site-nav__links a:hover{\n  opacity:1;\n}\n\n")

	// ---- Card links --------------------------------------------------------
	b.WriteString(".dh-card a{\n  text-decoration:none;\n  color:inherit;\n  display:block;\n}\n")
	b.WriteString(".dh-card:has(a):hover{\n  box-shadow:0 2px 8px rgba(0,0,0,0.08);\n  transform:translateY(-1px);\n  transition:all 0.15s;\n}\n\n")

	// ---- Footer ------------------------------------------------------------
	b.WriteString(".dh-footer{\n  text-align:center;\n  padding:2rem 1rem;\n  margin-top:3rem;\n  border-top:1px solid var(--dh-color-border);\n  color:var(--dh-color-muted);\n  font-size:0.875rem;\n}\n\n")

	// ---- Responsive ---------------------------------------------------------
	b.WriteString("@media (max-width:768px){\n")
	b.WriteString("  .dh-columns{grid-template-columns:1fr !important;}\n")
	b.WriteString("  .dh-grid{grid-template-columns:1fr !important;}\n")
	b.WriteString("  .dh-sidebar{grid-template-columns:1fr !important;}\n")
	fmt.Fprintf(&b, "  .dh-page{padding:%s %s;}\n", padMd, padSm)
	fmt.Fprintf(&b, "  .dh-section{padding-block:%s;}\n", padLg)
	b.WriteString("}\n")

	return b.String()
}
