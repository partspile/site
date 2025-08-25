package ad

import (
	"log"

	"github.com/parts-pile/site/db"
)

// GetAdsWithoutVectors returns ads that don't have vector embeddings
func GetAdsWithoutVectors() ([]Ad, error) {
	log.Printf("[GetAdsWithoutVectors] Querying for ads without vectors")
	query := `
		SELECT 
			a.id, a.title, a.description, a.price, a.created_at, 
			a.subcategory_id, a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_order,
			l.city, l.admin_area, l.country, l.latitude, l.longitude,
			0 as is_bookmarked
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		LEFT JOIN Location l ON a.location_id = l.id
		WHERE a.has_vector = 0 AND a.deleted_at IS NULL
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[GetAdsWithoutVectors] SQL error: %v", err)
		return nil, err
	}
	defer rows.Close()

	ads, err := scanAdRows(rows)
	if err != nil {
		log.Printf("[GetAdsWithoutVectors] Row scan error: %v", err)
		return nil, err
	}

	// Set has_vector to false for all ads (since we're querying for ads without vectors)
	for i := range ads {
		ads[i].HasVector = false
		// Get vehicle data
		ads[i].Make, ads[i].Years, ads[i].Models, ads[i].Engines = GetVehicleData(ads[i].ID)
	}

	log.Printf("[GetAdsWithoutVectors] Found %d ads without vectors from SQL query", len(ads))
	return ads, nil
}

// MarkAdAsHavingVector marks an ad as having a vector embedding
func MarkAdAsHavingVector(adID int) error {
	_, err := db.Exec("UPDATE Ad SET has_vector = 1 WHERE id = ?", adID)
	if err != nil {
		log.Printf("[MarkAdAsHavingVector] Failed to mark ad %d as having vector: %v", adID, err)
		return err
	}
	log.Printf("[MarkAdAsHavingVector] Successfully marked ad %d as having vector", adID)
	return nil
}
