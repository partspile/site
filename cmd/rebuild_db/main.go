package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/spec"
	"github.com/parts-pile/site/user"
)

func main() {
	// Initialize logging
	if err := logger.Init(config.LogLevel, config.LogFormat, config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	// Validate security configuration (initializes encryption key)
	config.SecurityCheck()

	// Get data directory from environment variable, default to cmd/rebuild_db for local development
	dataDir := os.Getenv("REBUILD_DB_DATA_DIR")
	if dataDir == "" {
		dataDir = "cmd/rebuild_db"
	}

	// Get PostgreSQL connection string from environment variable
	// Example: postgres://user:password@localhost:5432/dbname?sslmode=disable
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL must be set")
	}

	// Initialize database connection using db package
	if err := db.Init(databaseURL); err != nil {
		logger.Fatal("Failed to initialize database", "error", err, "databaseURL", databaseURL)
	}

	// Setup database (drop tables and execute schema)
	if err := setupDatabase(databaseURL); err != nil {
		logger.Fatal("Failed to setup database", "error", err)
	}

	categoryFile := dataDir + "/ad-category.json"

	// Import AdCategories
	categoryMap, adCategories, err := importAdCategories(categoryFile)
	if err != nil {
		logger.Fatal("Failed to import AdCategories", "error", err)
	}

	// Import SpecFields
	fieldIDMap, err := importSpecFields(dataDir)
	if err != nil {
		logger.Fatal("Failed to import SpecFields", "error", err)
	}

	// Import AdCategoryFields
	if err := importAdCategoryFields(categoryMap, fieldIDMap, adCategories); err != nil {
		logger.Fatal("Failed to import AdCategoryFields", "error", err)
	}

	// Initialize spec field cache (needed for GetAllChains)
	if err := spec.InitFieldCache(); err != nil {
		logger.Fatal("Failed to initialize field cache", "error", err)
	}

	// Import Users
	userIDMap, err := importUsers(dataDir)
	if err != nil {
		logger.Fatal("Failed to import Users", "error", err)
	}

	// Import specs for all categories (includes parts as part of spec chains)
	specLookupMaps, err := importAllSpecs(dataDir, categoryMap, adCategories)
	if err != nil {
		logger.Fatal("Failed to import specs", "error", err)
	}

	// Import ads for all categories
	if err := importAllAds(dataDir, categoryMap, userIDMap, adCategories, specLookupMaps); err != nil {
		logger.Fatal("Failed to import ads", "error", err)
	}

	logger.Info("Database rebuild and import complete")
	logger.Info("Vector embeddings will be processed by the main application background processor")
}

// flattenSpecJSON recursively flattens a nested JSON structure into [][]string.
// It traverses nested objects/maps and when it finds an array of strings,
// creates a row for each string with the full path of keys leading to it.
// This works generically for any nested JSON structure.
func flattenSpecJSON(data interface{}) [][]string {
	var result [][]string

	var traverse func(path []string, value interface{})
	traverse = func(path []string, value interface{}) {
		switch v := value.(type) {
		case []interface{}:
			// Array of strings - create a row for each string
			// If array is empty, create one row with just the path (for cases like 2-field specs)
			if len(v) == 0 {
				row := make([]string, len(path))
				copy(row, path)
				result = append(result, row)
			} else {
				for _, item := range v {
					str, ok := item.(string)
					if !ok {
						// Skip non-string items (shouldn't happen in spec files)
						continue
					}
					row := make([]string, len(path)+1)
					copy(row, path)
					row[len(path)] = str
					result = append(result, row)
				}
			}
		case map[string]interface{}:
			// Nested object - recurse for each key-value pair
			for key, val := range v {
				newPath := make([]string, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = key
				traverse(newPath, val)
			}
		case string:
			// Leaf string value - create a single row
			row := make([]string, len(path)+1)
			copy(row, path)
			row[len(path)] = v
			result = append(result, row)
		}
	}

	traverse(nil, data)
	return result
}

// readSpecFile reads a spec JSON file and returns it as [][]string.
func readSpecFile(filename string) ([][]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	logger.Info("Decoding JSON", "file", filename)
	var data interface{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}
	logger.Info("JSON decoded, flattening", "file", filename)

	result := flattenSpecJSON(data)
	logger.Info("Flattening complete", "file", filename, "rows", len(result))
	return result, nil
}

// setupDatabase drops all tables and executes the schema
func setupDatabase(databaseURL string) error {
	// Drop all tables to clean the database
	logger.Info("Dropping all existing tables")
	var dropSQL sql.NullString
	err := db.QueryRow(`
		SELECT COALESCE('DROP TABLE IF EXISTS ' || string_agg('"' || tablename || '"', ', ') || ' CASCADE', '')
		FROM pg_tables 
		WHERE schemaname = 'public'
	`).Scan(&dropSQL)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to generate DROP statement: %w", err)
	}

	dropSQLStr := ""
	if dropSQL.Valid {
		dropSQLStr = dropSQL.String
	}

	if dropSQLStr != "" {
		if _, err := db.Exec(dropSQLStr); err != nil {
			return fmt.Errorf("failed to drop tables: %w", err)
		}
		logger.Info("All existing tables dropped")
	} else {
		logger.Info("No tables to drop")
	}

	// Read and execute schema.sql
	schemaFile := "/workspace/schema.sql"
	if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
		schemaFile = "schema.sql"
	}

	schemaSQL, err := os.ReadFile(schemaFile)
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	// Use pgx directly to execute the entire schema file
	// pgx supports executing multiple statements in a single call
	logger.Info("Executing schema.sql using pgx")
	conn, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		return fmt.Errorf("failed to connect with pgx: %w", err)
	}
	defer conn.Close(context.Background())

	// Execute the entire schema file - pgx can handle multiple statements
	if _, err := conn.Exec(context.Background(), string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema.sql: %w", err)
	}
	logger.Info("Schema executed successfully")
	return nil
}

// AdCategoryForImport represents a category for import processing
type AdCategoryForImport struct {
	Name     string
	SpecFile string
	PartFile string
	AdFile   string
	Fields   []FieldForImport
}

type FieldForImport struct {
	FieldName          string `json:"field_name"`
	NextInChain        int    `json:"next_in_chain"`
	ShowInFilter       bool   `json:"show_in_filter"`
	IsRequired         bool   `json:"is_required"`
	SpecTableName      string `json:"spec_table_name"`
	SpecTableFieldName string `json:"spec_table_field_name"`
	AdSpecTable        string `json:"ad_spec_table"`
}

