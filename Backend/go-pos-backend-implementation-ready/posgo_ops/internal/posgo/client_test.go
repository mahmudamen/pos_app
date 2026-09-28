package posgo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestServer stands up a control-plane stub that records the last request it
// saw, so tests can assert on method, path, query and auth header as well as
// on the decoded result.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *http.Request) {
	t.Helper()
	seen := &http.Request{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = *r
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, seen
}

// writeJSON is the envelope every control-plane route replies with.
func writeJSON(t *testing.T, w http.ResponseWriter, status int, data any, meta map[string]any) {
	t.Helper()
	body := map[string]any{"data": data}
	if meta != nil {
		body["meta"] = meta
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

// writeErr is the failure envelope.
func writeErr(t *testing.T, w http.ResponseWriter, status int, code, message string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"code": code, "message": message},
	}); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

func TestLoginStoresTokenAndEchoesSession(t *testing.T) {
	srv, seen := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"access_token":  "access-123",
			"refresh_token": "refresh-123",
			"expires_in":    900,
			"tenant": map[string]any{
				"id": "saas-tenant", "currency_code": "EGP", "country_code": "EG",
			},
			"user": map[string]any{
				"id": "user-1", "display_name": "SaaS Admin", "role": "saas_admin",
			},
		}, nil)
	})

	c := NewClient(srv.URL, srv.Client())
	session, err := c.Login(context.Background(), "admin@posgo.saas", "admin", "saas-tenant", "dev-1", "console")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if session.AccessToken != "access-123" || c.Token() != "access-123" {
		t.Fatalf("token not stored: session=%q client=%q", session.AccessToken, c.Token())
	}
	if session.DisplayName != "SaaS Admin" || !session.IsAdmin() {
		t.Fatalf("session echo wrong: %+v", session)
	}
	if session.Currency != "EGP" || session.TenantID != "saas-tenant" {
		t.Fatalf("tenant echo wrong: %+v", session)
	}
	if seen.Method != http.MethodPost || seen.URL.Path != "/v1/auth/login" {
		t.Fatalf("unexpected request: %s %s", seen.Method, seen.URL.Path)
	}
	// Login must not send a bearer token; it is how the token is obtained.
	if auth := seen.Header.Get("Authorization"); auth != "" {
		t.Fatalf("login sent an Authorization header: %q", auth)
	}
	if ct := seen.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q", ct)
	}
}

func TestLoginRejectionCarriesServerCode(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeErr(t, w, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
	})

	c := NewClient(srv.URL, srv.Client())
	_, err := c.Login(context.Background(), "admin@posgo.saas", "wrong", "saas-tenant", "dev-1", "console")
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want *APIError, got %T (%v)", err, err)
	}
	if apiErr.Code != "invalid_credentials" {
		t.Fatalf("code = %q", apiErr.Code)
	}
	if !apiErr.IsAuthFailure() {
		t.Fatalf("401 should be an auth failure")
	}
	if !strings.Contains(apiErr.Error(), "invalid_credentials") {
		t.Fatalf("error text missing code: %q", apiErr.Error())
	}
	// A failed login must not leave a token behind.
	if c.Token() != "" {
		t.Fatalf("token set after failed login: %q", c.Token())
	}
}

func TestAuthenticatedCallsSendBearer(t *testing.T) {
	srv, seen := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{"total_tenants": 7}, nil)
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("token-abc")
	if _, err := c.Summary(context.Background()); err != nil {
		t.Fatalf("Summary: %v", err)
	}
	if got := seen.Header.Get("Authorization"); got != "Bearer token-abc" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestCallWithoutTokenFailsBeforeDial(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should not be reached without a token")
	})

	c := NewClient(srv.URL, srv.Client())
	_, err := c.Summary(context.Background())
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "no_session" {
		t.Fatalf("want no_session APIError, got %T %v", err, err)
	}
}

