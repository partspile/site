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
		selectClause = fmt.Sprintf("SELECT DISTINCT %s.%s FROM %s", tableName, fieldDef.ValueColumnName, tableName)
	} else {
		// For string fields, select the value column
		selectClause = fmt.Sprintf("SELECT DISTINCT %s.%s FROM %s", tableName, fieldDef.ValueColumnName, tableName)
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
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.%s = ?", parentTableName, parentDef.ValueColumnName))
			args = append(args, intValue)
		} else {
			// For string fields, compare with value column
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.%s = ?", parentTableName, parentDef.ValueColumnName))
			args = append(args, parentValue)
		}
	}

	query += " WHERE " + strings.Join(whereClauses, " AND ")

	// Order by clause
	if fieldDef.FieldType == "int" {
		query += fmt.Sprintf(" ORDER BY %s.%s DESC", tableName, fieldDef.ValueColumnName)
	} else {
		query += fmt.Sprintf(" ORDER BY %s.%s", tableName, fieldDef.ValueColumnName)
	}

	return query, args, nil
}

// BuildFieldQueryWithIntersection builds SQL query for getting field values
// Supports both single and multiple parent filter values using intersection logic
// parentFilters can contain comma-separated values (e.g., "2020,2021") or single values
func BuildFieldQueryWithIntersection(adCategoryID int, fieldDef FieldConfig, parentFilters map[string]string) (string, []interface{}, error) {
	// Get category fields to determine field order
	categoryFields, err := GetCategoryFields(adCategoryID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get category fields: %w", err)
	}

	// Find the current field's position
	currentFieldIndex := -1
	for i, f := range categoryFields {
		if f.Name == fieldDef.Name {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return "", nil, fmt.Errorf("field %s not found in category %d", fieldDef.Name, adCategoryID)
	}

	parentFields := categoryFields[:currentFieldIndex]

	tableName := fieldDef.TableName
	tableFieldName := fieldDef.TableFieldName

	// Convert parent filters to array format (handle comma-separated values)
	parentFiltersArray := make(map[string][]string)
	for _, parentField := range parentFields {
		parentValue, exists := parentFilters[parentField.Name]
		if !exists || parentValue == "" {
			return "", nil, fmt.Errorf("no values for parent field %s", parentField.Name)
		}
		// Split comma-separated values or use single value
		if strings.Contains(parentValue, ",") {
			parentFiltersArray[parentField.Name] = strings.Split(parentValue, ",")
		} else {
			parentFiltersArray[parentField.Name] = []string{parentValue}
		}
	}

	// Build a single query using IN clauses for each parent field
	// We'll use GROUP BY/HAVING to ensure intersection (all combinations)
	// Calculate total expected combinations
	totalCombinations := 1
	for _, parentField := range parentFields {
		values := parentFiltersArray[parentField.Name]
		totalCombinations *= len(values)
	}

	if totalCombinations == 0 {
		return "", nil, fmt.Errorf("no valid combinations found")
	}

	// Determine SELECT clause - always return as text for consistency
	var selectClause string
	if fieldDef.FieldType == "int" {
		selectClause = fmt.Sprintf(`SELECT DISTINCT CAST(%s.%s AS TEXT) FROM %s
	WHERE %s.id IN (
		SELECT spec.%s FROM Specification spec`, tableName, fieldDef.ValueColumnName, tableName, tableName, tableFieldName)
	} else {
		selectClause = fmt.Sprintf(`SELECT DISTINCT %s.%s FROM %s
	WHERE %s.id IN (
		SELECT spec.%s FROM Specification spec`, tableName, fieldDef.ValueColumnName, tableName, tableName, tableFieldName)
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

	// Build WHERE clause with IN clauses for each parent field
	whereClauses := []string{"spec.ad_category_id = ?"}
	args := []interface{}{adCategoryID}

	for _, parentField := range parentFields {
		values := parentFiltersArray[parentField.Name]
		if len(values) == 0 {
			continue
		}

		parentDef, exists := GetFieldDefinition(parentField.Name)
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
					placeholders = append(placeholders, "?")
					args = append(args, intValue)
				}
			}
			if len(intValues) > 0 {
				whereClauses = append(whereClauses,
					fmt.Sprintf("%s.%s IN (%s)", parentTableName, parentDef.ValueColumnName, strings.Join(placeholders, ",")))
			}
		} else {
			// String fields - use IN clause
			placeholders := make([]string, len(values))
			for i := range values {
				placeholders[i] = "?"
				args = append(args, values[i])
			}
			whereClauses = append(whereClauses,
				fmt.Sprintf("%s.%s IN (%s)", parentTableName, parentDef.ValueColumnName, strings.Join(placeholders, ",")))
		}
	}

	query += " WHERE " + strings.Join(whereClauses, " AND ")

	// Use GROUP BY and HAVING to ensure intersection (all combinations)
	// Group by the field we're selecting, and ensure it appears in all combinations
	query += fmt.Sprintf(` GROUP BY spec.%s
	HAVING COUNT(DISTINCT `, tableFieldName)

	// Build DISTINCT clause for all parent field combinations
	distinctParts := []string{}
	for _, parentField := range parentFields {
		parentDef, exists := GetFieldDefinition(parentField.Name)
		if !exists {
			continue
		}
		parentTableFieldName := parentDef.TableFieldName
		distinctParts = append(distinctParts, fmt.Sprintf("spec.%s", parentTableFieldName))
	}

	if len(distinctParts) > 0 {
		query += strings.Join(distinctParts, ", ") + ") = ?"
		args = append(args, totalCombinations)
	} else {
		// No parent fields, just close the query
		query += "1) = 1"
	}

	query += `)`
	// Order by the original value (before casting) for proper sorting
	if fieldDef.FieldType == "int" {
		query += fmt.Sprintf(" ORDER BY %s.%s DESC", tableName, fieldDef.ValueColumnName)
	} else {
		query += fmt.Sprintf(" ORDER BY %s.%s", tableName, fieldDef.ValueColumnName)
	}

	return query, args, nil
}

