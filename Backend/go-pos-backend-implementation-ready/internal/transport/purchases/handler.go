// Package purchases turns scanned Arabic supplier invoices into inventory
// movement. The OCR endpoint (/v1/purchases/ocr) parses an uploaded invoice
// image into draft line items matched against the tenant catalog; the apply
// endpoint (/v1/purchases) commits those lines as a purchase, bumping stock,
// purchase price (cost_minor) and unit — the "update inventory and price
// purchase and unit used" pipeline. Money is integer minor units and stock is
// integer base units, matching shops/products/sales. Everything is tenant-RLS
// scoped and gated behind the inventory.adjust permission.
package purchases

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/pos-api/internal/infrastructure/security"
	"github.com/example/pos-api/internal/ocr"
	httptransport "github.com/example/pos-api/internal/transport/http"
	notificationstransport "github.com/example/pos-api/internal/transport/notifications"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxOCRImageBytes = 10 << 20 // enforce a sane upper bound on top of MaxBodySize
	minMatchScore    = 40       // below this an OCR name is NOT auto-attached to a product
	maxItems         = 200
)

// Handler is the purchase + OCR HTTP surface.
type Handler struct {
	pool   *pgxpool.Pool
	tokens security.TokenManager
	ocr    ocr.Engine
	notify notificationstransport.Emit

	// OCR metering (migration 036_ocr_usage): per-tenant rolling-window
	// limits fed from config (OCR_DAY_LIMIT / OCR_WEEK_LIMIT / OCR_MONTH_LIMIT)
	// plus the tenant's scan-borrowing credit balance. meterWindows is the
	// configured cap (0 = unlimited); loadOCRUsage populates used counters.
	meterWindows OCRWindows
	meterLimits  OCRWindows
}

// SetNotifier wires the notifications emitter (nil = no-op).
func (h *Handler) SetNotifier(fn notificationstransport.Emit) { h.notify = fn }

// NewHandler wires the module. The OCR engine is built from config; a disabled
// engine makes /v1/purchases/ocr answer 503 cleanly while apply + list still
// work (manual entry).
func NewHandler(pool *pgxpool.Pool, tokens security.TokenManager, engine ocr.Engine, limits ...OCRWindows) *Handler {
	w := OCRWindows{}
	if len(limits) > 0 {
		w = limits[0]
	}
	return &Handler{pool: pool, tokens: tokens, ocr: engine, meterWindows: w, meterLimits: w}
}

// Register attaches the module routes to the /v1 group.
func (h *Handler) Register(router *gin.RouterGroup) {
	router.POST("/purchases/ocr", h.ocrScan)
	router.POST("/purchases/ocr/topup", h.ocrTopup)
	router.GET("/purchases/ocr/usage", h.ocrUsage)
	router.POST("/purchases", h.create)
	router.GET("/purchases", h.list)
}

