package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	defer tx.Rollback(ctx)
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
	default:
		return pushResult{CommandID: cmd.CommandID, Status: "rejected", ErrorCode: "unknown_command", ErrorDetail: "unknown operation: " + cmd.Operation}
	}
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
	if operation == "sale.create" && commandID != "" {
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