func TestTenantsSendsFiltersAndDecodesMeta(t *testing.T) {
	srv, seen := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, []map[string]any{
			{"id": "t1", "name": "Onboard Demo", "status": "active", "plan": "trial",
				"plan_features": []string{"pos.basic", "pos.subdomain"}, "revenue_minor": 1250},
			{"id": "t2", "name": "Cafe", "status": "disabled", "plan": "starter", "revenue_minor": -500},
		}, map[string]any{"page": 2, "limit": 25, "total": 4329})
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("token-abc")
	page, err := c.Tenants(context.Background(), Filters{
		Search: "  cafe  ", BusinessType: "restaurant", Status: "active", Page: 2, Limit: 25,
	})
	if err != nil {
		t.Fatalf("Tenants: %v", err)
	}

	query := seen.URL.Query()
	if query.Get("q") != "cafe" {
		t.Fatalf("q = %q, want trimmed %q", query.Get("q"), "cafe")
	}
	if query.Get("business_type") != "restaurant" || query.Get("status") != "active" {
		t.Fatalf("filters not forwarded: %v", query)
	}
	if query.Get("page") != "2" || query.Get("limit") != "25" {
		t.Fatalf("pagination not forwarded: %v", query)
	}
	if page.Total != 4329 || page.Page != 2 || page.Limit != 25 {
		t.Fatalf("meta not decoded: %+v", page)
	}
	if len(page.Tenants) != 2 {
		t.Fatalf("want 2 tenants, got %d", len(page.Tenants))
	}
	first := page.Tenants[0]
	if !first.HasFeature("pos.subdomain") || first.HasFeature("nope") {
		t.Fatalf("plan features wrong: %v", first.PlanFeatures)
	}
	if got := FormatMinor(first.RevenueMinor, "EGP"); got != "E£12.50" {
		t.Fatalf("revenue = %q", got)
	}
	if !page.Tenants[1].IsStopped() {
		t.Fatalf("disabled tenant should report IsStopped")
	}
}

func TestTenantsDefaultsPaginationWhenMetaAbsent(t *testing.T) {
	srv, seen := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, []map[string]any{}, nil)
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("t")
	page, err := c.Tenants(context.Background(), Filters{Limit: 10})
	if err != nil {
		t.Fatalf("Tenants: %v", err)
	}
	if page.Page != 1 || page.Limit != 10 {
		t.Fatalf("defaults not applied: %+v", page)
	}
	if got := seen.URL.Query().Get("limit"); got != "10" {
		t.Fatalf("limit = %q", got)
	}
}

func TestAnalyticsDecodesNestedBlocks(t *testing.T) {
	srv, seen := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, http.StatusOK, map[string]any{
			"tenant":        map[string]any{"id": "t1", "name": "Cafe", "currency_code": "EGP"},
			"counts":        map[string]any{"users": 6, "products": 15},
			"today":         map[string]any{"date": "2026-10-03", "revenue_minor": 97860, "sales_count": 4, "avg_sale_minor": 24465, "items_sold": 9},
			"revenue_trend": []map[string]any{{"day": "2026-10-02", "revenue_minor": 1000}},
			"top_products":  []map[string]any{{"product_name": "Coffee", "sku": "C1", "quantity": 3, "revenue_minor": 4500}},
			"recent_sales":  []map[string]any{{"id": "s1", "status": "completed", "total_minor": 350, "currency": "EGP", "payment_method": "cash", "cashier": "Mona"}},
		}, nil)
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("t")
	got, err := c.Analytics(context.Background(), "t1")
	if err != nil {
		t.Fatalf("Analytics: %v", err)
	}
	if seen.URL.Path != "/v1/saas/tenants/t1/analytics" {
		t.Fatalf("path = %q", seen.URL.Path)
	}
	if got.Counts.Users != 6 || got.Counts.Products != 15 {
		t.Fatalf("counts = %+v", got.Counts)
	}
	if got.Today.RevenueMinor != 97860 || got.Today.AvgSaleMinor != 24465 {
		t.Fatalf("today = %+v", got.Today)
	}
	if len(got.RevenueTrend) != 1 || len(got.TopProducts) != 1 || len(got.RecentSales) != 1 {
		t.Fatalf("lists not decoded: %+v", got)
	}
	if got.RecentSales[0].Cashier != "Mona" {
		t.Fatalf("recent sale = %+v", got.RecentSales[0])
	}
}

