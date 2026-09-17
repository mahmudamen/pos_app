package identity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// AccountStore owns the `accounts` table. Accounts are platform-level (no
// RLS): they precede any tenant context, and only server services reach them.
type AccountStore struct {
	audit AuditRecorder
}

func NewAccountStore(audit AuditRecorder) *AccountStore {
	return &AccountStore{audit: audit}
}

// Create inserts a new account for a freshly registered signup. Email
// verification state follows the policy: when requireEmail is true the account
// starts pending and the trial is withheld until verification completes;
// otherwise the claim is treated as satisfied (self-asserted until a mailer
// exists). The same applies to phone when the signup carried one.
func (s *AccountStore) Create(ctx context.Context, q Querier, email, phone string, requireEmail, requirePhone bool) (Account, error) {
	status := AccountStatusActive
	if requireEmail {
		status = AccountStatusPending
	}
	normalizedPhone := NormalizePhone(phone)
	var account Account
	err := q.QueryRow(ctx, `
		INSERT INTO accounts (id, primary_email, status, email_verified_at, phone_e164, phone_verified_at)
		VALUES (gen_random_uuid(), $1, $2,
		        CASE WHEN $3 THEN now() ELSE NULL END,
		        NULLIF($4, ''),
		        CASE WHEN ($5 AND $6 <> '') THEN now() ELSE NULL END)
		RETURNING id, primary_email, email_verified_at, COALESCE(phone_e164, ''), phone_verified_at, status, risk_level`,
		email, status, !requireEmail, normalizedPhone, !requirePhone, normalizedPhone).
		Scan(&account.ID, &account.PrimaryEmail, &account.EmailVerifiedAt, &account.PhoneE164, &account.PhoneVerifiedAt, &account.Status, &account.RiskLevel)
	if err != nil {
		if isUniqueViolation(err) {
			return Account{}, ErrStateConflict
		}
		return Account{}, err
	}
	return account, nil
}

// GetByID loads one account.
func (s *AccountStore) GetByID(ctx context.Context, q Querier, accountID string) (Account, error) {
	var a Account
	err := q.QueryRow(ctx, `
		SELECT id, primary_email, email_verified_at, COALESCE(phone_e164, ''), phone_verified_at, status, risk_level
		FROM accounts WHERE id = $1::uuid`, accountID).
		Scan(&a.ID, &a.PrimaryEmail, &a.EmailVerifiedAt, &a.PhoneE164, &a.PhoneVerifiedAt, &a.Status, &a.RiskLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	return a, err
}

// GetByEmail loads one account by its canonical (normalized) email.
func (s *AccountStore) GetByEmail(ctx context.Context, q Querier, email string) (Account, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return Account{}, ErrNotFound
	}
	var a Account
	err := q.QueryRow(ctx, `
		SELECT id, primary_email, email_verified_at, COALESCE(phone_e164, ''), phone_verified_at, status, risk_level
		FROM accounts WHERE lower(primary_email) = $1`, email).
		Scan(&a.ID, &a.PrimaryEmail, &a.EmailVerifiedAt, &a.PhoneE164, &a.PhoneVerifiedAt, &a.Status, &a.RiskLevel)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	return a, err
}

// MarkEmailVerified sets email_verified_at and promotes a pending account.
func (s *AccountStore) MarkEmailVerified(ctx context.Context, q Querier, accountID string) error {
	_, err := q.Exec(ctx, `
		UPDATE accounts SET email_verified_at = COALESCE(email_verified_at, now()),
			status = CASE WHEN status = 'pending' THEN 'active' ELSE status END,
			updated_at = now()
		WHERE id = $1::uuid`, accountID)
	return err
}

// SetPhone replaces the account phone and marks it verified. It returns
// ErrStateConflict when the phone already belongs to another account.
func (s *AccountStore) SetPhone(ctx context.Context, q Querier, accountID, phone string) error {
	_, err := q.Exec(ctx, `
		UPDATE accounts SET phone_e164 = NULLIF($2, ''), phone_verified_at = now(), updated_at = now()
		WHERE id = $1::uuid`, accountID, NormalizePhone(phone))
	if err != nil && isUniqueViolation(err) {
		return ErrStateConflict
	}
	return err
}

// VerifyPhone sets phone_verified_at without changing the stored number.
func (s *AccountStore) VerifyPhone(ctx context.Context, q Querier, accountID string) error {
	_, err := q.Exec(ctx, `
		UPDATE accounts SET phone_verified_at = now(), updated_at = now() WHERE id = $1::uuid`, accountID)
	return err
}

// ChangeEmail moves the account to a new, unverified email. Only callers who
// already proved ownership of the account may do this (the email-change token
// flow requires the current user's password), and the audit trail records it.
// A duplicate primary_email race surfaces as ErrStateConflict via the caller.
func (s *AccountStore) ChangeEmail(ctx context.Context, q Querier, accountID, newEmail string) error {
	if err := s.audit.Record(ctx, q, AuditEntry{
		Action:     ActionEmailChangeCompleted,
		AccountID:  accountID,
		EntityID:   accountID,
		EntityType: EntityAccount,
		Reason:     "email pending verification: " + newEmail,
	}); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `
		UPDATE accounts SET primary_email = $2, email_verified_at = NULL, updated_at = now()
		WHERE id = $1::uuid`, accountID, NormalizeEmail(newEmail))
	if err != nil && isUniqueViolation(err) {
		return ErrStateConflict
	}
	return err
}

// Suspend locks an account (admin action). Suspended accounts cannot pass the
// eligibility check and their members are stopped by the subscription gate.
func (s *AccountStore) Suspend(ctx context.Context, q Querier, accountID, reason string) error {
	_, err := q.Exec(ctx, `UPDATE accounts SET status = 'suspended', updated_at = now() WHERE id = $1::uuid`, accountID)
	return err
}

// OrganizationCount returns the number of organizations owned by the account.
// Ownership attribution uses tenants.owner_account_id (added and backfilled in
// migration 034); counting memberships through users is impossible because
// users is FORCE-RLS and no single tenant context spans an account.
func (s *AccountStore) OrganizationCount(ctx context.Context, q Querier, accountID string) (int, error) {
	var count int
	err := q.QueryRow(ctx, `
		SELECT COUNT(*) FROM tenants
		WHERE owner_account_id = $1::uuid AND status <> 'closed'`, accountID).Scan(&count)
	return count, err
}

func isUniqueViolation(err error) bool {
	var pgErr interface{ Code() string }
	if errors.As(err, &pgErr) {
		return pgErr.Code() == "23505"
	}
	return false
}
