package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// mintAuthAccess issues a token with the AUTH handler's own TokenManager
// (issuer "pos-api") so Parse accepts it; testutil.MintAccess uses a different
// test issuer and must not be used against the auth router.
func mintAuthAccess(t *testing.T, h *Handler, seed testutil.Seed, userID, role string) string {
	t.Helper()
	token, err := h.Tokens().IssueWithRole(time.Now(), security.AccessToken,
		seed.TenantID, userID, seed.DeviceID, "sess-lock", role)
	if err != nil {
		t.Fatalf("mint auth access token: %v", err)
	}
	return token
}

// PIN-LOCKOUT DB-backed slice: five failed verify-pin attempts lock the
// credential for 15 minutes (423 pin_locked), a valid pin resets the counter,
// and the unlock route clears the lock. Skips without a database.

type verifyPinResult struct {
	Data struct {
		Valid        bool `json:"valid"`
		HasPin       bool `json:"has_pin"`
		AttemptsLeft int  `json:"attempts_left"`
	} `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func verifyPin(t *testing.T, router http.Handler, token, pin string) (int, verifyPinResult) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/verify-pin", bytes.NewBufferString(`{"pin":"`+pin+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	var out verifyPinResult
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	return rec.Code, out
}

func unlockPin(t *testing.T, router http.Handler, token, userID string) int {
	t.Helper()
	body := `{"user_id":"` + userID + `"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/unlock-pin", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec.Code
}

func setupAuthHandler(t *testing.T) (*gin.Engine, *Handler, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(pool, authConfig())
	h.Register(router.Group("/v1"))
	return router, h, seed, pool
}

func TestAuthIntegration_PinLockoutAfterFiveFailures(t *testing.T) {
	router, h, seed, pool := setupAuthHandler(t)
	token := mintAuthAccess(t, h, seed, seed.ManagerID, "manager")
	testutil.SetManagerPIN(t, pool, seed.TenantID, seed.ManagerID, "1234")

	for i := 1; i <= 4; i++ {
		code, resp := verifyPin(t, router, token, "9999")
		if code != http.StatusOK || resp.Data.Valid || resp.Data.AttemptsLeft != 5-i {
			t.Fatalf("failed attempt %d: got %d %+v, want 200 valid=false attempts_left=%d", i, code, resp.Data, 5-i)
		}
	}
	code, resp := verifyPin(t, router, token, "9999")
	if code != http.StatusLocked || resp.Error == nil || resp.Error.Code != "pin_locked" {
		t.Fatalf("5th failure: expected 423 pin_locked, got %d %+v", code, resp.Error)
	}

	// Even the correct pin is refused while locked.
	code, _ = verifyPin(t, router, token, "1234")
	if code != http.StatusLocked {
		t.Fatalf("correct pin while locked: expected 423, got %d", code)
	}
}

func TestAuthIntegration_PinFailureCounterResetsOnSuccess(t *testing.T) {
	router, h, seed, pool := setupAuthHandler(t)
	token := mintAuthAccess(t, h, seed, seed.ManagerID, "manager")
	testutil.SetManagerPIN(t, pool, seed.TenantID, seed.ManagerID, "1234")

	for i := 0; i < 3; i++ {
		if code, _ := verifyPin(t, router, token, "0000"); code != http.StatusOK {
			t.Fatalf("attempt %d: expected 200, got %d", i, code)
		}
	}
	code, resp := verifyPin(t, router, token, "1234")
	if code != http.StatusOK || !resp.Data.Valid {
		t.Fatalf("valid pin: expected 200 valid=true, got %d %+v", code, resp.Data)
	}
	// The wrong counter was reset: three more failures must not lock yet.
	for i := 0; i < 3; i++ {
		if code, _ := verifyPin(t, router, token, "0000"); code != http.StatusOK {
			t.Fatalf("post-reset attempt %d: expected 200, got %d", i, code)
		}
	}
}

func TestAuthIntegration_ManagerUnlocksCashierPin(t *testing.T) {
	router, h, seed, pool := setupAuthHandler(t)
	managerToken := mintAuthAccess(t, h, seed, seed.ManagerID, "manager")
	cashierToken := mintAuthAccess(t, h, seed, seed.CashierID, "cashier")
	testutil.SetManagerPIN(t, pool, seed.TenantID, seed.CashierID, "5678")

	for i := 0; i < 5; i++ {
		verifyPin(t, router, cashierToken, "0000")
	}
	if code, _ := verifyPin(t, router, cashierToken, "5678"); code != http.StatusLocked {
		t.Fatalf("cashier should be locked, got %d", code)
	}

	if code := unlockPin(t, router, managerToken, seed.CashierID); code != http.StatusNoContent {
		t.Fatalf("manager unlock: expected 204, got %d", code)
	}
	code, resp := verifyPin(t, router, cashierToken, "5678")
	if code != http.StatusOK || !resp.Data.Valid {
		var codeStr string
		if resp.Error != nil {
			codeStr = resp.Error.Code
		}
		t.Fatalf("cashier pin after unlock: expected 200 valid=true, got %d code=%s %+v", code, codeStr, resp.Data)
	}
}

func TestAuthIntegration_ManagerCannotUnlockManager(t *testing.T) {
	router, h, seed, pool := setupAuthHandler(t)
	managerToken := mintAuthAccess(t, h, seed, seed.ManagerID, "manager")
	testutil.SetManagerPIN(t, pool, seed.TenantID, seed.ManagerID, "1357")

	if code := unlockPin(t, router, managerToken, seed.ManagerID); code != http.StatusForbidden {
		t.Fatalf("manager unlocking a manager: expected 403, got %d", code)
	}
}

func TestAuthIntegration_CashierCannotUnlock(t *testing.T) {
	router, h, seed, _ := setupAuthHandler(t)
	cashierToken := mintAuthAccess(t, h, seed, seed.CashierID, "cashier")
	if code := unlockPin(t, router, cashierToken, seed.CashierID); code != http.StatusForbidden {
		t.Fatalf("cashier unlock: expected 403, got %d", code)
	}
}

func TestAuthIntegration_LoginReturnsResolvedPermissions(t *testing.T) {
	router, seed, pool := setupAuthIntegration(t)
	testutil.SetManagerPIN(t, pool, seed.TenantID, seed.ManagerID, "1234")

	rec := doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Data struct {
			User struct {
				Permissions *struct {
					AccessLevel    string `json:"access_level"`
					MaxDiscountPct int    `json:"max_discount_pct"`
					Refund         bool   `json:"can_refund"`
					NegativeStock  bool   `json:"can_negative_stock"`
				} `json:"permissions"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse login: %v", err)
	}
	p := out.Data.User.Permissions
	if p == nil {
		t.Fatal("login: permissions missing from user payload")
	}
	if p.AccessLevel != "manager" || p.MaxDiscountPct != 50 || !p.Refund || p.NegativeStock {
		t.Fatalf("login permissions: got %+v, want manager level / 50%% / refund", p)
	}
	_ = pool
}
