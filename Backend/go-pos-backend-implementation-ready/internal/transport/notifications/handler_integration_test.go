package notifications

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
)

// hub returns a router wired to a real Postgres pool (skips without
// TEST_DATABASE_URL/DATABASE_URL) plus the seeded tenant.
func hub(t *testing.T) (*gin.Engine, *Handler, testutil.Seed) {
	t.Helper()
	connString := testutil.DatabaseURL(t)
	pool := testutil.Pool(t)
	testutil.Migrate(t, connString)
	seed := testutil.SeedTenant(t, pool)
	h := NewHandler(pool, testutil.TokenManager(), WebPushConfig{})
	router := gin.New()
	h.Register(router.Group("/v1"))
	return router, h, seed
}

func managerToken(t *testing.T, seed testutil.Seed) string {
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", "manager")
}

func TestInboxRoundTrip(t *testing.T) {
	router, h, seed := hub(t)
	ctx := context.Background()

	// Emit one critical refund + one warning low-stock notification.
	userID := seed.ManagerID
	h.Emitter()(ctx, seed.TenantID, userID, TypeRefund, "refund:sale-x", SeverityCritical,
		"Refund applied", "A refund was issued.", map[string]any{"sale_id": "sale-x"})
	h.Emitter()(ctx, seed.TenantID, "", TypeLowStock, "low_stock:prod-y", SeverityWarning,
		"Low stock: Cola", "Only 2 left.", map[string]any{"product_id": "prod-y", "stock_quantity": int64(2)})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/notifications", nil)
	request.Header.Set("Authorization", "Bearer "+managerToken(t, seed))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Data struct {
			Items  []Notification `json:"items"`
			Unread int64          `json:"unread"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if body.Data.Unread != 2 {
		t.Fatalf("expected 2 unread, got %d", body.Data.Unread)
	}
	if len(body.Data.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(body.Data.Items))
	}
	var refundID string
	for _, n := range body.Data.Items {
		if n.Type == TypeRefund && !n.Read {
			refundID = n.ID
		}
	}
	if refundID == "" {
		t.Fatal("refund notification not found in inbox")
	}

	// Mark one as read; the unread count drops.
	recorder = httptest.NewRecorder()
	rq := httptest.NewRequest(http.MethodPatch, "/v1/notifications/"+refundID+"/read", nil)
	rq.Header.Set("Authorization", "Bearer "+managerToken(t, seed))
	router.ServeHTTP(recorder, rq)
	if recorder.Code != http.StatusOK {
		t.Fatalf("mark read: expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	rq = httptest.NewRequest(http.MethodGet, "/v1/notifications/count", nil)
	rq.Header.Set("Authorization", "Bearer "+managerToken(t, seed))
	router.ServeHTTP(recorder, rq)
	if recorder.Code != http.StatusOK {
		t.Fatalf("count: expected 200, got %d", recorder.Code)
	}
	var countBody struct {
		Data struct {
			Unread int64 `json:"unread"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &countBody); err != nil {
		t.Fatalf("decode count: %v", err)
	}
	if countBody.Data.Unread != 1 {
		t.Fatalf("expected 1 unread after marking one read, got %d", countBody.Data.Unread)
	}
}

func TestSubscribeAndRejectCashier(t *testing.T) {
	router, _, seed := hub(t)

	// Cashier lacks notifications.read → 403.
	recorder := httptest.NewRecorder()
	rq := httptest.NewRequest(http.MethodPost, "/v1/notifications/push-subscribe",
		strings.NewReader(`{"endpoint":"https://push.example/e1","keys":{"p256dh":"aA=","auth":"aA=="}}`))
	rq.Header.Set("Authorization", "Bearer "+testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "", "cashier"))
	rq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, rq)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cashier subscribe: expected 403, got %d", recorder.Code)
	}

	// Manager subscribes → 200 and a row persists.
	recorder = httptest.NewRecorder()
	rq = httptest.NewRequest(http.MethodPost, "/v1/notifications/push-subscribe",
		strings.NewReader(`{"endpoint":"https://push.example/manager","keys":{"p256dh":"dHg=","auth":"YXV0aA=="}}`))
	rq.Header.Set("Authorization", "Bearer "+managerToken(t, seed))
	rq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, rq)
	if recorder.Code != http.StatusOK {
		t.Fatalf("manager subscribe: expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCashierCannotReadInbox(t *testing.T) {
	router, _, seed := hub(t)
	recorder := httptest.NewRecorder()
	rq := httptest.NewRequest(http.MethodGet, "/v1/notifications", nil)
	rq.Header.Set("Authorization", "Bearer "+testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "", "cashier"))
	router.ServeHTTP(recorder, rq)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("cashier read: expected 403, got %d", recorder.Code)
	}
}
