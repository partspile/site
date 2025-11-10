package part

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/spec"
)

type PartCategory struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type PartSubCategory struct {
	ID             int    `db:"id"`
	PartCategoryID int    `db:"part_category_id"`
	Name           string `db:"name"`
}

var (
	// Cache part categories and subcategories by ad category
	categories    = make(map[int][]string)            // adCat -> []categoryNames
	subCategories = make(map[int]map[string][]string) // adCat -> categoryName -> []subcategoryNames

	// Cache for expensive ad-related queries
	partCache *cache.Cache[[]string]
)

// getCategories returns the part categories for the given ad category
func getCategories(adCat int) ([]string, error) {
	query := "SELECT name FROM PartCategory WHERE ad_category_id = ? ORDER BY name"
	var names []string
	err := db.Select(&names, query, adCat)
	return names, err
}

// GetCategories returns all categories for a specific ad category
func GetCategories(adCat int) []string {
	if cats, exists := categories[adCat]; exists {
		return cats
	}

	cats, err := getCategories(adCat)
	if err != nil {
		logger.Error("Failed to load categories",
			"error", err, "component", "parts", "adCat", adCat)
		return []string{}
	}
	categories[adCat] = cats

	return cats
}

// getSubCategories returns the subcategories for the given ad category and category
// Now uses the new schema where PartSubCategory has ad_category_id directly
func getSubCategories(adCat int, category string) ([]string, error) {
	query := `
		SELECT psc.name 
		FROM PartSubCategory psc
		JOIN Specification spec ON spec.part_subcategory_id = psc.id
		JOIN PartCategory pc ON spec.part_category_id = pc.id
		WHERE pc.name = ? AND psc.ad_category_id = ? AND spec.ad_category_id = ?
		ORDER BY psc.name
	`
	var names []string
	err := db.Select(&names, query, category, adCat, adCat)
	return names, err
}

// GetSubCategories returns all subcategories for a specific ad category and category
func GetSubCategories(adCat int, category string) []string {
	// Check if subcategories map exists for this ad category
	if subCategoriesMap, exists := subCategories[adCat]; exists {
		// Check if subcategories exist for this specific category
		if subCats, exists := subCategoriesMap[category]; exists {
			return subCats
		}
	} else {
		// Initialize subcategories map for this ad category
		subCategories[adCat] = make(map[string][]string)
	}

	subCats, err := getSubCategories(adCat, category)
	if err != nil {
		logger.Error("Failed to load subcategories",
			"error", err, "component", "parts", "adCat",
			adCat, "category", category)
		return []string{}
	}

	subCategories[adCat][category] = subCats

	return subCats
}

// InitPartCache initializes the part cache for expensive queries
func InitPartCache() error {
	var err error
	partCache, err = cache.New(func(value []string) int64 {
		return int64(len(value) * 50) // Estimate 50 bytes per string
	}, "Part Data Cache")
	if err != nil {
		return fmt.Errorf("failed to initialize part cache: %w", err)
	}
	logger.Info("Initialized successfully",
		"component", "part-cache")
	return nil
}

// GetPartCacheStats returns cache statistics for admin monitoring
func GetPartCacheStats() map[string]any {
	return partCache.Stats()
}

// ClearPartCache clears all items from the part cache and returns updated stats
func ClearPartCache() map[string]any {
	partCache.Clear()
	logger.Info("Cache cleared",
		"component", "part-cache")
	return partCache.Stats()
}

// ============================================================================
// SQL QUERY BUILDERS (Internal helper functions)
// ============================================================================

