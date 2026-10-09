package support

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/pos-api/internal/identity"
	core "github.com/example/pos-api/internal/support"
	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// captureMailer records outbound envelopes instead of delivering them.
type captureMailer struct {
	sent []identity.EmailMessage
}

func (c *captureMailer) Send(_ context.Context, msg identity.EmailMessage) error {
	c.sent = append(c.sent, msg)
	return nil
}

func setupSupportIntegration(t *testing.T) (*gin.Engine, testutil.Seed, *pgxpool.Pool, *captureMailer) {
	t.Helper()
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	// support_messages is a global platform queue and the summary is global too,
	// so start each test from an empty inbox.
	if _, err := pool.Exec(context.Background(), `DELETE FROM support_messages`); err != nil {
		t.Fatalf("clear inbox: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`UPDATE tenants SET contact_email = $2 WHERE id = $1::uuid`,
		seed.TenantID, seed.ManagerEmail); err != nil {
		t.Fatalf("set contact email: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	h := NewHandler(pool, testutil.TokenManager(), "support", "support@xamltech.com", nil)
	cap := &captureMailer{}
	h.SetMailer(cap, "support@xamltech.com")
	h.Register(router.Group("/v1/saas"))
	return router, seed, pool, cap
}

func supportToken(t *testing.T, seed testutil.Seed) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-support", "saas_admin")
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, into any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), into); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
}

