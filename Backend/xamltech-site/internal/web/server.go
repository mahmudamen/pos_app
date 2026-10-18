// Package web serves the bilingual XamlTech marketing site.
package web

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/xamltech/xamltech-site/internal/config"
	"github.com/xamltech/xamltech-site/internal/content"
	"github.com/xamltech/xamltech-site/internal/i18n"
)

//go:embed templates
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

// route maps a URL path to a page slug and its content template name.
type route struct {
	Path    string // URL path, e.g. "/solutions/erp"
	Slug    string // content file + template name
	Index   bool   // whether the page should be indexed / listed in the sitemap
	Sitemap bool   // whether the page belongs in sitemap.xml
}

// routes is the full public route table. The first entry is the home page.
var routes = []route{
	{Path: "/", Slug: "home", Index: true, Sitemap: true},
	{Path: "/solutions/erp", Slug: "solutions-erp", Index: true, Sitemap: true},
	{Path: "/solutions/it-infrastructure", Slug: "solutions-it-infrastructure", Index: true, Sitemap: true},
	{Path: "/solutions/web", Slug: "solutions-web", Index: true, Sitemap: true},
	{Path: "/odoo-pos-egypt", Slug: "odoo-pos-egypt", Index: true, Sitemap: true},
	{Path: "/erp-agriculture-egypt", Slug: "erp-agriculture-egypt", Index: true, Sitemap: true},
	{Path: "/addons-store", Slug: "addons-store", Index: true, Sitemap: true},
	{Path: "/ai-solutions", Slug: "ai-solutions", Index: true, Sitemap: true},
	{Path: "/pricing", Slug: "pricing", Index: true, Sitemap: true},
	{Path: "/portfolio", Slug: "portfolio", Index: true, Sitemap: true},
	{Path: "/testimonials", Slug: "testimonials", Index: true, Sitemap: true},
	{Path: "/knowledge", Slug: "knowledge", Index: true, Sitemap: true},
	{Path: "/insights", Slug: "insights", Index: true, Sitemap: true},
	{Path: "/insights/ai-marketing-copilot", Slug: "insights-ai-marketing-copilot", Index: true, Sitemap: true},
	{Path: "/insights/ai-pos-copilot", Slug: "insights-ai-pos-copilot", Index: true, Sitemap: true},
	{Path: "/networktool", Slug: "networktool", Index: true, Sitemap: true},
	{Path: "/apk-download", Slug: "apk-download", Index: true, Sitemap: true},
	{Path: "/request-demo", Slug: "request-demo", Index: true, Sitemap: true},
	{Path: "/contact-us", Slug: "contact-us", Index: true, Sitemap: true},
	{Path: "/theme-info", Slug: "theme-info", Index: true, Sitemap: true},
	{Path: "/privacy-policy", Slug: "privacy-policy", Index: true, Sitemap: true},
	{Path: "/terms", Slug: "terms", Index: true, Sitemap: true},
}

// aliases redirect legacy paths to canonical ones.
var aliases = map[string]string{
	"/contactus": "/contact-us",
	"/home":      "/",
	"/blogs":     "/insights",
}

// Server is the HTTP handler for the whole site.
type Server struct {
	cfg       config.Config
	store     *content.Store
	tpl       map[string]*template.Template // slug -> template tree
	byPath    map[string]route
	static    http.Handler
	baseHost  string
	startedAt time.Time
	log       *slog.Logger
}

// New builds a Server, parsing all templates and wiring the file server.
func New(cfg config.Config, store *content.Store, log *slog.Logger) (*Server, error) {
	if log == nil {
		log = slog.Default()
	}
	s := &Server{
		cfg:       cfg,
		store:     store,
		tpl:       map[string]*template.Template{},
		byPath:    map[string]route{},
		baseHost:  strings.TrimPrefix(strings.TrimPrefix(cfg.BaseURL, "https://"), "http://"),
		startedAt: time.Now(),
		log:       log,
	}
	for _, r := range routes {
		s.byPath[r.Path] = r
		t, err := s.buildTemplate(r.Slug)
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", r.Slug, err)
		}
		s.tpl[r.Slug] = t
	}
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, err
	}
	s.static = http.StripPrefix("/static/", cacheControl(http.FileServer(http.FS(sub))))
	return s, nil
}

