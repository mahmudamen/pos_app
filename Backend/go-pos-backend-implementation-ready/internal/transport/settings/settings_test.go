package settings

import "testing"

func TestDefaultSettingValues(t *testing.T) {
	if got := defaultSettingValue(KeyDefaultPaymentMethod); got != "cash" {
		t.Errorf("default default_payment_method = %q, want cash", got)
	}
	if got := defaultSettingValue(KeyShowStockBadges); got != "true" {
		t.Errorf("default show_stock_badges = %q, want true", got)
	}
	if got := defaultSettingValue(KeyReceiptFooter); got != "" {
		t.Errorf("default receipt_footer = %q, want empty", got)
	}
	if got := defaultSettingValue("unknown.key"); got != "" {
		t.Errorf("default unknown = %q, want empty", got)
	}
}

func TestValidateSetting(t *testing.T) {
	cases := []struct {
		key, value string
		wantErr    bool
	}{
		{KeyDefaultPaymentMethod, "cash", false},
		{KeyDefaultPaymentMethod, "card", false},
		{KeyDefaultPaymentMethod, "mobile", false},
		{KeyDefaultPaymentMethod, "credit", true},
		{KeyDefaultPaymentMethod, "", true},
		{KeyShowStockBadges, "true", false},
		{KeyShowStockBadges, "false", false},
		{KeyShowStockBadges, "yes", true},
		{KeyReceiptFooter, "", false},
		{KeyReceiptFooter, "thank you for shopping", false},
		{KeyReceiptFooter, string(make([]rune, 501)), true},
		{"not.a.key", "anything", true},
	}
	for _, c := range cases {
		err := validateSetting(c.key, c.value)
		if (err != nil) != c.wantErr {
			t.Errorf("validateSetting(%q, %q) err = %v, wantErr %v", c.key, c.value, err, c.wantErr)
		}
	}
}

func TestValidatedValuesAreTrimmed(t *testing.T) {
	if err := validateSetting(KeyDefaultPaymentMethod, " card "); err != nil {
		t.Errorf("trimmed value should validate: %v", err)
	}
}

func TestCanManageSettings(t *testing.T) {
	for _, role := range []string{"owner", "manager", "saas_admin"} {
		if !canManageSettings(role) {
			t.Errorf("canManageSettings(%q) = false, want true", role)
		}
	}
	for _, role := range []string{"cashier", "", "guest"} {
		if canManageSettings(role) {
			t.Errorf("canManageSettings(%q) = true, want false", role)
		}
	}
}