// ScanLine is one OCR draft row sent back to the client for review/apply.
type ScanLine struct {
	Name           string  `json:"name"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	UnitPriceMinor int64   `json:"unit_price_minor"`
	TotalMinor     int64   `json:"total_minor"`
	Score          int     `json:"score"`
	ProductID      string  `json:"product_id,omitempty"`
	ProductName    string  `json:"product_name,omitempty"`
	MatchScore     int     `json:"match_score"`
}

type purchaseLineRequest struct {
	ProductID      *string `json:"product_id"`
	ProductName    string  `json:"product_name"`
	Quantity       float64 `json:"quantity"`
	Unit           string  `json:"unit"`
	UnitPriceMinor int64   `json:"unit_price_minor"`
	MatchScore     int     `json:"match_score"`
	RowText        string  `json:"row_text"`
}

type createPurchaseRequest struct {
	Supplier  string                `json:"supplier"`
	InvoiceNo string                `json:"invoice_no"`
	Currency  string                `json:"currency"`
	TaxMinor  int64                 `json:"tax_minor"`
	OCRText   string                `json:"ocr_text"`
	Items     []purchaseLineRequest `json:"items"`
}

// AppliedLine reports per-item inventory effect after a purchase is committed.
type AppliedLine struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	Unit         string `json:"unit"`
	StockDelta   int64  `json:"stock_delta"`
	NewStock     int64  `json:"new_stock"`
	NewCostMinor int64  `json:"new_cost_minor"`
	Created      bool   `json:"created"`
}

// ocrScan parses an uploaded invoice image into candidate lines. It returns
// suggestions only — no stock or price is touched until the client POSTs the
// same lines to /v1/purchases.
func (h *Handler) ocrScan(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "inventory", "adjust") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	if !h.ocr.Available() {
		writeError(c, http.StatusServiceUnavailable, "ocr_unavailable",
			"OCR is not configured; enable OCR_ENABLED and install tesseract with Arabic data")
		return
	}

	// OCR metering gate (migration 036_ocr_usage). Before spending any engine
	// time on this image, load the tenant's rolling-window counters from
	// ocr_usage and its scan-borrowing credit balance from tenants, and ask
	// the pure kernel whether this scan may be admitted. A tenant that is past
	// ANY active window limit — and whose credit balance is exhausted — is
	// answered 402 ocr_window_limit_reached; the shop tops up points and scans
	// resume.
	ctx := c.Request.Context()
	used, credits, err := h.loadOCRUsage(ctx, claims.TenantID)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable",
			"unable to load OCR usage")
		return
	}
	admit, scanBorrowedCredit := ShouldAdmit(used, h.meterLimits, credits)
	if !admit {
		if h.notify != nil {
			h.notify(ctx, claims.TenantID, "", notificationstransport.TypeOCRLimit,
				"ocr_limit", notificationstransport.SeverityCritical,
				"OCR scan limit reached",
				"The tenant hit its daily/weekly/monthly OCR scan window and the credit balance is exhausted.",
				map[string]any{"used": used, "credits_remaining": credits})
		}
		writeError(c, http.StatusPaymentRequired, "ocr_window_limit_reached",
			"OCR scan limit reached on the day, week and month windows and the credit balance is exhausted; top up OCR points to resume")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "an image file is required (form field 'file')")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "unable to read uploaded file")
		return
	}
	defer func() { _ = file.Close() }()
	image, err := io.ReadAll(io.LimitReader(file, maxOCRImageBytes+1))
	if err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "unable to read uploaded file")
		return
	}
	if len(image) == 0 || len(image) > maxOCRImageBytes {
		writeError(c, http.StatusBadRequest, "validation_error", "uploaded file is empty or exceeds 10MB")
		return
	}

	text, err := h.ocr.OCR(c.Request.Context(), image)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			writeError(c, http.StatusGatewayTimeout, "ocr_timeout", "OCR took too long; try a smaller image")
			return
		}
		writeError(c, http.StatusBadGateway, "ocr_failed", "OCR engine returned an error")
		return
	}
	res := ocr.ParseInvoice(text)
	if len(res.Items) == 0 {
		writeError(c, http.StatusUnprocessableEntity, "no_items_detected",
			"no recognizable purchase lines in the image: "+strings.Join(res.Discarded, " | "))
		return
	}

	// Match each scan line against the tenant catalog so the client can accept
	// suggestions with one tap per line.
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	type catalogProduct struct{ id, name string }
	catalog := make([]catalogProduct, 0)
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
	rows, err := tx.Query(ctx, `SELECT id::text, name FROM products WHERE tenant_id = $1 AND is_active`, tenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load catalog")
		return
	}
	for rows.Next() {
		var p catalogProduct
		if err := rows.Scan(&p.id, &p.name); err != nil {
			rows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load catalog")
			return
		}
		catalog = append(catalog, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load catalog")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load catalog")
		return
	}

	lines := make([]ScanLine, 0, len(res.Items))
	for _, item := range res.Items {
		scan := ScanLine{
			Name: item.Name, Quantity: item.Quantity, Unit: item.Unit,
			UnitPriceMinor: item.UnitPriceMinor, TotalMinor: item.TotalMinor, Score: item.Score,
		}
		if item.Name != "" {
			key := ocr.NormalizeKey(item.Name)
			bestID, bestName, bestScore := "", "", 0
			for _, p := range catalog {
				if s := ocr.MatchScore(key, ocr.NormalizeKey(p.name)); s > bestScore {
					bestID, bestName, bestScore = p.id, p.name, s
				}
			}
			if bestScore >= minMatchScore {
				scan.ProductID, scan.ProductName, scan.MatchScore = bestID, bestName, bestScore
			}
		}
		lines = append(lines, scan)
	}

	if err = h.bumpOCRUsage(ctx, claims.TenantID, scanBorrowedCredit); err != nil {
		// Metering is a ledger best-effort: a scan already succeeded, so the
		// counter write must never fail the response.
		_ = err
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"raw_text":  text,
			"lines":     lines,
			"discarded": res.Discarded,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// create commits a purchase. It is the authoritative inventory write: for each
// line it finds (or creates) the product, then bumps stock_quantity and sets
// cost_minor to the invoice unit price. The OCR pipeline is purely advisory —
// a client may also POST hand-typed lines, which is what makes manual invoice
// entry work without OCR enabled.
func (h *Handler) create(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	var req createPurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "validation_error", "invalid purchase payload")
		return
	}
	if !httptransport.HasPermission(claims.Role, "inventory", "adjust") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
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
	if len(req.Items) == 0 || len(req.Items) > maxItems {
		writeError(c, http.StatusBadRequest, "validation_error", "items must contain 1..200 lines")
		return
	}
	if len(req.Supplier) > 120 || len(req.InvoiceNo) > 64 {
		writeError(c, http.StatusBadRequest, "validation_error", "supplier or invoice_no is too long")
		return
	}
	if len(req.OCRText) > 200_000 {
		writeError(c, http.StatusBadRequest, "validation_error", "ocr_text is too large")
		return
	}
	if req.Currency == "" {
		req.Currency = "EGP"
	}
	if len(req.Currency) != 3 {
		writeError(c, http.StatusBadRequest, "validation_error", "currency must be a 3-letter code")
		return
	}
	if req.TaxMinor < 0 || req.TaxMinor > 1_000_000_000_000 {
		writeError(c, http.StatusBadRequest, "validation_error", "tax_minor is out of range")
		return
	}
	for i := range req.Items {
		line := &req.Items[i]
		if line.ProductID == nil && strings.TrimSpace(line.ProductName) == "" {
			writeError(c, http.StatusBadRequest, "validation_error", fmt.Sprintf("line %d: product_name is required", i+1))
			return
		}
		if line.Quantity <= 0 || line.Quantity > 1_000_000_000 {
			writeError(c, http.StatusBadRequest, "validation_error", fmt.Sprintf("line %d: quantity must be positive", i+1))
			return
		}
		if line.UnitPriceMinor < 0 || line.UnitPriceMinor > 1_000_000_000_000 {
			writeError(c, http.StatusBadRequest, "validation_error", fmt.Sprintf("line %d: unit_price_minor is out of range", i+1))
			return
		}
		if line.Unit == "" {
			line.Unit = "piece"
		} else if resolved := ocr.ResolveUnit(line.Unit); resolved != "" {
			line.Unit = resolved
		}
		if line.ProductID != nil {
			if _, err := uuid.Parse(*line.ProductID); err != nil {
				writeError(c, http.StatusBadRequest, "validation_error", fmt.Sprintf("line %d: product_id is invalid", i+1))
				return
			}
		}
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

	// Reject a re-posted invoice: same tenant + non-empty invoice_no.
	if strings.TrimSpace(req.InvoiceNo) != "" {
		var dup string
		err := tx.QueryRow(ctx, `SELECT id::text FROM purchases WHERE tenant_id = $1 AND invoice_no = $2`,
			tenantID, strings.TrimSpace(req.InvoiceNo)).Scan(&dup)
		if err == nil {
			writeError(c, http.StatusConflict, "duplicate_invoice", "an invoice with this number already exists")
			return
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to check invoice history")
			return
		}
	}

	// Load the live catalog once so name-matching does not repeat queries.
	type catalogRow struct {
		id, name, unit string
		cost, stock    int64
	}
	catalog := make([]catalogRow, 0)
	catalogRows, err := tx.Query(ctx, `
		SELECT id::text, name, unit, cost_minor, stock_quantity
		FROM products WHERE tenant_id = $1 AND is_active`, tenantID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load catalog")
		return
	}
	for catalogRows.Next() {
		var p catalogRow
		if err := catalogRows.Scan(&p.id, &p.name, &p.unit, &p.cost, &p.stock); err != nil {
			catalogRows.Close()
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load catalog")
			return
		}
		catalog = append(catalog, p)
	}
	catalogRows.Close()
	if err := catalogRows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load catalog")
		return
	}

	type applied struct {
		AppliedLine
		ItemTotal int64
	}
	appliedLines := make([]applied, 0, len(req.Items))

	for i := range req.Items {
		line := &req.Items[i]
		var (
			productID, productName, productUnit string
			costMinor, stock                    int64
			found, created                      bool
		)
		switch {
		case line.ProductID != nil:
			err := tx.QueryRow(ctx, `
				SELECT id::text, name, unit, cost_minor, stock_quantity
				FROM products WHERE tenant_id = $1 AND id = $2::uuid AND is_active FOR UPDATE`,
				tenantID, *line.ProductID).
				Scan(&productID, &productName, &productUnit, &costMinor, &stock)
			if errors.Is(err, pgx.ErrNoRows) {
				writeError(c, http.StatusNotFound, "product_not_found",
					fmt.Sprintf("line %d: product does not exist or is inactive", i+1))
				return
			}
			if err != nil {
				writeError(c, http.StatusInternalServerError, "internal_error", "unable to load product")
				return
			}
			found = true
		default:
			key := ocr.NormalizeKey(line.ProductName)
			best, bestScore := -1, 0
			for idx := range catalog {
				if s := ocr.MatchScore(key, ocr.NormalizeKey(catalog[idx].name)); s > bestScore {
					best, bestScore = idx, s
				}
			}
			if best >= 0 && bestScore >= minMatchScore {
				p := catalog[best]
				err := tx.QueryRow(ctx, `
					SELECT id::text, name, unit, cost_minor, stock_quantity
					FROM products WHERE tenant_id = $1 AND id = $2::uuid AND is_active FOR UPDATE`,
					tenantID, p.id).
					Scan(&productID, &productName, &productUnit, &costMinor, &stock)
				if err == nil {
					found = true
				}
			}
			if !found {
				sku := "INV-" + strings.ToUpper(uuid.NewString()[:8])
				newID := uuid.New()
				err := tx.QueryRow(ctx, `
					INSERT INTO products (id, tenant_id, name, sku, currency, price_minor, cost_minor, stock_quantity, unit)
					VALUES ($1, $2, $3, $4, $5, $6, $6, 0, $7)
					RETURNING name`,
					newID, tenantID, strings.TrimSpace(line.ProductName), sku,
					req.Currency, line.UnitPriceMinor, line.Unit).Scan(&productName)
				if err != nil {
					writeError(c, http.StatusInternalServerError, "internal_error", "unable to create product")
					return
				}
				productID, productUnit = newID.String(), line.Unit
				created = true
			}
		}

		// Quantity in stock terms: convert the invoice unit to the product unit
		// when a conversion exists; otherwise adopt the invoice unit on the
		// product (the "unit used" update the shop wants).
		effectiveUnit := productUnit
		if productUnit == "" {
			effectiveUnit = "piece"
		}
		stockDelta := quantityInBaseUnits(ctx, tx, line.Unit, effectiveUnit, line.Quantity)
		if stockDelta < 1 {
			stockDelta = 1
		}
		newCost := line.UnitPriceMinor
		var newStock int64
		if err := tx.QueryRow(ctx, `
			UPDATE products
			SET stock_quantity = stock_quantity + $1,
			    cost_minor     = $2,
			    unit           = $3,
			    updated_at     = now()
			WHERE tenant_id = $4 AND id = $5::uuid
			RETURNING stock_quantity`,
			stockDelta, newCost, effectiveUnit, tenantID, productID).Scan(&newStock); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to update stock")
			return
		}

		itemTotal := int64(0)
		if line.Quantity > 0 {
			itemTotal = int64(math.Round(line.Quantity * float64(line.UnitPriceMinor)))
		}
		appliedLines = append(appliedLines, applied{
			AppliedLine: AppliedLine{
				ProductID: productID, ProductName: productName, Unit: effectiveUnit,
				StockDelta: stockDelta, NewStock: newStock, NewCostMinor: newCost, Created: created,
			},
			ItemTotal: itemTotal,
		})
	}

	purchaseID := uuid.New()
	var subtotal int64
	for _, a := range appliedLines {
		subtotal += a.ItemTotal
	}
	total := subtotal + req.TaxMinor

	var createdAt string
	if err := tx.QueryRow(ctx, `
		INSERT INTO purchases (id, tenant_id, supplier, invoice_no, currency,
		                       subtotal_minor, tax_minor, total_minor, ocr_text, created_by)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), $5, $6, $7, $8, $9, $10)
		RETURNING created_at::text`,
		purchaseID, tenantID, strings.TrimSpace(req.Supplier),
		strings.TrimSpace(req.InvoiceNo), req.Currency, subtotal, req.TaxMinor,
		total, strings.TrimSpace(req.OCRText), userID).Scan(&createdAt); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(c, http.StatusConflict, "duplicate_invoice", "an invoice with this number already exists")
			return
		}
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save purchase")
		return
	}

	lines := make([]AppliedLine, 0, len(appliedLines))
	for i, a := range appliedLines {
		itemID := uuid.New()
		line := req.Items[i]
		if _, err := tx.Exec(ctx, `
			INSERT INTO purchase_items (id, tenant_id, purchase_id, product_id, product_name,
			                            quantity, unit, unit_price_minor, total_minor, match_score, row_text)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			itemID, tenantID, purchaseID, uuid.MustParse(a.ProductID), a.ProductName,
			fmt.Sprintf("%.3f", line.Quantity), a.Unit, line.UnitPriceMinor, a.ItemTotal,
			line.MatchScore, strings.TrimSpace(line.RowText)); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to save purchase line")
			return
		}
		lines = append(lines, a.AppliedLine)
	}

	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to save purchase")
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": gin.H{
			"purchase_id":    purchaseID.String(),
			"supplier":       strings.TrimSpace(req.Supplier),
			"invoice_no":     strings.TrimSpace(req.InvoiceNo),
			"currency":       req.Currency,
			"subtotal_minor": subtotal,
			"tax_minor":      req.TaxMinor,
			"total_minor":    total,
			"created_at":     createdAt,
			"lines":          lines,
		},
		"meta": gin.H{"request_id": c.GetString("request_id")},
	})
}

