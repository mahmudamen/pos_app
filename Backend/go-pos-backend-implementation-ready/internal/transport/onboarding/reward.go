package onboarding

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Reward type is a fixed server-side constant: the client may render the
// congratulations, it may never define the reward. Kept a constant because the
// 2026-10-07 decision deliberately does NOT implement the spec's
// ONBOARDING_30_DAY_EXTENSION (documented deviation, trial stays 15 days).
const rewardType = "ONBOARDING_ACTIVATION"

type rewardView struct {
	Eligible   bool    `json:"eligible"`
	Granted    bool    `json:"granted"`
	RewardType string  `json:"reward_type"`
	GrantedAt  *string `json:"granted_at,omitempty"`
}

// loadReward builds the caller's reward picture. Eligibility is never trusted
// from the client: a reward is claimable only once the business facts say the
// tenant activated (ready.Activated is derived from the real sales feed).
func loadReward(ctx context.Context, tx pgx.Tx, tenantID, userID string, ready readiness) (rewardView, error) {
	if !ready.Activated {
		return rewardView{Eligible: false, Granted: false, RewardType: rewardType}, nil
	}
	var rtype string
	var grantedAt *time.Time
	err := tx.QueryRow(ctx, `
		SELECT reward_type, granted_at FROM onboarding_rewards
		WHERE tenant_id = $1::uuid AND user_id = $2::uuid AND reward_type = $3`,
		tenantID, userID, rewardType).Scan(&rtype, &grantedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return rewardView{Eligible: true, Granted: false, RewardType: rewardType}, nil
	}
	if err != nil {
		return rewardView{}, err
	}
	out := rewardView{Eligible: true, Granted: true, RewardType: rtype}
	if grantedAt != nil {
		s := grantedAt.UTC().Format(time.RFC3339)
		out.GrantedAt = &s
	}
	return out, nil
}

// claimReward is the idempotent grant. Calling it before activation records
// nothing; calling it twice records once. The grant is a milestone (the trial
// stays 15 days), so no trial_ends_at is touched here.
func (h *Handler) claimReward(c *gin.Context) {
	tenantID, userID, ok := h.authorize(c, "write")
	if !ok {
		return
	}
	tx, ok := h.beginTx(c, tenantID)
	if !ok {
		return
	}
	defer func() { _ = tx.Rollback(c.Request.Context()) }()
	ctx := c.Request.Context()

	ready, err := readinessInTx(ctx, tx, tenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load onboarding readiness")
		return
	}
	if !ready.Activated {
		writeData(c, http.StatusOK, rewardView{Eligible: false, Granted: false, RewardType: rewardType})
		return
	}

	var rtype string
	var grantedAt *time.Time
	err = tx.QueryRow(ctx, `
		INSERT INTO onboarding_rewards (tenant_id, user_id, reward_type)
		VALUES ($1::uuid, $2::uuid, $3)
		ON CONFLICT (tenant_id, user_id, reward_type) DO NOTHING
		RETURNING reward_type, granted_at`,
		tenantID, userID, rewardType).Scan(&rtype, &grantedAt)
	replayed := errors.Is(err, pgx.ErrNoRows)
	if err != nil && !replayed {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to grant reward")
		return
	}
	if replayed {
		if err = tx.QueryRow(ctx, `
			SELECT reward_type, granted_at FROM onboarding_rewards
			WHERE tenant_id = $1::uuid AND user_id = $2::uuid AND reward_type = $3`,
			tenantID, userID, rewardType).Scan(&rtype, &grantedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load reward")
			return
		}
	}
	// Analytics completeness: server-side event rows belong in the same feed
	// the client reports into, so the summary endpoint sees the full story.
	if !replayed {
		if _, err = tx.Exec(ctx, `
			INSERT INTO onboarding_events (tenant_id, user_id, event_id, event, payload)
			VALUES ($1::uuid, $2::uuid, gen_random_uuid(), 'trial.reward_granted',
			        jsonb_build_object('reward_type', $3::text))`,
			tenantID, userID, rtype); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to record reward event")
			return
		}
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to grant reward")
		return
	}

	out := rewardView{Eligible: true, Granted: true, RewardType: rtype}
	out.GrantedAt = nil
	if grantedAt != nil {
		s := grantedAt.UTC().Format(time.RFC3339)
		out.GrantedAt = &s
	}
	data := gin.H{
		"eligible":    out.Eligible,
		"granted":     out.Granted,
		"reward_type": out.RewardType,
	}
	if out.GrantedAt != nil {
		data["granted_at"] = *out.GrantedAt
	}
	data["replayed"] = replayed
	writeData(c, http.StatusOK, data)
}
