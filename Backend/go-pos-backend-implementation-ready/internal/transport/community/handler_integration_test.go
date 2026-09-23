package community

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
)

func hub(t *testing.T) (*gin.Engine, testutil.Seed) {
	t.Helper()
	connString := testutil.DatabaseURL(t)
	pool := testutil.Pool(t)
	testutil.Migrate(t, connString)
	seed := testutil.SeedTenant(t, pool)
	h := NewHandler(pool, testutil.TokenManager())
	router := gin.New()
	h.Register(router.Group("/v1"))
	return router, seed
}

func token(t *testing.T, seed testutil.Seed, role string) string {
	t.Helper()
	return testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "", role)
}

func do(t *testing.T, router *gin.Engine, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestJobBoardRoundTrip(t *testing.T) {
	router, seed := hub(t)

	// Manager creates a job offer (cashier cannot).
	status := do(t, router, http.MethodPost, "/v1/community/jobs",
		token(t, seed, "cashier"), `{"title":"Barista"}`)
	if status.Code != http.StatusForbidden {
		t.Fatalf("cashier create job: expected 403, got %d", status.Code)
	}

	body := `{"title":"Barista","description":"Morning shift","employment_type":"full_time",
		"location":"Cairo","salary_minor":1800000,"skill_tags":["coffee","Grill","coffee"],
		"closes_at":"2026-12-31T23:59:59Z"}`
	status = do(t, router, http.MethodPost, "/v1/community/jobs", token(t, seed, "manager"), body)
	if status.Code != http.StatusCreated {
		t.Fatalf("create job: expected 201, got %d (%s)", status.Code, status.Body.String())
	}
	var created struct {
		Data struct {
			ID          string   `json:"id"`
			Title       string   `json:"title"`
			SalaryMinor int      `json:"salary_minor"`
			SkillTags   []string `json:"skill_tags"`
			Applied     bool     `json:"applied"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Data.ID == "" {
		t.Fatal("no job id returned")
	}

	// List: cashier can read, sees the applied flag false.
	status = do(t, router, http.MethodGet, "/v1/community/jobs", token(t, seed, "cashier"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("list jobs: expected 200, got %d", status.Code)
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list body %q: %v", status.Body.String(), err)
	}
	if len(list.Data) != 1 || list.Data[0]["id"] != created.Data.ID {
		t.Fatalf("expected one owned job, got %+v", list.Data)
	}

	// Cashier applies.
	status = do(t, router, http.MethodPost, "/v1/community/jobs/"+created.Data.ID+"/apply",
		token(t, seed, "cashier"), `{"cover_note":"I love coffee"}`)
	if status.Code != http.StatusCreated {
		t.Fatalf("apply: expected 201, got %d (%s)", status.Code, status.Body.String())
	}
	var applied struct {
		Data struct {
			ID          string `json:"id"`
			ApplicantID string `json:"applicant_id"`
			Status      string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &applied); err != nil {
		t.Fatalf("decode apply: %v", err)
	}
	if applied.Data.ApplicantID == "" || applied.Data.Status != "applied" {
		t.Fatalf("unexpected application: %+v", applied.Data)
	}

	// Duplicate apply → 409.
	status = do(t, router, http.MethodPost, "/v1/community/jobs/"+created.Data.ID+"/apply",
		token(t, seed, "cashier"), `{"cover_note":"again"}`)
	if status.Code != http.StatusConflict {
		t.Fatalf("duplicate apply: expected 409, got %d (%s)", status.Code, status.Body.String())
	}

	// Detail now reports applied for the cashier.
	status = do(t, router, http.MethodGet, "/v1/community/jobs/"+created.Data.ID,
		token(t, seed, "cashier"), "")
	var detail map[string]any
	_ = json.Unmarshal(status.Body.Bytes(), &detail)
	if detail["data"].(map[string]any)["applied"] != true {
		t.Fatalf("expected applied=true, got %+v", detail["data"])
	}

	// mine=1 shows only jobs the cashier applied to.
	status = do(t, router, http.MethodGet, "/v1/community/jobs?mine=1", token(t, seed, "cashier"), "")
	var mine struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(status.Body.Bytes(), &mine)
	if len(mine.Data) != 1 {
		t.Fatalf("mine filter: expected 1, got %+v", mine.Data)
	}

	// Manager views applications and transitions the status.
	status = do(t, router, http.MethodGet, "/v1/community/jobs/"+created.Data.ID+"/applications",
		token(t, seed, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("list applications: expected 200, got %d", status.Code)
	}
	var apps struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &apps); err != nil {
		t.Fatalf("decode apps %q: %v", status.Body.String(), err)
	}
	if len(apps.Data) != 1 || apps.Data[0]["applicant_id"] == "" {
		t.Fatalf("expected one application, got %+v", apps.Data)
	}

	status = do(t, router, http.MethodPatch, "/v1/community/applications/"+applied.Data.ID,
		token(t, seed, "manager"), `{"status":"under_review"}`)
	if status.Code != http.StatusOK {
		t.Fatalf("update status: expected 200, got %d (%s)", status.Code, status.Body.String())
	}

	// manager cannot apply to their own job? They can (any staff) - just verify
	// the applications list reflects the new status.
	status = do(t, router, http.MethodGet, "/v1/community/jobs/"+created.Data.ID+"/applications",
		token(t, seed, "manager"), "")
	_ = json.Unmarshal(status.Body.Bytes(), &apps)
	if apps.Data[0]["status"] != "under_review" {
		t.Fatalf("expected under_review, got %+v", apps.Data[0])
	}

	// Manager disables the job (soft delete).
	status = do(t, router, http.MethodDelete, "/v1/community/jobs/"+created.Data.ID,
		token(t, seed, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("delete job: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	status = do(t, router, http.MethodGet, "/v1/community/jobs", token(t, seed, "cashier"), "")
	_ = json.Unmarshal(status.Body.Bytes(), &list)
	if len(list.Data) != 0 {
		t.Fatalf("job should be hidden after soft delete, got %+v", list.Data)
	}
}

func TestJobBoardCrossTenantIsolation(t *testing.T) {
	router, seed := hub(t)

	status := do(t, router, http.MethodPost, "/v1/community/jobs",
		token(t, seed, "manager"), `{"title":"Barista"}`)
	if status.Code != http.StatusCreated {
		t.Fatalf("create owner job: expected 201, got %d (%s)", status.Code, status.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(status.Body.Bytes(), &created)

	// Another tenant sees no jobs and cannot apply to ours.
	other := testutil.SeedTenant(t, testutil.Pool(t))
	status = do(t, router, http.MethodGet, "/v1/community/jobs", token(t, other, "manager"), "")
	var otherList struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(status.Body.Bytes(), &otherList)
	if len(otherList.Data) != 0 {
		t.Fatalf("other tenant sees %d jobs, want 0", len(otherList.Data))
	}
	status = do(t, router, http.MethodPost, "/v1/community/jobs/"+created.Data.ID+"/apply",
		token(t, other, "manager"), `{"cover_note":""}`)
	if status.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant apply: expected 404, got %d", status.Code)
	}
}

func TestProfileRoundTrip(t *testing.T) {
	router, seed := hub(t)

	status := do(t, router, http.MethodGet, "/v1/community/profiles/me", token(t, seed, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("GET me before profile: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	var me struct {
		Data struct {
			HasProfile bool `json:"has_profile"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	if me.Data.HasProfile {
		t.Fatal("expected has_profile=false before upsert")
	}

	body := `{"headline":"Head Chef","bio":"Team lead & grill master","location":"Cairo",
		"years_experience":12,"skills":["Grill","Sous","Grill"],"is_chief":true}`
	status = do(t, router, http.MethodPut, "/v1/community/profiles/me", token(t, seed, "cashier"), body)
	if status.Code != http.StatusOK {
		t.Fatalf("upsert profile: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	var upserted struct {
		Data struct {
			HasProfile      bool     `json:"has_profile"`
			YearsExperience int      `json:"years_experience"`
			IsChief         bool     `json:"is_chief"`
			Skills          []string `json:"skills"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &upserted); err != nil {
		t.Fatalf("decode upsert: %v", err)
	}
	if !upserted.Data.HasProfile {
		t.Fatal("expected has_profile=true after upsert")
	}
	if upserted.Data.YearsExperience != 12 || !upserted.Data.IsChief {
		t.Fatal("profile fields not persisted")
	}
	if len(upserted.Data.Skills) != 2 || upserted.Data.Skills[0] != "grill" || upserted.Data.Skills[1] != "sous" {
		t.Fatalf("skills not normalized+deduped: %v", upserted.Data.Skills)
	}

	// List: cashier sees manager profile (has_profile join works).
	status = do(t, router, http.MethodGet, "/v1/community/profiles", token(t, seed, "cashier"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("list profiles: expected 200, got %d", status.Code)
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list body %q: %v", status.Body.String(), err)
	}
	found := false
	for _, item := range list.Data {
		if item["email"] == seed.ManagerEmail {
			found = true
			if item["has_profile"] != true {
				t.Fatal("manager should have profile")
			}
		}
	}
	if !found {
		t.Fatal("manager not in profile list")
	}
}

func TestCompanyRoundTrip(t *testing.T) {
	router, seed := hub(t)

	// cashier cannot create.
	status := do(t, router, http.MethodPost, "/v1/community/companies",
		token(t, seed, "cashier"), `{"name":"Bakery X"}`)
	if status.Code != http.StatusForbidden {
		t.Fatalf("cashier create company: expected 403, got %d", status.Code)
	}

	// manager creates.
	status = do(t, router, http.MethodPost, "/v1/community/companies",
		token(t, seed, "manager"), `{"name":"Bakery X","industry":"food","city":"Cairo","about":"Fresh sourdough"}`)
	if status.Code != http.StatusCreated {
		t.Fatalf("create company: expected 201, got %d (%s)", status.Code, status.Body.String())
	}
	var created struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create: %v", err)
	}
	if created.Data.ID == "" {
		t.Fatal("no company id returned")
	}

	// List as manager: owns it.
	status = do(t, router, http.MethodGet, "/v1/community/companies", token(t, seed, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("list companies: expected 200, got %d", status.Code)
	}
	var list struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(status.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list body %q: %v", status.Body.String(), err)
	}
	if len(list.Data) != 1 || list.Data[0]["id"] != created.Data.ID {
		t.Fatalf("expected one owned company, got %+v", list.Data)
	}

	// Let the cashier join as a member (owner adds them).
	status = do(t, router, http.MethodPost, "/v1/community/companies/"+created.Data.ID+"/members",
		token(t, seed, "manager"), `{"user_id":"`+seed.CashierID+`","role":"staff","title":"Line Cook"}`)
	if status.Code != http.StatusCreated {
		t.Fatalf("add member: expected 201, got %d (%s)", status.Code, status.Body.String())
	}

	// Cashier now sees membership under their profile detail.
	status = do(t, router, http.MethodGet, "/v1/community/profiles/"+seed.CashierID, token(t, seed, "cashier"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("cashier profile: expected 200, got %d", status.Code)
	}
	var cashierProfile map[string]any
	if err := json.Unmarshal(status.Body.Bytes(), &cashierProfile); err != nil {
		t.Fatalf("decode cashier profile %q: %v", status.Body.String(), err)
	}
	memberships := cashierProfile["memberships"].([]any)
	if len(memberships) != 1 || memberships[0].(map[string]any)["role"] != "staff" {
		t.Fatalf("expected one staff membership, got %+v", memberships)
	}

	// Company detail includes that member.
	status = do(t, router, http.MethodGet, "/v1/community/companies/"+created.Data.ID, token(t, seed, "cashier"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("company detail: expected 200, got %d", status.Code)
	}
	var detail map[string]any
	if err := json.Unmarshal(status.Body.Bytes(), &detail); err != nil {
		t.Fatalf("decode company detail %q: %v", status.Body.String(), err)
	}
	if members := len(detail["members"].([]any)); members != 1 {
		t.Fatalf("expected 1 member, got %d (%s)", members, status.Body.String())
	}

	// Remove the member -> cashier no longer sees it.
	status = do(t, router, http.MethodDelete, "/v1/community/companies/"+created.Data.ID+"/members/"+seed.CashierID,
		token(t, seed, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("remove member: expected 200, got %d (%s)", status.Code, status.Body.String())
	}
	status = do(t, router, http.MethodGet, "/v1/community/companies/"+created.Data.ID, token(t, seed, "cashier"), "")
	var after map[string]any
	_ = json.Unmarshal(status.Body.Bytes(), &after)
	if members := len(after["members"].([]any)); members != 0 {
		t.Fatalf("expected 0 members after removal, got %d", members)
	}

	// Owner deletes the company (soft delete).
	status = do(t, router, http.MethodDelete, "/v1/community/companies/"+created.Data.ID,
		token(t, seed, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("delete company: expected 200, got %d (%s)", status.Code, status.Body.String())
	}

	// Cross-tenant isolation: another tenant sees nothing.
	other := testutil.SeedTenant(t, testutil.Pool(t))
	status = do(t, router, http.MethodGet, "/v1/community/profiles", token(t, other, "manager"), "")
	if status.Code != http.StatusOK {
		t.Fatalf("other-tenant list: expected 200, got %d", status.Code)
	}
	var otherList struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(status.Body.Bytes(), &otherList)
	if len(otherList.Data) != 2 {
		t.Fatalf("other tenant sees %d profiles, want 2", len(otherList.Data))
	}
}
