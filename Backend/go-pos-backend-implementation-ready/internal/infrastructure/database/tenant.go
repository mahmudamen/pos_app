package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// WithTenant runs fn inside a transaction-local RLS tenant context.
func WithTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, fn func() error) error {
	if tenantID == uuid.Nil {
		return fmt.Errorf("tenant id is required")
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant context: %w", err)
	}
	if err := fn(); err != nil {
		return err
	}
	return nil
}
