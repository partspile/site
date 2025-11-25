package ad

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/parts-pile/site/db"
)

// AddAd creates a new ad in the database and returns the ad ID
func AddAd(ad *AdDetail) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var adID int
	hasVectorInt := 0
	if ad.HasVector {
		hasVectorInt = 1
	}
	// Use NULL for location_id when LocationID is 0 (empty location)
	var locationID interface{}
	if ad.LocationID == 0 {
		locationID = nil
	} else {
		locationID = ad.LocationID
	}
	err = tx.QueryRow(`
		INSERT INTO Ad (
			ad_category_id, title, description, price, 
			user_id, location_id, image_count, has_vector
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		ad.AdCategoryID, ad.Title, ad.Description, ad.Price,
		ad.UserID, locationID, ad.ImageCount, hasVectorInt).Scan(&adID)
	if err != nil {
		return fmt.Errorf("failed to insert ad: %w", err)
	}

	ad.ID = adID

	if err := saveSpecData(tx, ad); err != nil {
		return fmt.Errorf("failed to save spec data: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// saveSpecData saves spec associations dynamically based on category field configuration
func saveSpecData(tx *sql.Tx, ad *AdDetail) error {
	chains := adFieldChains[ad.AdCategoryID]
	if len(chains) == 0 {
		// No chains for this category, nothing to save
		return nil
	}

	// Process each chain
	for _, chain := range chains {
		if len(chain) == 0 {
			continue
		}

		// Get the ad spec table name from the first field
		firstField := chain[0]
		_, _, adSpecTableName, _ := firstField.GetTableNames()
		if adSpecTableName == "" {
			// Skip chains without spec tables
			continue
		}

		// Extract field values from ad.SpecData for this chain
		chainFieldValues := make(map[string][]string)
		for _, field := range chain {
			fieldName := field.Name()
			if values, exists := ad.SpecData[fieldName]; exists && len(values) > 0 {
				chainFieldValues[fieldName] = values
			}
		}

		// Skip if no values for this chain
		if len(chainFieldValues) == 0 {
			continue
		}

		// Generate all combinations of multi-value fields
		combinations := cartesianProduct(chainFieldValues)

		// Find all spec IDs for all combinations in one query
		specIDs, err := findSpecIDsForCombinations(
			tx, chain, combinations, ad.AdCategoryID)
		if err != nil {
			return fmt.Errorf("failed to find spec IDs for chain: %w", err)
		}

		// Insert all associations in one query
		if err := insertSpecAssociations(tx, ad.ID, specIDs,
			adSpecTableName); err != nil {
			return fmt.Errorf("failed to insert spec associations: %w", err)
		}
	}

	return nil
}

// findSpecIDsForCombinations finds all spec IDs for multiple combinations
// in a single SQL query using OR conditions
func findSpecIDsForCombinations(tx *sql.Tx, chain []Field,
	combinations []map[string]string, adCategoryID int) ([]int, error) {
	if len(combinations) == 0 || len(chain) == 0 {
		return nil, nil
	}

	// Get spec table name from the first field in the chain
	firstField := chain[0]
	specTableName, _, _, _ := firstField.GetTableNames()
	if specTableName == "" {
		return nil, nil
	}

	// Build field metadata - collect fields that have tables
	type fieldMeta struct {
		name               string
		tableName          string
		specTableFieldName string
		alias              string
	}
	fieldMetas := make([]fieldMeta, 0)
	for _, field := range chain {
		fieldName := field.Name()
		_, specTableFieldName, _, tableName := field.GetTableNames()
		if tableName == "" || specTableFieldName == "" {
			continue
		}
		fieldMetas = append(fieldMetas, fieldMeta{
			name:               fieldName,
			tableName:          tableName,
			specTableFieldName: specTableFieldName,
			alias:              tableName,
		})
	}

	if len(fieldMetas) == 0 {
		return nil, nil
	}

	// Build JOINs (same for all combinations)
	var joins []string
	for _, fm := range fieldMetas {
		joins = append(joins, fmt.Sprintf(
			"JOIN %s %s ON %s.%s = %s.id",
			fm.tableName, fm.alias, specTableName,
			fm.specTableFieldName, fm.alias))
	}

	// Build WHERE clause with OR conditions for each combination
	var args []interface{}
	argNum := 1

	// Add category filter
	args = append(args, adCategoryID)
	categoryParam := argNum
	argNum++

	// Build OR conditions: each combination becomes
	// (field1.value = $X AND field2.value = $Y AND ...)
	var orConditions []string
	for _, combo := range combinations {
		var andConditions []string
		for _, fm := range fieldMetas {
			value, exists := combo[fm.name]
			if !exists || value == "" {
				// Skip combinations with missing required fields
				andConditions = nil
				break
			}
			andConditions = append(andConditions,
				fmt.Sprintf("%s.value = $%d", fm.alias, argNum))
			args = append(args, value)
			argNum++
		}
		if len(andConditions) > 0 {
			orConditions = append(orConditions,
				"("+strings.Join(andConditions, " AND ")+")")
		}
	}

	if len(orConditions) == 0 {
		return nil, nil
	}

	// Build final query
	var whereClauses []string
	whereClauses = append(whereClauses,
		fmt.Sprintf("%s.ad_category_id = $%d", specTableName, categoryParam))
	whereClauses = append(whereClauses,
		"("+strings.Join(orConditions, " OR ")+")")

	query := fmt.Sprintf("SELECT DISTINCT %s.id FROM %s",
		specTableName, specTableName)
	if len(joins) > 0 {
		query += " " + strings.Join(joins, " ")
	}
	query += " WHERE " + strings.Join(whereClauses, " AND ")

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var specIDs []int
	for rows.Next() {
		var specID int
		if err := rows.Scan(&specID); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		specIDs = append(specIDs, specID)
	}

	return specIDs, rows.Err()
}

// cartesianProduct generates all combinations of multi-value fields
// (Cartesian product) - recursive
func cartesianProduct(data map[string][]string) []map[string]string {
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}

	var result []map[string]string

	var rec func(int, map[string]string)
	rec = func(i int, curr map[string]string) {
		if i == len(keys) {
			// Finished one combination → copy it and store
			copy := make(map[string]string)
			for k, v := range curr {
				copy[k] = v
			}
			result = append(result, copy)
			return
		}

		key := keys[i]
		for _, val := range data[key] {
			curr[key] = val   // set value
			rec(i+1, curr)    // go deeper
			delete(curr, key) // backtrack (clean up)
		}
	}

	if len(keys) > 0 {
		rec(0, map[string]string{})
	}
	return result
}

// insertSpecAssociations inserts multiple ad-spec association records into the
// appropriate AdSpec* table in a single SQL statement
func insertSpecAssociations(tx *sql.Tx, adID int, specIDs []int,
	adSpecTableName string) error {
	if len(specIDs) == 0 {
		return nil
	}

	// Build VALUES clause with all (ad_id, spec_id) pairs
	var valuesParts []string
	var args []interface{}
	argNum := 1

	for _, specID := range specIDs {
		valuesParts = append(valuesParts,
			fmt.Sprintf("($%d, $%d)", argNum, argNum+1))
		args = append(args, adID, specID)
		argNum += 2
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (ad_id, spec_id) VALUES %s ON CONFLICT DO NOTHING",
		adSpecTableName, strings.Join(valuesParts, ", "))
	_, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to insert %s associations: %w",
			adSpecTableName, err)
	}
	return nil
}
