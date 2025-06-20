package part

import (
	"database/sql"
)

var db *sql.DB

// InitDB sets the database connection for the part package
func InitDB(database *sql.DB) {
	db = database
}

func GetAllCategories() []string {
	rows, err := db.Query("SELECT name FROM PartCategory ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			continue
		}
		categories = append(categories, category)
	}
	return categories
}

func GetAllSubCategories() []string {
	rows, err := db.Query("SELECT DISTINCT name FROM PartSubCategory ORDER BY name")
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
