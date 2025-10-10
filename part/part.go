package part

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/parts-pile/site/db"
)

type Category struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type SubCategory struct {
	ID         int    `db:"id"`
	CategoryID int    `db:"category_id"`
	Name       string `db:"name"`
}

var (
	// Simple maps for static data that never changes
	allCategories    []string
	allSubCategories = make(map[string][]string) // category -> subcategories
)

// Initialize parts static data
func InitPartsData() error {
	// Load all categories
	categories, err := GetAllCategories()
	if err != nil {
		return fmt.Errorf("failed to load categories: %w", err)
	}

	allCategories = make([]string, len(categories))
	for i, cat := range categories {
		allCategories[i] = cat.Name
	}

	// Load subcategories for each category
	for _, categoryName := range allCategories {
		subCategories, err := GetSubCategoriesForCategory(categoryName)
		if err != nil {
			continue
		}

		subCategoryNames := make([]string, len(subCategories))
		for i, subCat := range subCategories {
			subCategoryNames[i] = subCat.Name
		}
		allSubCategories[categoryName] = subCategoryNames
	}

	log.Printf("[parts] Static data loaded - %d categories", len(allCategories))
	return nil
}

// ============================================================================
// STATIC DATA FUNCTIONS (No cache needed - loaded once)
// ============================================================================

// GetCategories returns all categories (static data, no cache needed)
func GetCategories() []string {
	return allCategories
}

// ============================================================================
// AD DATA FUNCTIONS (For tree view)
// ============================================================================

// GetAdCategoriesForAdIDs returns categories that have existing ads for make/year/model/engine, filtered by ad IDs (for tree view)
func GetAdCategoriesForAdIDs(adIDs []int, makeName, year, model, engine string) ([]string, error) {
	return GetCategoriesForAdIDs(adIDs, makeName, year, model, engine)
}

// GetAdSubCategoriesForAdIDs returns subcategories that have existing ads for make/year/model/engine/category, filtered by ad IDs (for tree view)
func GetAdSubCategoriesForAdIDs(adIDs []int, makeName, year, model, engine, category string) ([]string, error) {
	return GetSubCategoriesForAdIDs(adIDs, makeName, year, model, engine, category)
}

// GetAdCategories returns categories that have existing ads for make/year/model/engine (for tree view)
func GetAdCategories(makeName, year, model, engine string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)

	query := `
		SELECT DISTINCT pc.name
		FROM PartCategory pc
		JOIN PartSubCategory psc ON pc.id = psc.category_id
		JOIN Ad a ON psc.id = a.subcategory_id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ?
		ORDER BY pc.name
	`
	var categories []string
	err := db.Select(&categories, query, makeName, year, model, engine)
	return categories, err
}

// GetAdSubCategories returns subcategories that have existing ads for make/year/model/engine/category (for tree view)
func GetAdSubCategories(makeName, year, model, engine, category string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)
	category, _ = url.QueryUnescape(category)

	query := `
		SELECT DISTINCT psc.name
		FROM PartSubCategory psc
		JOIN PartCategory pc ON psc.category_id = pc.id
		JOIN Ad a ON psc.id = a.subcategory_id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ? AND pc.name = ?
		ORDER BY psc.name
	`
	var subCategories []string
	err := db.Select(&subCategories, query, makeName, year, model, engine, category)
	return subCategories, err
}

func GetAllCategories() ([]Category, error) {
	query := "SELECT id, name FROM PartCategory ORDER BY name"
	var categories []Category
	err := db.Select(&categories, query)
	return categories, err
}

func GetSubCategoriesForCategory(categoryName string) ([]SubCategory, error) {
	query := `
		SELECT psc.id, psc.category_id, psc.name 
		FROM PartSubCategory psc
		JOIN PartCategory pc ON psc.category_id = pc.id
		WHERE pc.name = ?
		ORDER BY psc.name
	`
	var subCategories []SubCategory
	err := db.Select(&subCategories, query, categoryName)
	return subCategories, err
}

