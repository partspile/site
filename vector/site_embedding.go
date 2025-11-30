package vector

import (
	"fmt"

	"github.com/parts-pile/site/db"
)

// getSiteActivities fetches combined bookmarks, clicks, and image clicks for all users
// in a specific category, calculates weights using exponential decay, and returns top N by weight
func getSiteActivities(categoryID int, campaignKey string, limit int) ([]AdActivity, error) {
	var args []interface{}
	args = append(args, categoryID)
	categoryFilter := "a.deleted_at IS NULL AND a.ad_category_id = $1"

	args = append(args, limit)
	limitPlaceholder := "$2"

	query := fmt.Sprintf(`
		SELECT COALESCE(
			json_agg(
				json_build_object(
					'ad_id', ad_id,
					'timestamp', timestamp AT TIME ZONE 'UTC',
					'embedding', array_to_json((embedding::vector)::real[]),
					'activity_type', activity_type,
					'weight', weight
				)
				ORDER BY weight DESC
			),
			'[]'::json
		) AS activities
		FROM (
			SELECT 
				ad_id,
				timestamp,
				embedding,
				activity_type,
				-- Calculate weight: baseWeight * EXP(-decayRate * ageDays)
				-- decayRate = LN(2) / halfLifeDays
				-- Cast to DOUBLE PRECISION for CockroachDB compatibility
				CASE activity_type
					WHEN 'bookmark' THEN 
						CAST(1.0 AS DOUBLE PRECISION) * EXP(-(LN(CAST(2.0 AS DOUBLE PRECISION)) / CAST(45.0 AS DOUBLE PRECISION)) * age_days)
					WHEN 'ad_click' THEN 
						CAST(0.7 AS DOUBLE PRECISION) * EXP(-(LN(CAST(2.0 AS DOUBLE PRECISION)) / CAST(30.0 AS DOUBLE PRECISION)) * age_days)
					WHEN 'image_click' THEN 
						CAST(0.4 AS DOUBLE PRECISION) * EXP(-(LN(CAST(2.0 AS DOUBLE PRECISION)) / CAST(20.0 AS DOUBLE PRECISION)) * age_days)
				END AS weight
			FROM (
				SELECT 
					ad_id,
					timestamp,
					embedding,
					activity_type,
					-- Calculate age in days (cast to DOUBLE PRECISION for CockroachDB compatibility)
					CAST(EXTRACT(EPOCH FROM (NOW() - timestamp)) AS DOUBLE PRECISION) / 86400.0 AS age_days
				FROM (
					SELECT 
						ba.ad_id, 
						ba.bookmarked_at AS timestamp, 
						a.embedding, 
						'bookmark' AS activity_type
					FROM BookmarkedAd ba
					JOIN Ad a ON ba.ad_id = a.id
					WHERE %s AND a.embedding IS NOT NULL
					
					UNION ALL
					
					SELECT 
						uac.ad_id, 
						uac.last_clicked_at AS timestamp, 
						a.embedding, 
						'ad_click' AS activity_type
					FROM UserAdClick uac
					JOIN Ad a ON uac.ad_id = a.id
					WHERE %s AND a.embedding IS NOT NULL
					
					UNION ALL
					
					SELECT 
						uaic.ad_id, 
						uaic.last_clicked_at AS timestamp, 
						a.embedding, 
						'image_click' AS activity_type
					FROM UserAdImageClick uaic
					JOIN Ad a ON uaic.ad_id = a.id
					WHERE %s AND a.embedding IS NOT NULL
				) combined
			) with_age
			ORDER BY weight DESC
			LIMIT %s
		) top_weighted
	`, categoryFilter, categoryFilter, categoryFilter, limitPlaceholder)

	var activities []AdActivity
	if err := db.QueryJSON(&activities, query, args...); err != nil {
		return nil, err
	}

	return activities, nil
}