// quantityInBaseUnits converts an invoice quantity into the product's unit via
// the unit_conversions table when a (from,to) factor exists; otherwise it
// returns the raw quantity (the invoice unit is adopted by the caller).
func quantityInBaseUnits(ctx context.Context, tx pgx.Tx, fromUnit, toUnit string, qty float64) int64 {
	if fromUnit == "" {
		fromUnit = "piece"
	}
	if fromUnit == toUnit {
		return int64(math.Round(qty))
	}
	var factor float64
	err := tx.QueryRow(ctx, `
		SELECT factor FROM unit_conversions WHERE from_unit = $1 AND to_unit = $2`,
		fromUnit, toUnit).Scan(&factor)
	if err != nil {
		// No conversion known: the invoice unit is adopted on the product, so
		// the quantity moves over as-is in that unit.
		return int64(math.Round(qty))
	}
	return int64(math.Round(qty * factor))
}

// PurchaseSummary is the lean list view.
type PurchaseSummary struct {
	ID            string `json:"id"`
	Supplier      string `json:"supplier"`
	InvoiceNo     string `json:"invoice_no"`
	Currency      string `json:"currency"`
	ItemCount     int    `json:"item_count"`
	SubtotalMinor int64  `json:"subtotal_minor"`
	TaxMinor      int64  `json:"tax_minor"`
	TotalMinor    int64  `json:"total_minor"`
	CreatedBy     string `json:"created_by"`
	CreatedAt     string `json:"created_at"`
}

