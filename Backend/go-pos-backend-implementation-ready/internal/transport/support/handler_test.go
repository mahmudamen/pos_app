package support

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
)

func adminRouter() (*gin.Engine, string) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testutil.TokenManager(), "support", "", nil).Register(router.Group("/v1/saas"))
	return router, ""
}

func do(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(rec, req)
	return rec
}

func TestSupportRoutes_RequireSaasAdmin(t *testing.T) {
	router, _ := adminRouter()
	for _, path := range []string{"/v1/saas/support/messages", "/v1/saas/support/summary"} {
		if rec := do(router, http.MethodGet, path, "", ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s without token: got %d, want 401", path, rec.Code)
		}
	}
	for _, path := range []string{
		"/v1/saas/support/messages/x/read",
		"/v1/saas/support/messages/x/archive",
		"/v1/saas/support/messages/x/unarchive",
	} {
		if rec := do(router, http.MethodPost, path, "", ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s without token: got %d, want 401", path, rec.Code)
		}
	}
}

func TestSupportRoutes_UnavailableWithoutPool(t *testing.T) {
	router, _ := adminRouter()
	token := testutil.MintAccess(t, "t", "u", "d", "s", "saas_admin")
	for _, path := range []string{"/v1/saas/support/messages", "/v1/saas/support/summary"} {
		if rec := do(router, http.MethodGet, path, token, ""); rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s without DB: got %d, want 503", path, rec.Code)
		}
	}
	for _, path := range []string{
		"/v1/saas/support/messages/x/archive",
		"/v1/saas/support/messages/x/unarchive",
	} {
		if rec := do(router, http.MethodPost, path, token, ""); rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s without DB: got %d, want 503", path, rec.Code)
		}
	}
}

func TestReplyMessage_Validation(t *testing.T) {
	router, _ := adminRouter()
	token := testutil.MintAccess(t, "t", "u", "d", "s", "saas_admin")
	// A reply with an empty body is a 400 even before the DB is consulted.
	rec := do(router, http.MethodPost, "/v1/saas/support/messages/abc/reply", token, `{"body_text":"  "}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty reply body: got %d, want 400", rec.Code)
	}
}

func TestSendMessage_Validation(t *testing.T) {
	router, _ := adminRouter()
	token := testutil.MintAccess(t, "t", "u", "d", "s", "saas_admin")
	rec := do(router, http.MethodPost, "/v1/saas/support/messages", token, `{"to":"not-an-email"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad send payload: got %d, want 400", rec.Code)
	}
}

func TestReplySubject(t *testing.T) {
	cases := map[string]string{
		"Invoice question": "Re: Invoice question",
		"Re: Invoice":      "Re: Invoice",
		"RE: nope":         "RE: nope",
		"":                 "Re:",
		"  Spaced  ":       "Re: Spaced",
	}
	for in, want := range cases {
		if got := replySubject(in); got != want {
			t.Errorf("replySubject(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPaginationClamps(t *testing.T) {
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/?page=0&limit=999", nil)
	page, limit := pagination(ctx)
	if page != 1 || limit != 25 {
		t.Fatalf("clamped = (%d,%d), want (1,25)", page, limit)
	}
	ctx2, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx2.Request = httptest.NewRequest(http.MethodGet, "/?page=3&limit=50", nil)
	if page, limit := pagination(ctx2); page != 3 || limit != 50 {
		t.Fatalf("explicit = (%d,%d), want (3,50)", page, limit)
	}
}
