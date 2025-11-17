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
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT m.value
		FROM Make m
		JOIN Specification spec ON m.id = spec.make_id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		WHERE ad_spec.ad_id IN (%s)
		ORDER BY m.value
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

// ============================================================================
// DATABASE QUERY FUNCTIONS (Internal - used by cached functions above)
// ============================================================================

// getMakesForAll returns all makes for a specific category
func getMakesForAll(adCat int) ([]string, error) {
	query := `
		SELECT DISTINCT m.value
		FROM Make m
		JOIN Specification spec ON m.id = spec.make_id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE a.ad_category_id = $1
		ORDER BY m.value
	`
	var makes []string
	err := db.Select(&makes, query, adCat)
	return makes, nil
}

// getYearsForAll returns all years for a specific make and category
func getYearsForAll(adCat int, makeName string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	query := `
		SELECT DISTINCT y.value
		FROM Year y
		JOIN Specification spec ON y.id = spec.year_id
		JOIN Make m ON spec.make_id = m.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE m.value = $1 AND a.ad_category_id = $2
		ORDER BY y.value DESC
	`
	var years []string
	err := db.Select(&years, query, makeName, adCat)
	if err != nil {
		return nil, err
	}
	return years, nil
}

// getModelsForAll returns all models for a specific make/year and category
func getModelsForAll(adCat int, makeName, year string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	query := `
		SELECT DISTINCT mo.value
		FROM Model mo
		JOIN Specification spec ON mo.id = spec.model_id
		JOIN Make m ON spec.make_id = m.id
		JOIN Year y ON spec.year_id = y.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE m.value = $1 AND y.value = $2 AND a.ad_category_id = $3
		ORDER BY mo.value
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
		SELECT DISTINCT e.value
		FROM Engine e
		JOIN Specification spec ON e.id = spec.engine_id
		JOIN Make m ON spec.make_id = m.id
		JOIN Year y ON spec.year_id = y.id
		JOIN Model mo ON spec.model_id = mo.id
		JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id
		JOIN Ad a ON ad_spec.ad_id = a.id
		WHERE m.value = $1 AND y.value = $2 AND mo.value = $3 AND a.ad_category_id = $4
		ORDER BY e.value
	`
	var engines []string
	err := db.Select(&engines, query, makeName, year, model, adCat)
	return engines, err
}
