package spec

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"

	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

type ParentCompanyInfo struct {
	Name    string
	Country string
}

var specCache *cache.Cache[[]string]

func InitSpecCache() error {
	var err error

	specCache, err = cache.New(func(value []string) int64 {
		return int64(len(value) * 30)
	}, "Spec Data Cache")
	if err != nil {
		return err
	}

	logger.Info("Cache initialized successfully",
		"component", "spec-cache")

	return nil
}

// ============================================================================
// CACHED FUNCTIONS FOR STATIC SPEC DATA
// ============================================================================

func GetMakes(adCat int) []string {
	cacheKey := fmt.Sprintf("makes:%d", adCat)

	if cached, found := specCache.Get(cacheKey); found {
		return cached
	}

	// Use dynamic GetFieldValues
	values, err := GetFieldValues(adCat, "make", map[string]string{})
	if err != nil {
		logger.Warn("Failed to get makes dynamically",
			"error", err, "component", "spec", "adCat", adCat)
		return []string{}
	}

	specCache.Set(cacheKey, values, int64(len(values)*30))
	return values
}

func GetYears(adCat int, makeName string) []string {
	cacheKey := fmt.Sprintf("years:%d:%s", adCat, makeName)

	if cached, found := specCache.Get(cacheKey); found {
		return cached
	}

	// Use dynamic GetFieldValues
	parentValues := map[string]string{"make": makeName}
	values, err := GetFieldValues(adCat, "year", parentValues)
	if err != nil {
		logger.Warn("Failed to get years dynamically",
			"error", err, "component", "spec", "adCat", adCat, "make", makeName)
		return []string{}
	}

	specCache.Set(cacheKey, values, int64(len(values)*5))
	return values
}

func GetModels(adCat int, makeName string, years []string) []string {
	if len(years) == 0 {
		return []string{}
	}

	// Create cache key with years as provided
	cacheKey := fmt.Sprintf("models:%d:%s:%s", adCat, makeName, strings.Join(years, ","))

	if cached, found := specCache.Get(cacheKey); found {
		return cached
	}

	// Use dynamic GetFieldValues with intersection logic for multiple years
	parentValuesArray := map[string][]string{"make": {makeName}, "year": years}
	query, args, err := BuildFieldQueryWithIntersection(adCat, "model", parentValuesArray)
	if err != nil {
		logger.Warn("Failed to build models query",
			"error", err, "component", "spec", "adCat", adCat, "make", makeName)
		return []string{}
	}

	var values []string
	err = db.Select(&values, query, args...)
	if err != nil {
		logger.Warn("Failed to query models",
			"error", err, "component", "spec", "adCat", adCat)
		return []string{}
	}

	specCache.Set(cacheKey, values, int64(len(values)*20))
	return values
}

func GetEngines(adCat int, makeName string, years []string, models []string) []string {
	if len(years) == 0 || len(models) == 0 {
		return []string{}
	}

	// Create cache key: engines:BMW:2020,2021:M3,X5
	cacheKey := fmt.Sprintf("engines:%d:%s:%s:%s", adCat, makeName, strings.Join(years, ","), strings.Join(models, ","))

	if cached, found := specCache.Get(cacheKey); found {
		return cached
	}

	// Use dynamic GetFieldValues with intersection logic
	parentValuesArray := map[string][]string{
		"make":  {makeName},
		"year":  years,
		"model": models,
	}
	query, args, err := BuildFieldQueryWithIntersection(adCat, "engine", parentValuesArray)
	if err != nil {
		logger.Warn("Failed to build engines query",
			"error", err, "component", "spec", "adCat", adCat)
		return []string{}
	}

	var engines []string
	err = db.Select(&engines, query, args...)
	if err != nil {
		logger.Warn("Failed to query engines",
			"error", err, "component", "spec", "adCat", adCat)
		return []string{}
	}

	specCache.Set(cacheKey, engines, int64(len(engines)*25))
	return engines
}

// ============================================================================
// SQL QUERY BUILDERS (Internal helper functions)
// ============================================================================

