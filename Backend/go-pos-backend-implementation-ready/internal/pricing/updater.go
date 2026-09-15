package pricing

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Result reports what a single refresh run changed.
type Result struct {
	Tenants  int `json:"tenants"`
	Products int `json:"products"`
}

// RunOnce refreshes prices for every demo-seeded tenant in one transaction.
// Tenant tables are FORCE RLS, so each tenant's context is set before its rows
// are touched — the same pattern the SaaS analytics handler uses.
func RunOnce(ctx context.Context, pool *pgxpool.Pool, variationPct int, rng *rand.Rand, logger *slog.Logger) (Result, error) {
	var result Result
	if pool == nil {
		return result, fmt.Errorf("pricing: database pool is nil")
	}
	items := EgyptCatalog()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("pricing: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `SELECT id FROM tenants WHERE is_demo_seeded = TRUE ORDER BY id`)
	if err != nil {
		return result, fmt.Errorf("pricing: list demo tenants: %w", err)
	}
	var tenantIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return result, fmt.Errorf("pricing: scan tenant: %w", err)
		}
		tenantIDs = append(tenantIDs, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("pricing: iterate tenants: %w", err)
	}

	for _, tenantID := range tenantIDs {
		if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
			return result, fmt.Errorf("pricing: set tenant %s: %w", tenantID, err)
		}
		tenantUpdated := 0
		for _, it := range items {
			price := VariedPrice(it.PriceMinor, variationPct, rng)
			cost := VariedPrice(it.CostMinor, variationPct, rng)
			tag, err := tx.Exec(ctx,
				`UPDATE products SET price_minor = $2, cost_minor = $3, unit = $4
				 WHERE barcode = $1 AND is_active`, it.Barcode, price, cost, it.Unit)
			if err != nil {
				return result, fmt.Errorf("pricing: update %s: %w", it.Barcode, err)
			}
			if tag.RowsAffected() > 0 {
				tenantUpdated++
			}
		}
		if tenantUpdated > 0 {
			result.Tenants++
			result.Products += tenantUpdated
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("pricing: commit: %w", err)
	}
	logger.Info("price refresh complete", "tenants", result.Tenants, "products", result.Products)
	return result, nil
}
