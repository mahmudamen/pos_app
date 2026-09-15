package registers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/example/pos-api/internal/transport/settings"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CLOSE-MANAGER DB-backed slice: pos.manager.close gates session close behind a
// manager PIN (owners/saas_admin bypass). Skips without a database.

func setupRegistersIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed, pool
}

func regToken(t *testing.T, seed testutil.Seed, userID, role string) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, userID, seed.DeviceID, "sess-reg", role)
}

func openSession(t *testing.T, router http.Handler, token string) string {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/registers/open", bytes.NewBufferString(`{"opening_cash_minor":100}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("open session: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse open: %v", err)
	}
	return out.Data.ID
}

func closeSession(t *testing.T, router http.Handler, token, sessionID, extra string) (int, string) {
	t.Helper()
	body := `{"closing_cash_minor":500`
	if extra != "" {
		body += `,` + extra
	}
	body += `}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/registers/"+sessionID+"/close", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestCloseManagerIntegration_CashierNeedsPinWhenEnabled(t *testing.T) {
	router, seed, pool := setupRegistersIntegration(t)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyManagerClose, "true")
	testutil.SetManagerPIN(t, pool, seed.TenantID, seed.CashierID, "1234")
	cashierToken := regToken(t, seed, seed.CashierID, "cashier")
	sessionID := openSession(t, router, cashierToken)

	code, body := closeSession(t, router, cashierToken, sessionID, "")
	if code != http.StatusForbidden {
		t.Fatalf("close without pin: expected 403, got %d: %s", code, body)
	}
	code, body = closeSession(t, router, cashierToken, sessionID, `"manager_pin":"9999"`)
	if code != http.StatusForbidden {
		t.Fatalf("close with wrong pin: expected 403, got %d: %s", code, body)
	}
	code, body = closeSession(t, router, cashierToken, sessionID, `"manager_pin":"1234"`)
	if code != http.StatusOK {
		t.Fatalf("close with valid pin: expected 200, got %d: %s", code, body)
	}
}

func TestCloseManagerIntegration_OwnerBypassesPin(t *testing.T) {
	router, seed, pool := setupRegistersIntegration(t)
	testutil.SetTenantSetting(t, pool, seed.TenantID, settings.KeyManagerClose, "true")
	cashierToken := regToken(t, seed, seed.CashierID, "cashier")
	sessionID := openSession(t, router, cashierToken)

	ownerToken := regToken(t, seed, seed.ManagerID, "owner")
	code, body := closeSession(t, router, ownerToken, sessionID, "")
	if code != http.StatusOK {
		t.Fatalf("owner close without pin: expected 200, got %d: %s", code, body)
	}
}

func TestCloseManagerIntegration_DisabledSettingNoPinNeeded(t *testing.T) {
	router, seed, _ := setupRegistersIntegration(t)
	cashierToken := regToken(t, seed, seed.CashierID, "cashier")
	sessionID := openSession(t, router, cashierToken)

	code, body := closeSession(t, router, cashierToken, sessionID, "")
	if code != http.StatusOK {
		t.Fatalf("close with setting off: expected 200, got %d: %s", code, body)
	}
}
