package platform

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/config"
	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type platformCapture struct {
	sent []identity.EmailMessage
}

func (c *platformCapture) Send(_ context.Context, msg identity.EmailMessage) error {
	c.sent = append(c.sent, msg)
	return nil
}

func setupPlatformMailIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool, *platformCapture) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET contact_email = $2 WHERE id = $1::uuid`,
		seed.TenantID, seed.ManagerEmail); err != nil {
		t.Fatalf("set contact email: %v", err)
	}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	cap := &platformCapture{}
	h := NewHandler(pool, testutil.TokenManager(), config.Config{})
	h.SetMailer(cap)
	h.Register(router.Group("/v1"))
	return router, seed, pool, cap
}

func postTenantAction(t *testing.T, router http.Handler, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	return rec
}

func TestPlatformIntegration_TenantLifecycleEmails(t *testing.T) {
	router, seed, _, cap := setupPlatformMailIntegration(t)
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-plat", "saas_admin")
	base := "/v1/platform/tenants/" + seed.TenantID

	rec := postTenantAction(t, router, base+"/suspend", token, `{"reason":"policy review"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspend: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 1 || cap.sent[0].To != seed.ManagerEmail {
		t.Fatalf("suspend emails = %+v", cap.sent)
	}
	if !strings.Contains(cap.sent[0].Subject, "إيقاف") && !strings.Contains(strings.ToLower(cap.sent[0].Subject), "suspended") {
		t.Fatalf("suspend subject = %q", cap.sent[0].Subject)
	}
	if !strings.Contains(cap.sent[0].Text, "policy review") {
		t.Fatalf("reason should be echoed: %q", cap.sent[0].Text)
	}

	rec = postTenantAction(t, router, base+"/activate", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("activate: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 2 {
		t.Fatalf("after activate, sends = %d, want 2", len(cap.sent))
	}

	rec = postTenantAction(t, router, base+"/stop", token, `{"reason":"fraud"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("stop: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 3 {
		t.Fatalf("after stop, sends = %d, want 3", len(cap.sent))
	}
}
