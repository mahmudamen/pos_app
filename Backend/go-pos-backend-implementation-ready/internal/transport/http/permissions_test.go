package http

import "testing"

func TestHasPermission(t *testing.T) {
	cases := []struct {
		role     string
		resource string
		action   string
		want     bool
	}{
		{"cashier", "pos", "open", true},
		{"cashier", "pos", "close", true},
		{"cashier", "pos", "sale", true},
		{"cashier", "pos", "discount", true},
		{"cashier", "pos", "refund", false},
		{"cashier", "pos", "read", true},
		{"cashier", "customers", "read", true},
		{"cashier", "customers", "write", false},
		{"manager", "customers", "read", true},
		{"manager", "customers", "write", true},
		{"manager", "pos", "refund", true},
		{"owner", "pos", "refund", true},
		{"saas_admin", "pos", "refund", true},
		{"cashier", "inventory", "adjust", false},
		{"manager", "pos", "open", true},
		{"manager", "inventory", "adjust", true},
		{"manager", "pos", "discount", true},
		{"owner", "pos", "open", true},
		{"owner", "inventory", "adjust", true},
		{"saas_admin", "saas", "admin", true},
		{"saas_admin", "inventory", "adjust", true},
		{"unknown", "pos", "open", false},
		{"", "pos", "open", false},
		{"cashier", "saas", "admin", false},
	}
	for _, tc := range cases {
		got := HasPermission(tc.role, tc.resource, tc.action)
		if got != tc.want {
			t.Errorf("HasPermission(%q, %q, %q) = %v, want %v",
				tc.role, tc.resource, tc.action, got, tc.want)
		}
	}
}
