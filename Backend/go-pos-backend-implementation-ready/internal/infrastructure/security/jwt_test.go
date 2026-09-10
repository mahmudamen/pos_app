package security

import (
	"testing"
	"time"
)

func TestTokenManagerIssuesAndParsesTypedTokens(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := manager.Issue(time.Now(), AccessToken, "tenant-1", "user-1", "device-1", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.Parse(raw, AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TenantID != "tenant-1" || claims.UserID != "user-1" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
	if _, err := manager.Parse(raw, RefreshToken); err == nil {
		t.Fatal("access token accepted as refresh token")
	}
}

func TestPasswordHashing(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple", 10)
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "correct horse battery staple") || CheckPassword(hash, "wrong password") {
		t.Fatal("password verification failed")
	}
}

func TestTokenManagerRejectsExpiredToken(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := manager.Issue(time.Now().Add(-5*time.Minute), AccessToken, "tenant", "user", "device", "session")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager.Parse(raw, AccessToken)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestTokenManagerRejectsWrongIssuer(t *testing.T) {
	manager1 := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	manager2 := TokenManager{
		Issuer: "other-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := manager1.Issue(time.Now(), AccessToken, "tenant", "user", "device", "session")
	if err != nil {
		t.Fatal(err)
	}
	_, err = manager2.Parse(raw, AccessToken)
	if err == nil {
		t.Fatal("expected error for wrong issuer")
	}
}

func TestTokenManagerRejectsShortSecret(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("short"),
		RefreshSecret: []byte("short"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	_, err := manager.Issue(time.Now(), AccessToken, "tenant", "user", "device", "session")
	if err == nil {
		t.Fatal("expected error for short secret")
	}
}

func TestTokenManagerRejectsParseWithShortSecret(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := manager.Issue(time.Now(), AccessToken, "tenant", "user", "device", "session")
	if err != nil {
		t.Fatal(err)
	}
	shortManager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("short"),
		RefreshSecret: []byte("short"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	_, err = shortManager.Parse(raw, AccessToken)
	if err == nil {
		t.Fatal("expected error for short secret on parse")
	}
}

func TestTokenManagerIssueWithRole(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := manager.IssueWithRole(time.Now(), AccessToken, "tenant", "user", "device", "session", "manager")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.Parse(raw, AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Role != "manager" {
		t.Errorf("Role: got %q, want %q", claims.Role, "manager")
	}
}

func TestTokenManagerRejectsRandomGarbage(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	_, err := manager.Parse("not.a.jwt.token", AccessToken)
	if err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestTokenManagerRejectsUnknownTokenType(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	_, err := manager.Issue(time.Now(), TokenType("unknown"), "tenant", "user", "device", "session")
	if err == nil {
		t.Fatal("expected error for unknown token type")
	}
}

func TestTokenManagerRoundTripPreservesAllClaims(t *testing.T) {
	manager := TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"),
		AccessTTL:     time.Minute, RefreshTTL: time.Hour,
	}
	raw, err := manager.IssueWithRole(time.Now(), AccessToken,
		"tenant-abc", "user-xyz", "device-123", "session-456", "cashier")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := manager.Parse(raw, AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	if claims.TenantID != "tenant-abc" {
		t.Errorf("TenantID: got %q", claims.TenantID)
	}
	if claims.UserID != "user-xyz" {
		t.Errorf("UserID: got %q", claims.UserID)
	}
	if claims.DeviceID != "device-123" {
		t.Errorf("DeviceID: got %q", claims.DeviceID)
	}
	if claims.SessionID != "session-456" {
		t.Errorf("SessionID: got %q", claims.SessionID)
	}
	if claims.Role != "cashier" {
		t.Errorf("Role: got %q", claims.Role)
	}
	if claims.TokenType != AccessToken {
		t.Errorf("TokenType: got %q", claims.TokenType)
	}
}

func TestCheckPasswordRejectsEmptyPassword(t *testing.T) {
	hash, err := HashPassword("password", 4)
	if err != nil {
		t.Fatal(err)
	}
	if CheckPassword(hash, "") {
		t.Fatal("empty password should not match")
	}
}
