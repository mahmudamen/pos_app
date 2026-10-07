package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Password-reset policy. The token is bound to one login (tenant + user), the
// same coordinates POST /v1/auth/login accepts, so a reset never needs a
// cross-tenant lookup and every follow-up statement runs under one tenant
// context.
const (
	// PasswordResetTTL is how long an emailed reset link stays usable.
	PasswordResetTTL = 30 * time.Minute
	// PasswordResetRateLimit caps reset *requests* per client IP per window.
	// It is a per-IP counter, so a shared NAT (an office, a carrier-grade NAT,
	// a shop behind one router) cannot be locked out by a handful of staff.
	PasswordResetRateLimit = 5
	// PasswordResetRateWindow is the window for PasswordResetRateLimit.
	PasswordResetRateWindow = 15 * time.Minute
)

// PasswordResetScope is what a consumed token authorizes: exactly one user in
// exactly one tenant.
type PasswordResetScope struct {
	TenantID string
	UserID   string
}

// CreatePasswordResetToken mints a single-use token bound to one login and
// stores only its digest. Any outstanding token for the same user is burnt
// first, so a resend never leaves two live links (and a leaked older link dies
// the moment a newer one is requested).
func CreatePasswordResetToken(ctx context.Context, q Querier, tenantID, userID string, ttl time.Duration) (string, error) {
	plain, hash, err := NewToken()
	if err != nil {
		return "", err
	}
	if _, err = q.Exec(ctx, `
		UPDATE password_reset_tokens SET consumed_at = now()
		WHERE user_id = $1::uuid AND consumed_at IS NULL`, userID); err != nil {
		return "", err
	}
	if _, err = q.Exec(ctx, `
		INSERT INTO password_reset_tokens (id, tenant_id, user_id, token_hash, expires_at)
		VALUES (gen_random_uuid(), $1::uuid, $2::uuid, $3, now() + make_interval(secs => $4))`,
		tenantID, userID, hash, int64(ttl.Seconds())); err != nil {
		return "", err
	}
	return plain, nil
}

// ConsumePasswordResetToken validates and single-uses a presented token. The
// conditional UPDATE (consumed_at IS NULL AND expires_at > now()) is atomic, so
// a replayed link can never reset twice. ErrInvalidToken is returned for every
// failure so callers cannot tell an unknown token from a burnt or expired one.
func ConsumePasswordResetToken(ctx context.Context, q Querier, plain string) (PasswordResetScope, error) {
	if plain == "" || len(plain) > 128 {
		return PasswordResetScope{}, ErrInvalidToken
	}
	hash := HashHex(plain)
	var scope PasswordResetScope
	err := q.QueryRow(ctx, `
		UPDATE password_reset_tokens
		SET consumed_at = now()
		WHERE token_hash = $1 AND consumed_at IS NULL AND expires_at > now()
		RETURNING tenant_id::text, user_id::text`, hash).
		Scan(&scope.TenantID, &scope.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PasswordResetScope{}, ErrInvalidToken
	}
	return scope, err
}
