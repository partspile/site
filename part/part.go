package part

import (
	"database/sql"
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
