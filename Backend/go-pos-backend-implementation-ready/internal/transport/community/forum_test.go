package community

import (
	"net/http"
	"strings"
	"testing"
)

// Forum unit suite: routes require a token, read rides community.read (guests
// are out), posting/commenting/liking ride the forum.write grant (every role,
// guest included), and every validation path answers before the pool check.

func TestForumRoutesRequireToken(t *testing.T) {
	router := badgeRouter(t)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/community/forum/categories", ""},
		{http.MethodPost, "/v1/community/forum/categories", `{"name":"Kitchen Talk"}`},
		{http.MethodPatch, "/v1/community/forum/categories/00000000-0000-0000-0000-000000000009", `{"name":"Renamed"}`},
		{http.MethodGet, "/v1/community/forum/posts", ""},
		{http.MethodPost, "/v1/community/forum/posts", `{"category_id":"00000000-0000-0000-0000-000000000009","title":"Hi"}`},
		{http.MethodGet, "/v1/community/forum/posts/00000000-0000-0000-0000-000000000009", ""},
		{http.MethodPatch, "/v1/community/forum/posts/00000000-0000-0000-0000-000000000009", `{"title":"New"}`},
		{http.MethodDelete, "/v1/community/forum/posts/00000000-0000-0000-0000-000000000009", ""},
		{http.MethodPost, "/v1/community/forum/posts/00000000-0000-0000-0000-000000000009/comments", `{"body":"Nice"}`},
		{http.MethodPost, "/v1/community/forum/posts/00000000-0000-0000-0000-000000000009/like", ""},
	} {
		recorder := doRequest(t, router, tc.method, tc.path, tc.body, "")
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without token: expected 401, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestForumReadBlockedForGuest(t *testing.T) {
	router := badgeRouter(t)
	for _, path := range []string{
		"/v1/community/forum/categories",
		"/v1/community/forum/posts",
		"/v1/community/forum/posts/00000000-0000-0000-0000-000000000009",
	} {
		recorder := doRequest(t, router, http.MethodGet, path, "", guestToken(t))
		if recorder.Code != http.StatusForbidden {
			t.Errorf("guest GET %s: expected 403, got %d", path, recorder.Code)
		}
	}
}

// The forum.write grant covers every role including guest — a request that
// clears RBAC with no database behind it must land on 503, not 403.
func TestForumWriteGrantedForEveryRole(t *testing.T) {
	router := badgeRouter(t)
	post := `{"category_id":"00000000-0000-0000-0000-000000000009","title":"Hello","body":"World"}`
	comment := `{"body":"Agreed"}`
	for _, tc := range []struct{ name, token, method, path, body string }{
		{"cashier post", cashierToken(t), http.MethodPost, "/v1/community/forum/posts", post},
		{"guest post", guestToken(t), http.MethodPost, "/v1/community/forum/posts", post},
		{"cashier comment", cashierToken(t), http.MethodPost,
			"/v1/community/forum/posts/00000000-0000-0000-0000-000000000009/comments", comment},
		{"guest like", guestToken(t), http.MethodPost,
			"/v1/community/forum/posts/00000000-0000-0000-0000-000000000009/like", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := doRequest(t, router, tc.method, tc.path, tc.body, tc.token)
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("expected 503 (RBAC passed, no db), got %d (%s)",
					recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestForumUnavailableWithoutDatabase(t *testing.T) {
	router := badgeRouter(t)
	validID := "00000000-0000-0000-0000-000000000009"
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodGet, "/v1/community/forum/categories", ""},
		{http.MethodGet, "/v1/community/forum/posts", ""},
		{http.MethodGet, "/v1/community/forum/posts/" + validID, ""},
		{http.MethodPost, "/v1/community/forum/categories", `{"name":"Kitchen Talk"}`},
		{http.MethodPatch, "/v1/community/forum/categories/" + validID, `{"name":"Renamed"}`},
		{http.MethodPost, "/v1/community/forum/posts",
			`{"category_id":"` + validID + `","title":"Hello"}`},
		{http.MethodPatch, "/v1/community/forum/posts/" + validID, `{"title":"New"}`},
		{http.MethodDelete, "/v1/community/forum/posts/" + validID, ""},
		{http.MethodPost, "/v1/community/forum/posts/" + validID + "/comments", `{"body":"Nice"}`},
		{http.MethodPost, "/v1/community/forum/posts/" + validID + "/like", ""},
	} {
		recorder := doRequest(t, router, tc.method, tc.path, tc.body, managerToken(t))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Errorf("%s %s without db: expected 503, got %d (%s)",
				tc.method, tc.path, recorder.Code, recorder.Body.String())
		}
	}
}

// Category management is community.write — a cashier cannot create boards
// even though forum.write lets them post inside one.
func TestForumCategoryWriteBlockedForCashier(t *testing.T) {
	router := badgeRouter(t)
	for _, tc := range []struct{ method, path, body string }{
		{http.MethodPost, "/v1/community/forum/categories", `{"name":"Kitchen Talk"}`},
		{http.MethodPatch, "/v1/community/forum/categories/00000000-0000-0000-0000-000000000009", `{"name":"Renamed"}`},
	} {
		recorder := doRequest(t, router, tc.method, tc.path, tc.body, cashierToken(t))
		if recorder.Code != http.StatusForbidden {
			t.Errorf("cashier %s %s: expected 403, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestCreateForumCategoryValidation(t *testing.T) {
	router := badgeRouter(t)
	longName := strings.Repeat("a", 101)
	longDescription := strings.Repeat("d", 501)
	for _, tc := range []struct{ name, body string }{
		{"missing name", `{"description":"Boards"}`},
		{"blank name", `{"name":"   "}`},
		{"name too long", `{"name":"` + longName + `"}`},
		{"description too long", `{"name":"Boards","description":"` + longDescription + `"}`},
		{"bad slug", `{"name":"Boards","slug":"Kitchen Talk"}`},
		{"bad slug uppercase", `{"name":"Boards","slug":"Kitchen-Talk"}`},
		{"bad slug leading dash", `{"name":"Boards","slug":"-boards"}`},
		{"invalid json", `{{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := doRequest(t, router, http.MethodPost, "/v1/community/forum/categories",
				tc.body, managerToken(t))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (%s)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestUpdateForumCategoryValidation(t *testing.T) {
	router := badgeRouter(t)
	recorder := doRequest(t, router, http.MethodPatch,
		"/v1/community/forum/categories/not-a-uuid", `{"name":"X"}`, managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid category id: expected 400, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodPatch,
		"/v1/community/forum/categories/00000000-0000-0000-0000-000000000009", `{{`, managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid json: expected 400, got %d", recorder.Code)
	}
}

func TestCreateForumPostValidation(t *testing.T) {
	router := badgeRouter(t)
	longTitle := strings.Repeat("t", 201)
	longBody := strings.Repeat("b", 10001)
	for _, tc := range []struct{ name, body string }{
		{"missing category", `{"title":"Hello"}`},
		{"missing title", `{"category_id":"00000000-0000-0000-0000-000000000009"}`},
		{"bad category id", `{"category_id":"nope","title":"Hello"}`},
		{"title too long", `{"category_id":"00000000-0000-0000-0000-000000000009","title":"` + longTitle + `"}`},
		{"blank title", `{"category_id":"00000000-0000-0000-0000-000000000009","title":"   "}`},
		{"body too long", `{"category_id":"00000000-0000-0000-0000-000000000009","title":"Hi","body":"` + longBody + `"}`},
		{"invalid json", `{{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := doRequest(t, router, http.MethodPost, "/v1/community/forum/posts",
				tc.body, cashierToken(t))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (%s)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestListForumPostsQueryValidation(t *testing.T) {
	router := badgeRouter(t)
	// sort/category_id are checked before the pool, so they answer 400 here.
	recorder := doRequest(t, router, http.MethodGet, "/v1/community/forum/posts?sort=newest",
		"", managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad sort: expected 400, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodGet,
		"/v1/community/forum/posts?category_id=not-a-uuid", "", managerToken(t))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("bad category id: expected 400, got %d", recorder.Code)
	}
	recorder = doRequest(t, router, http.MethodGet,
		"/v1/community/forum/posts?sort=top", "", managerToken(t))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("valid sort without db: expected 503, got %d", recorder.Code)
	}
}

func TestForumPostIdValidation(t *testing.T) {
	router := badgeRouter(t)
	for _, tc := range []struct{ method, path, body, token string }{
		{http.MethodGet, "/v1/community/forum/posts/not-a-uuid", "", managerToken(t)},
		{http.MethodPatch, "/v1/community/forum/posts/not-a-uuid", `{"title":"X"}`, managerToken(t)},
		{http.MethodDelete, "/v1/community/forum/posts/not-a-uuid", "", managerToken(t)},
		{http.MethodPost, "/v1/community/forum/posts/not-a-uuid/comments", `{"body":"Hi"}`, guestToken(t)},
		{http.MethodPost, "/v1/community/forum/posts/not-a-uuid/like", "", guestToken(t)},
	} {
		recorder := doRequest(t, router, tc.method, tc.path, tc.body, tc.token)
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("%s %s: expected 400, got %d", tc.method, tc.path, recorder.Code)
		}
	}
}

func TestCreateForumCommentValidation(t *testing.T) {
	router := badgeRouter(t)
	post := "00000000-0000-0000-0000-000000000009"
	longBody := strings.Repeat("c", 5001)
	for _, tc := range []struct{ name, body string }{
		{"missing body", `{}`},
		{"blank body", `{"body":"   "}`},
		{"body too long", `{"body":"` + longBody + `"}`},
		{"invalid json", `{{`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			recorder := doRequest(t, router, http.MethodPost,
				"/v1/community/forum/posts/"+post+"/comments", tc.body, guestToken(t))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d (%s)", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestForumSlug(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"Kitchen Talk", "kitchen-talk"},
		{"  Grilled & Chilled  ", "grilled-chilled"},
		{"A", "a"},
		{"---", "board"},
		{"", "board"},
		{"Café Corner", "caf-corner"},
		{"--dash--edges--", "dash-edges"},
	} {
		if got := forumSlug(tc.in); got != tc.want {
			t.Errorf("forumSlug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestForumSlugPattern(t *testing.T) {
	for _, valid := range []string{"a", "boards", "kitchen-talk", "pos2026", strings.Repeat("x", 100)} {
		if !forumSlugPattern.MatchString(valid) {
			t.Errorf("slug %q should match", valid)
		}
	}
	for _, invalid := range []string{
		"", "-boards", "boards-", "Kitchen", "kitchen talk", "kitchen--talk_", "ünïcode",
		strings.Repeat("x", 101),
	} {
		if forumSlugPattern.MatchString(invalid) {
			t.Errorf("slug %q should not match", invalid)
		}
	}
}
