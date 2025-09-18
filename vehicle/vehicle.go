package vehicle

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

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

// ParentCompany represents a parent company of a make
type ParentCompany struct {
	ID      int    `db:"id"`
	Name    string `db:"name"`
	Country string `db:"country"`
}

var (
	makesCache          []string
	yearsCache          = make(map[string][]string)
	allModelsCache      []string
	allEngineSizesCache []string
	yearRangeCache      []string

	// Cache for dynamic ad data (makes/years/models/engines with existing ads)
	adCache              *cache.Cache[[]string]
	cacheRefreshInterval = 30 * time.Minute
)

// Initialize vehicle cache and start background refresh
func InitVehicleCache() error {
	var err error

	// Cache for dynamic ad data
	adCache, err = cache.New[[]string](func(value []string) int64 {
		return int64(len(value) * 30)
	}, "Vehicle Ad Data Cache")
	if err != nil {
		return err
	}

	// Populate cache immediately on startup
	if err := refreshVehicleAdData(); err != nil {
		log.Printf("[vehicle-cache] Warning: Failed to populate cache on startup: %v", err)
		// Don't return error - let the background refresh handle it
	} else {
		log.Printf("[vehicle-cache] Cache populated successfully on startup")
	}

	// Start background refresh for ad data
	go refreshVehicleAdDataPeriodically()

	return nil
}

// Background goroutine that refreshes vehicle ad data every 30 minutes
func refreshVehicleAdDataPeriodically() {
	ticker := time.NewTicker(cacheRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("[vehicle-cache] Starting periodic vehicle ad data refresh")
			if err := refreshVehicleAdData(); err != nil {
				log.Printf("[vehicle-cache] Error refreshing vehicle ad data: %v", err)
			} else {
				log.Printf("[vehicle-cache] Vehicle ad data refresh completed successfully")
			}
		}
	}
}

// Refresh vehicle ad data (makes/years/models/engines with existing ads)
func refreshVehicleAdData() error {
	// Refresh makes with existing ads
	makes, err := GetAdMakes()
	if err != nil {
		return fmt.Errorf("failed to refresh ad makes: %w", err)
	}
	adCache.Set("ad:makes", makes, int64(len(makes)*50))

	log.Printf("[vehicle-cache] Vehicle ad data refreshed - %d makes with existing ads", len(makes))
	return nil
}

func GetMakes() []string {
	if makesCache != nil {
		return makesCache
	}
	query := "SELECT name FROM Make ORDER BY name"
	var makes []string
	err := db.Select(&makes, query)
	if err != nil {
		return nil
	}
	makesCache = makes
	return makes
}

func GetAllMakes() ([]Make, error) {
	query := "SELECT id, name, parent_company_id FROM Make ORDER BY name"
	var makes []Make
	err := db.Select(&makes, query)
	return makes, err
}

// MakeWithParentCompany represents a make with its parent company information
type MakeWithParentCompany struct {
	ID                int    `db:"id"`
	Name              string `db:"name"`
	ParentCompanyID   *int   `db:"parent_company_id"`
	ParentCompanyName string `db:"parent_company_name"`
}

func GetYears(makeName string) []string {
	if years, ok := yearsCache[makeName]; ok {
		return years
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
	yearsCache[makeName] = years
	return years
}

func GetAllModels() []string {
	if allModelsCache != nil {
		return allModelsCache
	}
	query := "SELECT DISTINCT name FROM Model ORDER BY name"
	var models []string
	err := db.Select(&models, query)
	if err != nil {
		return nil
	}
	allModelsCache = models
	return models
}

func GetModelsWithAvailability(makeName string, years []string) map[string]bool {
	allModels := make(map[string]bool)
	availableInAllYears := make(map[string]bool)

	if len(years) == 0 {
		return allModels
	}

	// Build placeholders for IN clause
	yearPlaceholders := make([]string, len(years))
	args := make([]interface{}, len(years)+1)
	args[0] = makeName
	for i, year := range years {
		yearPlaceholders[i] = "?"
		args[i+1] = year
	}
	query := `SELECT DISTINCT Model.name FROM Car
	JOIN Make ON Car.make_id = Make.id
	JOIN Model ON Car.model_id = Model.id
	JOIN Year ON Car.year_id = Year.id
	WHERE Make.name = ? AND Year.year IN (` + strings.Join(yearPlaceholders, ",") + `) ORDER BY Model.name`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var model string
		rows.Scan(&model)
		allModels[model] = true
		if _, exists := availableInAllYears[model]; !exists {
			availableInAllYears[model] = true
		}
	}

	// Second pass: check if each model exists in all selected years
	for model := range allModels {
		for _, year := range years {
			var count int
			err := db.QueryRow(`SELECT COUNT(*) FROM Car
			JOIN Make ON Car.make_id = Make.id
			JOIN Model ON Car.model_id = Model.id
			JOIN Year ON Car.year_id = Year.id
			WHERE Make.name = ? AND Model.name = ? AND Year.year = ?`, makeName, model, year).Scan(&count)
			if err != nil {
				return nil
			}
			if count == 0 {
				availableInAllYears[model] = false
				break
			}
		}
	}
	return availableInAllYears
}

