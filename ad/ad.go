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
	adDetail.SpecData = make(map[string][]string)

	fields := spec.GetDBFields(adDetail.AdCategoryID)

	// Get all chains for this category to determine which spec tables are used
	allChains, err := spec.GetAllChains(adDetail.AdCategoryID)
	if err != nil {
		return fmt.Errorf("failed to get chains: %w", err)
	}

	// Build map of spec table name -> AdSpec table info
	// Only include spec tables that are actually used by this category
	specTableMap := make(map[string]struct {
		adSpecTableName string
		idColumn        string
		validFields     map[string]bool
	})

	for _, chain := range allChains {
		specType := spec.GetSpecTypeFromChain(chain)
		specTableName := spec.GetSpecTableName(specType)

		// Map spec table to AdSpec table
		var adSpecTableName, idColumn string
		var validFields map[string]bool

		switch specTableName {
		case "SpecVehicle":
			adSpecTableName = "AdSpecVehicle"
			idColumn = "spec_vehicle_id"
			validFields = map[string]bool{
				"make_id": true, "year_id": true, "model_id": true, "engine_id": true,
			}
		case "SpecAgricultural":
			adSpecTableName = "AdSpecAgricultural"
			idColumn = "spec_agricultural_id"
			validFields = map[string]bool{
				"make_id": true, "year_id": true, "model_id": true,
			}
		case "SpecBicycle":
			adSpecTableName = "AdSpecBicycle"
			idColumn = "spec_bicycle_id"
			validFields = map[string]bool{
				"make_id": true, "model_id": true,
			}
		case "SpecPart":
			adSpecTableName = "AdSpecPart"
			idColumn = "spec_part_id"
			validFields = map[string]bool{
				"part_category_id": true, "part_subcategory_id": true,
			}
		default:
			continue // Unknown spec table
		}

		specTableMap[specTableName] = struct {
			adSpecTableName string
			idColumn        string
			validFields     map[string]bool
		}{adSpecTableName, idColumn, validFields}
	}

	// Build json_build_object query with subqueries for each field
	var jsonParts []string

	for _, field := range fields {
		config := field.Config()
		fieldName := config.Name
		tableName := config.TableName
		tableFieldName := config.TableFieldName

		// Find which spec table(s) contain this field
		var unionParts []string

		for specTableName, adSpecInfo := range specTableMap {
			// Skip if this field doesn't exist in this spec table
			if !adSpecInfo.validFields[tableFieldName] {
				continue
			}

			// Build FROM/JOIN clause matching original pattern:
			// Start from value table, join to spec, then to ad_spec
			fromClause := fmt.Sprintf(`
			 FROM %s
			 JOIN %s ON %s.id = %s.%s
			 JOIN %s ON %s.%s = %s.id
			 WHERE %s.ad_id = $1`,
				tableName,
				specTableName, tableName, specTableName, tableFieldName,
				adSpecInfo.adSpecTableName, adSpecInfo.adSpecTableName, adSpecInfo.idColumn, specTableName,
				adSpecInfo.adSpecTableName)

			unionParts = append(unionParts, fromClause)
		}

		if len(unionParts) == 0 {
			// Field doesn't exist in any spec table for this category, return empty array
			jsonParts = append(jsonParts, fmt.Sprintf("'%s', '[]'::json", fieldName))
			continue
		}

		// If only one spec table, use direct aggregation (like original)
		// If multiple spec tables, use UNION with subquery wrapper
		var subquery string
		if len(unionParts) == 1 {
			// Single spec table - use subquery for DISTINCT, then json_agg with ORDER BY
			// PostgreSQL doesn't support json_agg(DISTINCT value ORDER BY value)
			subquery = fmt.Sprintf(`COALESCE(
			(SELECT json_agg(value ORDER BY value)
			 FROM (SELECT DISTINCT %s.value %s) AS distinct_values),
			'[]'::json
		)`, tableName, unionParts[0])
		} else {
			// Multiple spec tables - need UNION with subquery wrapper
			// Build SELECT ... UNION SELECT ... then wrap
			var selectParts []string
			for _, fromClause := range unionParts {
				selectParts = append(selectParts, fmt.Sprintf("SELECT %s.value%s", tableName, fromClause))
			}
			unionQuery := strings.Join(selectParts, " UNION ")
			subquery = fmt.Sprintf(`COALESCE(
			(SELECT json_agg(value ORDER BY value)
			 FROM (SELECT DISTINCT value FROM (%s) AS unioned) AS distinct_values),
			'[]'::json
		)`, unionQuery)
		}

		jsonParts = append(jsonParts, fmt.Sprintf("'%s', %s", fieldName, subquery))
	}

	// Build final query with json_build_object
	query := fmt.Sprintf("SELECT json_build_object(%s) AS spec_fields",
		strings.Join(jsonParts, ",\n    "))

	// Execute query
	if err := db.QueryJSON(&adDetail.SpecData, query, adDetail.ID); err != nil {
		return fmt.Errorf("failed to query spec data: %w", err)
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
	if err := db.QueryJSON(&ads, query, args...); err != nil {
		return nil, err
	}

	logger.Debug("Returning ads by IDs",
		"component", "ad", "count", len(ads))
	return ads, nil
}

