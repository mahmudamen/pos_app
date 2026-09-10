package registers

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// queryOpenSession returns the single open session for a terminal (tenant +
// device), or pgx.ErrNoRows when the terminal has no open session.
func queryOpenSession(ctx context.Context, tx pgx.Tx, tenantID, deviceID string) (sessionRow, error) {
	var row sessionRow
	err := tx.QueryRow(ctx, `
		SELECT rs.id, rs.status, rs.opening_cash_minor, rs.closing_cash_minor,
		       rs.expected_cash_minor, rs.cash_difference_minor, rs.opened_at::text,
		       rs.closed_at::text, u.display_name
		FROM register_sessions rs
		JOIN users u ON u.tenant_id = rs.tenant_id AND u.id = rs.user_id
		WHERE rs.tenant_id = $1 AND rs.device_id = $2 AND rs.status = 'open'
		LIMIT 1`, tenantID, deviceID).
		Scan(&row.id, &row.status, &row.openingCashMinor, &row.closingCashMinor,
			&row.expectedCashMinor, &row.cashDifferenceMinor, &row.openedAt, &row.closedAt, &row.openedBy)
	return row, err
}

// queryTotals aggregates the Z-report numbers across all sales attached to a
// session, including the per-method tender split from sale_payments. Sale
// aggregates are computed over distinct sales while the per-method sums walk
// one row per payment line (split tender must not double-count sales).
func queryTotals(ctx context.Context, tx pgx.Tx, sessionID string) (accountTotals, error) {
	var totals accountTotals
	err := tx.QueryRow(ctx, `
		WITH sale_agg AS (
			SELECT s.id, s.tenant_id, s.subtotal_minor, s.discount_minor, s.tax_minor, s.total_minor
			FROM sales s
			WHERE s.register_session_id = $1
		)
		SELECT
			(SELECT COUNT(*) FROM sale_agg),
			(SELECT COALESCE(SUM(subtotal_minor), 0) FROM sale_agg),
			(SELECT COALESCE(SUM(discount_minor), 0) FROM sale_agg),
			(SELECT COALESCE(SUM(tax_minor), 0) FROM sale_agg),
			(SELECT COALESCE(SUM(total_minor), 0) FROM sale_agg),
			COALESCE(SUM(sp.amount_minor) FILTER (WHERE sp.method = 'cash'), 0),
			COALESCE(SUM(sp.amount_minor) FILTER (WHERE sp.method = 'card'), 0),
			COALESCE(SUM(sp.amount_minor) FILTER (WHERE sp.method = 'mobile'), 0)
		FROM sale_agg s
		LEFT JOIN sale_payments sp ON sp.tenant_id = s.tenant_id AND sp.sale_id = s.id`, sessionID).
		Scan(&totals.salesCount, &totals.subtotalMinor, &totals.discountMinor,
			&totals.taxMinor, &totals.totalMinor, &totals.cashMinor,
			&totals.cardMinor, &totals.mobileMinor)
	return totals, err
}
