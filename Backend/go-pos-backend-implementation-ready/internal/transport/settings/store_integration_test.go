package settings

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
)

type storeEnvelope struct {
	Data struct {
		Name    string `json:"name"`
		Address string `json:"address"`
	} `json:"data"`
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func setupStore(t *testing.T) (*gin.Engine, testutil.Seed) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(pool, testutil.TokenManager()).Register(router.Group("/v1"))
	return router, seed
}

func storeCall(t *testing.T, router http.Handler, method, token, body string) (int, storeEnvelope) {
	t.Helper()
	req := httptest.NewRequest(method, "/v1/store", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var env storeEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("parse %d body %q: %v", rec.Code, rec.Body.String(), err)
	}
	return rec.Code, env
}

func storeToken(t *testing.T, seed testutil.Seed, userID, role string) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, userID, seed.DeviceID, "sess-store", role)
}

func TestStoreRoundTrip(t *testing.T) {
	router, seed := setupStore(t)
	mgr := storeToken(t, seed, seed.ManagerID, "manager")

	code, env := storeCall(t, router, http.MethodGet, mgr, "")
	if code != http.StatusOK {
		t.Fatalf("get fresh: expected 200, got %d (%s)", code, env.Error.Code)
	}
	if env.Data.Name != "Test Store" || env.Data.Address != "" {
		t.Fatalf("fresh store should carry the seeded name only: %+v", env.Data)
	}

	code, env = storeCall(t, router, http.MethodPut, mgr,
		`{"name":"Al Fresco Cafe","address":"12 Nile Corniche"}`)
	if code != http.StatusOK || env.Error.Code != "" {
		t.Fatalf("update: expected 200, got %d (%s)", code, env.Error.Code)
	}
	if env.Data.Name != "Al Fresco Cafe" || env.Data.Address != "12 Nile Corniche" {
		t.Fatalf("updated store: %+v", env.Data)
	}

	code, env = storeCall(t, router, http.MethodGet, mgr, "")
	if code != http.StatusOK || env.Data.Address != "12 Nile Corniche" {
		t.Fatalf("get persisted: %+v (%d)", env.Data, code)
	}

	// Address may be cleared back to empty; name cannot.
	code, env = storeCall(t, router, http.MethodPut, mgr, `{"name":"Al Fresco Cafe","address":""}`)
	if code != http.StatusOK || env.Data.Address != "" {
		t.Fatalf("clear address: expected 200 empty, got %d %+v", code, env.Data)
	}
}

func TestStoreUpdateRestrictedToManagers(t *testing.T) {
	router, seed := setupStore(t)
	cashier := storeToken(t, seed, seed.CashierID, "cashier")
	if code, env := storeCall(t, router, http.MethodPut, cashier, `{"address":"Cairo"}`); code != http.StatusForbidden || env.Error.Code != "forbidden" {
		t.Fatalf("cashier PUT: expected 403 forbidden, got %d (%s)", code, env.Error.Code)
	}
}

func TestStoreTenantIsolation(t *testing.T) {
	router, seedA := setupStore(t)
	seedB := testutil.SeedTenant(t, testutil.Pool(t))

	mgrA := storeToken(t, seedA, seedA.ManagerID, "manager")
	mgrB := storeToken(t, seedB, seedB.ManagerID, "manager")

	storeCall(t, router, http.MethodPut, mgrA, `{"address":"Tenant A only"}`)
	_, envB := storeCall(t, router, http.MethodGet, mgrB, "")
	if envB.Data.Address == "Tenant A only" {
		t.Fatalf("tenant B sees tenant A store row: %+v", envB.Data)
	}
}