// importAdCategories imports AdCategory data from ad-category.json
func importAdCategories(categoryFile string) (map[string]int, []AdCategoryForImport, error) {
	categoryData, err := os.ReadFile(categoryFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read ad-category.json: %w", err)
	}

	type AdCategory struct {
		Name     string           `json:"name"`
		SpecFile string           `json:"spec_file,omitempty"`
		PartFile string           `json:"part_file,omitempty"`
		AdFile   string           `json:"ad_file,omitempty"`
		Fields   []FieldForImport `json:"fields,omitempty"`
	}
	var adCategories []AdCategory
	if err := json.Unmarshal(categoryData, &adCategories); err != nil {
		return nil, nil, fmt.Errorf("failed to parse ad-category.json: %w", err)
	}

	// Create map for category lookups
	categoryMap := make(map[string]int)
	for _, cat := range adCategories {
		_, err := db.Exec(`INSERT INTO AdCategory (name) VALUES ($1)`, cat.Name)
		if err != nil {
			logger.Warn("Failed to insert AdCategory", "name", cat.Name, "error", err)
		}
	}
	logger.Info("Inserted AdCategories", "count", len(adCategories))

	// Populate category map
	logger.Info("Querying AdCategory table to populate category map")
	type CategoryRow struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	var categoryRows []CategoryRow
	if err := db.Select(&categoryRows, `SELECT id, name FROM AdCategory`); err != nil {
		return nil, nil, fmt.Errorf("failed to query AdCategory: %w", err)
	}
	for _, row := range categoryRows {
		categoryMap[row.Name] = row.ID
	}
	logger.Info("Populated category map", "count", len(categoryRows))

	// Convert to the format expected by other functions
	adCategoriesForImport := make([]AdCategoryForImport, len(adCategories))
	for i, cat := range adCategories {
		adCategoriesForImport[i] = AdCategoryForImport{
			Name:     cat.Name,
			SpecFile: cat.SpecFile,
			PartFile: cat.PartFile,
			AdFile:   cat.AdFile,
			Fields:   cat.Fields,
		}
	}

	return categoryMap, adCategoriesForImport, nil
}

// importSpecFields imports SpecField data from spec-fields.json
func importSpecFields(dataBasePath string) (map[string]int, error) {
	fieldFile := dataBasePath + "/spec-fields.json"
	fieldData, err := os.ReadFile(fieldFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec-fields.json: %w", err)
	}

	type SpecFieldDef struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		FieldType   string `json:"field_type"`
		TableName   string `json:"table_name"`
		QdrantType  string `json:"qdrant_type"`
	}
	var fieldDefinitions []SpecFieldDef
	if err := json.Unmarshal(fieldData, &fieldDefinitions); err != nil {
		return nil, fmt.Errorf("failed to parse spec-fields.json: %w", err)
	}

	fieldIDMap := make(map[string]int)
	for _, fieldDef := range fieldDefinitions {
		var fieldID int
		err := db.QueryRow(`INSERT INTO SpecField (name, display_name, field_type, table_name, qdrant_type) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
			fieldDef.Name, fieldDef.DisplayName, fieldDef.FieldType, fieldDef.TableName, fieldDef.QdrantType).Scan(&fieldID)
		if err != nil {
			logger.Warn("Failed to insert SpecField", "name", fieldDef.Name, "error", err)
			continue
		}
		fieldIDMap[fieldDef.Name] = fieldID
	}
	logger.Info("Inserted SpecFields", "count", len(fieldIDMap))
	return fieldIDMap, nil
}

// importAdCategoryFields imports AdCategoryField mappings from ad-category.json
func importAdCategoryFields(categoryMap map[string]int, fieldIDMap map[string]int, adCategories []AdCategoryForImport) error {
	// Field mappings are now in ad-category.json with explicit field objects
	for _, cat := range adCategories {
		if len(cat.Fields) == 0 {
			logger.Debug("Category has no field mappings, skipping", "category", cat.Name)
			continue
		}

		categoryID, exists := categoryMap[cat.Name]
		if !exists {
			logger.Warn("Category not found for field mapping", "category", cat.Name)
			continue
		}

		// Insert fields using position in array for priority_order (0-based array index + 1 for 1-based priority_order)
		for order, field := range cat.Fields {
			fieldID, exists := fieldIDMap[field.FieldName]
			if !exists {
				logger.Warn("Field not found for category mapping", "field", field.FieldName, "category", cat.Name)
				continue
			}

			// Convert boolean to integer for database
			isRequired := 0
			if field.IsRequired {
				isRequired = 1
			}
			showInFilters := 0
			if field.ShowInFilter {
				showInFilters = 1
			}

			// Store next_in_chain exactly as read from JSON file (0-based array index)
			// No conversion - store as-is
			nextInChain := field.NextInChain

			priorityOrder := order + 1 // Convert from 0-based array index to 1-based priority_order

			_, err := db.Exec(`INSERT INTO AdCategoryField (ad_category_id, spec_field_id, spec_table_name, spec_table_field_name, ad_spec_table_name, priority_order, next_in_chain, is_required, show_in_filters) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
				categoryID, fieldID, field.SpecTableName, field.SpecTableFieldName, field.AdSpecTable, priorityOrder, nextInChain, isRequired, showInFilters)
			if err != nil {
				logger.Warn("Failed to insert AdCategoryField", "category", cat.Name, "field", field.FieldName, "error", err)
			}
		}
	}
	logger.Info("Populated AdCategoryField mappings")
	return nil
}

