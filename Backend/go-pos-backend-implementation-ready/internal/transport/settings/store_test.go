package settings

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetStoreRouteRegistered(t *testing.T) {
	router := newRouter()
	for _, method := range []string{http.MethodGet, http.MethodPut} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, "/v1/store", nil))
		if recorder.Code == http.StatusNotFound {
			t.Fatalf("%s /v1/store route not registered", method)
		}
	}
}

func TestGetStoreRequiresAuth(t *testing.T) {
	router := newRouter()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/store", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetStoreUnavailableWithoutDB(t *testing.T) {
	router := newRouter()
	for _, role := range []string{"owner", "manager", "cashier", "guest"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/store", nil)
		request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, role))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("role %s: expected 503, got %d", role, recorder.Code)
		}
	}
}

func TestUpdateStoreRequiresAuth(t *testing.T) {
	router := newRouter()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/v1/store",
		strings.NewReader(`{"address":"Cairo"}`)))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestUpdateStoreForbiddenForCashier(t *testing.T) {
	router := newRouter()
	request := httptest.NewRequest(http.MethodPut, "/v1/store",
		strings.NewReader(`{"address":"Cairo"}`))
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "cashier"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpdateStoreValidation(t *testing.T) {
	router := newRouter()
	cases := []struct {
		name string
		body string
		role string
	}{
		{"empty body", `{}`, "owner"},
		{"garbage", `not json`, "owner"},
		{"blank name", `{"name":"   "}`, "owner"},
		{"long name", `{"name":"` + strings.Repeat("x", 121) + `"}`, "manager"},
		{"long address", `{"address":"` + strings.Repeat("x", 256) + `"}`, "owner"},
	}
	for _, tc := range cases {
		request := httptest.NewRequest(http.MethodPut, "/v1/store",
			strings.NewReader(tc.body))
		request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, tc.role))
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d: %s", tc.name, recorder.Code, recorder.Body.String())
		}
	}
}

func TestValidateStoreNameAndAddress(t *testing.T) {
	if err := validateStoreName("My Shop"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
	for _, bad := range []string{"", "   ", strings.Repeat("a", maxStoreNameLen+1)} {
		if err := validateStoreName(bad); err == nil {
			t.Fatalf("name %q accepted", bad)
		}
	}
	if err := validateStoreAddress(strings.Repeat("a", maxStoreAddressLen)); err != nil {
		t.Fatalf("max-length address rejected: %v", err)
	}
	if err := validateStoreAddress(strings.Repeat("a", maxStoreAddressLen+1)); err == nil {
		t.Fatal("over-long address accepted")
	}
}
