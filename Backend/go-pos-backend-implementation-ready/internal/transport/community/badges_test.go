package community

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func badgeRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	return router
}

func TestBadgesRequireToken(t *testing.T) {
	router := badgeRouter(t)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/community/badges", ""},
		{http.MethodPost, "/v1/community/badges", `{"code":"top-seller","name":"Top Seller"}`},
		{http.MethodPost, "/v1/community/endorsements", `{"receiver_id":"00000000-0000-0000-0000-000000000002","skill":"grill"}`},
		{http.MethodGet, "/v1/community/profiles/00000000-0000-0000-0000-000000000002/badges", ""},
		{http.MethodGet, "/v1/community/profiles/00000000-0000-0000-0000-000000000002/endorsements", ""},
		{http.MethodDelete, "/v1/community/endorsements/00000000-0000-0000-0000-000000000009", ""},
	} {
		recorder := doRequest(t, router, tc.method, tc.path, tc.body, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: expected 401, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestBadgesReadBlockedForGuest(t *testing.T) {
	router := badgeRouter(t)
	recorder := doRequest(t, router, http.MethodGet, "/v1/community/badges", "", guestToken(t))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("guest list badges: expected 403, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodPost, "/v1/community/endorsements",
		`{"receiver_id":"00000000-0000-0000-0000-000000000002","skill":"grill"}`, guestToken(t))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("guest endorse: expected 403, got %d", recorder.Code)
	}
}

func TestBadgesWriteBlockedForCashier(t *testing.T) {
	router := badgeRouter(t)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/v1/community/badges", `{"code":"top-seller","name":"Top Seller"}`},
		{http.MethodPatch, "/v1/community/badges/00000000-0000-0000-0000-000000000009", `{"name":"Renamed"}`},
		{http.MethodPost, "/v1/community/profiles/00000000-0000-0000-0000-000000000002/badges",
			`{"badge_id":"00000000-0000-0000-0000-000000000009"}`},
		{http.MethodDelete, "/v1/community/profiles/00000000-0000-0000-0000-000000000002/badges/00000000-0000-0000-0000-000000000009", ""},
	} {
		recorder := doRequest(t, router, tc.method, tc.path, tc.body, cashierToken(t))
		if recorder.Code != http.StatusForbidden {
			t.Errorf("cashier %s %s: expected 403, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestBadgesUnavailableWithoutDatabase(t *testing.T) {
	router := badgeRouter(t)
	recorder := doRequest(t, router, http.MethodGet, "/v1/community/badges", "", managerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("list badges without db: expected 503, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodGet,
		"/v1/community/profiles/00000000-0000-0000-0000-000000000002/endorsements", "", managerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("list endorsements without db: expected 503, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodPost, "/v1/community/endorsements",
		`{"receiver_id":"00000000-0000-0000-0000-000000000003","skill":"grill"}`, managerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("create endorsement without db: expected 503, got %d", recorder.Code)
	}
}

func TestCreateBadgeValidation(t *testing.T) {
	router := badgeRouter(t)
	for _, tc := range []struct {
		name string
		body string
	}{
		{"missing name", `{"code":"top-seller"}`},
		{"missing code", `{"name":"Top Seller"}`},
		{"invalid code spacing", `{"code":"Top Seller","name":"Top Seller"}`},
		{"code uppercase", `{"code":"TopSeller","name":"Top Seller"}`},
		{"code leading dash", `{"code":"-top","name":"Top"}`},
		{"code too short", `{"code":"a","name":"Top"}`},
		{"name blank", `{"code":"top-seller","name":"   "}`},
		{"name too long", `{"code":"top-seller","name":"` + strings.Repeat("x", 101) + `"}`},
		{"description too long", `{"code":"top-seller","name":"Top","description":"` + strings.Repeat("x", 501) + `"}`},
		{"icon too long", `{"code":"top-seller","name":"Top","icon":"` + strings.Repeat("x", 61) + `"}`},
		{"bad json", `{{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := doRequest(t, router, http.MethodPost, "/v1/community/badges", tc.body, managerToken(t))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (%s)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestUpdateBadgeValidation(t *testing.T) {
	router := badgeRouter(t)
	recorder := doRequest(t, router, http.MethodPatch, "/v1/community/badges/not-a-uuid", `{"name":"X"}`, managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid badge id: expected 400, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodPatch, "/v1/community/badges/not-a-uuid", `{{`, managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid json: expected 400, got %d", recorder.Code)
	}
}

func TestAwardBadgeValidation(t *testing.T) {
	router := badgeRouter(t)
	for _, tc := range []struct{ path, body string }{
		{"/v1/community/profiles/not-a-uuid/badges", `{"badge_id":"00000000-0000-0000-0000-000000000009"}`},
		{"/v1/community/profiles/00000000-0000-0000-0000-000000000002/badges", `{}`},
		{"/v1/community/profiles/00000000-0000-0000-0000-000000000002/badges", `{"badge_id":"nope"}`},
		{"/v1/community/profiles/00000000-0000-0000-0000-000000000002/badges",
			`{"badge_id":"00000000-0000-0000-0000-000000000009","note":"` + strings.Repeat("x", 501) + `"}`},
	} {
		recorder := doRequest(t, router, http.MethodPost, tc.path, tc.body, managerToken(t))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("award %s %s: expected 400, got %d (%s)", tc.path, tc.body, recorder.Code, recorder.Body.String())
		}
	}
	recorder := doRequest(t, router, http.MethodDelete,
		"/v1/community/profiles/00000000-0000-0000-0000-000000000002/badges/not-a-uuid", "", managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("revoke invalid badge id: expected 400, got %d", recorder.Code)
	}
}

func TestEndorsementValidation(t *testing.T) {
	router := badgeRouter(t)
	selfID := "00000000-0000-0000-0000-000000000002" // tokenWithRole user id
	for _, tc := range []struct{ name, body string }{
		{"missing receiver", `{"skill":"grill"}`},
		{"missing skill", `{"receiver_id":"` + selfID + `"}`},
		{"whitespace skill", `{"receiver_id":"` + selfID + `","skill":"   "}`},
		{"skill too long", `{"receiver_id":"` + selfID + `","skill":"` + strings.Repeat("x", 61) + `"}`},
		{"note too long", `{"receiver_id":"` + selfID + `","skill":"grill","note":"` + strings.Repeat("x", 501) + `"}`},
		{"invalid receiver", `{"receiver_id":"nope","skill":"grill"}`},
		{"self endorsement", `{"receiver_id":"` + selfID + `","skill":"grill"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := doRequest(t, router, http.MethodPost, "/v1/community/endorsements", tc.body, managerToken(t))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (%s)", recorder.Code, recorder.Body.String())
			}
		})
	}

	recorder := doRequest(t, router, http.MethodDelete, "/v1/community/endorsements/not-a-uuid", "", managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid endorsement id: expected 400, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodGet,
		"/v1/community/profiles/not-a-uuid/endorsements", "", managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid receiver id: expected 400, got %d", recorder.Code)
	}
}

func TestValidateBadgeCode(t *testing.T) {
	for _, tc := range []struct {
		code string
		want bool
	}{
		{"top-seller", true},
		{"vip", true},
		{"a1-b2", true},
		{"", false},
		{"a", false},
		{"TopSeller", false},
		{"-top", false},
		{"top-", false},
		{"top seller", false},
		{"töp", false},
		{strings.Repeat("x", 61), false},
		{strings.Repeat("x", 60), true},
	} {
		if got := validateBadgeCode(tc.code); got != tc.want {
			t.Errorf("validateBadgeCode(%q) = %v, want %v", tc.code, got, tc.want)
		}
	}
}

func TestNormalizeEndorsementSkill(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{" Grill ", "grill"},
		{"Sous-Vide", "sous-vide"},
		{"  ", ""},
		{"", ""},
	} {
		if got := normalizeEndorsementSkill(tc.in); got != tc.want {
			t.Errorf("normalizeEndorsementSkill(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
