package registers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/registers/current", h.current)
	router.POST("/registers/open", h.open)
	router.POST("/registers/:id/close", h.close)
	router.GET("/registers", h.list)
	router.GET("/registers/:id", h.get)
}

type openRequest struct {
	OpeningCashMinor int64 `json:"opening_cash_minor"`
}

type closeRequest struct {
	ClosingCashMinor *int64 `json:"closing_cash_minor"`
	ManagerPIN       string `json:"manager_pin"`
}

// RegisterSession is one cashier shift on one terminal.
type RegisterSession struct {
	ID                  string         `json:"id"`
	Status              string         `json:"status"`
	OpeningCashMinor    int64          `json:"opening_cash_minor"`
	ClosingCashMinor    *int64         `json:"closing_cash_minor,omitempty"`
	ExpectedCashMinor   *int64         `json:"expected_cash_minor,omitempty"`
	CashDifferenceMinor *int64         `json:"cash_difference_minor,omitempty"`
	OpenedAt            string         `json:"opened_at"`
	ClosedAt            *string        `json:"closed_at,omitempty"`
	OpenedBy            string         `json:"opened_by"`
	Summary             SessionSummary `json:"summary"`
}

type sessionRow struct {
	id                  string
	status              string
	openingCashMinor    int64
	closingCashMinor    *int64
	expectedCashMinor   *int64
	cashDifferenceMinor *int64
	openedAt            string
	closedAt            *string
	openedBy            string
}

func (h *Handler) current(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "pos", "read") {
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
	deviceID, err := uuid.Parse(claims.DeviceID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	row, err := queryOpenSession(ctx, tx, tenantID.String(), deviceID.String())
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "no_open_session", "no open session for this terminal")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load session")
		return
	}
	totals, err := queryTotals(ctx, tx, row.id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally session")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load session")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": toSession(row, buildSummary(totals, row.openingCashMinor)),
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) open(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "pos", "open") {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var request openRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid session request")
		return
	}
	if request.OpeningCashMinor < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "opening cash cannot be negative")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	deviceID, err := uuid.Parse(claims.DeviceID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var existing string
	err = tx.QueryRow(ctx, `
		SELECT id FROM register_sessions
		WHERE tenant_id = $1 AND device_id = $2 AND status = 'open'`,
		tenantID, deviceID).Scan(&existing)
	if err == nil {
		writeError(c, http.StatusConflict, "session_already_open", "a session is already open for this terminal")
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to check open session")
		return
	}

	sessionID := uuid.New()
	var name string
	var openedAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO register_sessions (id, tenant_id, device_id, user_id, opening_cash_minor)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING (SELECT display_name FROM users u WHERE u.tenant_id = register_sessions.tenant_id AND u.id = register_sessions.user_id), opened_at::text`,
		sessionID, tenantID, deviceID, userID, request.OpeningCashMinor).Scan(&name, &openedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to open session")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to open session")
		return
	}

	summary := buildSummary(accountTotals{}, request.OpeningCashMinor)
	c.JSON(http.StatusCreated, gin.H{
		"data": RegisterSession{
			ID:               sessionID.String(),
			Status:           "open",
			OpeningCashMinor: request.OpeningCashMinor,
			OpenedAt:         openedAt,
			OpenedBy:         name,
			Summary:          summary,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) close(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "pos", "close") {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	sessionID := c.Param("id")
	if _, err := uuid.Parse(sessionID); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid session id")
		return
	}
	var request closeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid close request")
		return
	}
	if request.ClosingCashMinor != nil && *request.ClosingCashMinor < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "closing cash cannot be negative")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var row sessionRow
	err = tx.QueryRow(ctx, `
		SELECT rs.id, rs.status, rs.opening_cash_minor, rs.closing_cash_minor,
		       rs.expected_cash_minor, rs.cash_difference_minor, rs.opened_at::text,
		       rs.closed_at::text, u.display_name
		FROM register_sessions rs
		JOIN users u ON u.tenant_id = rs.tenant_id AND u.id = rs.user_id
		WHERE rs.id = $1
		FOR UPDATE OF rs`, sessionID).
		Scan(&row.id, &row.status, &row.openingCashMinor, &row.closingCashMinor,
			&row.expectedCashMinor, &row.cashDifferenceMinor, &row.openedAt, &row.closedAt, &row.openedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "session_not_found", "session not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load session")
		return
	}
	if row.status != "open" {
		writeError(c, http.StatusConflict, "session_not_open", "session is already closed")
		return
	}

	// ma_pos_customization parity: "Session Close: Manager Required". When the
	// tenant enables pos.manager.close, closing a register needs a valid
	// manager PIN (owners and saas_admin always bypass).
	var requireManagerClose bool
	if err = tx.QueryRow(ctx, `
		SELECT COALESCE((SELECT value FROM tenant_settings
		                 WHERE tenant_id = $1 AND key = 'pos.manager.close'), 'false') = 'true'`,
		tenantID).Scan(&requireManagerClose); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to read session policy")
		return
	}
	if requireManagerClose && claims.Role != "owner" && claims.Role != "saas_admin" {
		var pinHash string
		if err = tx.QueryRow(ctx, `SELECT COALESCE(manager_pin_hash, '') FROM users WHERE id = $1::uuid`, claims.UserID).Scan(&pinHash); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify manager PIN")
			return
		}
		if pinHash == "" || !security.CheckPassword(pinHash, request.ManagerPIN) {
			writeError(c, http.StatusForbidden, "manager_pin_required", "a valid manager PIN is required to close the session")
			return
		}
	}

	totals, err := queryTotals(ctx, tx, row.id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally session")
		return
	}
	expected := expectedCashMinor(row.openingCashMinor, totals.cashMinor)
	var closingCash, difference *int64
	if request.ClosingCashMinor != nil {
		closingCash = request.ClosingCashMinor
		diff := balanceDifference(*closingCash, expected)
		difference = &diff
	}

	var closedAt string
	err = tx.QueryRow(ctx, `
		UPDATE register_sessions
		SET status = 'closed', closed_at = NOW(),
		    closing_cash_minor = $2, expected_cash_minor = $3, cash_difference_minor = $4
		WHERE id = $1
		RETURNING closed_at::text`, row.id, closingCash, &expected, difference).Scan(&closedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to close session")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to close session")
		return
	}

	summary := buildSummary(totals, row.openingCashMinor)
	c.JSON(http.StatusOK, gin.H{
		"data": RegisterSession{
			ID:                  row.id,
			Status:              "closed",
			OpeningCashMinor:    row.openingCashMinor,
			ClosingCashMinor:    closingCash,
			ExpectedCashMinor:   &expected,
			CashDifferenceMinor: difference,
			OpenedAt:            row.openedAt,
			ClosedAt:            &closedAt,
			OpenedBy:            row.openedBy,
			Summary:             summary,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) list(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "pos", "read") {
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := int64((page - 1) * limit)

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var total int64
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM register_sessions`).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count sessions")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT rs.id, rs.status, rs.opening_cash_minor, rs.closing_cash_minor,
		       rs.expected_cash_minor, rs.cash_difference_minor, rs.opened_at::text,
		       rs.closed_at::text, u.display_name,
		       COUNT(s.id)::bigint, COALESCE(SUM(s.total_minor), 0)
		FROM register_sessions rs
		JOIN users u ON u.tenant_id = rs.tenant_id AND u.id = rs.user_id
		LEFT JOIN sales s ON s.tenant_id = rs.tenant_id AND s.register_session_id = rs.id
		GROUP BY rs.id, u.display_name
		ORDER BY rs.opened_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sessions")
		return
	}
	defer rows.Close()

	type listItem struct {
		ID                  string  `json:"id"`
		Status              string  `json:"status"`
		OpeningCashMinor    int64   `json:"opening_cash_minor"`
		ClosingCashMinor    *int64  `json:"closing_cash_minor,omitempty"`
		ExpectedCashMinor   *int64  `json:"expected_cash_minor,omitempty"`
		CashDifferenceMinor *int64  `json:"cash_difference_minor,omitempty"`
		OpenedAt            string  `json:"opened_at"`
		ClosedAt            *string `json:"closed_at,omitempty"`
		OpenedBy            string  `json:"opened_by"`
		SalesCount          int64   `json:"sales_count"`
		TotalMinor          int64   `json:"total_minor"`
	}
	items := make([]listItem, 0)
	for rows.Next() {
		var item listItem
		if err := rows.Scan(&item.ID, &item.Status, &item.OpeningCashMinor, &item.ClosingCashMinor,
			&item.ExpectedCashMinor, &item.CashDifferenceMinor, &item.OpenedAt, &item.ClosedAt,
			&item.OpenedBy, &item.SalesCount, &item.TotalMinor); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sessions")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sessions")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load sessions")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{
			"request_id": c.GetString("request_id"),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
}

