package spec

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/parts-pile/site/db"
)

// BuildFieldQuery builds a generic SQL query for getting field values
// parentFilters maps parent field names to their values (e.g., {"make": "BMW", "year": "2020"})
func BuildFieldQuery(adCategoryID int, fieldName string, parentFilters map[string]string) (string, []interface{}, error) {
	fieldDef, exists := GetFieldDefinition(fieldName)
	if !exists {
		return "", nil, fmt.Errorf("field definition not found: %s", fieldName)
	}

	// Get category fields to determine field order and dependencies
	categoryFields := GetCategoryFields(adCategoryID)

	// Find the current field's position in the field list
	currentFieldIndex := -1
	for i, f := range categoryFields {
		if f.Name() == fieldName {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return "", nil, fmt.Errorf("field %s not found in category %d", fieldName, adCategoryID)
	}

	// Build parent field filters - only include fields that come before current field
	parentFields := categoryFields[:currentFieldIndex]

	// Unified query building - all fields work the same way
	tableName := fieldDef.TableName
	tableFieldName := fieldDef.TableFieldName

	// Determine which spec table(s) to query from based on the field's chain
	// Get all chains to find which one contains this field
	allChains, err := GetAllChains(adCategoryID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get chains: %w", err)
	}

	// Find the chain that contains this field
	var targetChain []FieldConfig
	for _, chain := range allChains {
		for _, field := range chain {
			if field.Name == fieldName {
				targetChain = chain
				break
			}
		}
		if len(targetChain) > 0 {
			break
		}
	}

	if len(targetChain) == 0 {
		return "", nil, fmt.Errorf("field %s not found in any chain", fieldName)
	}

	// Get spec table name for this chain
	specType := GetSpecTypeFromChain(targetChain)
	specTableName := GetSpecTableName(specType)

	// Use table alias for cleaner queries
	// All fields use the same value column
	selectClause := fmt.Sprintf("SELECT DISTINCT t.value FROM %s t", tableName)

	// Build JOINs for the appropriate spec table and all parent fields
	joins := []string{fmt.Sprintf("JOIN %s spec ON t.id = spec.%s", specTableName, tableFieldName)}
	args := []interface{}{}

	// Add JOINs for parent fields
	for _, parentField := range parentFields {
		parentDef, exists := GetFieldDefinition(parentField.Name)
		if !exists {
			continue
		}

		parentTableName := parentDef.TableName
		parentTableFieldName := parentDef.TableFieldName
		joins = append(joins, fmt.Sprintf("JOIN %s ON spec.%s = %s.id",
			parentTableName, parentTableFieldName, parentTableName))
	}

	query := selectClause + " " + strings.Join(joins, " ")

	// Build WHERE clause
	whereClauses := []string{
		fmt.Sprintf("spec.ad_category_id = $%d", len(args)+1),
		fmt.Sprintf("t.ad_category_id = $%d", len(args)+2),
	}
	args = append(args, adCategoryID, adCategoryID)

	// Add filters for parent fields
	for _, parentField := range parentFields {
		parentValue, hasValue := parentFilters[parentField.Name()]
		if !hasValue || parentValue == "" {
			continue
		}

		parentDef, exists := GetFieldDefinition(parentField.Name())
		if !exists {
			continue
		}

		parentTableName := parentDef.TableName
		if parentDef.FieldType == "int" {
			// For integer fields, convert string to int and compare with integer column
			intValue, err := strconv.Atoi(parentValue)
			if err != nil {
				continue
			}
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.value = $%d", parentTableName, len(args)+1))
			args = append(args, intValue)
		} else {
			// For string fields, compare with value column
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.value = $%d", parentTableName, len(args)+1))
			args = append(args, parentValue)
		}
	}

	query += " WHERE " + strings.Join(whereClauses, " AND ")

	// Order by clause
	if fieldDef.FieldType == "int" {
		query += " ORDER BY t.value DESC"
	} else {
		query += " ORDER BY t.value"
	}

	return query, args, nil
}

// BuildFieldQueryWithIntersection builds SQL query for getting field values
// Supports both single and multiple parent filter values using intersection logic
// parentFilters can contain comma-separated values (e.g., "2020,2021") or single values
func BuildFieldQueryWithIntersection(adCategoryID int, field Field, parentFilters map[string]string) (string, []interface{}, error) {
	fieldDef := field.Name
	// Get category fields to determine field order
	categoryFields := GetCategoryFields(adCategoryID)

	// Find the current field's position
	currentFieldIndex := -1
	for i, f := range categoryFields {
		if f.Name() == fieldDef.Name {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return "", nil, fmt.Errorf("field %s not found in category %d", fieldDef.Name(), adCategoryID)
	}

	parentFields := categoryFields[:currentFieldIndex]

	tableName := fieldDef.TableName()
	tableFieldName := fieldDef.TableFieldName()

	// Convert parent filters to array format (handle comma-separated values)
	// Only include parent fields that have values - work with whatever data is available
	parentFiltersArray := make(map[string][]string)
	for _, parentField := range parentFields {
		parentValue, exists := parentFilters[parentField.Name()]
		if !exists || parentValue == "" {
			// Skip parent fields without values - they're optional
			continue
		}
		// Split comma-separated values or use single value
		if strings.Contains(parentValue, ",") {
			parentFiltersArray[parentField.Name()] = strings.Split(parentValue, ",")
		} else {
			parentFiltersArray[parentField.Name()] = []string{parentValue}
		}
	}

	// Build a single query using IN clauses for each parent field
	// We'll use GROUP BY/HAVING to ensure intersection (all combinations)
	// Calculate total expected combinations only for fields that have values
	totalCombinations := 1
	for _, values := range parentFiltersArray {
		if len(values) > 0 {
			totalCombinations *= len(values)
		}
	}

	// Determine which spec table(s) to query from based on the field's chain
	// Get all chains to find which one contains this field
	allChains, err := GetAllChains(adCategoryID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get chains: %w", err)
	}

	// Find the chain that contains this field
	var targetChain []FieldConfig
	for _, chain := range allChains {
		for _, field := range chain {
			if field.Name() == fieldDef.Name() {
				targetChain = chain
				break
			}
		}
		if len(targetChain) > 0 {
			break
		}
	}

	if len(targetChain) == 0 {
		return "", nil, fmt.Errorf("field %s not found in any chain", fieldDef.Name())
	}

	// Get spec table name for this chain
	specType := GetSpecTypeFromChain(targetChain)
	specTableName := GetSpecTableName(specType)

	// If no parent filters provided, return all values for this field
	// This happens when loading filter dropdowns without parent selections
	if len(parentFiltersArray) == 0 {
		// No parent filters provided - return all values for this field
		// This happens when loading the first field in a chain
		// For filters, show all available values from the table, not just those in Specification
		selectClause := fmt.Sprintf(`SELECT DISTINCT t.value FROM %s t
	WHERE t.ad_category_id = $1`, tableName)
		query := selectClause
		args := []interface{}{adCategoryID}

		// Order by clause
		if fieldDef.FieldType() == "int" {
			query += " ORDER BY t.value DESC"
		} else {
			query += " ORDER BY t.value"
		}

		return query, args, nil
	}

	// Determine SELECT clause - all fields are TEXT now
	selectClause := fmt.Sprintf(`SELECT DISTINCT t.value FROM %s t
	WHERE t.ad_category_id = $1 AND t.id IN (
		SELECT spec.%s FROM %s spec`, tableName, tableFieldName, specTableName)
	query := selectClause
	args := []interface{}{adCategoryID}

	// Add JOINs only for parent fields that have values
	for fieldName := range parentFiltersArray {
		parentDef, exists := GetFieldDefinition(fieldName)
		if !exists {
			continue
		}

		parentTableName := parentDef.TableName
		parentTableFieldName := parentDef.TableFieldName
		query += fmt.Sprintf(" JOIN %s ON spec.%s = %s.id",
			parentTableName, parentTableFieldName, parentTableName)
	}

	// Build WHERE clause with IN clauses for each parent field that has values
	whereClauses := []string{fmt.Sprintf("spec.ad_category_id = $%d", len(args)+1)}
	args = append(args, adCategoryID)

	for fieldName, values := range parentFiltersArray {
		if len(values) == 0 {
			continue
		}

		parentDef, exists := GetFieldDefinition(fieldName)
		if !exists {
			continue
		}

		parentTableName := parentDef.TableName
		if parentDef.FieldType == "int" {
			// Convert string values to ints for IN clause
			var intValues []int
			var placeholders []string
			for _, val := range values {
				intValue, err := strconv.Atoi(val)
				if err == nil {
					intValues = append(intValues, intValue)
					placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)+1))
					args = append(args, intValue)
				}
			}
			if len(intValues) > 0 {
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.value IN (%s)", parentTableName, strings.Join(placeholders, ",")))
			}
		} else {
			// String fields - use IN clause
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = fmt.Sprintf("$%d", len(args)+1)
				args = append(args, values[i])
			}
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.value IN (%s)", parentTableName, strings.Join(placeholders, ",")))
		}
	}

	query += " WHERE " + strings.Join(whereClauses, " AND ")

	// Use GROUP BY and HAVING to ensure intersection (all combinations)
	// Group by the field we're selecting, and ensure it appears in all combinations
	query += fmt.Sprintf(` GROUP BY spec.%s
	HAVING COUNT(DISTINCT `, tableFieldName)

	// Build DISTINCT clause for all parent field combinations (only fields with values)
	distinctParts := []string{}
	for fieldName := range parentFiltersArray {
		parentDef, exists := GetFieldDefinition(fieldName)
		if !exists {
			continue
		}
		parentTableFieldName := parentDef.TableFieldName
		distinctParts = append(distinctParts, fmt.Sprintf("spec.%s", parentTableFieldName))
	}

	if len(distinctParts) > 0 {
		query += strings.Join(distinctParts, ", ") + fmt.Sprintf(") = $%d", len(args)+1)
		args = append(args, totalCombinations)
	} else {
		// No parent fields, just close the query
		query += "1) = 1"
	}

	query += `)`
	// Order by the original value (before casting) for proper sorting
	if fieldDef.FieldType == "int" {
		query += " ORDER BY t.value DESC"
	} else {
		query += " ORDER BY t.value"
	}

	return query, args, nil
}

