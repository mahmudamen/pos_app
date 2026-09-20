// Web Push (RFC 8030) delivery for back-in-stock product requests. When a
// published product is edited through the catalog so that it transitions from
// "not available online" to "available online again" (was out of stock or not
// selforder-enabled, now in stock and enabled), every OPEN product_request
// that asks for that product and opted into browser notifications while on the
// self-order page gets a push. The push fires out-of-band (goroutine) so a
// slow push endpoint never delays the catalog PATCH.
//
// The sending identity is a VAPID key pair configured via VAPID_PUBLIC_KEY /
// VAPID_PRIVATE_KEY / VAPID_SUBJECT. When the keys are absent the service is
// disabled and the self-order page does not show the notify button.
package selforder

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
)

const backInStockTTL = 60 * 60 * 24 // 24h window for an offline browser

// NotifyBackInStock is invoked by the catalog handler after a product PATCH
// commits. It never blocks the caller: the DB read and the web-push POSTs run
// on a background goroutine, and a misconfigured or unreachable push service
// is logged and swallowed.
func (h *Handler) NotifyBackInStock(ctx context.Context, tenantID, productID string) {
	if h.pool == nil || h.vapid.PrivateKey == "" {
		return
	}
	go func() {
		background, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		h.sendBackInStock(background, tenantID, productID)
	}()
}

func (h *Handler) sendBackInStock(ctx context.Context, tenantID, productID string) {
	tenant, err := uuid.Parse(tenantID)
	if err != nil {
		return
	}
	product, err := uuid.Parse(productID)
	if err != nil {
		return
	}
	tx, err := h.tenantTx(ctx, tenant)
	if err != nil {
		slog.Warn("selforder: back-in-stock notify aborted", "err", err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var productName string
	var tenantSlug string
	if err := tx.QueryRow(ctx, `
		SELECT p.name, t.slug FROM products p JOIN tenants t ON t.id = p.tenant_id
		WHERE p.id = $1 AND p.is_active AND p.selforder_enabled AND p.stock_quantity > 0`,
		product).Scan(&productName, &tenantSlug); err != nil {
		// Not available online (yet) — nothing to announce.
		return
	}

	rows, err := tx.Query(ctx, `
		SELECT id, webpush FROM product_requests
		WHERE product_id = $1 AND status = 'open' AND webpush IS NOT NULL`, product)
	if err != nil {
		slog.Warn("selforder: loading back-in-stock subscribers", "err", err)
		return
	}
	type subscriber struct {
		id  uuid.UUID
		sub *webpush.Subscription
		raw []byte
	}
	subs := make([]subscriber, 0)
	for rows.Next() {
		var s subscriber
		var raw []byte
		if err := rows.Scan(&s.id, &raw); err != nil {
			continue
		}
		var wire webPushSub
		if json.Unmarshal(raw, &wire) != nil {
			continue
		}
		s.raw = raw
		s.sub = &webpush.Subscription{
			Endpoint: wire.Endpoint,
			Keys: webpush.Keys{
				P256dh: wire.Keys.P256DH,
				Auth:   wire.Keys.Auth,
			},
		}
		subs = append(subs, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		slog.Warn("selforder: iterating back-in-stock subscribers", "err", err)
		return
	}
	if len(subs) == 0 {
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"title": productName + " is back in stock",
		"body":  "Order it online now from " + tenantSlug + ".",
		"tag":   "back-in-stock:" + product.String(),
		"url":   "/selforder?tenant=" + tenantSlug,
	})

	options := &webpush.Options{
		Subscriber:      h.vapid.Subject,
		VAPIDPublicKey:  h.vapid.PublicKey,
		VAPIDPrivateKey: h.vapid.PrivateKey,
		TTL:             backInStockTTL,
	}
	for _, s := range subs {
		res, err := webpush.SendNotificationWithContext(ctx, payload, s.sub, options)
		if err == nil && res != nil {
			_ = res.Body.Close()
			if res.StatusCode >= 200 && res.StatusCode < 300 {
				_, _ = tx.Exec(ctx, `UPDATE product_requests SET notified_at = NOW() WHERE id = $1`, s.id)
				continue
			}
			if res.StatusCode == 404 || res.StatusCode == 410 {
				// Subscription is dead — drop it so we stop paying for it.
				_, _ = tx.Exec(ctx, `UPDATE product_requests SET webpush = NULL WHERE id = $1`, s.id)
			}
			slog.Debug("selforder: web push rejected", "status", res.StatusCode, "latestError", s.raw)
			continue
		}
		slog.Debug("selforder: web push failed", "err", err)
	}
	if err := tx.Commit(ctx); err != nil {
		slog.Warn("selforder: back-in-stock notify commit", "err", err)
	}
}
