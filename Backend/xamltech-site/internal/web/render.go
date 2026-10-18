package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/xamltech/xamltech-site/internal/i18n"
)

// Meta is the per-page SEO metadata.
type Meta struct {
	Title       string
	Description string
	OGImage     string
	Type        string
	Index       bool
}

// alternate is one hreflang link.
type alternate struct {
	Lang string
	Href string
}

// navItem is a resolved navigation entry.
type navItem struct {
	Label    string
	Href     string
	External bool
	Children []navItem
}

// footerLink is a resolved footer link.
type footerLink struct {
	Label    string
	Href     string
	External bool
}

// footerCol is one footer column.
type footerCol struct {
	Title string
	Links []footerLink
}

// localizeHref appends ?lang=en to internal links when serving English.
func localizeHref(lang i18n.Lang, href string) string {
	if href == "" || strings.HasPrefix(href, "http") ||
		strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "#") {
		return href
	}
	if lang == i18n.EN {
		sep := "?"
		if strings.Contains(href, "?") {
			sep = "&"
		}
		return href + sep + "lang=en"
	}
	return href
}

func (s *Server) buildNav(lang i18n.Lang, raw any) []navItem {
	items, _ := raw.([]any)
	out := make([]navItem, 0, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m == nil {
			continue
		}
		href := str(m["href"])
		ni := navItem{
			Label:    str(m["label"]),
			Href:     localizeHref(lang, href),
			External: strings.HasPrefix(href, "http"),
		}
		if kids, ok := m["children"].([]any); ok {
			for _, k := range kids {
				km, _ := k.(map[string]any)
				if km == nil {
					continue
				}
				khref := str(km["href"])
				ni.Children = append(ni.Children, navItem{
					Label:    str(km["label"]),
					Href:     localizeHref(lang, khref),
					External: strings.HasPrefix(khref, "http"),
				})
			}
		}
		out = append(out, ni)
	}
	return out
}

func (s *Server) buildFooter(lang i18n.Lang, raw any) []footerCol {
	cols, _ := raw.([]any)
	out := make([]footerCol, 0, len(cols))
	for _, c := range cols {
		m, _ := c.(map[string]any)
		if m == nil {
			continue
		}
		col := footerCol{Title: str(m["title"])}
		links, _ := m["links"].([]any)
		for _, l := range links {
			lm, _ := l.(map[string]any)
			if lm == nil {
				continue
			}
			href := str(lm["href"])
			col.Links = append(col.Links, footerLink{
				Label:    str(lm["label"]),
				Href:     localizeHref(lang, href),
				External: strings.HasPrefix(href, "http"),
			})
		}
		out = append(out, col)
	}
	return out
}

// pageData is the view model handed to every template.
type pageData struct {
	Lang       string
	Dir        string
	Locale     string
	SelfLabel  string
	Other      string
	OtherLabel string
	Slug       string
	Path       string
	Meta       Meta
	Strings    map[string]any
	C          map[string]any
	Nav        []navItem
	Footer     []footerCol
	Contact    map[string]any
	ClarityID  string
	Year       int
	BaseURL    string
	Canonical  string
	Alternates []alternate
	JSONLD     template.JS
	Query      string
}

// metaFrom pulls the "meta" object out of a page's content map.
func metaFrom(c map[string]any) Meta {
	m := Meta{Type: "website", Index: true}
	raw, _ := c["meta"].(map[string]any)
	if raw == nil {
		return m
	}
	m.Title = str(raw["title"])
	m.Description = str(raw["description"])
	m.OGImage = str(raw["og_image"])
	if t := str(raw["type"]); t != "" {
		m.Type = t
	}
	if v, ok := raw["index"].(bool); ok {
		m.Index = v
	}
	return m
}

func contactMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func str(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func (s *Server) stringOr(m map[string]any, key, def string) string {
	if v := str(m[key]); v != "" {
		return v
	}
	return def
}

// jsonLD builds the JSON-LD @graph for a page.
func (s *Server) jsonLD(rt route, lang i18n.Lang, meta Meta) string {
	base := s.cfg.BaseURL
	graph := []any{
		map[string]any{
			"@type":     "Organization",
			"@id":       base + "/#org",
			"name":      "XamlTech",
			"url":       base,
			"logo":      base + "/static/img/logo.png",
			"email":     s.cfg.ContactTo,
			"telephone": "+201201410140",
			"address": map[string]any{
				"@type":           "PostalAddress",
				"addressLocality": "Aswan",
				"addressCountry":  "EG",
			},
			"sameAs": []string{
				"https://github.com/XamlTech",
				"https://apps.odoo.com/apps/modules/browse?search=xamltech",
			},
		},
		map[string]any{
			"@type":      "WebSite",
			"@id":        base + "/#website",
			"url":        base,
			"name":       "XamlTech",
			"publisher":  map[string]any{"@id": base + "/#org"},
			"inLanguage": []string{"ar", "en"},
		},
	}
	if rt.Path != "/" {
		graph = append(graph, map[string]any{
			"@type": "BreadcrumbList",
			"itemListElement": []any{
				map[string]any{"@type": "ListItem", "position": 1, "name": "XamlTech", "item": base + "/"},
				map[string]any{"@type": "ListItem", "position": 2, "name": meta.Title, "item": base + rt.Path},
			},
		})
	}
	doc := map[string]any{"@context": "https://schema.org", "@graph": graph}
	b, err := json.Marshal(doc)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// funcs returns the template helpers shared by every page.
func funcs() template.FuncMap {
	return template.FuncMap{
		"raw":  func(s string) template.HTML { return template.HTML(s) },
		"js":   func(s string) template.JS { return template.JS(s) },
		"json": func(v any) template.JS { b, _ := json.Marshal(v); return template.JS(b) },
		"seq": func(n int) []int {
			out := make([]int, n)
			for i := range out {
				out[i] = i
			}
			return out
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"mul": func(a, b int) int { return a * b },
		"div": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a / b
		},
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"join": func(sep string, v []any) string {
			parts := make([]string, 0, len(v))
			for _, x := range v {
				parts = append(parts, str(x))
			}
			return strings.Join(parts, sep)
		},
		"href": func(lang, p string) string {
			if lang == "en" {
				return p + "?lang=en"
			}
			return p
		},
		"dict": func(values ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(values); i += 2 {
				m[str(values[i])] = values[i+1]
			}
			return m
		},
		"check": func(v any) bool {
			switch x := v.(type) {
			case bool:
				return x
			case string:
				return x != ""
			case nil:
				return false
			case []any:
				return len(x) > 0
			default:
				return true
			}
		},
	}
}

// loggingWriter records the response status for access logs.
type loggingWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
