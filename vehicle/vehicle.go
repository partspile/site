package vehicle

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/db"
)

type ParentCompanyInfo struct {
	Name    string
	Country string
}

var vehicleCache *cache.Cache[[]string]

func InitVehicleCache() error {
	var err error

	vehicleCache, err = cache.New(func(value []string) int64 {
		return int64(len(value) * 30)
	}, "Vehicle Data Cache")
	if err != nil {
		return err
	}

	log.Printf("[vehicle-cache] Cache initialized successfully")

	return nil
}

// ============================================================================
// CACHED FUNCTIONS FOR STATIC VEHICLE DATA
// ============================================================================

func GetMakes(adCat string) []string {
	adCatID := ad.GetAdCategoryID(adCat)
	cacheKey := fmt.Sprintf("makes:%d", adCatID)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	query, args := buildMakesQuery(adCatID)
	var makes []string
	err := db.Select(&makes, query, args...)
	if err != nil {
		return nil
	}

	vehicleCache.Set(cacheKey, makes, int64(len(makes)*30))
	return makes
}

func GetYears(adCat string, makeName string) []string {
	adCatID := ad.GetAdCategoryID(adCat)
	cacheKey := fmt.Sprintf("years:%d:%s", adCatID, makeName)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	query, args := buildYearsQuery(adCatID, makeName)
	var yearInts []int
	err := db.Select(&yearInts, query, args...)
	if err != nil {
		return nil
	}
	var years []string
	for _, year := range yearInts {
		years = append(years, strconv.Itoa(year))
	}

	vehicleCache.Set(cacheKey, years, int64(len(years)*5))
	return years
}

func GetModels(adCat string, makeName string, years []string) []string {
	if len(years) == 0 {
		return []string{}
	}

	adCatID := ad.GetAdCategoryID(adCat)
	// Create cache key with years as provided
	cacheKey := fmt.Sprintf("models:%d:%s:%s", adCatID, makeName, strings.Join(years, ","))

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	query, args := buildModelsQuery(adCatID, makeName, years)
	var models []string
	err := db.Select(&models, query, args...)
	if err != nil {
		return nil
	}

	vehicleCache.Set(cacheKey, models, int64(len(models)*20))
	return models
}

