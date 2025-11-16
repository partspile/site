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

		// Add JOINs for this field's table from all spec tables using COALESCE
		// Only one spec table will have the field, so COALESCE will pick the non-null value
		joins = append(joins, fmt.Sprintf(`LEFT JOIN %s ON (
			(ad_spec.spec_type = 'vehicle' AND spec_vehicle.%s = %s.id) OR
			(ad_spec.spec_type = 'agricultural' AND spec_agricultural.%s = %s.id) OR
			(ad_spec.spec_type = 'bicycle' AND spec_bicycle.%s = %s.id) OR
			(ad_spec.spec_type = 'part' AND spec_part.%s = %s.id)
		)`, tableName, tableFieldName, tableName, tableFieldName, tableName, tableFieldName, tableName, tableFieldName, tableName))

		// Build SELECT clause based on field type
		// Use COALESCE to get value from the appropriate spec table
		if fieldConfig.FieldType == "int" {
			// For integer fields (like year), use STRING_AGG with CAST (PostgreSQL)
			selectParts = append(selectParts,
				fmt.Sprintf("COALESCE(STRING_AGG(DISTINCT CAST(%s.value AS TEXT), ','), '') as %s",
					tableName, fieldName))
		} else {
			// For string fields, use STRING_AGG for both single and multi-value (PostgreSQL)
			selectParts = append(selectParts,
				fmt.Sprintf("COALESCE(STRING_AGG(DISTINCT %s.value, ','), '') as %s",
					tableName, fieldName))
		}
		fieldNames = append(fieldNames, fieldName)
	}

	// Build final query - use UNION ALL to combine results from all AdSpec* tables
	// Each AdSpec* table joins to its corresponding Spec* table
	var unionQueries []string

	adSpecTables := []struct {
		tableName     string
		specTableName string
		idColumn      string
	}{
		{"AdSpecVehicle", "SpecVehicle", "spec_vehicle_id"},
		{"AdSpecAgricultural", "SpecAgricultural", "spec_agricultural_id"},
		{"AdSpecBicycle", "SpecBicycle", "spec_bicycle_id"},
		{"AdSpecPart", "SpecPart", "spec_part_id"},
	}

	for _, adSpecTable := range adSpecTables {
		// Build query for this spec type - use $1 for ad_id in all subqueries
		query := fmt.Sprintf(`
			SELECT %s
			FROM %s ad_spec
			JOIN %s spec ON ad_spec.%s = spec.id
			%s
			WHERE ad_spec.ad_id = $1
		`, strings.Join(selectParts, ", "), adSpecTable.tableName, adSpecTable.specTableName, adSpecTable.idColumn, strings.Join(joins, " "))
		unionQueries = append(unionQueries, query)
	}

	// Combine with UNION ALL and group
	query := fmt.Sprintf(`SELECT %s FROM (
		%s
	) combined
	GROUP BY %s`, strings.Join(selectParts, ", "), strings.Join(unionQueries, " UNION ALL "), strings.Join(selectParts, ", "))

	// Execute query - use a map to store results dynamically
	// Note: query already uses PostgreSQL format ($1), so no rebinding needed
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
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = $1
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

	// Build PostgreSQL placeholders ($1, $2, etc.) based on current arg count
	placeholders := make([]string, len(ids))
	argNum := len(args) + 1
	for i := range ids {
		placeholders[i] = fmt.Sprintf("$%d", argNum)
		argNum++
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
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = $1
			LEFT JOIN Users u ON a.user_id = u.id
			WHERE a.id = $2
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
			LEFT JOIN Users u ON a.user_id = u.id
			WHERE a.id = $1
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
			CAST(a.click_count AS DOUBLE PRECISION) * 2.0 + 
			CAST(COALESCE((SELECT COUNT(*) FROM BookmarkedAd ba WHERE ba.ad_id = a.id), 0) AS DOUBLE PRECISION) * 3.0 + 
			100.0 / GREATEST(EXTRACT(EPOCH FROM (NOW() - a.created_at)) / 86400.0, 0.1)
		) DESC
		LIMIT $1
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
	_, err := db.Exec("UPDATE Ad SET deleted_at = CURRENT_TIMESTAMP WHERE id = $1", adID)
	return err
}

// RestoreAd restores an archived ad by clearing its deleted_at timestamp
func RestoreAd(adID int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = NULL WHERE id = $1", adID)
	return err
}

// GetUserActiveAdIDs returns the IDs of active ads for a user
func GetUserActiveAdIDs(userID int) ([]int, error) {
	var adIDs []int
	err := db.Select(&adIDs, "SELECT id FROM Ad WHERE user_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC", userID)
	return adIDs, err
}

// GetUserDeletedAdIDs returns the IDs of deleted ads for a user
func GetUserDeletedAdIDs(userID int) ([]int, error) {
	var adIDs []int
	err := db.Select(&adIDs, "SELECT id FROM Ad WHERE user_id = $1 AND deleted_at IS NOT NULL ORDER BY deleted_at DESC", userID)
	return adIDs, err
}

// ArchiveAdsByUserID archives all ads for a user
func ArchiveAdsByUserID(userID int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = $1 AND deleted_at IS NULL", userID)
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
