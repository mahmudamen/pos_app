package settings

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
)

func handlerTokens() security.TokenManager {
	return security.TokenManager{
		Issuer: "pos-api", AccessSecret: []byte("access-secret-that-is-at-least-32-bytes"),
		RefreshSecret: []byte("refresh-secret-that-is-at-least-32-bytes"), AccessTTL: time.Minute, RefreshTTL: time.Hour,
	}
}

func handlerRoleToken(t *testing.T, role string) string {
	t.Helper()
	raw, err := handlerTokens().IssueWithRole(time.Now(), security.AccessToken,
		"00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000002",
		"00000000-0000-0000-0000-000000000003", "00000000-0000-0000-0000-000000000004", role)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func newRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewHandler(nil, handlerTokens()).Register(router.Group("/v1"))
	return router
}

func TestSettingsRoutesRegistered(t *testing.T) {
	router := newRouter()
	for method, path := range map[string]string{
		http.MethodGet: "/v1/settings",
		http.MethodPut: "/v1/settings",
	} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(method, path, nil))
		if recorder.Code == http.StatusNotFound {
			t.Fatalf("%s %s route not registered", method, path)
		}
	}
}

func TestGetSettingsRequiresAuth(t *testing.T) {
	router := newRouter()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/v1/settings", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestGetSettingsUnavailableWithoutDB(t *testing.T) {
	router := newRouter()
	request := httptest.NewRequest(http.MethodGet, "/v1/settings", nil)
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "cashier"))
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPutSettingsRequiresAuth(t *testing.T) {
	router := newRouter()
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/v1/settings", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", recorder.Code)
	}
}

func TestPutSettingsForbiddenForCashier(t *testing.T) {
	router := newRouter()
	body := `{"settings":{"pos.default_payment_method":"card"}}`
	request := httptest.NewRequest(http.MethodPut, "/v1/settings", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "cashier"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPutSettingsBadRequestForInvalidJSON(t *testing.T) {
	router := newRouter()
	request := httptest.NewRequest(http.MethodPut, "/v1/settings", strings.NewReader("not json"))
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "owner"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPutSettingsBadRequestForEmptySettings(t *testing.T) {
	router := newRouter()
	body := `{"settings":{}}`
	request := httptest.NewRequest(http.MethodPut, "/v1/settings", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "owner"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPutSettingsBadRequestForUnknownKey(t *testing.T) {
	router := newRouter()
	body := `{"settings":{"billing.vat_number":"123"}}`
	request := httptest.NewRequest(http.MethodPut, "/v1/settings", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "manager"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPutSettingsBadRequestForInvalidPaymentMethod(t *testing.T) {
	router := newRouter()
	body := `{"settings":{"pos.default_payment_method":"credit"}}`
	request := httptest.NewRequest(http.MethodPut, "/v1/settings", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "owner"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPutSettingsUnavailableWithoutDB(t *testing.T) {
	router := newRouter()
	body := `{"settings":{"pos.show_stock_badges":"false"}}`
	request := httptest.NewRequest(http.MethodPut, "/v1/settings", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+handlerRoleToken(t, "manager"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}
