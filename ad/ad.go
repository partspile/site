package ad

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/user"
)

// Ad represents an advertisement in the system
type Ad struct {
	// Core database fields
	ID                int        `json:"id" db:"id"`
	AdCategoryID      int        `json:"ad_category_id" db:"ad_category_id"`
	Title             string     `json:"title" db:"title"`
	Description       string     `json:"description" db:"description"`
	Price             float64    `json:"price" db:"price"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	PartSubcategoryID int        `json:"part_subcategory_id" db:"part_subcategory_id"`
	UserID            int        `json:"user_id" db:"user_id"`
	ImageCount        int        `json:"image_count" db:"image_count"`
	LocationID        int        `json:"location_id" db:"location_id"`
	ClickCount        int        `json:"click_count" db:"click_count"`
	LastClickedAt     *time.Time `json:"last_clicked_at,omitempty" db:"last_clicked_at"`
	HasVector         bool       `json:"has_vector" db:"has_vector"`

	// Computed/derived fields from joins
	RawLocation     sql.NullString  `json:"raw_location,omitempty" db:"raw_location"`
	City            sql.NullString  `json:"city,omitempty" db:"city"`
	AdminArea       sql.NullString  `json:"admin_area,omitempty" db:"admin_area"`
	Country         sql.NullString  `json:"country,omitempty" db:"country"`
	PartSubcategory sql.NullString  `json:"part_subcategory,omitempty" db:"part_subcategory"`
	PartCategory    sql.NullString  `json:"part_category,omitempty" db:"part_category"`
	Latitude        sql.NullFloat64 `json:"latitude,omitempty" db:"latitude"`
	Longitude       sql.NullFloat64 `json:"longitude,omitempty" db:"longitude"`

	// Vehicle compatibility fields from vehicle joins
	Make    string   `json:"make" db:"make"`
	Years   []string `json:"years" db:"years"`
	Models  []string `json:"models" db:"models"`
	Engines []string `json:"engines" db:"engines"`

	// User-specific computed fields
	Bookmarked bool `json:"bookmarked" db:"is_bookmarked"`
}

// GetCategory returns the AdCategory for this ad
func (a Ad) GetCategory() AdCategory {
	return AdCategory(a.AdCategoryID)
}

// IsArchived returns true if the ad is archived (deleted)
func (a Ad) IsArchived() bool {
	return a.DeletedAt != nil
}

// GetAdsByIDs returns ads for a list of IDs
func GetAdsByIDs(ids []int, currentUser *user.User) ([]Ad, error) {
	return GetAdsByIDsWithDeleted(ids, currentUser, false)
}

// GetAdsByIDsWithDeleted returns ads for a list of IDs, optionally including deleted ads
func GetAdsByIDsWithDeleted(ids []int, currentUser *user.User, includeDeleted bool) ([]Ad, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args := buildAdQueryWithDeleted(ids, currentUser, includeDeleted)

	var ads []Ad
	err := db.Select(&ads, query, args...)
	if err != nil {
		return nil, err
	}

	// For single ID, just return the first result directly
	if len(ids) == 1 {
		return ads, nil
	}

	// For multiple IDs, create a map for quick lookup and preserve order
	adMap := make(map[int]Ad)
	for _, ad := range ads {
		adMap[ad.ID] = ad
	}

	// Preserve order of ids
	result := make([]Ad, 0, len(ids))
	for _, id := range ids {
		if ad, ok := adMap[id]; ok {
			result = append(result, ad)
		}
	}

	log.Printf("[GetAdsByIDs] Returning ads in order: %v", func() []int {
		debugResult := make([]int, len(result))
		for i, ad := range result {
			debugResult[i] = ad.ID
		}
		return debugResult
	}())
	return result, nil
}

// buildAdQueryWithDeleted builds the complete query for fetching ads with IDs and user context, optionally including deleted ads
func buildAdQueryWithDeleted(ids []int, currentUser *user.User, includeDeleted bool) (string, []interface{}) {
	var query string
	var args []interface{}

	if currentUser != nil {
		// Query with bookmark status
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.deleted_at, a.part_subcategory_id,
			       a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_count,
			       l.raw_text as raw_location, l.city, l.admin_area, l.country, l.latitude, l.longitude,
			       CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked, a.category_id
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.part_subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.part_category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
		`
		args = append(args, currentUser.ID)
	} else {
		// Query without bookmark status (default to false)
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.deleted_at, a.part_subcategory_id,
			       a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_count,
			       l.raw_text as raw_location, l.city, l.admin_area, l.country, l.latitude, l.longitude,
			       0 as is_bookmarked, a.category_id
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.part_subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.part_category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
		`
	}

	if !includeDeleted {
		query += " WHERE a.deleted_at IS NULL"
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}
	if includeDeleted {
		query += " WHERE a.id IN (" + strings.Join(placeholders, ",") + ")"
	} else {
		query += " AND a.id IN (" + strings.Join(placeholders, ",") + ")"
	}
	for _, id := range ids {
		args = append(args, id)
	}

	return query, args
}

// GetAdCategoryIDFromID returns the category for a given ad ID
func GetAdCategoryIDFromID(adID int) (AdCategory, error) {
	var categoryID int
	err := db.QueryRow("SELECT ad_category_id FROM Ad WHERE id = ?", adID).Scan(&categoryID)
	if err != nil {
		return CarParts, err
	}
	return AdCategory(categoryID), nil
}

// GetMostPopularAds returns the top n ads by popularity using SQL
func GetMostPopularAds(n int) []Ad {
	log.Printf("[GetMostPopularAds] Querying for top %d popular ads", n)
	query := `
		SELECT 
			a.id, a.title, a.description, a.price, a.created_at, 
			a.part_subcategory_id, a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_count,
			l.raw_text as raw_location, l.city, l.admin_area, l.country, l.latitude, l.longitude,
			0 as is_bookmarked, a.ad_category_id
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.part_subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.part_category_id = pc.id
		LEFT JOIN Location l ON a.location_id = l.id
		WHERE a.deleted_at IS NULL
		ORDER BY (
			a.click_count * 2 + 
			COALESCE((SELECT COUNT(*) FROM BookmarkedAd ba WHERE ba.ad_id = a.id), 0) * 3 + 
			100.0 / (julianday('now') - julianday(a.created_at))
		) DESC
		LIMIT ?
	`

	var ads []Ad
	err := db.Select(&ads, query, n)
	if err != nil {
		log.Printf("[GetMostPopularAds] SQL error: %v", err)
		return nil
	}

	log.Printf("[GetMostPopularAds] Found %d ads from SQL query", len(ads))
	return ads
}

// AddAd creates a new ad in the database and returns the ad ID
func AddAd(adObj Ad) (int, error) {
	// This is a placeholder implementation
	// In a real implementation, this would insert the ad into the database
	// and return the generated ID
	return 0, fmt.Errorf("AddAd not implemented yet")
}

// ArchiveAd archives an ad by setting its deleted_at timestamp
func ArchiveAd(adID int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = CURRENT_TIMESTAMP WHERE id = ?", adID)
	return err
}

// RestoreAd restores an archived ad by clearing its deleted_at timestamp
func RestoreAd(adID int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = NULL WHERE id = ?", adID)
	return err
}

// GetUserActiveAdIDs returns the IDs of active ads for a user
func GetUserActiveAdIDs(userID int) ([]int, error) {
	var adIDs []int
	err := db.Select(&adIDs, "SELECT id FROM Ad WHERE user_id = ? AND deleted_at IS NULL ORDER BY created_at DESC", userID)
	return adIDs, err
}

// GetUserDeletedAdIDs returns the IDs of deleted ads for a user
func GetUserDeletedAdIDs(userID int) ([]int, error) {
	var adIDs []int
	err := db.Select(&adIDs, "SELECT id FROM Ad WHERE user_id = ? AND deleted_at IS NOT NULL ORDER BY deleted_at DESC", userID)
	return adIDs, err
}

// ArchiveAdsByUserID archives all ads for a user
func ArchiveAdsByUserID(userID int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ? AND deleted_at IS NULL", userID)
	return err
}

// GetAdsForAll returns all ads (placeholder implementation)
func GetAdsForAll() ([]Ad, error) {
	// This is a placeholder implementation
	return nil, fmt.Errorf("GetAdsForAll not implemented yet")
}

// GetAdsForAdIDs returns ads for the given IDs (placeholder implementation)
func GetAdsForAdIDs(adIDs []int) ([]Ad, error) {
	// This is a placeholder implementation
	return nil, fmt.Errorf("GetAdsForAdIDs not implemented yet")
}
