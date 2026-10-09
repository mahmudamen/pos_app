// Package notify holds the small shared helpers that let platform-level
// workflows (billing, tenant administration) address a merchant. The recipient
// lives on `tenants.contact_email` (populated at signup); when a store predates
// that column the owner login is resolved inside a tenant context so FORCE RLS
// still lets the read through.
package notify

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ContactEmail returns the best available email for a tenant: the explicit
// contact address, else the owner user's email, else "".
func ContactEmail(ctx context.Context, pool *pgxpool.Pool, tenantID string) string {
	if pool == nil || tenantID == "" {
		return ""
	}
	var contact string
	if err := pool.QueryRow(ctx,
		`SELECT contact_email FROM tenants WHERE id = $1::uuid`, tenantID).Scan(&contact); err != nil {
		return ""
	}
	if contact != "" {
		return contact
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return ""
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		return ""
	}
	var owner string
	if err := tx.QueryRow(ctx, `
		SELECT email FROM users
		WHERE tenant_id = $1::uuid AND role = 'owner'
		ORDER BY created_at
		LIMIT 1`, tenantID).Scan(&owner); err != nil {
		return ""
	}
	return owner
}
