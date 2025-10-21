package ad

import (
	"strings"
)

// Ad category constants as strings
const (
	Car            = "Car"
	CarPart        = "CarPart"
	Motorcycle     = "Motorcycle"
	MotorcyclePart = "MotorcyclePart"
	Bicycle        = "Bicycle"
	BicyclePart    = "BicyclePart"
	Ag             = "Ag"
	AgPart         = "AgPart"
)

// GetAdCategoryID returns the database ID for a category string
func GetAdCategoryID(category string) int {
	switch category {
	case Car:
		return 1
	case CarPart:
		return 2
	case Motorcycle:
		return 3
	case MotorcyclePart:
		return 4
	case Bicycle:
		return 5
	case BicyclePart:
		return 6
	case Ag:
		return 7
	case AgPart:
		return 8
	default:
		return 2 // Default to CarPart
	}
}

// GetAdCategoryFromID returns the category string for a database ID
func GetAdCategoryFromID(id int) string {
	switch id {
	case 1:
		return Car
	case 2:
		return CarPart
	case 3:
		return Motorcycle
	case 4:
		return MotorcyclePart
	case 5:
		return Bicycle
	case 6:
		return BicyclePart
	case 7:
		return Ag
	case 8:
		return AgPart
	default:
		return CarPart // Default fallback
	}
}

// GetDisplayName returns the human-readable display name for the category
func GetDisplayName(category string) string {
	switch category {
	case Car:
		return "Cars"
	case CarPart:
		return "Car Parts"
	case Motorcycle:
		return "Motorcycles"
	case MotorcyclePart:
		return "Motorcycle Parts"
	case Bicycle:
		return "Bicycle"
	case BicyclePart:
		return "Bicycle Parts"
	case Ag:
		return "Ag Equipment"
	case AgPart:
		return "Ag Equipment Parts"
	default:
		return "Unknown"
	}
}

// GetTableInfo returns the vehicle table name, association table name, and vehicle ID column name
func GetTableInfo(category string) (vehicleTable, associationTable, vehicleIDColumn string) {
	if strings.HasSuffix(category, "Part") {
		vehicleTable = strings.TrimSuffix(category, "Part")
	} else {
		vehicleTable = category
	}
	associationTable = "Ad" + category
	vehicleIDColumn = strings.ToLower(vehicleTable)
	return
}

// ParseCategoryFromQuery parses a category string from query parameters, with fallback to CarParts
func ParseCategoryFromQuery(category string) string {
	if category == "" {
		return CarPart // Default fallback
	}

	// Validate the category string
	switch category {
	case Car, CarPart, Motorcycle, MotorcyclePart, Bicycle, BicyclePart, Ag, AgPart:
		return category
	default:
		return CarPart // Default fallback on error
	}
}

// GetAllCategories returns all valid category strings
func GetAllCategories() []string {
	return []string{
		Car, CarPart, Motorcycle, MotorcyclePart,
		Bicycle, BicyclePart, Ag, AgPart,
	}
}