// importUsers imports User data from user.json
func importUsers(dataBasePath string) (map[int]int, error) {
	userFile := dataBasePath + "/user.json"
	userData, err := os.ReadFile(userFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read user.json: %w", err)
	}

	type UserImport struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
		IsAdmin  bool   `json:"is_admin"`
	}
	var users []UserImport
	if err := json.Unmarshal(userData, &users); err != nil {
		return nil, fmt.Errorf("failed to parse user.json: %w", err)
	}

	// Map from user index (1-based, as used in ad JSON files) to actual user ID
	userIDMap := make(map[int]int)
	userCount := 0
	for i, u := range users {
		hash, salt, err := password.HashPassword(u.Password)
		if err != nil {
			logger.Warn("Failed to hash password for user", "name", u.Name, "error", err)
			continue
		}

		// Compute hashes for lookups
		nameHash := user.HashForLookup(u.Name)
		phoneHash := user.HashForLookup(u.Phone)

		// Encrypt with placeholder ID (0) - we'll re-encrypt with real ID after insert
		encryptedName, nameNonce, err := user.EncryptName(0, u.Name)
		if err != nil {
			logger.Warn("Failed to encrypt name for user", "name", u.Name, "error", err)
			continue
		}
		encryptedPhone, phoneNonce, err := user.EncryptPhone(0, u.Phone)
		if err != nil {
			logger.Warn("Failed to encrypt phone for user", "name", u.Name, "error", err)
			continue
		}

		// Insert user with encrypted data
		var userID int
		isAdminInt := 0
		if u.IsAdmin {
			isAdminInt = 1
		}
		err = db.QueryRow(`INSERT INTO Users (encrypted_name, name_nonce, name_hash, encrypted_phone, phone_nonce, phone_hash,
			password_hash, password_salt, password_algo, phone_verified,
			verification_code, notification_method, is_admin)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 0, '', 'sms', $10) RETURNING id`,
			encryptedName, nameNonce, nameHash, encryptedPhone, phoneNonce, phoneHash, hash, salt, "argon2id", isAdminInt).Scan(&userID)
		if err != nil {
			logger.Warn("Failed to insert user", "name", u.Name, "error", err)
			continue
		}

		// Re-encrypt with the real user ID
		encryptedName, nameNonce, err = user.EncryptName(userID, u.Name)
		if err != nil {
			logger.Warn("Failed to re-encrypt name for user", "name", u.Name, "error", err)
			continue
		}
		encryptedPhone, phoneNonce, err = user.EncryptPhone(userID, u.Phone)
		if err != nil {
			logger.Warn("Failed to re-encrypt phone for user", "name", u.Name, "error", err)
			continue
		}

		// Update with properly encrypted data
		_, err = db.Exec(`UPDATE Users SET encrypted_name = $1, name_nonce = $2, encrypted_phone = $3, phone_nonce = $4 WHERE id = $5`,
			encryptedName, nameNonce, encryptedPhone, phoneNonce, userID)
		if err != nil {
			logger.Warn("Failed to update encrypted data for user", "name", u.Name, "error", err)
			continue
		}

		// Initialize rock inventory
		_, err = db.Exec(`INSERT INTO UserRock (user_id, rock_count) VALUES ($1, 3)`, userID)
		if err != nil {
			logger.Warn("Failed to initialize rock inventory for user", "name", u.Name, "error", err)
			continue
		}

		// Map user index (1-based) to actual user ID for ad imports
		userIDMap[i+1] = userID
		userCount++
	}
	logger.Info("Inserted Users", "count", userCount)
	return userIDMap, nil
}

// buildCategorySeedMap creates a map of category names to their seed file paths
func buildCategorySeedMap(dataBasePath string, adCategories []AdCategoryForImport) map[string]struct {
	specFile string
	partFile string
	adFile   string
} {
	categorySeedMap := make(map[string]struct {
		specFile string
		partFile string
		adFile   string
	})

	// Process each category from ad-category.json and add to seed map if it has files
	for _, cat := range adCategories {
		if cat.SpecFile != "" || cat.AdFile != "" {
			seedFiles := struct {
				specFile string
				partFile string
				adFile   string
			}{
				specFile: dataBasePath + "/" + cat.SpecFile,
				partFile: "",
				adFile:   dataBasePath + "/" + cat.AdFile,
			}
			if cat.PartFile != "" {
				seedFiles.partFile = dataBasePath + "/" + cat.PartFile
			}
			categorySeedMap[cat.Name] = seedFiles
		}
	}

	return categorySeedMap
}

// importAllSpecs imports specs for all categories and returns a map of category name -> spec lookup maps
func importAllSpecs(dataBasePath string, categoryMap map[string]int, adCategories []AdCategoryForImport) (map[string]SpecLookupMaps, error) {
	categorySeedMap := buildCategorySeedMap(dataBasePath, adCategories)
	specLookupMaps := make(map[string]SpecLookupMaps)

	// Process each category that has spec files
	for categoryName, seedFiles := range categorySeedMap {
		if seedFiles.specFile == "" {
			continue
		}

		categoryID, exists := categoryMap[categoryName]
		if !exists {
			logger.Warn("Category not found, skipping", "category", categoryName)
			continue
		}

		logger.Info("Importing specs for category", "category", categoryName)

		// Import specs for this category and get lookup maps
		specLookupMapsForCategory, err := importSpecs(seedFiles.specFile, seedFiles.partFile, categoryID)
		if err != nil {
			logger.Warn("Failed to import specs for category", "category", categoryName, "error", err)
		}

		specLookupMaps[categoryName] = specLookupMapsForCategory
	}

	return specLookupMaps, nil
}

// importAllAds imports ads for all categories
func importAllAds(dataBasePath string, categoryMap map[string]int, userIDMap map[int]int, adCategories []AdCategoryForImport, specLookupMaps map[string]SpecLookupMaps) error {
	categorySeedMap := buildCategorySeedMap(dataBasePath, adCategories)

	// Process each category that has ad files
	for categoryName, seedFiles := range categorySeedMap {
		if seedFiles.adFile == "" {
			continue
		}

		categoryID, exists := categoryMap[categoryName]
		if !exists {
			logger.Warn("Category not found, skipping", "category", categoryName)
			continue
		}

		logger.Info("Importing ads for category", "category", categoryName)

		// Get spec lookup maps for this category
		specLookupMapsForCategory, exists := specLookupMaps[categoryName]
		if !exists {
			specLookupMapsForCategory = SpecLookupMaps{
				Vehicle:      make(map[string]int),
				Agricultural: make(map[string]int),
				Bicycle:      make(map[string]int),
				Part:         make(map[string]int),
			}
		}

		if err := importAds(seedFiles.adFile, categoryID, categoryMap, userIDMap, specLookupMapsForCategory); err != nil {
			logger.Warn("Failed to import ads for category", "category", categoryName, "error", err)
		}
	}

	return nil
}

// FieldConfig is an alias for spec.FieldConfig for convenience in rebuild_db
type FieldConfig = spec.FieldConfig

// buildFieldValueMaps creates []map[string]int from [][]string
// Each map corresponds to a field position, with value -> 0 (will be updated with IDs later)
func buildFieldValueMaps(specs [][]string, numFields int) []map[string]int {
	fieldMaps := make([]map[string]int, numFields)
	for i := range fieldMaps {
		fieldMaps[i] = make(map[string]int)
	}

	for _, row := range specs {
		if len(row) < numFields {
			continue
		}
		for i := 0; i < numFields; i++ {
			fieldMaps[i][row[i]] = 0 // Initialize to 0, will be updated with IDs
		}
	}

	return fieldMaps
}

