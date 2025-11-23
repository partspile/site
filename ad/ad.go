package ad

import (
	"fmt"
	"strings"
	"time"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// Ad represents the minimal advertisement data needed for list/grid/tree views
type Ad struct {
	// Core database fields
	ID           int        `json:"id"`
	AdCategoryID int        `json:"ad_category_id"`
	Title        string     `json:"title"`
	Price        float64    `json:"price"`
	CreatedAt    time.Time  `json:"created_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
	UserID       int        `json:"user_id"`
	ImageCount   int        `json:"image_count"`

	// Location fields from joins
	City      string `json:"city"`
	AdminArea string `json:"admin_area"`
	Country   string `json:"country"`

	// User-specific computed fields
	Bookmarked bool `json:"bookmarked"`
}

// AdDetail extends Ad with additional fields needed for detail view
type AdDetail struct {
	Ad // Embed the minimal Ad struct

	// Additional fields for detail view
	Description string `json:"description"`
	LocationID  int    `json:"location_id"`
	HasVector   bool   `json:"has_vector"`
	RawLocation string `json:"raw_location"`

	// Spec data from spec joins - map of field name to array of values
	// Single-value fields have arrays with one element, multi-value fields
	// have multiple elements
	SpecData map[string][]string `json:"spec_data"`

	// Computed fields
	OwnerUnreachable int `json:"owner_unreachable"`
}

// IsArchived returns true if the ad is archived (deleted)
func (a Ad) IsArchived() bool {
	return a.DeletedAt != nil
}

// populateSpecData fills in the spec data for the ad dynamically based on spec fields
func populateSpecData(adDetail *AdDetail) error {
	adDetail.SpecData = make(map[string][]string)

	fields := GetSpecFields(adDetail.AdCategoryID)
	if len(fields) == 0 {
		// No spec fields for this category, return empty map
		return nil
	}

	// Build JSON object keys for each spec field
	jsonKeys := make([]string, len(fields))
	for i, field := range fields {
		jsonKeys[i] = field.BuildSpecDataSubquery()
	}

	// Build the full query - return the object directly without wrapper
	query := fmt.Sprintf("SELECT json_build_object(%s)", strings.Join(jsonKeys, ", "))

	// Execute query directly into adDetail.SpecData
	return db.QueryJSON(&adDetail.SpecData, query, adDetail.ID)
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

	// Build bookmark expression and JOIN clause conditionally
	var bookmarkExpr string
	var bookmarkJoin string
	if userID != 0 {
		args = append(args, userID)
		bookmarkExpr = "(CASE WHEN ba.ad_id IS NOT NULL THEN true ELSE false END)::boolean"
		bookmarkJoin = "LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = $1"
		argNum++ // IDs start at $2
	} else {
		bookmarkExpr = "false"
		bookmarkJoin = ""
	}

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

	query := fmt.Sprintf(`
		SELECT json_agg(
			json_build_object(
				'id', a.id,
				'ad_category_id', a.ad_category_id,
				'title', a.title,
				'price', a.price,
				'created_at', a.created_at AT TIME ZONE 'UTC',
				'deleted_at', a.deleted_at AT TIME ZONE 'UTC',
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
		%s
		WHERE %s`,
		bookmarkExpr, orderByCase, bookmarkJoin, idFilter)

	if !includeDeleted {
		query += " AND a.deleted_at IS NULL"
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
	var args []interface{}
	var bookmarkExpr string
	var bookmarkJoin string
	var adIDParam string

	if userID != 0 {
		args = append(args, userID, adID)
		bookmarkExpr = "(CASE WHEN ba.ad_id IS NOT NULL THEN true ELSE false END)::boolean"
		bookmarkJoin = "LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = $1"
		adIDParam = "$2"
	} else {
		args = append(args, adID)
		bookmarkExpr = "false"
		bookmarkJoin = ""
		adIDParam = "$1"
	}

	query := fmt.Sprintf(`
		SELECT json_build_object(
			'id', a.id,
			'title', a.title,
			'description', a.description,
			'price', a.price,
			'created_at', a.created_at AT TIME ZONE 'UTC',
			'deleted_at', a.deleted_at AT TIME ZONE 'UTC',
			'user_id', a.user_id,
			'location_id', a.location_id,
			'image_count', a.image_count,
			'has_vector', (a.has_vector = 1)::boolean,
			'raw_location', l.raw_text,
			'city', l.city,
			'admin_area', l.admin_area,
			'country', l.country,
			'bookmarked', %s,
			'ad_category_id', a.ad_category_id,
			'owner_unreachable', CASE WHEN u.sms_opted_out = 1 AND u.notification_method = 'sms' THEN 1 ELSE 0 END
		) AS ad_detail
		FROM Ad a
		LEFT JOIN Location l ON a.location_id = l.id
		%s
		LEFT JOIN Users u ON a.user_id = u.id
		WHERE a.id = %s`,
		bookmarkExpr, bookmarkJoin, adIDParam)

	var adDetail AdDetail
	err := db.QueryJSON(&adDetail, query, args...)
	if err != nil {
		logger.Debug("GetAdDetailByID query error",
			"component", "ad", "adID", adID, "userID", userID, "error", err)
		return nil, fmt.Errorf("ad with ID %d not found", adID)
	}

	// Populate spec data (populateSpecData always initializes SpecData to non-nil)
	err = populateSpecData(&adDetail)
	if err != nil {
		logger.Debug("GetAdDetailByID populateSpecData error",
			"component", "ad", "adID", adID, "error", err)
		return nil, fmt.Errorf("failed to populate spec data: %w", err)
	}

	return &adDetail, nil
}

// GetMostPopularAds returns the top n ads by popularity using SQL
func GetMostPopularAds(n int) []Ad {
	logger.Debug("Querying for top popular ads",
		"component", "ad", "limit", n)
	query := `
		SELECT json_agg(
			json_build_object(
				'id', a.id,
				'ad_category_id', a.ad_category_id,
				'title', a.title,
				'price', a.price,
				'created_at', a.created_at AT TIME ZONE 'UTC',
				'deleted_at', a.deleted_at AT TIME ZONE 'UTC',
				'user_id', a.user_id,
				'image_count', a.image_count,
				'city', l.city,
				'admin_area', l.admin_area,
				'country', l.country,
				'bookmarked', false
			)
		) AS ads
		FROM (
			SELECT a.id, a.ad_category_id, a.title, a.price, a.created_at, a.deleted_at, a.user_id, a.image_count,
			       l.city, l.admin_area, l.country
			FROM Ad a
			LEFT JOIN Location l ON a.location_id = l.id
			WHERE a.deleted_at IS NULL
			ORDER BY (
				CAST(a.click_count AS DOUBLE PRECISION) * 2.0 + 
				CAST(COALESCE((SELECT COUNT(*) FROM BookmarkedAd ba WHERE ba.ad_id = a.id), 0) AS DOUBLE PRECISION) * 3.0 + 
				100.0 / GREATEST(EXTRACT(EPOCH FROM (NOW() - a.created_at)) / 86400.0, 0.1)
			) DESC
			LIMIT $1
		) a
	`

	var ads []Ad
	err := db.QueryJSON(&ads, query, n)
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

// GetAdsForAll returns all ads filtered by field values
func GetAdsForAll(fieldValues map[string]string, adCategoryID int, fields []Field) ([]Ad, error) {
	if len(fieldValues) == 0 {
		// No filters, return empty
		return []Ad{}, nil
	}

	// Get chains from adFieldChains - already grouped by spec table
	chains := adFieldChains[adCategoryID]
	if len(chains) == 0 {
		return []Ad{}, nil
	}

	// Filter chains to only those with field values
	activeChains := make([][]Field, 0)
	for _, chain := range chains {
		if len(chain) == 0 {
			continue
		}

		// Check if this chain has any fields with values
		hasValues := false
		for _, field := range chain {
			if value, exists := fieldValues[field.Name()]; exists && value != "" {
				hasValues = true
				break
			}
		}
		if hasValues {
			activeChains = append(activeChains, chain)
		}
	}

	if len(activeChains) == 0 {
		return []Ad{}, nil
	}

	// Build single query that joins all chains
	adIDs, err := findAdIDsForAllChains(activeChains, fieldValues, adCategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to find ad IDs: %w", err)
	}

	if len(adIDs) == 0 {
		return []Ad{}, nil
	}

	// Get ads for the filtered ad IDs
	return GetAdsByIDs(adIDs, 0)
}

// findAdIDsForAllChains finds ad IDs that match field values across all chains in a single query
func findAdIDsForAllChains(chains [][]Field, fieldValues map[string]string, adCategoryID int) ([]int, error) {
	if len(chains) == 0 {
		return []int{}, nil
	}

	// Build query that joins all chains together
	var joins []string
	var whereClauses []string
	var args []interface{}

	// Start from Ad table
	query := "SELECT DISTINCT a.id FROM Ad a"

	// Add category and deleted filter
	args = append(args, adCategoryID)
	whereClauses = append(whereClauses, "a.ad_category_id = $1")
	whereClauses = append(whereClauses, "a.deleted_at IS NULL")

	argNum := 2

	// Process each chain
	for chainIdx, chain := range chains {
		if len(chain) == 0 {
			continue
		}

		// Get spec table info from first field in chain
		firstField := chain[0]
		specTableName, _, adSpecTableName, _ := firstField.GetTableNames()
		if specTableName == "" || adSpecTableName == "" {
			continue
		}

		// Create unique aliases for this chain
		specAlias := fmt.Sprintf("spec%d", chainIdx)
		adSpecAlias := fmt.Sprintf("adspec%d", chainIdx)

		// Join AdSpec table
		joins = append(joins, fmt.Sprintf(
			"JOIN %s %s ON a.id = %s.ad_id",
			adSpecTableName, adSpecAlias, adSpecAlias))

		// Join Spec table
		joins = append(joins, fmt.Sprintf(
			"JOIN %s %s ON %s.spec_id = %s.id",
			specTableName, specAlias, adSpecAlias, specAlias))

		// Add category filter for this spec table
		whereClauses = append(whereClauses,
			fmt.Sprintf("%s.ad_category_id = $%d", specAlias, argNum))
		args = append(args, adCategoryID)
		argNum++

		// Process fields in this chain
		for _, field := range chain {
			fieldName := field.Name()
			value, hasValue := fieldValues[fieldName]
			if !hasValue || value == "" {
				continue
			}

			_, specTableFieldName, _, tableName := field.GetTableNames()
			if tableName == "" || specTableFieldName == "" {
				continue
			}

			// Create unique alias for this field table
			fieldAlias := fmt.Sprintf("%s_%d", tableName, chainIdx)

			// Join field table
			joins = append(joins, fmt.Sprintf(
				"JOIN %s %s ON %s.%s = %s.id",
				tableName, fieldAlias, specAlias, specTableFieldName, fieldAlias))

			// Add field value filter
			if field.IsMultiValue() {
				// Multi-value: split comma-separated values and use IN clause
				values := strings.Split(value, ",")
				var placeholders []string
				for _, v := range values {
					args = append(args, strings.TrimSpace(v))
					placeholders = append(placeholders, fmt.Sprintf("$%d", argNum))
					argNum++
				}
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.value IN (%s)", fieldAlias, strings.Join(placeholders, ", ")))
			} else {
				// Single value
				args = append(args, value)
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.value = $%d", fieldAlias, argNum))
				argNum++
			}
		}
	}

	if len(joins) == 0 {
		return []int{}, nil
	}

	// Build final query
	query += " " + strings.Join(joins, " ")
	query += " WHERE " + strings.Join(whereClauses, " AND ")

	var adIDs []int
	err := db.Select(&adIDs, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ad IDs: %w", err)
	}

	return adIDs, nil
}

// findAdIDsForChain finds ad IDs that match field values for a specific chain
// (kept for potential future use, but currently unused)
func findAdIDsForChain(chain []Field, fieldValues map[string]string, adCategoryID int) ([]int, error) {
	if len(chain) == 0 {
		return []int{}, nil
	}

	// Get spec table info from first field in chain
	firstField := chain[0]
	specTableName, _, adSpecTableName, _ := firstField.GetTableNames()
	if specTableName == "" || adSpecTableName == "" {
		return []int{}, nil
	}

	// Build field metadata
	type fieldMeta struct {
		name               string
		tableName          string
		specTableFieldName string
		alias              string
		isMultiValue       bool
	}
	fieldMetas := make([]fieldMeta, 0)
	for _, field := range chain {
		fieldName := field.Name()
		value, hasValue := fieldValues[fieldName]
		if !hasValue || value == "" {
			continue
		}

		_, specTableFieldName, _, tableName := field.GetTableNames()
		if tableName == "" || specTableFieldName == "" {
			continue
		}

		fieldMetas = append(fieldMetas, fieldMeta{
			name:               fieldName,
			tableName:          tableName,
			specTableFieldName: specTableFieldName,
			alias:              tableName,
			isMultiValue:       field.IsMultiValue(),
		})
	}

	if len(fieldMetas) == 0 {
		return nil, nil
	}

	// Build JOINs
	var joins []string
	for _, fm := range fieldMetas {
		joins = append(joins, fmt.Sprintf(
			"JOIN %s %s ON %s.%s = %s.id",
			fm.tableName, fm.alias, specTableName,
			fm.specTableFieldName, fm.alias))
	}

	// Build WHERE clauses
	var args []interface{}
	var whereClauses []string

	// Category filter
	args = append(args, adCategoryID)
	whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", specTableName, len(args)))

	// Field value filters
	for _, fm := range fieldMetas {
		value := fieldValues[fm.name]
		if fm.isMultiValue {
			// Multi-value: split comma-separated values and use IN clause
			values := strings.Split(value, ",")
			var placeholders []string
			for _, v := range values {
				args = append(args, strings.TrimSpace(v))
				placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
			}
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.value IN (%s)", fm.alias, strings.Join(placeholders, ", ")))
		} else {
			// Single value
			args = append(args, value)
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.value = $%d", fm.alias, len(args)))
		}
	}

	// Query: find spec IDs, then ad IDs
	specQuery := fmt.Sprintf(`
		SELECT DISTINCT %s.id
		FROM %s
		%s
		WHERE %s`,
		specTableName, specTableName,
		strings.Join(joins, " "),
		strings.Join(whereClauses, " AND "))

	var specIDs []int
	err := db.Select(&specIDs, specQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query spec IDs: %w", err)
	}

	if len(specIDs) == 0 {
		return []int{}, nil
	}

	// Find ad IDs that have these spec IDs
	specPlaceholders := make([]string, len(specIDs))
	specArgs := make([]interface{}, len(specIDs))
	for i, id := range specIDs {
		specPlaceholders[i] = fmt.Sprintf("$%d", i+1)
		specArgs[i] = id
	}

	adQuery := fmt.Sprintf(`
		SELECT DISTINCT ad_id
		FROM %s
		WHERE spec_id IN (%s)
		AND ad_id IN (SELECT id FROM Ad WHERE deleted_at IS NULL)`,
		adSpecTableName, strings.Join(specPlaceholders, ", "))

	var adIDs []int
	err = db.Select(&adIDs, adQuery, specArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to query ad IDs: %w", err)
	}

	return adIDs, nil
}

// GetAdsForAdIDs returns ads for the given IDs (placeholder implementation)
func GetAdsForAdIDs(adIDs []int) ([]Ad, error) {
	// This is a placeholder implementation
	return nil, fmt.Errorf("GetAdsForAdIDs not implemented yet")
}
