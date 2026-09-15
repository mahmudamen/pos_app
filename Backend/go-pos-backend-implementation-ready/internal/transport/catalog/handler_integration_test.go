package catalog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/gin-gonic/gin"
)

func authHeader(token string) string { return "Bearer " + token }

func toJSON(t *testing.T, v interface{}) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustParseJSON(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	return m
}

func TestCatalogIntegration_CreateListUpdateDeleteCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	tokens := testutil.TokenManager()
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-1", "manager")
	router := gin.New()
	NewHandler(pool, tokens).Register(router.Group("/v1"))

	// Create category
	body := `{"name":"Drinks","slug":"drinks"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/categories", bytes.NewBufferString(body))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	cat := mustParseJSON(t, rec.Body.Bytes())
	catData := cat["data"].(map[string]interface{})
	catID := catData["id"].(string)
	if catData["name"] != "Drinks" {
		t.Errorf("name: got %q, want %q", catData["name"], "Drinks")
	}

	// List categories — should include new category
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/categories", nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list categories: expected 200, got %d", rec.Code)
	}
	list := mustParseJSON(t, rec.Body.Bytes())
	data := list["data"].([]interface{})
	found := false
	for _, item := range data {
		if item.(map[string]interface{})["id"] == catID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("created category not found in list")
	}

	// Update category
	updateBody := `{"name":"Hot Drinks"}`
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPatch, "/v1/categories/"+catID, bytes.NewBufferString(updateBody))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update category: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	updated := mustParseJSON(t, rec.Body.Bytes())
	updatedData := updated["data"].(map[string]interface{})
	if updatedData["name"] != "Hot Drinks" {
		t.Errorf("updated name: got %q, want %q", updatedData["name"], "Hot Drinks")
	}

	// Delete category (soft-delete)
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/categories/"+catID, nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete category: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// List categories — deleted category absent
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/categories", nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list after delete: expected 200, got %d", rec.Code)
	}
	list2 := mustParseJSON(t, rec.Body.Bytes())
	data2 := list2["data"].([]interface{})
	for _, item := range data2 {
		if item.(map[string]interface{})["id"] == catID {
			t.Fatal("deleted category should not appear in list")
		}
	}
}

func TestCatalogIntegration_CreateProductWithCategory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	tokens := testutil.TokenManager()
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-1", "manager")
	router := gin.New()
	NewHandler(pool, tokens).Register(router.Group("/v1"))

	// Create category
	catBody := `{"name":"Food","slug":"food"}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/categories", bytes.NewBufferString(catBody))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create category: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	cat := mustParseJSON(t, rec.Body.Bytes())
	catID := cat["data"].(map[string]interface{})["id"].(string)

	// Create product with category
	prodBody := toJSON(t, map[string]interface{}{
		"name":           "Sandwich",
		"sku":            "SND-1",
		"price_minor":    550,
		"cost_minor":     200,
		"currency":       "USD",
		"stock_quantity": 10,
		"category_id":    catID,
	})
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(prodBody))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	prod := mustParseJSON(t, rec.Body.Bytes())
	prodData := prod["data"].(map[string]interface{})
	if prodData["category_id"] != catID {
		t.Errorf("category_id: got %q, want %q", prodData["category_id"], catID)
	}
}

func TestCatalogIntegration_ProductSoftDelete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	tokens := testutil.TokenManager()
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-1", "manager")
	router := gin.New()
	NewHandler(pool, tokens).Register(router.Group("/v1"))

	// Create product
	prodBody := `{"name":"Latte","sku":"LAT-1","price_minor":400,"currency":"USD","stock_quantity":5}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(prodBody))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	prod := mustParseJSON(t, rec.Body.Bytes())
	prodID := prod["data"].(map[string]interface{})["id"].(string)

	// Soft-delete product
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/v1/products/"+prodID, nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete product: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get product — soft-deleted rows stay viewable with is_active=false
	// (the sync pull decodes that as change_type: delete).
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/products/"+prodID, nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get soft-deleted product: expected 200, got %d", rec.Code)
	}
	body := mustParseJSON(t, rec.Body.Bytes())
	p := body["data"].(map[string]interface{})
	if p["is_active"] != false {
		t.Fatalf("soft-deleted product must carry is_active=false, got %v", p["is_active"])
	}
}

func TestCatalogIntegration_PaginationAndSearch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	tokens := testutil.TokenManager()
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-1", "manager")
	router := gin.New()
	NewHandler(pool, tokens).Register(router.Group("/v1"))

	// Create 3 products
	for i, name := range []string{"Espresso", "Latte", "Cappuccino"} {
		prodBody := toJSON(t, map[string]interface{}{
			"name":           name,
			"sku":            "SKU-" + string(rune('A'+i)),
			"price_minor":    300,
			"currency":       "USD",
			"stock_quantity": 5,
		})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(prodBody))
		req.Header.Set("Authorization", authHeader(token))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("create product %s: expected 201, got %d: %s", name, rec.Code, rec.Body.String())
		}
	}

	// Pagination: page=1, limit=2 → total=3, len=2
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/products?page=1&limit=2", nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list page 1: expected 200, got %d", rec.Code)
	}
	page1 := mustParseJSON(t, rec.Body.Bytes())
	meta1 := page1["meta"].(map[string]interface{})
	if int(meta1["total"].(float64)) != 3 {
		t.Errorf("total: got %v, want 3", meta1["total"])
	}
	data1 := page1["data"].([]interface{})
	if len(data1) != 2 {
		t.Errorf("page 1 len: got %d, want 2", len(data1))
	}

	// Search
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/products?search=espresso", nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("search: expected 200, got %d", rec.Code)
	}
	search := mustParseJSON(t, rec.Body.Bytes())
	searchData := search["data"].([]interface{})
	if len(searchData) != 1 {
		t.Errorf("search results: got %d, want 1", len(searchData))
	}
}

func TestCatalogIntegration_BarcodeLookup(t *testing.T) {
	gin.SetMode(gin.TestMode)
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	seed := testutil.SeedTenant(t, pool)
	tokens := testutil.TokenManager()
	token := testutil.MintAccess(t, seed.TenantID, seed.ManagerID, seed.DeviceID, "sess-1", "manager")
	router := gin.New()
	NewHandler(pool, tokens).Register(router.Group("/v1"))

	// Create product with barcode
	prodBody := `{"name":"Mocha","sku":"MOC-1","barcode":"9780123456789","price_minor":450,"currency":"USD","stock_quantity":3}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/products", bytes.NewBufferString(prodBody))
	req.Header.Set("Authorization", authHeader(token))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	// Lookup by barcode
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/products/barcode/9780123456789", nil)
	req.Header.Set("Authorization", authHeader(token))
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("barcode lookup: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	result := mustParseJSON(t, rec.Body.Bytes())
	data := result["data"].(map[string]interface{})
	if data["name"] != "Mocha" {
		t.Errorf("name: got %q, want %q", data["name"], "Mocha")
	}
}
