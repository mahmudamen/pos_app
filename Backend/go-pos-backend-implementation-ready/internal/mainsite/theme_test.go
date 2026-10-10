package mainsite

import (
	"strings"
	"testing"
)

// The AutoMark theme and the SVG asset helpers are the two pieces that changed
// when the site was re-skinned. These tests pin the design tokens, the icon
// markup and the "no emoji tiles" guarantee so a future edit cannot silently
// regress the visual system.

func TestThemeStylesheetCarriesAutoMarkTokens(t *testing.T) {
	for _, want := range []string{
		"--body:#03010E",
		"--light:#0B0C17",
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
			t.Errorf("stylesheet missing AutoMark token %q", want)
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
	for _, name := range []string{"whatsapp", "mail", "github", "apps"} {
		svg := string(iconHTML(name))
		if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "viewBox") {
			t.Errorf("icon %q did not render an svg: %q", name, svg)
		}
		if strings.Contains(svg, "currentColor") == false {
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

func TestRenderedPagesCarryThemeAndNoEmoji(t *testing.T) {
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
			} {
				if !strings.Contains(body, want) {
					t.Errorf("%s: missing theme marker %q", path, want)
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
