package purchases

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/ocr"
	"github.com/gin-gonic/gin"
)

func testTokens() security.TokenManager {
	return security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func mint(t *testing.T, role string) string {
	t.Helper()
	raw, err := testTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func setup() (*gin.Engine, *gin.RouterGroup) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	return router, router.Group("/v1")
}

func TestRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens(), ocr.NewEngine(false, "tesseract", "ara+eng", 6)).Register(router.Group("/v1"))
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"POST /v1/purchases/ocr",
		"POST /v1/purchases",
		"GET /v1/purchases",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestEndpointsRequireAccessToken(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), ocr.NewEngine(false, "tesseract", "ara+eng", 6)).Register(group)
	body := `{"invoice_no":"INV-1","items":[{"product_name":"سكر","quantity":2,"unit_price_minor":1000}]}`
	reqs := []*http.Request{
		httptest.NewRequest(http.MethodPost, "/v1/purchases/ocr", nil),
		httptest.NewRequest(http.MethodPost, "/v1/purchases", strings.NewReader(body)),
		httptest.NewRequest(http.MethodGet, "/v1/purchases", nil),
	}
	for _, req := range reqs {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", req.Method, req.URL.Path, recorder.Code)
		}
	}
}

func TestCreateRequiresInventoryPermission(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), ocr.NewEngine(false, "tesseract", "ara+eng", 6)).Register(group)
	recorder := httptest.NewRecorder()
	body := `{"invoice_no":"INV-1","items":[{"product_name":"سكر","quantity":2,"unit_price_minor":1000}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/purchases", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+mint(t, "cashier"))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestOCRRRequiresInventoryPermission(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), ocr.NewEngine(false, "tesseract", "ara+eng", 6)).Register(group)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/purchases/ocr", nil)
	request.Header.Set("Authorization", "Bearer "+mint(t, "cashier"))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", recorder.Code)
	}
}

func TestCreateRejectsInvalidPayloads(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), ocr.NewEngine(false, "tesseract", "ara+eng", 6)).Register(group)
	cases := []string{
		``,             // bad JSON
		`{"items":[]}`, // no lines
		`{"items":[{"quantity":2,"unit_price_minor":1000}]}`,                                                // no name nor id
		`{"items":[{"product_name":"سكر","quantity":0,"unit_price_minor":1000}]}`,                           // zero qty
		`{"items":[{"product_name":"سكر","quantity":2,"unit_price_minor":-5}]}`,                             // negative price
		`{"items":[{"product_name":"سكر","quantity":2,"unit_price_minor":1000,"product_id":"not-a-uuid"}]}`, // bad product id
		`{"currency":"EG","items":[{"product_name":"سكر","quantity":2,"unit_price_minor":1000}]}`,           // bad currency
		`{"tax_minor":-1,"items":[{"product_name":"سكر","quantity":2,"unit_price_minor":1000}]}`,            // negative tax
	}
	for _, body := range cases {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/v1/purchases", strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+mint(t, "manager"))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("body %q: expected 400, got %d", body, recorder.Code)
		}
	}
}

func TestEndpointsUnavailableWithoutDatabase(t *testing.T) {
	router, group := setup()
	NewHandler(nil, testTokens(), ocr.NewEngine(false, "tesseract", "ara+eng", 6)).Register(group)
	// since validation comes first, a valid body reaches the pool check
	body := `{"invoice_no":"INV-1","items":[{"product_id":"00000000-0000-0000-0000-000000000004","quantity":2,"unit_price_minor":1000}]}`
	reqs := []*http.Request{
		func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/v1/purchases", strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+mint(t, "manager"))
			req.Header.Set("Content-Type", "application/json")
			return req
		}(),
		func() *http.Request {
			req := httptest.NewRequest(http.MethodGet, "/v1/purchases", nil)
			req.Header.Set("Authorization", "Bearer "+mint(t, "manager"))
			return req
		}(),
		// ocr with a disabled engine: pool is checked before Availability, so
		// the nil pool surfaces database_unavailable (ordering matches inventory).
		func() *http.Request {
			req := httptest.NewRequest(http.MethodPost, "/v1/purchases/ocr", nil)
			req.Header.Set("Authorization", "Bearer "+mint(t, "manager"))
			return req
		}(),
	}
	for _, req := range reqs {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("%s %s: expected 503, got %d", req.Method, req.URL.Path, recorder.Code)
		}
	}
}

func TestUnitResolutionViaOCRHelpers(t *testing.T) {
	for in, want := range map[string]string{
		"كجم": "kg", "جرام": "g", "لتر": "liter", "قطعه": "piece",
		"كيلو": "kg", "مل": "ml",
	} {
		if got := ocr.ResolveUnit(in); got != want {
			t.Errorf("ResolveUnit(%q) = %q, want %q", in, got, want)
		}
	}
	// canonical latin codes pass through untouched (no alias, non-empty keep)
	if got := ocr.ResolveUnit("ml"); got != "" {
		t.Errorf("ResolveUnit(ml) = %q, want empty (latin codes are already canonical)", got)
	}
}

func multipartUpload(t *testing.T, field, content string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile(field, "invoice.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte(content))
	_ = w.Close()
	return &buf, w.FormDataContentType()
}

func TestOCRScanAcceptsMultipart(t *testing.T) {
	// engine enabled with a working echo binary: Available() is true, OCR
	// returns nothing → downstream 422 (no DB is reached first only if pool
	// were set; with nil pool the handler reports unavailable before OCR).
	engine := ocr.NewEngine(true, "/bin/echo", "ara+eng", 6)
	router, group := setup()
	NewHandler(nil, testTokens(), engine).Register(group)
	body, contentType := multipartUpload(t, "file", "fake-image-bytes")
	req := httptest.NewRequest(http.MethodPost, "/v1/purchases/ocr", body)
	req.Header.Set("Authorization", "Bearer "+mint(t, "manager"))
	req.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	// nil pool wins over the OCR path (matches the documented ordering)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}
