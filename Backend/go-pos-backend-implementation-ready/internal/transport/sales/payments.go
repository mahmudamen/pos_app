package sales

import "fmt"

var allowedPaymentMethods = map[string]bool{"cash": true, "card": true, "mobile": true}

type paymentRequest struct {
	Method      string `json:"method"`
	AmountMinor int64  `json:"amount_minor"`
}

// normalizePayments turns the client-supplied tender lines into the canonical
// set to persist. It is unit-testable without a database.
//
//   - nil/empty means cash pays the whole sale (backwards compatible with
//     clients that predate split payments).
//   - otherwise each line must use an allowed method with a positive amount
//     and the lines must exactly sum to the sale total.
func normalizePayments(payments []paymentRequest, totalMinor int64) ([]paymentRequest, error) {
	if len(payments) == 0 {
		return []paymentRequest{{Method: "cash", AmountMinor: totalMinor}}, nil
	}
	var sum int64
	for _, p := range payments {
		if !allowedPaymentMethods[p.Method] {
			return nil, fmt.Errorf("payment method %q is not allowed", p.Method)
		}
		if p.AmountMinor <= 0 {
			return nil, fmt.Errorf("payment amount must be positive")
		}
		sum += p.AmountMinor
	}
	if sum != totalMinor {
		return nil, fmt.Errorf("payments total %d does not match sale total %d", sum, totalMinor)
	}
	return payments, nil
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
