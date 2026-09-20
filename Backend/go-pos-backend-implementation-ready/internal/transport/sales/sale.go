package sales

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/transport/access"
	httptransport "github.com/example/pos-api/internal/transport/http"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// saleError carries the HTTP-facing fields for a failed sale application so
// the sync push machinery can record a typed "rejected" outcome without a gin
// context.
type saleError struct {
	status  int
	code    string
	message string
}

func (e *saleError) Error() string {
	return e.message
}

func newSaleError(status int, code, message string) *saleError {
	return &saleError{status: status, code: code, message: message}
}

// CreateSaleRequest aliases the (private) createSaleRequest binding type so
// the sync package can decode "sale.create" payloads without duplicating it.
type CreateSaleRequest = createSaleRequest

// SaleItemRequest aliases the line-item binding type so sibling packages
// (selforder, sync) can build a CreateSaleRequest without duplicating it.
type SaleItemRequest = saleItemRequest

// CreateSale applies a sale inside the caller's transaction. The tx must
// already have the tenant context set. It never commits and never touches the
// HTTP response; callers decide how to surface the outcome.
func CreateSale(
	ctx context.Context,
	tx pgx.Tx,
	tenantID, userID, deviceID uuid.UUID,
	role string,
	idempotencyKey string,
	limitPct int,
	request createSaleRequest,
) (Sale, error) {
	if err := ctx.Err(); err != nil {
		return Sale{}, newSaleError(499, "context_cancelled", "request cancelled")
	}

	var requestCustomerID *uuid.UUID
	if request.CustomerID != "" {
		parsed, parseErr := uuid.Parse(request.CustomerID)
		if parseErr != nil {
			return Sale{}, newSaleError(400, "validation_error", "customer_id is invalid")
		}
		var exists bool
		err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1 AND tenant_id = $2)`, parsed, tenantID).Scan(&exists)
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to check customer")
		}
		if !exists {
			return Sale{}, newSaleError(404, "customer_not_found", "customer not found")
		}
		requestCustomerID = &parsed
	}

	var requestSessionID *uuid.UUID
	if request.RegisterSessionID != "" {
		parsed, parseErr := uuid.Parse(request.RegisterSessionID)
		if parseErr != nil {
			return Sale{}, newSaleError(400, "validation_error", "session_id is invalid")
		}
		var status string
		err := tx.QueryRow(ctx, `
			SELECT status FROM register_sessions
			WHERE id = $1 AND tenant_id = $2 AND device_id = $3`,
			parsed, tenantID, deviceID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return Sale{}, newSaleError(404, "session_not_found", "session not found on this terminal")
		}
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to check session")
		}
		if status != "open" {
			return Sale{}, newSaleError(409, "session_not_open", "session is already closed")
		}
		requestSessionID = &parsed
	}

	var requestTableID *uuid.UUID
	if request.TableID != "" {
		parsed, parseErr := uuid.Parse(request.TableID)
		if parseErr != nil {
			return Sale{}, newSaleError(400, "validation_error", "table_id is invalid")
		}
		var tableStatus string
		err := tx.QueryRow(ctx, `SELECT status FROM restaurant_tables WHERE id = $1 AND tenant_id = $2`, parsed, tenantID).Scan(&tableStatus)
		if errors.Is(err, pgx.ErrNoRows) {
			return Sale{}, newSaleError(404, "table_not_found", "table not found")
		}
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to check table")
		}
		if tableStatus == "closed" {
			return Sale{}, newSaleError(409, "table_closed", "table is closed")
		}
		requestTableID = &parsed
	}

	var existing Sale
	err := tx.QueryRow(ctx, `SELECT id, status, subtotal_minor, discount_minor, tax_minor, total_minor, currency,
		COALESCE(payment_method, 'cash'), COALESCE(customer_id::text, ''),
		COALESCE((SELECT SUM(points_delta) FROM customer_loyalty_log cl WHERE cl.sale_id = sales.id), 0),
		created_at::text FROM sales WHERE idempotency_key = $1`, idempotencyKey).Scan(
		&existing.ID, &existing.Status, &existing.SubtotalMinor, &existing.DiscountMinor, &existing.TaxMinor, &existing.TotalMinor, &existing.Currency, &existing.PaymentMethod, &existing.CustomerID, &existing.LoyaltyPointsEarned, &existing.CreatedAt)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Sale{}, newSaleError(500, "internal_error", "unable to check idempotency")
	}

	return createSale(ctx, tx, tenantID, userID, deviceID, role, idempotencyKey, limitPct, request, requestCustomerID, requestSessionID, requestTableID)
}

type lockedItem struct {
	productID   uuid.UUID
	name        string
	sku         string
	quantity    int64
	price       int64
	hasVariants bool
	trackLots   bool
	variantID   *uuid.UUID
	variantName string
}

// allowNegativeStock reports whether the tenant permits the on-hand counter
// to go below zero. Defaults to false (strict stock) when unset.
func allowNegativeStock(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (bool, error) {
	var allowed bool
	err := tx.QueryRow(ctx, `
		SELECT COALESCE((SELECT value FROM tenant_settings
		                 WHERE tenant_id = $1 AND key = 'inventory.allow_negative_stock'), 'false') = 'true'`,
		tenantID).Scan(&allowed)
	if err != nil {
		return false, err
	}
	return allowed, nil
}

// discountPolicy holds the tenant-wide discount enforcement inputs.
type discountPolicy struct {
	Mode            DiscountMode
	GlobalMaxPct    int
	ManagerRequired bool
}

// loadDiscountPolicy reads the ma_pos_base discount settings. Defaults match
// the settings module: cap mode, no global ceiling, manager PIN required for
// blocked over-limit discounts.
func loadDiscountPolicy(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (discountPolicy, error) {
	var mode, maxPct, managerRequired string
	rows, err := tx.Query(ctx, `
		SELECT key, value FROM tenant_settings WHERE tenant_id = $1
		AND key IN ('pos.discount_mode', 'pos.max_discount_pct', 'pos.manager.discount')`, tenantID)
	if err != nil {
		return discountPolicy{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return discountPolicy{}, err
		}
		switch k {
		case "pos.discount_mode":
			mode = v
		case "pos.max_discount_pct":
			maxPct = v
		case "pos.manager.discount":
			managerRequired = v
		}
	}
	if err := rows.Err(); err != nil {
		return discountPolicy{}, err
	}
	policy := discountPolicy{Mode: ParseDiscountMode(mode), ManagerRequired: true}
	if managerRequired == "false" {
		policy.ManagerRequired = false
	}
	if n, parseErr := strconv.Atoi(maxPct); parseErr == nil && n > 0 && n <= 100 {
		policy.GlobalMaxPct = n
	}
	return policy, nil
}

func createSale(
	ctx context.Context,
	tx pgx.Tx,
	tenantID, userID, deviceID uuid.UUID,
	role string,
	idempotencyKey string,
	limitPct int,
	request createSaleRequest,
	requestCustomerID, requestSessionID, requestTableID *uuid.UUID,
) (Sale, error) {
	var subtotal int64
	currency := ""
	allowNegative, err := allowNegativeStock(ctx, tx, tenantID)
	if err != nil {
		return Sale{}, newSaleError(500, "internal_error", "unable to read inventory settings")
	}
	lockedItems := make([]lockedItem, 0, len(request.Items))
	for _, item := range request.Items {
		productID, parseErr := uuid.Parse(item.ProductID)
		if parseErr != nil {
			return Sale{}, newSaleError(400, "validation_error", "product_id is invalid")
		}
		var product lockedItem
		var stock int64
		err := tx.QueryRow(ctx, `
			SELECT id, name, sku, price_minor, currency, stock_quantity, has_variants, track_lots
			FROM products WHERE id = $1 AND is_active FOR UPDATE`, productID).Scan(
			&product.productID, &product.name, &product.sku, &product.price, &currency, &stock, &product.hasVariants, &product.trackLots)
		if errors.Is(err, pgx.ErrNoRows) {
			return Sale{}, newSaleError(404, "product_not_found", "product not found")
		}
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to load product")
		}
		if product.hasVariants {
			// Variant is authoritative for price and stock.
			if item.VariantID == "" {
				return Sale{}, newSaleError(400, "validation_error", "variant_id is required for this product")
			}
			variantID, parseErr := uuid.Parse(item.VariantID)
			if parseErr != nil {
				return Sale{}, newSaleError(400, "validation_error", "variant_id is invalid")
			}
			var vStock int64
			err := tx.QueryRow(ctx, `
				SELECT name, sku, price_minor, stock_quantity
				FROM product_variants WHERE id = $1 AND product_id = $2 AND is_active FOR UPDATE`,
				variantID, productID).Scan(&product.variantName, &product.sku, &product.price, &vStock)
			if errors.Is(err, pgx.ErrNoRows) {
				return Sale{}, newSaleError(404, "variant_not_found", "variant not found")
			}
			if err != nil {
				return Sale{}, newSaleError(500, "internal_error", "unable to load variant")
			}
			if vStock < item.Quantity && !allowNegative {
				return Sale{}, newSaleError(409, "insufficient_stock", "insufficient variant stock")
			}
			product.variantID = &variantID
		} else {
			if item.VariantID != "" {
				return Sale{}, newSaleError(400, "validation_error", "product does not have variants")
			}
			// Lot-tracked stock is consumed physically lot-by-lot (FEFO), so
			// it can never be oversold even when backorders are enabled.
			if product.trackLots {
				if stock < item.Quantity {
					return Sale{}, newSaleError(409, "insufficient_stock", "insufficient stock")
				}
			} else if stock < item.Quantity && !allowNegative {
				return Sale{}, newSaleError(409, "insufficient_stock", "insufficient stock")
			}
		}
		product.quantity = item.Quantity
		subtotal += product.price * item.Quantity
		lockedItems = append(lockedItems, product)
	}

	discount := request.DiscountMinor
	total := subtotal
	var discountCapped bool
	var discountWarning string
	if discount > 0 {
		if !httptransport.HasPermission(role, "pos", "discount") {
			return Sale{}, newSaleError(403, "permission_denied", "discount not allowed for this role")
		}
		perms := access.ResolveFromDB(ctx, tx, tenantID.String(), userID.String(), role)
		policy, err := loadDiscountPolicy(ctx, tx, tenantID)
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to read discount policy")
		}
		decision, err := EvaluateDiscount(perms, policy.Mode, policy.GlobalMaxPct, subtotal, discount)
		if err != nil {
			return Sale{}, newSaleError(400, "discount_error", err.Error())
		}
		if decision.NeedsManager {
			if !policy.ManagerRequired {
				return Sale{}, newSaleError(403, ErrDiscountOverLimit.Error(), ErrDiscountOverLimit.Error())
			}
			if request.ManagerPIN == "" {
				return Sale{}, newSaleError(403, "discount_manager_pin_required", ErrDiscountManagerPINRequired.Error())
			}
			var pinHash string
			if err = tx.QueryRow(ctx, `SELECT COALESCE(manager_pin_hash, '') FROM users WHERE id = $1::uuid`, userID).Scan(&pinHash); err != nil {
				return Sale{}, newSaleError(500, "internal_error", "unable to verify manager PIN")
			}
			if pinHash == "" || !security.CheckPassword(pinHash, request.ManagerPIN) {
				return Sale{}, newSaleError(403, "invalid_pin", ErrInvalidManagerPIN.Error())
			}
		}
		total = decision.TotalAfter
		discount = decision.AppliedDiscount
		discountCapped = decision.Capped
		discountWarning = decision.Warning
	} else if discount < 0 {
		return Sale{}, newSaleError(400, "discount_error", ErrNegativeDiscount.Error())
	}

	payments, tipsMinor, err := normalizePayments(request.Payments, total)
	if err != nil {
		return Sale{}, newSaleError(400, "validation_error", err.Error())
	}

	saleID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO sales (id, tenant_id, device_id, created_by, idempotency_key, subtotal_minor, discount_minor, total_minor, currency, payment_method, register_session_id, customer_id, table_id, tips_minor)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`, saleID, tenantID, deviceID, userID, idempotencyKey, subtotal, discount, total, currency, primaryPaymentMethod(payments), requestSessionID, requestCustomerID, requestTableID, tipsMinor)
	if err != nil {
		return Sale{}, newSaleError(409, "idempotency_conflict", "idempotency key is already in use")
	}
	for _, item := range lockedItems {
		switch {
		case item.trackLots:
			// First-expiry-first-out: consume lots oldest-first and record each
			// consumed lot on its own sale_item line for recall.
			if _, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2`, item.quantity, item.productID); err != nil {
				return Sale{}, newSaleError(500, "internal_error", "unable to update stock")
			}
			remaining := item.quantity
			for remaining > 0 {
				var lotID uuid.UUID
				var lotNumber string
				var lotQty int64
				err := tx.QueryRow(ctx, `
					SELECT id, lot_number, quantity FROM product_lots
					WHERE product_id = $1 AND quantity > 0
					ORDER BY expiry_date NULLS LAST, created_at
					LIMIT 1 FOR UPDATE`, item.productID).Scan(&lotID, &lotNumber, &lotQty)
				if errors.Is(err, pgx.ErrNoRows) {
					return Sale{}, newSaleError(409, "lot_not_found", "product has no lots covering the sale quantity")
				}
				if err != nil {
					return Sale{}, newSaleError(500, "internal_error", "unable to read lot")
				}
				consumed := remaining
				if consumed > lotQty {
					consumed = lotQty
				}
				if _, err = tx.Exec(ctx, `UPDATE product_lots SET quantity = quantity - $1 WHERE id = $2`, consumed, lotID); err != nil {
					return Sale{}, newSaleError(500, "internal_error", "unable to decrement lot")
				}
				if _, err = tx.Exec(ctx, `
					INSERT INTO sale_items (tenant_id, sale_id, product_id, product_name, sku, quantity, unit_price_minor, total_minor, lot_id, lot_number)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
					tenantID, saleID, item.productID, item.name, item.sku, consumed, item.price, item.price*consumed, lotID, lotNumber); err != nil {
					return Sale{}, newSaleError(500, "internal_error", "unable to save sale items")
				}
				remaining -= consumed
			}
		default:
			if item.variantID != nil {
				// Variants are authoritative for stock; the parent template
				// stock_quantity is informational and not decremented.
				if _, err = tx.Exec(ctx, `UPDATE product_variants SET stock_quantity = stock_quantity - $1 WHERE id = $2`, item.quantity, *item.variantID); err != nil {
					return Sale{}, newSaleError(500, "internal_error", "unable to update variant stock")
				}
			} else {
				if _, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2`, item.quantity, item.productID); err != nil {
					return Sale{}, newSaleError(500, "internal_error", "unable to update stock")
				}
			}
			if _, err = tx.Exec(ctx, `
				INSERT INTO sale_items (tenant_id, sale_id, product_id, product_name, sku, quantity, unit_price_minor, total_minor, variant_id, variant_name)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
				tenantID, saleID, item.productID, item.name, item.sku, item.quantity, item.price, item.price*item.quantity, item.variantID, item.variantName); err != nil {
				return Sale{}, newSaleError(500, "internal_error", "unable to save sale items")
			}
		}
	}
	for _, p := range payments {
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_payments (tenant_id, sale_id, method, amount_minor, tip_minor)
			VALUES ($1, $2, $3, $4, $5)`, tenantID, saleID, p.Method, p.AmountMinor, p.TipMinor)
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to save sale payments")
		}
	}
	if requestTableID != nil {
		if _, err = tx.Exec(ctx, `UPDATE restaurant_tables SET status = 'occupied' WHERE id = $1`, *requestTableID); err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to mark table occupied")
		}
	}

	loyaltyPoints := int64(0)
	if requestCustomerID != nil {
		loyaltyPoints, err = loyaltyPointsForSale(ctx, tx, total)
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to read loyalty settings")
		}
		if loyaltyPoints > 0 {
			_, err = tx.Exec(ctx, `
				UPDATE customers
				SET loyalty_points = loyalty_points + $1,
				    loyalty_points_total = loyalty_points_total + $1,
				    updated_at = NOW()
				WHERE id = $2`, loyaltyPoints, requestCustomerID)
			if err != nil {
				return Sale{}, newSaleError(500, "internal_error", "unable to update customer loyalty")
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO customer_loyalty_log (tenant_id, customer_id, sale_id, points_delta, reason)
				VALUES ($1, $2, $3, $4, 'sale')`, tenantID, requestCustomerID, saleID, loyaltyPoints)
			if err != nil {
				return Sale{}, newSaleError(500, "internal_error", "unable to record loyalty")
			}
		}
	}

	return Sale{
		ID:                  saleID.String(),
		Status:              "completed",
		SubtotalMinor:       subtotal,
		DiscountMinor:       discount,
		TaxMinor:            0,
		TotalMinor:          total,
		Currency:            currency,
		PaymentMethod:       primaryPaymentMethod(payments),
		CustomerID:          saleCustomerID(requestCustomerID),
		LoyaltyPointsEarned: loyaltyPoints,
		TipsMinor:           tipsMinor,
		TableID:             saleTableID(requestTableID),
		DiscountCapped:      discountCapped,
		DiscountWarning:     discountWarning,
	}, nil
}

// SaleError exposes a rejected sale for observability (logs, sync command
// results) without leaking internals.
func SaleError(err error) (status int, code, message string) {
	if err == nil {
		return 0, "", ""
	}
	if se, ok := err.(*saleError); ok {
		return se.status, se.code, se.message
	}
	return 500, "internal_error", fmt.Sprintf("unable to apply sale: %v", err)
}
