// Package posgo holds the wire types and pure helpers for the POS.Go control
// plane. Nothing here touches the network: the client in client.go does, and
// the pure formatting/label helpers stay unit-testable.
package posgo

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Tenant status values the control plane reports. These mirror the CHECK
// constraint on tenants.status widened by migration 042_tenant_stopped.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusClosed    = "closed"
	StatusDisabled  = "disabled"
	StatusTrial     = "trial"
)

// Tenant is one row of GET /v1/saas/tenants.
type Tenant struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Slug               string   `json:"slug"`
	Status             string   `json:"status"`
	BusinessType       string   `json:"business_type"`
	Plan               string   `json:"plan"`
	CountryCode        string   `json:"country_code"`
	CurrencyCode       string   `json:"currency_code"`
	DefaultLanguage    string   `json:"default_language"`
	Subdomain          string   `json:"subdomain"`
	SubscriptionStatus string   `json:"subscription_status"`
	PlanFeatures       []string `json:"plan_features"`
	MaxUsers           int      `json:"max_users"`
	MaxProducts        int      `json:"max_products"`
	Users              int      `json:"users"`
	Products           int      `json:"products"`
	TotalSales         int64    `json:"total_sales"`
	RevenueMinor       int64    `json:"revenue_minor"`
	CreatedAt          string   `json:"created_at"`
	TrialEndsAt        string   `json:"trial_ends_at"`
}

// HasFeature reports whether the tenant's plan grants the named entitlement.
// The server derives Subdomain from this list, so the two must agree.
func (t Tenant) HasFeature(name string) bool {
	for _, f := range t.PlanFeatures {
		if f == name {
			return true
		}
	}
	return false
}

// IsStopped reports whether the tenant is disabled, the state that blocks login
// with organization_stopped until it is reactivated.
func (t Tenant) IsStopped() bool { return t.Status == StatusDisabled }

// Summary is GET /v1/saas/summary. The platform tenant is already excluded by
// the server, so these counts are merchants only.
type Summary struct {
	TotalTenants      int             `json:"total_tenants"`
	TotalUsers        int             `json:"total_users"`
	TotalSales        int64           `json:"total_sales"`
	RevenueMinor      int64           `json:"revenue_minor"`
	SupportedCountry  string          `json:"supported_country"`
	SupportedCurrency string          `json:"supported_currency"`
	ByBusiness        []BusinessSlice `json:"by_business"`
}

// BusinessSlice is one per-business-type aggregate row.
type BusinessSlice struct {
	BusinessType string `json:"business_type"`
	Tenants      int    `json:"tenants"`
	Users        int    `json:"users"`
	RevenueMinor int64  `json:"revenue_minor"`
}

// Analytics is GET /v1/saas/tenants/:id/analytics.
type Analytics struct {
	Tenant       Tenant          `json:"tenant"`
	Counts       AnalyticsCounts `json:"counts"`
	Today        TodayStats      `json:"today"`
	RevenueTrend []TrendPoint    `json:"revenue_trend"`
	TopProducts  []TopProduct    `json:"top_products"`
	RecentSales  []RecentSale    `json:"recent_sales"`
}

// AnalyticsCounts holds the per-tenant user and product tallies.
type AnalyticsCounts struct {
	Users    int `json:"users"`
	Products int `json:"products"`
}

// TodayStats is the tenant's trading figures for the current day.
type TodayStats struct {
	Date         string `json:"date"`
	RevenueMinor int64  `json:"revenue_minor"`
	SalesCount   int64  `json:"sales_count"`
	AvgSaleMinor int64  `json:"avg_sale_minor"`
	ItemsSold    int64  `json:"items_sold"`
}

// TrendPoint is one day of the 7-day revenue trend.
type TrendPoint struct {
	Day          string `json:"day"`
	RevenueMinor int64  `json:"revenue_minor"`
}

// TopProduct is one row of the tenant's best sellers.
type TopProduct struct {
	ProductName  string `json:"product_name"`
	SKU          string `json:"sku"`
	Quantity     int64  `json:"quantity"`
	RevenueMinor int64  `json:"revenue_minor"`
}

// RecentSale is one row of the tenant's latest completed sales.
type RecentSale struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	TotalMinor    int64  `json:"total_minor"`
	Currency      string `json:"currency"`
	PaymentMethod string `json:"payment_method"`
	Cashier       string `json:"cashier"`
	CreatedAt     string `json:"created_at"`
}

// TenantPage is the paginated envelope body of GET /v1/saas/tenants.
type TenantPage struct {
	Tenants []Tenant
	Page    int
	Limit   int
	Total   int64
}

// currencySymbols maps the ISO codes the demo tenants use onto the glyph an
// operator expects to see. Anything unlisted falls back to the bare code so a
// new currency renders as "XYZ 1,234.50" instead of losing its currency.
var currencySymbols = map[string]string{
	"EGP": "E£",
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"SAR": "SAR ",
	"AED": "AED ",
}

