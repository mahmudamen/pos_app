package posgo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// pageMeta is the pagination block the API returns alongside a list body. It
// lives in the envelope's meta object rather than beside the rows.
type pageMeta struct {
	Page      int    `json:"page"`
	Limit     int    `json:"limit"`
	Total     int64  `json:"total"`
	RequestID string `json:"request_id"`
}

// envelope is the API's uniform response wrapper. Every /v1 route answers with
// {"data": ...} on success and {"error": {"code","message"}} on failure, so one
// decode path covers the whole control plane.
type envelope struct {
	Data  json.RawMessage `json:"data"`
	Meta  pageMeta        `json:"meta"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// APIError is a control-plane refusal carrying the server's own error code, so
// the view can react to a specific failure (a 403 saas_admin rejection) rather
// than pattern-matching on prose.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
	RequestID  string
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("api %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("api %d %s: %s", e.StatusCode, e.Code, e.Message)
}

// IsAuthFailure reports whether the session is no longer usable and the view
// should return to the login screen.
func (e *APIError) IsAuthFailure() bool { return e.StatusCode == http.StatusUnauthorized }

// Client talks to the POS control plane. It holds a bearer token that Login
// sets; every other method sends it.
//
// The token is mutex-guarded because the console reads it from two directions:
// the UI thread writes it on sign-in, sign-out and an expired session, while the
// resources' fetch goroutines read it to build each request. Signing out
// mid-refresh is normal, so without the guard that pair is a data race.
type Client struct {
	BaseURL string
	HTTP    *http.Client

	mu    sync.RWMutex
	token string
}

// NewClient builds a client for the given base URL. A nil http.Client gets a
// sane timeout so a hung VPS cannot wedge the UI indefinitely.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    httpClient,
	}
}

// Token returns the bearer token currently held, empty before Login.
func (c *Client) Token() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token
}

// SetToken installs a token directly, which lets an operator reuse a session
// exported from the web admin instead of retyping credentials.
func (c *Client) SetToken(token string) {
	c.mu.Lock()
	c.token = strings.TrimSpace(token)
	c.mu.Unlock()
}

// loginRequest is the POST /v1/auth/login body. The tenant_id must be the SaaS
// platform tenant: the control plane rejects any other tenant for a
// saas_admin, since saas.summary deliberately excludes the platform tenant.
type loginRequest struct {
	Email      string `json:"email"`
	Password   string `json:"password"`
	TenantID   string `json:"tenant_id"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

// loginResponse carries the tokens plus the tenant/user echo. Only the fields
// the console displays are decoded.
type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Tenant       struct {
		ID           string `json:"id"`
		BusinessType string `json:"business_type"`
		CountryCode  string `json:"country_code"`
		CurrencyCode string `json:"currency_code"`
	} `json:"tenant"`
	User struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	} `json:"user"`
}

// Session is what Login returns to the view: who signed in, as which role.
type Session struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	TenantID     string
	Currency     string
	Country      string
	UserID       string
	DisplayName  string
	Role         string
}

// IsAdmin reports whether the signed-in user may use the control plane. The
// server requires the saas_admin role exactly; anyone else gets a 403.
func (s Session) IsAdmin() bool { return s.Role == "saas_admin" }

// Login exchanges SaaS staff credentials for a session and stores the token.
func (c *Client) Login(ctx context.Context, email, password, tenantID, deviceID, deviceName string) (Session, error) {
	body := loginRequest{
		Email:      email,
		Password:   password,
		TenantID:   tenantID,
		DeviceID:   deviceID,
		DeviceName: deviceName,
	}
	var out loginResponse
	if err := c.do(ctx, http.MethodPost, "/v1/auth/login", body, &out, false); err != nil {
		return Session{}, err
	}
	c.SetToken(out.AccessToken)
	return Session{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
		ExpiresIn:    out.ExpiresIn,
		TenantID:     out.Tenant.ID,
		Currency:     out.Tenant.CurrencyCode,
		Country:      out.Tenant.CountryCode,
		UserID:       out.User.ID,
		DisplayName:  out.User.DisplayName,
		Role:         out.User.Role,
	}, nil
}

// Summary fetches the platform overview: totals plus the per-business-type
// breakdown that the overview tab renders.
func (c *Client) Summary(ctx context.Context) (Summary, error) {
	var out Summary
	err := c.do(ctx, http.MethodGet, "/v1/saas/summary", nil, &out, true)
	return out, err
}

