package meta

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/meta/countries", h.listCountries)
	router.GET("/meta/currencies", h.listCurrencies)
}

type Country struct {
	Code         string `json:"code"`
	NameEn       string `json:"name_en"`
	NameAr       string `json:"name_ar"`
	CurrencyCode string `json:"currency_code"`
	PhoneCode    string `json:"phone_code"`
}

type Currency struct {
	Code          string `json:"code"`
	NameEn        string `json:"name_en"`
	NameAr        string `json:"name_ar"`
	Symbol        string `json:"symbol"`
	DigitsDecimal int    `json:"digits_after_decimal"`
}

// listCountries returns the supported countries (no auth required). Demo: Egypt only.
func (h *Handler) listCountries(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	rows, err := h.pool.Query(ctx, `
		SELECT code, name_en, name_ar, currency_code, phone_code
		FROM countries ORDER BY name_en`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load countries")
		return
	}
	defer rows.Close()

	countries := make([]Country, 0)
	for rows.Next() {
		var item Country
		if err := rows.Scan(&item.Code, &item.NameEn, &item.NameAr, &item.CurrencyCode, &item.PhoneCode); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load countries")
			return
		}
		countries = append(countries, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load countries")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": countries, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

// listCurrencies returns the supported currencies (no auth required). Demo: EGP only.
func (h *Handler) listCurrencies(c *gin.Context) {
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	rows, err := h.pool.Query(ctx, `
		SELECT code, name_en, name_ar, symbol, digits_after_decimal
		FROM currencies ORDER BY code`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load currencies")
		return
	}
	defer rows.Close()

	currencies := make([]Currency, 0)
	for rows.Next() {
		var item Currency
		if err := rows.Scan(&item.Code, &item.NameEn, &item.NameAr, &item.Symbol, &item.DigitsDecimal); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load currencies")
			return
		}
		currencies = append(currencies, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load currencies")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": currencies, "meta": gin.H{"request_id": c.GetString("request_id")}})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
