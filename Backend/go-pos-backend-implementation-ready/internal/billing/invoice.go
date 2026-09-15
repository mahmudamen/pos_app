package billing

import "fmt"

// Invoice lifecycle states.
const (
	InvoiceOpen     = "open"
	InvoicePaid     = "paid"
	InvoiceVoid     = "void"
	InvoiceRefunded = "refunded"
)

// InvoiceStatuses lists every valid invoice state.
func InvoiceStatuses() []string {
	return []string{InvoiceOpen, InvoicePaid, InvoiceVoid, InvoiceRefunded}
}

// ValidInvoiceStatus reports whether s is a known invoice state.
func ValidInvoiceStatus(s string) bool {
	for _, candidate := range InvoiceStatuses() {
		if s == candidate {
			return true
		}
	}
	return false
}

var invoiceTransitions = map[string]map[string]bool{
	InvoiceOpen: {
		InvoicePaid: true,
		InvoiceVoid: true,
	},
	InvoicePaid: {
		InvoiceRefunded: true,
	},
	InvoiceVoid:     {},
	InvoiceRefunded: {},
}

// CanTransitionInvoice reports whether an invoice may move from -> to. Same
// state is allowed for idempotency.
func CanTransitionInvoice(from, to string) bool {
	if from == to {
		return ValidInvoiceStatus(from)
	}
	if !ValidInvoiceStatus(from) || !ValidInvoiceStatus(to) {
		return false
	}
	return invoiceTransitions[from][to]
}

// ValidateInvoiceTransition returns a descriptive error for a rejected move.
func ValidateInvoiceTransition(from, to string) error {
	if !ValidInvoiceStatus(to) {
		return fmt.Errorf("unknown invoice status %q", to)
	}
	if !CanTransitionInvoice(from, to) {
		return fmt.Errorf("cannot move invoice from %q to %q", from, to)
	}
	return nil
}

// Invoice is one billable charge against a tenant.
type Invoice struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenant_id"`
	SubscriptionID string `json:"subscription_id,omitempty"`
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	Provider       string `json:"provider"`
	ProviderRef    string `json:"provider_ref,omitempty"`
	Description    string `json:"description"`
}

// Validate checks the invariants a new invoice must satisfy.
func (i Invoice) Validate() error {
	if i.AmountMinor < 0 {
		return fmt.Errorf("amount must not be negative")
	}
	if len(i.Currency) != 3 {
		return fmt.Errorf("currency must be a 3-letter code")
	}
	if i.Description == "" {
		return fmt.Errorf("description is required")
	}
	if len(i.Description) > 255 {
		return fmt.Errorf("description must be at most 255 characters")
	}
	return nil
}

// InvoiceTotalMinor sums a set of invoices, ignoring void ones so a control
// panel "outstanding" figure is not inflated by cancelled bills.
func InvoiceTotalMinor(invoices []Invoice) int64 {
	var total int64
	for _, inv := range invoices {
		if inv.Status == InvoiceVoid {
			continue
		}
		total += inv.AmountMinor
	}
	return total
}
