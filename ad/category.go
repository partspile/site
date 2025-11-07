package ad

import (
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// Cached ad category names map (id -> name)
var adCategoryNames map[int]string

// Cached ad category IDs map (name -> id)
var adCategoryIDs map[string]int

// INitCategoryNames populates the cached adCategoryNames and adCategoryIDs maps from the database
func InitCategoryNames() {
	adCategoryNames = make(map[int]string)
	adCategoryIDs = make(map[string]int)

	rows, err := db.Query("SELECT id, name FROM AdCategory ORDER BY id")
	if err != nil {
		logger.Error("Failed to query AdCategory",
			"error", err, "component", "ad-category")
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			logger.Error("Failed to scan AdCategory row",
				"error", err, "component", "ad-category")
			continue
		}
		adCategoryNames[id] = name
		adCategoryIDs[name] = id
	}
}

// GetCategoryIDs returns all category IDs in order
func GetCategoryIDs() []int {
	var ids []int
	for i := 1; ; i++ {
		if _, exists := adCategoryNames[i]; !exists {
			break
		}
		ids = append(ids, i)
	}
	return ids
}

// GetCategoryName returns the name for a category ID
func GetCategoryName(adCat int) string {
	if name, exists := adCategoryNames[adCat]; exists {
		return name
	}
	return "Unknown Category"
}

// IsValidCategory returns true if the category ID is valid
func IsValidCategory(adCat int) bool {
	_, exists := adCategoryNames[adCat]
	return exists
}

// GetCategoryIDByName returns the category ID for a given category name, or 0 if not found
func GetCategoryIDByName(name string) int {
	if id, exists := adCategoryIDs[name]; exists {
		return id
	}
	return 0
}
