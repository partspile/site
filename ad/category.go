package ad

import (
	"fmt"
)

// AdCategory represents the different types of ads in the system
type AdCategory int

const (
	Cars AdCategory = iota + 1
	CarParts
	Motorcycles
	MotorcycleParts
	Bicycle
	BicycleParts
	AgEquipment
	AgEquipmentParts
)

// String returns the string representation of the category
func (c AdCategory) String() string {
	switch c {
	case Cars:
		return "Cars"
	case CarParts:
		return "CarParts"
	case Motorcycles:
		return "Motorcycles"
	case MotorcycleParts:
		return "MotorcycleParts"
	case Bicycle:
		return "Bicycle"
	case BicycleParts:
		return "BicycleParts"
	case AgEquipment:
		return "AgEquipment"
	case AgEquipmentParts:
		return "AgEquipmentParts"
	default:
		return "Unknown"
	}
}

// DisplayName returns the human-readable display name for the category
func (c AdCategory) DisplayName() string {
	switch c {
	case Cars:
		return "Cars"
	case CarParts:
		return "Car Parts"
	case Motorcycles:
		return "Motorcycles"
	case MotorcycleParts:
		return "Motorcycle Parts"
	case Bicycle:
		return "Bicycle"
	case BicycleParts:
		return "Bicycle Parts"
	case AgEquipment:
		return "Ag Equipment"
	case AgEquipmentParts:
		return "Ag Equipment Parts"
	default:
		return "Unknown"
	}
}

// FromString converts a string to a Category
func FromString(s string) (AdCategory, error) {
	switch s {
	case "Cars":
		return Cars, nil
	case "CarParts":
		return CarParts, nil
	case "Motorcycles":
		return Motorcycles, nil
	case "MotorcycleParts":
		return MotorcycleParts, nil
	case "Bicycle":
		return Bicycle, nil
	case "BicycleParts":
		return BicycleParts, nil
	case "AgEquipment":
		return AgEquipment, nil
	case "AgEquipmentParts":
		return AgEquipmentParts, nil
	default:
		return 0, fmt.Errorf("unknown category: %s", s)
	}
}

// FromID converts a database ID to a Category
func FromID(id int) (AdCategory, error) {
	if id < 1 || id > int(AgEquipmentParts) {
		return 0, fmt.Errorf("invalid category ID: %d", id)
	}
	return AdCategory(id), nil
}

// ToID converts a Category to its database ID
func (c AdCategory) ToID() int {
	return int(c)
}

// UsesYear returns true if this category uses year information
func (c AdCategory) UsesYear() bool {
	switch c {
	case Cars, CarParts, Motorcycles, MotorcycleParts, AgEquipment, AgEquipmentParts:
		return true
	case Bicycle, BicycleParts:
		return false
	default:
		return false
	}
}

// UsesEngine returns true if this category uses engine information
func (c AdCategory) UsesEngine() bool {
	switch c {
	case Cars, CarParts, Motorcycles, MotorcycleParts:
		return true
	case Bicycle, BicycleParts, AgEquipment, AgEquipmentParts:
		return false
	default:
		return false
	}
}

// UsesSubcategory returns true if this category uses part subcategories
func (c AdCategory) UsesSubcategory() bool {
	switch c {
	case CarParts, MotorcycleParts, BicycleParts, AgEquipmentParts:
		return true
	case Cars, Motorcycles, Bicycle, AgEquipment:
		return false
	default:
		return false
	}
}

// GetVehicleTableName returns the name of the vehicle table for this category
func (c AdCategory) GetVehicleTableName() string {
	switch c {
	case Cars, CarParts:
		return "Car"
	case Motorcycles, MotorcycleParts:
		return "Motorcycle"
	case Bicycle, BicycleParts:
		return "Bicycle"
	case AgEquipment, AgEquipmentParts:
		return "AgEquipment"
	default:
		return ""
	}
}

// GetAssociationTableName returns the name of the ad-vehicle association table for this category
func (c AdCategory) GetAssociationTableName() string {
	switch c {
	case Cars:
		return "AdCar"
	case CarParts:
		return "AdCarPart"
	case Motorcycles:
		return "AdMotorcycle"
	case MotorcycleParts:
		return "AdMotorcyclePart"
	case Bicycle:
		return "AdBicycle"
	case BicycleParts:
		return "AdBicyclePart"
	case AgEquipment:
		return "AdAgEquipment"
	case AgEquipmentParts:
		return "AdAgEquipmentPart"
	default:
		return ""
	}
}

// ParseCategoryFromQuery parses a category string from query parameters, with fallback to CarParts
func ParseCategoryFromQuery(categoryStr string) AdCategory {
	if categoryStr == "" {
		return CarParts // Default fallback
	}

	category, err := FromString(categoryStr)
	if err != nil {
		return CarParts // Default fallback on error
	}

	return category
}

// ParseCategoryFromID parses a category from a database ID
func ParseCategoryFromID(categoryID int) AdCategory {
	if categoryID < 1 || categoryID > int(AgEquipmentParts) {
		return CarParts // Default fallback
	}
	return AdCategory(categoryID)
}

// GetAllAdCategories returns all available categories
func GetAllAdCategories() []AdCategory {
	return []AdCategory{Cars, CarParts, Motorcycles, MotorcycleParts, Bicycle, BicycleParts, AgEquipment, AgEquipmentParts}
}
