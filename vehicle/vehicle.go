package vehicle

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/part"
)

type Make struct {
	ID              int    `db:"id"`
	Name            string `db:"name"`
	ParentCompanyID *int   `db:"parent_company_id"`
}

type Model struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type Year struct {
	ID   int `db:"id"`
	Year int `db:"year"`
}

// Parent company information with country
type ParentCompany struct {
	ID      int    `db:"id"`
	Name    string `db:"name"`
	Country string `db:"country"`
}

// Parent company information with country
type ParentCompanyInfo struct {
	Name    string
	Country string
}

// Make with its parent company information
type MakeWithParentCompany struct {
	ID                int    `db:"id"`
	Name              string `db:"name"`
	ParentCompanyID   *int   `db:"parent_company_id"`
	ParentCompanyName string `db:"parent_company_name"`
}

var (
	// Cache for vehicle data
	vehicleCache *cache.Cache[[]string]
)

// Initialize vehicle cache and start background refresh
func InitVehicleCache() error {
	var err error

	// Cache for vehicle data
	vehicleCache, err = cache.New[[]string](func(value []string) int64 {
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

func GetMakes() []string {
	cacheKey := "makes:all"

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	query := "SELECT name FROM Make ORDER BY name"
	var makes []string
	err := db.Select(&makes, query)
	if err != nil {
		return nil
	}

	vehicleCache.Set(cacheKey, makes, int64(len(makes)*30))
	return makes
}

func GetYears(makeName string) []string {
	cacheKey := fmt.Sprintf("years:%s", makeName)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	query := `SELECT DISTINCT Year.year FROM Car
	JOIN Make ON Car.make_id = Make.id
	JOIN Year ON Car.year_id = Year.id
	WHERE Make.name = ? ORDER BY Year.year`
	var yearInts []int
	err := db.Select(&yearInts, query, makeName)
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

func GetModels(makeName string, years []string) []string {
	if len(years) == 0 {
		return []string{}
	}

	// Create cache key with years as provided
	cacheKey := fmt.Sprintf("models:%s:%s", makeName, strings.Join(years, ","))

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	// For multiple years, we want models that exist in ALL selected years (intersection)
	// Build a query that finds models present in every selected year
	query := `SELECT DISTINCT Model.name FROM Model
	WHERE Model.id IN (
		SELECT Car.model_id FROM Car
		JOIN Make ON Car.make_id = Make.id
		JOIN Year ON Car.year_id = Year.id
		WHERE Make.name = ? AND Year.year = ?
	)`

	// Add additional year conditions for intersection
	for i := 1; i < len(years); i++ {
		query += ` AND Model.id IN (
			SELECT Car.model_id FROM Car
			JOIN Make ON Car.make_id = Make.id
			JOIN Year ON Car.year_id = Year.id
			WHERE Make.name = ? AND Year.year = ?
		)`
	}

	query += ` ORDER BY Model.name`

	// Build args: makeName repeated for each year
	args := make([]interface{}, len(years)*2)
	for i, year := range years {
		args[i*2] = makeName
		args[i*2+1] = year
	}

	var models []string
	err := db.Select(&models, query, args...)
	if err != nil {
		return nil
	}

	vehicleCache.Set(cacheKey, models, int64(len(models)*20))
	return models
}

func GetEngines(makeName string, years []string, models []string) []string {
	if len(years) == 0 || len(models) == 0 {
		return []string{}
	}

	// Create cache key: engines:BMW:2020,2021:M3,X5
	cacheKey := fmt.Sprintf("engines:%s:%s:%s", makeName, strings.Join(years, ","), strings.Join(models, ","))

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached
	}

	// For multiple years/models, we want engines that exist in ALL combinations (intersection)
	// Build a query that finds engines present in every year-model combination
	query := `SELECT DISTINCT Engine.name FROM Engine
	WHERE Engine.id IN (
		SELECT Car.engine_id FROM Car
		JOIN Make ON Car.make_id = Make.id
		JOIN Model ON Car.model_id = Model.id
		JOIN Year ON Car.year_id = Year.id
		WHERE Make.name = ? AND Year.year = ? AND Model.name = ?
	)`

	// Add additional year-model conditions for intersection
	for i := 1; i < len(years); i++ {
		for j := 0; j < len(models); j++ {
			query += ` AND Engine.id IN (
				SELECT Car.engine_id FROM Car
				JOIN Make ON Car.make_id = Make.id
				JOIN Model ON Car.model_id = Model.id
				JOIN Year ON Car.year_id = Year.id
				WHERE Make.name = ? AND Year.year = ? AND Model.name = ?
			)`
		}
	}

	query += ` ORDER BY Engine.name`

	// Build args: makeName, year, model for each combination
	args := make([]interface{}, len(years)*len(models)*3)
	argIndex := 0
	for _, year := range years {
		for _, model := range models {
			args[argIndex] = makeName
			args[argIndex+1] = year
			args[argIndex+2] = model
			argIndex += 3
		}
	}

	var engines []string
	err := db.Select(&engines, query, args...)
	if err != nil {
		return nil
	}

	vehicleCache.Set(cacheKey, engines, int64(len(engines)*25))
	return engines
}

// ============================================================================
// CACHED FUNCTIONS FOR AD DATA (Tree View Browse Mode when q=="")
// ============================================================================

// GetAdMakes returns makes that have existing ads
func GetAdMakes() ([]string, error) {
	cacheKey := "ad:makes"

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	makes, err := part.GetMakesForAll()
	if err != nil {
		return nil, err
	}

	vehicleCache.Set(cacheKey, makes, int64(len(makes)*50))
	return makes, nil
}

// GetAdYears returns years that have existing ads for a make
func GetAdYears(makeName string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:years:%s", makeName)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	years, err := part.GetYearsForAll(makeName)
	if err != nil {
		return nil, err
	}

	vehicleCache.Set(cacheKey, years, int64(len(years)*10))
	return years, nil
}

// GetAdModels returns models that have existing ads for make/year
func GetAdModels(makeName, year string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:models:%s:%s", makeName, year)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	models, err := part.GetModelsForAll(makeName, year)
	if err != nil {
		return nil, err
	}

	vehicleCache.Set(cacheKey, models, int64(len(models)*30))
	return models, nil
}

// GetAdEngines returns engines that have existing ads for make/year/model
func GetAdEngines(makeName, year, model string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:engines:%s:%s:%s", makeName, year, model)

	if cached, found := vehicleCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	engines, err := part.GetEnginesForAll(makeName, year, model)
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

// GetVehicleCacheStats returns cache statistics for admin monitoring
func GetVehicleCacheStats() map[string]interface{} {
	if vehicleCache == nil {
		return map[string]interface{}{
			"cache_type": "Vehicle Data Cache",
			"error":      "Cache not initialized",
		}
	}
	return vehicleCache.Stats()
}

// ClearVehicleCache clears all items from the vehicle cache
func ClearVehicleCache() {
	if vehicleCache != nil {
		vehicleCache.Clear()
		log.Printf("[vehicle-cache] Cache cleared")
	}
}
