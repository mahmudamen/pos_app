package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/testutil"
	httptransport "github.com/example/pos-api/internal/transport/http"
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

// setupSurfaceAuthIntegration is setupAuthIntegration with the three public
// domains enabled, i.e. how production is configured.
func setupSurfaceAuthIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	cfg := authConfig()
	cfg.Surfaces = config.SurfaceRouting{
		Enabled:       true,
		CompanyDomain: config.DefaultCompanyDomain,
		SaaSDomain:    config.DefaultSaaSDomain,
		POSDomain:     config.DefaultPOSDomain,
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(httptransport.SurfaceRouting(cfg.Surfaces))
	NewHandler(pool, cfg).Register(router.Group("/v1"))
	return router, seed, pool
}

func doPostOnHost(t *testing.T, router http.Handler, host, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

// seedPlatformAdmin adds a saas_admin to the seeded tenant with the same
// password, so the console-host sign-in can be exercised end to end.
func seedPlatformAdmin(t *testing.T, pool *pgxpool.Pool, seed testutil.Seed) string {
	t.Helper()
	ctx := context.Background()
	hash, err := security.HashPassword(seed.Password, 10)
	if err != nil {
		t.Fatalf("hash platform password: %v", err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", seed.TenantID); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}
	email := "saas-" + seed.Slug + "@example.com"
	if _, err = tx.Exec(ctx,
		`INSERT INTO users (tenant_id, email, password_hash, display_name, role)
		 VALUES ($1::uuid, $2, $3, 'Platform Operator', 'saas_admin')`, seed.TenantID, email, hash); err != nil {
		t.Fatalf("seed platform admin: %v", err)
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return email
}

func loginBodyFor(email string, seed testutil.Seed) string {
	b, _ := json.Marshal(map[string]string{
		"email": email, "password": seed.Password,
		"tenant_id": seed.TenantID, "device_id": seed.DeviceCode, "device_name": "Counter 1",
	})
	return string(b)
}

// TestAuthIntegration_SignInIsScopedToItsHost proves the two sign-in audiences
// are disjoint: a store manager cannot sign in on the console domain, and a
// platform operator cannot sign in on the POS domain.
func TestAuthIntegration_SignInIsScopedToItsHost(t *testing.T) {
	router, seed, pool := setupSurfaceAuthIntegration(t)
	platformEmail := seedPlatformAdmin(t, pool, seed)

	cases := []struct {
		name       string
		host       string
		email      string
		wantStatus int
		wantCode   string
	}{
		{"manager on POS host", config.DefaultPOSDomain, seed.ManagerEmail, http.StatusOK, ""},
		{"manager on console host", config.DefaultSaaSDomain, seed.ManagerEmail, http.StatusForbidden, "wrong_surface"},
		{"platform admin on console host", config.DefaultSaaSDomain, platformEmail, http.StatusOK, ""},
		{"platform admin on POS host", config.DefaultPOSDomain, platformEmail, http.StatusForbidden, "wrong_surface"},
		{"manager on company host", config.DefaultCompanyDomain, seed.ManagerEmail, http.StatusNotFound, "not_available_on_host"},
	}
	for _, c := range cases {
		rec := doPostOnHost(t, router, c.host, "/v1/auth/login", loginBodyFor(c.email, seed))
		if rec.Code != c.wantStatus {
			t.Errorf("%s: status = %d, want %d (%s)", c.name, rec.Code, c.wantStatus, rec.Body.String())
		}
		if c.wantCode != "" && !bytes.Contains(rec.Body.Bytes(), []byte(c.wantCode)) {
			t.Errorf("%s: body = %s, want code %s", c.name, rec.Body.String(), c.wantCode)
		}
	}
}

// TestAuthIntegration_RefreshKeepsRoleAndStaysOnItsHost covers the refresh
// path: the rotated access token must keep the role (every permission check
// reads that claim) and a session may not be renewed on the other domain.
func TestAuthIntegration_RefreshKeepsRoleAndStaysOnItsHost(t *testing.T) {
	router, seed, pool := setupSurfaceAuthIntegration(t)
	platformEmail := seedPlatformAdmin(t, pool, seed)

	loginRec := doPostOnHost(t, router, config.DefaultPOSDomain, "/v1/auth/login", loginBodyFor(seed.ManagerEmail, seed))
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", loginRec.Code, loginRec.Body.String())
	}
	var login struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginRec.Body.Bytes(), &login); err != nil {
		t.Fatalf("parse login: %v", err)
	}

	// A manager session cannot be renewed on the console domain...
	crossed := doPostOnHost(t, router, config.DefaultSaaSDomain, "/v1/auth/refresh",
		`{"refresh_token":"`+login.Data.RefreshToken+`"}`)
	if crossed.Code != http.StatusForbidden || !bytes.Contains(crossed.Body.Bytes(), []byte("wrong_surface")) {
		t.Fatalf("cross-surface refresh = %d %s, want 403 wrong_surface", crossed.Code, crossed.Body.String())
	}

	// ...but on its own domain it rotates and keeps the role claim.
	rot := doPostOnHost(t, router, config.DefaultPOSDomain, "/v1/auth/refresh",
		`{"refresh_token":"`+login.Data.RefreshToken+`"}`)
	if rot.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", rot.Code, rot.Body.String())
	}
	var rotBody struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rot.Body.Bytes(), &rotBody); err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	manager := security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     900000000000, RefreshTTL: 3600000000000,
	}
	claims, err := manager.Parse(rotBody.Data.AccessToken, security.AccessToken)
	if err != nil {
		t.Fatalf("parse refreshed access token: %v", err)
	}
	if claims.Role != "manager" {
		t.Errorf("refreshed access token role = %q, want manager", claims.Role)
	}

	// A platform session cannot be renewed on the POS domain either.
	platformRec := doPostOnHost(t, router, config.DefaultSaaSDomain, "/v1/auth/login", loginBodyFor(platformEmail, seed))
	if platformRec.Code != http.StatusOK {
		t.Fatalf("platform login: %d %s", platformRec.Code, platformRec.Body.String())
	}
	var platformLogin struct {
		Data struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(platformRec.Body.Bytes(), &platformLogin); err != nil {
		t.Fatalf("parse platform login: %v", err)
	}
	crossedPlatform := doPostOnHost(t, router, config.DefaultPOSDomain, "/v1/auth/refresh",
		`{"refresh_token":"`+platformLogin.Data.RefreshToken+`"}`)
	if crossedPlatform.Code != http.StatusForbidden || !bytes.Contains(crossedPlatform.Body.Bytes(), []byte("wrong_surface")) {
		t.Errorf("platform cross-surface refresh = %d %s, want 403 wrong_surface", crossedPlatform.Code, crossedPlatform.Body.String())
	}
	ownPlatform := doPostOnHost(t, router, config.DefaultSaaSDomain, "/v1/auth/refresh",
		`{"refresh_token":"`+platformLogin.Data.RefreshToken+`"}`)
	if ownPlatform.Code != http.StatusOK {
		t.Fatalf("platform refresh on console host: %d %s", ownPlatform.Code, ownPlatform.Body.String())
	}
	var platformRot struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(ownPlatform.Body.Bytes(), &platformRot); err != nil {
		t.Fatalf("parse platform refresh: %v", err)
	}
	platformClaims, err := manager.Parse(platformRot.Data.AccessToken, security.AccessToken)
	if err != nil {
		t.Fatalf("parse platform access token: %v", err)
	}
	if platformClaims.Role != "saas_admin" {
		t.Errorf("refreshed platform access token role = %q, want saas_admin", platformClaims.Role)
	}
}
