// Package access implements the ma_pos_base-inspired per-user POS security
// model: five access levels (none < cashier < advanced < manager < admin)
// with per-operation permission flags and a per-user discount cap. Effective
// permissions come from the level defaults unless a user opts into custom
// overrides (use_custom_permissions). It is shared by the users, auth and
// sales modules.
package access

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Level names (mirror the users_pos_security.access_level CHECK constraint).
const (
	LevelNone     = "none"
	LevelCashier  = "cashier"
	LevelAdvanced = "advanced"
	LevelManager  = "manager"
	LevelAdmin    = "admin"
)

// Permissions is the resolved per-user operation set for the POS.
type Permissions struct {
	AccessLevel         string `json:"access_level"`
	MaxDiscountPct      int    `json:"max_discount_pct"`
	DeleteOrder         bool   `json:"can_delete_order"`
	DeleteLine          bool   `json:"can_delete_line"`
	ChangeQty           bool   `json:"can_change_qty"`
	NegativeQty         bool   `json:"can_negative_qty"`
	PriceChange         bool   `json:"can_price_change"`
	Discount            bool   `json:"can_discount"`
	OpenSession         bool   `json:"can_open_session"`
	CloseSession        bool   `json:"can_close_session"`
	PaymentModification bool   `json:"can_payment_modification"`
	Refund              bool   `json:"can_refund"`
	NegativeStock       bool   `json:"can_negative_stock"`
}

// levelDefaults returns the built-in permission profile for an access level.
func levelDefaults(level string) Permissions {
	switch level {
	case LevelNone:
		return Permissions{AccessLevel: level}
	case LevelCashier:
		return Permissions{
			AccessLevel: level, MaxDiscountPct: 5,
			ChangeQty: true, Discount: true, OpenSession: true, CloseSession: true,
		}
	case LevelAdvanced:
		return Permissions{
			AccessLevel: level, MaxDiscountPct: 10,
			DeleteLine: true, ChangeQty: true, Discount: true,
			OpenSession: true, CloseSession: true, PaymentModification: true,
		}
	case LevelManager:
		return Permissions{
			AccessLevel: level, MaxDiscountPct: 50,
			DeleteOrder: true, DeleteLine: true, ChangeQty: true, NegativeQty: true,
			PriceChange: true, Discount: true, OpenSession: true, CloseSession: true,
			PaymentModification: true, Refund: true,
		}
	case LevelAdmin:
		return Permissions{
			AccessLevel: level, MaxDiscountPct: 100,
			DeleteOrder: true, DeleteLine: true, ChangeQty: true, NegativeQty: true,
			PriceChange: true, Discount: true, OpenSession: true, CloseSession: true,
			PaymentModification: true, Refund: true, NegativeStock: true,
		}
	default:
		return Permissions{AccessLevel: LevelNone}
	}
}

// DefaultLevelForRole maps the legacy tenant role onto a POS access level.
// saas_admin is the platform role and gets the admin profile cross-tenant.
func DefaultLevelForRole(role string) string {
	switch role {
	case "owner", "saas_admin":
		return LevelAdmin
	case "manager":
		return LevelManager
	case "cashier":
		return LevelCashier
	default:
		return LevelNone
	}
}

// ValidLevel reports whether level is one of the five access levels.
func ValidLevel(level string) bool {
	switch level {
	case LevelNone, LevelCashier, LevelAdvanced, LevelManager, LevelAdmin:
		return true
	default:
		return false
	}
}

// DiscountDisabledForLevel reports whether an access level may not discount at
// all (None tier has no discount and no UI buttons; Basic Cashier has none in
// the Odoo module, but our legacy cashier default allows 5%).
func DiscountDisabledForLevel(level string) bool {
	return level == LevelNone
}

// Row is the stored users_pos_security row subset the resolver needs.
type Row struct {
	AccessLevel    string
	MaxDiscountPct *int // NULL -> level default
	UseCustom      bool
	DeleteOrder    bool
	DeleteLine     bool
	ChangeQty      bool
	NegativeQty    bool
	PriceChange    bool
	Discount       bool
	OpenSession    bool
	CloseSession   bool
	PaymentModific bool
	Refund         bool
	NegativeStock  bool
}

