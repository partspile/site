package vector

import (
	"fmt"

	"github.com/partspile/site/internal/db"
)

// Buyer signals only: listings the user posted would pull browse
// ranking toward their own inventory.
func getUserActivities(userID, categoryID, limit int) ([]AdActivity, error) {
	query := `
		SELECT COALESCE(
			json_agg(
				json_build_object(
					'ad_id', ad_id,
					'timestamp', timestamp,
					'embedding', array_to_json((embedding::vector)::real[]),
					'activity_type', activity_type,
					'weight', weight
				)
				ORDER BY weight DESC
			),
			'[]'::json
		)
		FROM (
			SELECT ad_id, timestamp, embedding, activity_type,
				CASE activity_type
					WHEN 'bookmark' THEN
						1.0 * EXP(-(LN(2.0) / 45.0) * age_days)
					WHEN 'ad_click' THEN
						0.7 * EXP(-(LN(2.0) / 30.0) * age_days)
					WHEN 'image_click' THEN
						0.4 * EXP(-(LN(2.0) / 20.0) * age_days)
				END AS weight
			FROM (
				SELECT ad_id, timestamp, embedding, activity_type,
					EXTRACT(EPOCH FROM (NOW() - timestamp)) / 86400.0
						AS age_days
				FROM (
					SELECT b.ad_id, b.bookmarked_at AS timestamp,
						a.embedding, 'bookmark' AS activity_type
					FROM bookmarks b
					JOIN ads a ON b.ad_id = a.id
					WHERE b.user_id = $1
					  AND a.inactive_at IS NULL AND a.deleted_at IS NULL
					  AND a.category_id = $2
					  AND a.embedding IS NOT NULL

					UNION ALL

					SELECT uac.ad_id, uac.last_clicked_at,
						a.embedding, 'ad_click' AS activity_type
					FROM user_ad_clicks uac
					JOIN ads a ON uac.ad_id = a.id
					WHERE uac.user_id = $1
					  AND a.inactive_at IS NULL AND a.deleted_at IS NULL
					  AND a.category_id = $2
					  AND a.embedding IS NOT NULL

					UNION ALL

					SELECT uaic.ad_id, uaic.last_clicked_at,
						a.embedding, 'image_click' AS activity_type
					FROM user_ad_image_clicks uaic
					JOIN ads a ON uaic.ad_id = a.id
					WHERE uaic.user_id = $1
					  AND a.inactive_at IS NULL AND a.deleted_at IS NULL
					  AND a.category_id = $2
					AND a.embedding IS NOT NULL
				) combined
			) with_age
			ORDER BY weight DESC
			LIMIT $3
		) top_weighted`
	var activities []AdActivity
	if err := db.QueryJSON(&activities, query, userID, categoryID, limit); err != nil {
		return nil, fmt.Errorf("user activities: %w", err)
	}
	return activities, nil
}
