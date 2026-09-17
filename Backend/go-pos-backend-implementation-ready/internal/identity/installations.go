package identity

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// InstallationStore owns the `installations` table. Records are platform-level
// (no RLS); they are a risk signal, not an identity, and never exposed to other
// clients.
type InstallationStore struct{}

func NewInstallationStore() *InstallationStore { return &InstallationStore{} }

// Upsert registers or refreshes an installation by its client-generated random
// public id. The same public id presented twice (retry) refreshes the same
// row; a reinstall generates a new public id and thus a new row. The account
// and tenant links are applied by Link after authentication/signup.
func (s *InstallationStore) Upsert(ctx context.Context, q Querier, publicID, platform, appVersion string) (Installation, error) {
	var i Installation
	err := q.QueryRow(ctx, `
		INSERT INTO installations (id, installation_public_id, platform, app_version, last_seen_at)
		VALUES (gen_random_uuid(), $1::uuid, $2, $3, now())
		ON CONFLICT (installation_public_id) DO UPDATE
			SET platform = EXCLUDED.platform, app_version = EXCLUDED.app_version,
			    last_seen_at = now()
		RETURNING id::text, installation_public_id::text, COALESCE(account_id::text, ''), COALESCE(tenant_id::text, ''),
		          platform, app_version, first_seen_at, last_seen_at, last_authenticated_at,
		          COALESCE(device_integrity_status, ''), risk_level, revoked_at`,
		publicID, platform, appVersion).
		Scan(&i.ID, &i.InstallationPublicID, &i.AccountID, &i.TenantID,
			&i.Platform, &i.AppVersion, &i.FirstSeenAt, &i.LastSeenAt, &i.LastAuthenticatedAt,
			&i.DeviceIntegrityStatus, &i.RiskLevel, &i.RevokedAt)
	if err != nil {
		return Installation{}, err
	}
	return i, nil
}

// Link binds an installation to an account and organization after
// authentication/signup and records the last authenticated time.
func (s *InstallationStore) Link(ctx context.Context, q Querier, installationID, accountID, tenantID string) error {
	_, err := q.Exec(ctx, `
		UPDATE installations
		SET account_id = COALESCE($2::uuid, account_id),
		    tenant_id = COALESCE($3::uuid, tenant_id),
		    last_authenticated_at = now(), last_seen_at = now(), revoked_at = NULL
		WHERE id = $1::uuid`, installationID, nullable(accountID), nullable(tenantID))
	return err
}

// GetByPublicID loads an installation by its public id.
func (s *InstallationStore) GetByPublicID(ctx context.Context, q Querier, publicID string) (*Installation, error) {
	var i Installation
	err := q.QueryRow(ctx, `
		SELECT id::text, installation_public_id::text, COALESCE(account_id::text, ''), COALESCE(tenant_id::text, ''),
		       platform, app_version, first_seen_at, last_seen_at, last_authenticated_at,
		       COALESCE(device_integrity_status, ''), risk_level, revoked_at
		FROM installations WHERE installation_public_id = $1::uuid`, publicID).
		Scan(&i.ID, &i.InstallationPublicID, &i.AccountID, &i.TenantID,
			&i.Platform, &i.AppVersion, &i.FirstSeenAt, &i.LastSeenAt, &i.LastAuthenticatedAt,
			&i.DeviceIntegrityStatus, &i.RiskLevel, &i.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &i, err
}

// SetIntegrityStatus records a server-verified device attestation result.
// Only a server-verified value is trusted; client booleans are never accepted.
func (s *InstallationStore) SetIntegrityStatus(ctx context.Context, q Querier, installationID, status string) error {
	_, err := q.Exec(ctx, `
		UPDATE installations SET device_integrity_status = NULLIF($2, ''), updated_at = now()
		WHERE id = $1::uuid`, installationID, status)
	return err
}

// ListByAccount returns the account's installations (newest first). Multiple
// legitimate devices per account are expected and never blocked.
func (s *InstallationStore) ListByAccount(ctx context.Context, q Querier, accountID string) ([]Installation, error) {
	rows, err := q.Query(ctx, `
		SELECT id::text, installation_public_id::text, COALESCE(account_id::text, ''), COALESCE(tenant_id::text, ''),
		       platform, app_version, first_seen_at, last_seen_at, last_authenticated_at,
		       COALESCE(device_integrity_status, ''), risk_level, revoked_at
		FROM installations WHERE account_id = $1::uuid AND revoked_at IS NULL
		ORDER BY last_seen_at DESC`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Installation, 0)
	for rows.Next() {
		var i Installation
		if err := rows.Scan(&i.ID, &i.InstallationPublicID, &i.AccountID, &i.TenantID,
			&i.Platform, &i.AppVersion, &i.FirstSeenAt, &i.LastSeenAt, &i.LastAuthenticatedAt,
			&i.DeviceIntegrityStatus, &i.RiskLevel, &i.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// ActiveCount counts the account's non-revoked installations.
func (s *InstallationStore) ActiveCount(ctx context.Context, q Querier, accountID string) (int, error) {
	var count int
	err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM installations WHERE account_id = $1::uuid AND revoked_at IS NULL`, accountID).
		Scan(&count)
	return count, err
}

// Revoke rotates an installation away from an account.
func (s *InstallationStore) Revoke(ctx context.Context, q Querier, installationID, accountID string) error {
	_, err := q.Exec(ctx, `
		UPDATE installations SET revoked_at = now(), updated_at = now()
		WHERE id = $1::uuid AND account_id = $2::uuid AND revoked_at IS NULL`,
		installationID, accountID)
	return err
}
