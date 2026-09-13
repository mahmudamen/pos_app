package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
)

func newRouter(h *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	h.Register(router.Group("/v1"))
	return router
}

func TestRoutesAreRegistered(t *testing.T) {
	router := newRouter(NewHandler(nil))
	routes := map[string]bool{}
	for _, r := range router.Routes() {
		routes[r.Method+" "+r.Path] = true
	}
	for _, want := range []string{
		"GET /v1/meta/countries",
		"GET /v1/meta/currencies",
	} {
		if !routes[want] {
			t.Fatalf("route %s not registered", want)
		}
	}
}

func TestCountriesRequireNoAuthAndReachPool(t *testing.T) {
	router := newRouter(NewHandler(nil))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/meta/countries", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without a pool, got %d", rec.Code)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse 503 body: %v", err)
	}
	if body.Error.Code != "database_unavailable" {
		t.Fatalf("expected database_unavailable, got %q", body.Error.Code)
	}
}

func TestCurrenciesRequireNoAuthAndReachPool(t *testing.T) {
	router := newRouter(NewHandler(nil))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/meta/currencies", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without a pool, got %d", rec.Code)
	}
}

func TestCountriesReturnsEGYPT(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	router := newRouter(NewHandler(pool))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/meta/countries", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data []Country `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Code != "EG" {
		t.Fatalf("expected a single EG country, got %+v", body.Data)
	}
	if body.Data[0].NameAr == "" || body.Data[0].CurrencyCode == "" || body.Data[0].PhoneCode == "" {
		t.Fatalf("expected localized fields populated, got %+v", body.Data[0])
	}
}

func TestCurrenciesReturnsEGP(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	router := newRouter(NewHandler(pool))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/meta/currencies", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Data []Currency `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse body: %v", err)
	}
	if len(body.Data) != 1 || body.Data[0].Code != "EGP" {
		t.Fatalf("expected a single EGP currency, got %+v", body.Data)
	}
	if body.Data[0].Symbol == "" || body.Data[0].DigitsDecimal != 2 {
		t.Fatalf("expected formatted currency metadata, got %+v", body.Data[0])
	}
}