// buildMakesQuery builds the SQL query for getting all makes for a category
func buildMakesQuery(adCat int) (string, []interface{}) {
	query := "SELECT name FROM Make WHERE ad_category_id = ? ORDER BY name"
	args := []interface{}{adCat}
	return query, args
}

// buildYearsQuery builds the SQL query for getting years for a specific make and category
func buildYearsQuery(adCat int, makeName string) (string, []interface{}) {
	query := `SELECT DISTINCT Year.year FROM Specification spec
	JOIN Make ON spec.make_id = Make.id
	JOIN Year ON spec.year_id = Year.id
	WHERE Make.name = ? AND Make.ad_category_id = ? AND Year.ad_category_id = ?
	ORDER BY Year.year`
	args := []interface{}{makeName, adCat, adCat}
	return query, args
}

// buildModelsQuery builds the SQL query for finding models that exist in ALL selected years (intersection)
func buildModelsQuery(adCat int, makeName string, years []string) (string, []interface{}) {
	// For multiple years, we want models that exist in ALL selected years (intersection)
	// Build a query that finds models present in every selected year
	query := `SELECT DISTINCT Model.name FROM Model
	WHERE Model.id IN (
		SELECT spec.model_id FROM Specification spec
		JOIN Make ON spec.make_id = Make.id
		JOIN Year ON spec.year_id = Year.id
		WHERE Make.name = ? AND Year.year = ? AND Make.ad_category_id = ? AND Year.ad_category_id = ? AND Model.ad_category_id = ?
	)`

	// Add additional year conditions for intersection
	for i := 1; i < len(years); i++ {
		query += ` AND Model.id IN (
			SELECT spec.model_id FROM Specification spec
			JOIN Make ON spec.make_id = Make.id
			JOIN Year ON spec.year_id = Year.id
			WHERE Make.name = ? AND Year.year = ? AND Make.ad_category_id = ? AND Year.ad_category_id = ? AND Model.ad_category_id = ?
		)`
	}

	query += ` ORDER BY Model.name`

	// Build args: makeName, year, category_id repeated for each year
	args := make([]interface{}, len(years)*5)
	for i, year := range years {
		args[i*5] = makeName
		args[i*5+1] = year
		args[i*5+2] = adCat
		args[i*5+3] = adCat
		args[i*5+4] = adCat
	}

	return query, args
}

// buildEnginesQuery builds the SQL query for finding engines that exist in ALL year-model combinations (intersection)
func buildEnginesQuery(adCat int, makeName string, years []string, models []string) (string, []interface{}) {
	// For multiple years/models, we want engines that exist in ALL year-model combinations (intersection)
	// We need to find engines that exist for every combination of selected years and models

	// Build a query that finds engines present in every year-model combination
	// The logic: engine must exist in (year1, model1) AND (year1, model2) AND ... AND (year2, model1) AND (year2, model2) AND ...

	query := `SELECT DISTINCT Engine.name FROM Engine
	WHERE Engine.id IN (
		SELECT spec.engine_id FROM Specification spec
		JOIN Make ON spec.make_id = Make.id
		JOIN Model ON spec.model_id = Model.id
		JOIN Year ON spec.year_id = Year.id
		WHERE Make.name = ? AND Year.year = ? AND Model.name = ? AND Make.ad_category_id = ? AND Year.ad_category_id = ? AND Model.ad_category_id = ? AND Engine.ad_category_id = ?
	)`

	// Add additional year-model conditions for intersection
	// Skip the first combination since it's already in the base query
	combinationCount := 0
	for i := 0; i < len(years); i++ {
		for j := 0; j < len(models); j++ {
			if combinationCount > 0 {
				query += ` AND Engine.id IN (
					SELECT spec.engine_id FROM Specification spec
					JOIN Make ON spec.make_id = Make.id
					JOIN Model ON spec.model_id = Model.id
					JOIN Year ON spec.year_id = Year.id
					WHERE Make.name = ? AND Year.year = ? AND Model.name = ? AND Make.ad_category_id = ? AND Year.ad_category_id = ? AND Model.ad_category_id = ? AND Engine.ad_category_id = ?
				)`
			}
			combinationCount++
		}
	}

	query += ` ORDER BY Engine.name`

	// Build args: makeName, year, model, category_id for each combination
	args := make([]interface{}, len(years)*len(models)*7)
	argIndex := 0
	for _, year := range years {
		for _, model := range models {
			args[argIndex] = makeName
			args[argIndex+1] = year
			args[argIndex+2] = model
			args[argIndex+3] = adCat
			args[argIndex+4] = adCat
			args[argIndex+5] = adCat
			args[argIndex+6] = adCat
			argIndex += 7
		}
	}

	return query, args
}

