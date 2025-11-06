package ad

import (
	"strings"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// GetAdsWithoutVectors returns ads that don't have vector embeddings
func GetAdsWithoutVectors() ([]Ad, error) {
	logger.Debug("Querying for ads without vectors",
		"component", "ad")
	query := `
		SELECT 
			a.id, a.ad_category_id, a.title, a.price, a.created_at, a.deleted_at, a.user_id, a.image_count,
			l.city, l.admin_area, l.country,
			0 as is_bookmarked
		FROM Ad a
		LEFT JOIN Location l ON a.location_id = l.id
		WHERE a.has_vector = 0 AND a.deleted_at IS NULL
	`

	var ads []Ad
	err := db.Select(&ads, query)
	if err != nil {
		logger.Error("SQL error querying ads without vectors",
			"error", err, "component", "ad")
		return nil, err
	}

	logger.Debug("Found ads without vectors from SQL query",
		"component", "ad", "count", len(ads))
	return ads, nil
}

// MarkAdAsHavingVector marks an ad as having a vector embedding
func MarkAdAsHavingVector(adID int) error {
	return MarkAdsAsHavingVector([]int{adID})
}

// MarkAdsAsHavingVector marks multiple ads as having vector embeddings in a single SQL call
func MarkAdsAsHavingVector(adIDs []int) error {
	if len(adIDs) == 0 {
		return nil
	}

	// Build the SQL query with placeholders for all ad IDs
	placeholders := make([]string, len(adIDs))
	args := make([]interface{}, len(adIDs))
	for i, adID := range adIDs {
		placeholders[i] = "?"
		args[i] = adID
	}

	query := "UPDATE Ad SET has_vector = 1 WHERE id IN (" + strings.Join(placeholders, ",") + ")"

	_, err := db.Exec(query, args...)
	if err != nil {
		logger.Error("Failed to mark ads as having vector",
			"error", err, "component", "ad", "count", len(adIDs))
		return err
	}
	logger.Debug("Successfully marked ads as having vector",
		"component", "ad", "count", len(adIDs))
	return nil
}
