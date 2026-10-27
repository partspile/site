package vector

import (
	"fmt"
	"math"

	"github.com/partspile/site/internal/db"
)

func getSiteActivities(categoryID int, _ string, limit int) ([]AdActivity, error) {
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
			SELECT ad_id,
				MAX(timestamp) AS timestamp,
				(ARRAY_AGG(embedding))[1] AS embedding,
				(ARRAY_AGG(activity_type ORDER BY weight DESC))[1]
					AS activity_type,
				SUM(weight) AS weight
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
						WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL
						  AND a.category_id = $1
						  AND a.embedding IS NOT NULL

						UNION ALL

						SELECT uac.ad_id, uac.last_clicked_at,
							a.embedding, 'ad_click' AS activity_type
						FROM user_ad_clicks uac
						JOIN ads a ON uac.ad_id = a.id
						WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL
						  AND a.category_id = $1
						  AND a.embedding IS NOT NULL

						UNION ALL

						SELECT uaic.ad_id, uaic.last_clicked_at,
							a.embedding, 'image_click' AS activity_type
						FROM user_ad_image_clicks uaic
						JOIN ads a ON uaic.ad_id = a.id
						WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL
						  AND a.category_id = $1
						  AND a.embedding IS NOT NULL
					) combined
				) with_age
			) events
			GROUP BY ad_id
			ORDER BY weight DESC
			LIMIT $2
		) top_ads`
	var activities []AdActivity
	if err := db.QueryJSON(&activities, query, categoryID, limit); err != nil {
		return nil, fmt.Errorf("site activities: %w", err)
	}
	for i := range activities {
		activities[i].Weight = dampSiteWeight(activities[i].Weight)
	}
	return activities, nil
}

// log1p so many users clicking the current #1 cannot grow that
// ad's weight without bound.
func dampSiteWeight(sum float32) float32 {
	if sum <= 0 {
		return 0
	}
	return float32(math.Log1p(float64(sum)))
}