// Tenants fetches one filtered, paginated page of tenants.
func (c *Client) Tenants(ctx context.Context, f Filters) (TenantPage, error) {
	var rows []Tenant
	path := "/v1/saas/tenants" + f.Query()
	var meta pageMeta
	if err := c.doMeta(ctx, http.MethodGet, path, nil, &rows, &meta, true); err != nil {
		return TenantPage{}, err
	}
	page := TenantPage{Tenants: rows, Page: meta.Page, Limit: meta.Limit, Total: meta.Total}
	// A server that omits meta still yields a usable single page.
	if page.Page < 1 {
		page.Page = 1
	}
	if page.Limit < 1 {
		page.Limit = f.Limit
	}
	return page, nil
}

// Analytics fetches the drill-down for one tenant.
func (c *Client) Analytics(ctx context.Context, tenantID string) (Analytics, error) {
	if strings.TrimSpace(tenantID) == "" {
		return Analytics{}, fmt.Errorf("analytics: empty tenant id")
	}
	var out Analytics
	err := c.do(ctx, http.MethodGet, "/v1/saas/tenants/"+url.PathEscape(tenantID)+"/analytics", nil, &out, true)
	return out, err
}

// Activate re-activates a suspended tenant and audits organization.activated.
func (c *Client) Activate(ctx context.Context, tenantID string) error {
	return c.action(ctx, tenantID, "activate", "")
}

// Suspend flags a tenant, which stops it without disabling it outright.
func (c *Client) Suspend(ctx context.Context, tenantID string) error {
	return c.action(ctx, tenantID, "suspend", "")
}

// Stop disables a tenant: status becomes disabled, subscriptions cancel, and an
// organization.stopped audit row is written. Sign-in then fails with
// organization_stopped until the tenant is activated again.
func (c *Client) Stop(ctx context.Context, tenantID, reason string) error {
	return c.action(ctx, tenantID, "stop", reason)
}

// Backup streams the tenant's RLS-scoped JSON export and returns it verbatim.
// The response is not enveloped, so it is read as raw bytes.
func (c *Client) Backup(ctx context.Context, tenantID string) (string, error) {
	if strings.TrimSpace(tenantID) == "" {
		return "", fmt.Errorf("backup: empty tenant id")
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/v1/platform/tenants/"+url.PathEscape(tenantID)+"/backup", nil, true)
	if err != nil {
		return "", err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", fmt.Errorf("backup: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("backup: read body: %w", err)
	}
	if resp.StatusCode >= 300 {
		return "", decodeAPIError(resp.StatusCode, body)
	}
	return string(body), nil
}

// action posts a platform action that carries no meaningful response body.
func (c *Client) action(ctx context.Context, tenantID, action, reason string) error {
	if strings.TrimSpace(tenantID) == "" {
		return fmt.Errorf("%s: empty tenant id", action)
	}
	path := "/v1/platform/tenants/" + url.PathEscape(tenantID) + "/" + action
	var payload any
	if reason != "" {
		payload = map[string]string{"reason": reason}
	}
	var ignored map[string]any
	return c.do(ctx, http.MethodPost, path, payload, &ignored, true)
}

// do performs a JSON request and decodes the enveloped body into out.
func (c *Client) do(ctx context.Context, method, path string, body, out any, auth bool) error {
	return c.doMeta(ctx, method, path, body, out, nil, auth)
}

// doMeta is do plus an optional capture of the envelope's meta block, needed by
// the paginated tenant list.
func (c *Client) doMeta(ctx context.Context, method, path string, body, out any, meta *pageMeta, auth bool) error {
	req, err := c.newRequest(ctx, method, path, body, auth)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s %s: read body: %w", method, path, err)
	}
	if resp.StatusCode >= 300 {
		return decodeAPIError(resp.StatusCode, raw)
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("%s %s: decode envelope: %w", method, path, err)
	}
	if meta != nil {
		*meta = env.Meta
	}
	if out == nil {
		return nil
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("%s %s: decode data: %w", method, path, err)
	}
	return nil
}

// newRequest builds a request with the bearer token and JSON headers attached.
func (c *Client) newRequest(ctx context.Context, method, path string, body any, auth bool) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("%s %s: encode body: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("%s %s: build request: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth {
		token := c.Token()
		if token == "" {
			return nil, &APIError{StatusCode: http.StatusUnauthorized, Code: "no_session", Message: "not signed in"}
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req, nil
}

// decodeAPIError turns a non-2xx response into an APIError, tolerating a body
// that is not the usual envelope (a proxy 502 HTML page, for instance).
func decodeAPIError(status int, raw []byte) error {
	apiErr := &APIError{StatusCode: status, Message: strings.TrimSpace(string(raw))}
	var env envelope
	if err := json.Unmarshal(raw, &env); err == nil && env.Error != nil {
		apiErr.Code = env.Error.Code
		if env.Error.Message != "" {
			apiErr.Message = env.Error.Message
		}
		apiErr.RequestID = env.Meta.RequestID
	}
	if apiErr.Message == "" {
		apiErr.Message = http.StatusText(status)
	}
	return apiErr
}
