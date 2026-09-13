package sales

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func mustUUID(tag string) uuid.UUID {
	padded := tag + strings.Repeat("0", 32-len(tag))
	return uuid.MustParse(padded[:8] + "-" + padded[8:12] + "-" + padded[12:16] + "-" + padded[16:20] + "-" + padded[20:32])
}

func TestRefundSaleRejectsCashierRole(t *testing.T) {
	_, err := RefundSale(context.Background(), nil,
		mustUUID("a"), mustUUID("b"), "cashier", mustUUID("c"), "key", RefundRequest{})
	code, code2, _ := SaleError(err)
	if code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", code)
	}
	if code2 != "permission_denied" {
		t.Fatalf("expected permission_denied, got %q", code2)
	}
}

func TestRefundSaleRejectsUnknownRole(t *testing.T) {
	_, err := RefundSale(context.Background(), nil,
		mustUUID("1"), mustUUID("2"), "guest", mustUUID("3"), "key", RefundRequest{})
	_, code, _ := SaleError(err)
	if code != "permission_denied" {
		t.Fatalf("expected permission_denied, got %q", code)
	}
}

func TestRefundSaleAllowsManagerRole(t *testing.T) {
	_, err := RefundSale(context.Background(), nil,
		mustUUID("1"), mustUUID("2"), "manager", mustUUID("3"), "key", RefundRequest{Reason: strings.Repeat("x", 256)})
	_, code, _ := SaleError(err)
	if code != "validation_error" {
		t.Fatalf("expected validation_error, got %q", code)
	}
}

func TestRefundSaleHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RefundSale(ctx, nil,
		mustUUID("1"), mustUUID("2"), "manager", mustUUID("3"), "key", RefundRequest{})
	_, code, _ := SaleError(err)
	if code != "context_cancelled" {
		t.Fatalf("expected context_cancelled, got %q", code)
	}
}

func TestRefundRequiresAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost,
		"/v1/sales/00000000-0000-0000-0000-000000000001/refund", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestRefundReturnsUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens()).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost,
		"/v1/sales/00000000-0000-0000-0000-000000000001/refund", strings.NewReader(`{}`))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Idempotency-Key", "key-r")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 (nil pool before body validation), got %d", recorder.Code)
	}
}
