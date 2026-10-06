package onboarding

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Goal ids are the server's business outcomes. Clients never decide them.
const (
	GoalStoreConfigured    = "store_configured"
	GoalProductReady       = "product_ready"
	GoalTerminalConfigured = "terminal_configured"
	GoalCashSessionOpened  = "cash_session_opened"
	GoalFirstSaleCompleted = "first_sale_completed"
)

type tourDef struct {
	ID        string
	Version   int
	Goal      string
	StepCount int
}

// catalog is the tour library in activation order. Versions bump when the
// steps of a tour change so old clients get tour_version_mismatch instead of
// completing against stale definitions.
var catalog = []tourDef{
	{ID: "store_setup", Version: 1, Goal: GoalStoreConfigured, StepCount: 3},
	{ID: "first_product", Version: 1, Goal: GoalProductReady, StepCount: 3},
	{ID: "terminal_setup", Version: 1, Goal: GoalTerminalConfigured, StepCount: 3},
	{ID: "cash_session", Version: 1, Goal: GoalCashSessionOpened, StepCount: 4},
	{ID: "first_sale", Version: 1, Goal: GoalFirstSaleCompleted, StepCount: 5},
}

func lookupTour(id string) (tourDef, bool) {
	for _, t := range catalog {
		if t.ID == id {
			return t, true
		}
	}
	return tourDef{}, false
}

// goalSatisfied answers one business question per goal with a real query
// inside the caller's tenant-scoped transaction. Every subquery must be
// tenant-pinned: most tenant tables carry FORCE RLS, but `tenants` itself has
// no RLS at all, so an unfiltered EXISTS over it would read every tenant.
func goalSatisfied(ctx context.Context, tx pgx.Tx, tenantID, goal string) (bool, error) {
	var ok bool
	var err error
	switch goal {
	case GoalStoreConfigured:
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM tenants
				WHERE id = $1::uuid AND btrim(COALESCE(address, '')) <> ''
			) OR EXISTS (
				SELECT 1 FROM categories WHERE is_active
			) OR EXISTS (
				SELECT 1 FROM products WHERE is_active
			)`, tenantID).Scan(&ok)
	case GoalProductReady:
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM products WHERE is_active)`).Scan(&ok)
	case GoalTerminalConfigured:
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM register_sessions)`).Scan(&ok)
	case GoalCashSessionOpened:
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM register_sessions WHERE status = 'open')`).Scan(&ok)
	case GoalFirstSaleCompleted:
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sales s
				WHERE s.status = 'completed'
				  AND EXISTS (SELECT 1 FROM sale_payments p WHERE p.sale_id = s.id)
			)`).Scan(&ok)
	default:
		return false, fmt.Errorf("unknown goal %q", goal)
	}
	if err != nil {
		return false, err
	}
	return ok, nil
}

// readiness is the full activation picture for the tenant.
type readiness struct {
	StoreConfigured    bool `json:"store_configured"`
	ProductReady       bool `json:"product_ready"`
	TerminalConfigured bool `json:"terminal_configured"`
	CashSessionOpened  bool `json:"cash_session_opened"`
	FirstSaleCompleted bool `json:"first_sale_completed"`
	Activated          bool `json:"activated"`
}

// readinessInTx evaluates every goal inside an already tenant-scoped
// transaction so one request never needs two connections.
func readinessInTx(ctx context.Context, tx pgx.Tx, tenantID string) (readiness, error) {
	out := readiness{}
	checks := []struct {
		goal string
		dst  *bool
	}{
		{GoalStoreConfigured, &out.StoreConfigured},
		{GoalProductReady, &out.ProductReady},
		{GoalTerminalConfigured, &out.TerminalConfigured},
		{GoalCashSessionOpened, &out.CashSessionOpened},
		{GoalFirstSaleCompleted, &out.FirstSaleCompleted},
	}
	for _, check := range checks {
		ok, err := goalSatisfied(ctx, tx, tenantID, check.goal)
		if err != nil {
			return readiness{}, err
		}
		*check.dst = ok
	}
	out.Activated = out.FirstSaleCompleted
	return out, nil
}
