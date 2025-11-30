package vector

import (
	"fmt"

	"github.com/parts-pile/site/db"
)

// getSiteActivities fetches combined bookmarks and clicks for all users in a specific category,
// sorted by timestamp (newest first) and limited to the most recent N items
// Also fetches embeddings directly in the same query using JSON SQL
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
					'embedding', array_to_json((embedding::vector)::real[])
				)
			),
			'[]'::json
		) AS activities
		FROM (
			SELECT ad_id, timestamp, embedding
			FROM (
				SELECT ba.ad_id, ba.bookmarked_at AS timestamp, a.embedding
				FROM BookmarkedAd ba
				JOIN Ad a ON ba.ad_id = a.id
				WHERE %s AND a.embedding IS NOT NULL
				
				UNION ALL
				
				SELECT uac.ad_id, uac.last_clicked_at AS timestamp, a.embedding
				FROM UserAdClick uac
				JOIN Ad a ON uac.ad_id = a.id
				WHERE %s AND a.embedding IS NOT NULL
			) combined
			ORDER BY timestamp DESC
			LIMIT %s
		) limited
	`, categoryFilter, categoryFilter, limitPlaceholder)

	var activities []AdActivity
	if err := db.QueryJSON(&activities, query, args...); err != nil {
		return nil, err
	}

	return activities, nil
}