func GetEngines(adCat string, makeName string, years []string, models []string) []string {
	if len(years) == 0 || len(models) == 0 {
		return []string{}
	}

	adCatID := ad.GetAdCategoryID(adCat)
	// Create cache key: engines:BMW:2020,2021:M3,X5
	cacheKey := fmt.Sprintf("engines:%d:%s:%s:%s", adCatID, makeName, strings.Join(years, ","), strings.Join(models, ","))

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	query, args := buildEnginesQuery(adCatID, makeName, years, models)
	var engines []string
	err := db.Select(&engines, query, args...)
	if err != nil {
		return nil
	}

	vehicleCache.Set(cacheKey, engines, int64(len(engines)*25))
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
	query := `SELECT DISTINCT Year.year FROM Car
	JOIN Make ON Car.make_id = Make.id
	JOIN Year ON Car.year_id = Year.id
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
		SELECT Car.model_id FROM Car
		JOIN Make ON Car.make_id = Make.id
		JOIN Year ON Car.year_id = Year.id
		WHERE Make.name = ? AND Year.year = ? AND Make.ad_category_id = ? AND Year.ad_category_id = ? AND Model.ad_category_id = ?
	)`

	// Add additional year conditions for intersection
	for i := 1; i < len(years); i++ {
		query += ` AND Model.id IN (
			SELECT Car.model_id FROM Car
			JOIN Make ON Car.make_id = Make.id
			JOIN Year ON Car.year_id = Year.id
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
		SELECT Car.engine_id FROM Car
		JOIN Make ON Car.make_id = Make.id
		JOIN Model ON Car.model_id = Model.id
		JOIN Year ON Car.year_id = Year.id
		WHERE Make.name = ? AND Year.year = ? AND Model.name = ? AND Make.ad_category_id = ? AND Year.ad_category_id = ? AND Model.ad_category_id = ? AND Engine.ad_category_id = ?
	)`

	// Add additional year-model conditions for intersection
	// Skip the first combination since it's already in the base query
	combinationCount := 0
	for i := 0; i < len(years); i++ {
		for j := 0; j < len(models); j++ {
			if combinationCount > 0 {
				query += ` AND Engine.id IN (
					SELECT Car.engine_id FROM Car
					JOIN Make ON Car.make_id = Make.id
					JOIN Model ON Car.model_id = Model.id
					JOIN Year ON Car.year_id = Year.id
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
func GetAdMakes(adCat string) ([]string, error) {
	adCatID := ad.GetAdCategoryID(adCat)
	cacheKey := fmt.Sprintf("ad:makes:%d", adCatID)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	makes, err := getMakesForAll(adCatID)
	if err != nil {
		return nil, err
	}

	vehicleCache.Set(cacheKey, makes, int64(len(makes)*50))
	return makes, nil
}

// GetAdYears returns years that have existing ads for a make
func GetAdYears(adCat string, makeName string) ([]string, error) {
	adCatID := ad.GetAdCategoryID(adCat)
	cacheKey := fmt.Sprintf("ad:years:%d:%s", adCatID, makeName)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	years, err := getYearsForAll(adCatID, makeName)
	if err != nil {
		return nil, err
	}

	vehicleCache.Set(cacheKey, years, int64(len(years)*10))
	return years, nil
}

// GetAdModels returns models that have existing ads for make/year
func GetAdModels(adCat string, makeName, year string) ([]string, error) {
	adCatID := ad.GetAdCategoryID(adCat)
	cacheKey := fmt.Sprintf("ad:models:%d:%s:%s", adCatID, makeName, year)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	models, err := getModelsForAll(adCatID, makeName, year)
	if err != nil {
		return nil, err
	}

	vehicleCache.Set(cacheKey, models, int64(len(models)*30))
	return models, nil
}

// GetAdEngines returns engines that have existing ads for make/year/model
func GetAdEngines(adCat string, makeName, year, model string) ([]string, error) {
	adCatID := ad.GetAdCategoryID(adCat)
	cacheKey := fmt.Sprintf("ad:engines:%d:%s:%s:%s", adCatID, makeName, year, model)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	engines, err := getEnginesForAll(adCatID, makeName, year, model)
	if err != nil {
		return nil, err
	}

	vehicleCache.Set(cacheKey, engines, int64(len(engines)*25))
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
func GetAdMakesForAdIDs(adIDs []int) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	// Generate deterministic cache key
	cacheKey := genIDsKey(adIDs, "search:makes")

	// Check cache first
	makes, found := vehicleCache.Get(cacheKey)
	if found {
		log.Printf("[search-cache] Cache hit for makes: %s", cacheKey)
		return makes, nil
	}

	log.Printf("[search-cache] Cache miss for makes: %s", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT m.name
		FROM Make m
		JOIN Car c ON m.id = c.make_id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE ac.ad_id IN (%s)
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
	vehicleCache.Set(cacheKey, makes, int64(len(makes)*40))
	log.Printf("[search-cache] Cached %d makes for key: %s", len(makes), cacheKey)

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
	if cached, found := vehicleCache.Get(cacheKey); found {
		log.Printf("[search-cache] Cache hit for years: %s", cacheKey)
		return cached, nil
	}

	log.Printf("[search-cache] Cache miss for years: %s", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT y.year
		FROM Year y
		JOIN Car c ON y.id = c.year_id
		JOIN Make m ON c.make_id = m.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ? AND ac.ad_id IN (%s)
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
	vehicleCache.Set(cacheKey, years, int64(len(years)*15))
	log.Printf("[search-cache] Cached %d years for key: %s", len(years), cacheKey)

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
	if cached, found := vehicleCache.Get(cacheKey); found {
		log.Printf("[search-cache] Cache hit for models: %s", cacheKey)
		return cached, nil
	}

	log.Printf("[search-cache] Cache miss for models: %s", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT mo.name
		FROM Model mo
		JOIN Car c ON mo.id = c.model_id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ? AND y.year = ? AND ac.ad_id IN (%s)
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
	vehicleCache.Set(cacheKey, models, int64(len(models)*25))
	log.Printf("[search-cache] Cached %d models for key: %s", len(models), cacheKey)

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
	if cached, found := vehicleCache.Get(cacheKey); found {
		log.Printf("[search-cache] Cache hit for engines: %s", cacheKey)
		return cached, nil
	}

	log.Printf("[search-cache] Cache miss for engines: %s", cacheKey)

	// Create placeholders for the IN clause
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT e.name
		FROM Engine e
		JOIN Car c ON e.id = c.engine_id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND ac.ad_id IN (%s)
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
	vehicleCache.Set(cacheKey, engines, int64(len(engines)*30))
	log.Printf("[search-cache] Cached %d engines for key: %s", len(engines), cacheKey)

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
	return vehicleCache.Stats()
}

// ClearCache clears all items from the vehicle cache and returns updated stats
func ClearCache() map[string]any {
	vehicleCache.Clear()
	log.Printf("[vehicle-cache] Cache cleared")
	return vehicleCache.Stats()
}

// ============================================================================
// DATABASE QUERY FUNCTIONS (Internal - used by cached functions above)
// ============================================================================

// getMakesForAll returns all makes that have ads for a specific category
func getMakesForAll(adCat int) ([]string, error) {
	query := `
		SELECT DISTINCT m.name
		FROM Make m
		JOIN Car c ON m.id = c.make_id
		JOIN AdCar ac ON c.id = ac.car_id
		JOIN Ad a ON ac.ad_id = a.id
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
		JOIN Car c ON y.id = c.year_id
		JOIN Make m ON c.make_id = m.id
		JOIN AdCar ac ON c.id = ac.car_id
		JOIN Ad a ON ac.ad_id = a.id
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
		JOIN Car c ON mo.id = c.model_id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN AdCar ac ON c.id = ac.car_id
		JOIN Ad a ON ac.ad_id = a.id
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
		JOIN Car c ON e.id = c.engine_id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN AdCar ac ON c.id = ac.car_id
		JOIN Ad a ON ac.ad_id = a.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND a.ad_category_id = ?
		ORDER BY e.name
	`
	var engines []string
	err := db.Select(&engines, query, makeName, year, model, adCat)
	return engines, err
}