// buildAdCategoriesQuery builds the SQL query dynamically based on category field configuration
func buildAdCategoriesQuery(adCat int, fieldValues map[string]string) (string, []interface{}) {
	// Get category fields
	fieldConfigs := spec.GetCategoryFields(adCat)

	// Build base query - now PartCategory is joined via Specification
	baseQuery := `
		SELECT DISTINCT pc.name
		FROM PartCategory pc
		JOIN Specification spec ON spec.part_category_id = pc.id
		JOIN PartSubCategory psc ON spec.part_subcategory_id = psc.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
	`

	var joins []string
	var whereClauses []string
	var args []interface{}

	whereClauses = append(whereClauses, "a.ad_category_id = ?")
	args = append(args, adCat)

	// Add JOINs and WHERE clauses for each spec field (excluding part_category and part_subcategory)
	for _, fieldConfig := range fieldConfigs {
		// Skip part_category and part_subcategory - they're handled separately
		if fieldConfig.Name == "part_category" || fieldConfig.Name == "part_subcategory" {
			continue
		}

		value, hasValue := fieldValues[fieldConfig.Name]
		if !hasValue || value == "" {
			continue
		}

		tableName := fieldConfig.TableName
		tableFieldName := fieldConfig.TableFieldName
		joins = append(joins, fmt.Sprintf("JOIN %s ON spec.%s = %s.id",
			tableName, tableFieldName, tableName))

		if fieldConfig.FieldType == "int" {
			// For integer fields, convert string to int
			intValue, err := strconv.Atoi(value)
			if err == nil {
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.%s = ?", tableName, fieldConfig.ValueColumnName))
				args = append(args, intValue)
			}
		} else {
			// For string fields, compare with value column
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.%s = ?", tableName, fieldConfig.ValueColumnName))
			args = append(args, value)
		}
	}

	// Build final query
	query := baseQuery
	if len(joins) > 0 {
		query += " " + strings.Join(joins, " ")
	}
	query += " WHERE " + strings.Join(whereClauses, " AND ")
	query += " ORDER BY pc.name"

	return query, args
}

// buildAdSubCategoriesQuery builds the SQL query dynamically based on category field configuration
func buildAdSubCategoriesQuery(adCat int, fieldValues map[string]string, category string) (string, []interface{}) {
	// Get category fields
	fieldConfigs := spec.GetCategoryFields(adCat)

	// Build base query - now PartSubCategory has ad_category_id directly
	baseQuery := `
		SELECT DISTINCT psc.name
		FROM PartSubCategory psc
		JOIN Specification spec ON spec.part_subcategory_id = psc.id
		JOIN PartCategory pc ON spec.part_category_id = pc.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
	`

	var joins []string
	var whereClauses []string
	var args []interface{}

	whereClauses = append(whereClauses, "a.ad_category_id = ?")
	args = append(args, adCat)

	// Add JOINs and WHERE clauses for each spec field (excluding part_category and part_subcategory)
	for _, fieldConfig := range fieldConfigs {
		// Skip part_category and part_subcategory - they're handled separately
		if fieldConfig.Name == "part_category" || fieldConfig.Name == "part_subcategory" {
			continue
		}

		value, hasValue := fieldValues[fieldConfig.Name]
		if !hasValue || value == "" {
			continue
		}

		tableName := fieldConfig.TableName
		tableFieldName := fieldConfig.TableFieldName
		joins = append(joins, fmt.Sprintf("JOIN %s ON spec.%s = %s.id",
			tableName, tableFieldName, tableName))

		if fieldConfig.FieldType == "int" {
			// For integer fields, convert string to int
			intValue, err := strconv.Atoi(value)
			if err == nil {
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.%s = ?", tableName, fieldConfig.ValueColumnName))
				args = append(args, intValue)
			}
		} else {
			// For string fields, compare with value column
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.%s = ?", tableName, fieldConfig.ValueColumnName))
			args = append(args, value)
		}
	}

	// Add part category filter
	whereClauses = append(whereClauses, "pc.name = ?")
	args = append(args, category)

	// Build final query
	query := baseQuery
	if len(joins) > 0 {
		query += " " + strings.Join(joins, " ")
	}
	query += " WHERE " + strings.Join(whereClauses, " AND ")
	query += " ORDER BY psc.name"

	return query, args
}

// ============================================================================
// TREE VIEW FUNCTIONS BROWSE MODE (q == "")
// ============================================================================

// GetAdCategories returns categories that have existing ads for given field values (for tree view)
func GetAdCategories(adCat int, fieldValues map[string]string) ([]string, error) {
	// URL decode field values
	decodedValues := make(map[string]string)
	for k, v := range fieldValues {
		decoded, _ := url.QueryUnescape(v)
		decodedValues[k] = decoded
	}

	// Create cache key from sorted field values
	cacheKey := fmt.Sprintf("ad:categories:%d", adCat)
	for k, v := range decodedValues {
		cacheKey += fmt.Sprintf(":%s=%s", k, v)
	}

	// Check cache first
	if cached, found := partCache.Get(cacheKey); found {
		return cached, nil
	}

	query, args := buildAdCategoriesQuery(adCat, decodedValues)
	var categories []string
	err := db.Select(&categories, query, args...)
	if err != nil {
		return nil, err
	}

	// Cache the results
	partCache.Set(cacheKey, categories, int64(len(categories)*50))
	return categories, nil
}

