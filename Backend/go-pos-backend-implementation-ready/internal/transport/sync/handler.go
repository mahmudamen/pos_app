package sync

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Entity list synced from the server to POS devices, ordered by change_seq.
// change_type is computed per row: soft-deleted catalog rows decode as
// "delete"; everything else is "upsert".

const pullQuery = `
SELECT * FROM (
	SELECT 'categories' AS entity, id::text AS id, change_seq,
	       CASE WHEN is_active THEN 'upsert' ELSE 'delete' END AS change_type,
	       row_to_json(categories)::text AS data
	FROM categories WHERE change_seq > $1
	UNION ALL
	SELECT 'products', id::text, change_seq,
	       CASE WHEN is_active THEN 'upsert' ELSE 'delete' END,
	       row_to_json(products)::text
	FROM products WHERE change_seq > $1
	UNION ALL
	SELECT 'customers', id::text, change_seq, 'upsert',
	       row_to_json(customers)::text
	FROM customers WHERE change_seq > $1
	UNION ALL
	SELECT 'sales', id::text, change_seq, 'upsert',
	       row_to_json(sales)::text
	FROM sales WHERE change_seq > $1
	UNION ALL
	SELECT 'customer_loyalty_log', id::text, change_seq, 'upsert',
	       row_to_json(customer_loyalty_log)::text
	FROM customer_loyalty_log WHERE change_seq > $1
	UNION ALL
	SELECT 'register_sessions', id::text, change_seq, 'upsert',
	       row_to_json(register_sessions)::text
	FROM register_sessions WHERE change_seq > $1
	UNION ALL
	SELECT 'inventory_adjustments', id::text, change_seq, 'upsert',
	       row_to_json(inventory_adjustments)::text
	FROM inventory_adjustments WHERE change_seq > $1
	UNION ALL
	SELECT 'tenant_settings', key AS id, change_seq, 'upsert',
	       row_to_json(tenant_settings)::text
	FROM tenant_settings WHERE change_seq > $1
	UNION ALL
	SELECT 'product_variants', id::text, change_seq,
	       CASE WHEN is_active THEN 'upsert' ELSE 'delete' END,
	       row_to_json(product_variants)::text
	FROM product_variants WHERE change_seq > $1
	UNION ALL
	SELECT 'product_lots', id::text, change_seq, 'upsert',
	       row_to_json(product_lots)::text
	FROM product_lots WHERE change_seq > $1
	UNION ALL
	SELECT 'floors', id::text, change_seq,
	       CASE WHEN is_active THEN 'upsert' ELSE 'delete' END,
	       row_to_json(floors)::text
	FROM floors WHERE change_seq > $1
	UNION ALL
	SELECT 'restaurant_tables', id::text, change_seq,
	       CASE WHEN is_active THEN 'upsert' ELSE 'delete' END,
	       row_to_json(restaurant_tables)::text
	FROM restaurant_tables WHERE change_seq > $1
) changes
ORDER BY change_seq ASC
LIMIT $2`

const minChangeSeqQuery = `
SELECT MIN(m) FROM (
	SELECT MIN(change_seq) AS m FROM categories
	UNION ALL SELECT MIN(change_seq) FROM products
	UNION ALL SELECT MIN(change_seq) FROM customers
	UNION ALL SELECT MIN(change_seq) FROM sales
	UNION ALL SELECT MIN(change_seq) FROM customer_loyalty_log
	UNION ALL SELECT MIN(change_seq) FROM register_sessions
	UNION ALL SELECT MIN(change_seq) FROM inventory_adjustments
	UNION ALL SELECT MIN(change_seq) FROM tenant_settings
	UNION ALL SELECT MIN(change_seq) FROM product_variants
	UNION ALL SELECT MIN(change_seq) FROM product_lots
	UNION ALL SELECT MIN(change_seq) FROM floors
	UNION ALL SELECT MIN(change_seq) FROM restaurant_tables
) seq`

type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
}

func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager) *Handler {
	return &Handler{pool: pool, tokens: tokens}
}

func (h *Handler) Register(router *gin.RouterGroup) {
	router.GET("/sync/pull", h.pull)
	router.POST("/sync/push", h.push)
}

func (h *Handler) pull(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}

	cursor, err := strconv.ParseInt(c.DefaultQuery("cursor", "0"), 10, 64)
	if err != nil || cursor < 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "cursor is invalid")
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit < 1 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
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

	if cursor > 0 {
		expired, err := isCursorExpired(ctx, tx, cursor)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to check cursor")
			return
		}
		if expired {
			writeError(c, http.StatusConflict, "cursor_expired", "cursor is older than the retention window; full resynchronization required")
			return
		}
	}

	rows, err := tx.Query(ctx, pullQuery, cursor, limit+1)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to pull changes")
		return
	}

	items := make([]syncItem, 0, limit)
	hasMore := false
	for rows.Next() {
		var item syncItem
		var data []byte
		if err := rows.Scan(&item.Entity, &item.ID, &item.ChangeSeq, &item.ChangeType, &data); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to read changes")
			return
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, &item.Data); err != nil {
				rows.Close()
				writeError(c, http.StatusInternalServerError, "internal_error", "unable to decode change")
				return
			}
		}
		if len(items) >= limit {
			hasMore = true
			break
		}
		items = append(items, item)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to read changes")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to commit sync read")
		return
	}

	nextCursor := cursor
	if len(items) > 0 {
		nextCursor = items[len(items)-1].ChangeSeq
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"items":    items,
			"cursor":   nextCursor,
			"has_more": hasMore,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

type syncItem struct {
	Entity     string         `json:"entity"`
	ID         string         `json:"id"`
	ChangeSeq  int64          `json:"change_seq"`
	ChangeType string         `json:"change_type"`
	Data       map[string]any `json:"data"`
}

func isCursorExpired(ctx context.Context, tx pgx.Tx, cursor int64) (bool, error) {
	var minSeq int64
	err := tx.QueryRow(ctx, minChangeSeqQuery).Scan(&minSeq)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, err
	}
	return minSeq > cursor, nil
}

func (h *Handler) authenticate(c *gin.Context) (security.Claims, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is required")
		return security.Claims{}, false
	}
	claims, err := h.tokens.Parse(strings.TrimSpace(strings.TrimPrefix(header, "Bearer ")), security.AccessToken)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return security.Claims{}, false
	}
	return claims, true
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
