package server_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/transport/server"
	"github.com/gin-gonic/gin"
)

// newAuthPageEngine assembles the full route table with no database — the
// recovery pages carry no data, so this is also how they run in production.
func newAuthPageEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	server.Register(engine, server.Deps{Config: config.Config{}})
	return engine
}

func get(t *testing.T, engine *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestForgotPasswordPageServesPublicForm(t *testing.T) {
	rec := get(t, newAuthPageEngine(t), "/forgot-password")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("content type = %q, want text/html", ct)
	}
	if got := rec.Header().Get("X-Robots-Tag"); !strings.Contains(got, "noindex") {
		t.Fatalf("X-Robots-Tag = %q, want noindex", got)
	}
	for _, want := range []string{
		"<form", `id="tenant"`, `id="email"`, "/v1/auth/forgot-password",
		`content="noindex, nofollow"`,
	} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("page missing %q", want)
		}
	}
}

func TestResetPasswordPageCarriesTheTokenFromTheQuery(t *testing.T) {
	engine := newAuthPageEngine(t)
	rec := get(t, engine, "/reset-password?token=0123456789abcdef")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"/v1/auth/reset-password", "0123456789abcdef"} {
		if !strings.Contains(body, want) {
			t.Errorf("page missing %q", want)
		}
	}
	// Without a token the form is rendered but the script refuses to submit.
	empty := get(t, engine, "/reset-password")
	if empty.Code != http.StatusOK {
		t.Fatalf("expected 200 for a tokenless link, got %d", empty.Code)
	}
	if strings.Contains(empty.Body.String(), "0123456789abcdef") {
		t.Error("tokenless page leaked a token")
	}
}

// The recovery pages are reachable from an Arabic-first client, so both
// languages must render — and the Arabic one must be RTL.
func TestAuthPagesRenderArabicRTL(t *testing.T) {
	for path, arabicCopy := range map[string]string{
		"/forgot-password": "إعادة تعيين كلمة المرور",
		"/reset-password":  "اختر كلمة مرور جديدة",
	} {
		rec := get(t, newAuthPageEngine(t), path+"?lang=ar")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, rec.Code)
		}
		body := rec.Body.String()
		if !strings.Contains(body, `dir="rtl"`) {
			t.Errorf("%s: expected dir=rtl", path)
		}
		if !strings.Contains(body, arabicCopy) {
			t.Errorf("%s: expected %q, got %s", path, arabicCopy, excerpt(body))
		}
	}
}

// A missing key in one language renders the literal "<no value>" in production
// — only visible to a reader of that language.
func TestAuthPageStringsCoverBothLanguages(t *testing.T) {
	for _, path := range []string{"/forgot-password", "/reset-password"} {
		for _, lang := range []string{"ar", "en"} {
			rec := get(t, newAuthPageEngine(t), path+"?lang="+lang)
			if strings.Contains(rec.Body.String(), "<no value>") {
				t.Errorf("%s lang=%s rendered a missing string key", path, lang)
			}
		}
	}
}

// The email link points at whichever host served the POST, so both public
// hosts must actually publish the page the link lands on. internal/transport/
// http is where that host gate lives; this asserts the routes exist at all.
func TestRecoveryPagesRegisteredAlongsideSitePages(t *testing.T) {
	engine := newAuthPageEngine(t)
	var paths []string
	for _, r := range engine.Routes() {
		if r.Method == http.MethodGet {
			paths = append(paths, r.Path)
		}
	}
	for _, want := range []string{"/forgot-password", "/reset-password", "/private", "/pricing"} {
		if !slices.Contains(paths, want) {
			t.Errorf("route %s not registered", want)
		}
	}
}

func excerpt(body string) string {
	if len(body) > 240 {
		return body[:240]
	}
	return body
}
