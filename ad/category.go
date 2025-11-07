package ad

import (
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/spec"
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

// GetCategoryFields returns the ordered list of fields for a category
func GetCategoryFields(adCategoryID int) ([]spec.FieldConfig, error) {
	return spec.GetCategoryFields(adCategoryID)
}

// HasField returns true if the category has the specified field
func HasField(adCategoryID int, fieldName string) bool {
	return spec.HasField(adCategoryID, fieldName)
}

// HasYears returns true if the category has a year field
func HasYears(adCategoryID int) bool {
	return HasField(adCategoryID, "year")
}

// HasEngines returns true if the category has an engine field
func HasEngines(adCategoryID int) bool {
	return HasField(adCategoryID, "engine")
}

// IsPartCategory returns true if the category is for parts (not whole vehicles)
func IsPartCategory(adCategory int) bool {
	switch adCategory {
	case AdCategoryCarPart, AdCategoryMotorcyclePart,
		AdCategoryBicyclePart, AdCategoryAgPart:
		return true
	default:
		return false
	}
}
