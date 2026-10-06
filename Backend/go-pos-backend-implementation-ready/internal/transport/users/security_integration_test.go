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
				ChangeQty   bool   `json:"can_change_qty"`
			} `json:"permissions"`
			Security *struct {
				AccessLevel    string `json:"access_level"`
				MaxDiscountPct *int   `json:"max_discount_pct"`
				UseCustom      bool   `json:"use_custom_permissions"`
				DeleteLine     bool   `json:"can_delete_line"`
				ChangeQty      bool   `json:"can_change_qty"`
			} `json:"security"`
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
			// The raw block must echo the STORED row, not the resolved one:
			// can_change_qty was never set (raw false) but the advanced level
			// default resolves it to true — the editor needs both views.
			if u.Security == nil {
				t.Fatal("raw security block missing for owner viewer")
			}
			s := u.Security
			if s.AccessLevel != "advanced" || s.MaxDiscountPct == nil || *s.MaxDiscountPct != 10 ||
				!s.UseCustom || !s.DeleteLine || s.ChangeQty {
				t.Fatalf("raw security wrong: level=%s max=%v custom=%v delete_line=%v change_qty=%v",
					s.AccessLevel, s.MaxDiscountPct, s.UseCustom, s.DeleteLine, s.ChangeQty)
			}
			// With custom mode ON, Resolve mirrors the raw row — so the
			// resolved change_qty must agree with the stored false here.
			if u.Permissions.ChangeQty {
				t.Fatal("custom mode should resolve change_qty from the raw row (false)")
			}
			// Turn custom mode OFF: the stored can_* row is kept but Resolve
			// switches back to level defaults — this is exactly the delta the
			// raw security block exists to expose.
			rec3 := httptest.NewRecorder()
			req3 := httptest.NewRequest(http.MethodPut, "/v1/users/"+cashierID+"/security",
				bytes.NewBufferString(`{"access_level":"advanced","max_discount_pct":10,"use_custom_permissions":false}`))
			req3.Header.Set("Authorization", "Bearer "+ownerToken)
			req3.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(rec3, req3)
			if rec3.Code != http.StatusOK {
				t.Fatalf("update security (custom off): expected 200, got %d: %s", rec3.Code, rec3.Body.String())
			}
			listRec3 := httptest.NewRecorder()
			req4 := httptest.NewRequest(http.MethodGet, "/v1/users?page=1&limit=50", nil)
			req4.Header.Set("Authorization", "Bearer "+ownerToken)
			router.ServeHTTP(listRec3, req4)
			var list3 struct {
				Data []struct {
					ID          string `json:"id"`
					Permissions struct {
						ChangeQty bool `json:"can_change_qty"`
					} `json:"permissions"`
					Security *struct {
						UseCustom bool `json:"use_custom_permissions"`
						ChangeQty bool `json:"can_change_qty"`
					} `json:"security"`
				} `json:"data"`
			}
			if err := json.Unmarshal(listRec3.Body.Bytes(), &list3); err != nil {
				t.Fatal(err)
			}
			for _, v := range list3.Data {
				if v.ID != cashierID {
					continue
				}
				if v.Security == nil || v.Security.UseCustom || v.Security.ChangeQty {
					t.Fatalf("raw block after custom-off: %+v, want custom=false change_qty=false", v.Security)
				}
				if !v.Permissions.ChangeQty {
					t.Fatal("resolved permissions should keep the advanced change_qty default once custom is off")
				}
				return // success
			}
			t.Fatal("updated cashier missing from list")
		}
	}
	t.Fatal("updated cashier missing from list")
}

func TestUsersIntegration_RawSecurityHiddenFromCashiers(t *testing.T) {
	router, seed, _ := setupUsersIntegration(t)
	cashierToken := usersToken(t, seed, seed.CashierID, "cashier")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/users?page=1&limit=50", nil)
	req.Header.Set("Authorization", "Bearer "+cashierToken)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list as cashier: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Data []struct {
			Security json.RawMessage `json:"security"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Data) == 0 {
		t.Fatal("list as cashier: expected rows")
	}
	for _, u := range list.Data {
		if len(u.Security) != 0 {
			t.Fatalf("cashier must not see raw security blocks, got %s", u.Security)
		}
	}
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
