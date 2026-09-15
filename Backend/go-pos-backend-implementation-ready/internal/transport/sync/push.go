package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	httptransport "github.com/example/pos-api/internal/transport/http"
	businesssales "github.com/example/pos-api/internal/transport/sales"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	defaultPushLimit        = 100
	defaultDiscountLimitPct = 5
	idempotencyKeyKey       = "idempotency_key"
)

type pushCommand struct {
	CommandID  string         `json:"command_id" binding:"required"`
	Operation  string         `json:"operation" binding:"required"`
	Payload    map[string]any `json:"payload" binding:"required"`
	ClientTime string         `json:"client_time"`
}

type pushRequest struct {
	Commands []pushCommand `json:"commands" binding:"required,min=1"`
}

type pushResult struct {
	CommandID   string         `json:"command_id"`
	Status      string         `json:"status"`
	Replayed    bool           `json:"replayed"`
	Result      map[string]any `json:"result,omitempty"`
	ErrorCode   string         `json:"error_code,omitempty"`
	ErrorDetail string         `json:"error_detail,omitempty"`
}

type pushResponse struct {
	Results []pushResult `json:"results"`
}

func (h *Handler) push(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	deviceID, err := uuid.Parse(claims.DeviceID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	var req pushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid push request")
		return
	}
	if len(req.Commands) > defaultPushLimit {
		writeError(c, http.StatusBadRequest, "validation_error", "too many commands in one push")
		return
	}

	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}

	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID.String()); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to establish tenant context")
		return
	}

	results := make([]pushResult, 0, len(req.Commands))
	for _, cmd := range req.Commands {
		result := applyCommand(ctx, tx, h, cmd, tenantID, userID, deviceID, claims.Role)
		results = append(results, result)
	}

	if err = tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit sync push")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": pushResponse{Results: results},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

func applyCommand(ctx context.Context, tx pgx.Tx, h *Handler, cmd pushCommand, tenantID, userID, deviceID uuid.UUID, role string) pushResult {
	commandID, err := uuid.Parse(cmd.CommandID)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "command_id is invalid"}
	}

	payloadHash := hashPayload(cmd.Payload)

	inserted, stored, err := recordCommand(ctx, tx, commandID, deviceID, cmd.Operation, payloadHash, cmd.Payload, tenantID)
	if !inserted {
		if errors.Is(err, errCommandConflict) {
			return pushResult{CommandID: cmd.CommandID, Status: "conflict", ErrorCode: "command_conflict", ErrorDetail: "command_id is already in use with a different payload"}
		}
		if err != nil {
			return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to record command"}
		}
		// Duplicate (identical payload already recorded): reply to the caller
		// under its own command_id so the client can correlate the replay.
		stored.CommandID = cmd.CommandID
		return stored
	}

	result := apply(ctx, tx, h, cmd, tenantID, userID, deviceID, role)
	if err := writeCommandOutcome(ctx, tx, commandID, result); err != nil {
		result.Status = "rejected"
		result.ErrorCode = "internal_error"
		result.ErrorDetail = "unable to finalize command"
	}
	return result
}

func apply(ctx context.Context, tx pgx.Tx, h *Handler, cmd pushCommand, tenantID, userID, deviceID uuid.UUID, role string) pushResult {
	switch cmd.Operation {
	case "sale.create":
		return applyCreateSale(ctx, tx, cmd, tenantID, userID, deviceID, role)
	case "sale.refund":
		return applyRefundSale(ctx, tx, cmd, tenantID, userID, role)
	case "inventory.adjust":
		return applyInventoryAdjust(ctx, tx, cmd, tenantID, userID, role)
	case "category.create":
		return applyCategoryCreate(ctx, tx, cmd, tenantID, role)
	case "product.create":
		return applyProductCreate(ctx, tx, cmd, tenantID, role)
	default:
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "unknown_command", ErrorDetail: "unknown operation: " + cmd.Operation}
	}
}