func TestAnalyticsRejectsEmptyTenantID(t *testing.T) {
	c := NewClient("http://127.0.0.1:1", nil)
	c.SetToken("t")
	if _, err := c.Analytics(context.Background(), "  "); err == nil {
		t.Fatal("want error for empty tenant id")
	}
}

func TestPlatformActionsUsePost(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*Client) error
		want string
	}{
		{"activate", func(c *Client) error { return c.Activate(context.Background(), "t1") }, "/v1/platform/tenants/t1/activate"},
		{"suspend", func(c *Client) error { return c.Suspend(context.Background(), "t1") }, "/v1/platform/tenants/t1/suspend"},
		{"stop", func(c *Client) error { return c.Stop(context.Background(), "t1", "abuse") }, "/v1/platform/tenants/t1/stop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv, seen := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, http.StatusOK, map[string]any{"status": "ok"}, nil)
			})
			c := NewClient(srv.URL, srv.Client())
			c.SetToken("t")
			if err := tc.call(c); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if seen.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", seen.Method)
			}
			if seen.URL.Path != tc.want {
				t.Fatalf("path = %s, want %s", seen.URL.Path, tc.want)
			}
		})
	}
}

func TestStopSendsReasonOnlyWhenPresent(t *testing.T) {
	var bodies []string
	srv, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		bodies = append(bodies, string(raw))
		writeJSON(t, w, http.StatusOK, map[string]any{}, nil)
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("t")
	if err := c.Stop(context.Background(), "t1", "chargeback abuse"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := c.Stop(context.Background(), "t2", ""); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if !strings.Contains(bodies[0], "chargeback abuse") {
		t.Fatalf("reason not sent: %q", bodies[0])
	}
	if strings.TrimSpace(bodies[1]) != "" && strings.TrimSpace(bodies[1]) != "null" {
		t.Fatalf("empty reason should send no payload, got %q", bodies[1])
	}
}

func TestBackupReturnsRawJSONNotEnvelope(t *testing.T) {
	srv, seen := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"tenant_id":"t1","slug":"cafe","tables":{"users":[{}]}}`)); err != nil {
			t.Errorf("write: %v", err)
		}
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("t")
	raw, err := c.Backup(context.Background(), "t1")
	if err != nil {
		t.Fatalf("Backup: %v", err)
	}
	if seen.URL.Path != "/v1/platform/tenants/t1/backup" || seen.Method != http.MethodPost {
		t.Fatalf("unexpected request %s %s", seen.Method, seen.URL.Path)
	}
	// The backup body is the export itself, so it must not be unwrapped.
	if !strings.Contains(raw, `"tenant_id":"t1"`) || !strings.Contains(raw, `"users"`) {
		t.Fatalf("backup body mangled: %q", raw)
	}
}

func TestBackupSurfacesErrorEnvelope(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeErr(t, w, http.StatusForbidden, "forbidden", "saas_admin role required")
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("t")
	_, err := c.Backup(context.Background(), "t1")
	apiErr, ok := err.(*APIError)
	if !ok || apiErr.Code != "forbidden" || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("want forbidden APIError, got %T %v", err, err)
	}
}

func TestErrorToleratesNonEnvelopeBody(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		if _, err := w.Write([]byte("<html>502 Bad Gateway</html>")); err != nil {
			t.Errorf("write: %v", err)
		}
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("t")
	_, err := c.Summary(context.Background())
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("want *APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d", apiErr.StatusCode)
	}
	// The HTML body is the only diagnostic available, so it must survive.
	if !strings.Contains(apiErr.Message, "502") {
		t.Fatalf("message lost the body: %q", apiErr.Message)
	}
}

func TestRequestContextIsHonoured(t *testing.T) {
	srv, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		writeJSON(t, w, http.StatusOK, map[string]any{}, nil)
	})

	c := NewClient(srv.URL, srv.Client())
	c.SetToken("t")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := c.Summary(ctx); err == nil {
		t.Fatal("want context deadline error")
	}
}
