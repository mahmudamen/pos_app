// Package settings exposes per-tenant POS preferences (Odoo-style settings).
package settings

import (
	"errors"
	"strings"
)

// Validators define the known setting keys and how to validate a candidate value.
type validator func(value string) error

const (
	// KeyDefaultPaymentMethod is the method pre-selected on the payment sheet.
	KeyDefaultPaymentMethod = "pos.default_payment_method"
	// KeyShowStockBadges toggles the stock-on-hand badge on the product grid.
	KeyShowStockBadges = "pos.show_stock_badges"
	// KeyReceiptFooter is the message appended to receipts.
	KeyReceiptFooter = "pos.receipt_footer"
	// KeyAllowNegativeStock permits checkout and adjustments to drive the
	// on-hand counter below zero (backorders).
	KeyAllowNegativeStock = "inventory.allow_negative_stock"
)

var settingDefinitions = map[string]validator{
	KeyDefaultPaymentMethod: func(v string) error {
		switch v {
		case "cash", "card", "mobile":
			return nil
		}
		return errors.New("must be one of cash, card, mobile")
	},
	KeyShowStockBadges: func(v string) error {
		if v == "true" || v == "false" {
			return nil
		}
		return errors.New("must be true or false")
	},
	KeyReceiptFooter: func(v string) error {
		if len(v) > 500 {
			return errors.New("must be at most 500 characters")
		}
		return nil
	},
	KeyAllowNegativeStock: func(v string) error {
		if v == "true" || v == "false" {
			return nil
		}
		return errors.New("must be true or false")
	},
}

// defaultSettingValue is returned for keys the tenant has never configured.
func defaultSettingValue(key string) string {
	switch key {
	case KeyDefaultPaymentMethod:
		return "cash"
	case KeyShowStockBadges:
		return "true"
	case KeyAllowNegativeStock:
		return "false"
	default:
		return ""
	}
}

// settingKeys returns the sorted whitelist of known keys.
func settingKeys() []string {
	keys := make([]string, 0, len(settingDefinitions))
	for key := range settingDefinitions {
		keys = append(keys, key)
	}
	return keys
}

// canManageSettings reports whether a role may change tenant settings.
func canManageSettings(role string) bool {
	switch role {
	case "owner", "manager", "saas_admin":
		return true
	default:
		return false
	}
}

// hasBearer reports whether header starts with the Bearer scheme.
func hasBearer(header string) bool {
	return strings.HasPrefix(header, "Bearer ")
}

// trimBearer strips the Bearer prefix.
func trimBearer(header string) string {
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}

// validateSetting reports error for an unknown key or an invalid value.
func validateSetting(key, value string) error {
	check, ok := settingDefinitions[key]
	if !ok {
		return errors.New("unknown setting key")
	}
	if err := check(strings.TrimSpace(value)); err != nil {
		return err
	}
	return nil
}
