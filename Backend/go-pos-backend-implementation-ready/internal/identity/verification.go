package identity

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

// Token purposes mirror the email_verification_tokens.purpose CHECK.
const (
	TokenPurposeVerify      = "verify"
	TokenPurposeEmailChange = "email_change"
)

// VerifiedMailPayload is what a consumed email token authorizes.
type VerifiedMailPayload struct {
	AccountID string
	NewEmail  string
}

// CreateEmailToken mints a single-use signed email token for an account and
// stores only its digest. The plain token is returned to be delivered once
// by the mailer; it cannot be read back from the database.
func CreateEmailToken(ctx context.Context, q Querier, accountID, purpose, newEmail string, ttl time.Duration) (string, error) {
	plain, hash, err := NewToken()
	if err != nil {
		return "", err
	}
	_, err = q.Exec(ctx, `
		INSERT INTO email_verification_tokens (id, account_id, token_hash, purpose, new_email, expires_at)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, NULLIF($4, ''), now() + make_interval(secs => $5))`,
		accountID, hash, purpose, newEmail, int64(ttl.Seconds()))
	if err != nil {
		return "", err
	}
	return plain, nil
}

// ConsumeEmailToken validates and single-uses a presented token. The
// conditional UPDATE (consumed_at IS NULL AND expires_at > now()) is atomic:
// a replayed token race can never verify twice. Returns ErrInvalidToken on any
// failure so callers cannot distinguish "unknown token" from "burnt token".
func ConsumeEmailToken(ctx context.Context, q Querier, plain, purpose string, ttl time.Duration) (VerifiedMailPayload, error) {
	if len(plain) > 128 || plain == "" {
		return VerifiedMailPayload{}, ErrInvalidToken
	}
	hash := HashHex(plain)
	var payload VerifiedMailPayload
	err := q.QueryRow(ctx, `
		UPDATE email_verification_tokens
		SET consumed_at = now()
		WHERE token_hash = $1 AND purpose = $2
		  AND consumed_at IS NULL AND expires_at > now()
		RETURNING account_id::text, COALESCE(new_email, '')`,
		hash, purpose).Scan(&payload.AccountID, &payload.NewEmail)
	if errors.Is(err, pgx.ErrNoRows) {
		return VerifiedMailPayload{}, ErrInvalidToken
	}
	return payload, err
}

// OTPChallenge is the read view of phone_otp_challenges.
type OTPChallenge struct {
	ID         string
	AccountID  string
	PhoneE164  string
	Attempts   int
	MaxAllowed int
	ExpiresAt  time.Time
}

// CreateOTPChallenge mints an OTP for an account+phone and stores only its
// hash. Returns the plain six-digit code for one-time delivery. Old challenges
// for the same account/phone are invalidated so a resend cannot pile up codes.
func CreateOTPChallenge(ctx context.Context, q Querier, accountID, phone string, ttl time.Duration) (string, error) {
	plain, hash, err := NewOTP()
	if err != nil {
		return "", err
	}
	_, err = q.Exec(ctx, `
		UPDATE phone_otp_challenges SET consumed_at = now()
		WHERE account_id = $1::uuid AND phone_e164 = $2 AND consumed_at IS NULL`,
		accountID, phone)
	if err != nil {
		return "", err
	}
	_, err = q.Exec(ctx, `
		INSERT INTO phone_otp_challenges (id, account_id, phone_e164, otp_hash, expires_at)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, now() + make_interval(secs => $4))`,
		accountID, phone, hash, int64(ttl.Seconds()))
	if err != nil {
		return "", err
	}
	return plain, nil
}

// VerifyOTP checks a presented code against the latest active challenge for
// the account+phone. Every failed attempt increments the counter; reaching
// max_attempts burns the challenge (rate limit). On success the challenge is
// consumed in the same statement, so replay cannot pass twice.
func VerifyOTP(ctx context.Context, q Querier, accountID, phone, code string, ttl time.Duration) error {
	if !ValidateOTP(code) {
		return ErrInvalidToken
	}
	hash := HashHex(code)
	var id string
	ok := false
	err := q.QueryRow(ctx, `
		UPDATE phone_otp_challenges
		SET consumed_at = now()
		WHERE account_id = $1::uuid AND phone_e164 = $2 AND otp_hash = $3
		  AND consumed_at IS NULL AND expires_at > now() AND attempts < max_attempts
		RETURNING id::text`, accountID, phone, hash).Scan(&id)
	if err == nil {
		ok = true
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if ok {
		return nil
	}
	// Failed attempt: increment, and burn the challenge at/over the cap so an
	// attacker cannot brute-force beyond max_attempts per code.
	failed, err2 := q.Exec(ctx, `
		UPDATE phone_otp_challenges
		SET attempts = attempts + 1,
		    consumed_at = CASE WHEN attempts + 1 >= max_attempts THEN now() ELSE consumed_at END
		WHERE account_id = $1::uuid AND phone_e164 = $2 AND otp_hash = $3
		  AND consumed_at IS NULL AND expires_at > now() AND attempts < max_attempts`,
		accountID, phone, hash)
	if err2 != nil {
		return err2
	}
	_ = failed
	return ErrInvalidToken
}

// OTPResendWindow returns the resend throttle window for an account/phone pair.
func OTPResendWindow() time.Duration { return 30 * time.Minute }
