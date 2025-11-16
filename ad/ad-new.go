package ad

import (
	"database/sql"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/spec"
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

	if err := addFieldAssociations(tx, ad); err != nil {
		return 0, fmt.Errorf("failed to add field associations: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(adID), nil
}

// findSpec finds a spec ID from the appropriate spec table using dynamic field configuration
func findSpec(tx *sql.Tx, adCategoryID int, fieldValues map[string]string) (int, string, error) {
	// Get all chains to determine which spec table to use
	allChains, err := spec.GetAllChains(adCategoryID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get chains: %w", err)
	}

	// Find the first chain (lowest order) that matches the provided field values
	var targetChain []spec.FieldConfig
	var targetSpecType string
	minOrder := -1

	for order, chain := range allChains {
		// Check if this chain matches the field values
		matches := true
		for _, field := range chain {
			value, hasValue := fieldValues[field.Name]
			if !hasValue || value == "" {
				matches = false
				break
			}
		}
		if matches && (minOrder == -1 || order < minOrder) {
			minOrder = order
			targetChain = chain
			targetSpecType = spec.GetSpecTypeFromChain(chain)
		}
	}

	if len(targetChain) == 0 {
		return 0, "", fmt.Errorf("no matching chain found for field values: %v", fieldValues)
	}

	// Get spec table name
	specTableName := spec.GetSpecTableName(targetSpecType)

	// Build dynamic query
	query := fmt.Sprintf("SELECT spec.id FROM %s spec", specTableName)
	var joins []string
	var whereClauses []string
	var args []interface{}

	whereClauses = append(whereClauses, fmt.Sprintf("spec.ad_category_id = $%d", len(args)+1))
	args = append(args, adCategoryID)

	// Add JOINs and WHERE clauses for each field in the chain
	for _, fieldConfig := range targetChain {
		value, hasValue := fieldValues[fieldConfig.Name]
		if !hasValue || value == "" {
			continue
		}

		tableName := fieldConfig.TableName
		tableFieldName := fieldConfig.TableFieldName
		joins = append(joins, fmt.Sprintf("LEFT JOIN %s ON spec.%s = %s.id",
			tableName, tableFieldName, tableName))

		whereClauses = append(whereClauses,
			fmt.Sprintf("%s.value = $%d", tableName, len(args)+1))

		if fieldConfig.FieldType == "int" {
			intValue, err := strconv.Atoi(value)
			if err != nil {
				continue
			}
			args = append(args, intValue)
		} else {
			args = append(args, value)
		}
	}

	// Build final query
	if len(joins) > 0 {
		query += " " + strings.Join(joins, " ")
	}
	query += " WHERE " + strings.Join(whereClauses, " AND ")

	var specID int
	err = tx.QueryRow(query, args...).Scan(&specID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", fmt.Errorf("spec not found for field values: %v", fieldValues)
		}
		return 0, "", fmt.Errorf("error looking up spec: %w", err)
	}
	return specID, targetSpecType, nil
}

// addFieldAssociations handles spec associations dynamically based on category field configuration
func addFieldAssociations(tx *sql.Tx, ad AdDetail) error {
	// Get category fields
	fields := spec.GetCategoryFields(ad.AdCategoryID)

	// Convert Fields to FieldConfigs
	fieldConfigs := make([]spec.FieldConfig, len(fields))
	for i, field := range fields {
		fieldConfigs[i] = field.Config()
	}

	// Extract single-value fields from AdDetail.SpecData for spec lookup
	fieldValueMap := buildFieldValueMap(ad, fieldConfigs)

	// Get multi-value fields (fields that can have multiple selections)
	multiValueFields := getMultiValueFields(ad, fieldConfigs)

	// Generate all combinations of multi-value fields
	combinations := generateFieldCombinations(multiValueFields)

	// If no multi-value fields, process single combination
	if len(combinations) == 0 {
		combinations = []map[string]string{{}}
	}

	// Process each combination
	for _, combination := range combinations {
		// Merge combination with base field values
		combinedValues := make(map[string]string)
		maps.Copy(combinedValues, fieldValueMap)
		maps.Copy(combinedValues, combination)

		// Find spec for this combination
		specID, specType, err := findSpec(tx, ad.AdCategoryID, combinedValues)
		if err != nil {
			return err
		}

		// Insert association
		if err := insertSpecAssociation(tx, ad.ID, specID, specType); err != nil {
			return err
		}
	}

	return nil
}

// buildFieldValueMap builds a map of field values from AdDetail SpecData
func buildFieldValueMap(ad AdDetail, fieldConfigs []spec.FieldConfig) map[string]string {
	values := make(map[string]string)

	for _, fieldConfig := range fieldConfigs {
		// Only handle single-value fields here (multi-value handled separately)
		if fieldConfig.FieldMultiValue {
			continue
		}

		// Extract value from SpecData
		fieldValues, exists := ad.SpecData[fieldConfig.Name]
		if exists && len(fieldValues) > 0 && fieldValues[0] != "" {
			values[fieldConfig.Name] = fieldValues[0]
		}
	}

	return values
}

// getMultiValueFields returns fields that have multiple values from AdDetail SpecData
func getMultiValueFields(ad AdDetail, fieldConfigs []spec.FieldConfig) map[string][]string {
	multiValueFields := make(map[string][]string)

	// Initialize SpecData if nil
	if ad.SpecData == nil {
		ad.SpecData = make(map[string][]string)
	}

	for _, fieldConfig := range fieldConfigs {
		// Only handle multi-value fields
		if !fieldConfig.FieldMultiValue {
			continue
		}

		// Extract values from SpecData
		fieldValues, exists := ad.SpecData[fieldConfig.Name]
		if exists && len(fieldValues) > 0 {
			multiValueFields[fieldConfig.Name] = fieldValues
		}
	}

	return multiValueFields
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
		idColumn = "spec_vehicle_id"
	case "agricultural":
		tableName = "AdSpecAgricultural"
		idColumn = "spec_agricultural_id"
	case "bicycle":
		tableName = "AdSpecBicycle"
		idColumn = "spec_bicycle_id"
	case "part":
		tableName = "AdSpecPart"
		idColumn = "spec_part_id"
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
