package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var resetLinkPattern = regexp.MustCompile(`reset-password\?token=([0-9a-f]+)`)

// setupPasswordReset builds a handler with a capturing mailer, the way the
// register tests do: the emailed link is the only way a real user ever learns
// the token, so the test reads it out of the message rather than out of the
// database.
func setupPasswordReset(t *testing.T) (*gin.Engine, testutil.Seed, *captureMailer, *pgxpool.Pool) {
	t.Helper()
	pool := testutil.Pool(t)
	if pool == nil {
		t.Skip("no TEST_DATABASE_URL/DATABASE_URL")
	}
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(pool, authConfig())
	mailer := &captureMailer{}
	h.SetMailer(mailer)
	h.Register(router.Group("/v1"))
	return router, seed, mailer, pool
}

func forgotBody(tenant, email string) string {
	return fmt.Sprintf(`{"tenant_id":%q,"email":%q}`, tenant, email)
}

// loginWith is loginBody with an arbitrary password, so a rotated credential
// can be tried against the same device payload a real client sends.
func loginWith(email, password string, seed testutil.Seed) string {
	b, _ := json.Marshal(map[string]string{
		"email": email, "password": password,
		"tenant_id": seed.TenantID, "device_id": seed.DeviceCode, "device_name": "Counter 1",
	})
	return string(b)
}

func TestPasswordResetEndToEnd(t *testing.T) {
	router, seed, mailer, _ := setupPasswordReset(t)

	rec := doPost(t, router, "/v1/auth/forgot-password",
		forgotBody(seed.TenantID, seed.ManagerEmail), "")
	if rec.Code != http.StatusAccepted {
		t.Fatalf("forgot: expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected 1 reset email, got %d", len(mailer.sent))
	}
	msg := mailer.sent[0]
	if msg.To != seed.ManagerEmail {
		t.Fatalf("reset email To = %q, want %q", msg.To, seed.ManagerEmail)
	}
	match := resetLinkPattern.FindStringSubmatch(msg.Text)
	if match == nil {
		t.Fatalf("email carries no reset link: %s", msg.Text)
	}
	token := match[1]

	const newPassword = "RotatedPass2026"
	rec = doPost(t, router, "/v1/auth/reset-password",
		fmt.Sprintf(`{"token":%q,"password":%q}`, token, newPassword), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("reset: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	// The new credential works and the old one does not.
	rec = doPost(t, router, "/v1/auth/login", loginWith(seed.ManagerEmail, newPassword, seed), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login with new password: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	rec = doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login with old password: expected 401, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Single use: the same link cannot reset twice.
	rec = doPost(t, router, "/v1/auth/reset-password",
		fmt.Sprintf(`{"token":%q,"password":%q}`, token, "AnotherPass2026"), "")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_token") {
		t.Fatalf("replay: expected 400 invalid_token, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPasswordResetIsUniformForUnknownLogin(t *testing.T) {
	router, seed, mailer, _ := setupPasswordReset(t)

	known := doPost(t, router, "/v1/auth/forgot-password",
		forgotBody(seed.TenantID, seed.ManagerEmail), "")
	unknownEmail := doPost(t, router, "/v1/auth/forgot-password",
		forgotBody(seed.TenantID, "nobody-"+uuid.NewString()[:8]+"@example.com"), "")
	unknownTenant := doPost(t, router, "/v1/auth/forgot-password",
		forgotBody("no-such-store", seed.ManagerEmail), "")

	for name, rec := range map[string]*struct {
		Code int
		Body string
	}{
		"known":  {known.Code, known.Body.String()},
		"email":  {unknownEmail.Code, unknownEmail.Body.String()},
		"tenant": {unknownTenant.Code, unknownTenant.Body.String()},
	} {
		if rec.Code != http.StatusAccepted {
			t.Fatalf("%s: expected 202 (no enumeration), got %d", name, rec.Code)
		}
	}
	// Only the real login produced mail.
	if len(mailer.sent) != 1 {
		t.Fatalf("expected exactly 1 email for the known login, got %d", len(mailer.sent))
	}
	// Same data payload for all three, so the answers are not distinguishable.
	var data [3]map[string]any
	for i, rec := range []string{known.Body.String(), unknownEmail.Body.String(), unknownTenant.Body.String()} {
		var envelope struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal([]byte(rec), &envelope); err != nil {
			t.Fatalf("decode: %v", err)
		}
		data[i] = envelope.Data
	}
	if fmt.Sprint(data[0]) != fmt.Sprint(data[1]) || fmt.Sprint(data[0]) != fmt.Sprint(data[2]) {
		t.Fatalf("responses differ: known=%v email=%v tenant=%v", data[0], data[1], data[2])
	}
}

func TestPasswordResetRejectsUnknownToken(t *testing.T) {
	router, _, _, _ := setupPasswordReset(t)

	for name, token := range map[string]string{
		"random":    uuid.NewString(),
		"emptyish":  "deadbeef",
		"oversized": strings.Repeat("a", 200),
	} {
		t.Run(name, func(t *testing.T) {
			rec := doPost(t, router, "/v1/auth/reset-password",
				fmt.Sprintf(`{"token":%q,"password":"longenough1"}`, token), "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d body=%s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "invalid_token") {
				t.Fatalf("expected invalid_token, got %s", rec.Body.String())
			}
		})
	}
}

// A reset must not leave the old sessions alive: the password may have been
// stolen, and a stolen refresh token would otherwise outlive the fix.
func TestPasswordResetRevokesLiveSessions(t *testing.T) {
	router, seed, mailer, _ := setupPasswordReset(t)

	login := doPost(t, router, "/v1/auth/login", loginBody(seed), "")
	if login.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d body=%s", login.Code, login.Body.String())
	}
	var session struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(login.Body.Bytes(), &session); err != nil {
		t.Fatalf("decode login: %v", err)
	}
	if session.Data.RefreshToken == "" {
		t.Fatal("login returned no refresh token")
	}

	if rec := doPost(t, router, "/v1/auth/forgot-password",
		forgotBody(seed.TenantID, seed.ManagerEmail), ""); rec.Code != http.StatusAccepted {
		t.Fatalf("forgot: expected 202, got %d", rec.Code)
	}
	match := resetLinkPattern.FindStringSubmatch(mailer.sent[len(mailer.sent)-1].Text)
	if match == nil {
		t.Fatal("email carries no reset link")
	}
	if rec := doPost(t, router, "/v1/auth/reset-password",
		fmt.Sprintf(`{"token":%q,"password":"RotatedPass2026"}`, match[1]), ""); rec.Code != http.StatusOK {
		t.Fatalf("reset: expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}

	refresh := doPost(t, router, "/v1/auth/refresh",
		fmt.Sprintf(`{"refresh_token":%q}`, session.Data.RefreshToken), "")
	if refresh.Code == http.StatusOK {
		t.Fatalf("old refresh token must be revoked by the reset: got 200 %s", refresh.Body.String())
	}
}

func TestPasswordResetRateLimited(t *testing.T) {
	router, seed, _, _ := setupPasswordReset(t)
	body := forgotBody(seed.TenantID, seed.ManagerEmail)
	for i := 1; i <= identity.PasswordResetRateLimit; i++ {
		r := doPost(t, router, "/v1/auth/forgot-password", body, "")
		if r.Code == http.StatusTooManyRequests {
			t.Fatalf("attempt %d must pass the limiter", i)
		}
	}
	r := doPost(t, router, "/v1/auth/forgot-password", body, "")
	if r.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on attempt %d, got %d body=%s",
			identity.PasswordResetRateLimit, r.Code, r.Body.String())
	}
}
