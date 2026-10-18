package web

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xamltech/xamltech-site/internal/config"
	"github.com/xamltech/xamltech-site/internal/content"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	store, err := content.Load()
	if err != nil {
		t.Fatalf("content.Load: %v", err)
	}
	cfg := config.Config{
		BaseURL:         "https://xamltech.com",
		Env:             "development",
		DefLang:         "ar",
		SubmissionsFile: filepath.Join(t.TempDir(), "subs.jsonl"),
	}
	srv, err := New(cfg, store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv
}

// TestRoutesRender asserts every public route renders HTML 200 in both
// languages, with the requested language reflected on <html>.
func TestRoutesRender(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	for _, r := range routes {
		for _, tc := range []struct {
			url    string
			locale string
			dir    string
		}{
			{url: r.Path, locale: "ar", dir: "rtl"},
			{url: r.Path + "?lang=en", locale: "en", dir: "ltr"},
		} {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Errorf("%s -> %d", tc.url, rec.Code)
				continue
			}
			body := rec.Body.String()
			if !strings.Contains(body, "<!doctype html>") {
				t.Errorf("%s: not html", tc.url)
			}
			want := `<html lang="` + tc.locale + `" dir="` + tc.dir + `"`
			if !strings.Contains(body, want) {
				t.Errorf("%s: missing %q", tc.url, want)
			}
		}
	}
}

// TestLangSwitchIsInPlace ensures ?lang=en serves English without a redirect,
// so the content matches the canonical/hreflang URL.
func TestLangSwitchIsInPlace(t *testing.T) {
	srv := newTestServer(t)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/solutions/erp?lang=en", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("unexpected redirect to %q", loc)
	}
	if !strings.Contains(rec.Body.String(), `<html lang="en" dir="ltr"`) {
		t.Fatalf("expected English render")
	}
}

func TestNotFound(t *testing.T) {
	srv := newTestServer(t)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/does-not-exist", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestAliasesRedirect(t *testing.T) {
	srv := newTestServer(t)
	for from, to := range aliases {
		req := httptest.NewRequest(http.MethodGet, from, nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusMovedPermanently {
			t.Errorf("%s -> %d, want 301", from, rec.Code)
			continue
		}
		if got := rec.Header().Get("Location"); !strings.HasPrefix(got, to) {
			t.Errorf("%s -> %q, want prefix %q", from, got, to)
		}
	}
}

func TestSEOEndpoints(t *testing.T) {
	srv := newTestServer(t)
	for _, p := range []string{
		"/health", "/robots.txt", "/sitemap.xml", "/llms.txt",
		"/app-ads.txt", "/BingSiteAuth.xml", "/manifest.webmanifest", "/sw.js",
	} {
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s -> %d", p, rec.Code)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("%s: empty body", p)
		}
	}
}

func TestFormSubmission(t *testing.T) {
	srv := newTestServer(t)
	form := strings.NewReader("name=Ada&email=ada@example.com&message=Hello&lang=en")
	req := httptest.NewRequest(http.MethodPost, "/api/contact", form)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status %d, want 303", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/contact-us?sent=1" {
		t.Fatalf("Location %q", loc)
	}
	b, err := os.ReadFile(srv.cfg.SubmissionsFile)
	if err != nil {
		t.Fatalf("read submissions: %v", err)
	}
	if !strings.Contains(string(b), "ada@example.com") {
		t.Fatalf("submission not persisted: %s", b)
	}
}

func TestFormHoneypotAndValidation(t *testing.T) {
	srv := newTestServer(t)
	cases := []struct {
		name string
		body string
		want string
	}{
		{"honeypot", "website=bot&name=x&email=x@x.com&message=hi", "/contact-us?sent=1"},
		{"missing", "name=&email=&message=", "/contact-us?sent=error"},
		{"bad-email", "name=a&email=nope&message=hi", "/contact-us?sent=error"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodPost, "/api/contact", strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if loc := rec.Header().Get("Location"); loc != c.want {
			t.Errorf("%s: Location %q, want %q", c.name, loc, c.want)
		}
	}
}