func (s *Server) buildTemplate(slug string) (*template.Template, error) {
	// Partials (including the generic "content" fallback) and the layout are
	// always parsed; a page-specific template, when present, is parsed last so
	// its {{define "content"}} overrides the default.
	patterns := []string{
		"templates/partials/*.html",
		"templates/layout.html",
	}
	if _, err := templatesFS.ReadFile("templates/pages/" + slug + ".html"); err == nil {
		patterns = append(patterns, "templates/pages/"+slug+".html")
	}
	return template.New("layout").Funcs(funcs()).ParseFS(templatesFS, patterns...)
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/robots.txt", s.handleRobots)
	mux.HandleFunc("/sitemap.xml", s.handleSitemap)
	mux.HandleFunc("/llms.txt", s.handleLlms)
	mux.HandleFunc("/app-ads.txt", s.handleAppsAds)
	mux.HandleFunc("/BingSiteAuth.xml", s.handleBing)
	mux.HandleFunc("/manifest.webmanifest", s.handleManifest)
	mux.HandleFunc("/manifest.json", s.handleManifest)
	mux.HandleFunc("/sw.js", s.handleServiceWorker)
	mux.HandleFunc("/googlea7b16c2b9c88eee4.html", s.handleGoogleVerify)
	mux.HandleFunc("/api/contact", s.handleContact)
	mux.HandleFunc("/api/demo", s.handleDemo)
	mux.HandleFunc("/api/apk", s.handleApk)
	mux.HandleFunc("/", s.handleRoot)
	mux.Handle("/static/", s.static)
	return s.withMiddleware(mux)
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	p := path.Clean(r.URL.Path)
	if p != "/" && strings.HasSuffix(r.URL.Path, "/") {
		// normalise trailing slashes (except root) to the bare path
		p = strings.TrimSuffix(p, "/")
	}
	if target, ok := aliases[p]; ok {
		http.Redirect(w, r, s.langURL(r, target), http.StatusMovedPermanently)
		return
	}
	if r.URL.Path == "/favicon.ico" {
		s.serveStaticFile(w, r, "img/favicon.ico", "image/x-icon")
		return
	}
	rt, ok := s.byPath[p]
	if !ok {
		s.renderNotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.renderPage(w, r, rt)
}

func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, rt route) {
	lang := i18n.FromRequest(r, i18n.Lang(s.cfg.DefLang))
	// An explicit ?lang= wins and is remembered, but we render it in place
	// (no redirect) so the content matches the canonical/hreflang URL even
	// for clients that do not store cookies.
	if q := r.URL.Query().Get("lang"); q != "" && i18n.Valid(q) {
		lang = i18n.Normalize(q, i18n.Lang(s.cfg.DefLang))
		s.setLangCookie(w, lang)
	}
	c, ok := s.store.Page(rt.Slug, string(lang))
	if !ok {
		s.log.Error("missing content", "slug", rt.Slug, "lang", lang)
		http.Error(w, "content unavailable", http.StatusInternalServerError)
		return
	}
	data := s.pageData(r, rt, lang, c)
	t, ok := s.tpl[rt.Slug]
	if !ok {
		http.Error(w, "template missing", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var buf strings.Builder
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		s.log.Error("render failed", "slug", rt.Slug, "err", err)
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Length", fmt.Sprint(buf.Len()))
	gzipIfAccepted(w, r, buf.String())
}

func (s *Server) renderNotFound(w http.ResponseWriter, r *http.Request) {
	lang := i18n.FromRequest(r, i18n.Lang(s.cfg.DefLang))
	c, _ := s.store.Page("home", string(lang))
	data := s.pageData(r, route{Path: r.URL.Path, Slug: "home"}, lang, c)
	data.Meta.Title = "404 — " + s.stringOr(data.Strings, "brand", "XamlTech")
	data.Meta.Index = false
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_ = s.tpl["home"].ExecuteTemplate(w, "layout", data)
}

// pageData assembles the view model for a page render.
func (s *Server) pageData(r *http.Request, rt route, lang i18n.Lang, c map[string]any) pageData {
	meta := metaFrom(c)
	if !rt.Index {
		meta.Index = false
	}
	canonical := s.cfg.BaseURL + rt.Path
	if lang == i18n.EN {
		canonical = s.cfg.BaseURL + rt.Path + "?lang=en"
	}
	arURL := s.cfg.BaseURL + rt.Path
	enURL := s.cfg.BaseURL + rt.Path + "?lang=en"
	return pageData{
		Lang:       string(lang),
		Dir:        lang.Dir(),
		Locale:     lang.Locale(),
		SelfLabel:  lang.Label(),
		Other:      string(lang.Other()),
		OtherLabel: lang.Other().Label(),
		Slug:       rt.Slug,
		Path:       rt.Path,
		Meta:       meta,
		Strings:    s.store.Strings(string(lang)),
		C:          c,
		Year:       time.Now().Year(),
		BaseURL:    s.cfg.BaseURL,
		Canonical:  canonical,
		Alternates: []alternate{
			{Lang: "ar", Href: arURL},
			{Lang: "en", Href: enURL},
			{Lang: "x-default", Href: arURL},
		},
		Nav:       s.buildNav(lang, s.store.Strings(string(lang))["nav"]),
		Footer:    s.buildFooter(lang, s.store.Strings(string(lang))["footer"]),
		Contact:   contactMap(s.store.Strings(string(lang))["contact"]),
		ClarityID: s.cfg.ClarityID,
		JSONLD:    template.JS(s.jsonLD(rt, lang, meta)),
		Query:     r.URL.Query().Get("sent"),
	}
}

func (s *Server) serveStaticFile(w http.ResponseWriter, r *http.Request, name, contentType string) {
	data, err := staticFS.ReadFile("static/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(data)
}

func (s *Server) setLangCookie(w http.ResponseWriter, lang i18n.Lang) {
	http.SetCookie(w, &http.Cookie{
		Name:     i18n.CookieName,
		Value:    string(lang),
		Path:     "/",
		MaxAge:   365 * 24 * 3600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   s.cfg.IsProduction(),
	})
}

func (s *Server) cleanPath(p string) string {
	if _, ok := s.byPath[p]; ok {
		return p
	}
	if t, ok := aliases[p]; ok {
		return t
	}
	return p
}

// langURL builds a path carrying the current language query for redirects.
func (s *Server) langURL(r *http.Request, target string) string {
	lang := i18n.FromRequest(r, i18n.Lang(s.cfg.DefLang))
	if lang == i18n.AR {
		return target
	}
	return target + "?lang=" + string(lang)
}

func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if s.cfg.IsProduction() {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		start := time.Now()
		lrw := &loggingWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(lrw, r)
		s.log.Info("request",
			"method", r.Method, "path", r.URL.Path, "status", lrw.status,
			"dur_ms", time.Since(start).Milliseconds())
	})
}

func cacheControl(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=604800")
		h.ServeHTTP(w, r)
	})
}

// gzipIfAccepted writes body, compressed when the client supports it.
func gzipIfAccepted(w http.ResponseWriter, r *http.Request, body string) {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		_, _ = io.WriteString(w, body)
		return
	}
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Del("Content-Length")
	gz := gzip.NewWriter(w)
	_, _ = io.WriteString(gz, body)
	_ = gz.Close()
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"uptime_sec": int(time.Since(s.startedAt).Seconds()),
		"pages":      len(routes),
	})
}
