package platform

import "testing"

// Pure-unit coverage for the tenant stop/backup slice: JSON escaping used by
// the streaming export and the RLS-scoped table catalog asserting the
// leak-prevention invariants (no platform tables, no observability table).

func TestJsonEscape(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{"demo-restaurant", "demo-restaurant"},
		{`has "quotes"`, `has \"quotes\"`},
		{`back\slash`, `back\\slash`},
		{"new\nline", `new\nline`},
		{"tab\there", `tab\there`},
		{"émojis 🚀", `émojis 🚀`},
		{"", ""},
	}
	for _, tc := range cases {
		if got := jsonEscape(tc.in); got != tc.want {
			t.Errorf("jsonEscape(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBackupTenantTables(t *testing.T) {
	t.Run("exhaustive catalog", func(t *testing.T) {
		if len(tenantTables) != 28 {
			t.Fatalf("tenantTables has %d entries, want 28", len(tenantTables))
		}
	})
	t.Run("all names are unique lowercase identifiers", func(t *testing.T) {
		seen := make(map[string]bool, len(tenantTables))
		for _, name := range tenantTables {
			if seen[name] {
				t.Errorf("duplicate table %q in tenantTables", name)
			}
			if name == "" || name[0] < 'a' || name[0] > 'z' {
				t.Errorf("table %q is not a lowercase identifier", name)
			}
			for i := 1; i < len(name); i++ {
				c := name[i]
				ok := (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
				if !ok {
					t.Errorf("table %q contains invalid byte %q", name, c)
				}
			}
			seen[name] = true
		}
	})
	t.Run("covers the tenant business tables", func(t *testing.T) {
		for _, want := range []string{
			"users", "products", "categories", "sales", "sale_items",
			"sale_payments", "sale_refunds", "customers", "customer_loyalty_log",
			"product_variants", "product_lots", "floors", "restaurant_tables",
			"register_sessions", "inventory_adjustments", "tenant_settings",
			"sync_commands", "sessions", "devices",
		} {
			if !contains(tenantTables, want) {
				t.Errorf("tenantTables missing %q", want)
			}
		}
	})
	t.Run("excludes platform and observability tables", func(t *testing.T) {
		for _, forbidden := range []string{
			"plans", "subscriptions", "invoices", "billing_events",
			"accounts", "account_memberships", "organizations",
			"audit_log", "countries", "currencies",
			"client_events", "migrations",
		} {
			if contains(tenantTables, forbidden) {
				t.Errorf("tenantTables MUST NOT export %q (cross-tenant/platform RLS gap)", forbidden)
			}
		}
	})
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