func payloadString(payload map[string]any, key string) string {
	if v, ok := payload[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func payloadInt64(payload map[string]any, key string) (int64, bool) {
	switch v := payload[key].(type) {
	case float64:
		return int64(v), true
	case int64:
		return v, true
	case int:
		return int64(v), true
	case json.Number:
		n, err := v.Int64()
		return n, err == nil
	}
	return 0, false
}

func applyRefundSale(ctx context.Context, tx pgx.Tx, cmd pushCommand, tenantID, userID uuid.UUID, role string) pushResult {
	saleID, err := uuid.Parse(payloadString(cmd.Payload, "sale_id"))
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "sale_id is invalid"}
	}
	idempotencyKey := payloadString(cmd.Payload, idempotencyKeyKey)
	if idempotencyKey == "" {
		idempotencyKey = "sync:" + cmd.CommandID
	}
	marshaled, err := json.Marshal(cmd.Payload)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "invalid payload"}
	}
	var request businesssales.RefundRequest
	if err := json.Unmarshal(marshaled, &request); err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "invalid refund payload"}
	}

	refund, applyErr := businesssales.RefundSale(ctx, tx, tenantID, userID, role, saleID, idempotencyKey, request)
	if applyErr != nil {
		status, code, message := businesssales.SaleError(applyErr)
		return pushResult{CommandID: cmd.CommandID, Status: pushStatusFor(status), ErrorCode: code, ErrorDetail: message}
	}
	resultJSON := map[string]any{
		"id": refund.ID, "sale_id": refund.SaleID, "refund_minor": refund.RefundMinor,
		"reason": refund.Reason, "status": refund.Status, "created_at": refund.CreatedAt,
	}
	return pushResult{CommandID: cmd.CommandID, Status: "applied", Result: resultJSON}
}

func applyInventoryAdjust(ctx context.Context, tx pgx.Tx, cmd pushCommand, tenantID, userID uuid.UUID, role string) pushResult {
	productID, err := uuid.Parse(payloadString(cmd.Payload, "product_id"))
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "product_id is invalid"}
	}
	reason := payloadString(cmd.Payload, "reason")
	switch reason {
	case "damaged", "restock", "count":
	default:
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "reason must be one of damaged, restock, count"}
	}
	quantityDelta, ok := payloadInt64(cmd.Payload, "quantity_delta")
	if !ok || quantityDelta == 0 {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "quantity_delta must not be zero"}
	}
	note := payloadString(cmd.Payload, "note")
	if len(note) > 255 {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "note is too long"}
	}
	if !httptransport.HasPermission(role, "inventory", "adjust") {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "permission_denied", ErrorDetail: "insufficient permissions"}
	}

	var stock int64
	err = tx.QueryRow(ctx, `
		SELECT stock_quantity FROM products WHERE id = $1 AND is_active FOR UPDATE`, productID).Scan(&stock)
	if errors.Is(err, pgx.ErrNoRows) {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "product_not_found", ErrorDetail: "product not found"}
	}
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to load product"}
	}
	if stock+quantityDelta < 0 {
		return pushResult{CommandID: cmd.CommandID, Status: "conflict", ErrorCode: "insufficient_stock", ErrorDetail: "adjustment would make stock negative"}
	}

	adjustmentID := uuid.New()
	var productName, sku string
	err = tx.QueryRow(ctx, `SELECT name, sku FROM products WHERE id = $1`, productID).Scan(&productName, &sku)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to load product"}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO inventory_adjustments (id, tenant_id, product_id, reason, quantity_delta, note, created_by)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)`,
		adjustmentID, tenantID, productID, reason, quantityDelta, note, userID)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to save adjustment"}
	}
	if _, err = tx.Exec(ctx, `UPDATE products SET stock_quantity = stock_quantity + $1 WHERE id = $2`, quantityDelta, productID); err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to update stock"}
	}
	resultJSON := map[string]any{
		"id": adjustmentID.String(), "product_id": productID.String(), "product_name": productName,
		"sku": sku, "reason": reason, "quantity_delta": quantityDelta, "product_stock": stock + quantityDelta,
	}
	return pushResult{CommandID: cmd.CommandID, Status: "applied", Result: resultJSON}
}

