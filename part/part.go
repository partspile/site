package part

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/db"
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
		log.Printf("[parts] Failed to load categories for %s: %v", adCat, err)
		return []string{}
	}
	categories[adCat] = cats

	return cats
}

// getSubCategories returns the subcategories for the given ad category and category
func getSubCategories(adCat int, category string) ([]string, error) {
	query := `
		SELECT psc.name 
		FROM PartSubAdCategory psc
		JOIN PartCategory pc ON psc.part_category_id = pc.id
		WHERE pc.name = ? AND pc.ad_category_id = ?
		ORDER BY psc.name
	`
	var names []string
	err := db.Select(&names, query, category, adCat)
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
		log.Printf("[parts] Failed to load subcategories for %s/%s: %v", adCat, category, err)
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
	log.Printf("[part-cache] Initialized successfully")
	return nil
}

// GetPartCacheStats returns cache statistics for admin monitoring
func GetPartCacheStats() map[string]any {
	return partCache.Stats()
}

// ClearPartCache clears all items from the part cache and returns updated stats
func ClearPartCache() map[string]any {
	partCache.Clear()
	log.Printf("[part-cache] Cache cleared")
	return partCache.Stats()
}

// ============================================================================
// SQL QUERY BUILDERS (Internal helper functions)
// ============================================================================

// buildAdCategoriesQuery builds the SQL query for finding categories that have existing ads for make/year/model/engine
func buildAdCategoriesQuery(adCat int, makeName, year, model, engine string) (string, []interface{}) {
	var query string
	var args []interface{}

	// Build common query parts
	baseQuery := `
		SELECT DISTINCT pc.name
		FROM PartCategory pc
		JOIN PartSubAdCategory psc ON pc.id = psc.category_id
		JOIN Ad a ON psc.id = a.part_subcategory_id
		JOIN AdVehicle ac ON a.id = ac.ad_id
		JOIN Vehicle c ON ac.vehicle_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Model mo ON c.model_id = mo.id
	`

	// Add conditional JOINs and WHERE clauses based on ad category
	switch adCat {
	case ad.AdCategoryCar, ad.AdCategoryCarPart, ad.AdCategoryMotorcycle, ad.AdCategoryMotorcyclePart:
		// Cars and Motorcycles: make, year, model, engine
		query = baseQuery + `
			JOIN Year y ON c.year_id = y.id
			JOIN Engine e ON c.engine_id = e.id
			WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ?
			ORDER BY pc.name
		`
		args = []interface{}{makeName, year, model, engine}

	case ad.AdCategoryAg, ad.AdCategoryAgPart:
		// Ag Equipment: make, year, model (no engine)
		query = baseQuery + `
			JOIN Year y ON c.year_id = y.id
			WHERE m.name = ? AND y.year = ? AND mo.name = ?
			ORDER BY pc.name
		`
		args = []interface{}{makeName, year, model}

	case ad.AdCategoryBicycle, ad.AdCategoryBicyclePart:
		// Bicycles: make, model (no year, no engine)
		query = baseQuery + `
			WHERE m.name = ? AND mo.name = ?
			ORDER BY pc.name
		`
		args = []interface{}{makeName, model}

	default:
		// Unknown ad category - panic to catch programming errors
		panic(fmt.Sprintf("unsupported ad category: %v", adCat))
	}

	return query, args
}

// buildAdSubCategoriesQuery builds the SQL query for finding subcategories that have existing ads for make/year/model/engine/category
func buildAdSubCategoriesQuery(adCat int, makeName, year, model, engine, category string) (string, []interface{}) {
	var query string
	var args []interface{}

	// Build common query parts
	baseQuery := `
		SELECT DISTINCT psc.name
		FROM PartSubAdCategory psc
		JOIN PartCategory pc ON psc.part_category_id = pc.id
		JOIN Ad a ON psc.id = a.part_subcategory_id
		JOIN AdVehicle ac ON a.id = ac.ad_id
		JOIN Vehicle c ON ac.vehicle_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Model mo ON c.model_id = mo.id
	`

	// Add conditional JOINs and WHERE clauses based on ad category
	switch adCat {
	case ad.AdCategoryCar, ad.AdCategoryCarPart, ad.AdCategoryMotorcycle, ad.AdCategoryMotorcyclePart:
		// Cars and Motorcycles: make, year, model, engine
		query = baseQuery + `
			JOIN Year y ON c.year_id = y.id
			JOIN Engine e ON c.engine_id = e.id
			WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ? AND pc.name = ?
			ORDER BY psc.name
		`
		args = []interface{}{makeName, year, model, engine, category}

	case ad.AdCategoryAg, ad.AdCategoryAgPart:
		// Ag Equipment: make, year, model (no engine)
		query = baseQuery + `
			JOIN Year y ON c.year_id = y.id
			WHERE m.name = ? AND y.year = ? AND mo.name = ? AND pc.name = ?
			ORDER BY psc.name
		`
		args = []interface{}{makeName, year, model, category}

	case ad.AdCategoryBicycle, ad.AdCategoryBicyclePart:
		// Bicycles: make, model (no year, no engine)
		query = baseQuery + `
			WHERE m.name = ? AND mo.name = ? AND pc.name = ?
			ORDER BY psc.name
		`
		args = []interface{}{makeName, model, category}

	default:
		// Unknown ad category - panic to catch programming errors
		panic(fmt.Sprintf("unsupported ad category: %v", adCat))
	}

	return query, args
}

