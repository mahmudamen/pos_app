package registers

// accountTotals holds the Z-report aggregates pulled from the sales rows that
// reference a register session. Kept as a separate shape so buildSummary is a
// pure function that can be unit-tested without a database.
type accountTotals struct {
	salesCount    int64
	subtotalMinor int64
	discountMinor int64
	taxMinor      int64
	totalMinor    int64
	cashMinor     int64
	cardMinor     int64
	mobileMinor   int64
}

// SessionSummary is the Z-report the cashier sees on (and after) close.
type SessionSummary struct {
	SalesCount        int64 `json:"sales_count"`
	SubtotalMinor     int64 `json:"subtotal_minor"`
	DiscountMinor     int64 `json:"discount_minor"`
	TaxMinor          int64 `json:"tax_minor"`
	TotalMinor        int64 `json:"total_minor"`
	CashMinor         int64 `json:"cash_minor"`
	CardMinor         int64 `json:"card_minor"`
	MobileMinor       int64 `json:"mobile_minor"`
	ExpectedCashMinor int64 `json:"expected_cash_minor"`
}

// expectedCashMinor is what should be in the drawer: the opening float plus
// every cash tender taken during the session.
func expectedCashMinor(openingCashMinor, cashSalesMinor int64) int64 {
	return openingCashMinor + cashSalesMinor
}

// balanceDifference reconciles the drawer after close: negative means a
// shortage (too little cash counted), positive an overage.
func balanceDifference(closingCashMinor, expectedCashMinor int64) int64 {
	return closingCashMinor - expectedCashMinor
}

func buildSummary(t accountTotals, openingCashMinor int64) SessionSummary {
	return SessionSummary{
		SalesCount:        t.salesCount,
		SubtotalMinor:     t.subtotalMinor,
		DiscountMinor:     t.discountMinor,
		TaxMinor:          t.taxMinor,
		TotalMinor:        t.totalMinor,
		CashMinor:         t.cashMinor,
		CardMinor:         t.cardMinor,
		MobileMinor:       t.mobileMinor,
		ExpectedCashMinor: expectedCashMinor(openingCashMinor, t.cashMinor),
	}
}
