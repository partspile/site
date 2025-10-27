package ad

import (
	"log"

	"github.com/parts-pile/site/db"
)

// Ad categories
const (
	AdCategoryCar int = iota + 1
	AdCategoryCarPart
	AdCategoryMotorcycle
	AdCategoryMotorcyclePart
	AdCategoryBicycle
	AdCategoryBicyclePart
	AdCategoryAg
	AdCategoryAgPart
)

// Cached ad category names map
var adCategoryNames map[int]string

// SetAdCategoryNames populates the cached adCategoryNames map from the database
func SetAdCategoryNames() {
	adCategoryNames = make(map[int]string)

	rows, err := db.Query("SELECT id, name FROM AdCategory ORDER BY id")
	if err != nil {
		log.Printf("Failed to query AdCategory: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Printf("Failed to scan AdCategory row: %v", err)
			continue
		}
		adCategoryNames[id] = name
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

// GetCategoryNames returns all category names in order
func GetCategoryNames() []string {
	var names []string
	for _, id := range GetCategoryIDs() {
		names = append(names, GetCategoryName(id))
	}
	return names
}

// GetCategoryName returns the name for a category ID
func GetCategoryName(categoryID int) string {
	if name, exists := adCategoryNames[categoryID]; exists {
		return name
	}
	return "Unknown Category"
}

// IsValidCategory returns true if the category ID is valid
func IsValidCategory(categoryID int) bool {
	_, exists := adCategoryNames[categoryID]
	return exists
}

// HasYears returns true if the vehicle type for this category has years
func HasYears(adCategory int) bool {
	switch adCategory {
	case AdCategoryCar, AdCategoryCarPart,
		AdCategoryMotorcycle, AdCategoryMotorcyclePart,
		AdCategoryAg, AdCategoryAgPart:
		return true
	default:
		return false
	}
}

// HasEngines returns true if the vehicle type for this category has engines
func HasEngines(adCategory int) bool {
	switch adCategory {
	case AdCategoryCar, AdCategoryCarPart,
		AdCategoryMotorcycle, AdCategoryMotorcyclePart:
		return true
	default:
		return false
	}
}
