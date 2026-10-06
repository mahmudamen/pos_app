package onboarding

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

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
	router.GET("/onboarding", h.state)
	router.GET("/onboarding/tours", h.list)
	router.GET("/onboarding/readiness", h.readinessHandler)
	router.POST("/onboarding/tours/:id/start", h.start)
	router.PATCH("/onboarding/tours/:id", h.updateStep)
	router.POST("/onboarding/tours/:id/pause", h.pause)
	router.POST("/onboarding/tours/:id/resume", h.resume)
	router.POST("/onboarding/tours/:id/skip", h.skip)
	router.POST("/onboarding/tours/:id/complete", h.complete)
	router.POST("/onboarding/events", h.event)
}

type progressRow struct {
	TourID      string
	Version     int
	Status      string
	CurrentStep string
	StartedAt   *time.Time
	PausedAt    *time.Time
	CompletedAt *time.Time
	UpdatedAt   *time.Time
}

func (r progressRow) tourJSON(def tourDef, goalMet bool) gin.H {
	out := gin.H{
		"id":           def.ID,
		"version":      def.Version,
		"goal":         def.Goal,
		"status":       r.Status,
		"current_step": r.CurrentStep,
		"step_count":   def.StepCount,
		"goal_met":     goalMet,
	}
	for field, value := range map[string]*time.Time{
		"started_at":   r.StartedAt,
		"paused_at":    r.PausedAt,
		"completed_at": r.CompletedAt,
		"updated_at":   r.UpdatedAt,
	} {
		if value != nil {
			out[field] = value.UTC().Format(time.RFC3339)
		}
	}
	return out
}

var (
	errBadStatus = errors.New("invalid_status")
	errTourDone  = errors.New("tour_completed")
)

// applyTransition is the only place a tour status may change. Illegal moves
// are rejected instead of silently coerced.
func applyTransition(status, action string) (string, error) {
	if status == "" {
		status = "not_started"
	}
	switch action {
	case "start":
		if status == "completed" {
			return status, errTourDone
		}
		return "running", nil
	case "pause":
		if status != "running" {
			return status, errBadStatus
		}
		return "paused", nil
	case "resume":
		if status != "paused" && status != "running" {
			return status, errBadStatus
		}
		return "running", nil
	case "skip":
		if status == "completed" {
			return status, errTourDone
		}
		return "skipped", nil
	case "complete":
		return "completed", nil
	}
	return status, errors.New("unknown_action")
}

