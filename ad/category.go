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
var AdCategoryNames map[int]string

// SetAdCategoryNames populates the cached AdCategoryNames map from the database
func SetAdCategoryNames() {
	AdCategoryNames = make(map[int]string)

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
		AdCategoryNames[id] = name
	}
}

// CategoryIcon returns the appropriate icon for the category
func CategoryIcon(adCategory int) string {
	switch adCategory {
	case AdCategoryCar, AdCategoryCarPart:
		return "/images/car.svg"
	case AdCategoryMotorcycle, AdCategoryMotorcyclePart:
		return "/images/motorcycle.svg"
	case AdCategoryBicycle, AdCategoryBicyclePart:
		return "/images/bicycle.svg"
	case AdCategoryAg, AdCategoryAgPart:
		return "/images/ag.svg"
	default:
		return "/images/car.svg" // Default fallback
	}
}

// CategoryDisplayName returns the display name for a category ID
func CategoryDisplayName(adCategory int) string {
	if name, exists := AdCategoryNames[adCategory]; exists {
		return name
	}
	return "Unknown Category"
}

// GetTableInfo returns the association table, vehicle table, and vehicle ID column for a given ad category
func GetTableInfo(adCategory int) (associationTable, vehicleTable, vehicleIDColumn string) {
	switch adCategory {
	case AdCategoryCar, AdCategoryCarPart:
		return "AdCar", "Car", "car"
	case AdCategoryMotorcycle, AdCategoryMotorcyclePart:
		return "AdMotorcycle", "Motorcycle", "motorcycle"
	case AdCategoryBicycle, AdCategoryBicyclePart:
		return "AdBicycle", "Bicycle", "bicycle"
	case AdCategoryAg, AdCategoryAgPart:
		return "AdAg", "Ag", "ag"
	default:
		// Default fallback to car
		return "AdCar", "Car", "car"
	}
}
