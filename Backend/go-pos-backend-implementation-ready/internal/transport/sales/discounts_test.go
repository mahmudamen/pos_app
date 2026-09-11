package sales

import (
	"errors"
	"testing"
)

func TestValidateDiscount(t *testing.T) {
	cases := []struct {
		name            string
		role            string
		subtotal        int64
		discount        int64
		cashierLimitPct int
		wantTotal       int64
		wantErr         error
	}{
		{"zero discount", "cashier", 1000, 0, 5, 1000, nil},
		{"cashier within limit", "cashier", 1000, 40, 5, 960, nil},
		{"cashier at limit", "cashier", 1000, 50, 5, 950, nil},
		{"cashier over limit", "cashier", 1000, 60, 5, 0, ErrDiscountOverLimit},
		{"manager no cap", "manager", 1000, 900, 5, 100, nil},
		{"owner no cap", "owner", 1000, 1000, 5, 0, nil},
		{"saas_admin no cap", "saas_admin", 1000, 500, 5, 500, nil},
		{"negative discount", "cashier", 1000, -1, 5, 0, ErrNegativeDiscount},
		{"discount exceeds subtotal", "manager", 1000, 1001, 5, 0, ErrDiscountExceedsSubtotal},
		{"empty role forbidden", "", 1000, 100, 5, 0, ErrDiscountForbidden},
		{"unknown role forbidden", "guest", 1000, 100, 5, 0, ErrDiscountForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateDiscount(tc.role, tc.subtotal, tc.discount, tc.cashierLimitPct)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tc.wantErr)
				}
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("got error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantTotal {
				t.Errorf("got total %d, want %d", got, tc.wantTotal)
			}
		})
	}
}