func GetEnginesWithAvailability(makeName string, years []string, models []string) map[string]bool {
	allEngines := make(map[string]bool)
	availableInAllCombos := make(map[string]bool)

	if len(years) == 0 || len(models) == 0 {
		return allEngines
	}

	// Build placeholders for IN clauses
	yearPlaceholders := make([]string, len(years))
	modelPlaceholders := make([]string, len(models))
	args := make([]interface{}, 0, 1+len(years)+len(models))
	args = append(args, makeName)
	for i, year := range years {
		yearPlaceholders[i] = "?"
		args = append(args, year)
	}
	for i, model := range models {
		modelPlaceholders[i] = "?"
		args = append(args, model)
	}
	query := `SELECT DISTINCT Engine.name FROM Car
	JOIN Make ON Car.make_id = Make.id
	JOIN Model ON Car.model_id = Model.id
	JOIN Year ON Car.year_id = Year.id
	JOIN Engine ON Car.engine_id = Engine.id
	WHERE Make.name = ? AND Year.year IN (` + strings.Join(yearPlaceholders, ",") + `) AND Model.name IN (` + strings.Join(modelPlaceholders, ",") + `) ORDER BY Engine.name`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var engine string
		rows.Scan(&engine)
		allEngines[engine] = true
		if _, exists := availableInAllCombos[engine]; !exists {
			availableInAllCombos[engine] = true
		}
	}

	// Second pass: check if each engine exists for all selected year-model combinations
	for engine := range allEngines {
		for _, year := range years {
			for _, model := range models {
				var count int
				err := db.QueryRow(`SELECT COUNT(*) FROM Car
				JOIN Make ON Car.make_id = Make.id
				JOIN Model ON Car.model_id = Model.id
				JOIN Year ON Car.year_id = Year.id
				JOIN Engine ON Car.engine_id = Engine.id
				WHERE Make.name = ? AND Model.name = ? AND Year.year = ? AND Engine.name = ?`, makeName, model, year, engine).Scan(&count)
				if err != nil {
					return nil
				}
				if count == 0 {
					availableInAllCombos[engine] = false
					break
				}
			}
			if !availableInAllCombos[engine] {
				break
			}
		}
	}
	return availableInAllCombos
}

// ============================================================================
// CACHED FUNCTIONS FOR AD DATA (Tree View)
// ============================================================================

// GetAdMakes returns makes that have existing ads (for tree view)
func GetAdMakes() ([]string, error) {
	cacheKey := "ad:makes"

	if cached, found := adCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	makes, err := part.GetMakesForAll() // This JOINs with AdCar
	if err != nil {
		return nil, err
	}

	adCache.Set(cacheKey, makes, int64(len(makes)*50))
	return makes, nil
}

// GetAdYears returns years that have existing ads for a make (for tree view)
func GetAdYears(makeName string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:years:%s", makeName)

	if cached, found := adCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	years, err := part.GetYearsForAll(makeName)
	if err != nil {
		return nil, err
	}

	adCache.Set(cacheKey, years, int64(len(years)*10))
	return years, nil
}

// GetAdModels returns models that have existing ads for make/year (for tree view)
func GetAdModels(makeName, year string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:models:%s:%s", makeName, year)

	if cached, found := adCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	models, err := part.GetModelsForAll(makeName, year)
	if err != nil {
		return nil, err
	}

	adCache.Set(cacheKey, models, int64(len(models)*30))
	return models, nil
}

// GetAdEngines returns engines that have existing ads for make/year/model (for tree view)
func GetAdEngines(makeName, year, model string) ([]string, error) {
	cacheKey := fmt.Sprintf("ad:engines:%s:%s:%s", makeName, year, model)

	if cached, found := adCache.Get(cacheKey); found {
		return cached, nil
	}

	// Cache miss - query database and populate cache
	engines, err := part.GetEnginesForAll(makeName, year, model)
	if err != nil {
		return nil, err
	}

	adCache.Set(cacheKey, engines, int64(len(engines)*25))
	return engines, nil
}

func GetAllEngineSizes() []string {
	if allEngineSizesCache != nil {
		return allEngineSizesCache
	}
	query := "SELECT DISTINCT name FROM Engine ORDER BY name"
	var engines []string
	err := db.Select(&engines, query)
	if err != nil {
		return nil
	}
	allEngineSizesCache = engines
	return engines
}

func GetYearRange() []string {
	if yearRangeCache != nil {
		return yearRangeCache
	}
	currentYear := time.Now().Year() + 1
	years := make([]string, 0, currentYear-1900+1)
	for year := 1900; year <= currentYear; year++ {
		years = append(years, strconv.Itoa(year))
	}
	yearRangeCache = years
	return years
}

// AddParentCompany inserts a new parent company
func AddParentCompany(name, country string) (int, error) {
	res, err := db.Exec("INSERT INTO ParentCompany (name, country) VALUES (?, ?)", name, country)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// UpdateParentCompanyCountry updates the country for a parent company
func UpdateParentCompanyCountry(id int, country string) error {
	_, err := db.Exec("UPDATE ParentCompany SET country = ? WHERE id = ?", country, id)
	return err
}

// GetParentCompaniesForMake returns the parent company name for a given make
func GetParentCompaniesForMake(makeName string) ([]string, error) {
	rows, err := db.Query(`
		SELECT pc.name
		FROM ParentCompany pc
		JOIN Make m ON pc.id = m.parent_company_id
		WHERE m.name = ?
	`, makeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parentCompanies []string
	for rows.Next() {
		var pcName string
		if err := rows.Scan(&pcName); err != nil {
			return nil, err
		}
		parentCompanies = append(parentCompanies, pcName)
	}
	return parentCompanies, nil
}

// ParentCompanyInfo represents parent company information with country
type ParentCompanyInfo struct {
	Name    string
	Country string
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
