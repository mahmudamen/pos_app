package sales

import (
	"errors"
	"fmt"

	"github.com/example/pos-api/internal/transport/access"
)

// Discount errors signalled to the client.
var (
	ErrNegativeDiscount           = errors.New("discount must be non-negative")
	ErrDiscountExceedsSubtotal    = errors.New("discount exceeds subtotal")
	ErrDiscountForbidden          = errors.New("discount not allowed for this user")
	ErrDiscountOverLimit          = errors.New("discount exceeds the allowed limit")
	ErrDiscountManagerPINRequired = errors.New("a valid manager PIN is required for this discount")
	ErrInvalidManagerPIN          = errors.New("invalid manager PIN")
)

// DiscountMode is the tenant-wide enforcement policy for above-limit discounts
// (Odoo ma_pos_base parity).
type DiscountMode string

const (
	// DiscountModeCap quietly clamps the discount to the user's ceiling.
	DiscountModeCap DiscountMode = "cap"
	// DiscountModeBlock rejects the discount unless a manager PIN approves it.
	DiscountModeBlock DiscountMode = "block"
	// DiscountModeWarn allows the discount and warns the cashier.
	DiscountModeWarn DiscountMode = "warn"
)

// ParseDiscountMode decodes a tenant setting value.
func ParseDiscountMode(value string) DiscountMode {
	switch value {
	case "block":
		return DiscountModeBlock
	case "warn":
		return DiscountModeWarn
	default:
		return DiscountModeCap
	}
}

// DiscountDecision is the outcome of evaluating a discount request against the
// user's permissions and the tenant policy.
type DiscountDecision struct {
	// Allow reports whether the sale may proceed without further approval.
	Allow bool
	// NeedsManager reports whether a manager PIN is required to proceed.
	NeedsManager bool
	// TotalAfter is the final total when Allow or NeedsManager; otherwise 0.
	TotalAfter int64
	// AppliedDiscount is the discount actually recorded (clamped in cap mode).
	AppliedDiscount int64
	// Capped reports that cap mode reduced the discount to the user's ceiling.
	Capped bool
	// Warning is the human-readable notice in warn mode.
	Warning string
}

// EvaluateDiscount applies the access-level permissions and the tenant discount
// policy to a discount request. It is pure: callers resolve permissions and
// settings before invoking it. An above-ceiling discount is
//   - capped (mode cap): total clamps to the ceiling, best for a hard policy,
//   - blocked (mode block): NeedsManager, caller decides via manager PIN,
//   - warned (mode warn): allowed, with a notice.
//
// Admin users are always permitted without approval.
func EvaluateDiscount(perms access.Permissions, mode DiscountMode, globalMaxPct int, subtotal, discount int64) (DiscountDecision, error) {
	if discount < 0 {
		return DiscountDecision{}, ErrNegativeDiscount
	}
	if subtotal <= 0 || discount == 0 {
		return DiscountDecision{Allow: true, TotalAfter: subtotal - discount, AppliedDiscount: discount}, nil
	}
	if discount > subtotal {
		return DiscountDecision{}, ErrDiscountExceedsSubtotal
	}
	if !perms.Discount {
		return DiscountDecision{}, ErrDiscountForbidden
	}
	if perms.AccessLevel == access.LevelAdmin {
		return DiscountDecision{Allow: true, TotalAfter: subtotal - discount, AppliedDiscount: discount}, nil
	}
	limit := access.EffectiveDiscountPct(perms, globalMaxPct)
	if limit <= 0 {
		return DiscountDecision{}, ErrDiscountForbidden
	}
	pct := int(discount * 100 / subtotal)
	if pct <= limit {
		return DiscountDecision{Allow: true, TotalAfter: subtotal - discount, AppliedDiscount: discount}, nil
	}

	switch mode {
	case DiscountModeBlock:
		return DiscountDecision{NeedsManager: true, TotalAfter: subtotal - discount, AppliedDiscount: discount}, nil
	case DiscountModeWarn:
		return DiscountDecision{Allow: true, TotalAfter: subtotal - discount, AppliedDiscount: discount,
			Warning: fmt.Sprintf("discount of %d%% exceeds the allowed %d%% limit", pct, limit)}, nil
	default: // cap
		allowedTotal := subtotal * (100 - int64(limit)) / 100
		appliedDiscount := subtotal - allowedTotal
		return DiscountDecision{Allow: true, TotalAfter: allowedTotal, AppliedDiscount: appliedDiscount, Capped: true}, nil
	}
}
