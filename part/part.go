package part

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/parts-pile/site/ad"
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

// GetSubCategories returns all subcategories for a category (static data, no cache needed)
func GetSubCategories(categoryName string) []string {
	return allSubCategories[categoryName]
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
	return GetCategoriesAll(makeName, year, model, engine)
}

// GetAdSubCategories returns subcategories that have existing ads for make/year/model/engine/category (for tree view)
func GetAdSubCategories(makeName, year, model, engine, category string) ([]string, error) {
	return GetSubCategoriesForAll(makeName, year, model, engine, category)
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

func GetMakes(query string) ([]string, error) {
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

func GetYearsForMake(makeName string, query string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	// If there's a search query, filter years based on matching ads
	if query != "" {
		querySQL := `
			SELECT DISTINCT y.year
			FROM Year y
			JOIN Car c ON y.id = c.year_id
			JOIN Make m ON c.make_id = m.id
			JOIN AdCar ac ON c.id = ac.car_id
			JOIN Ad a ON ac.ad_id = a.id
			WHERE m.name = ? AND a.description LIKE ?
			ORDER BY y.year DESC
		`
		var yearInts []int
		err := db.Select(&yearInts, querySQL, makeName, "%"+query+"%")
		if err != nil {
			return nil, err
		}
		var years []string
		for _, year := range yearInts {
			years = append(years, fmt.Sprintf("%d", year))
		}
		return years, nil
	}

	// If no query, return all years for the make
	querySQL := `
		SELECT DISTINCT y.year
		FROM Year y
		JOIN Car c ON y.id = c.year_id
		JOIN Make m ON c.make_id = m.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ?
		ORDER BY y.year DESC
	`
	var yearInts []int
	err := db.Select(&yearInts, querySQL, makeName)
	if err != nil {
		return nil, err
	}
	var years []string
	for _, year := range yearInts {
		years = append(years, fmt.Sprintf("%d", year))
	}
	return years, nil
}

func GetModelsForMakeYear(makeName, year, query string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	// If there's a search query, filter models based on matching ads
	if query != "" {
		rows, err := db.Query(`
			SELECT DISTINCT mo.name
			FROM Model mo
			JOIN Car c ON mo.id = c.model_id
			JOIN Make m ON c.make_id = m.id
			JOIN Year y ON c.year_id = y.id
			JOIN AdCar ac ON c.id = ac.car_id
			JOIN Ad a ON ac.ad_id = a.id
			WHERE m.name = ? AND y.year = ? AND a.description LIKE ?
			ORDER BY mo.name
		`, makeName, year, "%"+query+"%")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var models []string
		for rows.Next() {
			var model string
			if err := rows.Scan(&model); err != nil {
				return nil, err
			}
			models = append(models, model)
		}
		return models, nil
	}

	// If no query, return all models for the make/year
	rows, err := db.Query(`
		SELECT DISTINCT mo.name
		FROM Model mo
		JOIN Car c ON mo.id = c.model_id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ? AND y.year = ?
		ORDER BY mo.name
	`, makeName, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []string
	for rows.Next() {
		var model string
		if err := rows.Scan(&model); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

func GetEnginesForMakeYearModel(makeName, year, model, query string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	// If there's a search query, filter engines based on matching ads
	if query != "" {
		rows, err := db.Query(`
			SELECT DISTINCT e.name
			FROM Engine e
			JOIN Car c ON e.id = c.engine_id
			JOIN Make m ON c.make_id = m.id
			JOIN Year y ON c.year_id = y.id
			JOIN Model mo ON c.model_id = mo.id
			JOIN AdCar ac ON c.id = ac.car_id
			JOIN Ad a ON ac.ad_id = a.id
			WHERE m.name = ? AND y.year = ? AND mo.name = ? AND a.description LIKE ?
			ORDER BY e.name
		`, makeName, year, model, "%"+query+"%")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var engines []string
		for rows.Next() {
			var engine string
			if err := rows.Scan(&engine); err != nil {
				return nil, err
			}
			engines = append(engines, engine)
		}
		return engines, nil
	}

	// If no query, return all engines for the make/year/model
	rows, err := db.Query(`
		SELECT DISTINCT e.name
		FROM Engine e
		JOIN Car c ON e.id = c.engine_id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ? AND y.year = ? AND mo.name = ?
		ORDER BY e.name
	`, makeName, year, model)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var engines []string
	for rows.Next() {
		var engine string
		if err := rows.Scan(&engine); err != nil {
			return nil, err
		}
		engines = append(engines, engine)
	}
	return engines, nil
}

func GetCategoriesForMakeYearModelEngine(makeName, year, model, engine, query string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)
	// If there's a search query, filter categories based on matching ads
	if query != "" {
		rows, err := db.Query(`
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
			WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ? AND a.description LIKE ?
			ORDER BY pc.name
		`, makeName, year, model, engine, "%"+query+"%")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var categories []string
		for rows.Next() {
			var category string
			if err := rows.Scan(&category); err != nil {
				return nil, err
			}
			categories = append(categories, category)
		}
		return categories, nil
	}

	// If no query, return all categories for the make/year/model/engine
	rows, err := db.Query(`
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
	`, makeName, year, model, engine)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			return nil, err
		}
		categories = append(categories, category)
	}
	return categories, nil
}

func GetSubCategoriesForMakeYearModelEngineCategory(makeName, year, model, engine, category, query string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)
	category, _ = url.QueryUnescape(category)
	// If there's a search query, filter subcategories based on matching ads
	if query != "" {
		rows, err := db.Query(`
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
			WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ? AND pc.name = ? AND a.description LIKE ?
			ORDER BY psc.name
		`, makeName, year, model, engine, category, "%"+query+"%")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var subCategories []string
		for rows.Next() {
			var subCategory string
			if err := rows.Scan(&subCategory); err != nil {
				return nil, err
			}
			subCategories = append(subCategories, subCategory)
		}
		return subCategories, nil
	}

	// If no query, return all subcategories for the make/year/model/engine/category
	rows, err := db.Query(`
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
	`, makeName, year, model, engine, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subCategories []string
	for rows.Next() {
		var subCategory string
		if err := rows.Scan(&subCategory); err != nil {
			return nil, err
		}
		subCategories = append(subCategories, subCategory)
	}
	return subCategories, nil
}

func GetAdsForNode(parts []string, q string) ([]ad.Ad, error) {
	// Decode all path segments
	decodedParts := make([]string, len(parts))
	for i, p := range parts {
		d, err := url.QueryUnescape(p)
		if err != nil {
			decodedParts[i] = p
		} else {
			decodedParts[i] = d
		}
	}
	query := `
		SELECT a.id, a.description, a.price, a.created_at, a.subcategory_id,
		       a.user_id, psc.name as subcategory, pc.name as category,
		       m.name, y.year, mo.name, e.name
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
	`
	var args []interface{}
	var conditions []string

	if len(decodedParts) > 0 && decodedParts[0] != "" {
		conditions = append(conditions, "m.name = ?")
		args = append(args, decodedParts[0])
	}
	if len(decodedParts) > 1 {
		conditions = append(conditions, "y.year = ?")
		args = append(args, decodedParts[1])
	}
	if len(decodedParts) > 2 {
		conditions = append(conditions, "mo.name = ?")
		args = append(args, decodedParts[2])
	}
	if len(decodedParts) > 3 {
		conditions = append(conditions, "e.name = ?")
		args = append(args, decodedParts[3])
	}
	if len(decodedParts) > 4 {
		conditions = append(conditions, "pc.name = ?")
		args = append(args, decodedParts[4])
	}
	if len(decodedParts) > 5 {
		conditions = append(conditions, "psc.name = ?")
		args = append(args, decodedParts[5])
	}

	// Add search query filter if provided
	if q != "" {
		conditions = append(conditions, "a.description LIKE ?")
		args = append(args, "%"+q+"%")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var ads []ad.Ad
	err := db.Select(&ads, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w; query: %s; args: %v", err, query, args)
	}

	// Populate vehicle data for each ad
	for i := range ads {
		fullAd, ok := ad.GetAdWithVehicle(ads[i].ID, nil)
		if ok {
			ads[i].Years = fullAd.Years
			ads[i].Models = fullAd.Models
			ads[i].Engines = fullAd.Engines
		}
	}

	// Only show ads at leaf nodes (make/year/model/engine)
	if len(decodedParts) < 4 {
		return nil, nil
	}

	return ads, nil
}

func GetAdsForNodeStructured(parts []string, sq ad.SearchQuery, userID int) ([]ad.Ad, error) {
	// Decode all path segments
	decodedParts := make([]string, len(parts))
	for i, p := range parts {
		d, err := url.QueryUnescape(p)
		if err != nil {
			decodedParts[i] = p
		} else {
			decodedParts[i] = d
		}
	}
	query := `
		SELECT a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
		       a.user_id, psc.name as subcategory, pc.name as category,
		       m.name, y.year, mo.name, e.name,
		       CASE WHEN fa.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked,
		       a.image_count, a.location_id,
		       l.city, l.admin_area, l.country
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		LEFT JOIN BookmarkedAd fa ON a.id = fa.ad_id AND fa.user_id = ?
		LEFT JOIN Location l ON a.location_id = l.id
	`
	var args []interface{}
	args = append(args, userID)
	var conditions []string

	if len(decodedParts) > 0 && decodedParts[0] != "" {
		conditions = append(conditions, "m.name = ?")
		args = append(args, decodedParts[0])
	} else if sq.Make != "" {
		conditions = append(conditions, "m.name = ?")
		args = append(args, sq.Make)
	}
	if len(decodedParts) > 1 {
		conditions = append(conditions, "y.year = ?")
		args = append(args, decodedParts[1])
	} else if len(sq.Years) > 0 {
		placeholders := make([]string, len(sq.Years))
		for i, y := range sq.Years {
			placeholders[i] = "?"
			args = append(args, y)
		}
		conditions = append(conditions, "y.year IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(decodedParts) > 2 {
		conditions = append(conditions, "mo.name = ?")
		args = append(args, decodedParts[2])
	} else if len(sq.Models) > 0 {
		placeholders := make([]string, len(sq.Models))
		for i, m := range sq.Models {
			placeholders[i] = "?"
			args = append(args, m)
		}
		conditions = append(conditions, "mo.name IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(decodedParts) > 3 {
		conditions = append(conditions, "e.name = ?")
		args = append(args, decodedParts[3])
	} else if len(sq.EngineSizes) > 0 {
		placeholders := make([]string, len(sq.EngineSizes))
		for i, e := range sq.EngineSizes {
			placeholders[i] = "?"
			args = append(args, e)
		}
		conditions = append(conditions, "e.name IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(decodedParts) > 4 {
		conditions = append(conditions, "pc.name = ?")
		args = append(args, decodedParts[4])
	} else if sq.Category != "" {
		conditions = append(conditions, "pc.name = ?")
		args = append(args, sq.Category)
	}
	if len(decodedParts) > 5 {
		conditions = append(conditions, "psc.name = ?")
		args = append(args, decodedParts[5])
	} else if sq.SubCategory != "" {
		conditions = append(conditions, "psc.name = ?")
		args = append(args, sq.SubCategory)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w; query: %s; args: %v", err, query, args)
	}
	defer rows.Close()

	var ads []ad.Ad
	for rows.Next() {
		var adID int
		var adObj ad.Ad
		var subcategory, category, makeName, modelName, engineName sql.NullString
		var year sql.NullInt64
		var isBookmarked int
		var locationID sql.NullInt64
		var city, adminArea, country sql.NullString
		if err := rows.Scan(&adID, &adObj.Title, &adObj.Description, &adObj.Price, &adObj.CreatedAt, &adObj.SubCategoryID, &adObj.UserID, &subcategory, &category, &makeName, &year, &modelName, &engineName, &isBookmarked, &adObj.ImageCount, &locationID, &city, &adminArea, &country); err != nil {
			return nil, err
		}
		adObj.ID = adID
		if category.Valid {
			adObj.Category = category
		}
		if makeName.Valid {
			adObj.Make = makeName.String
		}
		if year.Valid {
			adObj.Years = []string{fmt.Sprintf("%d", year.Int64)}
		}
		if modelName.Valid {
			adObj.Models = []string{modelName.String}
		}
		if engineName.Valid {
			adObj.Engines = []string{engineName.String}
		}
		adObj.Bookmarked = isBookmarked == 1

		// Handle location fields
		if locationID.Valid {
			adObj.LocationID = int(locationID.Int64)
		}
		if city.Valid {
			adObj.City = city
		}
		if adminArea.Valid {
			adObj.AdminArea = adminArea
		}
		if country.Valid {
			adObj.Country = country
		}

		// Populate all years, models, engines for the ad
		fullAd, ok := ad.GetAdWithVehicle(adID, nil)
		if ok {
			adObj.Years = fullAd.Years
			adObj.Models = fullAd.Models
			adObj.Engines = fullAd.Engines
		}
		ads = append(ads, adObj)
	}

	// Only show ads at leaf nodes (make/year/model/engine)
	if len(decodedParts) < 4 {
		return nil, nil
	}

	return ads, nil
}

// GetAdsForTreeView gets ads for tree view - always returns ads regardless of path length
// This is used to extract children from ads at any level
func GetAdsForTreeView(parts []string, sq ad.SearchQuery, userID int) ([]ad.Ad, error) {
	// Decode all path segments
	decodedParts := make([]string, len(parts))
	for i, p := range parts {
		d, err := url.QueryUnescape(p)
		if err != nil {
			decodedParts[i] = p
		} else {
			decodedParts[i] = d
		}
	}
	query := `
		SELECT a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
		       a.user_id, psc.name as subcategory, pc.name as category,
		       m.name, y.year, mo.name, e.name,
		       CASE WHEN fa.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked,
		       a.image_count, a.location_id,
		       l.city, l.admin_area, l.country
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		LEFT JOIN BookmarkedAd fa ON a.id = fa.ad_id AND fa.user_id = ?
		LEFT JOIN Location l ON a.location_id = l.id
	`
	var args []interface{}
	args = append(args, userID)
	var conditions []string

	if len(decodedParts) > 0 && decodedParts[0] != "" {
		conditions = append(conditions, "m.name = ?")
		args = append(args, decodedParts[0])
	} else if sq.Make != "" {
		conditions = append(conditions, "m.name = ?")
		args = append(args, sq.Make)
	}
	if len(decodedParts) > 1 {
		conditions = append(conditions, "y.year = ?")
		args = append(args, decodedParts[1])
	} else if len(sq.Years) > 0 {
		placeholders := make([]string, len(sq.Years))
		for i, y := range sq.Years {
			placeholders[i] = "?"
			args = append(args, y)
		}
		conditions = append(conditions, "y.year IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(decodedParts) > 2 {
		conditions = append(conditions, "mo.name = ?")
		args = append(args, decodedParts[2])
	} else if len(sq.Models) > 0 {
		placeholders := make([]string, len(sq.Models))
		for i, m := range sq.Models {
			placeholders[i] = "?"
			args = append(args, m)
		}
		conditions = append(conditions, "mo.name IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(decodedParts) > 3 {
		conditions = append(conditions, "e.name = ?")
		args = append(args, decodedParts[3])
	} else if len(sq.EngineSizes) > 0 {
		placeholders := make([]string, len(sq.EngineSizes))
		for i, e := range sq.EngineSizes {
			placeholders[i] = "?"
			args = append(args, e)
		}
		conditions = append(conditions, "e.name IN ("+strings.Join(placeholders, ",")+")")
	}
	if len(decodedParts) > 4 {
		conditions = append(conditions, "pc.name = ?")
		args = append(args, decodedParts[4])
	} else if sq.Category != "" {
		conditions = append(conditions, "pc.name = ?")
		args = append(args, sq.Category)
	}
	if len(decodedParts) > 5 {
		conditions = append(conditions, "psc.name = ?")
		args = append(args, decodedParts[5])
	} else if sq.SubCategory != "" {
		conditions = append(conditions, "psc.name = ?")
		args = append(args, sq.SubCategory)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w; query: %s; args: %v", err, query, args)
	}
	defer rows.Close()

	var ads []ad.Ad
	for rows.Next() {
		var adID int
		var adObj ad.Ad
		var subcategory, category, makeName, modelName, engineName sql.NullString
		var year sql.NullInt64
		var isBookmarked int
		var locationID sql.NullInt64
		var city, adminArea, country sql.NullString
		if err := rows.Scan(&adID, &adObj.Title, &adObj.Description, &adObj.Price, &adObj.CreatedAt, &adObj.SubCategoryID, &adObj.UserID, &subcategory, &category, &makeName, &year, &modelName, &engineName, &isBookmarked, &adObj.ImageCount, &locationID, &city, &adminArea, &country); err != nil {
			return nil, err
		}
		adObj.ID = adID
		if category.Valid {
			adObj.Category = category
		}
		if makeName.Valid {
			adObj.Make = makeName.String
		}
		if year.Valid {
			adObj.Years = []string{fmt.Sprintf("%d", year.Int64)}
		}
		if modelName.Valid {
			adObj.Models = []string{modelName.String}
		}
		if engineName.Valid {
			adObj.Engines = []string{engineName.String}
		}
		adObj.Bookmarked = isBookmarked == 1

		// Handle location fields
		if locationID.Valid {
			adObj.LocationID = int(locationID.Int64)
		}
		if city.Valid {
			adObj.City = city
		}
		if adminArea.Valid {
			adObj.AdminArea = adminArea
		}
		if country.Valid {
			adObj.Country = country
		}

		// Get vehicle data directly to avoid overwriting bookmark status
		adObj.Make, adObj.Years, adObj.Models, adObj.Engines = ad.GetVehicleData(adID)
		ads = append(ads, adObj)
	}

	// For tree view, always return ads regardless of path length
	// This allows us to extract children from ads at any level
	return ads, nil
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

// GetCategoriesAll returns all categories for a specific make/year/model/engine
func GetCategoriesAll(makeName, year, model, engine string) ([]string, error) {
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

// GetSubCategoriesForAll returns all subcategories for a specific make/year/model/engine/category
func GetSubCategoriesForAll(makeName, year, model, engine, category string) ([]string, error) {
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
