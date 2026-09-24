package community

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/ratelimit"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

const validUUID = "22222222-2222-2222-2222-222222222222"

func nationalRouter(t *testing.T, limiter ratelimit.Limiter) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(nil, testTokens())
	if limiter != nil {
		h.joinLimiter = limiter
	}
	h.Register(router.Group("/v1"))
	return router
}

func doRequest(t *testing.T, router *gin.Engine, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestNationalRoutesRequireToken(t *testing.T) {
	router := nationalRouter(t, nil)
	for _, path := range []string{
		"/v1/community/me",
		"/v1/community/invitations",
		"/v1/community/members",
		"/v1/community/members/" + validUUID,
	} {
		recorder := doRequest(t, router, http.MethodGet, path, "", "")
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401, got %d", path, recorder.Code)
		}
	}
	recorder := doRequest(t, router, http.MethodPost, "/v1/community/join", `{"code":"EG-X"}`, "")
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("join: expected 401, got %d", recorder.Code)
	}
}

func TestNationalUnavailableWithoutDatabase(t *testing.T) {
	router := nationalRouter(t, nil)
	// valid join body reaches the pool check -> 503
	recorder := doRequest(t, router, http.MethodPost, "/v1/community/join",
		`{"code":"EG-AAAA-BBBB-CCCC"}`, ownerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("join: expected 503, got %d", recorder.Code)
	}
	// me with nil pool -> 503
	recorder = doRequest(t, router, http.MethodGet, "/v1/community/me", "", ownerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("me: expected 503, got %d", recorder.Code)
	}
	// member-only endpoints hit the membership check (which needs the pool), so
	// with a nil pool they surface as 503, not 403.
	for _, path := range []string{
		"/v1/community/members",
		"/v1/community/invitations",
	} {
		recorder = doRequest(t, router, http.MethodGet, path, "", managerToken(t))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s: expected 503, got %d", path, recorder.Code)
		}
	}
}

func TestJoinValidation(t *testing.T) {
	router := nationalRouter(t, nil)
	// malformed JSON -> 400
	recorder := doRequest(t, router, http.MethodPost, "/v1/community/join", `{not-json`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", recorder.Code)
	}
	// missing code -> 400
	recorder = doRequest(t, router, http.MethodPost, "/v1/community/join", `{}`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty code, got %d", recorder.Code)
	}
	// blank code -> 400
	recorder = doRequest(t, router, http.MethodPost, "/v1/community/join", `{"code":"   "}`, managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for blank code, got %d", recorder.Code)
	}
}

func TestJoinRateLimitedPerIP(t *testing.T) {
	router := nationalRouter(t, ratelimit.NewMemory(2, time.Minute))
	for i := 0; i < 2; i++ {
		recorder := doRequest(t, router, http.MethodPost, "/v1/community/join", `{"code":"EG-X"}`, ownerToken(t))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("attempt %d: expected 503 (pool nil reached), got %d", i+1, recorder.Code)
		}
	}
	recorder := doRequest(t, router, http.MethodPost, "/v1/community/join", `{"code":"EG-X"}`, ownerToken(t))
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 after limit, got %d", recorder.Code)
	}
}

func TestCreateInvitationValidation(t *testing.T) {
	router := nationalRouter(t, nil)
	// bad recipient email -> 400 before the pool/membership check
	recorder := doRequest(t, router, http.MethodPost, "/v1/community/invitations",
		`{"email":"not-an-email"}`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad email, got %d", recorder.Code)
	}
	// unparseable expires_at -> 400
	recorder = doRequest(t, router, http.MethodPost, "/v1/community/invitations",
		`{"expires_at":"tomorrow"}`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad expires_at, got %d", recorder.Code)
	}
	// valid body reaches the pool check -> 503
	recorder = doRequest(t, router, http.MethodPost, "/v1/community/invitations",
		`{"email":"peer@example.com","note":"hi","max_uses":3}`, ownerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (pool nil), got %d", recorder.Code)
	}
}

func TestInvitationRevokeAndMemberValidation(t *testing.T) {
	router := nationalRouter(t, nil)
	// bad invitation id -> 400
	recorder := doRequest(t, router, http.MethodDelete, "/v1/community/invitations/not-a-uuid", "", managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad invitation id, got %d", recorder.Code)
	}
	// valid invitation id -> reaches pool check -> 503
	recorder = doRequest(t, router, http.MethodDelete, "/v1/community/invitations/"+validUUID, "", managerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (pool nil), got %d", recorder.Code)
	}
	// member routes: invalid uuid -> 400
	for _, path := range []string{
		"/v1/community/members/not-a-uuid",
	} {
		recorder = doRequest(t, router, http.MethodGet, path, "", cashierToken(t))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", path, recorder.Code)
		}
	}
}

func TestUpdateMemberValidation(t *testing.T) {
	router := nationalRouter(t, nil)
	// nothing to update -> 400
	recorder := doRequest(t, router, http.MethodPatch, "/v1/community/members/"+validUUID, `{}`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", recorder.Code)
	}
	// bad status -> 400
	recorder = doRequest(t, router, http.MethodPatch, "/v1/community/members/"+validUUID,
		`{"status":"fired"}`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad status, got %d", recorder.Code)
	}
	// bad role -> 400
	recorder = doRequest(t, router, http.MethodPatch, "/v1/community/members/"+validUUID,
		`{"role":"emperor"}`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad role, got %d", recorder.Code)
	}
	// bad member uuid -> 400
	recorder = doRequest(t, router, http.MethodPatch, "/v1/community/members/not-a-uuid",
		`{"status":"active"}`, ownerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad member id, got %d", recorder.Code)
	}
	// valid body with valid uuid reaches pool check -> 503
	recorder = doRequest(t, router, http.MethodPatch, "/v1/community/members/"+validUUID,
		`{"status":"muted"}`, ownerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (pool nil), got %d", recorder.Code)
	}
}

func TestNationalRejectMalformedUUIDClaims(t *testing.T) {
	router := nationalRouter(t, nil)
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"not-a-tenant-id", "not-a-user-id",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", "owner")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/community/me", "/v1/community/members"} {
		recorder := doRequest(t, router, http.MethodGet, path, "", raw)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s with malformed uuid claims: expected 401, got %d", path, recorder.Code)
		}
	}
	recorder := doRequest(t, router, http.MethodPost, "/v1/community/join", `{"code":"EG-X"}`, raw)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("join with malformed uuid claims: expected 401, got %d", recorder.Code)
	}
}

func TestNationalPureHelpers(t *testing.T) {
	if got := normalizeCode("eg-1234-abcd"); got != "1234ABCD" {
		t.Fatalf("normalizeCode: got %q", got)
	}
	if got := normalizeCode("EG1234ABCD"); got != "1234ABCD" {
		t.Fatalf("normalizeCode uppercase: got %q", got)
	}
	if got := normalizeCode("EG"); got != "" {
		t.Fatalf("normalizeCode bare eg: got %q", got)
	}
	if levelForScore(199) != "bronze" || levelForScore(200) != "silver" ||
		levelForScore(500) != "gold" || levelForScore(1500) != "platinum" {
		t.Fatalf("levelForScore mapping broken")
	}
	c1, d1 := newInviteCode()
	c2, d2 := newInviteCode()
	if c1 == c2 || len(c1) != 12 || !strings.HasPrefix(d1, "EG-") || len(d1) != 15 {
		t.Fatalf("newInviteCode degenerate output: %q %q", c1, d1)
	}
	_ = d2
	if codeHash(c1) == codeHash("something-else") || len(codeHash(c1)) != 64 {
		t.Fatalf("codeHash broken")
	}
}