// GetFieldValues returns values for a field given parent filters
func GetFieldValues(adCategoryID int, fieldName string, parentValues map[string]string) ([]string, error) {
	fieldDef, exists := GetFieldDefinition(fieldName)
	if !exists {
		return nil, fmt.Errorf("field definition not found: %s", fieldName)
	}

	query, args, err := BuildFieldQueryWithIntersection(adCategoryID, fieldDef, parentValues)

	if err != nil {
		return nil, err
	}

	var values []string
	err = db.Select(&values, query, args...)
	return values, err
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
	categoryFields, err := GetCategoryFields(adCategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get category fields: %w", err)
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
		return nil, fmt.Errorf("field %s not found in category %d", fieldName, adCategoryID)
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
		selectClause = fmt.Sprintf("SELECT DISTINCT %s.%s FROM %s", tableName, fieldDef.ValueColumnName, tableName)
	} else {
		// For string fields, select the value column
		selectClause = fmt.Sprintf("SELECT DISTINCT %s.%s FROM %s", tableName, fieldDef.ValueColumnName, tableName)
	}

	// Build JOINs for Specification table and all parent fields
	joins := []string{fmt.Sprintf("JOIN Specification spec ON %s.id = spec.%s", tableName, tableFieldName)}
	joins = append(joins, "JOIN AdSpecification ad_spec ON spec.id = ad_spec.specification_id")
	args := []interface{}{}

	// Add JOINs for parent fields
	for _, parentField := range parentFields {
		parentDef, exists := GetFieldDefinition(parentField.Name)
		if !exists {
			continue
		}
		parentTableName := parentDef.TableName
		parentTableFieldName := parentDef.TableFieldName
		joins = append(joins, fmt.Sprintf("JOIN %s ON spec.%s = %s.id", parentTableName, parentTableFieldName, parentTableName))
	}

	// Build WHERE clause with parent filters
	whereConditions := []string{}
	for _, parentField := range parentFields {
		parentDef, exists := GetFieldDefinition(parentField.Name)
		if !exists {
			continue
		}
		parentValue, exists := parentValues[parentField.Name]
		if !exists {
			continue
		}

		parentTableName := parentDef.TableName
		if parentDef.FieldType == "int" {
			// For integer fields, compare the integer value
			intValue, err := strconv.Atoi(parentValue)
			if err != nil {
				continue
			}
			whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = ?", parentTableName, parentDef.ValueColumnName))
			args = append(args, intValue)
		} else {
			// For string fields, compare the value column
			whereConditions = append(whereConditions, fmt.Sprintf("%s.%s = ?", parentTableName, parentDef.ValueColumnName))
			args = append(args, parentValue)
		}
	}

	// Add ad ID filter
	placeholders := make([]string, len(adIDs))
	for i := range adIDs {
		placeholders[i] = "?"
	}
	whereConditions = append(whereConditions, fmt.Sprintf("ad_spec.ad_id IN (%s)", strings.Join(placeholders, ",")))
	for _, id := range adIDs {
		args = append(args, id)
	}

	// Build ORDER BY clause
	var orderBy string
	if fieldDef.FieldType == "int" {
		orderBy = fmt.Sprintf("ORDER BY %s.%s DESC", tableName, fieldDef.ValueColumnName)
	} else {
		orderBy = fmt.Sprintf("ORDER BY %s.%s", tableName, fieldDef.ValueColumnName)
	}

	// Build final query
	query := fmt.Sprintf("%s %s", selectClause, strings.Join(joins, " "))
	if len(whereConditions) > 0 {
		query += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	query += " " + orderBy

	// Execute query
	var results []string
	if fieldDef.FieldType == "int" {
		var intResults []int
		err := db.Select(&intResults, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to query field values: %w", err)
		}
		// Convert integers to strings
		for _, val := range intResults {
			results = append(results, fmt.Sprintf("%d", val))
		}
	} else {
		err := db.Select(&results, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to query field values: %w", err)
		}
	}

	return results, nil
}
