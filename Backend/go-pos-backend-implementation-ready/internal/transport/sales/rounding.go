package sales

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ParseRoundingMode maps a pos.rounding_mode setting value to its minor-unit
// denomination. "" and "off" mean the tenant does not round; "25"/"50"/"100"
// round the payable to the nearest 0.25/0.50/1.00 EGP (minor units are piastres).
func ParseRoundingMode(mode string) (denom int64, ok bool) {
	switch mode {
	case "", "off":
		return 0, true
	case "25":
		return 25, true
	case "50":
		return 50, true
	case "100":
		return 100, true
	}
	return 0, false
}

// RoundingDelta is the amount added to a sale's nominal total to reach the
// rounded payable cash amount. It is always >= 0 (nearest-half-up, clamped so a
// total is never rounded below its nominal value), which satisfies the
// sales.rounding_minor CHECK constraint. The zero value means no adjustment:
// either rounding is off or the total already sits on a denomination boundary.
func RoundingDelta(mode string, totalMinor int64) (int64, error) {
	denom, ok := ParseRoundingMode(mode)
	if !ok {
		return 0, fmt.Errorf("unknown rounding mode %q", mode)
	}
	if denom == 0 || totalMinor <= 0 {
		return 0, nil
	}
	rem := totalMinor % denom
	if rem == 0 {
		return 0, nil
	}
	if rem*2 >= denom {
		return denom - rem, nil
	}
	return 0, nil
}

// roundingMode returns the tenant's pos.rounding_mode value ("off" when the
// tenant has never configured it).
func roundingMode(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) (string, error) {
	var mode string
	err := tx.QueryRow(ctx, `
		SELECT COALESCE((SELECT value FROM tenant_settings
		                 WHERE tenant_id = $1 AND key = 'pos.rounding_mode'), 'off')`,
		tenantID).Scan(&mode)
	if err != nil {
		return "", err
	}
	return mode, nil
}
