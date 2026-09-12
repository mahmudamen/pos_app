package database_test

import (
	"context"
	"testing"

	"github.com/example/pos-api/internal/infrastructure/database"
	"github.com/example/pos-api/internal/testutil"
	"github.com/google/uuid"
)

// TestRLSIsolatesTenants verifies that transaction-local tenant context hides
// rows of other tenants at the SQL layer, including writes.
func TestRLSIsolatesTenants(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	ctx := context.Background()

	tenantA := testutil.SeedTenant(t, pool)
	tenantB := testutil.SeedTenant(t, pool)

	insertProduct := func(tenantID, sku string) {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer tx.Rollback(ctx)
		err = database.WithTenant(ctx, tx, mustParseUUID(t, tenantID), func() error {
			_, err := tx.Exec(ctx, `
				INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, currency, stock_quantity)
				VALUES ($1::uuid, NULL, $2, $2, $2, 1000, 'EGP', 5)`, tenantID, sku)
			return err
		})
		if err != nil {
			t.Fatalf("insert product: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit insert: %v", err)
		}
	}
	insertProduct(tenantA.TenantID, "A-001")
	insertProduct(tenantA.TenantID, "A-002")
	insertProduct(tenantB.TenantID, "B-001")

	countProducts := func(tenantID string) int {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer tx.Rollback(ctx)
		count := 0
		err = database.WithTenant(ctx, tx, mustParseUUID(t, tenantID), func() error {
			return tx.QueryRow(ctx, `SELECT count(*) FROM products`).Scan(&count)
		})
		if err != nil {
			t.Fatalf("count products: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit count: %v", err)
		}
		return count
	}

	if got := countProducts(tenantA.TenantID); got != 2 {
		t.Fatalf("tenant A sees %d products, want 2", got)
	}
	if got := countProducts(tenantB.TenantID); got != 1 {
		t.Fatalf("tenant B sees %d products, want 1", got)
	}

	// A cross-tenant UPDATE must affect zero rows because RLS filters the row.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin update: %v", err)
	}
	defer tx.Rollback(ctx)
	var updated int64
	err = database.WithTenant(ctx, tx, mustParseUUID(t, tenantB.TenantID), func() error {
		result, err := tx.Exec(ctx, `UPDATE products SET stock_quantity = 999 WHERE sku = 'A-001'`)
		if err != nil {
			return err
		}
		updated = result.RowsAffected()
		return nil
	})
	if err != nil {
		t.Fatalf("cross-tenant update: %v", err)
	}
	if updated != 0 {
		t.Fatalf("cross-tenant update affected %d rows, want 0", updated)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit update: %v", err)
	}

	// Tenant A's stock is untouched by B's attempt.
	txRead, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin verify: %v", err)
	}
	defer txRead.Rollback(ctx)
	var stock int64
	err = database.WithTenant(ctx, txRead, mustParseUUID(t, tenantA.TenantID), func() error {
		return txRead.QueryRow(ctx, `SELECT stock_quantity FROM products WHERE sku = 'A-001'`).Scan(&stock)
	})
	if err != nil {
		t.Fatalf("verify stock: %v", err)
	}
	if stock != 5 {
		t.Fatalf("tenant A stock changed to %d, want 5", stock)
	}
}

func mustParseUUID(t *testing.T, value string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(value)
	if err != nil {
		t.Fatalf("parse uuid %q: %v", value, err)
	}
	return id
}