func (h *Handler) list(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "inventory", "adjust") {
		writeError(c, http.StatusForbidden, "permission_denied", "insufficient permissions")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		writeError(c, http.StatusUnauthorized, "unauthorized", "authorization is invalid")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	offset := int64((page - 1) * limit)

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
	var total int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM purchases`).Scan(&total); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to count purchases")
		return
	}
	rows, err := tx.Query(ctx, `
		SELECT p.id::text, p.supplier, p.invoice_no, p.currency,
		       (SELECT COUNT(*) FROM purchase_items pi
		         WHERE pi.tenant_id = p.tenant_id AND pi.purchase_id = p.id),
		       p.subtotal_minor, p.tax_minor, p.total_minor,
		       COALESCE(u.display_name, ''), p.created_at::text
		FROM purchases p
		LEFT JOIN users u ON u.tenant_id = p.tenant_id AND u.id = p.created_by
		ORDER BY p.created_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load purchases")
		return
	}
	defer rows.Close()
	items := make([]PurchaseSummary, 0)
	for rows.Next() {
		var s PurchaseSummary
		if err := rows.Scan(&s.ID, &s.Supplier, &s.InvoiceNo, &s.Currency, &s.ItemCount,
			&s.SubtotalMinor, &s.TaxMinor, &s.TotalMinor, &s.CreatedBy, &s.CreatedAt); err != nil {
			writeError(c, http.StatusInternalServerError, "internal_error", "unable to load purchases")
			return
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load purchases")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to load purchases")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": items,
		"meta": gin.H{
			"request_id": c.GetString("request_id"),
			"page":       page,
			"limit":      limit,
			"total":      total,
		},
	})
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

