package ad

import (
	"database/sql"
	"fmt"
	"maps"

	"github.com/parts-pile/site/db"
)

// AddAd creates a new ad in the database and returns the ad ID
func AddAd(ad AdDetail) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var adID int
	err = tx.QueryRow(`
		INSERT INTO Ad (
			ad_category_id, title, description, price, 
			user_id, location_id, image_count, has_vector
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		ad.AdCategoryID, ad.Title, ad.Description, ad.Price,
		ad.UserID, ad.LocationID, ad.ImageCount, ad.HasVector).Scan(&adID)
	if err != nil {
		return 0, fmt.Errorf("failed to insert ad: %w", err)
	}

	ad.ID = adID

	if err := saveSpecData(tx, ad); err != nil {
		return 0, fmt.Errorf("failed to save spec data: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(adID), nil
}

// saveSpecData saves spec associations dynamically based on category field configuration
func saveSpecData(tx *sql.Tx, ad AdDetail) error {
	fields := GetSpecFields(ad.AdCategoryID)

	// TODO
}

// generateFieldCombinations generates all combinations of multi-value fields
func generateFieldCombinations(multiValueFields map[string][]string) []map[string]string {
	if len(multiValueFields) == 0 {
		return []map[string]string{}
	}

	// Get field names and their values
	fieldNames := make([]string, 0, len(multiValueFields))
	fieldValues := make([][]string, 0, len(multiValueFields))
	for name, values := range multiValueFields {
		fieldNames = append(fieldNames, name)
		fieldValues = append(fieldValues, values)
	}

	var combinations []map[string]string
	generateCombinationsRecursive(fieldNames, fieldValues, 0, make(map[string]string), &combinations)
	return combinations
}

// generateCombinationsRecursive recursively generates all combinations
func generateCombinationsRecursive(fieldNames []string, fieldValues [][]string, depth int, current map[string]string, result *[]map[string]string) {
	if depth == len(fieldNames) {
		combination := make(map[string]string)
		maps.Copy(combination, current)
		*result = append(*result, combination)
		return
	}

	fieldName := fieldNames[depth]
	for _, value := range fieldValues[depth] {
		current[fieldName] = value
		generateCombinationsRecursive(fieldNames, fieldValues, depth+1, current, result)
	}
}

// insertSpecAssociation inserts the ad-spec association record into the appropriate AdSpec* table
func insertSpecAssociation(tx *sql.Tx, adID int, specID int, specType string) error {
	var tableName, idColumn string
	switch specType {
	case "vehicle":
		tableName = "AdSpecVehicle"
		idColumn = "spec_id"
	case "agricultural":
		tableName = "AdSpecAgricultural"
		idColumn = "spec_id"
	case "bicycle":
		tableName = "AdSpecBicycle"
		idColumn = "spec_id"
	case "part":
		tableName = "AdSpecPart"
		idColumn = "spec_id"
	default:
		return fmt.Errorf("unknown spec type: %s", specType)
	}

	query := fmt.Sprintf("INSERT INTO %s (ad_id, %s) VALUES ($1, $2) ON CONFLICT DO NOTHING", tableName, idColumn)
	_, err := tx.Exec(query, adID, specID)
	if err != nil {
		return fmt.Errorf("failed to insert %s association: %w", tableName, err)
	}
	return nil
}
