package community

import (
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// BadgeDefinition is the public wire shape of a tenant badge catalog entry.
type BadgeDefinition struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Criteria    any    `json:"criteria"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
	EarnedCount int    `json:"earned_count"`
}

// UserBadge is the public wire shape of one badge awarded to one user.
type UserBadge struct {
	BadgeID       string `json:"badge_id"`
	Code          string `json:"code"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Icon          string `json:"icon"`
	Note          string `json:"note"`
	AwardedAt     string `json:"awarded_at"`
	AwardedByName string `json:"awarded_by_name"`
}

type createBadgeRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Criteria    any    `json:"criteria"`
}

type updateBadgeRequest struct {
	Code        *string `json:"code"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
	Criteria    *any    `json:"criteria"`
	IsActive    *bool   `json:"is_active"`
}

type awardBadgeRequest struct {
	BadgeID string `json:"badge_id" binding:"required"`
	Note    string `json:"note"`
}

// badgeCodePattern is the kebab-case shape of badge_definitions.code:
// 2-60 chars, lowercase alnum + dashes, no leading/trailing dash.
var badgeCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,58}[a-z0-9]$`)

// validateBadgeCode reports whether code is a valid per-tenant badge slug.
func validateBadgeCode(code string) bool {
	return badgeCodePattern.MatchString(code)
}

func (h *Handler) listBadges(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	page, limit := pageLimits(c)
	q := strings.TrimSpace(c.Query("q"))
	// The manage screen needs the full catalog; the default view is active-only.
	includeInactive := c.Query("all") == "1" || c.Query("all") == "true"

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	where := "TRUE"
	args := []interface{}{}
	next := 1
	if !includeInactive {
		where += " AND is_active = TRUE"
	}
	if q != "" {
		where += " AND (name ILIKE $" + strconv.Itoa(next) + " OR code ILIKE $" + strconv.Itoa(next) + ")"
		args = append(args, "%"+q+"%")
		next++
	}
	var total int64
	if err = tx.QueryRow(ctx, "SELECT COUNT(*) FROM badge_definitions WHERE "+where, args...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count badges")
		return
	}

	listWhere := where
	listArgs := append(append([]interface{}{}, args...), limit, (page-1)*limit)
	listWhere += " ORDER BY name LIMIT $" + strconv.Itoa(next) + " OFFSET $" + strconv.Itoa(next+1)
	rows, err := tx.Query(ctx, `
		SELECT b.id::text, b.code, b.name, b.description, b.icon, b.criteria, b.is_active,
			b.created_at::text,
			(SELECT COUNT(*) FROM user_badges ub WHERE ub.badge_id = b.id)::int
		FROM badge_definitions b
		WHERE `+listWhere, listArgs...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
		return
	}
	defer rows.Close()

	items := make([]BadgeDefinition, 0)
	for rows.Next() {
		var item BadgeDefinition
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Description, &item.Icon,
			&item.Criteria, &item.IsActive, &item.CreatedAt, &item.EarnedCount); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"request_id": c.GetString("request_id"), "page": page, "limit": limit, "total": total},
	})
}

