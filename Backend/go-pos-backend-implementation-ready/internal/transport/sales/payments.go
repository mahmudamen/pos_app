package sales

import "fmt"

var allowedPaymentMethods = map[string]bool{"cash": true, "card": true, "mobile": true}

type paymentRequest struct {
	Method      string `json:"method"`
	AmountMinor int64  `json:"amount_minor"`
	TipMinor    int64  `json:"tip_minor,omitempty"`
}

// normalizePayments turns the client-supplied tender lines into the canonical
// set to persist. It is unit-testable without a database.
//
//   - nil/empty means cash pays the whole sale (backwards compatible with
//     clients that predate split payments).
//   - otherwise each line must use an allowed method with a positive amount
//     and the lines must exactly sum to the sale total.
//   - tip_minor (>= 0) is a gratuity on top of the exact tender sum; the
//     aggregate is returned so callers can store sales.tips_minor.
func normalizePayments(payments []paymentRequest, totalMinor int64) ([]paymentRequest, int64, error) {
	var tips int64
	if len(payments) == 0 {
		return []paymentRequest{{Method: "cash", AmountMinor: totalMinor}}, 0, nil
	}
	var sum int64
	for _, p := range payments {
		if !allowedPaymentMethods[p.Method] {
			return nil, 0, fmt.Errorf("payment method %q is not allowed", p.Method)
		}
		if p.AmountMinor <= 0 {
			return nil, 0, fmt.Errorf("payment amount must be positive")
		}
		if p.TipMinor < 0 {
			return nil, 0, fmt.Errorf("tip amount must not be negative")
		}
		sum += p.AmountMinor
		tips += p.TipMinor
	}
	if sum != totalMinor {
		return nil, 0, fmt.Errorf("payments total %d does not match sale total %d", sum, totalMinor)
	}
	return payments, tips, nil
}

// primaryPaymentMethod is the method of the largest line (stable first on
// ties), "cash" when there are no lines — used for sales.payment_method.
func primaryPaymentMethod(payments []paymentRequest) string {
	if len(payments) == 0 {
		return "cash"
	}
	primary := payments[0]
	for _, p := range payments[1:] {
		if p.AmountMinor > primary.AmountMinor {
			primary = p
		}
	}
	return primary.Method
}
