package community

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/example/pos-api/internal/testutil"
)

func TestBadgeAwardRoundTrip(t *testing.T) {
	router, seed := hub(t)

	// Cashier cannot manage the badge catalog (community.write).
	status := do(t, router, http.MethodPost, "/v1/community/badges",
		token(t, seed, "cashier"), `{"code":"top-seller","name":"Top Seller"}`)
	if status.Code != http.StatusForbidden {
		t.Fatalf("cashier create badge: expected 403, got %d", status.Code)
	}

	body := `{"code":"top-seller","name":"Top Seller","description":"Best monthly sales",
		"icon":"star","criteria":{"sales":10}}`
	status = do(t, router, http.MethodPost, "/v1/community/badges", token(t, seed, "manager"), body)
	if status.Code != http.StatusCreated {
		t.Fatalf("create badge: expected 201, got %d (%s)", status.Code, status.Body.String())
	}
	var created struct {
		Data struct {
			ID          string `json:"id"`
			Code        string `json:"code"`
			Icon        string `json:"icon"`
			IsActive    bool   `json:"is_active"`
			EarnedCount int    `json:"earned_count"`
			Criteria    any    `json:"criteria"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Data.ID == "" || created.Data.Code != "top-seller" || !created.Data.IsActive {
		t.Fatalf("unexpected badge: %+v", created.Data)
	}
	// criteria must survive the JSONB round trip as an object, not base64.
	if criteria, ok := created.Data.Criteria.(map[string]any); !ok || criteria["sales"] != float64(10) {
		t.Fatalf("criteria round trip: got %#v", created.Data.Criteria)
	}

	// Duplicate code → 409.
	status = do(t, router, http.MethodPost, "/v1/community/badges", token(t, seed, "manager"),
		`{"code":"top-seller","name":"Again"}`)
	if status.Code != http.StatusConflict {
		t.Fatalf("duplicate badge code: expected 409, got %d (%s)", status.Code, status.Body.String())
	}
	// Invalid code → 400.
	status = do(t, router, http.MethodPost, "/v1/community/badges", token(t, seed, "manager"),
		`{"code":"Top Seller","name":"Nope"}`)
	if status.Code != http.StatusBadRequest {
		t.Fatalf("invalid badge code: expected 400, got %d (%s)", status.Code, status.Body.String())
	}

	// List: cashier can read, earned_count starts at 0.
	status = do(t, router, http.MethodGet, "/v1/community/badges", token(t, seed, "cashier"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("list badges: expected 200, got %d", status.Code)
	}
	var list struct {
		Data []struct {
			ID          string `json:"id"`
			EarnedCount int    `json:"earned_count"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list %q: %v", status.Body.String(), err)
	}
	if len(list.Data) != 1 || list.Data[0].ID != created.Data.ID || list.Data[0].EarnedCount != 0 {
		t.Fatalf("expected one badge with 0 awards, got %+v", list.Data)
	}

	// Award: cashier cannot, manager can.
	status = do(t, router, http.MethodPost, "/v1/community/profiles/"+seed.CashierID+"/badges",
		token(t, seed, "cashier"), `{"badge_id":"`+created.Data.ID+`"}`)
	if status.Code != http.StatusForbidden {
		t.Fatalf("cashier award: expected 403, got %d", status.Code)
	}
	status = do(t, router, http.MethodPost, "/v1/community/profiles/"+seed.CashierID+"/badges",
		token(t, seed, "manager"), `{"badge_id":"`+created.Data.ID+`","note":"10 sales in September"}`)
	if status.Code != http.StatusCreated {
		t.Fatalf("award: expected 201, got %d (%s)", status.Code, status.Body.String())
	}
	var awarded struct {
		Data struct {
			BadgeID       string `json:"badge_id"`
			Code          string `json:"code"`
			Note          string `json:"note"`
			AwardedByName string `json:"awarded_by_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &awarded); err != nil {
		t.Fatalf("decode award: %v", err)
	}
	if awarded.Data.Code != "top-seller" || awarded.Data.Note != "10 sales in September" {
		t.Fatalf("unexpected award: %+v", awarded.Data)
	}

	// Duplicate award → 409; unknown user → 404; unknown badge → 404.
	status = do(t, router, http.MethodPost, "/v1/community/profiles/"+seed.CashierID+"/badges",
		token(t, seed, "manager"), `{"badge_id":"`+created.Data.ID+`"}`)
	if status.Code != http.StatusConflict {
		t.Fatalf("duplicate award: expected 409, got %d (%s)", status.Code, status.Body.String())
	}
	status = do(t, router, http.MethodPost, "/v1/community/profiles/11111111-1111-1111-1111-111111111111/badges",
		token(t, seed, "manager"), `{"badge_id":"`+created.Data.ID+`"}`)
	if status.Code != http.StatusNotFound {
		t.Fatalf("award unknown user: expected 404, got %d (%s)", status.Code, status.Body.String())
	}
	status = do(t, router, http.MethodPost, "/v1/community/profiles/"+seed.CashierID+"/badges",
		token(t, seed, "manager"), `{"badge_id":"22222222-2222-2222-2222-222222222222"}`)
	if status.Code != http.StatusNotFound {
		t.Fatalf("award unknown badge: expected 404, got %d (%s)", status.Code, status.Body.String())
	}

	// Profile badge list: earned_count reflected, cashier can read it.
	status = do(t, router, http.MethodGet, "/v1/community/profiles/"+seed.CashierID+"/badges",
		token(t, seed, "cashier"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("list profile badges: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	var profileBadges struct {
		Data []struct {
			Code          string `json:"code"`
			AwardedByName string `json:"awarded_by_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &profileBadges); err != nil {
		t.Fatalf("decode profile badges: %v", err)
	}
	if len(profileBadges.Data) != 1 || profileBadges.Data[0].Code != "top-seller" {
		t.Fatalf("expected the awarded badge, got %+v", profileBadges.Data)
	}
	// Unknown profile → 404.
	status = do(t, router, http.MethodGet, "/v1/community/profiles/11111111-1111-1111-1111-111111111111/badges",
		token(t, seed, "manager"), "")
	if status.Code != http.StatusNotFound {
		t.Fatalf("unknown profile badges: expected 404, got %d", status.Code)
	}

	// Revoke: cashier cannot, manager can, second revoke → 404.
	status = do(t, router, http.MethodDelete,
		"/v1/community/profiles/"+seed.CashierID+"/badges/"+created.Data.ID, token(t, seed, "cashier"), "")
	if status.Code != http.StatusForbidden {
		t.Fatalf("cashier revoke: expected 403, got %d", status.Code)
	}
	status = do(t, router, http.MethodDelete,
		"/v1/community/profiles/"+seed.CashierID+"/badges/"+created.Data.ID, token(t, seed, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("revoke: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	status = do(t, router, http.MethodDelete,
		"/v1/community/profiles/"+seed.CashierID+"/badges/"+created.Data.ID, token(t, seed, "manager"), "")
	if status.Code != http.StatusNotFound {
		t.Fatalf("revoke again: expected 404, got %d (%s)", status.Code, status.Body.String())
	}

	// Deactivate: hidden from the default list, visible with ?all=1.
	status = do(t, router, http.MethodPatch, "/v1/community/badges/"+created.Data.ID,
		token(t, seed, "manager"), `{"is_active":false,"name":"Top Seller (retired)"}`)
	if status.Code != http.StatusOK {
		t.Fatalf("patch badge: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	status = do(t, router, http.MethodGet, "/v1/community/badges", token(t, seed, "manager"), "")
	_ = json.Unmarshal(status.Body.Bytes(), &list)
	if len(list.Data) != 0 {
		t.Fatalf("inactive badge hidden by default, got %+v", list.Data)
	}
	status = do(t, router, http.MethodGet, "/v1/community/badges?all=1", token(t, seed, "manager"), "")
	_ = json.Unmarshal(status.Body.Bytes(), &list)
	if len(list.Data) != 1 {
		t.Fatalf("?all=1 should include inactive badges, got %+v", list.Data)
	}

	// Cross-tenant isolation: another tenant sees an empty catalog and cannot
	// award our badges or reach our users.
	other := testutil.SeedTenant(t, testutil.Pool(t))
	status = do(t, router, http.MethodGet, "/v1/community/badges", token(t, other, "manager"), "")
	var otherList struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(status.Body.Bytes(), &otherList)
	if len(otherList.Data) != 0 {
		t.Fatalf("other tenant sees %d badges, want 0", len(otherList.Data))
	}
	status = do(t, router, http.MethodPost, "/v1/community/profiles/"+seed.CashierID+"/badges",
		token(t, other, "manager"), `{"badge_id":"`+created.Data.ID+`"}`)
	if status.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant award: expected 404, got %d (%s)", status.Code, status.Body.String())
	}
}

func TestEndorsementRoundTrip(t *testing.T) {
	router, seed := hub(t)
	manager := token(t, seed, "manager") // user = seed.ManagerID
	cashierUser := testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "", "cashier")

	// Manager endorses the cashier (skill is normalized to lowercase).
	status := do(t, router, http.MethodPost, "/v1/community/endorsements", manager,
		`{"receiver_id":"`+seed.CashierID+`","skill":"Grill","note":"Perfect sear"}`)
	if status.Code != http.StatusCreated {
		t.Fatalf("endorse: expected 201, got %d (%s)", status.Code, status.Body.String())
	}
	var endorsed struct {
		Data struct {
			ID    string `json:"id"`
			Skill string `json:"skill"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &endorsed); err != nil {
		t.Fatalf("decode endorse: %v", err)
	}
	if endorsed.Data.ID == "" || endorsed.Data.Skill != "grill" {
		t.Fatalf("unexpected endorsement: %+v", endorsed.Data)
	}

	// Duplicate → 409; self → 400.
	status = do(t, router, http.MethodPost, "/v1/community/endorsements", manager,
		`{"receiver_id":"`+seed.CashierID+`","skill":"grill"}`)
	if status.Code != http.StatusConflict {
		t.Fatalf("duplicate endorse: expected 409, got %d (%s)", status.Code, status.Body.String())
	}
	status = do(t, router, http.MethodPost, "/v1/community/endorsements", manager,
		`{"receiver_id":"`+seed.ManagerID+`","skill":"grill"}`)
	if status.Code != http.StatusBadRequest {
		t.Fatalf("self endorse: expected 400, got %d (%s)", status.Code, status.Body.String())
	}

	// Cashier endorses the manager back.
	status = do(t, router, http.MethodPost, "/v1/community/endorsements", cashierUser,
		`{"receiver_id":"`+seed.ManagerID+`","skill":"Speed","note":"Fastest line cook"}`)
	if status.Code != http.StatusCreated {
		t.Fatalf("cashier endorse: expected 201, got %d (%s)", status.Code, status.Body.String())
	}

	type group struct {
		Skill  string `json:"skill"`
		Count  int    `json:"count"`
		Givers []struct {
			ID        string `json:"id"`
			GiverID   string `json:"giver_id"`
			GiverName string `json:"giver_name"`
		} `json:"givers"`
		MyEndorsementID string `json:"my_endorsement_id"`
	}

	// Manager's own profile: one endorsement from the cashier, not their own.
	status = do(t, router, http.MethodGet, "/v1/community/profiles/"+seed.ManagerID+"/endorsements", manager, "")
	if status.Code != http.StatusOK {
		t.Fatalf("list manager endorsements: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	var managerResp struct {
		Data []group `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &managerResp); err != nil {
		t.Fatalf("decode groups: %v", err)
	}
	managerGroups := managerResp.Data
	if len(managerGroups) != 1 || managerGroups[0].Skill != "speed" || managerGroups[0].Count != 1 {
		t.Fatalf("manager groups: %+v", managerGroups)
	}
	if managerGroups[0].MyEndorsementID != "" || managerGroups[0].Givers[0].GiverID != seed.CashierID ||
		managerGroups[0].Givers[0].GiverName == "" {
		t.Fatalf("manager group detail: %+v", managerGroups[0])
	}

	// Cashier's profile as seen by the manager: my_endorsement_id points at
	// the manager's own endorsement.
	status = do(t, router, http.MethodGet, "/v1/community/profiles/"+seed.CashierID+"/endorsements", manager, "")
	var cashierResp struct {
		Data []group `json:"data"`
	}
	_ = json.Unmarshal(status.Body.Bytes(), &cashierResp)
	cashierGroups := cashierResp.Data
	if len(cashierGroups) != 1 || cashierGroups[0].MyEndorsementID != endorsed.Data.ID {
		t.Fatalf("cashier groups: %+v", cashierGroups)
	}

	// Withdraw: the cashier cannot delete someone else's; the manager can
	// moderate; the giver can always withdraw their own.
	status = do(t, router, http.MethodDelete, "/v1/community/endorsements/"+endorsed.Data.ID, cashierUser, "")
	if status.Code != http.StatusForbidden {
		t.Fatalf("cashier delete others': expected 403, got %d", status.Code)
	}
	status = do(t, router, http.MethodDelete, "/v1/community/endorsements/22222222-2222-2222-2222-222222222222", manager, "")
	if status.Code != http.StatusNotFound {
		t.Fatalf("delete unknown endorsement: expected 404, got %d", status.Code)
	}
	status = do(t, router, http.MethodDelete, "/v1/community/endorsements/"+endorsed.Data.ID, manager, "")
	if status.Code != http.StatusOK {
		t.Fatalf("manager moderate delete: expected 200, got %d (%s)", status.Code, status.Body.String())
	}

	// Cashier profile now has no endorsements left.
	status = do(t, router, http.MethodGet, "/v1/community/profiles/"+seed.CashierID+"/endorsements", manager, "")
	var after struct {
		Data []group `json:"data"`
	}
	_ = json.Unmarshal(status.Body.Bytes(), &after)
	if len(after.Data) != 0 {
		t.Fatalf("expected no endorsements after withdraw, got %+v", after.Data)
	}

	// Unknown receiver → 404.
	status = do(t, router, http.MethodGet, "/v1/community/profiles/11111111-1111-1111-1111-111111111111/endorsements", manager, "")
	if status.Code != http.StatusNotFound {
		t.Fatalf("unknown receiver: expected 404, got %d", status.Code)
	}

	// Cross-tenant: another tenant cannot see or reach our users.
	other := testutil.SeedTenant(t, testutil.Pool(t))
	status = do(t, router, http.MethodGet, "/v1/community/profiles/"+seed.ManagerID+"/endorsements",
		token(t, other, "manager"), "")
	if status.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant list endorsements: expected 404, got %d", status.Code)
	}
	status = do(t, router, http.MethodPost, "/v1/community/endorsements", token(t, other, "manager"),
		`{"receiver_id":"`+seed.CashierID+`","skill":"grill"}`)
	if status.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant endorse: expected 404, got %d (%s)", status.Code, status.Body.String())
	}
}
