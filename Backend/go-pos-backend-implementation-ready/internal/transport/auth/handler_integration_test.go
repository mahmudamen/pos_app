package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// It is intentional that this file carries a DB-backed integration test (AUTH-011):
// the login → refresh-rotation → reuse-revocation chain cannot be exercised with a
// nil pool. Tests skip when no TEST_DATABASE_URL / DATABASE_URL is configured,
// keeping `go test ./...` green in CI without a database.

func authConfig() config.Config {
	return config.Config{
		JWTIssuer: "pos-api", JWTAccessSecret: "access-secret-that-is-at-least-32-bytes",
		JWTRefreshSecret: "refresh-secret-that-is-at-least-32-bytes",
		JWTAccessTTL:     900000000000, JWTRefreshTTL: 3600000000000, BcryptCost: 10,
		MaxSessionsPerUser: 3,
	}
}

func setupAuthIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, authConfig()).Register(router.Group("/v1"))
	return router, seed, pool
}

func loginBody(seed testutil.Seed) string {
	payload := map[string]string{
		"email": seed.ManagerEmail, "password": seed.Password,
		"tenant_id": seed.TenantID, "device_id": seed.DeviceCode, "device_name": "Counter 1",
	}
	b, _ := json.Marshal(payload)
	return string(b)
}

func doPost(t *testing.T, router http.Handler, path, body, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	router.ServeHTTP(rec, req)
	return rec
}

func TestAuthIntegration_LoginHappyPath(t *testing.T) {
	router, seed, _ := setupAuthIntegration(t)
	rec := doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			User         struct {
				ID    string `json:"id"`
				Role  string `json:"role"`
				Email string `json:"-"`
			} `json:"user"`
			Tenant struct {
				ID string `json:"id"`
			} `json:"tenant"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse login response: %v", err)
	}
	if resp.Data.AccessToken == "" || resp.Data.RefreshToken == "" {
		t.Fatal("login: expected access + refresh tokens")
	}
	if resp.Data.User.ID != seed.ManagerID {
		t.Fatalf("login: expected user %s, got %s", seed.ManagerID, resp.Data.User.ID)
	}
	if resp.Data.User.Role != "manager" {
		t.Fatalf("login: expected manager role, got %q", resp.Data.User.Role)
	}
	if resp.Data.Tenant.ID != seed.TenantID {
		t.Fatalf("login: expected tenant %s, got %s", seed.TenantID, resp.Data.Tenant.ID)
	}
	if seed.ManagerEmail == "" {
		t.Fatal("testutil.Seed should expose the manager email for integration logins")
	}
}

func TestAuthIntegration_LoginRejectsWrongPassword(t *testing.T) {
	router, seed, _ := setupAuthIntegration(t)
	payload := map[string]string{
		"email": seed.ManagerEmail, "password": "wrong-password",
		"tenant_id": seed.TenantID, "device_id": seed.DeviceCode, "device_name": "Counter 1",
	}
	b, _ := json.Marshal(payload)
	rec := doPost(t, router, "/v1/auth/login", string(b), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login wrong password: expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthIntegration_RefreshRotationAndReuseRevocation(t *testing.T) {
	router, seed, _ := setupAuthIntegration(t)
	loginRec := doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", loginRec.Code)
	}
	var login struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &login); err != nil {
		t.Fatalf("parse login: %v", err)
	}
	firstRefresh := login.Data.RefreshToken
	if firstRefresh == "" {
		t.Fatal("login: missing refresh token")
	}

	// Rotate: presenting the refresh token yields a new access + refresh pair.
	rot1 := doPost(t, router, "/v1/auth/refresh", `{"refresh_token":"`+firstRefresh+`"}`, "")
	if rot1.Code != http.StatusOK {
		t.Fatalf("refresh: expected 200, got %d: %s", rot1.Code, rot1.Body.String())
	}
	var rotBody struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rot1.Body.Bytes(), &rotBody); err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	if rotBody.Data.RefreshToken == "" || rotBody.Data.RefreshToken == firstRefresh {
		t.Fatal("refresh rotation: expected a NEW refresh token")
	}

	// Reusing the already-rotated token must fail AND revoke the session family.
	reuse := doPost(t, router, "/v1/auth/refresh", `{"refresh_token":"`+firstRefresh+`"}`, "")
	if reuse.Code != http.StatusUnauthorized {
		t.Fatalf("refresh reuse: expected 401, got %d: %s", reuse.Code, reuse.Body.String())
	}

	// The newly issued token is also dead after the family revocation.
	postReuse := doPost(t, router, "/v1/auth/refresh", `{"refresh_token":"`+rotBody.Data.RefreshToken+`"}`, "")
	if postReuse.Code != http.StatusUnauthorized {
		t.Fatalf("refresh after family revocation: expected 401, got %d: %s", postReuse.Code, postReuse.Body.String())
	}
}

func TestAuthIntegration_LogoutRevokesSession(t *testing.T) {
	router, seed, _ := setupAuthIntegration(t)
	rec := doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d", rec.Code)
	}
	var login struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &login); err != nil {
		t.Fatalf("parse login: %v", err)
	}

	logoutRec := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+login.Data.AccessToken)
	router.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusNoContent {
		t.Fatalf("logout: expected 204, got %d: %s", logoutRec.Code, logoutRec.Body.String())
	}

	// A logout token must no longer authenticate.
	badRec := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
	badReq.Header.Set("Authorization", "Bearer "+login.Data.AccessToken)
	router.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusUnauthorized {
		t.Fatalf("logout after successful logout: expected 401, got %d: %s", badRec.Code, badRec.Body.String())
	}
}

// TestAuthIntegration_LoginBlockedForDisabledTenant proves that a tenant
// stopped via the platform (status 'disabled') can no longer sign in: the
// login gate must reject it with 403 organization_stopped (distinct from the
// suspended 403 so operators can tell pause apart from stop).
func TestAuthIntegration_LoginBlockedForDisabledTenant(t *testing.T) {
	router, seed, pool := setupAuthIntegration(t)
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET status = 'disabled' WHERE id = $1::uuid`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	rec := doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("login for disabled tenant: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("organization_stopped")) {
		t.Fatalf("disabled login should carry organization_stopped: %s", rec.Body.String())
	}
	// Reverting to 'active' re-opens login.
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET status = 'active' WHERE id = $1::uuid`, seed.TenantID); err != nil {
		t.Fatal(err)
	}
	rec = doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login after reactivate: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
