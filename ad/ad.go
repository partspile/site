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

// Ad represents the minimal advertisement data needed for list/grid views
type Ad struct {
	// Core database fields
	ID           int        `json:"id" db:"id"`
	AdCategoryID int        `json:"ad_category_id" db:"ad_category_id"`
	Title        string     `json:"title" db:"title"`
	Price        float64    `json:"price" db:"price"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	UserID       int        `json:"user_id" db:"user_id"`
	ImageCount   int        `json:"image_count" db:"image_count"`

	// Location fields from joins
	City      sql.NullString `json:"city,omitempty" db:"city"`
	AdminArea sql.NullString `json:"admin_area,omitempty" db:"admin_area"`
	Country   sql.NullString `json:"country,omitempty" db:"country"`

	// User-specific computed fields
	Bookmarked bool `json:"bookmarked" db:"is_bookmarked"`
}

// AdDetail extends Ad with additional fields needed for detail view
type AdDetail struct {
	Ad // Embed the minimal Ad struct

	// Additional fields for detail view
	Description       string         `json:"description" db:"description"`
	PartSubcategoryID int            `json:"part_subcategory_id" db:"part_subcategory_id"`
	LocationID        int            `json:"location_id" db:"location_id"`
	HasVector         bool           `json:"has_vector" db:"has_vector"`
	RawLocation       sql.NullString `json:"raw_location,omitempty" db:"raw_location"`
	PartSubcategory   sql.NullString `json:"part_subcategory,omitempty" db:"part_subcategory"`
	PartCategory      sql.NullString `json:"part_category,omitempty" db:"part_category"`

	// Vehicle compatibility fields from vehicle joins
	Make    string   `json:"make" db:"make"`
	Years   []string `json:"years" db:"years"`
	Models  []string `json:"models" db:"models"`
	Engines []string `json:"engines" db:"engines"`
}

// GetAdCategory returns the AdCategory string for this ad
func (a Ad) GetAdCategory() string {
	return GetAdCategoryFromID(a.AdCategoryID)
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

	log.Printf("[GetAdsByIDs] Returning %d ads", len(ads))
	return ads, nil
}

// buildAdQueryWithDeleted builds the minimal query for fetching ads with IDs and user context, optionally including deleted ads
func buildAdQueryWithDeleted(ids []int, currentUser *user.User, includeDeleted bool) (string, []interface{}) {
	var query string
	var args []interface{}

	if currentUser != nil {
		// Query with bookmark status - minimal fields only
		query = `
			SELECT a.id, a.ad_category_id, a.title, a.price, a.created_at, a.deleted_at, a.user_id, a.image_count,
			       l.city, l.admin_area, l.country,
			       CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked
			FROM Ad a
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
		`
		args = append(args, currentUser.ID)
	} else {
		// Query without bookmark status (default to false) - minimal fields only
		query = `
			SELECT a.id, a.ad_category_id, a.title, a.price, a.created_at, a.deleted_at, a.user_id, a.image_count,
			       l.city, l.admin_area, l.country,
			       0 as is_bookmarked
			FROM Ad a
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

	// Add ORDER BY to preserve the input order using CASE statement
	query += " ORDER BY CASE a.id"
	for i, id := range ids {
		query += fmt.Sprintf(" WHEN %d THEN %d", id, i)
	}
	query += " END"

	for _, id := range ids {
		args = append(args, id)
	}

	return query, args
}

// GetAdByID returns a single ad by ID
func GetAdByID(adID int, currentUser *user.User) (*Ad, error) {
	ads, err := GetAdsByIDs([]int{adID}, currentUser)
	if err != nil {
		return nil, err
	}
	if len(ads) == 0 {
		return nil, fmt.Errorf("ad with ID %d not found", adID)
	}
	return &ads[0], nil
}

// GetAdDetailByID returns the full AdDetail for a single ad ID
func GetAdDetailByID(adID int, currentUser *user.User) (*AdDetail, error) {
	var query string
	var args []interface{}

	if currentUser != nil {
		// Query with bookmark status - full fields for detail view
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.deleted_at, a.part_subcategory_id,
			       a.user_id, psc.name as part_subcategory, pc.name as part_category, a.location_id, a.image_count, a.has_vector,
			       l.raw_text as raw_location, l.city, l.admin_area, l.country,
			       CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked, a.ad_category_id
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.part_subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.part_category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
			WHERE a.id = ?
		`
		args = append(args, currentUser.ID, adID)
	} else {
		// Query without bookmark status (default to false) - full fields for detail view
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.deleted_at, a.part_subcategory_id,
			       a.user_id, psc.name as part_subcategory, pc.name as part_category, a.location_id, a.image_count, a.has_vector,
			       l.raw_text as raw_location, l.city, l.admin_area, l.country,
			       0 as is_bookmarked, a.ad_category_id
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.part_subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.part_category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
			WHERE a.id = ?
		`
		args = append(args, adID)
	}

	var adDetails []AdDetail
	err := db.Select(&adDetails, query, args...)
	if err != nil {
		return nil, err
	}

	if len(adDetails) == 0 {
		return nil, fmt.Errorf("ad with ID %d not found", adID)
	}

	return &adDetails[0], nil
}

// GetMostPopularAds returns the top n ads by popularity using SQL
func GetMostPopularAds(n int) []Ad {
	log.Printf("[GetMostPopularAds] Querying for top %d popular ads", n)
	query := `
		SELECT 
			a.id, a.ad_category_id, a.title, a.price, a.created_at, a.deleted_at, a.user_id, a.image_count,
			l.city, l.admin_area, l.country,
			0 as is_bookmarked
		FROM Ad a
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
