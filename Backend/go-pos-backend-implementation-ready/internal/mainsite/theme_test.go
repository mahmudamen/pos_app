package mainsite

import (
	"strings"
	"testing"
)

// The dual dark/light theme, the toggle and the SVG helpers are the pieces that
// define the site's look. These tests pin the design tokens, the toggle markup
// and the "no emoji tiles" guarantee so a future edit cannot silently regress
// the visual system.

func TestThemeStylesheetCarriesBrandTokens(t *testing.T) {
	for _, want := range []string{
		"--bg:#03010E",
		"--bg-2:#0B0C17",
		"--border:#202128",
		"--primary:#937AFF",
		"--primary-light:#4D36D0",
		"--secondary:#FF5353",
		`Inter Tight`,
		`Urbanist`,
		`IBM Plex Sans Arabic`,
		"--grad:linear-gradient(0deg,var(--primary-light),var(--primary))",
	} {
		if !strings.Contains(siteCSS, want) {
			t.Errorf("stylesheet missing brand token %q", want)
		}
	}
}

func TestThemeStylesheetHasLightVariant(t *testing.T) {
	for _, want := range []string{
		"html[data-theme=light]{",
		"color-scheme:light",
		"--bg:#f5f6fb",
		"--heading:#0c0e18",
		"--primary:#5b3ef0",
	} {
		if !strings.Contains(siteCSS, want) {
			t.Errorf("stylesheet missing light-theme token %q", want)
		}
	}
	// Light and dark headings must be distinct ink, or one theme goes invisible.
	if !strings.Contains(siteCSS, "--heading:#ffffff") {
		t.Error("stylesheet missing dark-theme heading ink")
	}
}

func TestThemeToggleStyled(t *testing.T) {
	for _, want := range []string{
		".theme-toggle{",
		".theme-toggle .i-sun{display:grid}",
		".theme-toggle .i-moon{display:none}",
		"html[data-theme=light] .theme-toggle .i-sun{display:none}",
		"html[data-theme=light] .theme-toggle .i-moon{display:grid}",
	} {
		if !strings.Contains(siteCSS, want) {
			t.Errorf("stylesheet missing theme-toggle rule %q", want)
		}
	}
}

func TestThemeStylesheetDroppedOldPalette(t *testing.T) {
	for _, gone := range []string{"#bf5b3a", "#14a6a6", "#4451c7", "#faf7f2", "brand-mark"} {
		if strings.Contains(siteCSS, gone) {
			t.Errorf("stylesheet still references removed token %q", gone)
		}
	}
}

func TestThemeStylesheetNeutralizesArabicLetterSpacing(t *testing.T) {
	if !strings.Contains(siteCSS, "html[dir=rtl] h1") {
		t.Fatal("stylesheet must reset letter-spacing for RTL headings")
	}
	if !strings.Contains(siteCSS, "letter-spacing:normal") {
		t.Fatal("stylesheet must set letter-spacing:normal for Arabic")
	}
}

func TestIconHTMLRendersSVG(t *testing.T) {
	for _, name := range []string{"whatsapp", "mail", "github", "apps", "sun", "moon"} {
		svg := string(iconHTML(name))
		if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "viewBox") {
			t.Errorf("icon %q did not render an svg: %q", name, svg)
		}
		if !strings.Contains(svg, "currentColor") {
			t.Errorf("icon %q should inherit the surrounding text colour", name)
		}
	}
	if got := string(iconHTML("does-not-exist")); got != "" {
		t.Errorf("unknown icon should render empty, got %q", got)
	}
}

func TestBrandMarkAndFaviconUseVioletGradient(t *testing.T) {
	mark := string(brandMarkHTML())
	for _, want := range []string{"<svg", `class="mark"`, "#4D36D0", "#937AFF", `fill="url(#xm)"`} {
		if !strings.Contains(mark, want) {
			t.Errorf("brand mark missing %q", want)
		}
	}
	for _, want := range []string{"<svg", "#4D36D0", "#937AFF", `rx="16"`} {
		if !strings.Contains(faviconSVG, want) {
			t.Errorf("favicon missing %q", want)
		}
	}
}

func TestRenderedPagesCarryThemeToggleAndNoEmoji(t *testing.T) {
	e := newTestEngine()

	emoji := []string{"💬", "✉️", "🐙", "🛒", "🎯", "📊", "📦", "🔐", "✨", "🌾", "🖧"}
	dead := []string{`class="brand-mark"`, `class="ico"`}

	for _, slug := range registrySlugs() {
		for _, l := range []Lang{EN, AR} {
			path := langURLFor(l, slug)
			body := get(t, e, path).Body.String()

			for _, want := range []string{
				"--primary:#937AFF",
				`class="mark"`,
				"fonts.googleapis.com",
				"IBM+Plex+Sans+Arabic",
				`id="theme-toggle"`,
				"prefers-color-scheme",
				`data-theme`,
				"localStorage",
			} {
				if !strings.Contains(body, want) {
					t.Errorf("%s: missing marker %q", path, want)
				}
			}
			for _, bad := range dead {
				if strings.Contains(body, bad) {
					t.Errorf("%s: stale markup %q rendered", path, bad)
				}
			}
			for _, em := range emoji {
				if strings.Contains(body, em) {
					t.Errorf("%s: emoji %q leaked into the page", path, em)
				}
			}
		}
	}
}

func TestFaviconAndMarkHaveBalancedSVGTags(t *testing.T) {
	for name, svg := range map[string]string{"favicon": faviconSVG, "mark": brandMarkSVG} {
		if got := strings.Count(svg, "<svg"); got != 1 {
			t.Errorf("%s: want exactly one <svg>, got %d", name, got)
		}
		if got, want := strings.Count(svg, "</svg>"), strings.Count(svg, "<svg"); got != want {
			t.Errorf("%s: unbalanced svg tags (%d open, %d close)", name, want, got)
		}
	}
}

func TestThemeIconsAreDistinctShapes(t *testing.T) {
	sun := siteIcons["sun"]
	moon := siteIcons["moon"]
	if sun == "" || moon == "" {
		t.Fatal("sun and moon icons must be defined")
	}
	if sun == moon {
		t.Fatal("sun and moon icons must differ")
	}
}
