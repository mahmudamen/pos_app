package community

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// JobOffer is the public wire shape of one job posting.
type JobOffer struct {
	ID             string   `json:"id"`
	CompanyID      string   `json:"company_id"`
	CompanyName    string   `json:"company_name"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	EmploymentType string   `json:"employment_type"`
	Location       string   `json:"location"`
	SalaryMinor    int64    `json:"salary_minor"`
	SalaryCurrency string   `json:"salary_currency"`
	SkillTags      []string `json:"skill_tags"`
	IsActive       bool     `json:"is_active"`
	ClosesAt       string   `json:"closes_at"`
	CreatedBy      string   `json:"created_by"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	Applied        bool     `json:"applied"`
	Applications   int      `json:"applications"`
}

// JobApplication is the public wire shape of one application.
type JobApplication struct {
	ID          string `json:"id"`
	JobOfferID  string `json:"job_offer_id"`
	JobTitle    string `json:"job_title"`
	ApplicantID string `json:"applicant_id"`
	Name        string `json:"display_name"`
	Headline    string `json:"headline"`
	CoverNote   string `json:"cover_note"`
	Resume      any    `json:"resume"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

type createJobRequest struct {
	CompanyID      *string  `json:"company_id"`
	Title          string   `json:"title" binding:"required"`
	Description    string   `json:"description"`
	EmploymentType string   `json:"employment_type"`
	Location       string   `json:"location"`
	SalaryMinor    int64    `json:"salary_minor"`
	SalaryCurrency string   `json:"salary_currency"`
	SkillTags      []string `json:"skill_tags"`
	ClosesAt       *string  `json:"closes_at"`
}

type updateJobRequest struct {
	CompanyID      *string   `json:"company_id"`
	Title          *string   `json:"title"`
	Description    *string   `json:"description"`
	EmploymentType *string   `json:"employment_type"`
	Location       *string   `json:"location"`
	SalaryMinor    *int64    `json:"salary_minor"`
	SalaryCurrency *string   `json:"salary_currency"`
	SkillTags      *[]string `json:"skill_tags"`
	ClosesAt       *string   `json:"closes_at"`
	IsActive       *bool     `json:"is_active"`
}

type applyRequest struct {
	CoverNote string `json:"cover_note"`
}

type applicationStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

const (
	employmentTypeFullTime  = "full_time"
	employmentTypePartTime  = "part_time"
	employmentTypeContract  = "contract"
	employmentTypeFreelance = "freelance"
)

var validEmploymentTypes = map[string]bool{
	employmentTypeFullTime:  true,
	employmentTypePartTime:  true,
	employmentTypeContract:  true,
	employmentTypeFreelance: true,
}

var validApplicationStatuses = map[string]bool{
	"applied":      true,
	"under_review": true,
	"accepted":     true,
	"rejected":     true,
}

func (h *Handler) listJobs(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "read") {
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
	companyID := strings.TrimSpace(c.Query("company_id"))
	typ := strings.TrimSpace(c.Query("type"))
	mine := c.Query("mine") == "1" || c.Query("mine") == "true"

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

	where := "jo.is_active = TRUE"
	countWhere := where
	listArgs := []interface{}{limit, offset}
	countArgs := []interface{}{}

	listNext := 3
	countNext := 1
	if q != "" {
		pattern := "%" + q + "%"
		cond := " AND (jo.title ILIKE $" + strconv.Itoa(listNext) +
			" OR EXISTS (SELECT 1 FROM unnest(jo.skill_tags) s WHERE s ILIKE $" + strconv.Itoa(listNext) + "))"
		countCond := " AND (jo.title ILIKE $" + strconv.Itoa(countNext) +
			" OR EXISTS (SELECT 1 FROM unnest(jo.skill_tags) s WHERE s ILIKE $" + strconv.Itoa(countNext) + "))"
		where += cond
		countWhere += countCond
		listArgs = append(listArgs, pattern)
		countArgs = append(countArgs, pattern)
		listNext++
		countNext++
	}
	if companyID != "" {
		where += " AND jo.company_id = $" + strconv.Itoa(listNext)
		countWhere += " AND jo.company_id = $" + strconv.Itoa(countNext)
		listArgs = append(listArgs, companyID)
		countArgs = append(countArgs, companyID)
		listNext++
		countNext++
	}
	if typ != "" {
		where += " AND jo.employment_type = $" + strconv.Itoa(listNext)
		countWhere += " AND jo.employment_type = $" + strconv.Itoa(countNext)
		listArgs = append(listArgs, typ)
		countArgs = append(countArgs, typ)
		listNext++
		countNext++
	}
	if mine {
		where += " AND EXISTS (SELECT 1 FROM job_applications ja WHERE ja.job_offer_id = jo.id AND ja.applicant_id = $" + strconv.Itoa(listNext) + "::uuid)"
		countWhere += " AND EXISTS (SELECT 1 FROM job_applications ja WHERE ja.job_offer_id = jo.id AND ja.applicant_id = $" + strconv.Itoa(countNext) + "::uuid)"
		listArgs = append(listArgs, claims.UserID)
		countArgs = append(countArgs, claims.UserID)
		listNext++
		countNext++
	}

	var total int64
	countSQL := `SELECT COUNT(*) FROM job_offers jo WHERE ` + countWhere
	if err = tx.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count jobs")
		return
	}

	// The applied subquery needs the caller id as its own parameter.
	callerIdx := listNext
	listArgs = append(listArgs, claims.UserID)
	sql := `SELECT jo.id::text, COALESCE(jo.company_id::text, ''), COALESCE(c.name, ''),
		jo.title, jo.description, jo.employment_type, jo.location, jo.salary_minor,
		jo.salary_currency, jo.skill_tags, jo.is_active, COALESCE(jo.closes_at::text, ''),
		jo.created_by::text, jo.created_at::text, jo.updated_at::text,
		EXISTS (SELECT 1 FROM job_applications ja WHERE ja.job_offer_id = jo.id AND ja.applicant_id = $` + strconv.Itoa(callerIdx) + `::uuid),
		(SELECT COUNT(*) FROM job_applications ja WHERE ja.job_offer_id = jo.id)
		FROM job_offers jo
		LEFT JOIN companies c ON c.id = jo.company_id
		WHERE ` + where + ` ORDER BY jo.created_at DESC LIMIT $1 OFFSET $2`
	rows, err := tx.Query(ctx, sql, listArgs...)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load jobs")
		return
	}
	defer rows.Close()

	items := make([]JobOffer, 0)
	for rows.Next() {
		var item JobOffer
		if err := rows.Scan(&item.ID, &item.CompanyID, &item.CompanyName, &item.Title, &item.Description,
			&item.EmploymentType, &item.Location, &item.SalaryMinor, &item.SalaryCurrency, &item.SkillTags,
			&item.IsActive, &item.ClosesAt, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
			&item.Applied, &item.Applications); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load jobs")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load jobs")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load jobs")
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

func (h *Handler) createJob(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "write") {
		return
	}
	var request createJobRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid job request")
		return
	}
	request.Title = strings.TrimSpace(request.Title)
	request.Description = strings.TrimSpace(request.Description)
	request.Location = strings.TrimSpace(request.Location)
	request.EmploymentType = strings.TrimSpace(request.EmploymentType)
	if request.EmploymentType == "" {
		request.EmploymentType = employmentTypeFullTime
	}
	if !validEmploymentTypes[request.EmploymentType] {
		writeError(c, http.StatusBadRequest, "validation_error", "employment_type must be full_time, part_time, contract or freelance")
		return
	}
	if request.Title == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "title is required")
		return
	}
	if len(request.Title) > 200 || len(request.Description) > 8000 || len(request.Location) > 200 {
		writeError(c, http.StatusBadRequest, "validation_error", "job fields are too long")
		return
	}
	if request.SalaryMinor < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "salary_minor cannot be negative")
		return
	}
	salaryCurrency := strings.ToUpper(strings.TrimSpace(request.SalaryCurrency))
	if salaryCurrency == "" {
		salaryCurrency = "EGP"
	}
	if len(salaryCurrency) != 3 {
		writeError(c, http.StatusBadRequest, "validation_error", "salary_currency must be a 3-letter code")
		return
	}
	skills := normalizeSkills(request.SkillTags)
	var closesAt any
	if request.ClosesAt != nil && strings.TrimSpace(*request.ClosesAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.ClosesAt))
		if err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "closes_at must be RFC3339")
			return
		}
		closesAt = parsed
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
	var companyID any
	companyName := ""
	if request.CompanyID != nil && strings.TrimSpace(*request.CompanyID) != "" {
		parsedID, err := uuid.Parse(strings.TrimSpace(*request.CompanyID))
		if err != nil {
			writeError(c, http.StatusBadRequest, "validation_error", "invalid company id")
			return
		}
		var companyExists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`, parsedID).Scan(&companyExists); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify company")
			return
		}
		if !companyExists {
			writeError(c, http.StatusNotFound, "company_not_found", "company not found")
			return
		}
		companyID = parsedID
		if err = tx.QueryRow(ctx, `SELECT name FROM companies WHERE id = $1`, parsedID).Scan(&companyName); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load company")
			return
		}
	}

	jobID := uuid.New()
	var createdAt, updatedAt string
	var companyIDString string
	if companyID != nil {
		companyIDString = companyID.(uuid.UUID).String()
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO job_offers (id, tenant_id, company_id, title, description, employment_type,
				location, salary_minor, salary_currency, skill_tags, closes_at, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING created_at::text, updated_at::text`,
		jobID, claims.TenantID, companyID, request.Title, request.Description, request.EmploymentType,
		request.Location, request.SalaryMinor, salaryCurrency, skills, closesAt, claims.UserID).Scan(&createdAt, &updatedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save job")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save job")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": JobOffer{
			ID: jobID.String(), CompanyID: companyIDString, CompanyName: companyName,
			Title: request.Title, Description: request.Description,
			EmploymentType: request.EmploymentType, Location: request.Location,
			SalaryMinor: request.SalaryMinor, SalaryCurrency: salaryCurrency, SkillTags: skills,
			IsActive: true, CreatedBy: claims.UserID, CreatedAt: createdAt, UpdatedAt: updatedAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) getJob(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "read") {
		return
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid job id")
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
	var item JobOffer
	err = tx.QueryRow(ctx, `
		SELECT jo.id::text, COALESCE(jo.company_id::text, ''), COALESCE(c.name, ''),
			jo.title, jo.description, jo.employment_type, jo.location, jo.salary_minor,
			jo.salary_currency, jo.skill_tags, jo.is_active, COALESCE(jo.closes_at::text, ''),
			jo.created_by::text, jo.created_at::text, jo.updated_at::text,
			EXISTS (SELECT 1 FROM job_applications ja WHERE ja.job_offer_id = jo.id AND ja.applicant_id = $2::uuid),
			(SELECT COUNT(*) FROM job_applications ja WHERE ja.job_offer_id = jo.id)
		FROM job_offers jo
		LEFT JOIN companies c ON c.id = jo.company_id
		WHERE jo.id = $1`, jobID, claims.UserID).Scan(
		&item.ID, &item.CompanyID, &item.CompanyName, &item.Title, &item.Description,
		&item.EmploymentType, &item.Location, &item.SalaryMinor, &item.SalaryCurrency, &item.SkillTags,
		&item.IsActive, &item.ClosesAt, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
		&item.Applied, &item.Applications)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "job_not_found", "job not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load job")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load job")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": item,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) updateJob(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "write") {
		return
	}
	var request updateJobRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid job request")
		return
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid job id")
		return
	}
	if request.Title != nil && strings.TrimSpace(*request.Title) == "" {
		writeError(c, http.StatusBadRequest, "validation_error", "title is required")
		return
	}
	if request.EmploymentType != nil && !validEmploymentTypes[*request.EmploymentType] {
		writeError(c, http.StatusBadRequest, "validation_error", "employment_type must be full_time, part_time, contract or freelance")
		return
	}
	if request.SalaryMinor != nil && *request.SalaryMinor < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "salary_minor cannot be negative")
		return
	}
	var closesAt any
	if request.ClosesAt != nil {
		if strings.TrimSpace(*request.ClosesAt) == "" {
			closesAt = nil
		} else {
			parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*request.ClosesAt))
			if err != nil {
				writeError(c, http.StatusBadRequest, "validation_error", "closes_at must be RFC3339")
				return
			}
			closesAt = parsed
		}
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

	title, description, employmentType, location, salaryCurrency := "", "", "", "", ""
	var salaryMinor int64
	skills := []string{}
	isActive := true
	var companyID any
	err = tx.QueryRow(ctx, `
		SELECT title, description, employment_type, location, salary_minor, salary_currency,
			skill_tags, is_active, company_id
		FROM job_offers WHERE id = $1`, jobID).Scan(
		&title, &description, &employmentType, &location, &salaryMinor, &salaryCurrency,
		&skills, &isActive, &companyID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "job_not_found", "job not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load job")
		return
	}
	if request.Title != nil {
		title = strings.TrimSpace(*request.Title)
	}
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	if request.EmploymentType != nil {
		employmentType = *request.EmploymentType
	}
	if request.Location != nil {
		location = strings.TrimSpace(*request.Location)
	}
	if request.SalaryMinor != nil {
		salaryMinor = *request.SalaryMinor
	}
	if request.SalaryCurrency != nil {
		salaryCurrency = strings.ToUpper(strings.TrimSpace(*request.SalaryCurrency))
		if len(salaryCurrency) != 3 {
			writeError(c, http.StatusBadRequest, "validation_error", "salary_currency must be a 3-letter code")
			return
		}
	}
	if request.SkillTags != nil {
		skills = normalizeSkills(*request.SkillTags)
	}
	if request.IsActive != nil {
		isActive = *request.IsActive
	}
	if request.CompanyID != nil {
		if strings.TrimSpace(*request.CompanyID) == "" {
			companyID = nil
		} else {
			parsedID, err := uuid.Parse(strings.TrimSpace(*request.CompanyID))
			if err != nil {
				writeError(c, http.StatusBadRequest, "validation_error", "invalid company id")
				return
			}
			var companyExists bool
			if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM companies WHERE id = $1)`, parsedID).Scan(&companyExists); err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify company")
				return
			}
			if !companyExists {
				writeError(c, http.StatusNotFound, "company_not_found", "company not found")
				return
			}
			companyID = parsedID
		}
	}

	var item JobOffer
	err = tx.QueryRow(ctx, `
		UPDATE job_offers SET title = $1, description = $2, employment_type = $3, location = $4,
			salary_minor = $5, salary_currency = $6, skill_tags = $7, is_active = $8,
			company_id = $9, closes_at = $10, updated_at = NOW()
		WHERE id = $11
		RETURNING id::text, created_by::text, created_at::text, updated_at::text`,
		title, description, employmentType, location, salaryMinor, salaryCurrency, skills,
		isActive, companyID, closesAt, jobID).Scan(&item.ID, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update job")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update job")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": item,
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) deleteJob(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "write") {
		return
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid job id")
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
	tag, err := tx.Exec(ctx, `UPDATE job_offers SET is_active = FALSE, updated_at = NOW() WHERE id = $1`, jobID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete job")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "job_not_found", "job not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to delete job")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}

func (h *Handler) applyToJob(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "read") {
		return
	}
	var request applyRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid application request")
		return
	}
	request.CoverNote = strings.TrimSpace(request.CoverNote)
	if len(request.CoverNote) > 2000 {
		writeError(c, http.StatusBadRequest, "validation_error", "cover_note is too long")
		return
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid job id")
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
	var jobExists bool
	if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM job_offers WHERE id = $1 AND is_active = TRUE)`, jobID).Scan(&jobExists); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify job")
		return
	}
	if !jobExists {
		writeError(c, http.StatusNotFound, "job_not_found", "job not found")
		return
	}
	// Snapshot the applicant's current resume.
	var resume any = map[string]any{}
	if err = tx.QueryRow(ctx, `SELECT resume FROM user_profiles WHERE user_id = $1`, claims.UserID).Scan(&resume); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load resume")
		return
	}
	appID := uuid.New()
	var createdAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO job_applications (id, tenant_id, job_offer_id, applicant_id, cover_note, resume)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at::text`,
		appID, claims.TenantID, jobID, claims.UserID, request.CoverNote, resume).Scan(&createdAt)
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			writeError(c, http.StatusConflict, "already_applied", "already applied to this job")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save application")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save application")
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data": JobApplication{
			ID: appID.String(), JobOfferID: jobID.String(), ApplicantID: claims.UserID,
			CoverNote: request.CoverNote, Resume: resume, Status: "applied", CreatedAt: createdAt,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func (h *Handler) listApplications(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "manage") {
		return
	}
	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid job id")
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
	var jobTitle string
	err = tx.QueryRow(ctx, `SELECT title FROM job_offers WHERE id = $1`, jobID).Scan(&jobTitle)
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(c, http.StatusNotFound, "job_not_found", "job not found")
		return
	}
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load job")
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT ja.id::text, ja.job_offer_id::text, jo.title, ja.applicant_id::text, u.display_name,
			COALESCE(p.headline, ''), ja.cover_note, COALESCE(ja.resume, '{}'::jsonb),
			ja.status, ja.created_at::text
		FROM job_applications ja
		JOIN job_offers jo ON jo.id = ja.job_offer_id
		JOIN users u ON u.id = ja.applicant_id
		LEFT JOIN user_profiles p ON p.user_id = ja.applicant_id
		WHERE ja.job_offer_id = $1
		ORDER BY ja.created_at DESC`, jobID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load applications")
		return
	}
	defer rows.Close()

	items := make([]JobApplication, 0)
	for rows.Next() {
		var item JobApplication
		if err := rows.Scan(&item.ID, &item.JobOfferID, &item.JobTitle, &item.ApplicantID, &item.Name,
			&item.Headline, &item.CoverNote, &item.Resume, &item.Status, &item.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load applications")
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load applications")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load applications")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{"request_id": c.GetString("request_id"), "applied_total": len(items)},
	})
}

func (h *Handler) updateApplicationStatus(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !h.can(c, claims.Role, "jobs", "manage") {
		return
	}
	var request applicationStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid application request")
		return
	}
	request.Status = strings.ToLower(strings.TrimSpace(request.Status))
	if !validApplicationStatuses[request.Status] {
		writeError(c, http.StatusBadRequest, "validation_error", "status must be applied, under_review, accepted or rejected")
		return
	}
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid application id")
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
	tag, err := tx.Exec(ctx, `UPDATE job_applications SET status = $1 WHERE id = $2`, request.Status, appID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update application")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusNotFound, "application_not_found", "application not found")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to update application")
		return
	}
	c.JSON(http.StatusOK, gin.H{"meta": gin.H{"request_id": c.GetString("request_id")}})
}
