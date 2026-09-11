package sales

import "fmt"

// DiscountError types for discount validation.
type DiscountError string

func (d DiscountError) Error() string { return string(d) }

const (
	ErrNegativeDiscount        DiscountError = "discount_must_be_non_negative"
	ErrDiscountExceedsSubtotal DiscountError = "discount_exceeds_subtotal"
	ErrDiscountForbidden       DiscountError = "discount_not_allowed_for_role"
	ErrDiscountOverLimit       DiscountError = "discount_limit_exceeded"
)

// validateDiscount returns the expected total (subtotal - discount) or
// an appropriate error.  Manager / owner / saas_admin have no cap;
// cashier discount is limited to cashierLimitPct percent of subtotal.
func validateDiscount(role string, subtotal, discount int64, cashierLimitPct int) (int64, error) {
	if discount < 0 {
		return 0, ErrNegativeDiscount
	}
	if subtotal <= 0 || discount == 0 {
		return subtotal - discount, nil
	}
	if discount > subtotal {
		return 0, ErrDiscountExceedsSubtotal
	}
	if role == "" || !roleCanDiscount(role) {
		return 0, ErrDiscountForbidden
	}
	pct := discount * 100 / subtotal
	if role == "cashier" && int(pct) > cashierLimitPct {
		return 0, fmt.Errorf("%w: %d%% > %d%% limit", ErrDiscountOverLimit, pct, cashierLimitPct)
	}
	return subtotal - discount, nil
}

func roleCanDiscount(role string) bool {
	switch role {
	case "owner", "manager", "cashier", "saas_admin":
		return true
	default:
		return false
	}
}