// GetFieldValues returns values for a field given parent filters
// This is a convenience wrapper around the Field method
func GetFieldValues(adCategoryID int, fieldName string, parentValues map[string]string) ([]string, error) {
	field, err := GetCategoryField(adCategoryID, fieldName)
	if err != nil {
		return nil, fmt.Errorf("field not found: %s", fieldName)
	}

	return field.GetValues(parentValues)
}

// GetAdFieldValuesForAdIDs returns field values filtered by ad IDs and parent values
// This is used for search mode in tree view where we have a filtered set of ad IDs
func GetAdFieldValuesForAdIDs(adCategoryID int, fieldName string, adIDs []int, parentValues map[string]string) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	fieldDef, exists := GetFieldDefinition(fieldName)
	if !exists {
		return nil, fmt.Errorf("field definition not found: %s", fieldName)
	}

	// Get category fields to determine field order and dependencies
	categoryFields := GetCategoryFields(adCategoryID)

	// Find the current field's position in the field list
	currentFieldIndex := -1
	for i, f := range categoryFields {
		if f.Name == fieldName {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return nil, fmt.Errorf("field %s not found in category %d", fieldName, adCategoryID)
	}

	// Build parent field filters - only include fields that come before current field
	parentFields := categoryFields[:currentFieldIndex]

	// Unified query building - all fields work the same way
	tableName := fieldDef.TableName

	// All fields use the same value column - use table alias for simplicity
	selectClause := fmt.Sprintf("SELECT DISTINCT t.value FROM %s t", tableName)

	// Build queries for each AdSpec* table using UNION ALL
	adSpecTables := []struct {
		adSpecTable string
		specTable   string
		idColumn    string
	}{
		{"AdSpecVehicle", "SpecVehicle", "spec_vehicle_id"},
		{"AdSpecAgricultural", "SpecAgricultural", "spec_agricultural_id"},
		{"AdSpecBicycle", "SpecBicycle", "spec_bicycle_id"},
		{"AdSpecPart", "SpecPart", "spec_part_id"},
	}

	var unionQueries []string
	var allArgs []interface{}

	// Build placeholders for ad IDs
	adIDPlaceholders := make([]string, len(adIDs))
	for i := range adIDs {
		adIDPlaceholders[i] = fmt.Sprintf("$%d", len(allArgs)+1)
		allArgs = append(allArgs, adIDs[i])
	}

	for _, adSpecTable := range adSpecTables {
		var joins []string
		var whereConditions []string

		// Join to spec table
		joins = append(joins, fmt.Sprintf("JOIN %s spec ON %s_ad.%s = spec.id",
			adSpecTable.specTable, adSpecTable.specTable, adSpecTable.idColumn))
		joins = append(joins, fmt.Sprintf("JOIN %s %s_ad ON %s_ad.ad_id IN (%s)",
			adSpecTable.adSpecTable, adSpecTable.specTable, adSpecTable.specTable, strings.Join(adIDPlaceholders, ",")))

		// Add JOINs for parent fields
		for _, parentField := range parentFields {
			parentDef, exists := GetFieldDefinition(parentField.Name)
			if !exists {
				continue
			}
			parentTableName := parentDef.TableName
			parentTableFieldName := parentDef.TableFieldName
			joins = append(joins, fmt.Sprintf("JOIN %s ON spec.%s = %s.id",
				parentTableName, parentTableFieldName, parentTableName))
		}

		// Build WHERE clause with parent filters
		paramNum := len(allArgs) + 1
		for _, parentField := range parentFields {
			parentDef, exists := GetFieldDefinition(parentField.Name)
			if !exists {
				continue
			}
			parentValue, exists := parentValues[parentField.Name()]
			if !exists {
				continue
			}

			parentTableName := parentDef.TableName
			if parentDef.FieldType == "int" {
				intValue, err := strconv.Atoi(parentValue)
				if err != nil {
					continue
				}
				whereConditions = append(whereConditions, fmt.Sprintf("%s.value = $%d", parentTableName, paramNum))
				allArgs = append(allArgs, intValue)
				paramNum++
			} else {
				whereConditions = append(whereConditions, fmt.Sprintf("%s.value = $%d", parentTableName, paramNum))
				allArgs = append(allArgs, parentValue)
				paramNum++
			}
		}

		// Build query for this spec type
		query := fmt.Sprintf("%s %s", selectClause, strings.Join(joins, " "))
		if len(whereConditions) > 0 {
			query += " WHERE " + strings.Join(whereConditions, " AND ")
		}
		unionQueries = append(unionQueries, query)
	}

	// Build ORDER BY clause
	var orderBy string
	if fieldDef.FieldType == "int" {
		orderBy = "ORDER BY t.value DESC"
	} else {
		orderBy = "ORDER BY t.value"
	}

	// Combine with UNION ALL
	query := fmt.Sprintf("%s %s", strings.Join(unionQueries, " UNION ALL "), orderBy)

	// Execute query
	var results []string
	if fieldDef.FieldType == "int" {
		var intResults []int
		err := db.Select(&intResults, query, allArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query field values: %w", err)
		}
		// Convert integers to strings
		for _, val := range intResults {
			results = append(results, fmt.Sprintf("%d", val))
		}
	} else {
		err := db.Select(&results, query, allArgs...)
		if err != nil {
			return nil, fmt.Errorf("failed to query field values: %w", err)
		}
	}

	return results, nil
}
