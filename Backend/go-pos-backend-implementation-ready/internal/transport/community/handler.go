package community

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

// Handler serves the tenant-scoped community workstream (slice 1: staff
// profiles + companies; slices 2-4 add jobs, badges/endorsements, forum in
// the same package). See docs/23_COMMUNITY_JOBS_PROFILES.md.
type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	group := router.Group("/community")

	group.GET("/profiles", h.listProfiles)
	group.GET("/profiles/me", h.getOwnProfile)
	group.PUT("/profiles/me", h.upsertOwnProfile)
	group.GET("/profiles/:id", h.getProfile)

	group.GET("/companies", h.listCompanies)
	group.POST("/companies", h.createCompany)
	group.GET("/companies/:id", h.getCompany)
	group.PATCH("/companies/:id", h.updateCompany)
	group.DELETE("/companies/:id", h.deleteCompany)
	group.POST("/companies/:id/members", h.addMember)
	group.DELETE("/companies/:id/members/:userId", h.removeMember)
}

// StaffProfile is the public wire shape of one staff member's profile.
type StaffProfile struct {
	UserID          string   `json:"user_id"`
	DisplayName     string   `json:"display_name"`
	Email           string   `json:"email"`
	Headline        string   `json:"headline"`
	Bio             string   `json:"bio"`
	AvatarURL       string   `json:"avatar_url"`
	Location        string   `json:"location"`
	YearsExperience int16    `json:"years_experience"`
	IsChief         bool     `json:"is_chief"`
	Skills          []string `json:"skills"`
	Resume          any      `json:"resume"`
	HasProfile      bool     `json:"has_profile"`
}

type Company struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Industry    string `json:"industry"`
	Website     string `json:"website"`
	LogoURL     string `json:"logo_url"`
	City        string `json:"city"`
	IsActive    bool   `json:"is_active"`
	CreatedAt   string `json:"created_at"`
}

