package community

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/example/pos-api/internal/testutil"
)

// Forum DB-backed round trip (requires TEST_DATABASE_URL/DATABASE_URL):
// board creation, listing with aggregates, posting as a cashier, comments,
// like toggling, author-vs-moderator edit rules, pin/close, closed-thread
// rejection, hard delete, and cross-tenant isolation under FORCE RLS.
func TestForumRoundTrip(t *testing.T) {
	router, seed := hub(t)
	cashier := testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "", "cashier")

	// Manager creates a board (cashier cannot manage boards).
	status := do(t, router, http.MethodPost, "/v1/community/forum/categories",
		cashier, `{"name":"Kitchen Talk"}`)
	if status.Code != http.StatusForbidden {
		t.Fatalf("cashier create category: expected 403, got %d", status.Code)
	}
	rec := do(t, router, http.MethodPost, "/v1/community/forum/categories",
		token(t, seed, "manager"), `{"name":"Kitchen Talk","description":"Boards for the floor team"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var created struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode category: %v", err)
	}
	if created.Data.Slug != "kitchen-talk" {
		t.Fatalf("auto slug = %q, want kitchen-talk", created.Data.Slug)
	}
	categoryID := created.Data.ID

	// Duplicate name -> 409.
	rec = do(t, router, http.MethodPost, "/v1/community/forum/categories",
		token(t, seed, "manager"), `{"name":"Kitchen Talk"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate category: expected 409, got %d", rec.Code)
	}

	// Cashier lists boards and sees the post count.
	rec = do(t, router, http.MethodGet, "/v1/community/forum/categories", cashier, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("list categories: expected 200, got %d", rec.Code)
	}
	var list struct {
		Data []ForumCategory `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Meta.Total != 1 || len(list.Data) != 1 {
		t.Fatalf("category list total = %d/%d, want 1", list.Meta.Total, len(list.Data))
	}
	if list.Data[0].PostCount != 0 || !list.Data[0].IsActive {
		t.Fatalf("category = %+v, want active with 0 posts", list.Data[0])
	}

	// Cashier starts a thread (forum.write).
	rec = do(t, router, http.MethodPost, "/v1/community/forum/posts", cashier,
		`{"category_id":"`+categoryID+`","title":"Grill station tips","body":"Preheat before service."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("cashier create post: expected 201, got %d (%s)", rec.Code, rec.Body.String())
	}
	var post struct {
		Data ForumPost `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("decode post: %v", err)
	}
	if post.Data.AuthorID != seed.CashierID || !post.Data.IsAuthor ||
		post.Data.CategoryName != "Kitchen Talk" {
		t.Fatalf("post echo = %+v", post.Data)
	}
	postID := post.Data.ID

	// Manager opens the thread: empty comments, zero counts, not the author.
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts/"+postID,
		token(t, seed, "manager"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("get post: expected 200, got %d", rec.Code)
	}
	var detail struct {
		Data     ForumPost      `json:"data"`
		Comments []ForumComment `json:"comments"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode detail: %v", err)
	}
	if len(detail.Comments) != 0 || detail.Data.CommentCount != 0 ||
		detail.Data.LikeCount != 0 || detail.Data.IsAuthor || detail.Data.LikedByMe {
		t.Fatalf("detail = %+v comments=%d", detail.Data, len(detail.Comments))
	}

	// Two replies land; the board now shows a post count of 1.
	for i, body := range []string{`{"body":"Agreed, rest the grates."}`, `{"body":"Adding a checklist."}`} {
		rec = do(t, router, http.MethodPost, "/v1/community/forum/posts/"+postID+"/comments",
			token(t, seed, "manager"), body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("comment %d: expected 201, got %d (%s)", i, rec.Code, rec.Body.String())
		}
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/categories", cashier, "")
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Data[0].PostCount != 1 {
		t.Fatalf("post_count = %d, want 1", list.Data[0].PostCount)
	}

	// Like toggles: on -> off -> on, with the count following.
	toggle := func(token string, wantLiked bool, wantCount int) {
		t.Helper()
		rec := do(t, router, http.MethodPost, "/v1/community/forum/posts/"+postID+"/like", token, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("toggle like: expected 200, got %d (%s)", rec.Code, rec.Body.String())
		}
		var payload struct {
			Data struct {
				Liked     bool `json:"liked"`
				LikeCount int  `json:"like_count"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode like: %v", err)
		}
		if payload.Data.Liked != wantLiked || payload.Data.LikeCount != wantCount {
			t.Fatalf("like = %+v, want liked=%v count=%d", payload.Data, wantLiked, wantCount)
		}
	}
	toggle(token(t, seed, "manager"), true, 1)
	toggle(token(t, seed, "manager"), false, 0)
	toggle(token(t, seed, "manager"), true, 1)

	// Author edits the title; a cashier cannot pin (no community.write).
	rec = do(t, router, http.MethodPatch, "/v1/community/forum/posts/"+postID, cashier,
		`{"title":"Grill station tips (v2)"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("author edit: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("decode edit: %v", err)
	}
	if post.Data.Title != "Grill station tips (v2)" {
		t.Fatalf("title = %q", post.Data.Title)
	}
	rec = do(t, router, http.MethodPatch, "/v1/community/forum/posts/"+postID, cashier,
		`{"is_pinned":true}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cashier pin: expected 403, got %d", rec.Code)
	}
	rec = do(t, router, http.MethodPatch, "/v1/community/forum/posts/"+postID,
		token(t, seed, "manager"), `{"is_pinned":true,"is_closed":true}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("moderator pin+close: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("decode pin: %v", err)
	}
	if !post.Data.IsPinned || !post.Data.IsClosed {
		t.Fatalf("flags = pinned:%v closed:%v, want both true", post.Data.IsPinned, post.Data.IsClosed)
	}

	// Closed threads reject replies with 409.
	rec = do(t, router, http.MethodPost, "/v1/community/forum/posts/"+postID+"/comments",
		token(t, seed, "manager"), `{"body":"One more thing"}`)
	if rec.Code != http.StatusConflict {
		t.Fatalf("comment on closed post: expected 409, got %d", rec.Code)
	}

	// List filters: pinned sort keeps the thread, category filter matches,
	// search finds it, unknown sort is rejected.
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts?sort=pinned", cashier, "")
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(postID)) {
		t.Fatalf("pinned list: code=%d contains=%v", rec.Code, bytes.Contains(rec.Body.Bytes(), []byte(postID)))
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts?category_id="+categoryID, cashier, "")
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(postID)) {
		t.Fatalf("category filter: code=%d", rec.Code)
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts?q=grill", cashier, "")
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte(postID)) {
		t.Fatalf("search: code=%d", rec.Code)
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts?q=nothing-matches", cashier, "")
	if rec.Code != http.StatusOK || bytes.Contains(rec.Body.Bytes(), []byte(postID)) {
		t.Fatalf("empty search: code=%d", rec.Code)
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts?sort=newest", cashier, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad sort: expected 400, got %d", rec.Code)
	}

	// A second tenant sees an empty forum (FORCE RLS isolation).
	other := testutil.SeedTenant(t, testutil.Pool(t))
	otherToken := testutil.MintAccess(t, other.TenantID, other.ManagerID, other.DeviceID, "", "manager")
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts", otherToken, "")
	if rec.Code != http.StatusOK || bytes.Contains(rec.Body.Bytes(), []byte(postID)) {
		t.Fatalf("cross-tenant posts: code=%d leaked=%v", rec.Code, bytes.Contains(rec.Body.Bytes(), []byte(postID)))
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts/"+postID, otherToken, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant detail: expected 404, got %d", rec.Code)
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/categories", otherToken, "")
	if rec.Code != http.StatusOK || bytes.Contains(rec.Body.Bytes(), []byte(categoryID)) {
		t.Fatalf("cross-tenant categories: code=%d leaked=%v", rec.Code, bytes.Contains(rec.Body.Bytes(), []byte(categoryID)))
	}

	// Manager (not the author) hard-deletes the thread; it is gone for all.
	rec = do(t, router, http.MethodDelete, "/v1/community/forum/posts/"+postID,
		token(t, seed, "manager"), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("moderator delete: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts/"+postID, cashier, "")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("deleted post: expected 404, got %d", rec.Code)
	}
}

// Cashiers can neither moderate nor delete threads they did not write.
func TestForumModerationBlockedForCashier(t *testing.T) {
	router, seed := hub(t)
	cashier := testutil.MintAccess(t, seed.TenantID, seed.CashierID, seed.DeviceID, "", "cashier")

	rec := do(t, router, http.MethodPost, "/v1/community/forum/categories",
		token(t, seed, "manager"), `{"name":"General"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category: %d (%s)", rec.Code, rec.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// Manager authors a thread; the cashier may comment but not edit/delete it.
	rec = do(t, router, http.MethodPost, "/v1/community/forum/posts",
		token(t, seed, "manager"),
		`{"category_id":"`+created.Data.ID+`","title":"Shift handover"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("manager create post: %d (%s)", rec.Code, rec.Body.String())
	}
	var post struct {
		Data ForumPost `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &post); err != nil {
		t.Fatalf("decode: %v", err)
	}
	postID := post.Data.ID

	rec = do(t, router, http.MethodPost, "/v1/community/forum/posts/"+postID+"/comments",
		cashier, `{"body":"Noted."}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("cashier comment: %d (%s)", rec.Code, rec.Body.String())
	}
	rec = do(t, router, http.MethodPatch, "/v1/community/forum/posts/"+postID, cashier,
		`{"title":"Hijacked"}`)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cashier edit others: expected 403, got %d", rec.Code)
	}
	rec = do(t, router, http.MethodDelete, "/v1/community/forum/posts/"+postID, cashier, "")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("cashier delete others: expected 403, got %d", rec.Code)
	}
	// The thread is untouched.
	rec = do(t, router, http.MethodGet, "/v1/community/forum/posts/"+postID, cashier, "")
	if rec.Code != http.StatusOK || !bytes.Contains(rec.Body.Bytes(), []byte("Shift handover")) {
		t.Fatalf("post survived: code=%d", rec.Code)
	}
}
