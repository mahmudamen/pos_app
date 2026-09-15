// Package settings exposes per-tenant POS preferences (Odoo-style settings).
package settings

import (
	"errors"
	"strconv"
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

	// ma_pos_base parity (Odoo POS Security Framework):
	// KeyDiscountMode is the discount enforcement policy
	// (cap = auto-reduce, block = reject until manager approves, warn = allow).
	KeyDiscountMode = "pos.discount_mode"
	// KeyMaxDiscountPct is the tenant-wide ceiling on any user's discount
	// (0 = no global limit; users can't exceed it via their own cap).
	KeyMaxDiscountPct = "pos.max_discount_pct"
	// KeyManagerRefund gates refunds behind the manager PIN.
	KeyManagerRefund = "pos.manager.refund"
	// KeyManagerDiscount requires a manager PIN when discount exceeds the cap.
	KeyManagerDiscount = "pos.manager.discount"
	// KeyManagerDelete requires a manager PIN for order/line deletion.
	KeyManagerDelete = "pos.manager.delete"
	// KeyManagerNegative requires a manager PIN for negative stock sells.
	KeyManagerNegative = "pos.manager.negative"
	// KeyManagerClose requires the manager PIN to close a register session.
	KeyManagerClose = "pos.manager.close"

	// UI visibility toggles per POS (hide customers/discount/price/qty/
	// numpad/delete buttons). Not every toggle has a Flutter counterpart yet;
	// unsupported ones default to visible.
	KeyShowCustomer = "pos.ui.show_customer"
	KeyShowDiscount = "pos.ui.show_discount"
	KeyShowPrice    = "pos.ui.show_price"
	KeyShowQty      = "pos.ui.show_qty"
	KeyShowNumpad   = "pos.ui.show_numpad"
	KeyShowDelete   = "pos.ui.show_delete"

	// ma_rs_pos_stock parity (Odoo POS Stock Management):
	// KeyStockType selects the quantity shown on badges (on_hand or available).
	KeyStockType = "pos.stock_type"
	// KeyBlockOutOfStock forbids adding zero-stock products to a cart.
	KeyBlockOutOfStock = "pos.block_out_of_stock"
	// KeyLowStockThreshold is the on-hand level at or under which a product
	// turns amber ("Low Stock"); 0 = treat only exact zero as out.
	KeyLowStockThreshold = "pos.low_stock_threshold"
	// KeyLowStockWarning shows the amber badge and the low-stock-only filter.
	KeyLowStockWarning = "pos.low_stock_warning"
	// KeyValidateStockPayment re-checks stock against the last catalog load
	// before checkout (the server always locks stock; this is the client
	// double-safety from the module).
	KeyValidateStockPayment = "pos.validate_stock_payment"
	// KeyRefreshButton shows a manual refresh action on the POS navbar.
	KeyRefreshButton = "pos.refresh_button"
)

func isTrueFalse(v string) error {
	if v == "true" || v == "false" {
		return nil
	}
	return errors.New("must be true or false")
}

func isDiscountMode(v string) error {
	switch v {
	case "cap", "block", "warn":
		return nil
	}
	return errors.New("must be one of cap, block, warn")
}

func isPercent(v string) error {
	value := strings.TrimSpace(v)
	if value == "" {
		return errors.New("must be a percentage between 0 and 100")
	}
	if value == "0" {
		return nil
	}
	if len(value) > 1 && value[0] == '0' {
		return errors.New("must be a percentage between 0 and 100")
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return errors.New("must be a percentage between 0 and 100")
		}
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 1 || n > 100 {
		return errors.New("must be a percentage between 0 and 100")
	}
	return nil
}

func isStockType(v string) error {
	switch v {
	case "on_hand", "available":
		return nil
	}
	return errors.New("must be one of on_hand, available")
}

func isThreshold(v string) error {
	value := strings.TrimSpace(v)
	if value == "" {
		return errors.New("must be a number between 0 and 99999")
	}
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return errors.New("must be a number between 0 and 99999")
		}
	}
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 || n > 99999 {
		return errors.New("must be a number between 0 and 99999")
	}
	return nil
}

var settingDefinitions = map[string]validator{
	KeyDefaultPaymentMethod: func(v string) error {
		switch v {
		case "cash", "card", "mobile":
			return nil
		}
		return errors.New("must be one of cash, card, mobile")
	},
	KeyShowStockBadges: isTrueFalse,
	KeyReceiptFooter: func(v string) error {
		if len(v) > 500 {
			return errors.New("must be at most 500 characters")
		}
		return nil
	},
	KeyAllowNegativeStock:   isTrueFalse,
	KeyDiscountMode:         isDiscountMode,
	KeyMaxDiscountPct:       isPercent,
	KeyManagerRefund:        isTrueFalse,
	KeyManagerDiscount:      isTrueFalse,
	KeyManagerDelete:        isTrueFalse,
	KeyManagerNegative:      isTrueFalse,
	KeyManagerClose:         isTrueFalse,
	KeyShowCustomer:         isTrueFalse,
	KeyShowDiscount:         isTrueFalse,
	KeyShowPrice:            isTrueFalse,
	KeyShowQty:              isTrueFalse,
	KeyShowNumpad:           isTrueFalse,
	KeyShowDelete:           isTrueFalse,
	KeyStockType:            isStockType,
	KeyBlockOutOfStock:      isTrueFalse,
	KeyLowStockThreshold:    isThreshold,
	KeyLowStockWarning:      isTrueFalse,
	KeyValidateStockPayment: isTrueFalse,
	KeyRefreshButton:        isTrueFalse,
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
	case KeyDiscountMode:
		return "cap"
	case KeyMaxDiscountPct:
		return "0"
	case KeyManagerRefund:
		return "true"
	case KeyManagerDiscount:
		return "true"
	case KeyManagerDelete:
		return "false"
	case KeyManagerNegative:
		return "false"
	case KeyManagerClose:
		return "false"
	case KeyShowCustomer, KeyShowDiscount, KeyShowPrice,
		KeyShowQty, KeyShowNumpad, KeyShowDelete:
		return "true"
	case KeyStockType:
		return "on_hand"
	case KeyBlockOutOfStock:
		return "true"
	case KeyLowStockThreshold:
		return "5"
	case KeyLowStockWarning:
		return "true"
	case KeyValidateStockPayment:
		return "true"
	case KeyRefreshButton:
		return "true"
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
