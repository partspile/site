package vector

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// AdActivity represents a user's interaction with an ad (bookmark, click, image click, or ad created)
type AdActivity struct {
	AdID         int       `json:"ad_id"`
	Timestamp    time.Time `json:"timestamp"`
	Embedding    []float32 `json:"embedding"`     // nil if embedding is missing
	ActivityType string    `json:"activity_type"` // bookmark, ad_click, image_click, ad_created
	Weight       float32   `json:"weight"`        // Pre-calculated weight from SQL
}

// GetUserPersonalizedEmbedding loads from cache first, then generates new embedding if not found.
// categoryID must be a valid category ID (greater than 0)
func GetUserPersonalizedEmbedding(userID int, categoryID int, forceRecompute bool) ([]float32, error) {

	logger.Debug("GetUserPersonalizedEmbedding called",
		"userID", userID, "categoryID", categoryID, "forceRecompute", forceRecompute)

	if !forceRecompute {
		// Try cache first
		cached, err := GetUserEmbedding(userID, categoryID)
		if err == nil && cached != nil {
			logger.Debug("Cache hit for user category embedding",
				"userID", userID, "categoryID", categoryID)
			return cached, nil
		}

		// Cache miss, will generate new embedding below
		logger.Debug("Cache miss for user category embedding, will generate new",
			"userID", userID, "categoryID", categoryID)
	}

	logger.Debug("Calculating personalized user embedding",
		"userID", userID, "categoryID", categoryID)

	// Fetch combined bookmarks and clicks, sorted by timestamp (newest first), limited
	limit := config.VectorUserEmbeddingLimit
	activities, err := getUserActivities(userID, categoryID, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch user activities: %w", err)
	}

	if len(activities) == 0 {
		return nil, fmt.Errorf("no user activity to aggregate")
	}

	logger.Debug("User activities",
		"userID", userID, "categoryID", categoryID, "count", len(activities))

	vectors, weights := calculateWeightedVectors(activities)

	logger.Debug("Vectors aggregated",
		"userID", userID, "categoryID", categoryID, "vectors", len(vectors),
		"weights", len(weights))

	if len(vectors) == 0 {
		logger.Debug("No vectors could be aggregated from user activity",
			"userID", userID, "categoryID", categoryID)
		return nil, fmt.Errorf("no user activity to aggregate")
	}

	emb := AggregateEmbeddings(vectors, weights)

	// Cache the result for future use
	if err := SetUserEmbedding(userID, categoryID, emb); err != nil {
		logger.Warn("Failed to cache user category embedding",
			"error", err, "userID", userID, "categoryID", categoryID)
	}

	logger.Info("Successfully created user category embedding",
		"userID", userID, "categoryID", categoryID, "vectors", len(vectors),
		"activities", len(activities))
	return emb, nil
}

// getUserActivities fetches combined bookmarks, clicks, image clicks, and ads created for a user
// in a specific category, calculates weights using exponential decay, and returns top N by weight
func getUserActivities(userID int, categoryID int, limit int) ([]AdActivity, error) {
	query := `
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
					WHEN 'ad_created' THEN 
						CAST(0.8 AS DOUBLE PRECISION) * EXP(-(LN(CAST(2.0 AS DOUBLE PRECISION)) / CAST(60.0 AS DOUBLE PRECISION)) * age_days)
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
					WHERE ba.user_id = $1 
					  AND a.deleted_at IS NULL 
					  AND a.ad_category_id = $2 
					  AND a.embedding IS NOT NULL
					
					UNION ALL
					
					SELECT 
						uac.ad_id, 
						uac.last_clicked_at AS timestamp, 
						a.embedding, 
						'ad_click' AS activity_type
					FROM UserAdClick uac
					JOIN Ad a ON uac.ad_id = a.id
					WHERE uac.user_id = $1 
					  AND a.deleted_at IS NULL 
					  AND a.ad_category_id = $2 
					  AND a.embedding IS NOT NULL
					
					UNION ALL
					
					SELECT 
						uaic.ad_id, 
						uaic.last_clicked_at AS timestamp, 
						a.embedding, 
						'image_click' AS activity_type
					FROM UserAdImageClick uaic
					JOIN Ad a ON uaic.ad_id = a.id
					WHERE uaic.user_id = $1 
					  AND a.deleted_at IS NULL 
					  AND a.ad_category_id = $2 
					  AND a.embedding IS NOT NULL
					
					UNION ALL
					
					SELECT 
						a.id AS ad_id, 
						a.created_at AS timestamp, 
						a.embedding, 
						'ad_created' AS activity_type
					FROM Ad a
					WHERE a.user_id = $1 
					  AND a.deleted_at IS NULL 
					  AND a.ad_category_id = $2 
					  AND a.embedding IS NOT NULL
				) combined
			) with_age
			ORDER BY weight DESC
			LIMIT $3
		) top_weighted
	`

	var activities []AdActivity
	if err := db.QueryJSON(&activities, query, userID, categoryID, limit); err != nil {
		return nil, err
	}

	return activities, nil
}
