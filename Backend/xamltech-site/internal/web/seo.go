package web

import (
	"embed"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

//go:embed files
var filesFS embed.FS

func (s *Server) serveFile(w http.ResponseWriter, name, contentType string, cache string) {
	data, err := filesFS.ReadFile("files/" + name)
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", contentType)
	if cache != "" {
		w.Header().Set("Cache-Control", cache)
	}
	_, _ = w.Write(data)
}

func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, "robots.txt", "text/plain; charset=utf-8", "public, max-age=86400")
}

func (s *Server) handleLlms(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, "llms.txt", "text/plain; charset=utf-8", "public, max-age=86400")
}

func (s *Server) handleAppsAds(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, "app-ads.txt", "text/plain; charset=utf-8", "public, max-age=86400")
}

func (s *Server) handleBing(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, "BingSiteAuth.xml", "application/xml; charset=utf-8", "public, max-age=86400")
}

func (s *Server) handleGoogleVerify(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, "googlea7b16c2b9c88eee4.html", "text/html; charset=utf-8", "public, max-age=86400")
}

func (s *Server) handleManifest(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, "manifest.json", "application/manifest+json; charset=utf-8", "public, max-age=86400")
}

func (s *Server) handleServiceWorker(w http.ResponseWriter, r *http.Request) {
	s.serveFile(w, "sw.js", "application/javascript; charset=utf-8", "no-cache")
}

// --- sitemap -----------------------------------------------------------------

type sitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
	Alternates []sitemapAlt
}

type sitemapAlt struct {
	XMLName  xml.Name `xml:"http://www.w3.org/1999/xhtml xhtml:link"`
	Rel      string   `xml:"rel,attr"`
	Hreflang string   `xml:"hreflang,attr"`
	Href     string   `xml:"href,attr"`
}

type sitemapIndex struct {
	XMLName xml.Name     `xml:"urlset"`
	Xmlns   string       `xml:"xmlns,attr"`
	XmlnsX  string       `xml:"xmlns:xhtml,attr"`
	URLs    []sitemapURL `xml:"url"`
}

func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	base := s.cfg.BaseURL
	now := time.Now().UTC().Format("2006-01-02")
	doc := sitemapIndex{Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9", XmlnsX: "http://www.w3.org/1999/xhtml"}
	for _, rt := range routes {
		if !rt.Sitemap {
			continue
		}
		arURL := base + rt.Path
		enURL := base + rt.Path + "?lang=en"
		prio := "0.8"
		freq := "monthly"
		if rt.Path == "/" {
			prio, freq = "1.0", "weekly"
		}
		alts := []sitemapAlt{
			{Rel: "alternate", Hreflang: "ar", Href: arURL},
			{Rel: "alternate", Hreflang: "en", Href: enURL},
			{Rel: "alternate", Hreflang: "x-default", Href: arURL},
		}
		doc.URLs = append(doc.URLs,
			sitemapURL{Loc: arURL, LastMod: now, ChangeFreq: freq, Priority: prio, Alternates: alts},
			sitemapURL{Loc: enURL, LastMod: now, ChangeFreq: freq, Priority: prio, Alternates: alts},
		)
	}
	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		http.Error(w, "sitemap error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(xml.Header))
	_, _ = w.Write(out)
}

var _ = fmt.Sprint
var _ = strings.TrimSpace