// ============================================================================
// CACHED FUNCTIONS FOR AD DATA (Tree View Browse Mode when q=="")
// ============================================================================

// GetAdMakes returns makes that have existing ads
func GetAdMakes(adCat int) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:makes:%d", adCat)

	if cached, found := specCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	makes, err := getMakesForAll(adCat)
	if err != nil {
		return nil, err
	}

	specCache.Set(cacheKey, makes, int64(len(makes)*50))
	return makes, nil
}

// GetAdYears returns years that have existing ads for a make
func GetAdYears(adCat int, makeName string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:years:%d:%s", adCat, makeName)

	if cached, found := specCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	years, err := getYearsForAll(adCat, makeName)
	if err != nil {
		return nil, err
	}

	specCache.Set(cacheKey, years, int64(len(years)*10))
	return years, nil
}

// GetAdModels returns models that have existing ads for make/year
func GetAdModels(adCat int, makeName, year string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:models:%d:%s:%s", adCat, makeName, year)

	if cached, found := specCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	models, err := getModelsForAll(adCat, makeName, year)
	if err != nil {
		return nil, err
	}

	specCache.Set(cacheKey, models, int64(len(models)*30))
	return models, nil
}

// GetAdEngines returns engines that have existing ads for make/year/model
func GetAdEngines(adCat int, makeName, year, model string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:engines:%d:%s:%s:%s", adCat, makeName, year, model)

	if cached, found := specCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	engines, err := getEnginesForAll(adCat, makeName, year, model)
	if err != nil {
		return nil, err
	}

	specCache.Set(cacheKey, engines, int64(len(engines)*25))
	return engines, nil
}

// ============================================================================
// FUNCTIONS FOR AD DATA (Tree View Search Mode when q!="")
// ============================================================================

// genIDsKey creates a consistent cache key for adID sets
func genIDsKey(adIDs []int, operation string) string {
	if len(adIDs) == 0 {
		return fmt.Sprintf("%s:empty", operation)
	}

	// Hash-based (collision-resistant)
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%s:", operation))) // Namespace and operation type

	// Write count first for uniqueness
	var countBuf [4]byte
	binary.LittleEndian.PutUint32(countBuf[:], uint32(len(adIDs)))
	hash.Write(countBuf[:])

	// Write each ID in consistent binary format
	for _, id := range adIDs {
		binary.LittleEndian.PutUint32(countBuf[:], uint32(id))
		hash.Write(countBuf[:])
	}

	sum := hash.Sum(nil)
	return fmt.Sprintf("%s:%x", operation, sum[:16]) // 128-bit key
}

