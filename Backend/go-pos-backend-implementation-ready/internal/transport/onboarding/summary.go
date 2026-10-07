package onboarding

import (
	"net/http"
	"sort"

	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/gin-gonic/gin"
)

// tourAggregate is one tour's progress counts across the whole tenant.
type tourAggregate struct {
	ID         string `json:"id"`
	Started    int    `json:"started"`
	Completed  int    `json:"completed"`
	Skipped    int    `json:"skipped"`
	InProgress int    `json:"in_progress"`
}

type totals struct {
	Started    int `json:"started"`
	Completed  int `json:"completed"`
	Skipped    int `json:"skipped"`
	InProgress int `json:"in_progress"`
}

// aggregateTours folds (tour, status, count) rows into one aggregate per tour
// in catalog order. Rows are per (user, tour); a starting row is any row whose
// status is not the placeholder not_started.
func aggregateTours(statusCounts map[string]map[string]int) []tourAggregate {
	out := make([]tourAggregate, 0, len(catalog))
	for _, def := range catalog {
		byStatus := statusCounts[def.ID]
		agg := tourAggregate{ID: def.ID}
		advance := 0
		for status, count := range byStatus {
			switch status {
			case "completed":
				agg.Completed += count
			case "skipped":
				agg.Skipped += count
			case "running", "paused":
				agg.InProgress += count
			}
			if status != "not_started" {
				advance += count
			}
		}
		agg.Started = advance
		out = append(out, agg)
	}
	return out
}

// aggregateTotals sums the per-tour aggregates.
func aggregateTotals(tours []tourAggregate) totals {
	out := totals{}
	for _, t := range tours {
		out.Started += t.Started
		out.Completed += t.Completed
		out.Skipped += t.Skipped
		out.InProgress += t.InProgress
	}
	return out
}

// completionAndSkipRate reports rounded rates (0..1); division by zero yields 0.
func completionAndSkipRate(started, completed, skipped int) (completion, skip float64) {
	if started == 0 {
		return 0, 0
	}
	return round3(float64(completed) / float64(started)), round3(float64(skipped) / float64(started))
}

func round3(v float64) float64 {
	scaled := int(v*1000 + 0.5)
	return float64(scaled) / 1000
}

// eventCounts holds the reported onboarding/activation events by name.
type eventCounts map[string]int

// sortedKeys returns event names in stable order for deterministic JSON.
func (e eventCounts) sortedKeys() []string {
	keys := make([]string, 0, len(e))
	for k := range e {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// summary aggregates tenant-wide onboarding and activation metrics for the
// onboarding center (EPIC 13). It reads manager-visible analytical data, so it
// rides the dashboard.read permission rather than onboarding.read.
func (h *Handler) summary(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "dashboard", "read") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tx, ok := h.beginTx(c, claims.TenantID)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	ctx := c.Request.Context()

	events := eventCounts{}
	rows, err := tx.Query(ctx, `SELECT event, COUNT(*) FROM onboarding_events GROUP BY event`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate onboarding events")
		return
	}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate onboarding events")
			return
		}
		events[name] = count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate onboarding events")
		return
	}

	statusCounts := map[string]map[string]int{}
	rows, err = tx.Query(ctx, `
		SELECT tour_id, status, COUNT(*) FROM onboarding_progress GROUP BY tour_id, status`)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate tour progress")
		return
	}
	for rows.Next() {
		var tour, status string
		var count int
		if err := rows.Scan(&tour, &status, &count); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate tour progress")
			return
		}
		if statusCounts[tour] == nil {
			statusCounts[tour] = map[string]int{}
		}
		statusCounts[tour][status] = count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate tour progress")
		return
	}

	tours := aggregateTours(statusCounts)
	totals := aggregateTotals(tours)
	completion, skip := completionAndSkipRate(totals.Started, totals.Completed, totals.Skipped)

	var rewardGrants int
	if err = tx.QueryRow(ctx, `
		SELECT COUNT(*) FROM onboarding_rewards
		WHERE tenant_id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID`).Scan(&rewardGrants); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate rewards")
		return
	}

	var firstSale *int64
	var tenantCreatedAt *int64
	if err = tx.QueryRow(ctx, `
		SELECT (EXTRACT(EPOCH FROM MIN(s.created_at))::bigint)
		FROM sales s WHERE s.status = 'completed'`).Scan(&firstSale); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to aggregate sales")
		return
	}
	if err = tx.QueryRow(ctx, `
		SELECT (EXTRACT(EPOCH FROM created_at))::bigint
		FROM tenants WHERE id = NULLIF(current_setting('app.current_tenant', TRUE), '')::UUID`).Scan(&tenantCreatedAt); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load tenant")
		return
	}
	var timeToFirstSale *int64
	if firstSale != nil && tenantCreatedAt != nil {
		diff := *firstSale - *tenantCreatedAt
		timeToFirstSale = &diff
	}

	var activated bool
	if err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sales s
			WHERE s.status = 'completed'
			  AND EXISTS (SELECT 1 FROM sale_payments p WHERE p.sale_id = s.id)
		)`).Scan(&activated); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to verify activation")
		return
	}

	eventJSON := gin.H{}
	for _, key := range events.sortedKeys() {
		eventJSON[key] = events[key]
	}

	data := gin.H{
		"events":                     eventJSON,
		"tours":                      tours,
		"totals":                     totals,
		"rates":                      gin.H{"completion_rate": completion, "skip_rate": skip},
		"reward_grants":              rewardGrants,
		"time_to_first_sale_seconds": timeToFirstSale,
		"activated":                  activated,
	}
	writeData(c, http.StatusOK, data)
}