func (h *Handler) get(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "pos", "read") {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	sessionID := c.Param("id")
	if _, err := uuid.Parse(sessionID); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid session id")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var row sessionRow
	err = tx.QueryRow(ctx, `
		SELECT rs.id, rs.status, rs.opening_cash_minor, rs.closing_cash_minor,
		       rs.expected_cash_minor, rs.cash_difference_minor, rs.opened_at::text,
		       rs.closed_at::text, u.display_name
		FROM register_sessions rs
		JOIN users u ON u.tenant_id = rs.tenant_id AND u.id = rs.user_id
		WHERE rs.id = $1`, sessionID).
		Scan(&row.id, &row.status, &row.openingCashMinor, &row.closingCashMinor,
			&row.expectedCashMinor, &row.cashDifferenceMinor, &row.openedAt, &row.closedAt, &row.openedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "session_not_found", "session not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load session")
		return
	}

	totals, err := queryTotals(ctx, tx, row.id)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to tally session")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load session")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": toSession(row, buildSummary(totals, row.openingCashMinor)),
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) can(c *gin.Context, role, resource, action string) bool {
	if !httptransport.HasPermission(role, resource, action) {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return false
	}
	return true
}

func (h *Handler) authenticate(c *gin.Context) (security.Claims, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return security.Claims{}, false
	}
	claims, err := h.tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), security.AccessToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, false
	}
	return claims, true
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}

func toSession(row sessionRow, summary SessionSummary) RegisterSession {
	return RegisterSession{
		ID:                  row.id,
		Status:              row.status,
		OpeningCashMinor:    row.openingCashMinor,
		ClosingCashMinor:    row.closingCashMinor,
		ExpectedCashMinor:   row.expectedCashMinor,
		CashDifferenceMinor: row.cashDifferenceMinor,
		OpenedAt:            row.openedAt,
		ClosedAt:            row.closedAt,
		OpenedBy:            row.openedBy,
		Summary:             summary,
	}
}
