package sync

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

func testTokens() security.TokenManager {
	return security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func pullToken(t *testing.T) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003",
		"00000000-0000-0000-0000-000000000004",
		"cashier")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func setup() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router, router.Group("/v1")
}

func TestPullRouteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	found := false
	for _, r := range router.Routes() {
		if r.Method == http.MethodGet && r.Path == "/v1/sync/pull" {
			found = true
		}
	}
	if !found {
		t.Fatal("GET /v1/sync/pull not registered")
	}
}

func TestPullRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/sync/pull", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestPullUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/sync/pull?cursor=100", nil)
	request.Header.Set("Authorization", "Bearer "+pullToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestPullRejectsInvalidCursor(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, cursor := range []string{"abc", "-5"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, "/v1/sync/pull?cursor="+cursor, nil)
		request.Header.Set("Authorization", "Bearer "+pullToken(t))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("cursor %q: expected 400, got %d", cursor, recorder.Code)
		}
	}
}

func TestPullDefaultsCursorToZero(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/sync/pull", nil)
	request.Header.Set("Authorization", "Bearer "+pullToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (nil pool reached with default cursor), got %d", recorder.Code)
	}
}

func TestPushRouteRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	found := false
	for _, r := range router.Routes() {
		if r.Method == http.MethodPost && r.Path == "/v1/sync/push" {
			found = true
		}
	}
	if !found {
		t.Fatal("POST /v1/sync/push not registered")
	}
}

func TestPushRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/sync/push", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestPushRejectsInvalidBody(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	for _, body := range []string{"not-json", `{"commands":[]}`, `{"commands":1200}`} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/sync/push", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+pullToken(t))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %q: expected 400, got %d", body, recorder.Code)
		}
	}
}

func TestPushUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/sync/push", strings.NewReader(`{"commands":[{"command_id":"11111111-1111-1111-1111-111111111111","operation":"sale.create","payload":{}}]}`))
	request.Header.Set("Authorization", "Bearer "+pullToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (nil pool reached with valid body), got %d", recorder.Code)
	}
}

func TestApplyRejectsInvalidCommandID(t *testing.T) {
	result := applyCommand(context.Background(), nil, &Handler{}, pushCommand{
		CommandID: "not-a-uuid",
		Operation: "sale.create",
		Payload:   map[string]any{},
	}, uuid.Nil, uuid.Nil, uuid.Nil, "cashier")
	if result.Status != "rejected" {
		t.Fatalf("expected rejected, got %q", result.Status)
	}
	if result.ErrorCode != "validation_error" {
		t.Fatalf("expected validation_error, got %q", result.ErrorCode)
	}
}

func TestApplyRejectsUnknownOperation(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, pushCommand{
		CommandID: "11111111-1111-1111-1111-111111111111",
		Operation: "foo.create",
		Payload:   map[string]any{},
	}, uuid.Nil, uuid.Nil, uuid.Nil, "cashier")
	if result.Status != "rejected" {
		t.Fatalf("expected rejected, got %q", result.Status)
	}
	if result.ErrorCode != "unknown_command" {
		t.Fatalf("expected unknown_command, got %q", result.ErrorCode)
	}
}

func TestHashPayloadIsDeterministic(t *testing.T) {
	a := hashPayload(map[string]any{"items": []any{}})
	b := hashPayload(map[string]any{"items": []any{}})
	if a != b {
		t.Fatal("hash must be deterministic for identical payloads")
	}
	if len(a) != 64 {
		t.Fatalf("expected sha256 hex (64 chars), got %d", len(a))
	}
	c := hashPayload(map[string]any{"items": []any{map[string]any{"quantity": 2}}})
	if c == a {
		t.Fatal("different payloads must hash differently")
	}
}

func cmd(op string, payload map[string]any) pushCommand {
	return pushCommand{
		CommandID: "11111111-1111-1111-1111-111111111111",
		Operation: op,
		Payload:   payload,
	}
}

