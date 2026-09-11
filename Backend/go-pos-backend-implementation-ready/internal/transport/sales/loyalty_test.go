package sales

import (
	"testing"

	"github.com/google/uuid"
)

func TestSaleCustomerID(t *testing.T) {
	id := uuid.New()
	if got := saleCustomerID(&id); got != id.String() {
		t.Fatalf("expected %s, got %s", id, got)
	}
	if got := saleCustomerID(nil); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestLoyaltyRateFromValue(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"", defaultPointsPer100},
		{"  ", defaultPointsPer100},
		{"0", 0},
		{"1", 1},
		{"3", 3},
		{"abc", defaultPointsPer100},
		{"-2", defaultPointsPer100},
	}
	for _, tc := range tests {
		if got := loyaltyRateFromValue(tc.input); got != tc.want {
			t.Errorf("loyaltyRateFromValue(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

func TestPointsForTotal(t *testing.T) {
	tests := []struct {
		total int64
		rate  int64
		want  int64
	}{
		{0, 1, 0},
		{99, 1, 0},
		{100, 1, 1},
		{350, 1, 3},
		{350, 3, 9},
		{1000, 0, 0},
		{-100, 1, 0},
	}
	for _, tc := range tests {
		if got := pointsForTotal(tc.total, tc.rate); got != tc.want {
			t.Errorf("pointsForTotal(%d, %d) = %d, want %d", tc.total, tc.rate, got, tc.want)
		}
	}
}
