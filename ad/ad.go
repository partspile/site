package ad

import (
	"fmt"
	"strings"
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/spec"
)

// Ad represents the minimal advertisement data needed for list/grid/tree views
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
	City      string `json:"city" db:"city"`
	AdminArea string `json:"admin_area" db:"admin_area"`
	Country   string `json:"country" db:"country"`

	// User-specific computed fields
	Bookmarked bool `json:"bookmarked" db:"is_bookmarked"`
}

// AdDetail extends Ad with additional fields needed for detail view
type AdDetail struct {
	Ad // Embed the minimal Ad struct

	// Additional fields for detail view
	Description string `json:"description" db:"description"`
	LocationID  int    `json:"location_id" db:"location_id"`
	HasVector   bool   `json:"has_vector" db:"has_vector"`
	RawLocation string `json:"raw_location" db:"raw_location"`

	// Spec data from spec joins - map of field name to array of values
	// Single-value fields have arrays with one element, multi-value fields
	// have multiple elements
	SpecData map[string][]string `json:"spec_data"`

	// Computed fields
	OwnerUnreachable int `json:"owner_unreachable" db:"owner_unreachable"`
}

// IsArchived returns true if the ad is archived (deleted)
func (a Ad) IsArchived() bool {
	return a.DeletedAt != nil
}

