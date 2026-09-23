package community

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func testTokens() security.TokenManager {
	return security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func tokenWithRole(t *testing.T, role string) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004",
		role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func ownerToken(t *testing.T) string   { return tokenWithRole(t, "owner") }
func managerToken(t *testing.T) string { return tokenWithRole(t, "manager") }
func cashierToken(t *testing.T) string { return tokenWithRole(t, "cashier") }
func guestToken(t *testing.T) string   { return tokenWithRole(t, "guest") }

func setup() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router, router.Group("/v1")
}

func TestRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	registered := map[string]bool{}
	for _, r := range router.Routes() {
		registered[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /v1/community/profiles",
		"GET /v1/community/profiles/me",
		"PUT /v1/community/profiles/me",
		"GET /v1/community/profiles/:id",
		"GET /v1/community/companies",
		"POST /v1/community/companies",
		"GET /v1/community/companies/:id",
		"PATCH /v1/community/companies/:id",
		"DELETE /v1/community/companies/:id",
		"POST /v1/community/companies/:id/members",
		"DELETE /v1/community/companies/:id/members/:userId",
	} {
		if !registered[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestProfilesRequireToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, path := range []string{
		"/v1/community/profiles", "/v1/community/profiles/me",
		"/v1/community/profiles/11111111-1111-1111-1111-111111111111",
		"/v1/community/companies", "/v1/community/companies/22222222-2222-2222-2222-222222222222",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s: expected 401, got %d", path, recorder.Code)
		}
	}
}

func TestProfilesReadBlockedForGuest(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/community/profiles", nil)
	request.Header.Set("Authorization", "Bearer "+guestToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestProfilesUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, token := range []string{ownerToken(t), cashierToken(t)} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/v1/community/profiles", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected 503, got %d", recorder.Code)
		}
	}
}

func TestUpsertProfileValidation(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)

	longHeadline := strings.Repeat("a", 141)
	body := `{"headline":"` + longHeadline + `"}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/v1/community/profiles/me", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+cashierToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for long headline, got %d", recorder.Code)
	}

	body = `{"years_experience":-1}`
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, "/v1/community/profiles/me", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+cashierToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative years, got %d", recorder.Code)
	}

	body = `{"skills":[" Grill","grill  ","sous"]}`
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPut, "/v1/community/profiles/me", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+cashierToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (pool nil reached after validation), got %d", recorder.Code)
	}
}

func TestCreateCompanyValidation(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/community/companies", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer "+ownerToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty name, got %d", recorder.Code)
	}

	// cashier cannot create companies
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/v1/community/companies", strings.NewReader(`{"name":"Bakery X"}`))
	request.Header.Set("Authorization", "Bearer "+cashierToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for cashier, got %d", recorder.Code)
	}

	// valid request reaches the pool check (503 without DB)
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/v1/community/companies", strings.NewReader(`{"name":"Bakery X"}`))
	request.Header.Set("Authorization", "Bearer "+managerToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (pool nil), got %d", recorder.Code)
	}
}

func TestAddMemberValidation(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/community/companies/22222222-2222-2222-2222-222222222222/members",
		strings.NewReader(`{"user_id":"not-a-uuid"}`))
	request.Header.Set("Authorization", "Bearer "+ownerToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad user id, got %d", recorder.Code)
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/v1/community/companies/22222222-2222-2222-2222-222222222222/members",
		strings.NewReader(`{"user_id":"11111111-1111-1111-1111-111111111111","role":"executive"}`))
	request.Header.Set("Authorization", "Bearer "+ownerToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad role, got %d", recorder.Code)
	}
}

func TestNormalizeSkills(t *testing.T) {
	got := normalizeSkills([]string{" Grill", "grill  ", "sous", "Sous", "pastry", ""})
	want := []string{"grill", "pastry", "sous"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("skills[%d] = %q, want %q (full %v)", i, got[i], want[i], got)
		}
	}
}
