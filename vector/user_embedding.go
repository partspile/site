package vector

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// AdActivity represents a user's interaction with an ad (bookmark or click)
type AdActivity struct {
	AdID      int       `json:"ad_id"`
	Timestamp time.Time `json:"timestamp"`
	Embedding []float32 `json:"embedding"` // nil if embedding is missing
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

// getUserActivities fetches combined bookmarks and clicks for a user in a specific category,
// sorted by timestamp (newest first) and limited to the most recent N items
// Also fetches embeddings directly in the same query using JSON SQL
func getUserActivities(userID int, categoryID int, limit int) ([]AdActivity, error) {
	query := `
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
				WHERE ba.user_id = $1 AND a.deleted_at IS NULL AND a.ad_category_id = $2 AND a.embedding IS NOT NULL
				
				UNION ALL
				
				SELECT uac.ad_id, uac.last_clicked_at AS timestamp, a.embedding
				FROM UserAdClick uac
				JOIN Ad a ON uac.ad_id = a.id
				WHERE uac.user_id = $1 AND a.deleted_at IS NULL AND a.ad_category_id = $2 AND a.embedding IS NOT NULL
			) combined
			ORDER BY timestamp DESC
			LIMIT $3
		) limited
	`

	var activities []AdActivity
	if err := db.QueryJSON(&activities, query, userID, categoryID, limit); err != nil {
		return nil, err
	}

	return activities, nil
}