func TestApplyRefundSaleRejectsInvalidSaleID(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("sale.refund", map[string]any{}), uuid.Nil, uuid.Nil, uuid.Nil, "manager")
	if result.Status != "rejected" || result.ErrorCode != "validation_error" {
		t.Fatalf("expected rejected/validation_error, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyRefundSaleRejectsNonManager(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("sale.refund", map[string]any{"sale_id": "00000000-0000-0000-0000-000000000001"}), uuid.Nil, uuid.Nil, uuid.Nil, "cashier")
	if result.Status != "rejected" || result.ErrorCode != "permission_denied" {
		t.Fatalf("expected rejected/permission_denied, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyInventoryAdjustRejectsInvalidReason(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("inventory.adjust", map[string]any{"product_id": "00000000-0000-0000-0000-000000000001", "reason": "mystery"}), uuid.Nil, uuid.Nil, uuid.Nil, "manager")
	if result.Status != "rejected" || result.ErrorCode != "validation_error" {
		t.Fatalf("expected rejected/validation_error, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyInventoryAdjustRejectsZeroDelta(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("inventory.adjust", map[string]any{"product_id": "00000000-0000-0000-0000-000000000001", "reason": "count", "quantity_delta": 0}), uuid.Nil, uuid.Nil, uuid.Nil, "manager")
	if result.Status != "rejected" || result.ErrorCode != "validation_error" {
		t.Fatalf("expected rejected/validation_error, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyInventoryAdjustRejectsCashier(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("inventory.adjust", map[string]any{"product_id": "00000000-0000-0000-0000-000000000001", "reason": "count", "quantity_delta": 1}), uuid.Nil, uuid.Nil, uuid.Nil, "cashier")
	if result.Status != "rejected" || result.ErrorCode != "permission_denied" {
		t.Fatalf("expected rejected/permission_denied, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyCategoryCreateRejectsBlankSlug(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("category.create", map[string]any{"name": "Drinks"}), uuid.Nil, uuid.Nil, uuid.Nil, "manager")
	if result.Status != "rejected" || result.ErrorCode != "validation_error" {
		t.Fatalf("expected rejected/validation_error, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyCategoryCreateRejectsCashier(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("category.create", map[string]any{"name": "Drinks", "slug": "drinks"}), uuid.Nil, uuid.Nil, uuid.Nil, "cashier")
	if result.Status != "rejected" || result.ErrorCode != "permission_denied" {
		t.Fatalf("expected rejected/permission_denied, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyProductCreateRejectsInvalidPrice(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("product.create", map[string]any{"name": "Cola", "sku": "COLA", "price_minor": -5}), uuid.Nil, uuid.Nil, uuid.Nil, "manager")
	if result.Status != "rejected" || result.ErrorCode != "validation_error" {
		t.Fatalf("expected rejected/validation_error, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestApplyProductCreateRejectsCashier(t *testing.T) {
	result := apply(context.Background(), nil, &Handler{}, cmd("product.create", map[string]any{"name": "Cola", "sku": "COLA", "price_minor": 100, "currency": "EGP"}), uuid.Nil, uuid.Nil, uuid.Nil, "cashier")
	if result.Status != "rejected" || result.ErrorCode != "permission_denied" {
		t.Fatalf("expected rejected/permission_denied, got %q/%q", result.Status, result.ErrorCode)
	}
}

func TestPushStatusFor(t *testing.T) {
	cases := []struct {
		http int
		want string
	}{
		{http.StatusConflict, "conflict"},
		{http.StatusNotFound, "rejected"},
		{http.StatusBadRequest, "rejected"},
		{http.StatusForbidden, "rejected"},
	}
	for _, c := range cases {
		if got := pushStatusFor(c.http); got != c.want {
			t.Fatalf("pushStatusFor(%d) = %q, want %q", c.http, got, c.want)
		}
	}
}

func TestIsUniqueViolation(t *testing.T) {
	pgErr := &pgconn.PgError{Code: "23505", Message: "duplicate key value"}
	if !isUniqueViolation(pgErr) {
		t.Fatal("expected PgError 23505 to be a unique violation")
	}
	for _, other := range []error{
		&pgconn.PgError{Code: "23503"},
		&pgconn.PgError{Code: "23502"},
		errors.New("some other error"),
		nil,
	} {
		if other == nil {
			if isUniqueViolation(nil) {
				t.Fatal("expected nil error to not be a unique violation")
			}
			continue
		}
		if isUniqueViolation(other) {
			t.Fatalf("expected %v to not be a unique violation", other)
		}
	}
	if !isUniqueViolation(fmt.Errorf("wrapped: %w", &pgconn.PgError{Code: "23505"})) {
		t.Fatal("expected a wrapped 23505 to be a unique violation")
	}
}
