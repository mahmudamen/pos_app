package identity

import (
	"strings"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	cases := map[string]string{
		"  UPPER@EXAMPLE.COM ": "upper@example.com",
		"a@b.c":                "a@b.c",
		"  ":                   "",
		"not-an-email":         "",
		" x@ ":                 "", // no domain part
	}
	for in, want := range cases {
		if got := NormalizeEmail(in); got != want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"+201001234567":    "+201001234567",
		"01001234567":      "01001234567",
		"+2 010 123 4567":  "+20101234567",
		"":                 "",
		"123456":           "",                 // below E.164 min (7)
		"+123456789012345": "+123456789012345", // 15 digits = valid E.164 max
		"not-a-phone":      "",
	}
	for in, want := range cases {
		if got := NormalizePhone(in); got != want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTokenHashingRoundTrip(t *testing.T) {
	plain, hash, err := NewToken()
	if err != nil {
		t.Fatalf("NewToken: %v", err)
	}
	if len(plain) != 64 {
		t.Errorf("plain token length = %d, want 64 hex chars", len(plain))
	}
	if HashHex(plain) != hash {
		t.Error("HashHex(plain) != stored hash")
	}
	if hash == plain {
		t.Error("stored value must never equal the plaintext token")
	}
}

func TestNewOTPAndValidate(t *testing.T) {
	plain, hash, err := NewOTP()
	if err != nil {
		t.Fatalf("NewOTP: %v", err)
	}
	if len(plain) != 6 {
		t.Errorf("otp length = %d, want 6", len(plain))
	}
	for _, r := range plain {
		if r < '0' || r > '9' {
			t.Errorf("otp contains non-digit %q", r)
		}
	}
	if HashHex(plain) != hash {
		t.Error("HashHex(plain) != stored hash")
	}
	if !ValidateOTP(plain) {
		t.Error("ValidateOTP rejected a valid 6-digit code")
	}
	if ValidateOTP("abc123") || ValidateOTP("12345") || ValidateOTP("1234567") || ValidateOTP("") {
		t.Error("ValidateOTP accepted an invalid code")
	}
	// Extra whitespace must never pass (codes are never trimmed server-side).
	if spacing := strings.TrimSpace(plain); spacing != plain && ValidateOTP(spacing+" ") {
		t.Error("ValidateOTP must not accept padded codes")
	}
}