// buildAdQueryWithDeleted builds the minimal query for fetching ads with IDs and user context, optionally including deleted ads
func buildAdQueryWithDeleted(ids []int, userID int, includeDeleted bool) (string, []interface{}) {
	var args []interface{}
	argNum := 1

	// Build WHERE clause for IDs
	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = fmt.Sprintf("$%d", argNum)
		argNum++
	}
	idFilter := "a.id IN (" + strings.Join(placeholders, ",") + ")"

	// Build ORDER BY CASE to preserve input order
	orderByCase := "CASE a.id"
	for i, id := range ids {
		orderByCase += fmt.Sprintf(" WHEN %d THEN %d", id, i)
	}
	orderByCase += " END"

	var query string
	if userID != 0 {
		args = append(args, userID)
		bookmarkCase := "CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END"
		query = fmt.Sprintf(`
			SELECT json_agg(
				json_build_object(
					'id', a.id,
					'ad_category_id', a.ad_category_id,
					'title', a.title,
					'price', a.price,
					'created_at', a.created_at,
					'deleted_at', a.deleted_at,
					'user_id', a.user_id,
					'image_count', a.image_count,
					'city', l.city,
					'admin_area', l.admin_area,
					'country', l.country,
					'bookmarked', %s
				)
				ORDER BY %s
			) AS ads
			FROM Ad a
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = $1
			WHERE %s`,
			bookmarkCase, orderByCase, idFilter)
		if !includeDeleted {
			query += " AND a.deleted_at IS NULL"
		}
	} else {
		query = fmt.Sprintf(`
			SELECT json_agg(
				json_build_object(
					'id', a.id,
					'ad_category_id', a.ad_category_id,
					'title', a.title,
					'price', a.price,
					'created_at', a.created_at,
					'deleted_at', a.deleted_at,
					'user_id', a.user_id,
					'image_count', a.image_count,
					'city', l.city,
					'admin_area', l.admin_area,
					'country', l.country,
					'bookmarked', 0
				)
				ORDER BY %s
			) AS ads
			FROM Ad a
			LEFT JOIN Location l ON a.location_id = l.id
			WHERE %s`,
			orderByCase, idFilter)
		if !includeDeleted {
			query += " AND a.deleted_at IS NULL"
		}
	}

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
		logger.Debug("GetAdDetailByID query error",
			"component", "ad", "adID", adID, "userID", userID, "error", err)
		return nil, err
	}

	if len(adDetails) == 0 {
		logger.Debug("GetAdDetailByID no results",
			"component", "ad", "adID", adID, "userID", userID)
		return nil, fmt.Errorf("ad with ID %d not found", adID)
	}

	adDetail := &adDetails[0]

	// Populate spec data (populateSpecData always initializes SpecData to non-nil)
	err = populateSpecData(adDetail)
	if err != nil {
		logger.Debug("GetAdDetailByID populateSpecData error",
			"component", "ad", "adID", adID, "error", err)
		return nil, fmt.Errorf("failed to populate spec data: %w", err)
	}

	return adDetail, nil
}

// formatSQLQuery formats a SQL query string for better readability in logs
func formatSQLQuery(query string) string {
	// First, fix common formatting issues
	formatted := query

	// Fix "LEFT" appearing on separate line from "JOIN"
	formatted = strings.ReplaceAll(formatted, " LEFT\n", " LEFT ")
	formatted = strings.ReplaceAll(formatted, "\nLEFT JOIN", " LEFT JOIN")
	formatted = strings.ReplaceAll(formatted, " LEFT JOIN", "\n  LEFT JOIN")

	// Replace common SQL keywords with newline + keyword for better formatting
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "RIGHT JOIN",
		"INNER JOIN", "OUTER JOIN", "UNION ALL", "GROUP BY", "ORDER BY",
		"HAVING", "LIMIT", "OFFSET",
	}

	// Add newline before each keyword (case-insensitive)
	for _, keyword := range keywords {
		replacement := "\n" + strings.ToUpper(keyword)
		formatted = strings.ReplaceAll(formatted, " "+keyword+" ", " "+replacement+" ")
		formatted = strings.ReplaceAll(formatted, " "+keyword+"\n", " "+replacement+"\n")
		formatted = strings.ReplaceAll(formatted, "\n"+keyword+" ", "\n"+replacement+" ")
		formatted = strings.ReplaceAll(formatted, "("+keyword+" ", "("+replacement+" ")
	}

	// Clean up multiple consecutive newlines
	for strings.Contains(formatted, "\n\n\n") {
		formatted = strings.ReplaceAll(formatted, "\n\n\n", "\n\n")
	}

	// Add indentation after newlines (simple 2-space indent)
	lines := strings.Split(formatted, "\n")
	indented := make([]string, 0, len(lines))
	indentLevel := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Decrease indent for closing parentheses and certain keywords
		if strings.HasPrefix(trimmed, ")") || strings.HasPrefix(trimmed, "END") {
			indentLevel--
			if indentLevel < 0 {
				indentLevel = 0
			}
		}

		// Add indented line
		indent := strings.Repeat("  ", indentLevel)
		indented = append(indented, indent+trimmed)

		// Increase indent for opening parentheses and subqueries
		if strings.HasSuffix(trimmed, "(") || strings.Contains(trimmed, "FROM (") {
			indentLevel++
		}
	}

	return strings.Join(indented, "\n")
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
