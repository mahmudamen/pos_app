package registers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type listRow struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

func listSessions(t *testing.T, router http.Handler, token, query string) (int, []listRow, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/registers"+query, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(rec, req)
	var out struct {
		Data []listRow `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("parse list %q: %v: %s", query, err, rec.Body.String())
	}
	return rec.Code, out.Data, rec.Body.String()
}

func TestListSessionsIntegration_FiltersAndDeviceFields(t *testing.T) {
	router, seed, _ := setupRegistersIntegration(t)
	cashierToken := regToken(t, seed, seed.CashierID, "cashier")
	sessionID := openSession(t, router, cashierToken)

	code, items, body := listSessions(t, router, cashierToken, "?status=open")
	if code != http.StatusOK || len(items) != 1 {
		t.Fatalf("status=open: expected 200 with 1 row, got %d/%d: %s", code, len(items), body)
	}
	if items[0].ID != sessionID {
		t.Fatalf("status=open: expected %s, got %s", sessionID, items[0].ID)
	}
	if items[0].DeviceID != seed.DeviceID {
		t.Fatalf("device_id: expected %s, got %s", seed.DeviceID, items[0].DeviceID)
	}
	if items[0].DeviceName == "" {
		t.Fatal("device_name: expected the terminal name, got empty")
	}

	if code, items, body = listSessions(t, router, cashierToken, "?status=closed"); code != http.StatusOK || len(items) != 0 {
		t.Fatalf("status=closed before close: expected 0 rows, got %d: %s", len(items), body)
	}
	if code, items, body = listSessions(t, router, cashierToken, "?device_id="+otherTerminalID); code != http.StatusOK || len(items) != 0 {
		t.Fatalf("foreign device: expected 0 rows, got %d: %s", len(items), body)
	}
	if code, items, body = listSessions(t, router, cashierToken, "?from=2099-01-01"); code != http.StatusOK || len(items) != 0 {
		t.Fatalf("future window: expected 0 rows, got %d: %s", len(items), body)
	}
	if code, items, body = listSessions(t, router, cashierToken, "?from=2026-10-01&to=2099-01-01"); code != http.StatusOK || len(items) != 1 {
		t.Fatalf("wide window: expected 1 row, got %d: %s", len(items), body)
	}

	closeSession(t, router, cashierToken, sessionID, "")
	code, items, body = listSessions(t, router, cashierToken, "?status=closed&device_id="+seed.DeviceID)
	if code != http.StatusOK || len(items) != 1 || items[0].ID != sessionID {
		t.Fatalf("closed+own device: expected the session, got %d rows: %s", len(items), body)
	}
}