// GetAdMakesForAdIDs returns makes filtered by the provided ad IDs
// Maintains backward compatibility - internally uses generic function
func GetAdMakesForAdIDs(adIDs []int) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	// For makes, we need adCategoryID - get it from first ad
	// Since we don't have adCategoryID here, we'll use the old implementation
	// TODO: Consider passing adCategoryID as parameter in future
	// Generate deterministic cache key
	cacheKey := genIDsKey(adIDs, "search:makes")

	// Check cache first
	makes, found := specCache.Get(cacheKey)
	if found {
		logger.Debug("Cache hit for makes",
			"component", "search-cache", "cacheKey", cacheKey)
		return makes, nil
	}

	logger.Debug("Cache miss for makes",
		"component", "search-cache", "cacheKey", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT m.name
		FROM Make m
		JOIN Specification spec ON m.id = spec.make_id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		WHERE ad_spec.ad_id IN (%s)
		ORDER BY m.name
	`, strings.Join(placeholders, ","))

	var args []interface{}
	for _, id := range adIDs {
		args = append(args, id)
	}

	err := db.Select(&makes, query, args...)
	if err != nil {
		return nil, err
	}

	// Cache the result
	specCache.Set(cacheKey, makes, int64(len(makes)*40))
	logger.Debug("Cached makes",
		"component", "search-cache", "count", len(makes),
		"cacheKey", cacheKey)

	return makes, nil
}

// GetAdYearsForAdIDs returns years for a specific make, filtered by ad IDs
func GetAdYearsForAdIDs(adIDs []int, makeName string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	makeName, _ = url.QueryUnescape(makeName)

	// Generate deterministic cache key with make name
	cacheKey := genIDsKey(adIDs, fmt.Sprintf("search:years:%s", makeName))

	// Check cache first
	if cached, found := specCache.Get(cacheKey); found {
		logger.Debug("Cache hit for years",
			"component", "search-cache", "cacheKey", cacheKey)
		return cached, nil
	}

	logger.Debug("Cache miss for years",
		"component", "search-cache", "cacheKey", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT y.year
		FROM Year y
		JOIN Specification spec ON y.id = spec.year_id
		JOIN Make m ON spec.make_id = m.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		WHERE m.name = ? AND ad_spec.ad_id IN (%s)
		ORDER BY y.year DESC
	`, strings.Join(placeholders, ","))

	var args []interface{}
	args = append(args, makeName)
	for _, id := range adIDs {
		args = append(args, id)
	}

	var yearInts []int
	err := db.Select(&yearInts, query, args...)
	if err != nil {
		return nil, err
	}

	var years []string
	for _, year := range yearInts {
		years = append(years, fmt.Sprintf("%d", year))
	}

	// Cache the result
	specCache.Set(cacheKey, years, int64(len(years)*15))
	specCache.Set(cacheKey, years, int64(len(years)*20))
	logger.Debug("Cached years",
		"component", "search-cache", "count", len(years),
		"cacheKey", cacheKey)

	return years, nil
}

// GetAdModelsForAdIDs returns models for a specific make/year, filtered by ad IDs
func GetAdModelsForAdIDs(adIDs []int, makeName, year string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)

	// Generate deterministic cache key with make and year
	cacheKey := genIDsKey(adIDs, fmt.Sprintf("search:models:%s:%s", makeName, year))

	// Check cache first
	if cached, found := specCache.Get(cacheKey); found {
		logger.Debug("Cache hit for models",
			"component", "search-cache", "cacheKey", cacheKey)
		return cached, nil
	}

	logger.Debug("Cache miss for models",
		"component", "search-cache", "cacheKey", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT mo.name
		FROM Model mo
		JOIN Specification spec ON mo.id = spec.model_id
		JOIN Make m ON spec.make_id = m.id
		JOIN Year y ON spec.year_id = y.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		WHERE m.name = ? AND y.year = ? AND ad_spec.ad_id IN (%s)
		ORDER BY mo.name
	`, strings.Join(placeholders, ","))

	var args []interface{}
	args = append(args, makeName, year)
	for _, id := range adIDs {
		args = append(args, id)
	}

	var models []string
	err := db.Select(&models, query, args...)
	if err != nil {
		return nil, err
	}

	// Cache the result
	specCache.Set(cacheKey, models, int64(len(models)*25))
	logger.Debug("Cached models",
		"component", "search-cache", "count", len(models),
		"cacheKey", cacheKey)

	return models, nil
}

// GetAdEnginesForAdIDs returns engines for a specific make/year/model, filtered by ad IDs
func GetAdEnginesForAdIDs(adIDs []int, makeName, year, model string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)

	// Generate deterministic cache key with make, year, and model
	cacheKey := genIDsKey(adIDs, fmt.Sprintf("search:engines:%s:%s:%s", makeName, year, model))

	// Check cache first
	if cached, found := specCache.Get(cacheKey); found {
		logger.Debug("Cache hit for engines",
			"component", "search-cache", "cacheKey", cacheKey)
		return cached, nil
	}

	logger.Debug("Cache miss for engines",
		"component", "search-cache", "cacheKey", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT e.name
		FROM Engine e
		JOIN Specification spec ON e.id = spec.engine_id
		JOIN Make m ON spec.make_id = m.id
		JOIN Year y ON spec.year_id = y.id
		JOIN Model mo ON spec.model_id = mo.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND ad_spec.ad_id IN (%s)
		ORDER BY e.name
	`, strings.Join(placeholders, ","))

	var args []interface{}
	args = append(args, makeName, year, model)
	for _, id := range adIDs {
		args = append(args, id)
	}

	var engines []string
	err := db.Select(&engines, query, args...)
	if err != nil {
		return nil, err
	}

	// Cache the result
	specCache.Set(cacheKey, engines, int64(len(engines)*30))
	logger.Debug("Cached engines",
		"component", "search-cache", "count", len(engines),
		"cacheKey", cacheKey)

	return engines, nil
}

