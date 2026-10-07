package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/identity"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// captureMailer records sends for assertions and can be told to fail.
type captureMailer struct {
	sent []identity.EmailMessage
	err  error
}

func (m *captureMailer) Send(_ context.Context, msg identity.EmailMessage) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, msg)
	return nil
}

func TestRegisterSendsWelcomeEmail(t *testing.T) {
	pool := testutil.Pool(t)
	if pool == nil {
		t.Skip("no TEST_DATABASE_URL/DATABASE_URL")
	}
	testutil.Migrate(t, testutil.DatabaseURL(t))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(pool, authConfig())
	mailer := &captureMailer{}
	h.SetMailer(mailer)
	h.Register(router.Group("/v1"))

	email := fmt.Sprintf("welcome-%s@example.com", uuid.NewString()[:8])
	rec := doPost(t, router, "/v1/auth/register", registerBody("coffee_shop", email), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d body=%s", rec.Code, rec.Body.String())
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected 1 welcome email, got %d", len(mailer.sent))
	}
	msg := mailer.sent[0]
	if msg.To != email {
		t.Fatalf("welcome To = %q, want %q", msg.To, email)
	}
	if msg.From != identity.SupportEmail {
		t.Fatalf("welcome From = %q, want %q", msg.From, identity.SupportEmail)
	}
	if !strings.Contains(msg.Text, "Onboard Demo") || !strings.Contains(msg.HTML, "Onboard Owner") {
		t.Fatalf("welcome body missing store/owner: text=%q", msg.Text)
	}
}

func TestRegisterSucceedsWhenWelcomeEmailFails(t *testing.T) {
	pool := testutil.Pool(t)
	if pool == nil {
		t.Skip("no TEST_DATABASE_URL/DATABASE_URL")
	}
	testutil.Migrate(t, testutil.DatabaseURL(t))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(pool, authConfig())
	h.SetMailer(&failingMailer{})
	h.Register(router.Group("/v1"))

	email := fmt.Sprintf("failmail-%s@example.com", uuid.NewString()[:8])
	rec := doPost(t, router, "/v1/auth/register", registerBody("book_store", email), "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("registration must succeed even if email fails: got %d body=%s", rec.Code, rec.Body.String())
	}
}

type failingMailer struct{}

func (failingMailer) Send(context.Context, identity.EmailMessage) error {
	return errors.New("smtp down")
}