type CompanyMember struct {
	ID        string `json:"id"`
	CompanyID string `json:"company_id"`
	UserID    string `json:"user_id"`
	Name      string `json:"display_name"`
	Role      string `json:"role"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
}

type upsertProfileRequest struct {
	Headline        string   `json:"headline"`
	Bio             string   `json:"bio"`
	AvatarURL       string   `json:"avatar_url"`
	Location        string   `json:"location"`
	YearsExperience int16    `json:"years_experience"`
	IsChief         bool     `json:"is_chief"`
	Skills          []string `json:"skills"`
	Resume          any      `json:"resume"`
}

type createCompanyRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Industry    string `json:"industry"`
	Website     string `json:"website"`
	LogoURL     string `json:"logo_url"`
	City        string `json:"city"`
}

type updateCompanyRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
	Industry    *string `json:"industry"`
	Website     *string `json:"website"`
	LogoURL     *string `json:"logo_url"`
	City        *string `json:"city"`
	IsActive    *bool   `json:"is_active"`
}

type addMemberRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Role   string `json:"role"`
	Title  string `json:"title"`
}

func (h *Handler) listProfiles(c *gin.Context) {
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
	q := strings.TrimSpace(c.Query("q"))
	role := strings.TrimSpace(c.Query("role"))

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

	where := "u.is_active = TRUE"
	countWhere := where
	args := []interface{}{limit, offset}
	next := 3
	if q != "" {
		pattern := "%" + q + "%"
		where += " AND (u.display_name ILIKE $" + strconv.Itoa(next) +
			" OR p.headline ILIKE $" + strconv.Itoa(next) +
			" OR EXISTS (SELECT 1 FROM unnest(p.skills) s WHERE s ILIKE $" + strconv.Itoa(next) + "))"
		countWhere = where
		args = append(args, pattern)
		next++
	}
	if role == "chief" {
		where += " AND p.user_id IS NOT NULL AND p.is_chief = TRUE"
		countWhere = where
	}

	var total int64
	countSQL := `SELECT COUNT(*) FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id WHERE ` + countWhere
	if err = tx.QueryRow(ctx, countSQL, args[2:]...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count profiles")
		return
	}

	sql := `SELECT u.id::text, u.display_name, u.email, COALESCE(p.headline, ''),
		COALESCE(p.bio, ''), COALESCE(p.avatar_url, ''), COALESCE(p.location, ''),
		COALESCE(p.years_experience, 0), COALESCE(p.is_chief, FALSE),
		COALESCE(p.skills, '{}'), COALESCE(p.resume, '{}'::jsonb), (p.user_id IS NOT NULL)
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id
		WHERE ` + where + ` ORDER BY u.display_name LIMIT $1 OFFSET $2`
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profiles")
		return
	}
	defer rows.Close()

	items := make([]StaffProfile, 0)
	for rows.Next() {
		var item StaffProfile
		if err := rows.Scan(&item.UserID, &item.DisplayName, &item.Email, &item.Headline,
			&item.Bio, &item.AvatarURL, &item.Location, &item.YearsExperience, &item.IsChief,
			&item.Skills, &item.Resume, &item.HasProfile); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profiles")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profiles")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profiles")
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

func (h *Handler) getOwnProfile(c *gin.Context) {
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
	item, err := loadProfile(ctx, tx, claims.UserID, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profile")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profile")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": item,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) getProfile(c *gin.Context) {
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
	item, err := loadProfile(ctx, tx, userID.String(), tenantID.String())
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "profile_not_found", "profile not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profile")
		return
	}
	// Company memberships enrich the detail payload.
	memberships, err := loadMemberships(ctx, tx, userID.String())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load memberships")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load profile")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":        item,
		"memberships": memberships,
		"meta":        gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) upsertOwnProfile(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var request upsertProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid profile request")
		return
	}
	request.Headline = strings.TrimSpace(request.Headline)
	request.Bio = strings.TrimSpace(request.Bio)
	request.AvatarURL = strings.TrimSpace(request.AvatarURL)
	request.Location = strings.TrimSpace(request.Location)
	if len(request.Headline) > 140 || len(request.Bio) > 2000 || len(request.AvatarURL) > 500 || len(request.Location) > 120 {
		writeError(c, http.StatusBadRequest, "validation_error", "profile fields are too long")
		return
	}
	if request.YearsExperience < 0 || request.YearsExperience > 100 {
		writeError(c, http.StatusBadRequest, "validation_error", "years_experience is out of range")
		return
	}
	skills := normalizeSkills(request.Skills)
	if len(skills) > 50 {
		writeError(c, http.StatusBadRequest, "validation_error", "too many skills")
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
	resume := request.Resume
	if resume == nil {
		resume = map[string]any{}
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

	// Caller must be a real user of this tenant.
	var userExists bool
	if err = tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND tenant_id = $2)`,
		claims.UserID, tenantID).Scan(&userExists); err != nil || !userExists {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_profiles (user_id, tenant_id, headline, bio, avatar_url, location,
				years_experience, is_chief, skills, resume, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			headline = EXCLUDED.headline, bio = EXCLUDED.bio, avatar_url = EXCLUDED.avatar_url,
			location = EXCLUDED.location, years_experience = EXCLUDED.years_experience,
			is_chief = EXCLUDED.is_chief, skills = EXCLUDED.skills, resume = EXCLUDED.resume,
			updated_at = NOW()`,
		claims.UserID, tenantID, request.Headline, request.Bio, request.AvatarURL, request.Location,
		request.YearsExperience, request.IsChief, skills, resume)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save profile")
		return
	}
	item, err := loadProfile(ctx, tx, claims.UserID, tenantID.String())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save profile")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save profile")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": item,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) listCompanies(c *gin.Context) {
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

	where := "c.is_active = TRUE"
	args := []interface{}{limit, int64((page - 1) * limit)}
	if q != "" {
		where += " AND c.name ILIKE $3"
		args = append(args, "%"+q+"%")
	}
	var total int64
	if err = tx.QueryRow(ctx, `SELECT COUNT(*) FROM companies c WHERE `+where, args[2:]...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count companies")
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT c.id::text, c.name, c.slug, c.description, c.industry, c.website,
			c.logo_url, c.city, c.is_active, c.created_at::text, COUNT(m.id)::int
		FROM companies c LEFT JOIN company_members m ON m.company_id = c.id AND m.is_active = TRUE
		WHERE `+where+` GROUP BY c.id ORDER BY c.name LIMIT $1 OFFSET $2`, args...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load companies")
		return
	}
	defer rows.Close()

	type listItem struct {
		Company
		MembersCount int `json:"members_count"`
	}
	items := make([]listItem, 0)
	for rows.Next() {
		var item listItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Slug, &item.Description, &item.Industry,
			&item.Website, &item.LogoURL, &item.City, &item.IsActive, &item.CreatedAt, &item.MembersCount); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load companies")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load companies")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load companies")
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

