package part

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/db"
)

type Category struct {
	ID   int
	Name string
}

type SubCategory struct {
	ID         int
	CategoryID int
	Name       string
}

func GetAllCategories() ([]Category, error) {
	rows, err := db.Query("SELECT id, name FROM PartCategory ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func GetSubCategoriesForCategory(categoryName string) ([]SubCategory, error) {
	rows, err := db.Query(`
		SELECT psc.id, psc.category_id, psc.name 
		FROM PartSubCategory psc
		JOIN PartCategory pc ON psc.category_id = pc.id
		WHERE pc.name = ?
		ORDER BY psc.name
	`, categoryName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subCategories []SubCategory
	for rows.Next() {
		var sc SubCategory
		if err := rows.Scan(&sc.ID, &sc.CategoryID, &sc.Name); err != nil {
			return nil, err
		}
		subCategories = append(subCategories, sc)
	}
	return subCategories, nil
}

func GetMakes(query string) ([]string, error) {
	// If there's a search query, filter makes based on matching ads
	if query != "" {
		// Parse the query to get structured search criteria
		// For now, we'll do a simple text search on ad descriptions
		rows, err := db.Query(`
			SELECT DISTINCT m.name
			FROM Make m
			JOIN Car c ON m.id = c.make_id
			JOIN AdCar ac ON c.id = ac.car_id
			JOIN Ad a ON ac.ad_id = a.id
			WHERE a.description LIKE ?
			ORDER BY m.name
		`, "%"+query+"%")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var makes []string
		for rows.Next() {
			var make string
			if err := rows.Scan(&make); err != nil {
				return nil, err
			}
			makes = append(makes, make)
		}
		return makes, nil
	}

	// If no query, return all makes
	rows, err := db.Query(`
		SELECT DISTINCT m.name
		FROM Make m
		JOIN Car c ON m.id = c.make_id
		JOIN AdCar ac ON c.id = ac.car_id
		ORDER BY m.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var makes []string
	for rows.Next() {
		var make string
		if err := rows.Scan(&make); err != nil {
			return nil, err
		}
		makes = append(makes, make)
	}
	return makes, nil
}

func GetYearsForMake(makeName string, query string) ([]string, error) {
	makeName, _ = url.QueryUnescape(makeName)
	// If there's a search query, filter years based on matching ads
	if query != "" {
		rows, err := db.Query(`
			SELECT DISTINCT y.year
			FROM Year y
			JOIN Car c ON y.id = c.year_id
			JOIN Make m ON c.make_id = m.id
			JOIN AdCar ac ON c.id = ac.car_id
			JOIN Ad a ON ac.ad_id = a.id
			WHERE m.name = ? AND a.description LIKE ?
			ORDER BY y.year DESC
		`, makeName, "%"+query+"%")
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var years []string
		for rows.Next() {
			var year string
			if err := rows.Scan(&year); err != nil {
				return nil, err
			}
			years = append(years, year)
		}
		return years, nil
	}

	// If no query, return all years for the make
	rows, err := db.Query(`
		SELECT DISTINCT y.year
		FROM Year y
		JOIN Car c ON y.id = c.year_id
		JOIN Make m ON c.make_id = m.id
		JOIN AdCar ac ON c.id = ac.car_id
		WHERE m.name = ?
		ORDER BY y.year DESC
	`, makeName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []string
	for rows.Next() {
		var year string
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		years = append(years, year)
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

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w; query: %s; args: %v", err, query, args)
	}
	defer rows.Close()

	var ads []ad.Ad
	var adIDs []int
	for rows.Next() {
		var adID int
		var adObj ad.Ad
		var subcategory, category, makeName, modelName, engineName sql.NullString
		var year sql.NullInt64
		if err := rows.Scan(&adID, &adObj.Description, &adObj.Price, &adObj.CreatedAt, &adObj.SubCategoryID, &adObj.UserID, &subcategory, &category, &makeName, &year, &modelName, &engineName); err != nil {
			return nil, err
		}
		adObj.ID = adID
		if category.Valid {
			adObj.Category = category.String
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
		// Populate all years, models, engines for the ad
		fullAd, ok := ad.GetAdWithVehicle(adID, nil)
		if ok {
			adObj.Years = fullAd.Years
			adObj.Models = fullAd.Models
			adObj.Engines = fullAd.Engines
		}
		ads = append(ads, adObj)
		adIDs = append(adIDs, adID)
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
		       a.image_order, a.location_id,
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
		var imageOrder sql.NullString
		var locationID sql.NullInt64
		var city, adminArea, country sql.NullString
		if err := rows.Scan(&adID, &adObj.Title, &adObj.Description, &adObj.Price, &adObj.CreatedAt, &adObj.SubCategoryID, &adObj.UserID, &subcategory, &category, &makeName, &year, &modelName, &engineName, &isBookmarked, &imageOrder, &locationID, &city, &adminArea, &country); err != nil {
			return nil, err
		}
		adObj.ID = adID
		if category.Valid {
			adObj.Category = category.String
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

		// Handle image order
		if imageOrder.Valid {
			_ = json.Unmarshal([]byte(imageOrder.String), &adObj.ImageOrder)
		}

		// Handle location fields
		if locationID.Valid {
			adObj.LocationID = int(locationID.Int64)
		}
		if city.Valid {
			adObj.City = city.String
		}
		if adminArea.Valid {
			adObj.AdminArea = adminArea.String
		}
		if country.Valid {
			adObj.Country = country.String
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
		       a.image_order, a.location_id,
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
		var imageOrder sql.NullString
		var locationID sql.NullInt64
		var city, adminArea, country sql.NullString
		if err := rows.Scan(&adID, &adObj.Title, &adObj.Description, &adObj.Price, &adObj.CreatedAt, &adObj.SubCategoryID, &adObj.UserID, &subcategory, &category, &makeName, &year, &modelName, &engineName, &isBookmarked, &imageOrder, &locationID, &city, &adminArea, &country); err != nil {
			return nil, err
		}
		adObj.ID = adID
		if category.Valid {
			adObj.Category = category.String
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

		// Handle image order
		if imageOrder.Valid {
			_ = json.Unmarshal([]byte(imageOrder.String), &adObj.ImageOrder)
		}

		// Handle location fields
		if locationID.Valid {
			adObj.LocationID = int(locationID.Int64)
		}
		if city.Valid {
			adObj.City = city.String
		}
		if adminArea.Valid {
			adObj.AdminArea = adminArea.String
		}
		if country.Valid {
			adObj.Country = country.String
		}

		// Get vehicle data directly to avoid overwriting bookmark status
		adObj.Make, adObj.Years, adObj.Models, adObj.Engines = ad.GetVehicleData(adID)
		ads = append(ads, adObj)
	}

	// For tree view, always return ads regardless of path length
	// This allows us to extract children from ads at any level
	return ads, nil
}
