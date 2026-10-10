package mainsite

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// siteURL is the canonical public origin for the company website.
const siteURL = "https://xamltech.com"

// redirects maps legacy paths (both languages) to their new homes.
var redirects = map[string]string{
	"/contactus": "/contact-us",
	"/contact":   "/contact-us",
	"/private":   "/privacy-policy",
	"/privacy":   "/privacy-policy",
	"/store":     "/addons-store",
	"/shop":      "/addons-store",
	"/blog":      "/insights",
}

// viewData is the resolved, language-specific data a page template needs.
type viewData struct {
	Lang     Lang
	HTMLang  string
	Dir      string
	Year     string
	Brand    string
	BrandSub bi
	BrandTag bi
	Active   string
	Nav      []navGroup
	Footer   []footerCol
	Page     page

	AltHref  string
	AltLabel bi

	Canonical string
	EnHref    string
	ArHref    string

	DemoCTA bi
	Rights  bi
	Privacy bi
	Terms   bi
	More    bi

	Whatsapp  string
	Github    string
	Email     string
	OdooApps  string
	PlayStore string

	CSS     template.CSS
	OrgJSON template.JS
}

// footerCol is a footer link column.
type footerCol struct {
	Title bi
	Links []link
}

func footerCols() []footerCol {
	return []footerCol{
		{Title: more("Solutions", "الحلول"), Links: []link{
			{Label: more("ERP & Odoo", "أنظمة أودو و ERP"), Href: "/solutions/erp"},
			{Label: more("IT Infrastructure", "البنية التحتية"), Href: "/solutions/it-infrastructure"},
			{Label: more("Web Development", "تطوير المواقع"), Href: "/solutions/web"},
			{Label: more("AI Copilots", "المساعدون الأذكياء"), Href: "/ai-solutions"},
			{Label: more("Agriculture ERP", "ERP الزراعة"), Href: "/erp-agriculture-egypt"},
		}},
		{Title: more("Products", "المنتجات"), Links: []link{
			{Label: more("Odoo POS", "نقاط بيع أودو"), Href: "/odoo-pos-egypt"},
			{Label: more("Addons Store", "متجر الإضافات"), Href: "/addons-store"},
			{Label: more("Pricing", "الأسعار"), Href: "/pricing"},
			{Label: more("Network Copilot", "Network Copilot"), Href: "/networktool"},
			{Label: more("Download Apps", "حمّل التطبيقات"), Href: "/apk-download"},
		}},
		{Title: more("Company", "الشركة"), Links: []link{
			{Label: more("About", "من نحن"), Href: "/about"},
			{Label: more("Portfolio", "أعمالنا"), Href: "/portfolio"},
			{Label: more("Testimonials", "آراء العملاء"), Href: "/testimonials"},
			{Label: more("Insights", "المدونة"), Href: "/insights"},
			{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us"},
		}},
	}
}

// funcs is the template function map.
var funcs = template.FuncMap{
	"t": func(b bi, l Lang) string { return b.L(l) },
	"langURL": func(l Lang, path string) string {
		if path == "" || strings.HasPrefix(path, "http") || strings.HasPrefix(path, "mailto:") ||
			strings.HasPrefix(path, "tel:") || strings.HasPrefix(path, "#") {
			return path
		}
		if l == AR {
			if path == "/" {
				return "/ar"
			}
			return "/ar" + path
		}
		return path
	},
	"dict": func(kv ...any) (map[string]any, error) {
		if len(kv)%2 != 0 {
			return nil, fmt.Errorf("dict: odd number of arguments")
		}
		m := make(map[string]any, len(kv)/2)
		for i := 0; i < len(kv); i += 2 {
			k, ok := kv[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict: key %v is not a string", kv[i])
			}
			m[k] = kv[i+1]
		}
		return m, nil
	},
	"moreText": func() bi { return more("Learn more", "اعرف المزيد") },
	"icon":     iconHTML,
	"mark":     brandMarkHTML,
}

var siteTmpl = template.Must(template.New("site").Funcs(funcs).Parse(siteTemplates))

// orgJSON renders the Organization structured-data block.
func orgJSON() template.JS {
	doc := map[string]any{
		"@context": "https://schema.org",
		"@type":    "Organization",
		"name":     brandName,
		"url":      siteURL,
		"logo":     siteURL + "/favicon.svg",
		"email":    emailMain,
		"address": map[string]any{
			"@type":           "PostalAddress",
			"addressLocality": "Aswan",
			"addressCountry":  "EG",
		},
		"areaServed": "EG",
		"sameAs": []string{
			githubURL,
			odooAppsURL,
		},
	}
	b, _ := json.Marshal(doc)
	return template.JS(b)
}

// normalizePath strips a leading /ar prefix and reports the active language.
func normalizePath(raw string) (Lang, string) {
	lang := EN
	path := raw
	if raw == "/ar" {
		return AR, "/"
	}
	if strings.HasPrefix(raw, "/ar/") {
		lang = AR
		path = strings.TrimPrefix(raw, "/ar")
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	if path == "" {
		path = "/"
	}
	return lang, path
}

// buildView resolves the language-specific view model for a page.
func buildView(p page, lang Lang, path string) viewData {
	alt := lang.Other()
	altLabel := more("العربية", "English")
	brand := brandName
	htmlang := "en"
	if lang == AR {
		brand = brandNameAr
		htmlang = "ar"
	}
	return viewData{
		Lang:      lang,
		HTMLang:   htmlang,
		Dir:       lang.Dir(),
		Year:      time.Now().Format("2006"),
		Brand:     brand,
		BrandSub:  more(brandSub, brandSubAr),
		BrandTag:  more(brandTagline, brandTagAr),
		Active:    p.Nav,
		Nav:       nav(),
		Footer:    footerCols(),
		Page:      p,
		AltHref:   langURLFor(alt, path),
		AltLabel:  altLabel,
		Canonical: siteURL + langURLFor(lang, path),
		EnHref:    siteURL + langURLFor(EN, path),
		ArHref:    siteURL + langURLFor(AR, path),
		DemoCTA:   more("Free Demo", "تجربة مجانية"),
		Rights:    more("All rights reserved.", "جميع الحقوق محفوظة."),
		Privacy:   more("Privacy Policy", "سياسة الخصوصية"),
		Terms:     more("Terms of Service", "شروط الخدمة"),
		More:      more("Learn more", "اعرف المزيد"),
		Whatsapp:  whatsapp,
		Github:    githubURL,
		Email:     emailMain,
		OdooApps:  odooAppsURL,
		PlayStore: playStoreURL,
		CSS:       template.CSS(siteCSS),
		OrgJSON:   orgJSON(),
	}
}

// langURLFor is the Go-side twin of the template langURL helper.
func langURLFor(l Lang, path string) string {
	if l == AR {
		if path == "/" {
			return "/ar"
		}
		return "/ar" + path
	}
	return path
}

// Handler renders the company website for any unmatched path. It is meant to be
// attached with engine.NoRoute so the site serves every route it owns while
// leaving /health/* and static assets to explicit handlers registered first.
func Handler() gin.HandlerFunc {
	reg := registry()
	return func(c *gin.Context) {
		lang, path := normalizePath(c.Request.URL.Path)

		if target, ok := redirects[path]; ok {
			c.Redirect(http.StatusMovedPermanently, langURLFor(lang, target))
			return
		}

		build, ok := reg[path]
		if !ok {
			renderNotFound(c, lang)
			return
		}

		p := build()
		vd := buildView(p, lang, path)
		var buf bytes.Buffer
		if err := siteTmpl.ExecuteTemplate(&buf, "site", vd); err != nil {
			c.Error(err) //nolint:errcheck
			c.String(http.StatusInternalServerError, "template error")
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.Header("Cache-Control", "public, max-age=300")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write(buf.Bytes())
	}
}

func renderNotFound(c *gin.Context, lang Lang) {
	p := page{
		Slug:    "/404",
		Title:   more("Page not found", "الصفحة غير موجودة"),
		Meta:    more("The page you're looking for doesn't exist.", "الصفحة التي تبحث عنها غير موجودة."),
		Eyebrow: more("404", "٤٠٤"),
		H1:      more("Page not found.", "الصفحة غير موجودة."),
		Lead:    more("The page you're looking for doesn't exist or has moved.", "الصفحة التي تبحث عنها غير موجودة أو تم نقلها."),
		HeroCTAs: []link{
			{Label: more("Back to Home", "العودة للرئيسية"), Href: "/", Kind: "primary"},
			{Label: more("Contact Us", "تواصل معنا"), Href: "/contact-us", Kind: "ghost"},
		},
		Robots: "noindex,follow",
	}
	vd := buildView(p, lang, "/404")
	var buf bytes.Buffer
	_ = siteTmpl.ExecuteTemplate(&buf, "site", vd)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusNotFound)
	_, _ = c.Writer.Write(buf.Bytes())
}

// Register wires the website routes onto a gin engine. The caller is expected to
// use this as the engine's NoRoute handler so any unclaimed path renders the
// site (matching how the Odoo origin served every URL).
func Register(engine *gin.Engine) {
	engine.GET("/favicon.svg", func(c *gin.Context) {
		c.Header("Content-Type", "image/svg+xml")
		c.Header("Cache-Control", "public, max-age=86400")
		c.String(http.StatusOK, faviconSVG)
	})
	engine.GET("/robots.txt", func(c *gin.Context) {
		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, robotsTxt())
	})
	engine.GET("/sitemap.xml", func(c *gin.Context) {
		c.Header("Content-Type", "application/xml; charset=utf-8")
		c.String(http.StatusOK, sitemapXML())
	})
	engine.NoRoute(Handler())
}

// robotsTxt points crawlers at the sitemap.
func robotsTxt() string {
	return "User-agent: *\nAllow: /\nSitemap: " + siteURL + "/sitemap.xml\n"
}

// sitemapXML lists every page in both languages with hreflang alternates.
func sitemapXML() string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">` + "\n")
	for _, slug := range registrySlugs() {
		p := registry()[slug]()
		if p.Robots != "" && strings.HasPrefix(p.Robots, "noindex") {
			continue
		}
		for _, l := range []Lang{EN, AR} {
			loc := siteURL + langURLFor(l, slug)
			b.WriteString("  <url>\n    <loc>" + loc + "</loc>\n")
			b.WriteString(`    <xhtml:link rel="alternate" hreflang="en" href="` + siteURL + langURLFor(EN, slug) + `"/>` + "\n")
			b.WriteString(`    <xhtml:link rel="alternate" hreflang="ar" href="` + siteURL + langURLFor(AR, slug) + `"/>` + "\n")
			b.WriteString(`    <xhtml:link rel="alternate" hreflang="x-default" href="` + siteURL + langURLFor(EN, slug) + `"/>` + "\n")
			b.WriteString("  </url>\n")
		}
	}
	b.WriteString("</urlset>\n")
	return b.String()
}

// registrySlugs returns the page slugs in a stable order for the sitemap.
func registrySlugs() []string {
	return []string{
		"/", "/about", "/solutions", "/solutions/erp", "/solutions/it-infrastructure",
		"/solutions/web", "/odoo-pos-egypt", "/erp-agriculture-egypt", "/ai-solutions",
		"/addons-store", "/pricing", "/portfolio", "/testimonials", "/insights",
		"/insights/ai-marketing-copilot", "/insights/ai-pos-copilot", "/knowledge",
		"/networktool", "/contact-us", "/request-demo", "/apk-download",
		"/privacy-policy", "/terms",
	}
}
