package ad

import (
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/facet"
)

// ExpiresAtFromSaleEnd is end of sale_end_date (UTC) plus the sale-end delay.
func ExpiresAtFromSaleEnd(dateYYYYMMDD string) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", dateYYYYMMDD, time.UTC)
	if err != nil {
		return time.Time{}, err
	}
	endOfSaleDay := day.AddDate(0, 0, 1)
	return endOfSaleDay.Add(config.AdExpireSaleEndDelay), nil
}

// SaleEndDateString returns the sale_end_date facet value if present.
func SaleEndDateString(facets map[string]facet.Value) (string, bool) {
	v, ok := facets["sale_end_date"]
	if !ok {
		return "", false
	}
	s := v.DateString()
	if s == "" {
		return "", false
	}
	return s, true
}

// ComputeExpiresAt prefers sale_end_date + delay when present; otherwise
// now + AdExpireMonths.
func ComputeExpiresAt(facets map[string]facet.Value, now time.Time) time.Time {
	if s, ok := SaleEndDateString(facets); ok {
		if t, err := ExpiresAtFromSaleEnd(s); err == nil {
			return t
		}
	}
	return now.AddDate(0, config.AdExpireMonths, 0)
}

// RenewEligible is true when expires_at is within AdExpireRenewWithinMonths.
func RenewEligible(expiresAt, now time.Time) bool {
	limit := now.AddDate(0, config.AdExpireRenewWithinMonths, 0)
	return !expiresAt.After(limit)
}