func (h *Handler) createBadge(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	var request createBadgeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid badge request")
		return
	}
	if err := validateBadgePayload(c, request.Code, request.Name, request.Description, request.Icon); err != nil {
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	criteria := request.Criteria
	if criteria == nil {
		criteria = map[string]any{}
	}
	badgeID := uuid.New()
	var createdAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO badge_definitions (id, tenant_id, code, name, description, icon, criteria)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at::text`,
		badgeID, claims.TenantID, request.Code, request.Name,
		strings.TrimSpace(request.Description), strings.TrimSpace(request.Icon), criteria).Scan(&createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(c, http.StatusConflict, "code_taken", "badge code already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save badge")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save badge")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": BadgeDefinition{
			ID: badgeID.String(), Code: request.Code, Name: request.Name,
			Description: strings.TrimSpace(request.Description), Icon: strings.TrimSpace(request.Icon),
			Criteria: criteria, IsActive: true, CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) updateBadge(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	badgeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid badge id")
		return
	}
	var request updateBadgeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid badge request")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	var current BadgeDefinition
	err = tx.QueryRow(ctx, `
		SELECT id::text, code, name, description, icon, criteria, is_active, created_at::text
		FROM badge_definitions WHERE id = $1`, badgeID).Scan(
		&current.ID, &current.Code, &current.Name, &current.Description, &current.Icon,
		&current.Criteria, &current.IsActive, &current.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "badge_not_found", "badge not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badge")
		return
	}
	if request.Code != nil {
		current.Code = strings.TrimSpace(*request.Code)
	}
	if request.Name != nil {
		current.Name = strings.TrimSpace(*request.Name)
	}
	if request.Description != nil {
		current.Description = strings.TrimSpace(*request.Description)
	}
	if request.Icon != nil {
		current.Icon = strings.TrimSpace(*request.Icon)
	}
	if request.Criteria != nil {
		current.Criteria = *request.Criteria
	}
	if request.IsActive != nil {
		current.IsActive = *request.IsActive
	}
	if err := validateBadgePayload(c, current.Code, current.Name, current.Description, current.Icon); err != nil {
		return
	}

	err = tx.QueryRow(ctx, `
		UPDATE badge_definitions SET code = $1, name = $2, description = $3, icon = $4,
			criteria = $5, is_active = $6
		WHERE id = $7
		RETURNING created_at::text`,
		current.Code, current.Name, current.Description, current.Icon,
		current.Criteria, current.IsActive, badgeID).Scan(&current.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(c, http.StatusConflict, "code_taken", "badge code already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update badge")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update badge")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": current,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// validateBadgePayload checks the shared badge field rules and writes the
// 400 itself when they fail (returns the gin error for caller convenience).
func validateBadgePayload(c *gin.Context, code, name, description, icon string) error {
	if !validateBadgeCode(code) {
		err := errors.New("invalid code")
		writeError(c, http.StatusBadRequest, "validation_error", "code must be kebab-case (a-z, 0-9, dashes)")
		return err
	}
	if strings.TrimSpace(name) == "" || len(name) > 100 {
		err := errors.New("invalid name")
		writeError(c, http.StatusBadRequest, "validation_error", "name is required and must be at most 100 characters")
		return err
	}
	if len(description) > 500 || len(icon) > 60 {
		err := errors.New("invalid fields")
		writeError(c, http.StatusBadRequest, "validation_error", "badge fields are too long")
		return err
	}
	return nil
}

func (h *Handler) listUserBadges(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var userExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify user")
		return
	}
	if !userExists {
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT ub.badge_id::text, b.code, b.name, b.description, b.icon, ub.note,
			ub.awarded_at::text, COALESCE(u.display_name, '')
		FROM user_badges ub
		JOIN badge_definitions b ON b.id = ub.badge_id
		LEFT JOIN users u ON u.id = ub.awarded_by
		WHERE ub.user_id = $1
		ORDER BY ub.awarded_at DESC`, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
		return
	}
	defer rows.Close()

	items := make([]UserBadge, 0)
	for rows.Next() {
		var item UserBadge
		if err := rows.Scan(&item.BadgeID, &item.Code, &item.Name, &item.Description, &item.Icon,
			&item.Note, &item.AwardedAt, &item.AwardedByName); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
			return
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badges")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"request_id": c.GetString("request_id"), "total": len(items)},
	})
}

func (h *Handler) awardBadge(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	var request awardBadgeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid award request")
		return
	}
	badgeID, err := uuid.Parse(request.BadgeID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid badge id")
		return
	}
	if len(request.Note) > 500 {
		writeError(c, http.StatusBadRequest, "validation_error", "note is too long")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	var userExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&userExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify user")
		return
	}
	if !userExists {
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}
	var badgeExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM badge_definitions WHERE id = $1)`, badgeID).Scan(&badgeExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify badge")
		return
	}
	if !badgeExists {
		writeError(c, http.StatusNotFound, "badge_not_found", "badge not found")
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_badges (user_id, badge_id, tenant_id, awarded_by, note)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, badgeID, claims.TenantID, claims.UserID, strings.TrimSpace(request.Note))
	if err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			writeError(c, http.StatusConflict, "already_awarded", "badge already awarded to this user")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to award badge")
		return
	}

	var awarded UserBadge
	if err = tx.QueryRow(ctx, `
		SELECT ub.badge_id::text, b.code, b.name, b.description, b.icon, ub.note,
			ub.awarded_at::text, COALESCE(u.display_name, '')
		FROM user_badges ub
		JOIN badge_definitions b ON b.id = ub.badge_id
		LEFT JOIN users u ON u.id = ub.awarded_by
		WHERE ub.user_id = $1 AND ub.badge_id = $2`,
		userID, badgeID).Scan(&awarded.BadgeID, &awarded.Code, &awarded.Name, &awarded.Description,
		&awarded.Icon, &awarded.Note, &awarded.AwardedAt, &awarded.AwardedByName); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load badge")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to award badge")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": awarded,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) revokeBadge(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	badgeID, err := uuid.Parse(c.Param("badgeId"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid badge id")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}
	tag, err := tx.Exec(ctx, `DELETE FROM user_badges WHERE user_id = $1 AND badge_id = $2`, userID, badgeID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke badge")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "badge_not_awarded", "badge is not awarded to this user")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to revoke badge")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}
