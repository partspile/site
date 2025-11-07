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
	categoryFields, err := GetCategoryFields(adCategoryID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get category fields: %w", err)
	}

	// Find the current field's position in the field list
	currentFieldIndex := -1
	for i, f := range categoryFields {
		if f.Name == fieldName {
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

	// Determine SELECT clause based on field type
	var selectClause string
	if fieldDef.FieldType == "int" {
		// For integer fields, select the integer value directly
		selectClause = fmt.Sprintf("SELECT DISTINCT %s.%s FROM %s", tableName, getValueColumnName(tableName), tableName)
	} else {
		// For string fields, select the name column
		selectClause = fmt.Sprintf("SELECT DISTINCT %s.name FROM %s", tableName, tableName)
	}

	// Build JOINs for Specification table and all parent fields
	joins := []string{fmt.Sprintf("JOIN Specification spec ON %s.id = spec.%s", tableName, tableFieldName)}
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
		"spec.ad_category_id = ?",
		fmt.Sprintf("%s.ad_category_id = ?", tableName),
	}
	args = append(args, adCategoryID, adCategoryID)

	// Add filters for parent fields
	for _, parentField := range parentFields {
		parentValue, hasValue := parentFilters[parentField.Name]
		if !hasValue || parentValue == "" {
			continue
		}

		parentDef, exists := GetFieldDefinition(parentField.Name)
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
			valueCol := getValueColumnName(parentTableName)
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.%s = ?", parentTableName, valueCol))
			args = append(args, intValue)
		} else {
			// For string fields, compare with name column
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.name = ?", parentTableName))
			args = append(args, parentValue)
		}
	}

	query += " WHERE " + strings.Join(whereClauses, " AND ")
	
	// Order by clause
	if fieldDef.FieldType == "int" {
		valueCol := getValueColumnName(tableName)
		query += fmt.Sprintf(" ORDER BY %s.%s DESC", tableName, valueCol)
	} else {
		query += fmt.Sprintf(" ORDER BY %s.name", tableName)
	}

	return query, args, nil
}

// getValueColumnName returns the column name for the value in a table
// For Year table, it's "year", for other tables it's "name"
func getValueColumnName(tableName string) string {
	if tableName == "Year" {
		return "year"
	}
	return "name"
}

