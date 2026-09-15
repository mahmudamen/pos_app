package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// USER-SECURITY DB-backed slice: create provisions a users_pos_security row,
// updateSecurity upserts it and returns resolved permissions, and listUsers
// resolves the profile for every row. Skips without a database.

func setupUsersIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed, pool
}

func usersToken(t *testing.T, seed testutil.Seed, userID, role string) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, userID, seed.DeviceID, "sess-users", role)
}

type securityResponse struct {
	Data struct {
		Permissions *struct {
			AccessLevel    string `json:"access_level"`
			MaxDiscountPct int    `json:"max_discount_pct"`
			DeleteOrder    bool   `json:"can_delete_order"`
			DeleteLine     bool   `json:"can_delete_line"`
			Refund         bool   `json:"can_refund"`
		} `json:"permissions"`
	} `json:"data"`
	Error *struct {
		Code string `json:"code"`
	} `json:"error"`
}

func TestUsersIntegration_CreateUserProvisionsAdminLevel(t *testing.T) {
	router, seed, _ := setupUsersIntegration(t)
	ownerToken := usersToken(t, seed, seed.ManagerID, "owner")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/users",
		bytes.NewBufferString(`{"email":"new-owner@example.com","display_name":"New Owner","password":"password123","role":"owner"}`))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create owner: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var out securityResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	p := out.Data.Permissions
	if p == nil || p.AccessLevel != "admin" || p.MaxDiscountPct != 100 || !p.DeleteOrder {
		t.Fatalf("owner permissions: got %+v, want admin/100/delete", p)
	}
}

func TestUsersIntegration_UpdateSecurityUpsertsAndResolves(t *testing.T) {
	router, seed, _ := setupUsersIntegration(t)
	ownerToken := usersToken(t, seed, seed.ManagerID, "owner")

	// Cashier starts at the cashier level (5% cap, no refund).
	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/v1/users?page=1&limit=50", nil)
	listReq.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list users: expected 200, got %d: %s", listRec.Code, listRec.Body.String())
	}
	var list struct {
		Data []struct {
			ID          string `json:"id"`
			Role        string `json:"role"`
			Permissions struct {
				AccessLevel string `json:"access_level"`
				Refund      bool   `json:"can_refund"`
			} `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	var cashierID string
	seededCashierFound := false
	for _, u := range list.Data {
		if u.Role == "cashier" {
			seededCashierFound = true
			cashierID = u.ID
			if u.Permissions.AccessLevel != "cashier" || u.Permissions.Refund {
				t.Fatalf("seeded cashier profile: got level=%s refund=%v, want cashier/false", u.Permissions.AccessLevel, u.Permissions.Refund)
			}
		}
	}
	if !seededCashierFound || cashierID == "" {
		t.Fatal("list: seeded cashier missing")
	}

	// Promote the cashier to advanced with a custom 10% + line-delete cap.
	body := `{"access_level":"advanced","max_discount_pct":10,"use_custom_permissions":true,"can_delete_line":true}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/users/"+cashierID+"/security", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update security: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var out securityResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	p := out.Data.Permissions
	if p == nil || p.AccessLevel != "advanced" || p.MaxDiscountPct != 10 || !p.DeleteLine || p.Refund {
		t.Fatalf("resolved permissions wrong: %+v", p)
	}

	// login-permissions resolution was covered by auth; verify persistence via list.
	listRec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/v1/users?page=1&limit=50", nil)
	req2.Header.Set("Authorization", "Bearer "+ownerToken)
	router.ServeHTTP(listRec2, req2)
	var list2 struct {
		Data []struct {
			ID          string `json:"id"`
			Permissions struct {
				AccessLevel string `json:"access_level"`
				DeleteLine  bool   `json:"can_delete_line"`
			} `json:"permissions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(listRec2.Body.Bytes(), &list2); err != nil {
		t.Fatal(err)
	}
	for _, u := range list2.Data {
		if u.ID == cashierID {
			if u.Permissions.AccessLevel != "advanced" || !u.Permissions.DeleteLine {
				t.Fatalf("persisted profile wrong: level=%s delete_line=%v", u.Permissions.AccessLevel, u.Permissions.DeleteLine)
			}
			return // success
		}
	}
	t.Fatal("updated cashier missing from list")
}

func TestUsersIntegration_ManagerCanEditCashierButNotManager(t *testing.T) {
	router, seed, _ := setupUsersIntegration(t)
	managerToken := usersToken(t, seed, seed.ManagerID, "manager")

	body := `{"access_level":"advanced"}`
	for _, target := range []struct {
		name string
		id   string
		want int
	}{
		{"cashier target", seed.CashierID, http.StatusOK},
		{"manager target", seed.ManagerID, http.StatusForbidden},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPut, "/v1/users/"+target.id+"/security", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+managerToken)
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		if rec.Code != target.want {
			t.Fatalf("%s: expected %d, got %d: %s", target.name, target.want, rec.Code, rec.Body.String())
		}
	}
}

func TestUsersIntegration_CashierCannotEditSecurity(t *testing.T) {
	router, seed, _ := setupUsersIntegration(t)
	cashierToken := usersToken(t, seed, seed.CashierID, "cashier")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/v1/users/"+seed.CashierID+"/security",
		bytes.NewBufferString(`{"access_level":"advanced"}`))
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cashier edit: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}
}
