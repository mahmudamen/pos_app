package sales

import "testing"

func TestParseRoundingMode(t *testing.T) {
	cases := []struct {
		mode  string
		denom int64
		ok    bool
	}{
		{"", 0, true},
		{"off", 0, true},
		{"25", 25, true},
		{"50", 50, true},
		{"100", 100, true},
		{"250", 0, false},
		{"0", 0, false},
		{"cap", 0, false},
	}
	for _, c := range cases {
		denom, ok := ParseRoundingMode(c.mode)
		if denom != c.denom || ok != c.ok {
			t.Errorf("ParseRoundingMode(%q) = (%d, %v), want (%d, %v)", c.mode, denom, ok, c.denom, c.ok)
		}
	}
}

// RoundingDelta implements nearest-half-up rounding to a cash denomination,
// clamped so the payable is never below the nominal total (rounding_minor is
// always >= 0, satisfying the sales.rounding_minor CHECK). Under the clamp a
// total that would round down simply keeps its nominal value (delta 0), and a
// total that rounds up moves to the next denomination boundary.
func TestRoundingDeltaNearestHalfUp(t *testing.T) {
	cases := []struct {
		mode       string
		totalMinor int64
		want       int64
	}{
		{"off", 1234, 0},  // rounding off
		{"", 1234, 0},     // default is off
		{"25", 2000, 0},   // 20.00 exact
		{"25", 2512, 0},   // 25.12 -> 25.12 (rem 12 < half) stays nominal
		{"25", 2513, 12},  // 25.13 -> 25.25 (rem 13 >= half)
		{"25", 2550, 0},   // 25.50 exact
		{"50", 1024, 0},   // 10.24 -> 10.24 stays nominal
		{"50", 1025, 25},  // 10.25 -> 10.50 (tie rounds up)
		{"50", 1026, 24},  // 10.26 -> 10.50
		{"100", 1200, 0},  // 12.00 exact
		{"100", 1349, 0},  // 13.49 -> 13.49 stays nominal
		{"100", 1350, 50}, // 13.50 -> 14.00 (tie rounds up)
		{"100", 1351, 49}, // 13.51 -> 14.00
		{"100", 1450, 50}, // 14.50 -> 15.00
	}
	for _, c := range cases {
		got, err := RoundingDelta(c.mode, c.totalMinor)
		if err != nil {
			t.Errorf("RoundingDelta(%q, %d) unexpected error: %v", c.mode, c.totalMinor, err)
			continue
		}
		if got != c.want {
			t.Errorf("RoundingDelta(%q, %d) = %d, want %d", c.mode, c.totalMinor, got, c.want)
		}
		if c.mode != "off" && got < 0 {
			t.Errorf("RoundingDelta(%q, %d) = %d, must never be negative", c.mode, c.totalMinor, got)
		}
	}
}

func TestRoundingDeltaInvalidMode(t *testing.T) {
	if _, err := RoundingDelta("250", 1234); err == nil {
		t.Errorf("expected error for unknown mode, got nil")
	}
}
