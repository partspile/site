package ad

import (
	"time"

	"github.com/partspile/site/internal/config"
)

// ComputeExpiresAt is the listing lifetime measured from now.
func ComputeExpiresAt(now time.Time) time.Time {
	return now.AddDate(0, config.AdExpireMonths, 0)
}

// RenewEligible is true when expires_at is within AdExpireRenewWithinMonths.
func RenewEligible(expiresAt, now time.Time) bool {
	limit := now.AddDate(0, config.AdExpireRenewWithinMonths, 0)
	return !expiresAt.After(limit)
}
