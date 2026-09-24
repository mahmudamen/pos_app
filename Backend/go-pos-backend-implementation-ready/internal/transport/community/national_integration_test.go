package community

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
)

// nationalHub reuses hub() but also exposes a saas_admin token minted for the
// same seeded tenant (saas_admin is the bootstrap inviter per docs/24).
// tokenFor mints an access token for a *specific* user (the shared `token`
// helper always impersonates the manager, which cannot model national joins by
// cashier/other-tenant actors).
func tokenFor(t *testing.T, seed testutil.Seed, userID, role string) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, userID, seed.DeviceID, "", role)
}

// resetNational wipes the (deliberately non-tenant-scoped) national tables so
// each test sees only the members it creates. Safe because package tests run
// sequentially against a shared dev database.
func resetNational(t *testing.T) {
	t.Helper()
	if _, err := testutil.Pool(t).Exec(context.Background(),
		"TRUNCATE community_invitations, community_members"); err != nil {
		t.Fatalf("reset national tables: %v", err)
	}
}

func saasToken(t *testing.T, router *gin.Engine, seed testutil.Seed) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, router, http.MethodPost, "/v1/community/invitations", token(t, seed, "saas_admin"),
		`{"note":"launch code","max_uses":10}`)
}

func decodeInvite(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Data struct {
			Code   string `json:"code"`
			Email  string `json:"email"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode invite %q: %v", recorder.Body.String(), err)
	}
	if body.Data.Code == "" || body.Data.Status != "active" {
		t.Fatalf("unexpected invite payload: %+v", body.Data)
	}
	return body.Data.Code
}

func join(t *testing.T, router *gin.Engine, tok, code string) *httptest.ResponseRecorder {
	t.Helper()
	return do(t, router, http.MethodPost, "/v1/community/join", tok, `{"code":"`+code+`"}`)
}

func TestNationalJoinRoundTrip(t *testing.T) {
	resetNational(t)
	router, seed := hub(t)

	// saas_admin bootstraps the first launch code (no membership needed).
	rec := saasToken(t, router, seed)
	if rec.Code != http.StatusCreated {
		t.Fatalf("saas_admin create launch code: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	launch := decodeInvite(t, rec)

	// A non-member manager cannot issue their own invitation yet.
	rec = do(t, router, http.MethodPost, "/v1/community/invitations", token(t, seed, "manager"),
		`{"note":"too soon"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("non-member create invitation: expected 403, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Manager joins via the launch code.
	rec = join(t, router, token(t, seed, "manager"), launch)
	if rec.Code != http.StatusCreated {
		t.Fatalf("manager join: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var member struct {
		Data struct {
			UserID      string `json:"user_id"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
			Status      string `json:"status"`
			Level       string `json:"level"`
			JoinedVia   string `json:"joined_via"`
			Score       int    `json:"expertise_score"`
			InvitedBy   string `json:"invited_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &member); err != nil {
		t.Fatalf("decode join %q: %v", rec.Body.String(), err)
	}
	if member.Data.UserID != seed.ManagerID || member.Data.DisplayName == "" ||
		member.Data.Role != "member" || member.Data.Status != "active" ||
		member.Data.Level != "bronze" || member.Data.Score != 10 ||
		member.Data.JoinedVia != "invitation" {
		t.Fatalf("unexpected member row: %+v", member.Data)
	}

	// Duplicate join -> 409.
	rec = join(t, router, token(t, seed, "manager"), launch)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate join: expected 409, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Manager is now a member and can issue colleague invitations.
	rec = do(t, router, http.MethodPost, "/v1/community/invitations", token(t, seed, "manager"),
		`{"email":"cashier@example.com","note":"join us","max_uses":1}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("manager create invitation: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	code := decodeInvite(t, rec)

	// Cashier joins with the single-use code.
	cashierRec := join(t, router, tokenFor(t, seed, seed.CashierID, "cashier"), code)
	if cashierRec.Code != http.StatusCreated {
		t.Fatalf("cashier join: expected 201, got %d (%s)", cashierRec.Code, cashierRec.Body.String())
	}

	// /me reflects the membership; a third peer joining with the now-used code
	// gets 410 (used up).
	rec = do(t, router, http.MethodGet, "/v1/community/me", tokenFor(t, seed, seed.CashierID, "cashier"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("me: expected 200, got %d", rec.Code)
	}
	other := testutil.SeedTenant(t, testutil.Pool(t))
	rec = join(t, router, tokenFor(t, other, other.ManagerID, "manager"), code)
	if rec.Code != http.StatusGone {
		t.Fatalf("used-up code: expected 410, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Invalid code -> 403.
	rec = join(t, router, token(t, seed, "manager"), "EG-ZZZZZZZZZZZZ")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("invalid code: expected 403, got %d (%s)", rec.Code, rec.Body.String())
	}
}

func TestNationalInvitationLifecycle(t *testing.T) {
	resetNational(t)
	router, seed := hub(t)

	rec := saasToken(t, router, seed)
	launch := decodeInvite(t, rec)
	if rec := join(t, router, token(t, seed, "manager"), launch); rec.Code != http.StatusCreated {
		t.Fatalf("manager join: expected 201, got %d", rec.Code)
	}

	// Expired invitation -> 410.
	rec = do(t, router, http.MethodPost, "/v1/community/invitations", token(t, seed, "manager"),
		`{"expires_at":"2000-01-01T00:00:00Z"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create expired invitation: expected 201, got %d", rec.Code)
	}
	expired := decodeInvite(t, rec)
	rec = join(t, router, tokenFor(t, seed, seed.CashierID, "cashier"), expired)
	if rec.Code != http.StatusGone {
		t.Fatalf("expired code join: expected 410, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Revoked invitation -> 410.
	rec = do(t, router, http.MethodPost, "/v1/community/invitations", token(t, seed, "manager"),
		`{"note":"revoke me"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create revocable invitation: expected 201, got %d", rec.Code)
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &created)
	revokable := decodeInvite(t, rec)
	rec = do(t, router, http.MethodDelete, "/v1/community/invitations/"+created.Data.ID,
		token(t, seed, "manager"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = join(t, router, tokenFor(t, seed, seed.CashierID, "cashier"), revokable)
	if rec.Code != http.StatusGone {
		t.Fatalf("revoked code join: expected 410, got %d (%s)", rec.Code, rec.Body.String())
	}

	// A member from another tenant cannot revoke someone else's invitation.
	other := testutil.SeedTenant(t, testutil.Pool(t))
	if rec := join(t, router, tokenFor(t, other, other.ManagerID, "manager"), launch); rec.Code != http.StatusCreated {
		t.Fatalf("other-tenant manager join: expected 201, got %d", rec.Code)
	}
	rec = do(t, router, http.MethodDelete, "/v1/community/invitations/"+created.Data.ID,
		tokenFor(t, other, other.ManagerID, "manager"), "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant revoke: expected 404, got %d", rec.Code)
	}

	// Inviter's invitation list shows the state.
	rec = do(t, router, http.MethodGet, "/v1/community/invitations", token(t, seed, "manager"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list invitations: expected 200, got %d", rec.Code)
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode invite list %q: %v", rec.Body.String(), err)
	}
	if len(list.Data) == 0 {
		t.Fatal("expected non-empty invitation list")
	}
	revoked := false
	expiredFlag := false
	for _, item := range list.Data {
		if item["status"] == "revoked" {
			revoked = true
		}
		if item["status"] == "active" && item["expires_at"] != "" {
			expiredFlag = true
		}
	}
	if !revoked || !expiredFlag {
		t.Fatalf("expected revoked + expired-in-list statuses, got %+v", list.Data)
	}
}

func TestNationalEgyptGateAndDirectory(t *testing.T) {
	resetNational(t)
	router, seed := hub(t)

	// A foreign tenant cannot join.
	us := testutil.SeedTenant(t, testutil.Pool(t))
	if _, err := testutil.Pool(t).Exec(context.Background(),
		`UPDATE tenants SET country_code = 'US' WHERE id = $1::uuid`, us.TenantID); err != nil {
		t.Fatalf("set country US: %v", err)
	}
	rec := do(t, router, http.MethodPost, "/v1/community/invitations", token(t, seed, "saas_admin"),
		`{"max_uses":5}`)
	code := decodeInvite(t, rec)
	rec = join(t, router, tokenFor(t, us, us.ManagerID, "manager"), code)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("foreign join: expected 403, got %d (%s)", rec.Code, rec.Body.String())
	}
	if body := rec.Body.String(); !contains(body, "egypt_only") {
		t.Fatalf("expected egypt_only code, got %s", body)
	}

	// Two EG members join (same launch code, max_uses=5).
	if rec := join(t, router, token(t, seed, "manager"), code); rec.Code != http.StatusCreated {
		t.Fatalf("seed manager join: expected 201, got %d", rec.Code)
	}
	other := testutil.SeedTenant(t, testutil.Pool(t))
	if rec := join(t, router, tokenFor(t, other, other.ManagerID, "manager"), code); rec.Code != http.StatusCreated {
		t.Fatalf("other manager join: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Directory is cross-tenant: both EG members visible to a member.
	rec = do(t, router, http.MethodGet, "/v1/community/members", token(t, seed, "manager"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list members: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	var list struct {
		Data []map[string]any `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list.Meta.Total != 2 {
		t.Fatalf("expected 2 members, got %+v", list.Data)
	}
	// The directory hides suspended members.
	rec = do(t, router, http.MethodPatch, "/v1/community/members/"+other.ManagerID,
		token(t, seed, "saas_admin"), `{"status":"suspended"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("suspend: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, router, http.MethodGet, "/v1/community/members", token(t, seed, "manager"), "")
	_ = json.Unmarshal(rec.Body.Bytes(), &list)
	if list.Meta.Total != 1 {
		t.Fatalf("expected 1 member after suspension, got %+v", list.Data)
	}
	// Detail still shows the suspended member with their status preserved.
	rec = do(t, router, http.MethodGet, "/v1/community/members/"+other.ManagerID,
		token(t, seed, "manager"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get suspended member: expected 200, got %d", rec.Code)
	}
	var detail map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &detail)
	if detail["data"].(map[string]any)["status"] != "suspended" {
		t.Fatalf("expected suspended status, got %+v", detail["data"])
	}
}

func TestNationalModerationGate(t *testing.T) {
	resetNational(t)
	router, seed := hub(t)

	code := decodeInvite(t, saasToken(t, router, seed))
	if rec := join(t, router, token(t, seed, "manager"), code); rec.Code != http.StatusCreated {
		t.Fatalf("manager join: expected 201, got %d", rec.Code)
	}
	if rec := join(t, router, tokenFor(t, seed, seed.CashierID, "cashier"), code); rec.Code != http.StatusCreated {
		t.Fatalf("cashier join: expected 201, got %d", rec.Code)
	}

	// Plain member cannot moderate.
	rec := do(t, router, http.MethodPatch, "/v1/community/members/"+seed.ManagerID,
		tokenFor(t, seed, seed.CashierID, "cashier"), `{"status":"muted"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("member moderate: expected 403, got %d (%s)", rec.Code, rec.Body.String())
	}

	// saas_admin promotes the manager to moderator.
	rec = do(t, router, http.MethodPatch, "/v1/community/members/"+seed.ManagerID,
		token(t, seed, "saas_admin"), `{"role":"moderator"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("promote: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Moderator can now mute the cashier.
	rec = do(t, router, http.MethodPatch, "/v1/community/members/"+seed.CashierID,
		token(t, seed, "manager"), `{"status":"muted"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("moderator mute: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, router, http.MethodGet, "/v1/community/members/"+seed.CashierID,
		token(t, seed, "manager"), "")
	var detail map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &detail)
	if detail["data"].(map[string]any)["status"] != "muted" {
		t.Fatalf("expected muted status, got %+v", detail["data"])
	}

	// Validation: bad status/role rejected before the pool.
	rec = do(t, router, http.MethodPatch, "/v1/community/members/"+seed.CashierID,
		token(t, seed, "manager"), `{"status":"banned"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad status: expected 400, got %d", rec.Code)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) == 0 || (len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