// buildFieldTables builds field tables dynamically and updates fieldMaps with IDs
func buildFieldTables(categoryID int, fields []FieldConfig, fieldMaps []map[string]int) error {
	for i, field := range fields {
		if i >= len(fieldMaps) {
			continue
		}
		fieldMap := fieldMaps[i]

		// Load existing values
		valueList := make([]interface{}, 0, len(fieldMap))
		for val := range fieldMap {
			valueList = append(valueList, val)
		}
		if len(valueList) == 0 {
			continue
		}

		// Build query to load existing IDs
		placeholders := make([]string, len(valueList))
		for j := range placeholders {
			placeholders[j] = fmt.Sprintf("$%d", j+2)
		}

		var query string
		var args []interface{}

		// All fields are now stored as TEXT, so treat them all as strings
		query = fmt.Sprintf(`SELECT id, value FROM %s WHERE value IN (%s) AND ad_category_id=$1`,
			field.TableName, strings.Join(placeholders, ","))
		args = append([]interface{}{categoryID}, valueList...)
		rows, err := db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int
				var value string
				if rows.Scan(&id, &value) == nil {
					fieldMap[value] = id
				}
			}
		}

		// Insert missing values
		missingValues := make([]string, 0)
		for val, id := range fieldMap {
			if id == 0 {
				missingValues = append(missingValues, val)
			}
		}

		if len(missingValues) == 0 {
			continue
		}

		// Batch insert missing values
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		// All fields are now stored as TEXT, so treat them all as strings
		values := make([]string, len(missingValues))
		insertArgs := make([]interface{}, len(missingValues)*2)
		for j, val := range missingValues {
			argStart := j*2 + 1
			values[j] = fmt.Sprintf("($%d, $%d)", argStart, argStart+1)
			insertArgs[j*2] = val
			insertArgs[j*2+1] = categoryID
		}
		insertQuery := fmt.Sprintf(`INSERT INTO %s (value, ad_category_id) VALUES %s RETURNING id, value`,
			field.TableName, strings.Join(values, ","))
		insertRows, err := tx.Query(insertQuery, insertArgs...)
		if err == nil {
			for insertRows.Next() {
				var id int
				var value string
				if insertRows.Scan(&id, &value) == nil {
					fieldMap[value] = id
				}
			}
			insertRows.Close()
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	return nil
}

// SpecLookupMaps contains lookup maps for each spec type
type SpecLookupMaps struct {
	Vehicle      map[string]int
	Agricultural map[string]int
	Bicycle      map[string]int
	Part         map[string]int
}

