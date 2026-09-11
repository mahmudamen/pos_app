package sales

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const defaultPointsPer100 int64 = 1

func saleCustomerID(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func loyaltyRateFromValue(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultPointsPer100
	}
	rate, err := strconv.ParseInt(value, 10, 64)
	if err != nil || rate < 0 {
		return defaultPointsPer100
	}
	return rate
}

func pointsForTotal(totalMinor int64, rate int64) int64 {
	if totalMinor <= 0 {
		return 0
	}
	points := (totalMinor / 100) * rate
	if points < 0 {
		return 0
	}
	return points
}

func loyaltyPointsForSale(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, totalMinor int64) (int64, error) {
	var value string
	err := q.QueryRow(ctx, `SELECT value FROM tenant_settings WHERE key = 'loyalty.points_per_100'`).Scan(&value)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	return pointsForTotal(totalMinor, loyaltyRateFromValue(value)), nil
}
