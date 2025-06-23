package part

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/parts-pile/site/ad"
)

var db *sql.DB

// InitDB sets the database connection for the part package
func InitDB(database *sql.DB) {
	db = database
}

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

func GetAllSubCategories() ([]SubCategory, error) {
	rows, err := db.Query("SELECT id, category_id, name FROM PartSubCategory ORDER BY name")
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

func GetSubCategoriesForCategory(categoryName string) []string {
	rows, err := db.Query(`
		SELECT psc.name 
		FROM PartSubCategory psc
		JOIN PartCategory pc ON psc.category_id = pc.id
		WHERE pc.name = ?
		ORDER BY psc.name
	`, categoryName)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var subCategories []string
	for rows.Next() {
		var subCategory string
		if err := rows.Scan(&subCategory); err != nil {
			continue
		}
		subCategories = append(subCategories, subCategory)
	}
	return subCategories
}

func GetMakes(query string) ([]string, error) {
	// The query parameter is not used yet, but will be used to filter makes
	// based on the search query.
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
	// query is not used yet
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
	// query is not used yet
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
	// query is not used yet
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
	// query is not used yet
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
	// query is not used yet
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
	query := `
		SELECT DISTINCT a.id, a.description, a.price, a.created_at, a.subcategory_id,
		       a.user_id, psc.name as subcategory, pc.name as category
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

	if len(parts) > 0 && parts[0] != "" {
		conditions = append(conditions, "m.name = ?")
		args = append(args, parts[0])
	}
	if len(parts) > 1 {
		conditions = append(conditions, "y.year = ?")
		args = append(args, parts[1])
	}
	if len(parts) > 2 {
		conditions = append(conditions, "mo.name = ?")
		args = append(args, parts[2])
	}
	if len(parts) > 3 {
		conditions = append(conditions, "e.name = ?")
		args = append(args, parts[3])
	}
	if len(parts) > 4 {
		conditions = append(conditions, "pc.name = ?")
		args = append(args, parts[4])
	}
	if len(parts) > 5 {
		conditions = append(conditions, "psc.name = ?")
		args = append(args, parts[5])
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
		var ad ad.Ad
		var subcategory, category sql.NullString
		if err := rows.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt, &ad.SubCategoryID, &ad.UserID, &subcategory, &category); err != nil {
			return nil, err
		}
		if subcategory.Valid {
			ad.SubCategory = subcategory.String
		}
		if category.Valid {
			ad.Category = category.String
		}
		ads = append(ads, ad)
	}

	// This is N+1, but OK for now. The tree view will limit queries.
	for i := range ads {
		// This is a placeholder for getting full vehicle data for an ad.
		// A proper implementation would do a more efficient query.
		ads[i].Years = []string{"2024"}
		ads[i].Models = []string{"Some Model"}
		ads[i].Engines = []string{"V8"}
		if len(parts) > 0 {
			ads[i].Make = parts[0]
		}
	}

	return ads, nil
}