// buildSpecsForChain builds specs for a single chain and inserts them into the appropriate spec table
func buildSpecsForChain(categoryID int, chain []FieldConfig, data [][]string, fieldMaps []map[string]int, specType string) (map[string]int, error) {
	specLookupMap := make(map[string]int)
	tableName := spec.GetSpecTableName(specType)

	// Build field names for RETURNING clause
	fieldNames := make([]string, len(chain))
	for i, field := range chain {
		fieldNames[i] = field.TableFieldName
	}

	type specEntry struct {
		adCategoryID int
		fieldIDs     []int
		fieldValues  []string
	}
	var allSpecs []specEntry

	// Build spec entries
	for _, row := range data {
		if len(row) < len(chain) {
			continue
		}

		fieldIDs := make([]int, len(chain))
		valid := true
		for i := range chain {
			value := row[i]
			id, exists := fieldMaps[i][value]
			if !exists || id == 0 {
				logger.Warn("Field value not found in map", "field", chain[i].Name, "value", value)
				valid = false
				break
			}
			fieldIDs[i] = id
		}

		if valid {
			allSpecs = append(allSpecs, specEntry{
				adCategoryID: categoryID,
				fieldIDs:     fieldIDs,
				fieldValues:  row,
			})
		}
	}

	if len(allSpecs) == 0 {
		return specLookupMap, nil
	}

	// Batch insert specs
	const batchSize = 1000
	tx, err := db.Begin()
	if err != nil {
		return specLookupMap, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i := 0; i < len(allSpecs); i += batchSize {
		end := i + batchSize
		if end > len(allSpecs) {
			end = len(allSpecs)
		}
		batch := allSpecs[i:end]

		values := make([]string, len(batch))
		args := make([]interface{}, len(batch)*(len(chain)+1))
		argIdx := 0

		for j, spec := range batch {
			valueParts := make([]string, len(chain)+1)
			valueParts[0] = fmt.Sprintf("$%d", argIdx+1)
			args[argIdx] = spec.adCategoryID
			argIdx++

			for k, fieldID := range spec.fieldIDs {
				valueParts[k+1] = fmt.Sprintf("$%d", argIdx+1)
				args[argIdx] = fieldID
				argIdx++
			}
			values[j] = "(" + strings.Join(valueParts, ", ") + ")"
		}

		columns := []string{"ad_category_id"}
		for _, field := range chain {
			columns = append(columns, field.TableFieldName)
		}

		query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES %s ON CONFLICT DO NOTHING RETURNING id, %s`,
			tableName, strings.Join(columns, ", "), strings.Join(values, ", "), strings.Join(fieldNames, ", "))
		rows, err := tx.Query(query, args...)
		if err != nil {
			logger.Warn("Failed to insert spec batch", "table", tableName, "error", err)
			continue
		}

		rowIdx := 0
		for rows.Next() {
			if rowIdx >= len(batch) {
				break
			}
			spec := batch[rowIdx]
			var specID int
			scanArgs := make([]interface{}, len(chain)+1)
			scanArgs[0] = &specID
			for j := range chain {
				var fieldID int
				scanArgs[j+1] = &fieldID
			}
			if rows.Scan(scanArgs...) == nil {
				key := strings.Join(spec.fieldValues, ":")
				specLookupMap[key] = specID
			}
			rowIdx++
		}
		rows.Close()
	}

	if err := tx.Commit(); err != nil {
		return specLookupMap, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return specLookupMap, nil
}

// importSpecs loads spec data from JSON files and imports it into the database
// Handles multiple chains: first chain from specFile, part chain from partFile (if provided)
// Creates specs in separate tables per chain type to avoid cartesian product explosion
// Returns lookup maps per spec type: spec_type -> field value combination (":" separated) -> spec ID
func importSpecs(specFile string, partFile string, categoryID int) (SpecLookupMaps, error) {
	result := SpecLookupMaps{
		Vehicle:      make(map[string]int),
		Agricultural: make(map[string]int),
		Bicycle:      make(map[string]int),
		Part:         make(map[string]int),
	}
	// Get all chains for this category
	allChains, err := spec.GetAllChains(categoryID)
	if err != nil {
		return result, fmt.Errorf("failed to get all chains: %w", err)
	}

	if len(allChains) == 0 {
		return result, fmt.Errorf("no chains found for category %d", categoryID)
	}

	// Find the first chain (lowest order) that has database tables and part_category chain
	// Filter out chains that don't have any fields with TableName (like location, price)
	var firstChain []FieldConfig
	var partChain []FieldConfig
	minOrder := -1
	for order, chain := range allChains {
		// Only consider chains that have at least one field with a TableName
		hasTableFields := false
		for _, field := range chain {
			if field.TableName != "" {
				hasTableFields = true
				break
			}
		}
		if !hasTableFields {
			continue // Skip chains without database table fields
		}

		if minOrder == -1 || order < minOrder {
			minOrder = order
			firstChain = chain
		}
		if len(chain) > 0 && chain[0].Name == "part_category" {
			partChain = chain
		}
	}

	if len(firstChain) == 0 {
		return result, fmt.Errorf("no first chain found for category %d", categoryID)
	}

	// Filter firstChain to only include fields with TableName (for spec import)
	// This ensures we only validate against fields that are in the spec files
	filteredFirstChain := make([]spec.FieldConfig, 0, len(firstChain))
	for _, field := range firstChain {
		if field.TableName != "" {
			filteredFirstChain = append(filteredFirstChain, field)
		}
	}
	firstChain = filteredFirstChain

	logger.Info("Found chains for category", "categoryID", categoryID, "firstChain", len(firstChain), "partChain", len(partChain), "partFile", partFile)

	// Step 1: Read and import part file FIRST (before spec validation)
	// This ensures part categories are imported even if spec import fails
	var parts [][]string
	var partChainFieldMaps []map[string]int
	if len(partChain) > 0 && partFile != "" {
		parts, err = readPartFile(partFile)
		if err != nil {
			logger.Warn("Failed to read part file, skipping", "error", err, "file", partFile)
			parts = [][]string{} // Empty, will create specs without part fields
		} else {
			logger.Info("Read part file", "file", partFile, "rows", len(parts))

			// Validate part file structure
			expectedPartLen := len(partChain)
			if len(parts) > 0 {
				actualPartLen := len(parts[0])
				if actualPartLen != expectedPartLen {
					logger.Warn("Part file structure mismatch, skipping part import", "expected", expectedPartLen, "actual", actualPartLen, "file", partFile)
					parts = [][]string{}
				} else {
					// Import part categories even if spec import fails
					partChainFieldMaps = buildFieldValueMaps(parts, len(partChain))
					if err := buildFieldTables(categoryID, partChain, partChainFieldMaps); err != nil {
						logger.Warn("Failed to build part chain field tables", "error", err)
						partChainFieldMaps = nil // Clear maps if import failed
					} else {
						logger.Info("Imported part categories", "categoryID", categoryID, "partCategories", len(partChainFieldMaps[0]), "partSubcategories", len(partChainFieldMaps[1]))
					}
				}
			}
		}
	}

	// Step 2: Read spec file for first chain
	logger.Info("Reading spec file", "file", specFile)
	specs, err := readSpecFile(specFile)
	if err != nil {
		logger.Debug("Failed to read spec file, skipping", "error", err, "file", specFile)
		return result, nil // Skip missing files
	}
	logger.Info("Read spec file", "file", specFile, "rows", len(specs))

	if len(specs) == 0 {
		logger.Debug("Spec file is empty, skipping", "file", specFile)
		return result, nil
	}

	// Validate spec file structure
	expectedLen := len(firstChain)
	if len(specs) > 0 {
		actualLen := len(specs[0])
		if actualLen != expectedLen {
			return result, fmt.Errorf("spec file structure mismatch: file has %d fields but category expects %d fields. Spec file: %s, Category ID: %d. Please fix the spec file to match the category's field configuration", actualLen, expectedLen, specFile, categoryID)
		}
	}

	// Validate all rows
	var invalidRows []string
	for i, row := range specs {
		if len(row) != expectedLen {
			rowStr := strings.Join(row, " -> ")
			if len(rowStr) > 100 {
				rowStr = rowStr[:100] + "..."
			}
			invalidRows = append(invalidRows, fmt.Sprintf("  Row %d: has %d fields (expected %d) - path: %s", i, len(row), expectedLen, rowStr))
		}
	}

	if len(invalidRows) > 0 {
		maxShow := 10
		if len(invalidRows) < maxShow {
			maxShow = len(invalidRows)
		}
		errorMsg := fmt.Sprintf("spec file has %d rows with incorrect field counts (expected %d fields). File: %s. Invalid rows:\n%s", len(invalidRows), expectedLen, specFile, strings.Join(invalidRows[:maxShow], "\n"))
		if len(invalidRows) > maxShow {
			errorMsg += fmt.Sprintf("\n... and %d more invalid rows", len(invalidRows)-maxShow)
		}
		errorMsg += "\nPlease fix the spec file to have consistent field counts matching the category configuration."
		return result, fmt.Errorf("%s", errorMsg)
	}

	// Step 3: Process first chain separately (no cartesian product)
	firstChainFieldMaps := buildFieldValueMaps(specs, len(firstChain))
	if err := buildFieldTables(categoryID, firstChain, firstChainFieldMaps); err != nil {
		return result, fmt.Errorf("failed to build first chain field tables: %w", err)
	}

	firstChainSpecType := spec.GetSpecTypeFromChain(firstChain)
	firstChainLookup, err := buildSpecsForChain(categoryID, firstChain, specs, firstChainFieldMaps, firstChainSpecType)
	if err != nil {
		return result, fmt.Errorf("failed to build specs for first chain: %w", err)
	}

	// Store lookup map in appropriate result field
	switch firstChainSpecType {
	case "vehicle":
		result.Vehicle = firstChainLookup
	case "agricultural":
		result.Agricultural = firstChainLookup
	case "bicycle":
		result.Bicycle = firstChainLookup
	}

	logger.Info("Imported first chain specs", "specType", firstChainSpecType, "count", len(firstChainLookup))

	// Step 4: Process part chain separately (if exists)
	if len(partChain) > 0 && len(parts) > 0 {
		// Reuse part chain field maps from Step 1 (already populated with IDs)
		if partChainFieldMaps == nil {
			partChainFieldMaps = buildFieldValueMaps(parts, len(partChain))
			if err := buildFieldTables(categoryID, partChain, partChainFieldMaps); err != nil {
				logger.Warn("Failed to rebuild part chain field tables", "error", err)
			}
		}

		if partChainFieldMaps != nil {
			partChainLookup, err := buildSpecsForChain(categoryID, partChain, parts, partChainFieldMaps, "part")
			if err != nil {
				logger.Warn("Failed to build specs for part chain", "error", err)
			} else {
				result.Part = partChainLookup
				logger.Info("Imported part chain specs", "count", len(partChainLookup))
			}
		}
	}

	logger.Info("Imported specs", "specFile", specFile, "partFile", partFile,
		"vehicle", len(result.Vehicle), "agricultural", len(result.Agricultural),
		"bicycle", len(result.Bicycle), "part", len(result.Part))
	return result, nil
}

// convertAdToSpecRows converts ad JSON structure to [][]string format
// Handles arrays by creating a row for each combination (Cartesian product)
// Example: {"make": "HONDA", "years": ["2020", "2021"], "models": ["CIVIC"]}
// becomes: [["HONDA", "2020", "CIVIC"], ["HONDA", "2021", "CIVIC"]]
func convertAdToSpecRows(ad map[string]interface{}, fields []FieldConfig) [][]string {
	// Extract field values from ad, handling both single values and arrays
	fieldValues := make([][]string, len(fields))
	for i, field := range fields {
		fieldName := field.Name
		value, exists := ad[fieldName]
		// Try plural form if singular not found (e.g., "year" -> "years")
		if !exists {
			pluralName := fieldName + "s"
			if pluralValue, pluralExists := ad[pluralName]; pluralExists {
				value = pluralValue
				exists = true
			}
		}
		if !exists {
			// Field not present in ad, use empty array
			fieldValues[i] = []string{""}
			continue
		}

		// Handle both single values and arrays
		switch v := value.(type) {
		case []interface{}:
			// Array of values
			values := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					values = append(values, str)
				}
			}
			if len(values) == 0 {
				fieldValues[i] = []string{""}
			} else {
				fieldValues[i] = values
			}
		case string:
			// Single value
			fieldValues[i] = []string{v}
		default:
			// Unknown type, use empty
			fieldValues[i] = []string{""}
		}
	}

	// Generate Cartesian product of all field values
	var result [][]string
	var generate func(int, []string)
	generate = func(fieldIdx int, current []string) {
		if fieldIdx >= len(fields) {
			// Base case: we have a complete combination
			row := make([]string, len(current))
			copy(row, current)
			result = append(result, row)
			return
		}

		// Recursive case: try each value for this field
		for _, value := range fieldValues[fieldIdx] {
			generate(fieldIdx+1, append(current, value))
		}
	}

	generate(0, []string{})
	return result
}

// SpecIDWithType represents a spec ID with its type
type SpecIDWithType struct {
	ID       int
	SpecType string
}

// findSpecIDs finds specification IDs matching the given field value combinations for a specific chain
// Uses the lookup map built during spec import - no SQL queries needed!
// adSpecRows: [][]string where each row is a field value combination for this chain
// specLookupMap: map[string]int where key is "value1:value2:value3" and value is spec ID
// specType: the type of spec table ("vehicle", "agricultural", "bicycle", "part")
// Returns: []SpecIDWithType of matching spec IDs with their types
func findSpecIDs(adSpecRows [][]string, specLookupMap map[string]int, specType string) []SpecIDWithType {
	if len(adSpecRows) == 0 || len(specLookupMap) == 0 {
		return []SpecIDWithType{}
	}

	specIDs := make(map[int]bool) // Use map to deduplicate

	// For each row, create lookup key and find spec ID
	for _, row := range adSpecRows {
		// Create lookup key: "value1:value2:value3"
		key := strings.Join(row, ":")
		if specID, exists := specLookupMap[key]; exists && specID > 0 {
			specIDs[specID] = true
		} else if len(adSpecRows) <= 5 {
			// Only log for first few rows to avoid spam
			logger.Debug("Spec lookup key not found", "key", key, "specType", specType, "lookupMapSize", len(specLookupMap))
		}
	}

	// Convert map to slice with spec types
	result := make([]SpecIDWithType, 0, len(specIDs))
	for id := range specIDs {
		result = append(result, SpecIDWithType{ID: id, SpecType: specType})
	}

	return result
}

// readPartFile reads a part JSON file and flattens it to [][]string format
// Part files have structure: {"category": ["subcat1", "subcat2", ...]}
// Returns: [["category", "subcat1"], ["category", "subcat2"], ...]
func readPartFile(jsonFile string) ([][]string, error) {
	partData, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read part file: %w", err)
	}

	var partMap map[string][]string
	if err := json.Unmarshal(partData, &partMap); err != nil {
		return nil, fmt.Errorf("failed to parse part file: %w", err)
	}

	var result [][]string
	for category, subcategories := range partMap {
		for _, subcategory := range subcategories {
			result = append(result, []string{category, subcategory})
		}
	}

	return result, nil
}

// importAds loads ad data from a JSON file and imports it into the database
// specLookupMap: map[string]int where key is "value1:value2:value3" and value is spec ID
// This function is metadata-driven and works generically for any field configuration
func importAds(jsonFile string, categoryID int, categoryMap map[string]int, userIDMap map[int]int, specLookupMaps SpecLookupMaps) error {
	adData, err := os.ReadFile(jsonFile)
	if err != nil {
		logger.Debug("Failed to read ad file, skipping", "error", err, "file", jsonFile)
		return nil // Skip missing files
	}

	// Parse ads as generic map to support any field configuration
	var ads []map[string]interface{}
	if err := json.Unmarshal(adData, &ads); err != nil {
		logger.Warn("Failed to parse ad file", "error", err, "file", jsonFile)
		return err
	}

	if len(ads) == 0 {
		logger.Info("No ads to import", "file", jsonFile)
		return nil
	}

	// Get all field chains for this category
	allChains, err := spec.GetAllChains(categoryID)
	if err != nil {
		return fmt.Errorf("failed to get all chains: %w", err)
	}

	if len(allChains) == 0 {
		return fmt.Errorf("no chains found for category %d", categoryID)
	}

	// Find the first chain (lowest order) that has database tables and part_category chain
	// Filter out chains that don't have any fields with TableName (like location, price)
	var firstChain []FieldConfig
	var partChain []FieldConfig
	minOrder := -1
	for order, chain := range allChains {
		// Only consider chains that have at least one field with a TableName
		hasTableFields := false
		for _, field := range chain {
			if field.TableName != "" {
				hasTableFields = true
				break
			}
		}
		if !hasTableFields {
			continue // Skip chains without database table fields
		}

		if minOrder == -1 || order < minOrder {
			minOrder = order
			firstChain = chain
		}
		if len(chain) > 0 && chain[0].Name == "part_category" {
			partChain = chain
		}
	}

	if len(firstChain) == 0 {
		return fmt.Errorf("no first chain found for category %d", categoryID)
	}

	// Filter firstChain to only include fields with TableName (for ad import)
	// This ensures we only match against fields that are in the spec lookup maps
	filteredFirstChain := make([]FieldConfig, 0, len(firstChain))
	for _, field := range firstChain {
		if field.TableName != "" {
			filteredFirstChain = append(filteredFirstChain, field)
		}
	}
	firstChain = filteredFirstChain

	// Get part categories and subcategories
	partCategoryMap := make(map[string]int)
	subcategoryMap := make(map[string]int)

	rows, err := db.Query(`SELECT id, value FROM PartCategory WHERE ad_category_id=$1`, categoryID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var value string
			if err := rows.Scan(&id, &value); err == nil {
				partCategoryMap[value] = id
			}
		}
	}
	if err != nil {
		logger.Warn("Failed to query PartCategory", "error", err, "categoryID", categoryID)
	}

	rows2, err := db.Query(`SELECT id, value FROM PartSubCategory WHERE ad_category_id=$1`, categoryID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id int
			var value string
			if err := rows2.Scan(&id, &value); err == nil {
				subcategoryMap[value] = id
			}
		}
	}
	if err != nil {
		logger.Warn("Failed to query PartSubCategory", "error", err, "categoryID", categoryID)
	}

	logger.Info("Loaded part categories for ads", "categoryID", categoryID, "partCategories", len(partCategoryMap), "partSubcategories", len(subcategoryMap))

	// Collect unique locations and batch insert
	type locationData struct {
		rawText   string
		city      string
		adminArea string
		country   string
		latitude  *float64
		longitude *float64
	}
	locationSet := make(map[string]locationData)
	locationIDMap := make(map[string]int)

	for _, ad := range ads {
		locationMap, _ := ad["location"].(map[string]interface{})
		city, _ := locationMap["city"].(string)
		adminArea, _ := locationMap["admin_area"].(string)
		country, _ := locationMap["country"].(string)
		locationKey := fmt.Sprintf("%s, %s, %s", city, adminArea, country)
		if _, exists := locationSet[locationKey]; !exists {
			var latitude *float64
			var longitude *float64
			if lat, ok := locationMap["latitude"].(float64); ok {
				latitude = &lat
			}
			if lon, ok := locationMap["longitude"].(float64); ok {
				longitude = &lon
			}
			locationSet[locationKey] = locationData{
				rawText:   locationKey,
				city:      city,
				adminArea: adminArea,
				country:   country,
				latitude:  latitude,
				longitude: longitude,
			}
		}
	}

	// Load existing locations
	if len(locationSet) > 0 {
		locationKeys := make([]interface{}, 0, len(locationSet))
		for key := range locationSet {
			locationKeys = append(locationKeys, key)
		}
		placeholders := make([]string, len(locationKeys))
		for i := range placeholders {
			placeholders[i] = fmt.Sprintf("$%d", i+1)
		}
		query := fmt.Sprintf(`SELECT id, raw_text FROM Location WHERE raw_text IN (%s)`, strings.Join(placeholders, ","))
		rows, err := db.Query(query, locationKeys...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int
				var rawText string
				if rows.Scan(&id, &rawText) == nil {
					locationIDMap[rawText] = id
				}
			}
		}
	}

	// Batch insert missing locations
	missingLocations := make([]locationData, 0)
	for key, loc := range locationSet {
		if _, exists := locationIDMap[key]; !exists {
			missingLocations = append(missingLocations, loc)
		}
	}
	if len(missingLocations) > 0 {
		tx, err := db.Begin()
		if err == nil {
			values := make([]string, len(missingLocations))
			args := make([]interface{}, len(missingLocations)*6)
			for i, loc := range missingLocations {
				argStart := i*6 + 1
				values[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)", argStart, argStart+1, argStart+2, argStart+3, argStart+4, argStart+5)
				args[i*6] = loc.rawText
				args[i*6+1] = loc.city
				args[i*6+2] = loc.adminArea
				args[i*6+3] = loc.country
				args[i*6+4] = loc.latitude
				args[i*6+5] = loc.longitude
			}
			query := fmt.Sprintf(`INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES %s ON CONFLICT DO NOTHING RETURNING id, raw_text`, strings.Join(values, ","))
			rows, err := tx.Query(query, args...)
			if err == nil {
				for rows.Next() {
					var id int
					var rawText string
					if rows.Scan(&id, &rawText) == nil {
						locationIDMap[rawText] = id
					}
				}
				rows.Close()
			}
			tx.Commit()
		}
	}

	// Process ads and collect data for batch inserts
	type adBatchEntry struct {
		title          string
		description    string
		price          float64
		createdAt      string
		userID         int
		locationID     int
		imageCount     int
		categoryID     int
		partCategoryID *int
		subcategoryID  *int
		specIDs        []SpecIDWithType // spec IDs with types to link to this ad
	}

	adBatch := make([]adBatchEntry, 0, len(ads))

	for _, ad := range ads {
		// Extract common ad fields
		title, _ := ad["title"].(string)
		description, _ := ad["description"].(string)
		price, _ := ad["price"].(float64)
		createdAt, _ := ad["created_at"].(string)
		userIDFloat, _ := ad["user_id"].(float64)
		userID := int(userIDFloat)

		// Validate user ID
		actualUserID, exists := userIDMap[userID]
		if !exists {
			logger.Warn("User ID not found in userIDMap, skipping ad", "userID", userID, "title", title)
			continue
		}

		// Get location
		locationMap, _ := ad["location"].(map[string]interface{})
		city, _ := locationMap["city"].(string)
		adminArea, _ := locationMap["admin_area"].(string)
		country, _ := locationMap["country"].(string)
		locationKey := fmt.Sprintf("%s, %s, %s", city, adminArea, country)
		locationID, exists := locationIDMap[locationKey]
		if !exists {
			logger.Warn("Location not found after batch insert, skipping ad", "title", title)
			continue
		}

		// Get part_category_id and part_subcategory_id
		var partCategoryID *int
		var subcategoryID *int
		category, _ := ad["category"].(string)
		subcategory, _ := ad["subcategory"].(string)
		if category != "" {
			partCatID, exists := partCategoryMap[category]
			if !exists {
				logger.Warn("No valid part category found for ad, skipping", "title", title, "category", category)
				continue
			}
			partCategoryID = &partCatID

			subcatID := 0
			if subcategory != "" {
				if id, exists := subcategoryMap[subcategory]; exists {
					subcatID = id
				}
			}

			// If no subcategory found, use the first available one for this category
			if subcatID == 0 && len(subcategoryMap) > 0 {
				for _, id := range subcategoryMap {
					subcatID = id
					break
				}
			}

			if subcatID == 0 {
				logger.Warn("No valid subcategory found for ad, skipping", "title", title)
				continue
			}
			subcategoryID = &subcatID
		}

		// Find specs for each chain separately (no cartesian product)
		var allSpecIDs []SpecIDWithType

		// Process first chain
		firstChainSpecRows := convertAdToSpecRows(ad, firstChain)
		firstChainSpecType := spec.GetSpecTypeFromChain(firstChain)
		var firstChainLookup map[string]int
		switch firstChainSpecType {
		case "vehicle":
			firstChainLookup = specLookupMaps.Vehicle
		case "agricultural":
			firstChainLookup = specLookupMaps.Agricultural
		case "bicycle":
			firstChainLookup = specLookupMaps.Bicycle
		}
		firstChainSpecIDs := findSpecIDs(firstChainSpecRows, firstChainLookup, firstChainSpecType)
		if len(firstChainSpecIDs) == 0 {
			// Log sample lookup keys for debugging
			sampleKeys := make([]string, 0, 3)
			for i, row := range firstChainSpecRows {
				if i >= 3 {
					break
				}
				sampleKeys = append(sampleKeys, strings.Join(row, ":"))
			}
			logger.Warn("No matching specs found for ad first chain, skipping", "title", title, "sampleKeys", sampleKeys, "lookupMapSize", len(firstChainLookup))
			continue
		}
		allSpecIDs = append(allSpecIDs, firstChainSpecIDs...)

		// Process part chain (if exists)
		if len(partChain) > 0 && partCategoryID != nil && subcategoryID != nil {
			// Build part chain rows
			partCategoryName := ""
			partSubcategoryName := ""
			for name, id := range partCategoryMap {
				if id == *partCategoryID {
					partCategoryName = name
					break
				}
			}
			for name, id := range subcategoryMap {
				if id == *subcategoryID {
					partSubcategoryName = name
					break
				}
			}
			if partCategoryName != "" && partSubcategoryName != "" {
				partRows := [][]string{{partCategoryName, partSubcategoryName}}
				partSpecIDs := findSpecIDs(partRows, specLookupMaps.Part, "part")
				if len(partSpecIDs) == 0 {
					logger.Warn("No matching specs found for ad part chain, skipping", "title", title)
					continue
				}
				allSpecIDs = append(allSpecIDs, partSpecIDs...)
			}
		}

		// Generate between 0 and 5 images per ad
		numImages := rand.Intn(6)

		adBatch = append(adBatch, adBatchEntry{
			title:          title,
			description:    description,
			price:          price,
			createdAt:      createdAt,
			userID:         actualUserID,
			locationID:     locationID,
			imageCount:     numImages,
			categoryID:     categoryID,
			partCategoryID: partCategoryID,
			subcategoryID:  subcategoryID,
			specIDs:        allSpecIDs,
		})
	}

	// Batch insert ads
	const adBatchSize = 1000
	adIDMap := make(map[int]int) // index in adBatch -> ad ID

	for i := 0; i < len(adBatch); i += adBatchSize {
		end := i + adBatchSize
		if end > len(adBatch) {
			end = len(adBatch)
		}
		batch := adBatch[i:end]

		tx, err := db.Begin()
		if err != nil {
			logger.Warn("Failed to begin transaction for ad batch", "error", err)
			continue
		}

		values := make([]string, len(batch))
		args := make([]interface{}, len(batch)*9)
		for j, ad := range batch {
			argStart := j*9 + 1
			values[j] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)", argStart, argStart+1, argStart+2, argStart+3, argStart+4, argStart+5, argStart+6, argStart+7, argStart+8)
			args[j*9] = ad.title
			args[j*9+1] = ad.description
			args[j*9+2] = ad.price
			args[j*9+3] = ad.createdAt
			args[j*9+4] = ad.userID
			args[j*9+5] = ad.locationID
			args[j*9+6] = ad.imageCount
			args[j*9+7] = ad.categoryID
			args[j*9+8] = 0 // has_vector
		}

		query := fmt.Sprintf(`INSERT INTO Ad (title, description, price, created_at, user_id, location_id, image_count, ad_category_id, has_vector) VALUES %s RETURNING id`, strings.Join(values, ","))
		rows, err := tx.Query(query, args...)
		if err != nil {
			tx.Rollback()
			logger.Warn("Failed to insert ad batch", "error", err)
			continue
		}

		rowIdx := 0
		for rows.Next() {
			var adID int
			if rows.Scan(&adID) == nil {
				adIDMap[i+rowIdx] = adID
			}
			rowIdx++
		}
		rows.Close()

		if err := tx.Commit(); err != nil {
			logger.Warn("Failed to commit ad batch transaction", "error", err)
			continue
		}
	}

	// Batch insert AdSpec* relationships - group by spec type
	adSpecByType := make(map[string][]struct {
		adID   int
		specID int
	})

	for idx, ad := range adBatch {
		adID, exists := adIDMap[idx]
		if !exists {
			continue
		}

		for _, specIDWithType := range ad.specIDs {
			if adSpecByType[specIDWithType.SpecType] == nil {
				adSpecByType[specIDWithType.SpecType] = make([]struct {
					adID   int
					specID int
				}, 0)
			}
			adSpecByType[specIDWithType.SpecType] = append(adSpecByType[specIDWithType.SpecType], struct {
				adID   int
				specID int
			}{adID: adID, specID: specIDWithType.ID})
		}
	}

	// Insert into appropriate AdSpec* table for each spec type
	specTableMap := map[string]string{
		"vehicle":      "AdSpecVehicle",
		"agricultural": "AdSpecAgricultural",
		"bicycle":      "AdSpecBicycle",
		"part":         "AdSpecPart",
	}
	specIDColumnMap := map[string]string{
		"vehicle":      "spec_vehicle_id",
		"agricultural": "spec_agricultural_id",
		"bicycle":      "spec_bicycle_id",
		"part":         "spec_part_id",
	}

	for specType, pairs := range adSpecByType {
		tableName := specTableMap[specType]
		idColumn := specIDColumnMap[specType]

		const specBatchSize = 1000
		for i := 0; i < len(pairs); i += specBatchSize {
			end := i + specBatchSize
			if end > len(pairs) {
				end = len(pairs)
			}
			batch := pairs[i:end]

			values := make([]string, len(batch))
			args := make([]interface{}, len(batch)*2)
			for j, pair := range batch {
				argStart := j*2 + 1
				values[j] = fmt.Sprintf("($%d, $%d)", argStart, argStart+1)
				args[j*2] = pair.adID
				args[j*2+1] = pair.specID
			}

			query := fmt.Sprintf(`INSERT INTO %s (ad_id, %s) VALUES %s ON CONFLICT DO NOTHING`, tableName, idColumn, strings.Join(values, ","))
			_, err := db.Exec(query, args...)
			if err != nil {
				logger.Warn("Failed to insert AdSpec batch", "table", tableName, "error", err)
			}
		}
	}

	logger.Info("Imported ads", "count", len(adBatch), "file", jsonFile)
	return nil
}