// ============================================================================
// TREE VIEW FUNCTIONS BROWSE MODE (q == "")
// ============================================================================

// GetAdCategories returns categories that have existing ads for make/year/model/engine (for tree view)
func GetAdCategories(adCat int, makeName, year, model, engine string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)

	// Create cache key
	cacheKey := fmt.Sprintf("ad:categories:%d:%s:%s:%s:%s", adCat, makeName, year, model, engine)

	// Check cache first
	if cached, found := partCache.Get(cacheKey); found {
		return cached, nil
	}

	query, args := buildAdCategoriesQuery(adCat, makeName, year, model, engine)
	var categories []string
	err := db.Select(&categories, query, args...)
	if err != nil {
		return nil, err
	}

	// Cache the results
	partCache.Set(cacheKey, categories, int64(len(categories)*50))
	return categories, nil
}

// GetAdSubCategories returns subcategories that have existing ads for make/year/model/engine/category (for tree view)
func GetAdSubCategories(adCat int, makeName, year, model, engine, category string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)
	category, _ = url.QueryUnescape(category)

	// Create cache key
	cacheKey := fmt.Sprintf("ad:subcategories:%d:%s:%s:%s:%s:%s", adCat, makeName, year, model, engine, category)

	// Check cache first
	if cached, found := partCache.Get(cacheKey); found {
		return cached, nil
	}

	query, args := buildAdSubCategoriesQuery(adCat, makeName, year, model, engine, category)
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

// GetCategoriesForAds returns categories for a specific make/year/model/engine, filtered by ad IDs
func GetCategoriesForAds(adIDs []int, makeName, year, model, engine string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	// URL decode the parameters
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT pc.name
		FROM PartCategory pc
		JOIN PartSubAdCategory psc ON pc.id = psc.category_id
		JOIN Ad a ON psc.id = a.part_subcategory_id
		JOIN AdVehicle ac ON a.id = ac.ad_id
		JOIN Vehicle c ON ac.vehicle_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ? AND a.id IN (%s)
		ORDER BY pc.name
	`, strings.Join(placeholders, ","))

	var args []interface{}
	args = append(args, makeName, year, model, engine)
	for _, id := range adIDs {
		args = append(args, id)
	}

	log.Printf("[GetCategoriesForAdIDs] Query: %s", query)
	log.Printf("[GetCategoriesForAdIDs] Args: %v", args)

	var categories []string
	err := db.Select(&categories, query, args...)
	log.Printf("[GetCategoriesForAdIDs] Result: %d categories, error: %v", len(categories), err)
	return categories, err
}

// GetSubCategoriesForAds returns subcategories for a specific make/year/model/engine/category, filtered by ad IDs
func GetSubCategoriesForAds(adIDs []int, makeName, year, model, engine, category string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	// URL decode the parameters
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)
	category, _ = url.QueryUnescape(category)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT psc.name
		FROM PartSubAdCategory psc
		JOIN PartCategory pc ON psc.part_category_id = pc.id
		JOIN Ad a ON psc.id = a.part_subcategory_id
		JOIN AdVehicle ac ON a.id = ac.ad_id
		JOIN Vehicle c ON ac.vehicle_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ? AND pc.name = ? AND a.id IN (%s)
		ORDER BY psc.name
	`, strings.Join(placeholders, ","))

	var args []interface{}
	args = append(args, makeName, year, model, engine, category)
	for _, id := range adIDs {
		args = append(args, id)
	}

	var subCategories []string
	err := db.Select(&subCategories, query, args...)
	return subCategories, err
}
