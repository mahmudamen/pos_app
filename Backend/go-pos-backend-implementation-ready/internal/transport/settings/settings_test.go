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
	if got := defaultSettingValue(KeyAllowNegativeStock); got != "false" {
		t.Errorf("default allow_negative_stock = %q, want false", got)
	}
	if got := defaultSettingValue(KeyDiscountMode); got != "cap" {
		t.Errorf("default discount_mode = %q, want cap", got)
	}
	if got := defaultSettingValue(KeyMaxDiscountPct); got != "0" {
		t.Errorf("default max_discount_pct = %q, want 0 (no global limit)", got)
	}
	for _, key := range []string{KeyManagerRefund, KeyManagerDiscount} {
		if got := defaultSettingValue(key); got != "true" {
			t.Errorf("default %s = %q, want true", key, got)
		}
	}
	for _, key := range []string{KeyManagerClose, KeyManagerDelete, KeyManagerNegative} {
		if got := defaultSettingValue(key); got != "false" {
			t.Errorf("default %s = %q, want false", key, got)
		}
	}
	for _, key := range []string{KeyShowCustomer, KeyShowDiscount, KeyShowPrice, KeyShowQty, KeyShowNumpad, KeyShowDelete} {
		if got := defaultSettingValue(key); got != "true" {
			t.Errorf("default %s = %q, want true", key, got)
		}
	}
	if got := defaultSettingValue(KeyStockType); got != "on_hand" {
		t.Errorf("default stock_type = %q, want on_hand", got)
	}
	if got := defaultSettingValue(KeyBlockOutOfStock); got != "true" {
		t.Errorf("default block_out_of_stock = %q, want true", got)
	}
	if got := defaultSettingValue(KeyLowStockThreshold); got != "5" {
		t.Errorf("default low_stock_threshold = %q, want 5", got)
	}
	for _, key := range []string{KeyLowStockWarning, KeyValidateStockPayment, KeyRefreshButton} {
		if got := defaultSettingValue(key); got != "true" {
			t.Errorf("default %s = %q, want true", key, got)
		}
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
		{KeyAllowNegativeStock, "true", false},
		{KeyAllowNegativeStock, "false", false},
		{KeyAllowNegativeStock, "yes", true},

		// ma_pos_base parity keys.
		{KeyDiscountMode, "cap", false},
		{KeyDiscountMode, "block", false},
		{KeyDiscountMode, "warn", false},
		{KeyDiscountMode, "off", true},
		{KeyDiscountMode, "", true},
		{KeyMaxDiscountPct, "0", false},
		{KeyMaxDiscountPct, "100", false},
		{KeyMaxDiscountPct, "50", false},
		{KeyMaxDiscountPct, "101", true},
		{KeyMaxDiscountPct, "-1", true},
		{KeyMaxDiscountPct, "abc", true},
		{KeyMaxDiscountPct, "05", true},
		{KeyManagerRefund, "true", false},
		{KeyManagerRefund, "false", false},
		{KeyManagerRefund, "1", true},
		{KeyManagerDiscount, "true", false},
		{KeyManagerDelete, "false", false},
		{KeyManagerNegative, "true", false},
		{KeyManagerClose, "false", false},
		{KeyShowCustomer, "true", false},
		{KeyShowCustomer, "false", false},
		{KeyShowCustomer, "on", true},
		{KeyShowDiscount, "true", false},
		{KeyShowPrice, "true", false},
		{KeyShowQty, "false", false},
		{KeyShowNumpad, "true", false},
		{KeyShowDelete, "true", false},

		// ma_rs_pos_stock parity keys.
		{KeyStockType, "on_hand", false},
		{KeyStockType, "available", false},
		{KeyStockType, "none", true},
		{KeyStockType, "", true},
		{KeyBlockOutOfStock, "true", false},
		{KeyBlockOutOfStock, "false", false},
		{KeyBlockOutOfStock, "1", true},
		{KeyLowStockThreshold, "0", false},
		{KeyLowStockThreshold, "5", false},
		{KeyLowStockThreshold, "99999", false},
		{KeyLowStockThreshold, "100000", true},
		{KeyLowStockThreshold, "-1", true},
		{KeyLowStockThreshold, "ten", true},
		{KeyLowStockWarning, "false", false},
		{KeyLowStockWarning, "true", false},
		{KeyLowStockWarning, "yes", true},
		{KeyValidateStockPayment, "false", false},
		{KeyValidateStockPayment, "true", false},
		{KeyValidateStockPayment, "", true},
		{KeyRefreshButton, "false", false},
		{KeyRefreshButton, "true", false},
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

func TestSettingKeysCoversEveryDefinition(t *testing.T) {
	keys := settingKeys()
	if len(keys) != len(settingDefinitions) {
		t.Fatalf("settingKeys returned %d keys, want %d", len(keys), len(settingDefinitions))
	}
	seen := map[string]bool{}
	for _, k := range keys {
		if k == "" {
			t.Fatal("settingKeys returned a blank key")
		}
		if seen[k] {
			t.Fatalf("settingKeys returned duplicate key %q", k)
		}
		seen[k] = true
		if _, ok := settingDefinitions[k]; !ok {
			t.Fatalf("settingKeys returned key %q not in settingDefinitions", k)
		}
	}
	for key := range settingDefinitions {
		if !seen[key] {
			t.Fatalf("settingKeys missing definition key %q", key)
		}
	}
}