func GetSubCategoryIDByName(subcategoryName string) (int, error) {
	query := `SELECT id FROM PartSubCategory WHERE name = ?`
	var id int
	err := db.QueryRow(query, subcategoryName).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func GetSubCategoryNameByID(subcategoryID int) (string, error) {
	query := `SELECT name FROM PartSubCategory WHERE id = ?`
	var name string
	err := db.QueryRow(query, subcategoryID).Scan(&name)
	if err != nil {
		return "", err
	}
	return name, nil
}

func getMakes(query string) ([]string, error) {
	// If there's a search query, filter makes based on matching ads
	if query != "" {
		querySQL := `
			SELECT DISTINCT m.name
			FROM Make m
			JOIN Car c ON m.id = c.make_id
			JOIN AdCar ac ON c.id = ac.car_id
			JOIN Ad a ON ac.ad_id = a.id
			WHERE a.description LIKE ?
			ORDER BY m.name
		`
		var makes []string
		err := db.Select(&makes, querySQL, "%"+query+"%")
		return makes, err
	}

	// If no query, return all makes
	querySQL := `
		SELECT DISTINCT m.name
		FROM Make m
		JOIN Car c ON m.id = c.make_id
		JOIN AdCar ac ON c.id = ac.car_id
		ORDER BY m.name
	`
	var makes []string
	err := db.Select(&makes, querySQL)
	return makes, err
}

// ============================================================================
// NEW TREE VIEW FUNCTIONS - Filtered by ad IDs (for search mode)
// ============================================================================

// GetCategoriesForAdIDs returns categories for a specific make/year/model/engine, filtered by ad IDs
func GetCategoriesForAdIDs(adIDs []int, makeName, year, model, engine string) ([]string, error) {
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
		JOIN PartSubCategory psc ON pc.id = psc.category_id
		JOIN Ad a ON psc.id = a.subcategory_id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
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

// GetSubCategoriesForAdIDs returns subcategories for a specific make/year/model/engine/category, filtered by ad IDs
func GetSubCategoriesForAdIDs(adIDs []int, makeName, year, model, engine, category string) ([]string, error) {
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
		FROM PartSubCategory psc
		JOIN PartCategory pc ON psc.category_id = pc.id
		JOIN Ad a ON psc.id = a.subcategory_id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
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

// ============================================================================
// NEW TREE VIEW FUNCTIONS - Browse mode (when adIDs is nil/empty)
// ============================================================================

// GetMakesForAll returns all makes that have ads
func GetMakesForAll() ([]string, error) {
	query := `
		SELECT DISTINCT m.name
		FROM Make m
		JOIN Car c ON m.id = c.make_id
		JOIN AdCar ac ON c.id = ac.car_id
		ORDER BY m.name
	`
	var makes []string
	err := db.Select(&makes, query)
	return makes, err
}

// GetYearsForAll returns all years for a specific make
func GetYearsForAll(makeName string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	query := `
		SELECT DISTINCT y.year
		FROM Year y
		JOIN Car c ON y.id = c.year_id
		JOIN Make m ON c.make_id = m.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ?
		ORDER BY y.year DESC
	`
	var yearInts []int
	err := db.Select(&yearInts, query, makeName)
	if err != nil {
		return nil, err
	}

	var years []string
	for _, year := range yearInts {
		years = append(years, fmt.Sprintf("%d", year))
	}
	return years, nil
}

// GetModelsForAll returns all models for a specific make/year
func GetModelsForAll(makeName, year string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	query := `
		SELECT DISTINCT mo.name
		FROM Model mo
		JOIN Car c ON mo.id = c.model_id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ? AND y.year = ?
		ORDER BY mo.name
	`
	var models []string
	err := db.Select(&models, query, makeName, year)
	return models, err
}

// GetEnginesForAll returns all engines for a specific make/year/model
func GetEnginesForAll(makeName, year, model string) ([]string, error) {
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
		WHERE m.name = ? AND y.year = ? AND mo.name = ?
		ORDER BY e.name
	`
	var engines []string
	err := db.Select(&engines, query, makeName, year, model)
	return engines, err
}
