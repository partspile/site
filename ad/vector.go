package ad

import (
	"log"
	"strings"

	"github.com/parts-pile/site/db"
)

// GetAdsWithoutVectors returns ads that don't have vector embeddings
func GetAdsWithoutVectors() ([]Ad, error) {
	log.Printf("[GetAdsWithoutVectors] Querying for ads without vectors")
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
		log.Printf("[GetAdsWithoutVectors] SQL error: %v", err)
		return nil, err
	}

	log.Printf("[GetAdsWithoutVectors] Found %d ads without vectors from SQL query", len(ads))
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
		log.Printf("[MarkAdsAsHavingVector] Failed to mark %d ads as having vector: %v", len(adIDs), err)
		return err
	}
	log.Printf("[MarkAdsAsHavingVector] Successfully marked %d ads as having vector", len(adIDs))
	return nil
}
