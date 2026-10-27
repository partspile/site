package ad

import (
	"time"

	"github.com/partspile/site/internal/db"
)

// MonthlyFunStat is a month-end snapshot of public site totals.
type MonthlyFunStat struct {
	Month              time.Time `db:"month"`
	RegisteredUsers    int       `db:"registered_users"`
	UsersWithActiveAds int       `db:"users_with_active_ads"`
	ActiveAds          int       `db:"active_ads"`
}

// MonthlyFunStats returns one row per calendar month from the earliest
// user or ad through the current month. Counts are as-of the end of
// each past month (or now for the current month): a user or ad is
// included if it already existed and had not yet been deleted or
// inactivated.
func MonthlyFunStats() ([]MonthlyFunStat, error) {
	var rows []MonthlyFunStat
	err := db.Select(&rows, `
		WITH first_at AS (
			SELECT COALESCE(
				(
					SELECT MIN(created_at) FROM (
						SELECT created_at FROM users
						UNION ALL
						SELECT created_at FROM ads
					) t
				),
				CURRENT_TIMESTAMP
			) AS created_at
		),
		months AS (
			SELECT generate_series(
				date_trunc('month', created_at),
				date_trunc('month', CURRENT_TIMESTAMP),
				interval '1 month'
			) AS month
			FROM first_at
		),
		asof AS (
			SELECT
				month,
				LEAST(
					month + interval '1 month',
					CURRENT_TIMESTAMP
				) AS as_of
			FROM months
		)
		SELECT
			a.month,
			(
				SELECT COUNT(*)::int FROM users u
				WHERE u.created_at <= a.as_of
				  AND (
					u.deleted_at IS NULL
					OR u.deleted_at > a.as_of
				  )
			) AS registered_users,
			(
				SELECT COUNT(DISTINCT ad.user_id)::int FROM ads ad
				WHERE ad.created_at <= a.as_of
				  AND (
					ad.inactive_at IS NULL
					OR ad.inactive_at > a.as_of
				  )
				  AND (
					ad.deleted_at IS NULL
					OR ad.deleted_at > a.as_of
				  )
			) AS users_with_active_ads,
			(
				SELECT COUNT(*)::int FROM ads ad
				WHERE ad.created_at <= a.as_of
				  AND (
					ad.inactive_at IS NULL
					OR ad.inactive_at > a.as_of
				  )
				  AND (
					ad.deleted_at IS NULL
					OR ad.deleted_at > a.as_of
				  )
			) AS active_ads
		FROM asof a
		ORDER BY a.month`)
	return rows, err
}
