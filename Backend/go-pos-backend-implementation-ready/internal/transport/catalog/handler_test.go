package catalog

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func testTokens() security.TokenManager {
	return security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func validToken(t *testing.T) string {
	t.Helper()
	raw, err := testTokens().Issue(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestProductsRequireAccessToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestProductsReturnUnavailableWithoutDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestListCategoriesRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/categories", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestListCategoriesReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/categories", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestCreateCategoryRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/categories", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestCreateCategoryReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"name":"Drinks","slug":"drinks"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/categories", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateCategoryReturnsBadRequestForMissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"name":""}`
	request := httptest.NewRequest(http.MethodPost, "/v1/categories", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateCategoryReturnsBadRequestForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/categories", strings.NewReader("not json"))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateProductRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/products", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestCreateProductReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"name":"Espresso","sku":"ESP-1","price_minor":280,"currency":"USD","stock_quantity":10}`
	request := httptest.NewRequest(http.MethodPost, "/v1/products", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateProductReturnsBadRequestForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/products", strings.NewReader("not json"))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetProductRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/products/00000000-0000-0000-0000-000000000001", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetProductReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/products/00000000-0000-0000-0000-000000000001", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestUpdateProductRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPatch, "/v1/products/00000000-0000-0000-0000-000000000001", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestUpdateProductReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"name":"Updated"}`
	request := httptest.NewRequest(http.MethodPatch, "/v1/products/00000000-0000-0000-0000-000000000001", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateProductReturnsBadRequestForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/products/00000000-0000-0000-0000-000000000001", strings.NewReader("not json"))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestGetByBarcodeRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/products/barcode/12345", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetByBarcodeReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/products/barcode/12345", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", recorder.Code)
	}
}

func TestValidProductAcceptsValidInput(t *testing.T) {
	if !validProduct("Espresso", "ESP-1", "USD", 280, 100, 10) {
		t.Fatal("expected validProduct to return true for valid input")
	}
}

func TestValidProductRejectsEmptyName(t *testing.T) {
	if validProduct("", "ESP-1", "USD", 280, 100, 10) {
		t.Fatal("expected validProduct to return false for empty name")
	}
}

func TestValidProductRejectsEmptySku(t *testing.T) {
	if validProduct("Espresso", "", "USD", 280, 100, 10) {
		t.Fatal("expected validProduct to return false for empty sku")
	}
}

func TestValidProductRejectsWrongCurrencyLength(t *testing.T) {
	if validProduct("Espresso", "ESP-1", "USDX", 280, 100, 10) {
		t.Fatal("expected validProduct to return false for 4-char currency")
	}
}

func TestValidProductRejectsNegativePrice(t *testing.T) {
	if validProduct("Espresso", "ESP-1", "USD", -1, 100, 10) {
		t.Fatal("expected validProduct to return false for negative price")
	}
}

func TestValidProductRejectsNegativeCost(t *testing.T) {
	if validProduct("Espresso", "ESP-1", "USD", 280, -1, 10) {
		t.Fatal("expected validProduct to return false for negative cost")
	}
}

func TestValidProductRejectsNegativeStock(t *testing.T) {
	if validProduct("Espresso", "ESP-1", "USD", 280, 100, -1) {
		t.Fatal("expected validProduct to return false for negative stock")
	}
}

func TestApplyPatchUpdatesName(t *testing.T) {
	p := &Product{Name: "Old"}
	name := "New"
	applyPatch(p, productPatchRequest{Name: &name})
	if p.Name != "New" {
		t.Errorf("Name: got %q, want %q", p.Name, "New")
	}
}

func TestApplyPatchUpdatesPriceMinor(t *testing.T) {
	p := &Product{PriceMinor: 100}
	price := int64(200)
	applyPatch(p, productPatchRequest{PriceMinor: &price})
	if p.PriceMinor != 200 {
		t.Errorf("PriceMinor: got %d, want 200", p.PriceMinor)
	}
}

func TestApplyPatchUpdatesIsActive(t *testing.T) {
	p := &Product{IsActive: true}
	active := false
	applyPatch(p, productPatchRequest{IsActive: &active})
	if p.IsActive {
		t.Error("expected IsActive to be false")
	}
}

func TestApplyPatchDoesNotClearUnsetFields(t *testing.T) {
	p := &Product{Name: "Original", SKU: "SKU-1", PriceMinor: 100}
	newName := "Updated"
	applyPatch(p, productPatchRequest{Name: &newName})
	if p.SKU != "SKU-1" {
		t.Errorf("SKU should be unchanged, got %q", p.SKU)
	}
	if p.PriceMinor != 100 {
		t.Errorf("PriceMinor should be unchanged, got %d", p.PriceMinor)
	}
}

func TestWriteErrorReturnsCorrectJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		writeError(c, 404, "not_found", "resource missing")
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/test", nil))
	if recorder.Code != 404 {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"code":"not_found"`) {
		t.Errorf("expected error code in response: %s", body)
	}
	if !strings.Contains(body, `"message":"resource missing"`) {
		t.Errorf("expected error message in response: %s", body)
	}
}

func TestUpdateCategoryRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPatch, "/v1/categories/00000000-0000-0000-0000-000000000001", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestUpdateCategoryReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"name":"Updated"}`
	request := httptest.NewRequest(http.MethodPatch, "/v1/categories/00000000-0000-0000-0000-000000000001", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateCategoryReturnsBadRequestForInvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/categories/00000000-0000-0000-0000-000000000001", strings.NewReader("not json"))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateCategoryReturnsBadRequestForEmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"name":" "}`
	request := httptest.NewRequest(http.MethodPatch, "/v1/categories/00000000-0000-0000-0000-000000000001", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateCategoryReturnsBadRequestForEmptySlug(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	body := `{"slug":" "}`
	request := httptest.NewRequest(http.MethodPatch, "/v1/categories/00000000-0000-0000-0000-000000000001", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDeleteCategoryRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/v1/categories/00000000-0000-0000-0000-000000000001", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestDeleteCategoryReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/v1/categories/00000000-0000-0000-0000-000000000001", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestDeleteProductRequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/v1/products/00000000-0000-0000-0000-000000000001", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestDeleteProductReturnsUnavailableWithoutDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, testTokens()).Register(router.Group("/v1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/v1/products/00000000-0000-0000-0000-000000000001", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestApplyPatchUpdatesCategorySlug(t *testing.T) {
	p := &Product{CategoryID: "old-cat"}
	newSlug := "new-slug"
	applyPatch(p, productPatchRequest{CategoryID: &newSlug})
	if p.CategoryID != "new-slug" {
		t.Errorf("CategoryID: got %q, want %q", p.CategoryID, "new-slug")
	}
}

func TestApplyPatchUpdatesCostMinor(t *testing.T) {
	p := &Product{CostMinor: 50}
	cost := int64(75)
	applyPatch(p, productPatchRequest{CostMinor: &cost})
	if p.CostMinor != 75 {
		t.Errorf("CostMinor: got %d, want 75", p.CostMinor)
	}
}

func TestApplyPatchUpdatesStockQuantity(t *testing.T) {
	p := &Product{StockQuantity: 10}
	stock := int64(20)
	applyPatch(p, productPatchRequest{StockQuantity: &stock})
	if p.StockQuantity != 20 {
		t.Errorf("StockQuantity: got %d, want 20", p.StockQuantity)
	}
}

func TestApplyPatchUpdatesBarcode(t *testing.T) {
	p := &Product{Barcode: "OLD"}
	barcode := "NEW123"
	applyPatch(p, productPatchRequest{Barcode: &barcode})
	if p.Barcode != "NEW123" {
		t.Errorf("Barcode: got %q, want %q", p.Barcode, "NEW123")
	}
}

func TestApplyPatchUpdatesCurrency(t *testing.T) {
	p := &Product{Currency: "USD"}
	currency := "eur"
	applyPatch(p, productPatchRequest{Currency: &currency})
	if p.Currency != "EUR" {
		t.Errorf("Currency: got %q, want %q", p.Currency, "EUR")
	}
}
