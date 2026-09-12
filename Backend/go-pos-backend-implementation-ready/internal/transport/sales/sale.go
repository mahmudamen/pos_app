package sales

import (
	"context"
	"errors"
	"fmt"

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

	return createSale(ctx, tx, tenantID, userID, deviceID, role, idempotencyKey, limitPct, request, requestCustomerID, requestSessionID)
}

func createSale(
	ctx context.Context,
	tx pgx.Tx,
	tenantID, userID, deviceID uuid.UUID,
	role string,
	idempotencyKey string,
	limitPct int,
	request createSaleRequest,
	requestCustomerID, requestSessionID *uuid.UUID,
) (Sale, error) {
	var subtotal int64
	currency := ""
	type lockedItem struct {
		productID uuid.UUID
		name      string
		sku       string
		quantity  int64
		price     int64
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
			SELECT id, name, sku, price_minor, currency, stock_quantity
			FROM products WHERE id = $1 AND is_active FOR UPDATE`, productID).Scan(
			&product.productID, &product.name, &product.sku, &product.price, &currency, &stock)
		if errors.Is(err, pgx.ErrNoRows) {
			return Sale{}, newSaleError(404, "product_not_found", "product not found")
		}
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to load product")
		}
		if stock < item.Quantity {
			return Sale{}, newSaleError(409, "insufficient_stock", "insufficient stock")
		}
		product.quantity = item.Quantity
		subtotal += product.price * item.Quantity
		lockedItems = append(lockedItems, product)
	}

	discount := request.DiscountMinor
	total := subtotal
	if discount > 0 {
		if !httptransport.HasPermission(role, "pos", "discount") {
			return Sale{}, newSaleError(403, "permission_denied", "discount not allowed for this role")
		}
		validated, err := validateDiscount(role, subtotal, discount, limitPct)
		if err != nil {
			return Sale{}, newSaleError(400, "discount_error", err.Error())
		}
		total = validated
	} else if discount < 0 {
		return Sale{}, newSaleError(400, "discount_error", ErrNegativeDiscount.Error())
	}

	payments, err := normalizePayments(request.Payments, total)
	if err != nil {
		return Sale{}, newSaleError(400, "validation_error", err.Error())
	}

	saleID := uuid.New()
	_, err = tx.Exec(ctx, `
		INSERT INTO sales (id, tenant_id, device_id, created_by, idempotency_key, subtotal_minor, discount_minor, total_minor, currency, payment_method, register_session_id, customer_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`, saleID, tenantID, deviceID, userID, idempotencyKey, subtotal, discount, total, currency, primaryPaymentMethod(payments), requestSessionID, requestCustomerID)
	if err != nil {
		return Sale{}, newSaleError(409, "idempotency_conflict", "idempotency key is already in use")
	}
	for _, item := range lockedItems {
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_items (tenant_id, sale_id, product_id, product_name, sku, quantity, unit_price_minor, total_minor)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, tenantID, saleID, item.productID, item.name, item.sku, item.quantity, item.price, item.price*item.quantity)
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to save sale items")
		}
		_, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity - $1 WHERE id = $2`, item.quantity, item.productID)
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to update stock")
		}
	}
	for _, p := range payments {
		_, err = tx.Exec(ctx, `
			INSERT INTO sale_payments (tenant_id, sale_id, method, amount_minor)
			VALUES ($1, $2, $3, $4)`, tenantID, saleID, p.Method, p.AmountMinor)
		if err != nil {
			return Sale{}, newSaleError(500, "internal_error", "unable to save sale payments")
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