// GetParentCompanyInfoForMake returns the parent company information for a given make
func GetParentCompanyInfoForMake(makeName string) (*ParentCompanyInfo, error) {
	rows, err := db.Query(`
		SELECT pc.name, pc.country
		FROM ParentCompany pc
		JOIN Make m ON pc.id = m.parent_company_id
		WHERE m.name = ?
	`, makeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var pcInfo ParentCompanyInfo
		if err := rows.Scan(&pcInfo.Name, &pcInfo.Country); err != nil {
			return nil, err
		}
		return &pcInfo, nil
	}
	return nil, nil
}

// GetCacheStats returns cache statistics for admin monitoring
func GetCacheStats() map[string]any {
	return specCache.Stats()
}

// ClearCache clears all items from the spec cache and returns updated stats
func ClearCache() map[string]any {
	specCache.Clear()
	logger.Info("Cache cleared",
		"component", "spec-cache")
	return specCache.Stats()
}

// ============================================================================
// DATABASE QUERY FUNCTIONS (Internal - used by cached functions above)
// ============================================================================

// getMakesForAll returns all makes that have ads for a specific category
func getMakesForAll(adCat int) ([]string, error) {
	query := `
		SELECT DISTINCT m.name
		FROM Make m
		JOIN Specification spec ON m.id = spec.make_id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE a.ad_category_id = ?
		ORDER BY m.name
	`
	var makes []string
	err := db.Select(&makes, query, adCat)
	return makes, err
}

// getYearsForAll returns all years for a specific make and category
func getYearsForAll(adCat int, makeName string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	query := `
		SELECT DISTINCT y.year
		FROM Year y
		JOIN Specification spec ON y.id = spec.year_id
		JOIN Make m ON spec.make_id = m.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE m.name = ? AND a.ad_category_id = ?
		ORDER BY y.year DESC
	`
	var yearInts []int
	err := db.Select(&yearInts, query, makeName, adCat)
	if err != nil {
		return nil, err
	}

	var years []string
	for _, year := range yearInts {
		years = append(years, fmt.Sprintf("%d", year))
	}
	return years, nil
}

// getModelsForAll returns all models for a specific make/year and category
func getModelsForAll(adCat int, makeName, year string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	query := `
		SELECT DISTINCT mo.name
		FROM Model mo
		JOIN Specification spec ON mo.id = spec.model_id
		JOIN Make m ON spec.make_id = m.id
		JOIN Year y ON spec.year_id = y.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE m.name = ? AND y.year = ? AND a.ad_category_id = ?
		ORDER BY mo.name
	`
	var models []string
	err := db.Select(&models, query, makeName, year, adCat)
	return models, err
}

// getEnginesForAll returns all engines for a specific make/year/model and category
func getEnginesForAll(adCat int, makeName, year, model string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	query := `
		SELECT DISTINCT e.name
		FROM Engine e
		JOIN Specification spec ON e.id = spec.engine_id
		JOIN Make m ON spec.make_id = m.id
		JOIN Year y ON spec.year_id = y.id
		JOIN Model mo ON spec.model_id = mo.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND a.ad_category_id = ?
		ORDER BY e.name
	`
	var engines []string
	err := db.Select(&engines, query, makeName, year, model, adCat)
	return engines, err
}