// GetAdSubCategories returns subcategories that have existing ads for given field values and category (for tree view)
func GetAdSubCategories(adCat int, fieldValues map[string]string, category string) ([]string, error) {
	// URL decode field values
	decodedValues := make(map[string]string)
	for k, v := range fieldValues {
		decoded, _ := url.QueryUnescape(v)
		decodedValues[k] = decoded
	}
	category, _ = url.QueryUnescape(category)

	// Create cache key from sorted field values
	cacheKey := fmt.Sprintf("ad:subcategories:%d", adCat)
	for k, v := range decodedValues {
		cacheKey += fmt.Sprintf(":%s=%s", k, v)
	}
	cacheKey += fmt.Sprintf(":%s", category)

	// Check cache first
	if cached, found := partCache.Get(cacheKey); found {
		return cached, nil
	}

	query, args := buildAdSubCategoriesQuery(adCat, decodedValues, category)
	var subCategories []string
	err := db.Select(&subCategories, query, args...)
	if err != nil {
		return nil, err
	}

	// Cache the results
	partCache.Set(cacheKey, subCategories, int64(len(subCategories)*50))
	return subCategories, nil
}

// ============================================================================
// TREE VIEW FUNCTIONS SEARCH MODE (q != "")
// ============================================================================

// GetCategoriesForAds returns categories for specific field values, filtered by ad IDs
func GetCategoriesForAds(adIDs []int, fieldValues map[string]string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	// URL decode field values
	decodedValues := make(map[string]string)
	for k, v := range fieldValues {
		decoded, _ := url.QueryUnescape(v)
		decodedValues[k] = decoded
	}

	// Get ad category from first ad (all ads should have same category)
	var adCat int
	err := db.Get(&adCat, "SELECT ad_category_id FROM Ad WHERE id = ?", adIDs[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get ad category: %w", err)
	}

	// Build dynamic query
	baseQuery, baseArgs := buildAdCategoriesQuery(adCat, decodedValues)
	if baseQuery == "" {
		return nil, fmt.Errorf("failed to build query")
	}

	// Add ad ID filter
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := strings.Replace(baseQuery, "ORDER BY pc.name",
		fmt.Sprintf("AND a.id IN (%s) ORDER BY pc.name", strings.Join(placeholders, ",")), 1)

	args := append(baseArgs, make([]interface{}, len(adIDs))...)
	for i, id := range adIDs {
		args[len(baseArgs)+i] = id
	}

	var categories []string
	err = db.Select(&categories, query, args...)
	return categories, err
}

// GetSubCategoriesForAds returns subcategories for specific field values and category, filtered by ad IDs
func GetSubCategoriesForAds(adIDs []int, fieldValues map[string]string, category string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	// URL decode field values
	decodedValues := make(map[string]string)
	for k, v := range fieldValues {
		decoded, _ := url.QueryUnescape(v)
		decodedValues[k] = decoded
	}
	category, _ = url.QueryUnescape(category)

	// Get ad category from first ad
	var adCat int
	err := db.Get(&adCat, "SELECT ad_category_id FROM Ad WHERE id = ?", adIDs[0])
	if err != nil {
		return nil, fmt.Errorf("failed to get ad category: %w", err)
	}

	// Build dynamic query
	baseQuery, baseArgs := buildAdSubCategoriesQuery(adCat, decodedValues, category)
	if baseQuery == "" {
		return nil, fmt.Errorf("failed to build query")
	}

	// Add ad ID filter
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := strings.Replace(baseQuery, "ORDER BY psc.name",
		fmt.Sprintf("AND a.id IN (%s) ORDER BY psc.name", strings.Join(placeholders, ",")), 1)

	args := append(baseArgs, make([]interface{}, len(adIDs))...)
	for i, id := range adIDs {
		args[len(baseArgs)+i] = id
	}

	var subCategories []string
	err = db.Select(&subCategories, query, args...)
	return subCategories, err
}