func TestSupportIntegration_IngestListReadReply(t *testing.T) {
	router, seed, pool, cap := setupSupportIntegration(t)
	token := supportToken(t, seed)
	ctx := context.Background()
	store := core.NewPGStore(pool)
	sender := seed.ManagerEmail

	first, err := store.Ingest(ctx, core.InboundMessage{
		Mailbox: "support", ProviderRef: "101", MessageID: "<m1@store>",
		FromName: "Store", FromEmail: sender, Subject: "Need a refund",
		BodyText: "Please refund order 12.", ReceivedAt: time.Now(),
	})
	if err != nil || !first {
		t.Fatalf("ingest first: stored=%v err=%v", first, err)
	}
	// Same UID again → replay-safe no-op.
	if again, err := store.Ingest(ctx, core.InboundMessage{
		Mailbox: "support", ProviderRef: "101", FromEmail: sender, Subject: "Need a refund",
	}); err != nil || again {
		t.Fatalf("replay ingest: stored=%v err=%v", again, err)
	}
	// A reply (In-Reply-To the first) must join the same thread.
	if _, err := store.Ingest(ctx, core.InboundMessage{
		Mailbox: "support", ProviderRef: "102", InReplyTo: "<m1@store>",
		FromEmail: sender, Subject: "Re: Need a refund", BodyText: "Any update?",
	}); err != nil {
		t.Fatalf("ingest reply: %v", err)
	}

	var threadID, tenantID string
	if err := pool.QueryRow(ctx,
		`SELECT thread_id::text, tenant_id::text FROM support_messages WHERE provider_ref = '101'`).
		Scan(&threadID, &tenantID); err != nil {
		t.Fatalf("load first row: %v", err)
	}
	if tenantID != seed.TenantID {
		t.Fatalf("tenant attribution = %q, want %q", tenantID, seed.TenantID)
	}
	var replyThread string
	if err := pool.QueryRow(ctx,
		`SELECT thread_id::text FROM support_messages WHERE provider_ref = '102'`).Scan(&replyThread); err != nil {
		t.Fatalf("load reply row: %v", err)
	}
	if replyThread != threadID {
		t.Fatalf("reply thread = %q, want %q", replyThread, threadID)
	}

	// List → 2 unread.
	rec := do(router, http.MethodGet, "/v1/saas/support/messages", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list: got %d: %s", rec.Code, rec.Body.String())
	}
	var list struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	mustDecode(t, rec, &list)
	if list.Meta.Total != 2 || len(list.Data) != 2 {
		t.Fatalf("list total=%d rows=%d, want 2", list.Meta.Total, len(list.Data))
	}

	// Summary → 2 unread.
	rec = do(router, http.MethodGet, "/v1/saas/support/summary", token, "")
	var summary struct {
		Data struct {
			Unread int `json:"unread"`
		} `json:"data"`
	}
	mustDecode(t, rec, &summary)
	if summary.Data.Unread != 2 {
		t.Fatalf("unread = %d, want 2", summary.Data.Unread)
	}

	// The first message's id, for read/reply.
	var firstID string
	if err := pool.QueryRow(ctx,
		`SELECT id::text FROM support_messages WHERE provider_ref = '101'`).Scan(&firstID); err != nil {
		t.Fatalf("load first id: %v", err)
	}
	rec = do(router, http.MethodGet, "/v1/saas/support/messages/"+firstID, token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get message: got %d", rec.Code)
	}
	rec = do(router, http.MethodPost, "/v1/saas/support/messages/"+firstID+"/read", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("mark read: got %d: %s", rec.Code, rec.Body.String())
	}

	// Reply → outbound row + a real send to the sender.
	rec = do(router, http.MethodPost, "/v1/saas/support/messages/"+firstID+"/reply", token,
		`{"body_text":"We have processed your refund."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("reply: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 1 {
		t.Fatalf("sends = %d, want 1", len(cap.sent))
	}
	if cap.sent[0].To != sender {
		t.Fatalf("reply to = %q, want %q", cap.sent[0].To, sender)
	}
	if cap.sent[0].Subject != "Re: Need a refund" {
		t.Fatalf("reply subject = %q", cap.sent[0].Subject)
	}
	var direction, outThread string
	if err := pool.QueryRow(ctx,
		`SELECT direction, thread_id::text FROM support_messages WHERE direction = 'outbound' ORDER BY created_at DESC LIMIT 1`).
		Scan(&direction, &outThread); err != nil {
		t.Fatalf("load outbound: %v", err)
	}
	if outThread != threadID {
		t.Fatalf("outbound thread = %q, want %q", outThread, threadID)
	}

	// Thread filter returns all three messages.
	rec = do(router, http.MethodGet, "/v1/saas/support/messages?thread_id="+threadID, token, "")
	mustDecode(t, rec, &list)
	if list.Meta.Total != 3 {
		t.Fatalf("thread total = %d, want 3", list.Meta.Total)
	}
}

func TestSupportIntegration_SendAndErrors(t *testing.T) {
	router, seed, pool, cap := setupSupportIntegration(t)
	token := supportToken(t, seed)

	// Compose a fresh outbound message.
	body := fmt.Sprintf(`{"to":"customer@example.com","subject":"Welcome","body_text":"Hello","tenant_id":%q}`, seed.TenantID)
	rec := do(router, http.MethodPost, "/v1/saas/support/messages", token, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send: got %d: %s", rec.Code, rec.Body.String())
	}
	if len(cap.sent) != 1 || cap.sent[0].To != "customer@example.com" {
		t.Fatalf("send capture = %+v", cap.sent)
	}
	var count int
	if err := pool.QueryRow(context.Background(),
		`SELECT COUNT(*) FROM support_messages WHERE direction = 'outbound' AND tenant_id = $1::uuid`,
		seed.TenantID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("outbound rows = %d, want 1", count)
	}

	// Invalid tenant id is rejected before send.
	rec = do(router, http.MethodPost, "/v1/saas/support/messages", token,
		`{"to":"x@example.com","subject":"s","body_text":"b","tenant_id":"nope"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad tenant: got %d, want 400", rec.Code)
	}
	// Unknown message id is a 404.
	rec = do(router, http.MethodGet, "/v1/saas/support/messages/00000000-0000-0000-0000-000000000000", token, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("missing message: got %d, want 404", rec.Code)
	}
	rec = do(router, http.MethodPost, "/v1/saas/support/messages/00000000-0000-0000-0000-000000000000/read", token, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("read missing: got %d, want 404", rec.Code)
	}
}

func TestSupportIntegration_ArchiveUnarchive(t *testing.T) {
	router, seed, pool, _ := setupSupportIntegration(t)
	token := supportToken(t, seed)
	ctx := context.Background()
	store := core.NewPGStore(pool)

	if _, err := store.Ingest(ctx, core.InboundMessage{
		Mailbox: "support", ProviderRef: "301", FromEmail: seed.ManagerEmail,
		Subject: "Pin reset", BodyText: "I locked myself out.", ReceivedAt: time.Now(),
	}); err != nil {
		t.Fatalf("ingest: %v", err)
	}
	var id string
	if err := pool.QueryRow(ctx,
		`SELECT id::text FROM support_messages WHERE provider_ref = '301'`).Scan(&id); err != nil {
		t.Fatalf("load id: %v", err)
	}

	// Archive → summary moves unread→archived and the status filter finds it.
	rec := do(router, http.MethodPost, "/v1/saas/support/messages/"+id+"/archive", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("archive: got %d: %s", rec.Code, rec.Body.String())
	}
	var summary struct {
		Data struct {
			Unread   int `json:"unread"`
			Archived int `json:"archived"`
		} `json:"data"`
	}
	rec = do(router, http.MethodGet, "/v1/saas/support/summary", token, "")
	mustDecode(t, rec, &summary)
	if summary.Data.Unread != 0 || summary.Data.Archived != 1 {
		t.Fatalf("summary after archive = %+v, want unread 0 / archived 1", summary.Data)
	}
	rec = do(router, http.MethodGet, "/v1/saas/support/messages?status=archived", token, "")
	var list struct {
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	mustDecode(t, rec, &list)
	if list.Meta.Total != 1 {
		t.Fatalf("archived list total = %d, want 1", list.Meta.Total)
	}

	// Re-archiving the same message is a no-op success.
	rec = do(router, http.MethodPost, "/v1/saas/support/messages/"+id+"/archive", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("re-archive: got %d", rec.Code)
	}

	// Unarchive → back to read.
	rec = do(router, http.MethodPost, "/v1/saas/support/messages/"+id+"/unarchive", token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("unarchive: got %d: %s", rec.Code, rec.Body.String())
	}
	var status string
	if err := pool.QueryRow(ctx,
		`SELECT status FROM support_messages WHERE id = $1::uuid`, id).Scan(&status); err != nil {
		t.Fatalf("load status: %v", err)
	}
	if status != core.StatusRead {
		t.Fatalf("status after unarchive = %q, want %q", status, core.StatusRead)
	}

	// Unknown ids are 404s on both actions.
	for _, act := range []string{"archive", "unarchive"} {
		rec = do(router, http.MethodPost, "/v1/saas/support/messages/00000000-0000-0000-0000-000000000000/"+act, token, "")
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s missing: got %d, want 404", act, rec.Code)
		}
	}
}
