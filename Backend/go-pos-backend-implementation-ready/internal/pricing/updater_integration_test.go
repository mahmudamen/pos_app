package pricing

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testBarcode = "6221010000017" // Juhayna Fresh Milk 1L in the bundled catalog

func testSlug() string {
	b := make([]byte, 4)
	if _, err := cryptorand.Read(b); err != nil {
		return "pricing-test"
	}
	return "pricing-" + hex.EncodeToString(b)
}

// insertDemoProduct creates a demo-seeded tenant with a single product carrying
// the given barcode (and starting price), and returns the tenant id in cleanup.
func insertDemoProduct(t *testing.T, pool *pgxpool.Pool, demoSeeded bool, start int64) string {
	t.Helper()
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var tenantID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO tenants (name, slug, is_demo_seeded) VALUES ($1, $2, $3) RETURNING id::text`,
		"Pricing Test", testSlug(), demoSeeded).Scan(&tenantID); err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM tenants WHERE id = $1::uuid`, tenantID)
	})
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}
	var categoryID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO categories (tenant_id, name, slug) VALUES ($1::uuid, 'Test', 'test') RETURNING id::text`,
		tenantID).Scan(&categoryID); err != nil {
		t.Fatalf("seed category: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity, unit)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $6, 'EGP', 10, 'piece')`,
		tenantID, categoryID, "Demo Milk", "DEMO-1", testBarcode, start); err != nil {
		t.Fatalf("seed product: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	return tenantID
}

func productPrice(t *testing.T, pool *pgxpool.Pool, tenantID string) int64 {
	t.Helper()
	var price int64
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant', $1, true)`, tenantID); err != nil {
		t.Fatalf("set tenant context: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT price_minor FROM products WHERE barcode = $1`, testBarcode).Scan(&price); err != nil {
		t.Fatalf("read price: %v", err)
	}
	return price
}

func TestRunOnceUpdatesOnlyDemoSeededTenants(t *testing.T) {
	pool := testutil.Pool(t)
	testutil.Migrate(t, testutil.DatabaseURL(t))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	demoTenant := insertDemoProduct(t, pool, true, 1)
	realTenant := insertDemoProduct(t, pool, false, 999_999)

	result, err := RunOnce(context.Background(), pool, 0, nil, logger)
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	// RunOnce sweeps every demo-seeded tenant. On a fresh DB that is exactly 1
	// (this fixture); on a shared dev DB other demo-seeded stores accumulate
	// (e.g. merchant signups), so only the lower bound is meaningful here. The
	// critical guarantees — the demo fixture is updated, the real tenant is
	// untouched — are still asserted below.
	if result.Tenants < 1 || result.Products < 1 {
		t.Fatalf("expected at least 1 tenant/1 product updated, got %d/%d", result.Tenants, result.Products)
	}
	if got := productPrice(t, pool, demoTenant); got != 4200 {
		t.Fatalf("demo tenant price: got %d, want 4200 (catalog)", got)
	}
	if got := productPrice(t, pool, realTenant); got != 999_999 {
		t.Fatalf("non-demo tenant must be untouched, got %d", got)
	}
}
