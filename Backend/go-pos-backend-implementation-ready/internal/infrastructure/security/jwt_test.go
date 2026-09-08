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