// loadOCRUsage reads the tenant's rolling-window counters from ocr_usage and
// its scan-borrowing credit balance (tenants.ocr_credits_remaining). It runs
// in a tenant-scoped transaction so RLS FORCE sees the right tenant on the
// same connection as the reads.
func (h *Handler) loadOCRUsage(ctx context.Context, tenantID string) (OCRWindows, int64, error) {
	if h.pool == nil {
		return OCRWindows{}, 0, nil
	}
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return OCRWindows{}, 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		return OCRWindows{}, 0, err
	}
	q := `SELECT
		COALESCE((SELECT scans_used FROM ocr_usage WHERE tenant_id = $1 AND window_kind = 'day' AND window_start = CURRENT_DATE), 0),
		COALESCE((SELECT scans_used FROM ocr_usage WHERE tenant_id = $1 AND window_kind = 'week' AND window_start = date_trunc('week', CURRENT_DATE)::date), 0),
		COALESCE((SELECT scans_used FROM ocr_usage WHERE tenant_id = $1 AND window_kind = 'month' AND window_start = date_trunc('month', CURRENT_DATE)::date), 0),
		COALESCE((SELECT ocr_credits_remaining FROM tenants WHERE id = $1), 0)`
	var used OCRWindows
	var credits int64
	if err := tx.QueryRow(ctx, q, tenantID).Scan(&used.Day, &used.Week, &used.Month, &credits); err != nil {
		return OCRWindows{}, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return OCRWindows{}, 0, err
	}
	return used, credits, nil
}

