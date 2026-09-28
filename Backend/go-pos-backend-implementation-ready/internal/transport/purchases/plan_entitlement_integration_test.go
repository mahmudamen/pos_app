package purchases_test

// This file is a guard, not a feature test.
//
// `tenants.plan` is a free-text column whose default, 'standard', was never
// inserted into the plans catalog. Nothing noticed, because until this release
// `plans.features` was display-only and no code joined on it. The moment
// features became enforceable, the same latent gap became a revenue-affecting
// bug: /v1/purchases/ocr resolves entitlements with a LEFT JOIN and fails
// closed, so all 3,187 'standard' tenants — every existing customer — would
// have been refused invoice capture with a 402 that looked like a genuine
// plan gap.
//
// Migration 050 seeds the missing rows. These tests exist so the next dangling
// plan code is a failed build rather than a support ticket.

import (
	"context"
	"testing"

	"github.com/example/pos-api/internal/testutil"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestEveryTenantPlanCodeResolves is the invariant that actually matters.
func TestEveryTenantPlanCodeResolves(t *testing.T) {
	pool := purchasesPool(t)
	var dangling []string
	rows, err := pool.Query(context.Background(),
		`SELECT DISTINCT t.plan
		   FROM tenants t
		   LEFT JOIN plans p ON p.code = t.plan
		  WHERE p.code IS NULL`)
	if err != nil {
		t.Fatalf("query dangling plan codes: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatalf("scan: %v", err)
		}
		dangling = append(dangling, code)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(dangling) > 0 {
		t.Fatalf("tenant plan codes with no plans row: %v. Any code listed here is "+
			"silently entitled to nothing, so every feature-gated route fails closed "+
			"for those tenants. Add a plans row in a migration rather than special-casing "+
			"the handler; see 050_resolve_dangling_plan_default.sql", dangling)
	}
}

// The repair migration must not leak 'standard' or 'test' onto the public
// pricing page, which selects FROM plans WHERE is_active. If a future edit
// flips is_active, a legacy default tier appears as a plan on sale.
func TestLegacyDefaultTiersAreNotSellable(t *testing.T) {
	pool := purchasesPool(t)
	rows, err := pool.Query(context.Background(),
		`SELECT code FROM plans WHERE code IN ('standard','test') AND is_active`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	var active []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatalf("scan: %v", err)
		}
		active = append(active, code)
	}
	if len(active) > 0 {
		t.Fatalf("legacy default tiers %v must stay is_active=false, or they render "+
			"on /pricing as purchasable plans", active)
	}
}

// A tenant on the default 'standard' tier must actually reach invoice capture.
// This is the end-to-end form of the bug: the route answered 402 for every
// existing customer until migration 050 landed.
func TestDefaultStandardTenantIsEntitledToInvoiceCapture(t *testing.T) {
	pool := purchasesPool(t)
	var entitled bool
	err := pool.QueryRow(context.Background(),
		`SELECT COALESCE(bool_or(p.features @> '["ocr_capture"]'::jsonb), false)
		   FROM plans p WHERE p.code = 'standard'`).Scan(&entitled)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	if !entitled {
		t.Fatal("the 'standard' tier must include ocr_capture; migration 049 grants it " +
			"to every plan because the metering windows are the control surface")
	}
}

// purchasesPool returns the integration pool. testutil.DatabaseURL already
// skips the test when no TEST_DATABASE_URL / DATABASE_URL is set, so the plan
// invariants hold under `make test` offline and are enforced when a real
// Postgres is available.
func purchasesPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	return testutil.Pool(t)
}