var (
	stepPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,63}$`)
	eventPattern  = regexp.MustCompile(`^[a-z][a-z0-9_.]{0,63}$`)
	maxEventBytes = 8192
)

func validateStep(step string) error {
	if stepPattern.MatchString(step) {
		return nil
	}
	return errors.New("invalid_step")
}

func validateEvent(eventID, event string, payload json.RawMessage) error {
	if _, err := uuid.Parse(eventID); err != nil {
		return errors.New("invalid_event_id")
	}
	if !eventPattern.MatchString(event) {
		return errors.New("invalid_event")
	}
	if len(payload) > maxEventBytes {
		return errors.New("payload_too_large")
	}
	if len(payload) > 0 {
		var probe map[string]any
		if err := json.Unmarshal(payload, &probe); err != nil || probe == nil {
			return errors.New("invalid_payload")
		}
	}
	return nil
}

// recommendTour picks what the onboarding center should offer next: an
// in-flight tour first, then the earliest goal the user has neither finished
// nor declined (skipped tours stay restartable from the tour list, but a
// deliberate skip is not re-offered immediately).
func recommendTour(status map[string]string) string {
	for _, t := range catalog {
		if s := status[t.ID]; s == "running" || s == "paused" {
			return t.ID
		}
	}
	for _, t := range catalog {
		if s := status[t.ID]; s != "completed" && s != "skipped" {
			return t.ID
		}
	}
	return ""
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}

func writeData(c *gin.Context, status int, data any) {
	c.JSON(status, gin.H{"data": data, "meta": gin.H{"request_id": c.GetString("request_id")}})
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

// authorize is the shared preamble: token, RBAC, pool, tenant+user identity.
func (h *Handler) authorize(c *gin.Context, action string) (tenantID, userID string, ok bool) {
	claims, ok := h.authenticate(c)
	if !ok {
		return "", "", false
	}
	if !httptransport.HasPermission(claims.Role, "onboarding", action) {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return "", "", false
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return "", "", false
	}
	if _, err := uuid.Parse(claims.TenantID); err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return "", "", false
	}
	if _, err := uuid.Parse(claims.UserID); err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return "", "", false
	}
	return claims.TenantID, claims.UserID, true
}

// beginTx opens a transaction already pinned to the caller's tenant (FORCE
// RLS means no row is visible until app.current_tenant is set).
func (h *Handler) beginTx(c *gin.Context, tenantID string) (pgx.Tx, bool) {
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return nil, false
	}
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		_ = tx.Rollback(ctx)
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return nil, false
	}
	return tx, true
}

func (h *Handler) loadProgress(c *gin.Context, tx pgx.Tx, tenantID, userID string) (map[string]progressRow, error) {
	ctx := c.Request.Context()
	rows, err := tx.Query(ctx, `
		SELECT tour_id, tour_version, status, current_step,
		       started_at, paused_at, completed_at, updated_at
		FROM onboarding_progress
		WHERE tenant_id = $1::uuid AND user_id = $2::uuid`,
		tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]progressRow{}
	for rows.Next() {
		var r progressRow
		if err := rows.Scan(&r.TourID, &r.Version, &r.Status, &r.CurrentStep,
			&r.StartedAt, &r.PausedAt, &r.CompletedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		out[r.TourID] = r
	}
	return out, rows.Err()
}

func (h *Handler) toursJSON(rows map[string]progressRow, ready readiness) []gin.H {
	met := map[string]bool{
		GoalStoreConfigured:    ready.StoreConfigured,
		GoalProductReady:       ready.ProductReady,
		GoalTerminalConfigured: ready.TerminalConfigured,
		GoalCashSessionOpened:  ready.CashSessionOpened,
		GoalFirstSaleCompleted: ready.FirstSaleCompleted,
	}
	out := make([]gin.H, 0, len(catalog))
	for _, def := range catalog {
		row, ok := rows[def.ID]
		if !ok {
			row = progressRow{TourID: def.ID, Version: def.Version, Status: "not_started"}
		}
		out = append(out, row.tourJSON(def, met[def.Goal]))
	}
	return out
}

func statusMap(rows map[string]progressRow) map[string]string {
	out := make(map[string]string, len(catalog))
	for _, def := range catalog {
		out[def.ID] = "not_started"
		if r, ok := rows[def.ID]; ok {
			out[def.ID] = r.Status
		}
	}
	return out
}

// loadView is the shared read path: readiness + progress in one tx.
func (h *Handler) loadView(c *gin.Context, tenantID, userID string) ([]gin.H, readiness, map[string]string, bool) {
	tx, ok := h.beginTx(c, tenantID)
	if !ok {
		return nil, readiness{}, nil, false
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	ctx := c.Request.Context()
	ready, err := readinessInTx(ctx, tx, tenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load onboarding readiness")
		return nil, readiness{}, nil, false
	}
	rows, err := h.loadProgress(c, tx, tenantID, userID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load onboarding progress")
		return nil, readiness{}, nil, false
	}
	return h.toursJSON(rows, ready), ready, statusMap(rows), true
}

func (h *Handler) state(c *gin.Context) {
	tenantID, userID, ok := h.authorize(c, "read")
	if !ok {
		return
	}
	tours, ready, statuses, ok := h.loadView(c, tenantID, userID)
	if !ok {
		return
	}
	writeData(c, http.StatusOK, gin.H{
		"tours":       tours,
		"readiness":   ready,
		"recommended": recommendTour(statuses),
	})
}

func (h *Handler) list(c *gin.Context) {
	tenantID, userID, ok := h.authorize(c, "read")
	if !ok {
		return
	}
	tours, _, _, ok := h.loadView(c, tenantID, userID)
	if !ok {
		return
	}
	writeData(c, http.StatusOK, gin.H{"tours": tours})
}

func (h *Handler) readinessHandler(c *gin.Context) {
	tenantID, _, ok := h.authorize(c, "read")
	if !ok {
		return
	}
	tx, ok := h.beginTx(c, tenantID)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	ready, err := readinessInTx(c.Request.Context(), tx, tenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load onboarding readiness")
		return
	}
	writeData(c, http.StatusOK, ready)
}

type startRequest struct {
	TourVersion *int `json:"tour_version"`
}

// mutate is the shared body of start/pause/resume/skip/complete: lock the
// row, apply the transition, persist it. complete additionally re-verifies
// the business goal against the database.
func (h *Handler) mutate(c *gin.Context, action string) {
	tenantID, userID, ok := h.authorize(c, "write")
	if !ok {
		return
	}
	tourID := c.Param("id")
	def, found := lookupTour(tourID)
	if !found {
		writeError(c, http.StatusNotFound, "tour_not_found", "unknown tour")
		return
	}
	if action == "start" && c.Request.ContentLength != 0 {
		var req startRequest
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
			return
		}
		if req.TourVersion != nil && *req.TourVersion != def.Version {
			writeError(c, http.StatusConflict, "tour_version_mismatch", "tour definition is out of date")
			return
		}
	}
	tx, ok := h.beginTx(c, tenantID)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	ctx := c.Request.Context()

	var status, currentStep string
	err := tx.QueryRow(ctx, `
		SELECT status, current_step FROM onboarding_progress
		WHERE tenant_id = $1::uuid AND user_id = $2::uuid AND tour_id = $3
		FOR UPDATE`, tenantID, userID, tourID).Scan(&status, &currentStep)
	if errors.Is(err, pgx.ErrNoRows) {
		status, currentStep = "not_started", ""
	} else if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tour progress")
		return
	}

	// An already-completed tour is a no-op, not a re-verification: goals can
	// flip back (a session closes, a product is archived) and that must not
	// retroactively fail a tour the user finished.
	if action == "complete" && status == "completed" {
		writeData(c, http.StatusOK, gin.H{"tour": tourID, "status": "completed"})
		return
	}

	next, terr := applyTransition(status, action)
	if terr != nil {
		if errors.Is(terr, errTourDone) {
			writeError(c, http.StatusConflict, "tour_completed", "tour is already completed")
			return
		}
		writeError(c, http.StatusConflict, "invalid_status", "tour cannot perform that action in its current status")
		return
	}

	if action == "complete" {
		met, gerr := goalSatisfied(ctx, tx, tenantID, def.Goal)
		if gerr != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify business goal")
			return
		}
		if !met {
			writeError(c, http.StatusConflict, "goal_not_met", "business goal is not satisfied yet")
			return
		}
	}

	if _, err = tx.Exec(ctx, `
		INSERT INTO onboarding_progress
			(tenant_id, user_id, tour_id, tour_version, status, current_step,
			 started_at, paused_at, completed_at, skipped_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6,
			CASE WHEN $5 = 'running' THEN NOW() END,
			CASE WHEN $5 = 'paused' THEN NOW() END,
			CASE WHEN $5 = 'completed' THEN NOW() END,
			CASE WHEN $5 = 'skipped' THEN NOW() END)
		ON CONFLICT (tenant_id, user_id, tour_id) DO UPDATE SET
			status = EXCLUDED.status,
			tour_version = EXCLUDED.tour_version,
			current_step = EXCLUDED.current_step,
			started_at = COALESCE(onboarding_progress.started_at, EXCLUDED.started_at),
			paused_at = CASE WHEN $5 = 'paused' THEN EXCLUDED.paused_at ELSE onboarding_progress.paused_at END,
			completed_at = CASE WHEN $5 = 'completed' THEN EXCLUDED.completed_at ELSE onboarding_progress.completed_at END,
			skipped_at = CASE WHEN $5 = 'skipped' THEN EXCLUDED.skipped_at ELSE onboarding_progress.skipped_at END,
			updated_at = NOW()`,
		tenantID, userID, tourID, def.Version, next, currentStep); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save tour progress")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save tour progress")
		return
	}
	writeData(c, http.StatusOK, gin.H{"tour": tourID, "status": next})
}

func (h *Handler) start(c *gin.Context)    { h.mutate(c, "start") }
func (h *Handler) pause(c *gin.Context)    { h.mutate(c, "pause") }
func (h *Handler) resume(c *gin.Context)   { h.mutate(c, "resume") }
func (h *Handler) skip(c *gin.Context)     { h.mutate(c, "skip") }
func (h *Handler) complete(c *gin.Context) { h.mutate(c, "complete") }

type stepRequest struct {
	CurrentStep string `json:"current_step"`
}

func (h *Handler) updateStep(c *gin.Context) {
	tenantID, userID, ok := h.authorize(c, "write")
	if !ok {
		return
	}
	tourID := c.Param("id")
	if _, found := lookupTour(tourID); !found {
		writeError(c, http.StatusNotFound, "tour_not_found", "unknown tour")
		return
	}
	var req stepRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if err := validateStep(req.CurrentStep); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_step", "current_step must be a stable lowercase step id")
		return
	}
	tx, ok := h.beginTx(c, tenantID)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	ctx := c.Request.Context()
	tag, err := tx.Exec(ctx, `
		UPDATE onboarding_progress
		SET current_step = $4, updated_at = NOW()
		WHERE tenant_id = $1::uuid AND user_id = $2::uuid AND tour_id = $3
		  AND status IN ('running', 'paused')`,
		tenantID, userID, tourID, req.CurrentStep)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save tour progress")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(c, http.StatusConflict, "tour_not_running", "tour must be started before its progress can be saved")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save tour progress")
		return
	}
	writeData(c, http.StatusOK, gin.H{"tour": tourID, "current_step": req.CurrentStep})
}

type eventRequest struct {
	EventID string          `json:"event_id"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

func (h *Handler) event(c *gin.Context) {
	tenantID, userID, ok := h.authorize(c, "write")
	if !ok {
		return
	}
	var req eventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", "invalid request body")
		return
	}
	if req.Payload == nil {
		req.Payload = json.RawMessage("{}")
	}
	if err := validateEvent(req.EventID, req.Event, req.Payload); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_event", "event_id, event or payload is invalid")
		return
	}
	tx, ok := h.beginTx(c, tenantID)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	ctx := c.Request.Context()

	var insertedID string
	err := tx.QueryRow(ctx, `
		INSERT INTO onboarding_events (tenant_id, user_id, event_id, event, payload)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5::jsonb)
		ON CONFLICT (tenant_id, event_id) DO NOTHING
		RETURNING id::text`,
		tenantID, userID, req.EventID, req.Event, string(req.Payload)).Scan(&insertedID)
	replayed := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !replayed {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record event")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to record event")
		return
	}
	status := http.StatusCreated
	if replayed {
		status = http.StatusOK
	}
	writeData(c, status, gin.H{"event_id": req.EventID, "event": req.Event, "replayed": replayed})
}