// bumpOCRUsage records one admitted scan against the tenant's day/week/month
// windows and, when the scan was admitted by borrowing a credit, decrements
// the credit balance. Best-effort; the caller ignores errors on purpose.
func (h *Handler) bumpOCRUsage(ctx context.Context, tenantID string, borrowed bool) error {
	if h.pool == nil {
		return nil
	}
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", tenantID); err != nil {
		return err
	}
	for _, w := range []struct {
		kind  string
		start string
	}{
		{"day", "CURRENT_DATE"},
		{"week", "date_trunc('week', CURRENT_DATE)::date"},
		{"month", "date_trunc('month', CURRENT_DATE)::date"},
	} {
		upsert := `INSERT INTO ocr_usage (tenant_id, window_kind, window_start, scans_used)
			VALUES ($1, $2, ` + w.start + `, 1)
			ON CONFLICT (tenant_id, window_kind)
			DO UPDATE SET scans_used = ocr_usage.scans_used + 1, updated_at = now()`
		if _, err := tx.Exec(ctx, upsert, tenantID, w.kind); err != nil {
			return err
		}
	}
	if borrowed {
		if _, err := tx.Exec(ctx,
			`UPDATE tenants SET ocr_credits_remaining = GREATEST(ocr_credits_remaining - 1, 0) WHERE id = $1`,
			tenantID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// ocrTopup adds OCR scan credits to the tenant (POST /v1/purchases/ocr/topup).
// The shop tops up points when the window limit is reached; the balance is
// then burnable one scan per point past the window caps.
func (h *Handler) ocrTopup(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	if !httptransport.HasPermission(claims.Role, "inventory", "adjust") {
		writeError(c, http.StatusForbidden, "permission_denied", "manager or owner required")
		return
	}
	if h.pool == nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var req struct {
		Points int64 `json:"points"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Points <= 0 {
		writeError(c, http.StatusBadRequest, "validation_error", "points must be a positive integer")
		return
	}
	ctx := c.Request.Context()
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant', $1, true)", claims.TenantID); err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	var balance int64
	if err := tx.QueryRow(ctx,
		`UPDATE tenants SET ocr_credits_remaining = ocr_credits_remaining + $2 WHERE id = $1 RETURNING ocr_credits_remaining`,
		claims.TenantID, req.Points).Scan(&balance); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to top up OCR credits")
		return
	}
	if err := tx.Commit(ctx); err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "unable to top up OCR credits")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"credits_remaining": balance}})
}

// ocrUsage reports the tenant's current meter state (used per window, configured
// limits, remaining credits) for the client to render limits + top-up UI.
func (h *Handler) ocrUsage(c *gin.Context) {
	claims, ok := h.authenticate(c)
	if !ok {
		return
	}
	used, credits, err := h.loadOCRUsage(c.Request.Context(), claims.TenantID)
	if err != nil {
		writeError(c, http.StatusServiceUnavailable, "database_unavailable", "database unavailable")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": MeterState{Used: used, Limits: h.meterLimits, Credits: credits},
	})
}

func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "request_id": c.GetString("request_id")}})
}
