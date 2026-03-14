package render

import (
	"fmt"
	"strings"

	"github.com/drafthaus/drafthaus/internal/draft"
)

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
	default:
		return "0.5rem"
	}
}

func densityMultiplier(d string) float64 {
	switch d {
	case "compact":
		return 0.5
	case "spacious":
		return 1.5
	default:
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

	gap := fmt.Sprintf("%.4grem", 1.0*spacing)
	_ = fmt.Sprintf("%.4grem", 0.5*spacing*density) // padSm - reserved
	_ = fmt.Sprintf("%.4grem", 1.0*spacing*density) // padMd - reserved
	padLg := fmt.Sprintf("%.4grem", 2.0*spacing*density)
	padXl := fmt.Sprintf("%.4grem", 3.0*spacing*density)

	colorVar := func(name string) string {
		return fmt.Sprintf("var(--dh-color-%s, #000)", name)
	}
	fontVar := func(name string) string {
		return fmt.Sprintf("var(--dh-font-%s, sans-serif)", name)
	}

	var b strings.Builder

	// ---- Custom properties --------------------------------------------------
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
	b.WriteString("  --dh-sidebar-width: 300px;\n")
	b.WriteString("}\n\n")

	// ---- Reset --------------------------------------------------------------
	b.WriteString(`*,*::before,*::after{box-sizing:border-box}
body,h1,h2,h3,h4,h5,h6,p,figure,blockquote,dl,dd{margin:0}
ul,ol{margin:0}
html{line-height:1.5;-webkit-text-size-adjust:100%;scroll-behavior:smooth}
img,picture,video,canvas,svg{display:block;max-width:100%}
input,button,textarea,select{font:inherit}
`)
	b.WriteString("\n")

	// ---- Base typography with scale -----------------------------------------
	fmt.Fprintf(&b, `body{
  font-family:%s;
  color:%s;
  background:%s;
  line-height:1.6;
  font-size:1rem;
  -webkit-font-smoothing:antialiased;
  -moz-osx-font-smoothing:grayscale;
}

`, fontVar("body"), colorVar("text"), colorVar("background"))

	fmt.Fprintf(&b, `h1,h2,h3,h4,h5,h6{
  font-family:%s;
  line-height:1.15;
  color:%s;
  letter-spacing:-0.02em;
  font-weight:700;
}

h1{font-size:clamp(2.25rem,5vw,3.5rem);letter-spacing:-0.03em}
h2{font-size:clamp(1.5rem,3vw,2.25rem)}
h3{font-size:clamp(1.25rem,2vw,1.5rem)}
h4{font-size:1.125rem}
h5{font-size:1rem}
h6{font-size:0.875rem}

`, fontVar("heading"), colorVar("text"))

	fmt.Fprintf(&b, `a{color:%s;transition:color 0.15s}
a:hover{opacity:0.8}
p{line-height:1.7}

`, colorVar("primary"))

	// ---- Layout -------------------------------------------------------------
	fmt.Fprintf(&b, `.dh-page{
  max-width:64rem;
  padding:%s %s;
  margin:0 auto;
}

`, padXl, padLg)

	fmt.Fprintf(&b, `.dh-stack{
  display:flex;
  flex-direction:column;
  gap:%s;
}

`, gap)

	fmt.Fprintf(&b, `.dh-columns{
  display:grid;
  gap:%s;
}

.dh-col{min-width:0}

`, padLg)

	fmt.Fprintf(&b, `.dh-grid{
  display:grid;
  gap:%s;
  grid-template-columns:repeat(auto-fill,minmax(18rem,1fr));
}

`, padLg)

	fmt.Fprintf(&b, `.dh-section{
  padding-block:%s;
}

.dh-section:first-child{
  padding-top:%s;
  padding-bottom:%s;
  text-align:center;
}

.dh-section:first-child .dh-heading{
  margin-bottom:0.75rem;
}

.dh-section:first-child .dh-text{
  font-size:1.25rem;
  color:%s;
  max-width:40rem;
  margin-left:auto;
  margin-right:auto;
}

`, padXl, padXl, padLg, colorVar("muted"))

	fmt.Fprintf(&b, `.dh-sidebar{
  display:grid;
  gap:%s;
  grid-template-columns:1fr var(--dh-sidebar-width,300px);
  align-items:start;
}

`, padLg)

	fmt.Fprintf(&b, `.dh-container{
  max-width:64rem;
  margin:0 auto;
  padding:0 %s;
}

`, padLg)

	// ---- Content ------------------------------------------------------------
	fmt.Fprintf(&b, `.dh-text{
  line-height:1.7;
  color:%s;
}

`, colorVar("text"))

	fmt.Fprintf(&b, `.dh-heading{
  font-family:%s;
  line-height:1.15;
}

`, fontVar("heading"))

	fmt.Fprintf(&b, `.dh-richtext{
  line-height:1.8;
  font-size:1.0625rem;
}
.dh-richtext p{margin-bottom:1.25em}
.dh-richtext h1,.dh-richtext h2,.dh-richtext h3{margin-top:2em;margin-bottom:0.75em}
.dh-richtext ul,.dh-richtext ol{margin-bottom:1.25em;padding-left:1.5em}
.dh-richtext li{margin-bottom:0.25em}
.dh-richtext blockquote{
  border-left:3px solid %s;
  padding-left:1.25em;
  margin:1.5em 0;
  color:%s;
  font-style:italic;
}

`, colorVar("primary"), colorVar("muted"))

	fmt.Fprintf(&b, `.dh-image img{
  max-width:100%%;
  height:auto;
  border-radius:%s;
  display:block;
}
.dh-image{margin:1.5rem 0}

`, radius)

	b.WriteString(`.dh-video video{max-width:100%;display:block}
.dh-embed iframe{width:100%;border:none;display:block;aspect-ratio:16/9}

`)

	fmt.Fprintf(&b, `.dh-code{
  background:%s;
  padding:1.25rem 1.5rem;
  border-radius:%s;
  overflow-x:auto;
  font-family:%s;
  font-size:0.875rem;
  line-height:1.6;
  border:1px solid %s;
  margin:1.5rem 0;
}
.dh-code code{background:none;padding:0;font-size:inherit}

`, colorVar("surface"), radius, fontVar("mono"), colorVar("border"))

	// ---- Cards --------------------------------------------------------------
	fmt.Fprintf(&b, `.dh-card{
  border:1px solid %s;
  border-radius:%s;
  padding:%s;
  background:%s;
  transition:all 0.2s ease;
}
.dh-card .dh-heading{
  margin-bottom:0.5rem;
}
.dh-card .dh-text{
  color:%s;
  font-size:0.9375rem;
  line-height:1.6;
}
.dh-card a{
  text-decoration:none;
  color:inherit;
  display:block;
}
.dh-card:hover{
  border-color:%s;
  box-shadow:0 4px 12px rgba(0,0,0,0.06);
  transform:translateY(-2px);
}

`, colorVar("border"), radius, padLg, colorVar("background"), colorVar("muted"), colorVar("primary"))

	// ---- Badges -------------------------------------------------------------
	fmt.Fprintf(&b, `.dh-badge{
  display:inline-block;
  padding:0.25rem 0.75rem;
  border-radius:9999px;
  font-size:0.75rem;
  font-weight:600;
  background:%s;
  color:%s;
  letter-spacing:0.025em;
  text-transform:uppercase;
}
.dh-badge--success{background:#dcfce7;color:#166534}
.dh-badge--warning{background:#fef9c3;color:#854d0e}
.dh-badge--error{background:#fee2e2;color:#991b1b}

`, colorVar("surface"), colorVar("muted"))

	fmt.Fprintf(&b, `.dh-price{
  font-weight:800;
  font-variant-numeric:tabular-nums;
  font-size:1.25rem;
  color:%s;
}

`, colorVar("text"))

	// ---- Interactive --------------------------------------------------------
	fmt.Fprintf(&b, `.dh-action{
  display:inline-flex;
  align-items:center;
  justify-content:center;
  padding:0.75rem 1.75rem;
  border-radius:%s;
  cursor:pointer;
  font-weight:600;
  font-size:0.9375rem;
  text-decoration:none;
  background:%s;
  color:#fff;
  border:2px solid %s;
  transition:all 0.15s ease;
  letter-spacing:0.01em;
}
.dh-action:hover{
  transform:translateY(-1px);
  box-shadow:0 4px 12px rgba(0,0,0,0.15);
  opacity:1;
}
.dh-action--secondary{
  background:transparent;
  color:%s;
  border-color:%s;
}
.dh-action--secondary:hover{
  background:%s;
}

`, radius, colorVar("primary"), colorVar("primary"),
		colorVar("primary"), colorVar("primary"), colorVar("surface"))

	fmt.Fprintf(&b, `.dh-form{display:flex;flex-direction:column;gap:%s}

`, gap)

	fmt.Fprintf(&b, `.dh-field{display:flex;flex-direction:column;gap:0.375rem}
.dh-field label{font-size:0.875rem;font-weight:600;color:%s}
.dh-field input,.dh-field textarea,.dh-field select{
  padding:0.625rem 0.875rem;
  border:1px solid %s;
  border-radius:%s;
  font:inherit;
  background:%s;
  color:%s;
  transition:border-color 0.15s;
}
.dh-field input:focus,.dh-field textarea:focus,.dh-field select:focus{
  outline:none;
  border-color:%s;
  box-shadow:0 0 0 3px rgba(37,99,235,0.1);
}

`, colorVar("text"), colorVar("border"), radius, colorVar("background"), colorVar("text"), colorVar("primary"))

	fmt.Fprintf(&b, `.dh-search{position:relative}
.dh-search__input{
  width:100%%;
  padding:0.625rem 0.875rem;
  border:1px solid %s;
  border-radius:%s;
  font:inherit;
}
.dh-search__results{
  position:absolute;top:100%%;left:0;right:0;
  background:%s;
  border:1px solid %s;
  border-radius:%s;
  z-index:10;
  box-shadow:0 8px 24px rgba(0,0,0,0.1);
}

`, colorVar("border"), radius, colorVar("background"), colorVar("border"), radius)

	// ---- Navigation ---------------------------------------------------------
	fmt.Fprintf(&b, `.dh-nav{display:flex;align-items:center;gap:%s;flex-wrap:wrap}

`, gap)

	fmt.Fprintf(&b, `.dh-link{color:%s;text-decoration:none;transition:color 0.15s}
.dh-link:hover{text-decoration:underline}

`, colorVar("primary"))

	fmt.Fprintf(&b, `.dh-breadcrumb ol{display:flex;list-style:none;padding:0;margin:0;gap:0.5rem;align-items:center}
.dh-breadcrumb li+li::before{content:"/";color:%s}

`, colorVar("muted"))

	fmt.Fprintf(&b, `.dh-pagination{display:flex;justify-content:center;align-items:center;gap:%s}

`, gap)

	// ---- Callout / List / Table ---------------------------------------------
	fmt.Fprintf(&b, `.dh-callout{
  border-left:4px solid %s;
  padding:1rem 1.25rem;
  background:%s;
  border-radius:0 %s %s 0;
  margin:1.5rem 0;
}

`, colorVar("primary"), colorVar("surface"), radius, radius)

	b.WriteString(`.dh-list{padding-left:1.5rem;line-height:1.7}

`)

	fmt.Fprintf(&b, `.dh-table{width:100%%;border-collapse:collapse;font-size:0.9375rem}
.dh-table th,.dh-table td{padding:0.75rem 1rem;border-bottom:1px solid %s;text-align:left}
.dh-table th{font-weight:600;color:%s;font-size:0.8125rem;text-transform:uppercase;letter-spacing:0.05em}
.dh-table tr:hover td{background:%s}

`, colorVar("border"), colorVar("muted"), colorVar("surface"))

	fmt.Fprintf(&b, `.dh-map{
  aspect-ratio:16/9;
  background:%s;
  display:flex;align-items:center;justify-content:center;
  border-radius:%s;
  overflow:hidden;
}

`, colorVar("surface"), radius)

	// ---- Site nav -----------------------------------------------------------
	fmt.Fprintf(&b, `.dh-site-nav{
  background:%s;
  color:%s;
  padding:0 1.5rem;
  position:sticky;
  top:0;
  z-index:100;
  backdrop-filter:blur(8px);
}
.dh-site-nav__inner{
  max-width:64rem;
  margin:0 auto;
  display:flex;
  align-items:center;
  justify-content:space-between;
  height:3.5rem;
}
.dh-site-nav__brand{
  font-weight:800;
  font-size:1.125rem;
  color:inherit;
  text-decoration:none;
  letter-spacing:-0.02em;
}
.dh-site-nav__links{
  display:flex;
  gap:2rem;
}
.dh-site-nav__links a{
  color:inherit;
  text-decoration:none;
  opacity:0.7;
  font-size:0.875rem;
  font-weight:500;
  transition:opacity 0.15s;
  letter-spacing:0.01em;
}
.dh-site-nav__links a:hover{opacity:1}

`, colorVar("text"), colorVar("background"))

	// ---- Footer -------------------------------------------------------------
	fmt.Fprintf(&b, `.dh-footer{
  text-align:center;
  padding:2.5rem 1rem;
  margin-top:4rem;
  border-top:1px solid %s;
  color:%s;
  font-size:0.8125rem;
  letter-spacing:0.01em;
}

`, colorVar("border"), colorVar("muted"))

	// ---- Responsive ---------------------------------------------------------
	b.WriteString(`@media (max-width:768px){
  .dh-columns{grid-template-columns:1fr !important}
  .dh-grid{grid-template-columns:1fr !important}
  .dh-sidebar{grid-template-columns:1fr !important}
  .dh-page{padding:2rem 1rem}
  .dh-section{padding-block:2rem}
  .dh-site-nav__links{gap:1rem}
  .dh-card{padding:1.25rem}
}
`)

	return b.String()
}
