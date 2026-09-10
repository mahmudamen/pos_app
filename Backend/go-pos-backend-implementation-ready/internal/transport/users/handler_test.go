package users

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

func validToken(t *testing.T) string {
	t.Helper()
	raw, err := testTokens().Issue(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestListUsersRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/users", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestListUsersReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestListUsersAcceptsPaginationParams(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users?page=2&limit=10", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (no DB), got %d", recorder.Code)
	}
}

func TestCreateUserRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/users", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestCreateUserReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","display_name":"Cashier","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "manager"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader("not json"))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForMissingEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"display_name":"Cashier","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForMissingDisplayName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForMissingPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","display_name":"Cashier","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForShortPassword(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","display_name":"Cashier","password":"short","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForInvalidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","display_name":"Cashier","password":"password123","role":"admin"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForInvalidEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"not-an-email","display_name":"Cashier","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForEmptyEmail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":" ","display_name":"Cashier","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserReturnsBadRequestForEmptyDisplayName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","display_name":" ","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUserRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))

	// List users - should return 401 without auth (not 404)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/users", nil))
	if recorder.Code == http.StatusNotFound {
		t.Fatal("GET /v1/users route not registered")
	}

	// Create user - should return 401 without auth (not 404)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/users", nil))
	if recorder.Code == http.StatusNotFound {
		t.Fatal("POST /v1/users route not registered")
	}
}

func roleToken(t *testing.T, role string) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestCanCreateRoleHierarchy(t *testing.T) {
	cases := []struct {
		caller, target string
		want           bool
	}{
		{"owner", "owner", true},
		{"owner", "manager", true},
		{"owner", "cashier", true},
		{"manager", "cashier", true},
		{"manager", "manager", false},
		{"manager", "owner", false},
		{"cashier", "cashier", false},
		{"cashier", "manager", false},
		{"saas_admin", "owner", true},
		{"saas_admin", "cashier", true},
	}
	for _, c := range cases {
		if got := canCreateRole(c.caller, c.target); got != c.want {
			t.Errorf("canCreateRole(%q, %q) = %v, want %v", c.caller, c.target, got, c.want)
		}
	}
}

func TestCreateUserForbiddenForCashier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","display_name":"Cashier","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "cashier"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserForbiddenForManagerCreatingManager(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"user@example.com","display_name":"Manager","password":"password123","role":"manager"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "manager"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserAllowedManagerCreatesCashier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"cashier2@example.com","display_name":"Cashier 2","password":"password123","role":"cashier"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "manager"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (no DB), got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateUserAllowedOwnerCreatesOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"email":"owner2@example.com","display_name":"Owner 2","password":"password123","role":"owner"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/users", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+roleToken(t, "owner"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (no DB), got %d: %s", recorder.Code, recorder.Body.String())
	}
}
