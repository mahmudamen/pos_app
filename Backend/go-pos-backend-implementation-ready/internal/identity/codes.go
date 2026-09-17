package identity

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"regexp"
	"strings"
)

// Verification secrets policy:
//   - email tokens are 32 random bytes hex-encoded; only their sha256 hash is
//     stored, the raw value is handed to the mailer exactly once.
//   - phone OTPs are 6 random digits; only their sha256 hash is stored, the raw
//     value is handed to the SMS sender once. OTPs are never logged and never
//     stored in plaintext.
const (
	tokenBytes   = 32
	otpMaxLength = 6
)

var (
	emailStrip = regexp.MustCompile(`\s+`)
	phoneStrip = regexp.MustCompile(`[^\d+]`)
)

// NormalizeEmail lowercases and trims an email address into the canonical
// form used as the account key. Returns "" when the result is unusable.
func NormalizeEmail(raw string) string {
	cleaned := emailStrip.ReplaceAllString(strings.TrimSpace(raw), "")
	cleaned = strings.ToLower(cleaned)
	if len(cleaned) == 0 || len(cleaned) > 254 {
		return ""
	}
	at := strings.IndexByte(cleaned, '@')
	if at < 1 || at == len(cleaned)-1 {
		return ""
	}
	domain := cleaned[at+1:]
	if !strings.Contains(domain, ".") {
		return ""
	}
	return cleaned
}

// NormalizePhone converts a phone fingerprint to a loose E.164 shape: strips
// non-digits, keeps a leading '+', and rejects values outside 7–16 digits
// (the ITU-T E.164 range). It does not pretend to be libphonenumber; the
// product is Egypt-first and delivery providers normalize on send.
func NormalizePhone(raw string) string {
	cleaned := phoneStrip.ReplaceAllString(strings.TrimSpace(raw), "")
	if cleaned == "" {
		return ""
	}
	digits := strings.TrimPrefix(cleaned, "+")
	if len(digits) < 7 || len(digits) > 15 {
		return ""
	}
	if strings.HasPrefix(cleaned, "+") {
		return "+" + digits
	}
	return digits
}

// NewToken returns a random hex token and its sha256 digest. Only the digest
// may be persisted.
func NewToken() (plain, hash string, err error) {
	var raw [tokenBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(raw[:])
	return plain, HashHex(plain), nil
}

// NewOTP returns a random of digits and its sha256 digest. Only the digest may
// be persisted.
func NewOTP() (plain, hash string, err error) {
	const digitsToFill = 8 // generate 8, use two fewer so it is never biased by mod
	var raw [8]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	// #nosec G404 -- crypto/rand is used above; this reuses the same entropy.
	var sb strings.Builder
	for i := 0; i < otpMaxLength; i++ {
		b := raw[i] % 10
		// avoid leading-zero short codes: force first digit into 1..9
		if i == 0 && b == 0 {
			b = 1 + raw[i]%9
		}
		sb.WriteByte('0' + b)
	}
	plain = sb.String()
	return plain, HashHex(plain), nil
}

// HashHex returns the lowercase sha256 hex digest of s.
func HashHex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// ValidateOTP reports whether code has exactly 6 digits.
func ValidateOTP(code string) bool {
	if len(code) != otpMaxLength {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// ErrInvalidToken is returned when a verification token/OTP is malformed,
// unknown, expired, or already consumed. A uniform error avoids enumeration.
var ErrInvalidToken = errors.New("invalid or expired verification token")