// FormatMinor renders integer minor units as a human amount. The API stores
// money as int64 minor units everywhere, so the division happens here and
// nowhere else. A nil currency falls back to EGP, the platform's supported
// currency, which is what GET /v1/saas/summary reports.
func FormatMinor(minor int64, currency string) string {
	code := strings.ToUpper(strings.TrimSpace(currency))
	symbol, ok := currencySymbols[code]
	if !ok {
		if code == "" {
			code = "EGP"
			symbol = currencySymbols[code]
		} else {
			symbol = code + " "
		}
	}
	negative := minor < 0
	if negative {
		minor = -minor
	}
	// Whole and fraction are computed separately so a 1000s-separated integer
	// never leaks into the fraction, which is the classic minor-units bug.
	whole := strconv.FormatInt(minor/100, 10)
	frac := strconv.FormatInt(minor%100, 10)
	if len(frac) < 2 {
		frac = "0" + frac
	}
	out := symbol + groupThousands(whole) + "." + frac
	if negative {
		return "-" + out
	}
	return out
}

// groupThousands inserts thin separators every three digits. Operands stay in
// the ASCII range so rune slicing is safe.
func groupThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	var b strings.Builder
	lead := len(digits) % 3
	if lead > 0 {
		b.WriteString(digits[:lead])
	}
	for i := lead; i < len(digits); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

// FormatCount renders large counts compactly for narrow table cells: 7735
// becomes 7.7k so a column does not push the layout off screen.
func FormatCount(n int64) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 10_000:
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	default:
		return strconv.FormatInt(n, 10)
	}
}

// StatusPill is the short, colour-free label for a tenant status. The colour
// itself is applied by the view, not encoded here, so the same string works in
// a table cell and in an accessibility announcement.
func StatusPill(status string) string {
	switch status {
	case StatusActive:
		return "ACTIVE"
	case StatusSuspended:
		return "SUSPENDED"
	case StatusDisabled:
		return "STOPPED"
	case StatusClosed:
		return "CLOSED"
	default:
		if status == "" {
			return "UNKNOWN"
		}
		return strings.ToUpper(status)
	}
}

// ShortID trims a UUID down to its first segment for narrow cells. The full ID
// stays on the tenant detail screen, where it can be copied.
func ShortID(id string) string {
	if i := strings.Index(id, "-"); i > 0 {
		return id[:i]
	}
	return id
}

// FormatDate renders an RFC3339 timestamp as a local date, falling back to the
// raw string when the API sends something unparseable or empty.
func FormatDate(ts string) string {
	if ts == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ts
	}
	return t.Local().Format("2006-01-02")
}

// RelativeAge renders how long ago a timestamp was, at a resolution that stays
// useful for an ops screen. Zero-value times render as a dash.
func RelativeAge(ts string, now time.Time) string {
	if ts == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return "-"
	}
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// Filters is the operator's current control-plane query. The zero value means
// "no filter", which is exactly what the API expects for omitted parameters.
type Filters struct {
	Search       string
	BusinessType string
	Status       string
	Plan         string
	Page         int
	Limit        int
}

// Values are the dropdown options offered for the business-type and status
// filters, in a deliberate display order rather than alphabetical.
var (
	BusinessTypeOptions = []string{"", "restaurant", "coffee_shop", "grocery", "book_store", "mobile_shop", "computer_shop"}
	StatusOptions       = []string{"", StatusActive, StatusSuspended, StatusDisabled, StatusClosed}
)

// Query renders the filters as a URL query string, omitting every unset field
// so the server applies its own defaults instead of being pinned to a blank.
// The leading "?" is included so callers can concatenate directly.
func (f Filters) Query() string {
	values := url.Values{}
	if q := strings.TrimSpace(f.Search); q != "" {
		values.Set("q", q)
	}
	if f.BusinessType != "" {
		values.Set("business_type", f.BusinessType)
	}
	if f.Status != "" {
		values.Set("status", f.Status)
	}
	if f.Plan != "" {
		values.Set("plan", f.Plan)
	}
	page := f.Page
	if page < 1 {
		page = 1
	}
	limit := f.Limit
	if limit < 1 || limit > 100 {
		limit = 25
	}
	values.Set("page", strconv.Itoa(page))
	values.Set("limit", strconv.Itoa(limit))
	return "?" + values.Encode()
}

// TotalPages is how many pages the current filter set spans, never below 1 so
// the pager can always render "1 / N" on an empty result set.
func TotalPages(total int64, limit int) int {
	if limit < 1 {
		return 1
	}
	pages := int((total + int64(limit) - 1) / int64(limit))
	if pages < 1 {
		return 1
	}
	return pages
}