// Resolve computes effective permissions for a user. With an empty row (the
// user has no users_pos_security entry) it falls back to the role's level.
func Resolve(role string, row *Row) Permissions {
	level := DefaultLevelForRole(role)
	if row != nil && row.AccessLevel != "" {
		if ValidLevel(row.AccessLevel) {
			level = row.AccessLevel
		}
	}
	perms := levelDefaults(level)
	if row == nil {
		return perms
	}
	if !row.UseCustom {
		// Only the per-user discount cap overrides the level even without
		// custom flags; the operation flags stay at the level defaults.
		if row.MaxDiscountPct != nil {
			perms.MaxDiscountPct = *row.MaxDiscountPct
		}
		return perms
	}
	perms.MaxDiscountPct = levelDefaults(level).MaxDiscountPct
	if row.MaxDiscountPct != nil {
		perms.MaxDiscountPct = *row.MaxDiscountPct
	}
	perms.DeleteOrder = row.DeleteOrder
	perms.DeleteLine = row.DeleteLine
	perms.ChangeQty = row.ChangeQty
	perms.NegativeQty = row.NegativeQty
	perms.PriceChange = row.PriceChange
	perms.Discount = row.Discount
	perms.OpenSession = row.OpenSession
	perms.CloseSession = row.CloseSession
	perms.PaymentModification = row.PaymentModific
	perms.Refund = row.Refund
	perms.NegativeStock = row.NegativeStock
	return perms
}

// EffectiveDiscountPct applies the tenant-wide cap on top of the per-user
// limit (Odoo: user limit can't exceed the config limit). globalMaxPct <= 0
// means "no global limit"; admin is always uncapped.
func EffectiveDiscountPct(perms Permissions, globalMaxPct int) int {
	if perms.AccessLevel == LevelAdmin {
		return 100
	}
	if globalMaxPct <= 0 {
		return perms.MaxDiscountPct
	}
	if perms.MaxDiscountPct <= 0 {
		return 0
	}
	if perms.MaxDiscountPct < globalMaxPct {
		return perms.MaxDiscountPct
	}
	return globalMaxPct
}

// ScanRow reads a users_pos_security row from the API-facing shape build above.
// It keeps the resolver decoupled from the raw SQL columns.
func ScanRow(level string, max *int, custom, do, dl, cq, nq, pc, d, os, cs, pm, r, ns bool) *Row {
	return &Row{
		AccessLevel: level, MaxDiscountPct: max, UseCustom: custom,
		DeleteOrder: do, DeleteLine: dl, ChangeQty: cq, NegativeQty: nq,
		PriceChange: pc, Discount: d, OpenSession: os, CloseSession: cs,
		PaymentModific: pm, Refund: r, NegativeStock: ns,
	}
}

// ParseLevel decodes a request body access_level with a friendly error.
func ParseLevel(value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", errors.New("access_level is required")
	}
	if !ValidLevel(v) {
		return "", fmt.Errorf("access_level must be one of none, cashier, advanced, manager, admin")
	}
	return v, nil
}

// ResolveFromDB loads a user's users_pos_security row and computes the
// effective POS access profile within the caller's transaction (tenant context
// assumed active). Missing rows fall back to the legacy role level, so
// pre-027 users keep their existing behaviour. Shared by the users, auth and
// sales modules.
func ResolveFromDB(ctx context.Context, tx pgx.Tx, tenantID, userID, role string) Permissions {
	var level *string
	var maxDiscount *int
	var custom, do, dl, cq, nq, pc, d, os, cs, pm, r, ns bool
	err := tx.QueryRow(ctx, `
		SELECT s.access_level, s.max_discount_pct, s.use_custom_permissions,
		       s.can_delete_order, s.can_delete_line, s.can_change_qty, s.can_negative_qty,
		       s.can_price_change, s.can_discount, s.can_open_session, s.can_close_session,
		       s.can_payment_modification, s.can_refund, s.can_negative_stock
		FROM users_pos_security s
		WHERE s.tenant_id = $1::uuid AND s.user_id = $2::uuid`,
		tenantID, userID).Scan(&level, &maxDiscount, &custom,
		&do, &dl, &cq, &nq, &pc, &d, &os, &cs, &pm, &r, &ns)
	if err != nil {
		return Resolve(role, nil)
	}
	var row *Row
	if level != nil {
		row = ScanRow(*level, maxDiscount, custom, do, dl, cq, nq, pc, d, os, cs, pm, r, ns)
	}
	return Resolve(role, row)
}
