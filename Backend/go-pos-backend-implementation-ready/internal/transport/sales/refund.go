package sales

import (
	"context"
	"errors"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RefundRequest carries the optional reason plus the manager PIN that must
// match the acting user's configured PIN (when one is set).
type RefundRequest struct {
	Reason     string `json:"reason"`
	ManagerPIN string `json:"manager_pin"`
}

// Refund is one refund transaction against a completed sale.
type Refund struct {
	ID          string `json:"id"`
	SaleID      string `json:"sale_id"`
	RefundMinor int64  `json:"refund_minor"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
}

// RefundSale reverses a completed sale inside the caller's transaction (the tx
// must already have the tenant context set). It restores stock to the exact
// lots/variants/products the sale consumed, releases the restaurant table,
// claws back the loyalty points the sale earned, records the refund in
// sale_refunds, and marks the sale status 'refunded'. It is idempotent on
// (tenant_id, idempotency_key) so a retried POST never double-refunds.
func RefundSale(
	ctx context.Context,
	tx pgx.Tx,
	tenantID, userID uuid.UUID,
	role string,
	saleID uuid.UUID,
	idempotencyKey string,
	request RefundRequest,
) (Refund, error) {
	if err := ctx.Err(); err != nil {
		return Refund{}, newSaleError(499, "context_cancelled", "request cancelled")
	}
	if !httptransport.HasPermission(role, "pos", "refund") {
		return Refund{}, newSaleError(403, "permission_denied", "refunds not allowed for this role")
	}
	reason := strings.TrimSpace(request.Reason)
	if len(reason) > 255 {
		return Refund{}, newSaleError(400, "validation_error", "reason is too long")
	}

	// Idempotent replay: a refund already recorded under this key is the same
	// refund, whatever happened in between.
	var existing Refund
	err := tx.QueryRow(ctx, `
		SELECT id, sale_id::text, refund_minor, COALESCE(reason, ''), status, COALESCE(created_by::text, ''), created_at::text
		FROM sale_refunds WHERE idempotency_key = $1`, idempotencyKey).Scan(
		&existing.ID, &existing.SaleID, &existing.RefundMinor, &existing.Reason, &existing.Status, &existing.CreatedBy, &existing.CreatedAt)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Refund{}, newSaleError(500, "internal_error", "unable to check refund idempotency")
	}

	// Load and lock the sale so a concurrent refund on the same sale cannot
	// sneak in between the read and the mark.
	var saleStatus, customerID, tableID string
	var totalMinor int64
	err = tx.QueryRow(ctx, `
		SELECT status, total_minor, COALESCE(customer_id::text, ''), COALESCE(table_id::text, '')
		FROM sales WHERE id = $1 FOR UPDATE`, saleID).Scan(&saleStatus, &totalMinor, &customerID, &tableID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Refund{}, newSaleError(404, "sale_not_found", "sale not found")
	}
	if err != nil {
		return Refund{}, newSaleError(500, "internal_error", "unable to load sale")
	}
	if saleStatus != "completed" {
		return Refund{}, newSaleError(409, "sale_not_refundable", "only completed sales can be refunded")
	}

	// B1 PIN gate: when the acting user has configured a manager PIN, a refund
	// must be confirmed with it. No PIN configured → no gate.
	var pinHash string
	err = tx.QueryRow(ctx, `SELECT COALESCE(manager_pin_hash, '') FROM users WHERE id = $1`, userID).Scan(&pinHash)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Refund{}, newSaleError(500, "internal_error", "unable to read user")
	}
	if pinHash != "" {
		if request.ManagerPIN == "" || !security.CheckPassword(pinHash, request.ManagerPIN) {
			return Refund{}, newSaleError(403, "invalid_pin", "a valid manager PIN is required to refund")
		}
	}

	rows, err := tx.Query(ctx, `
		SELECT product_id, quantity, COALESCE(variant_id::text, ''), COALESCE(lot_id::text, '')
		FROM sale_items WHERE sale_id = $1`, saleID)
	if err != nil {
		return Refund{}, newSaleError(500, "internal_error", "unable to load sale items")
	}
	type restoreItem struct {
		productID uuid.UUID
		qty       int64
		variantID string
		lotID     string
	}
	var items []restoreItem
	for rows.Next() {
		var it restoreItem
		if err := rows.Scan(&it.productID, &it.qty, &it.variantID, &it.lotID); err != nil {
			rows.Close()
			return Refund{}, newSaleError(500, "internal_error", "unable to decode sale items")
		}
		items = append(items, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return Refund{}, newSaleError(500, "internal_error", "unable to read sale items")
	}
	// The tx's single connection is busy while a result set stays open, so all
	// writes batch strictly after rows.Close().
	for _, it := range items {
		switch {
		case it.lotID != "":
			// Lots re-join the FEFO pool and the parent product regains stock,
			// mirroring the buy-side decrement.
			if _, err = tx.Exec(ctx, `UPDATE product_lots SET quantity = quantity + $1 WHERE id = $2`, it.qty, it.lotID); err != nil {
				return Refund{}, newSaleError(500, "internal_error", "unable to restore lot")
			}
			if _, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, it.qty, it.productID); err != nil {
				return Refund{}, newSaleError(500, "internal_error", "unable to restore product stock")
			}
		case it.variantID != "":
			// Variant stock is authoritative; the parent template is not
			// touched, mirroring the buy-side.
			if _, err = tx.Exec(ctx, `UPDATE product_variants SET stock_quantity = stock_quantity + $1 WHERE id = $2`, it.qty, it.variantID); err != nil {
				return Refund{}, newSaleError(500, "internal_error", "unable to restore variant stock")
			}
		default:
			if _, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, it.qty, it.productID); err != nil {
				return Refund{}, newSaleError(500, "internal_error", "unable to restore product stock")
			}
		}
	}

	if tableID != "" {
		if _, err = tx.Exec(ctx, `UPDATE restaurant_tables SET status = 'available' WHERE id = $1::uuid`, tableID); err != nil {
			return Refund{}, newSaleError(500, "internal_error", "unable to release table")
		}
	}

	if customerID != "" {
		loyaltyPoints, err := loyaltyPointsForSale(ctx, tx, totalMinor)
		if err != nil {
			return Refund{}, newSaleError(500, "internal_error", "unable to read loyalty settings")
		}
		if loyaltyPoints > 0 {
			if _, err = tx.Exec(ctx, `
				UPDATE customers
				SET loyalty_points = GREATEST(loyalty_points - $1, 0),
				    updated_at = NOW()
				WHERE id = $2::uuid`, loyaltyPoints, customerID); err != nil {
				return Refund{}, newSaleError(500, "internal_error", "unable to update customer loyalty")
			}
			if _, err = tx.Exec(ctx, `
				INSERT INTO customer_loyalty_log (tenant_id, customer_id, sale_id, points_delta, reason)
				VALUES ($1, $2, $3, $4, 'refund')`, tenantID, customerID, saleID, -loyaltyPoints); err != nil {
				return Refund{}, newSaleError(500, "internal_error", "unable to record loyalty claw-back")
			}
		}
	}

	refundID := uuid.New()
	var createdByName, createdAt string
	err = tx.QueryRow(ctx, `
		INSERT INTO sale_refunds (id, tenant_id, sale_id, created_by, idempotency_key, refund_minor, reason)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''))
		RETURNING created_at::text,
		          (SELECT display_name FROM users u WHERE u.tenant_id = sale_refunds.tenant_id AND u.id = sale_refunds.created_by)`,
		refundID, tenantID, saleID, userID, idempotencyKey, totalMinor, reason).Scan(&createdAt, &createdByName)
	if err != nil {
		// A concurrent refund slipped through with the same key after our
		// select: replay that stored outcome instead of erroring.
		var raced Refund
		if rerr := tx.QueryRow(ctx, `
			SELECT id, sale_id::text, refund_minor, COALESCE(reason, ''), status, COALESCE(created_by::text, ''), created_at::text
			FROM sale_refunds WHERE idempotency_key = $1`, idempotencyKey).Scan(
			&raced.ID, &raced.SaleID, &raced.RefundMinor, &raced.Reason, &raced.Status, &raced.CreatedBy, &raced.CreatedAt); rerr == nil {
			return raced, nil
		}
		return Refund{}, newSaleError(409, "idempotency_conflict", "idempotency key is already in use")
	}
	if _, err = tx.Exec(ctx, `UPDATE sales SET status = 'refunded' WHERE id = $1`, saleID); err != nil {
		return Refund{}, newSaleError(500, "internal_error", "unable to mark sale refunded")
	}

	return Refund{
		ID:          refundID.String(),
		SaleID:      saleID.String(),
		RefundMinor: totalMinor,
		Reason:      reason,
		Status:      "completed",
		CreatedBy:   createdByName,
		CreatedAt:   createdAt,
	}, nil
}