func (h *Handler) createCompany(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	var request createCompanyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid company request")
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name is required")
		return
	}
	if len(request.Name) > 160 || len(request.Slug) > 100 || len(request.Description) > 2000 ||
		len(request.Industry) > 100 || len(request.Website) > 500 || len(request.LogoURL) > 500 || len(request.City) > 120 {
		writeError(c, http.StatusBadRequest, "validation_error", "company fields are too long")
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

	companyID := uuid.New()
	var createdAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO companies (id, tenant_id, name, slug, description, industry, website, logo_url, city)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at::text`,
		companyID, claims.TenantID, request.Name, strings.TrimSpace(request.Slug),
		strings.TrimSpace(request.Description), strings.TrimSpace(request.Industry),
		strings.TrimSpace(request.Website), strings.TrimSpace(request.LogoURL), strings.TrimSpace(request.City)).Scan(&createdAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save company")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save company")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": Company{
			ID: companyID.String(), Name: request.Name, Slug: strings.TrimSpace(request.Slug),
			Description: strings.TrimSpace(request.Description), Industry: strings.TrimSpace(request.Industry),
			Website: strings.TrimSpace(request.Website), LogoURL: strings.TrimSpace(request.LogoURL),
			City: strings.TrimSpace(request.City), IsActive: true, CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) getCompany(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "read") {
		return
	}
	companyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid company id")
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

	var company Company
	err = tx.QueryRow(ctx, `
		SELECT id::text, name, slug, description, industry, website, logo_url, city, is_active, created_at::text
		FROM companies WHERE id = $1`, companyID).Scan(
		&company.ID, &company.Name, &company.Slug, &company.Description, &company.Industry,
		&company.Website, &company.LogoURL, &company.City, &company.IsActive, &company.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "company_not_found", "company not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load company")
		return
	}

	members, err := loadCompanyMembers(ctx, tx, companyID.String())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load company members")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load company")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":    company,
		"members": members,
		"meta":    gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) updateCompany(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	var request updateCompanyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid company request")
		return
	}
	companyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid company id")
		return
	}
	if request.Name != nil && strings.TrimSpace(*request.Name) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "name is required")
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

	name, slug, description, industry, website, logoURL, city, isActive := "", "", "", "", "", "", "", true
	err = tx.QueryRow(ctx, `
		SELECT name, slug, description, industry, website, logo_url, city, is_active
		FROM companies WHERE id = $1`, companyID).Scan(&name, &slug, &description, &industry, &website, &logoURL, &city, &isActive)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "company_not_found", "company not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load company")
		return
	}
	if request.Name != nil {
		name = strings.TrimSpace(*request.Name)
	}
	if request.Slug != nil {
		slug = strings.TrimSpace(*request.Slug)
	}
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	if request.Industry != nil {
		industry = strings.TrimSpace(*request.Industry)
	}
	if request.Website != nil {
		website = strings.TrimSpace(*request.Website)
	}
	if request.LogoURL != nil {
		logoURL = strings.TrimSpace(*request.LogoURL)
	}
	if request.City != nil {
		city = strings.TrimSpace(*request.City)
	}
	if request.IsActive != nil {
		isActive = *request.IsActive
	}

	row := tx.QueryRow(ctx, `
		UPDATE companies SET name = $1, slug = $2, description = $3, industry = $4,
			website = $5, logo_url = $6, city = $7, is_active = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING id::text, name, slug, description, industry, website, logo_url, city, is_active, created_at::text`,
		name, slug, description, industry, website, logoURL, city, isActive, companyID)
	var company Company
	if err := row.Scan(&company.ID, &company.Name, &company.Slug, &company.Description, &company.Industry,
		&company.Website, &company.LogoURL, &company.City, &company.IsActive, &company.CreatedAt); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update company")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update company")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": company,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) deleteCompany(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	companyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid company id")
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
	tag, err := tx.Exec(ctx, `UPDATE companies SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, companyID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete company")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "company_not_found", "company not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete company")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) addMember(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	var request addMemberRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid member request")
		return
	}
	companyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid company id")
		return
	}
	userID, err := uuid.Parse(request.UserID)
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}
	role := strings.TrimSpace(request.Role)
	if role == "" {
		role = "staff"
	}
	if role != "owner" && role != "manager" && role != "staff" {
		writeError(c, http.StatusBadRequest, "validation_error", "role must be owner, manager or staff")
		return
	}
	if len(request.Title) > 160 {
		writeError(c, http.StatusBadRequest, "validation_error", "title is too long")
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

	var companyExists, userExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`, companyID).Scan(&companyExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify company")
		return
	}
	if !companyExists {
		writeError(c, http.StatusNotFound, "company_not_found", "company not found")
		return
	}
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1 AND tenant_id = $2)`, userID, claims.TenantID).Scan(&userExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify user")
		return
	}
	if !userExists {
		writeError(c, http.StatusNotFound, "user_not_found", "user not found")
		return
	}

	var member CompanyMember
	member.CompanyID = companyID.String()
	member.UserID = userID.String()
	member.Role = role
	member.Title = strings.TrimSpace(request.Title)
	err = tx.QueryRow(ctx, `
		INSERT INTO company_members (id, tenant_id, company_id, user_id, role, title)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, created_at::text`,
		uuid.New(), claims.TenantID, companyID, userID, role, member.Title).Scan(&member.ID, &member.CreatedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to add member")
		return
	}
	_ = tx.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, userID).Scan(&member.Name)
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to add member")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": member,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) removeMember(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "community", "write") {
		return
	}
	companyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid company id")
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
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
	tag, err := tx.Exec(ctx, `
		UPDATE company_members SET is_active = FALSE
		WHERE company_id = $1 AND user_id = $2`, companyID, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to remove member")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "member_not_found", "member not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to remove member")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
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

func pageLimits(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return page, limit
}
