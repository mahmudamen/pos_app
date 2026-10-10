package mainsite

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestEngine() *gin.Engine {
	gin.SetMode(gin.TestMode)
	e := gin.New()
	Register(e)
	return e
}

func get(t *testing.T, e *gin.Engine, path string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	e.ServeHTTP(w, req)
	return w
}

func TestAllPagesRenderInBothLanguages(t *testing.T) {
	e := newTestEngine()
	for _, slug := range registrySlugs() {
		for _, l := range []Lang{EN, AR} {
			path := langURLFor(l, slug)
			w := get(t, e, path)
			if w.Code != http.StatusOK {
				t.Fatalf("%s: want 200, got %d", path, w.Code)
			}
			body := w.Body.String()
			if !strings.HasPrefix(body, "<!doctype html>") {
				t.Fatalf("%s: not an html document", path)
			}
			if l == AR && !strings.Contains(body, `dir="rtl"`) {
				t.Fatalf("%s: arabic page missing dir=rtl", path)
			}
			if l == EN && !strings.Contains(body, `dir="ltr"`) {
				t.Fatalf("%s: english page missing dir=ltr", path)
			}
			if !strings.Contains(body, `lang="`+string(l)+`"`) {
				t.Fatalf("%s: missing html lang", path)
			}
			if !strings.Contains(body, "XamlTech") && !strings.Contains(body, "زامل") {
				t.Fatalf("%s: missing brand in body", path)
			}
			if !strings.Contains(body, `href="`+siteURL+langURLFor(l, slug)+`"`) {
				t.Fatalf("%s: canonical URL missing", path)
			}
		}
	}
}

func TestBothLanguagesPresentOnEachPage(t *testing.T) {
	e := newTestEngine()
	for _, slug := range registrySlugs() {
		w := get(t, e, slug)
		body := w.Body.String()
		if !strings.Contains(body, `hreflang="en"`) || !strings.Contains(body, `hreflang="ar"`) {
			t.Fatalf("%s: missing hreflang alternates", slug)
		}
		if !strings.Contains(body, `href="/ar`+slug+`"`) && !strings.Contains(body, `href="/ar"`) {
			t.Fatalf("%s: missing arabic switch link", slug)
		}
	}
}

func TestArabicRootAndLanguageSwitchRoundTrip(t *testing.T) {
	e := newTestEngine()
	w := get(t, e, "/ar")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `dir="rtl"`) {
		t.Fatalf("GET /ar should render the arabic home: %d", w.Code)
	}
	w = get(t, e, "/ar/solutions/erp")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "أنظمة أودو") {
		t.Fatalf("arabic solution page missing arabic copy: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `href="/solutions/erp"`) {
		t.Fatal("arabic page must link back to the english version")
	}
}

func TestUnknownPathReturns404(t *testing.T) {
	e := newTestEngine()
	for _, p := range []string{"/does-not-exist", "/ar/does-not-exist"} {
		w := get(t, e, p)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s: want 404, got %d", p, w.Code)
		}
		if !strings.Contains(w.Body.String(), "404") {
			t.Fatalf("%s: 404 page missing marker", p)
		}
	}
}

func TestLegacyRedirects(t *testing.T) {
	e := newTestEngine()
	cases := map[string]string{
		"/contactus":    "/contact-us",
		"/contact":      "/contact-us",
		"/private":      "/privacy-policy",
		"/blog":         "/insights",
		"/ar/contactus": "/ar/contact-us",
		"/ar/private":   "/ar/privacy-policy",
	}
	for from, to := range cases {
		w := get(t, e, from)
		if w.Code != http.StatusMovedPermanently {
			t.Fatalf("%s: want 301, got %d", from, w.Code)
		}
		loc := w.Header().Get("Location")
		if loc != to {
			t.Fatalf("%s: want redirect to %s, got %s", from, to, loc)
		}
	}
}

func TestSearchEngineFiles(t *testing.T) {
	e := newTestEngine()

	w := get(t, e, "/robots.txt")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Sitemap: "+siteURL+"/sitemap.xml") {
		t.Fatalf("robots.txt missing sitemap: %d", w.Code)
	}

	w = get(t, e, "/sitemap.xml")
	if w.Code != http.StatusOK {
		t.Fatalf("sitemap: %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `<loc>`+siteURL+`/</loc>`) || !strings.Contains(body, `<loc>`+siteURL+`/ar</loc>`) {
		t.Fatal("sitemap missing home URLs")
	}
	if !strings.Contains(body, `hreflang="ar"`) {
		t.Fatal("sitemap missing hreflang alternates")
	}
	if strings.Contains(body, "privacy-policy") || strings.Contains(body, "/terms") {
		t.Fatal("noindex pages must not appear in the sitemap")
	}

	w = get(t, e, "/favicon.svg")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "<svg") {
		t.Fatalf("favicon: %d", w.Code)
	}
}

func TestSectionsMeetBilingualDuty(t *testing.T) {
	// Every bilingual value used anywhere must have an Arabic twin — a cheap
	// guard that no page mixes languages.
	for _, build := range registry() {
		p := build()
		walkBi(p, func(b bi) {
			if b.En != "" && b.Ar == "" {
				t.Errorf("english-only copy: %q", b.En)
			}
		})
	}
}

// walkBi visits every bi value reachable from a page.
func walkBi(p page, visit func(bi)) {
	visit(p.Title)
	visit(p.Meta)
	visit(p.Eyebrow)
	visit(p.H1)
	visit(p.Lead)
	visit(p.HeroNote)
	for _, l := range p.HeroCTAs {
		visit(l.Label)
	}
	for _, f := range p.HeroTiles {
		visit(f.Title)
	}
	for _, s := range p.Sections {
		walkSection(s, visit)
	}
	for _, f := range p.FAQ {
		visit(f.Q)
		visit(f.A)
	}
	walkSection(p.Final, visit)
}

func walkSection(s section, visit func(bi)) {
	visit(s.Label)
	visit(s.Title)
	visit(s.Lead)
	visit(s.Note)
	for _, f := range s.Features {
		visit(f.Title)
		visit(f.Body)
	}
	for _, c := range s.Cards {
		visit(c.Eyebrow)
		visit(c.Badge)
		visit(c.Title)
		visit(c.Body)
		visit(c.Price)
		visit(c.CTA)
		for _, m := range c.Meta {
			visit(m)
		}
	}
	for _, st := range s.Steps {
		visit(st.Title)
		visit(st.Body)
	}
	for _, pl := range s.Plans {
		visit(pl.Name)
		visit(pl.Desc)
		visit(pl.Price)
		visit(pl.Period)
		visit(pl.Badge)
		visit(pl.CTA)
		for _, f := range pl.Features {
			visit(f)
		}
	}
	for _, si := range s.Stats {
		visit(si.Value)
		visit(si.Label)
	}
	for _, b := range s.Blocks {
		visit(b.Heading)
		for _, x := range b.Body {
			visit(x)
		}
		for _, x := range b.List {
			visit(x)
		}
	}
	for _, f := range s.FAQ {
		visit(f.Q)
		visit(f.A)
	}
	for _, l := range s.Links {
		visit(l.Label)
	}
}

func TestLanguageDefaultsAreTemplatesOnly(t *testing.T) {
	// Guard: h1 must not contain HTML-injectable characters we forgot to escape.
	for _, build := range registry() {
		p := build()
		if strings.Contains(p.H1.En, "<") {
			t.Errorf("unsanitized h1 on %s: %s", p.Slug, p.H1.En)
		}
	}
}