// BuildFieldQueryWithIntersection builds query for fields that need intersection logic
// (e.g., models that exist in ALL selected years, engines that exist in ALL year-model combinations)
func BuildFieldQueryWithIntersection(adCategoryID int, fieldName string, parentFilters map[string][]string) (string, []interface{}, error) {
	fieldDef, exists := GetFieldDefinition(fieldName)
	if !exists {
		return "", nil, fmt.Errorf("field definition not found: %s", fieldName)
	}

	// Intersection queries work for all field types

	// Get category fields to determine field order
	categoryFields, err := GetCategoryFields(adCategoryID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get category fields: %w", err)
	}

	// Find the current field's position
	currentFieldIndex := -1
	for i, f := range categoryFields {
		if f.Name == fieldName {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return "", nil, fmt.Errorf("field %s not found in category %d", fieldName, adCategoryID)
	}

	parentFields := categoryFields[:currentFieldIndex]

	// Build combinations from parent filters
	// For example, if years=["2020","2021"] and models=["M3","X5"], we need:
	// (year=2020 AND model=M3) AND (year=2020 AND model=X5) AND (year=2021 AND model=M3) AND (year=2021 AND model=X5)
	combinations := buildCombinations(parentFields, parentFilters)

	if len(combinations) == 0 {
		return "", nil, fmt.Errorf("no valid combinations found")
	}

	tableName := fieldDef.TableName
	tableFieldName := fieldDef.TableFieldName
	
	// Determine SELECT clause based on field type
	var selectClause string
	if fieldDef.FieldType == "int" {
		valueCol := getValueColumnName(tableName)
		selectClause = fmt.Sprintf(`SELECT DISTINCT %s.%s FROM %s
	WHERE %s.id IN (
		SELECT spec.%s FROM Specification spec`, tableName, valueCol, tableName, tableName, tableFieldName)
	} else {
		selectClause = fmt.Sprintf(`SELECT DISTINCT %s.name FROM %s
	WHERE %s.id IN (
		SELECT spec.%s FROM Specification spec`, tableName, tableName, tableName, tableFieldName)
	}
	query := selectClause

	// Add JOINs for parent fields
	for _, parentField := range parentFields {
		parentDef, exists := GetFieldDefinition(parentField.Name)
		if !exists {
			continue
		}

		parentTableName := parentDef.TableName
		parentTableFieldName := parentDef.TableFieldName
		query += fmt.Sprintf(" JOIN %s ON spec.%s = %s.id",
			parentTableName, parentTableFieldName, parentTableName)
	}

	// Build WHERE clause for first combination
	whereClauses := []string{"spec.ad_category_id = ?"}
	args := []interface{}{adCategoryID}

	firstCombination := combinations[0]
	for _, parentField := range parentFields {
		value, exists := firstCombination[parentField.Name]
		if !exists {
			continue
		}

		parentDef, exists := GetFieldDefinition(parentField.Name)
		if !exists {
			continue
		}

		parentTableName := parentDef.TableName
		if parentDef.FieldType == "int" {
			// For integer fields, convert string to int
			intValue, err := strconv.Atoi(value)
			if err == nil {
				valueCol := getValueColumnName(parentTableName)
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.%s = ?", parentTableName, valueCol))
				args = append(args, intValue)
			}
		} else {
			// For string fields, compare with name column
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.name = ?", parentTableName))
			args = append(args, value)
		}
	}

	query += " WHERE " + strings.Join(whereClauses, " AND ")

	// Add intersection conditions for remaining combinations
	for i := 1; i < len(combinations); i++ {
		query += fmt.Sprintf(`) AND %s.id IN (
			SELECT spec.%s FROM Specification spec`, tableName, tableFieldName)

		// Add JOINs again
		for _, parentField := range parentFields {
			parentDef, exists := GetFieldDefinition(parentField.Name)
			if !exists {
				continue
			}

			parentTableName := parentDef.TableName
			parentTableFieldName := parentDef.TableFieldName
			query += fmt.Sprintf(" JOIN %s ON spec.%s = %s.id",
				parentTableName, parentTableFieldName, parentTableName)
		}

		whereClauses = []string{"spec.ad_category_id = ?"}
		args = append(args, adCategoryID)

		combination := combinations[i]
		for _, parentField := range parentFields {
			value, exists := combination[parentField.Name]
			if !exists {
				continue
			}

			parentDef, exists := GetFieldDefinition(parentField.Name)
			if !exists {
				continue
			}

			parentTableName := parentDef.TableName
			if parentDef.FieldType == "int" {
				// For integer fields, convert string to int
				intValue, err := strconv.Atoi(value)
				if err == nil {
					valueCol := getValueColumnName(parentTableName)
					whereClauses = append(whereClauses,
						fmt.Sprintf("%s.%s = ?", parentTableName, valueCol))
					args = append(args, intValue)
				}
			} else {
				// For string fields, compare with name column
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.name = ?", parentTableName))
				args = append(args, value)
			}
		}

		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query += `)`
	if fieldDef.FieldType == "int" {
		valueCol := getValueColumnName(tableName)
		query += fmt.Sprintf(" ORDER BY %s.%s DESC", tableName, valueCol)
	} else {
		query += fmt.Sprintf(" ORDER BY %s.name", tableName)
	}

	return query, args, nil
}

// buildCombinations builds all combinations from parent filters
// For example: years=["2020","2021"], models=["M3","X5"] -> 4 combinations
func buildCombinations(parentFields []FieldConfig, parentFilters map[string][]string) []map[string]string {
	if len(parentFields) == 0 {
		return []map[string]string{{}}
	}

	// Get values for each parent field
	fieldValues := make([][]string, len(parentFields))
	for i, field := range parentFields {
		values, exists := parentFilters[field.Name]
		if !exists || len(values) == 0 {
			return []map[string]string{} // No values means no combinations
		}
		fieldValues[i] = values
	}

	// Generate all combinations using cartesian product
	var combinations []map[string]string
	generateCombinations(parentFields, fieldValues, 0, map[string]string{}, &combinations)

	return combinations
}

// generateCombinations recursively generates all combinations
func generateCombinations(parentFields []FieldConfig, fieldValues [][]string, depth int, current map[string]string, result *[]map[string]string) {
	if depth == len(parentFields) {
		// Make a copy of current map
		combination := make(map[string]string)
		for k, v := range current {
			combination[k] = v
		}
		*result = append(*result, combination)
		return
	}

	field := parentFields[depth]
	for _, value := range fieldValues[depth] {
		current[field.Name] = value
		generateCombinations(parentFields, fieldValues, depth+1, current, result)
	}
}

// GetFieldValues returns values for a field given parent filters
func GetFieldValues(adCategoryID int, fieldName string, parentValues map[string]string) ([]string, error) {
	fieldDef, exists := GetFieldDefinition(fieldName)
	if !exists {
		return nil, fmt.Errorf("field definition not found: %s", fieldName)
	}

	// Check if we need intersection logic (for fields like model, engine)
	categoryFields, err := GetCategoryFields(adCategoryID)
	if err != nil {
		return nil, err
	}

	// Find current field index
	currentFieldIndex := -1
	for i, f := range categoryFields {
		if f.Name == fieldName {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return nil, fmt.Errorf("field %s not found in category", fieldName)
	}

	// Check if any parent field has multiple values (needs intersection)
	needsIntersection := false
	parentFiltersArray := make(map[string][]string)
	for parentName, parentValue := range parentValues {
		// Check if this is a multi-value field (like years, models, engines)
		// For now, assume if parentValue contains comma, it's multiple values
		if strings.Contains(parentValue, ",") {
			needsIntersection = true
			parentFiltersArray[parentName] = strings.Split(parentValue, ",")
		} else {
			parentFiltersArray[parentName] = []string{parentValue}
		}
	}

	var query string
	var args []interface{}
	var err2 error

	if needsIntersection {
		query, args, err2 = BuildFieldQueryWithIntersection(adCategoryID, fieldName, parentFiltersArray)
	} else {
		query, args, err2 = BuildFieldQuery(adCategoryID, fieldName, parentValues)
	}

	if err2 != nil {
		return nil, err2
	}

	// Execute query - convert based on field type
	if fieldDef.FieldType == "int" {
		var intValues []int
		err := db.Select(&intValues, query, args...)
		if err != nil {
			return nil, err
		}
		var stringValues []string
		for _, val := range intValues {
			stringValues = append(stringValues, strconv.Itoa(val))
		}
		return stringValues, nil
	} else {
		var values []string
		err := db.Select(&values, query, args...)
		if err != nil {
			return nil, err
		}
		return values, nil
	}
}

