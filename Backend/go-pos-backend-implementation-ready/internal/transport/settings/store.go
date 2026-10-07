package settings

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Store profile (tenant identity) endpoints. The tenant row has no RLS, so
// every statement pins the caller's own tenant id from the verified claims —
// never from a request body.

const (
	maxStoreNameLen    = 120
	maxStoreAddressLen = 255
)

// validateStoreName checks a non-empty trimmed store name.
func validateStoreName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return errors.New("name must not be blank")
	}
	if len([]rune(trimmed)) > maxStoreNameLen {
		return errors.New("name is too long")
	}
	return nil
}

// validateStoreAddress checks a store address (may be cleared to empty).
func validateStoreAddress(address string) error {
	if len([]rune(strings.TrimSpace(address))) > maxStoreAddressLen {
		return errors.New("address is too long")
	}
	return nil
}

type storeProfile struct {
	Name            string `json:"name"`
	Address         string `json:"address"`
	BusinessType    string `json:"business_type"`
	CountryCode     string `json:"country_code"`
	CurrencyCode    string `json:"currency_code"`
	DefaultLanguage string `json:"default_language"`
	Plan            string `json:"plan"`
}

type updateStoreRequest struct {
	Name    *string `json:"name"`
	Address *string `json:"address"`
}

func (h *Handler) getStore(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	ctx := c.Request.Context()
	var profile storeProfile
	err = h.pool.QueryRow(ctx, `
		SELECT name, COALESCE(address, ''),
		       COALESCE(business_type, ''), COALESCE(country_code, ''),
		       COALESCE(currency_code, ''), COALESCE(default_language, ''),
		       COALESCE(plan, '')
		FROM tenants WHERE id = $1::uuid`, tenantID.String()).
		Scan(&profile.Name, &profile.Address, &profile.BusinessType,
			&profile.CountryCode, &profile.CurrencyCode,
			&profile.DefaultLanguage, &profile.Plan)
	if err != nil {
		writeError(c, http.StatusNotFound, "store_not_found", "store profile is unavailable")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profile, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) updateStore(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !canManageSettings(claims.Role) {
		writeError(c, http.StatusForbidden, "forbidden", "owner or manager role is required")
		return
	}
	var request updateStoreRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid store request")
		return
	}
	if request.Name == nil && request.Address == nil {
		writeError(c, http.StatusBadRequest, "validation_error", "nothing to update")
		return
	}
	if request.Name != nil {
		if err := validateStoreName(*request.Name); err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid name: "+err.Error())
			return
		}
	}
	if request.Address != nil {
		if err := validateStoreAddress(*request.Address); err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid address: "+err.Error())
			return
		}
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	ctx := c.Request.Context()
	name := request.Name
	address := request.Address
	tag, err := h.pool.Exec(ctx, `
		UPDATE tenants SET
			name = CASE WHEN $2::bool THEN $3 ELSE name END,
			address = CASE WHEN $4::bool THEN $5 ELSE COALESCE(address, '') END,
			updated_at = NOW()
		WHERE id = $1::uuid`,
		tenantID.String(), name != nil, stringValue(name),
		address != nil, stringValue(address))
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save store profile")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "store_not_found", "store profile is unavailable")
		return
	}
	h.getStore(c)
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