func applyCategoryCreate(ctx context.Context, tx pgx.Tx, cmd pushCommand, tenantID uuid.UUID, role string) pushResult {
	name := payloadString(cmd.Payload, "name")
	slug := payloadString(cmd.Payload, "slug")
	if name == "" || slug == "" {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "name and slug are required"}
	}
	if !httptransport.HasPermission(role, "catalog", "write") {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "permission_denied", ErrorDetail: "insufficient permissions"}
	}
	var categoryID string
	err := tx.QueryRow(ctx, `
		INSERT INTO categories (tenant_id, name, slug) VALUES ($1, $2, $3)
		RETURNING id::text`, tenantID, name, slug).Scan(&categoryID)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "conflict", ErrorCode: "category_conflict", ErrorDetail: "category slug is already in use"}
	}
	return pushResult{CommandID: cmd.CommandID, Status: "applied", Result: map[string]any{"id": categoryID, "name": name, "slug": slug}}
}

func applyProductCreate(ctx context.Context, tx pgx.Tx, cmd pushCommand, tenantID uuid.UUID, role string) pushResult {
	name := payloadString(cmd.Payload, "name")
	sku := payloadString(cmd.Payload, "sku")
	currency := strings.ToUpper(payloadString(cmd.Payload, "currency"))
	if currency == "" {
		currency = "EGP"
	}
	priceMinor, ok := payloadInt64(cmd.Payload, "price_minor")
	if !ok || priceMinor < 0 {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "price_minor is invalid"}
	}
	costMinor, _ := payloadInt64(cmd.Payload, "cost_minor")
	if costMinor < 0 {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "cost_minor is invalid"}
	}
	stock, _ := payloadInt64(cmd.Payload, "stock_quantity")
	if stock < 0 || name == "" || sku == "" || len(currency) != 3 {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "invalid product payload"}
	}
	if !httptransport.HasPermission(role, "catalog", "write") {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "permission_denied", ErrorDetail: "insufficient permissions"}
	}

	var maxProducts int
	if err := tx.QueryRow(ctx, `SELECT max_products FROM tenants WHERE id = $1`, tenantID).Scan(&maxProducts); err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to load tenant plan"}
	}
	if maxProducts > 0 {
		var existing int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM products`).Scan(&existing); err != nil {
			return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to count products"}
		}
		if existing >= maxProducts {
			return pushResult{CommandID: cmd.CommandID, Status: "conflict", ErrorCode: "plan_limit_exceeded", ErrorDetail: "tenant product limit reached"}
		}
	}

	categoryID := payloadString(cmd.Payload, "category_id")
	barcode := payloadString(cmd.Payload, "barcode")
	var productID string
	err := tx.QueryRow(ctx, `
		INSERT INTO products (tenant_id, category_id, name, sku, barcode, price_minor, cost_minor, currency, stock_quantity)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, NULLIF($5, ''), $6, $7, $8, $9)
		RETURNING id::text`,
		tenantID, categoryID, name, sku, barcode, priceMinor, costMinor, currency, stock).Scan(&productID)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "conflict", ErrorCode: "product_conflict", ErrorDetail: "product SKU or barcode is already in use"}
	}
	return pushResult{CommandID: cmd.CommandID, Status: "applied", Result: map[string]any{"id": productID, "name": name, "sku": sku, "price_minor": priceMinor}}
}

func applyCreateSale(ctx context.Context, tx pgx.Tx, cmd pushCommand, tenantID, userID, deviceID uuid.UUID, role string) pushResult {
	payload, err := json.Marshal(cmd.Payload)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "invalid payload"}
	}
	var request businesssales.CreateSaleRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "validation_error", ErrorDetail: "invalid sale payload"}
	}

	idempotencyKey, _ := cmd.Payload[idempotencyKeyKey].(string)
	if strings.TrimSpace(idempotencyKey) == "" {
		idempotencyKey = "sync:" + cmd.CommandID
	}

	sale, applyErr := businesssales.CreateSale(ctx, tx, tenantID, userID, deviceID, role, idempotencyKey, defaultDiscountLimitPct, request)
	if applyErr != nil {
		status, code, message := businesssales.SaleError(applyErr)
		return pushResult{CommandID: cmd.CommandID, Status: pushStatusFor(status), Result: nil, ErrorCode: code, ErrorDetail: message}
	}
	marshaled, err := json.Marshal(sale)
	if err != nil {
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "internal_error", ErrorDetail: "unable to encode result"}
	}
	var saleJSON map[string]any
	_ = json.Unmarshal(marshaled, &saleJSON)
	return pushResult{CommandID: cmd.CommandID, Status: "applied", Result: saleJSON}
}

// pushStatusFor maps a sale application failure to the push result status.
// 409 (insufficient_stock, session_not_open, idempotency_conflict) is a
// true conflict; everything else — including 404 (product/session not found)
// and 400 (validation) — is a plain rejection.
func pushStatusFor(httpStatus int) string {
	if httpStatus == http.StatusConflict {
		return "conflict"
	}
	return "rejected"
}

var errCommandConflict = errors.New("command conflict")

// recordCommand registers the command in sync_commands so retries are
// replay-safe. Returns:
//   - inserted=true with the created pending command, or
//   - inserted=false, replaying the stored outcome for an identical command
//     (same command_id), or
//   - errCommandConflict when the same command_id was seen with a different
//     payload (a genuine conflict on command identity).
func recordCommand(ctx context.Context, tx pgx.Tx, commandID, deviceID uuid.UUID, operation, payloadHash string, payload map[string]any, tenantID uuid.UUID) (bool, pushResult, error) {
	// A unique violation aborts the whole tx in Postgres, so the INSERT attempt
	// must run inside a savepoint that we can roll back to and keep going.
	if _, err := tx.Exec(ctx, "SAVEPOINT sync_cmd"); err != nil {
		return false, pushResult{}, err
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO sync_commands (id, tenant_id, device_id, operation, payload_hash, payload, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')`,
		commandID, tenantID, deviceID, operation, payloadHash, payload)
	if err == nil {
		if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT sync_cmd"); err != nil {
			return false, pushResult{}, err
		}
		return true, pushResult{}, nil
	}
	if _, rbErr := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT sync_cmd"); rbErr != nil {
		return false, pushResult{}, rbErr
	}
	if !isUniqueViolation(err) {
		return false, pushResult{}, err
	}

	var storedStatus, storedError, storedHash string
	var storedResult []byte
	err = tx.QueryRow(ctx, `
		SELECT status, COALESCE(result::text, ''), COALESCE(error_code, ''), payload_hash
		FROM sync_commands WHERE id = $1 AND tenant_id = $2`, commandID, tenantID).Scan(&storedStatus, &storedResult, &storedError, &storedHash)
	if errors.Is(err, pgx.ErrNoRows) {
		// Same device+operation+payload hash under a different command_id:
		// dedupe and replay that record.
		err = tx.QueryRow(ctx, `
			SELECT status, COALESCE(result::text, ''), COALESCE(error_code, '')
			FROM sync_commands WHERE tenant_id = $1 AND device_id = $2 AND operation = $3 AND payload_hash = $4
			ORDER BY created_at DESC LIMIT 1`, tenantID, deviceID, operation, payloadHash).Scan(&storedStatus, &storedResult, &storedError)
		if errors.Is(err, pgx.ErrNoRows) {
			return false, pushResult{}, errCommandConflict
		}
		if err != nil {
			return false, pushResult{}, err
		}
		return false, storedOutcome(operation, "", storedStatus, storedResult, storedError), nil
	}
	if err != nil {
		return false, pushResult{}, err
	}
	// Same command_id but a different payload is a genuine conflict on command
	// identity: the caller reused an id for a new intent.
	if storedHash != payloadHash {
		return false, pushResult{}, errCommandConflict
	}
	return false, storedOutcome(operation, commandID.String(), storedStatus, storedResult, storedError), nil
}

func storedOutcome(operation, commandID, status string, result []byte, errorCode string) pushResult {
	out := pushResult{
		Status:    status,
		Replayed:  true,
		ErrorCode: errorCode,
	}
	if commandID != "" {
		out.CommandID = commandID
	}
	if len(result) > 0 {
		var m map[string]any
		if json.Unmarshal(result, &m) == nil {
			out.Result = m
		}
	}
	return out
}

func writeCommandOutcome(ctx context.Context, tx pgx.Tx, commandID uuid.UUID, result pushResult) error {
	var resultJSON []byte
	if result.Result != nil {
		var err error
		resultJSON, err = json.Marshal(result.Result)
		if err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `
		UPDATE sync_commands SET status = $1, result = $2, error_code = $3, processed_at = now()
		WHERE id = $4`, result.Status, resultJSON, result.ErrorCode, commandID)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func hashPayload(payload map[string]any) string {
	data, _ := json.Marshal(payload)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