// populateSpecData fills in the spec data for the ad dynamically based on category fields
func populateSpecData(adDetail *AdDetail) error {
	// Initialize SpecData map
	adDetail.SpecData = make(map[string][]string)

	// Get category fields to determine which fields to query
	fieldConfigs := spec.GetCategoryFields(adDetail.AdCategoryID)

	// Build dynamic SELECT clause for all fields
	var selectParts []string
	var joins []string
	var fieldNames []string // Column names (same as field names)

	for _, fieldConfig := range fieldConfigs {
		tableName := fieldConfig.TableName
		tableFieldName := fieldConfig.TableFieldName
		fieldName := fieldConfig.Name

		// Add JOIN for this field's table
		joins = append(joins, fmt.Sprintf("LEFT JOIN %s ON spec.%s = %s.id",
			tableName, tableFieldName, tableName))

		// Build SELECT clause based on field type
		if fieldConfig.FieldType == "int" {
			// For integer fields (like year), use GROUP_CONCAT with CAST
			valueCol := fieldConfig.ValueColumnName
			selectParts = append(selectParts,
				fmt.Sprintf("COALESCE(GROUP_CONCAT(DISTINCT CAST(%s.%s AS TEXT)), '') as %s",
					tableName, valueCol, fieldName))
		} else {
			// For string fields, use GROUP_CONCAT for both single and multi-value
			selectParts = append(selectParts,
				fmt.Sprintf("COALESCE(GROUP_CONCAT(DISTINCT %s.name), '') as %s",
					tableName, fieldName))
		}
		fieldNames = append(fieldNames, fieldName)
	}

	// Build final query
	query := fmt.Sprintf(`
		SELECT %s
		FROM AdSpecification ad_spec
		JOIN Specification spec ON ad_spec.specification_id = spec.id
		%s
		WHERE ad_spec.ad_id = ?
	`, strings.Join(selectParts, ", "), strings.Join(joins, " "))

	// Execute query - use a map to store results dynamically
	rows, err := db.Query(query, adDetail.ID)
	if err != nil {
		return fmt.Errorf("failed to query spec data: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil // No spec data is okay
	}

	// Scan row into values
	values := make([]interface{}, len(fieldNames))
	valuePtrs := make([]interface{}, len(fieldNames))
	for i := range values {
		valuePtrs[i] = &values[i]
	}
	if err := rows.Scan(valuePtrs...); err != nil {
		return fmt.Errorf("failed to scan row: %w", err)
	}

	// Map column values to field names
	for i, fieldName := range fieldNames {
		if values[i] == nil {
			continue
		}

		valueStr := fmt.Sprintf("%v", values[i])
		if valueStr == "" {
			continue
		}

		adDetail.SpecData[fieldName] = strings.Split(valueStr, ",")
	}

	return nil
}

// GetAdsByIDs returns ads for a list of IDs
func GetAdsByIDs(ids []int, userID int) ([]Ad, error) {
	return GetAdsByIDsWithDeleted(ids, userID, false)
}

// GetAdsByIDsWithDeleted returns ads for a list of IDs, optionally including deleted ads
func GetAdsByIDsWithDeleted(ids []int, userID int, includeDeleted bool) ([]Ad, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args := buildAdQueryWithDeleted(ids, userID, includeDeleted)

	var ads []Ad
	err := db.Select(&ads, query, args...)
	if err != nil {
		return nil, err
	}

	logger.Debug("Returning ads by IDs",
		"component", "ad", "count", len(ads))
	return ads, nil
}

// buildAdQueryWithDeleted builds the minimal query for fetching ads with IDs and user context, optionally including deleted ads
func buildAdQueryWithDeleted(ids []int, userID int, includeDeleted bool) (string, []interface{}) {
	var query string
	var args []interface{}

	if userID != 0 {
		// Query with bookmark status - minimal fields only
		query = `
			SELECT a.id, a.ad_category_id, a.title, a.price, a.created_at, a.deleted_at, a.user_id, a.image_count,
			       l.city, l.admin_area, l.country,
			       CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked
			FROM Ad a
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
		`
		args = append(args, userID)
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
func GetAdByID(adID int, userID int) (*Ad, error) {
	ads, err := GetAdsByIDs([]int{adID}, userID)
	if err != nil {
		return nil, err
	}
	if len(ads) == 0 {
		return nil, fmt.Errorf("ad with ID %d not found", adID)
	}
	return &ads[0], nil
}

// GetAdDetailByID returns the full AdDetail for a single ad ID
func GetAdDetailByID(adID int, userID int) (*AdDetail, error) {
	var query string
	var args []interface{}

	if userID != 0 {
		// Query with bookmark status - full fields for detail view
		query = `
			SELECT a.id, a.title, a.description, a.price,
				a.created_at, a.deleted_at, a.user_id,
				a.location_id, a.image_count, a.has_vector,
				l.raw_text as raw_location, l.city, l.admin_area, l.country,
				CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked,
				a.ad_category_id,
				CASE WHEN u.sms_opted_out = 1 AND u.notification_method = 'sms' THEN 1 ELSE 0 END as owner_unreachable
			FROM Ad a
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
			LEFT JOIN User u ON a.user_id = u.id
			WHERE a.id = ?
		`
		args = append(args, userID, adID)
	} else {
		// Query without bookmark status (default to false) - full fields for detail view
		query = `
			SELECT a.id, a.title, a.description, a.price,
				a.created_at, a.deleted_at, a.user_id,
				a.location_id, a.image_count, a.has_vector,
				l.raw_text as raw_location, l.city, l.admin_area, l.country,
				0 as is_bookmarked, a.ad_category_id,
				CASE WHEN u.sms_opted_out = 1 AND u.notification_method = 'sms' THEN 1 ELSE 0 END as owner_unreachable
			FROM Ad a
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN User u ON a.user_id = u.id
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

	adDetail := &adDetails[0]

	// Populate spec data
	err = populateSpecData(adDetail)
	if err != nil {
		return nil, fmt.Errorf("failed to populate spec data: %w", err)
	}

	return adDetail, nil
}

// GetMostPopularAds returns the top n ads by popularity using SQL
func GetMostPopularAds(n int) []Ad {
	logger.Debug("Querying for top popular ads",
		"component", "ad", "limit", n)
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
		logger.Error("SQL error querying popular ads",
			"error", err, "component", "ad")
		return nil
	}

	logger.Debug("Found ads from SQL query",
		"component", "ad", "count", len(ads))
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
