package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/password"
	"github.com/parts-pile/site/user"
)

type MakeYearModel map[string]map[string]map[string][]string

func main() {
	// Initialize logging
	if err := logger.Init(config.LogLevel, config.LogFormat, config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	// Validate security configuration (initializes encryption key)
	config.SecurityCheck()

	parentFile := "cmd/rebuild_db/parent.json"
	makeParentFile := "cmd/rebuild_db/make-parent.json"
	typeFile := "cmd/rebuild_db/ad-category.json"
	schemaFile := "schema.sql"

	// Get PostgreSQL connection string from environment variable
	// Example: postgres://user:password@localhost:5432/dbname?sslmode=disable
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Fatal("DATABASE_URL must be set")
	}

	// Open DB for direct access
	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		logger.Fatal("Failed to open DB", "error", err, "databaseURL", databaseURL)
	}
	defer database.Close()

	// Test connection
	if err := database.Ping(); err != nil {
		logger.Fatal("Failed to ping database", "error", err)
	}

	// Drop all tables to clean the database
	logger.Info("Dropping all existing tables")
	var dropSQL string
	err = database.QueryRow(`
		SELECT 'DROP TABLE IF EXISTS ' || string_agg('"' || tablename || '"', ', ') || ' CASCADE'
		FROM pg_tables 
		WHERE schemaname = 'public'
	`).Scan(&dropSQL)
	if err != nil && err != sql.ErrNoRows {
		logger.Fatal("Failed to generate DROP statement", "error", err)
	}

	if dropSQL != "" {
		if _, err := database.Exec(dropSQL); err != nil {
			logger.Fatal("Failed to drop tables", "error", err)
		}
		logger.Info("All existing tables dropped")
	} else {
		logger.Info("No tables to drop")
	}

	// Read and execute schema.sql
	schemaSQL, err := os.ReadFile(schemaFile)
	if err != nil {
		logger.Fatal("Failed to read schema.sql", "error", err, "schemaFile", schemaFile)
	}

	// Use pgx directly to execute the entire schema file
	// pgx supports executing multiple statements in a single call
	logger.Info("Executing schema.sql using pgx")
	conn, err := pgx.Connect(context.Background(), databaseURL)
	if err != nil {
		logger.Fatal("Failed to connect with pgx", "error", err)
	}
	defer conn.Close(context.Background())

	// Execute the entire schema file - pgx can handle multiple statements
	if _, err := conn.Exec(context.Background(), string(schemaSQL)); err != nil {
		logger.Fatal("Failed to execute schema.sql", "error", err)
	}
	logger.Info("Schema executed successfully")

	// Import ad-category.json (AdCategory)
	typeData, err := os.ReadFile(typeFile)
	if err != nil {
		logger.Fatal("Failed to read ad-category.json", "error", err, "file", typeFile)
	}
	type AdCategory struct {
		Name       string   `json:"name"`
		SpecFile   string   `json:"spec_file,omitempty"`
		PartFile   string   `json:"part_file,omitempty"`
		AdFile     string   `json:"ad_file,omitempty"`
		FieldNames []string `json:"field_names,omitempty"`
	}
	var adCategories []AdCategory
	if err := json.Unmarshal(typeData, &adCategories); err != nil {
		logger.Fatal("Failed to parse type.json", "error", err, "file", typeFile)
	}

	// Create map for category lookups
	categoryMap := make(map[string]int)
	for _, cat := range adCategories {
		_, err := database.Exec(`INSERT INTO AdCategory (name) VALUES ($1)`, cat.Name)
		if err != nil {
			logger.Warn("Failed to insert AdCategory", "name", cat.Name, "error", err)
		}
	}
	logger.Info("Inserted AdCategories", "count", len(adCategories))

	// Populate category map
	logger.Info("Querying AdCategory table to populate category map")
	categoryRows, err := database.Query(`SELECT id, name FROM AdCategory`)
	if err != nil {
		logger.Fatal("Failed to query AdCategory", "error", err)
	}
	defer categoryRows.Close()

	categoryCount := 0
	for categoryRows.Next() {
		var id int
		var name string
		if err := categoryRows.Scan(&id, &name); err != nil {
			logger.Fatal("Failed to scan ad_category_id row", "error", err)
		}
		categoryMap[name] = id
		categoryCount++
	}
	if err := categoryRows.Err(); err != nil {
		logger.Fatal("Error iterating category rows", "error", err)
	}
	logger.Info("Populated category map", "count", categoryCount)

	// Populate SpecificationField table with all field definitions
	fieldFile := "cmd/rebuild_db/spec-fields.json"
	fieldData, err := os.ReadFile(fieldFile)
	if err != nil {
		logger.Fatal("Failed to read spec-fields.json", "error", err, "file", fieldFile)
	}
	type SpecificationFieldDef struct {
		Name            string `json:"name"`
		DisplayName     string `json:"display_name"`
		FieldType       string `json:"field_type"`
		TableName       string `json:"table_name"`
		TableFieldName  string `json:"table_field_name"`
		ValueColumnName string `json:"value_column_name"`
		QdrantType      string `json:"qdrant_type"`
	}
	var fieldDefinitions []SpecificationFieldDef
	if err := json.Unmarshal(fieldData, &fieldDefinitions); err != nil {
		logger.Fatal("Failed to parse spec-fields.json", "error", err, "file", fieldFile)
	}

	fieldIDMap := make(map[string]int)
	for _, fieldDef := range fieldDefinitions {
		var fieldID int
		err := database.QueryRow(`INSERT INTO SpecificationField (name, display_name, field_type, table_name, table_field_name, value_column_name, qdrant_type) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			fieldDef.Name, fieldDef.DisplayName, fieldDef.FieldType, fieldDef.TableName, fieldDef.TableFieldName, fieldDef.ValueColumnName, fieldDef.QdrantType).Scan(&fieldID)
		if err != nil {
			logger.Warn("Failed to insert SpecificationField", "name", fieldDef.Name, "error", err)
			continue
		}
		fieldIDMap[fieldDef.Name] = fieldID
	}
	logger.Info("Inserted SpecificationFields", "count", len(fieldIDMap))

	// Populate AdCategoryField table mapping categories to their fields
	// Field mappings are now in ad-category.json
	// Chain syntax: "field->" indicates field is chained to next field
	for _, cat := range adCategories {
		if len(cat.FieldNames) == 0 {
			logger.Debug("Category has no field mappings, skipping", "category", cat.Name)
			continue
		}

		categoryID, exists := categoryMap[cat.Name]
		if !exists {
			logger.Warn("Category not found for field mapping", "category", cat.Name)
			continue
		}

		// Parse field names and extract chain information
		// Build order -> fieldID map for calculating next_in_chain
		orderToFieldID := make(map[int]int)
		for order, fieldNameWithChain := range cat.FieldNames {
			// Remove "*" prefix (indicates show in filters) and "->" suffix (indicates chained) for database lookup
			fieldName := strings.TrimPrefix(fieldNameWithChain, "*") // Remove "*" prefix if present
			fieldName = strings.TrimSuffix(fieldName, "->")          // Remove "->" suffix if present
			fieldID, exists := fieldIDMap[fieldName]
			if !exists {
				logger.Warn("Field not found for category mapping", "field", fieldName, "category", cat.Name)
				continue
			}
			orderToFieldID[order+1] = fieldID
		}

		// Insert fields and calculate next_in_chain
		for order, fieldNameWithChain := range cat.FieldNames {
			// Parse field name: remove "*" prefix (indicates show in filters) and "->" suffix (indicates chained)
			fieldName := fieldNameWithChain
			showInFilters := 0
			if strings.HasPrefix(fieldName, "*") {
				showInFilters = 1
				fieldName = fieldName[1:] // Remove "*" prefix
			}
			fieldName = strings.TrimSuffix(fieldName, "->") // Remove "->" suffix if present

			fieldID, exists := fieldIDMap[fieldName]
			if !exists {
				continue
			}

			// Calculate next_in_chain: if field has "->" suffix, next field's order, else 0
			nextInChain := 0
			isChained := strings.HasSuffix(fieldNameWithChain, "->")
			if isChained && order+1 < len(cat.FieldNames) {
				// This field chains to the next field
				nextOrder := order + 2 // order is 0-based, DB order is 1-based
				nextInChain = nextOrder
			}

			_, err := database.Exec(`INSERT INTO AdCategoryField (ad_category_id, specification_field_id, priority_order, next_in_chain, is_required, show_in_filters) VALUES ($1, $2, $3, $4, $5, $6)`,
				categoryID, fieldID, order+1, nextInChain, 1, showInFilters)
			if err != nil {
				logger.Warn("Failed to insert AdCategoryField", "category", cat.Name, "field", fieldName, "error", err)
			}
		}
	}
	logger.Info("Populated AdCategoryField mappings")

	// Import parent.json
	parentData, err := os.ReadFile(parentFile)
	if err != nil {
		logger.Fatal("Failed to read parent.json", "error", err, "file", parentFile)
	}
	type ParentCompany struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	}
	var parentCompanies []ParentCompany
	if err := json.Unmarshal(parentData, &parentCompanies); err != nil {
		logger.Fatal("Failed to parse parent.json", "error", err, "file", parentFile)
	}
	for _, pc := range parentCompanies {
		_, err := database.Exec(`INSERT INTO ParentCompany (name, country) VALUES ($1, $2)`, pc.Name, pc.Country)
		if err != nil {
			logger.Warn("Failed to insert ParentCompany", "name", pc.Name, "error", err)
		}
	}
	logger.Info("Inserted ParentCompanies", "count", len(parentCompanies))

	// Import user.json
	userFile := "cmd/rebuild_db/user.json"
	userData, err := os.ReadFile(userFile)
	if err != nil {
		logger.Fatal("Failed to read user.json", "error", err, "file", userFile)
	}
	type UserImport struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
		IsAdmin  bool   `json:"is_admin"`
	}
	var users []UserImport
	if err := json.Unmarshal(userData, &users); err != nil {
		logger.Fatal("Failed to parse user.json", "error", err, "file", userFile)
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
		err = database.QueryRow(`INSERT INTO Users (encrypted_name, name_nonce, name_hash, encrypted_phone, phone_nonce, phone_hash,
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
		_, err = database.Exec(`UPDATE Users SET encrypted_name = $1, name_nonce = $2, encrypted_phone = $3, phone_nonce = $4 WHERE id = $5`,
			encryptedName, nameNonce, encryptedPhone, phoneNonce, userID)
		if err != nil {
			logger.Warn("Failed to update encrypted data for user", "name", u.Name, "error", err)
			continue
		}

		// Initialize rock inventory
		_, err = database.Exec(`INSERT INTO UserRock (user_id, rock_count) VALUES ($1, 3)`, userID)
		if err != nil {
			logger.Warn("Failed to initialize rock inventory for user", "name", u.Name, "error", err)
			continue
		}

		// Map user index (1-based) to actual user ID for ad imports
		userIDMap[i+1] = userID
		userCount++
	}
	logger.Info("Inserted Users", "count", userCount)

	// Import make-parent.json
	makeParentData, err := os.ReadFile(makeParentFile)
	if err != nil {
		logger.Fatal("Failed to read make-parent.json", "error", err, "file", makeParentFile)
	}
	type MakeParent struct {
		Make          string `json:"make"`
		ParentCompany string `json:"parent_company"`
	}
	var makeParents []MakeParent
	if err := json.Unmarshal(makeParentData, &makeParents); err != nil {
		logger.Fatal("Failed to parse make-parent.json", "error", err, "file", makeParentFile)
	}

	// Define MakeParentData for use in import functions
	makeParentsData := make([]struct {
		Make          string
		ParentCompany string
	}, len(makeParents))
	for i, mp := range makeParents {
		makeParentsData[i] = struct {
			Make          string
			ParentCompany string
		}{mp.Make, mp.ParentCompany}
	}

	// Create a map of parent company names to IDs for efficient lookup
	parentCompanyMap := make(map[string]int)
	rows, err := database.Query(`SELECT id, name FROM ParentCompany`)
	if err != nil {
		logger.Fatal("Failed to query ParentCompany", "error", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			logger.Fatal("Failed to scan ParentCompany row", "error", err)
		}
		parentCompanyMap[name] = id
	}

	// Create category to seed file mapping
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
				specFile: "cmd/rebuild_db/" + cat.SpecFile,
				partFile: "",
				adFile:   "cmd/rebuild_db/" + cat.AdFile,
			}
			if cat.PartFile != "" {
				seedFiles.partFile = "cmd/rebuild_db/" + cat.PartFile
			}
			categorySeedMap[cat.Name] = seedFiles
		}
	}

	// Process each category that has seed data
	for categoryName, seedFiles := range categorySeedMap {
		categoryID, exists := categoryMap[categoryName]
		if !exists {
			logger.Warn("Category not found, skipping", "category", categoryName)
			continue
		}

		logger.Info("Processing category", "category", categoryName)

		// Import specs for this category
		if err := importSpecs(database, seedFiles.specFile, categoryID, makeParentsData, parentCompanyMap); err != nil {
			logger.Warn("Failed to import specs for category", "category", categoryName, "error", err)
		}

		// Import parts for this category
		if seedFiles.partFile != "" {
			if err := importParts(database, seedFiles.partFile, categoryID); err != nil {
				logger.Warn("Failed to import parts for category", "category", categoryName, "error", err)
			}
		}

		// Import ads for this category
		if err := importAds(database, seedFiles.adFile, categoryID, categoryMap, userIDMap); err != nil {
			logger.Warn("Failed to import ads for category", "category", categoryName, "error", err)
		}
	}

	logger.Info("Database rebuild and import complete")
	logger.Info("Vector embeddings will be processed by the main application background processor")
}

// splitSQLStatements splits a SQL script into individual statements
// Handles semicolons inside string literals and comments
func splitSQLStatements(sql string) []string {
	var statements []string
	var current strings.Builder
	inString := false
	stringChar := byte(0)
	escapeNext := false

	for i := 0; i < len(sql); i++ {
		char := sql[i]

		if escapeNext {
			current.WriteByte(char)
			escapeNext = false
			continue
		}

		if char == '\\' && inString {
			escapeNext = true
			current.WriteByte(char)
			continue
		}

		if !inString {
			// Check for string start
			if char == '\'' || char == '"' {
				inString = true
				stringChar = char
				current.WriteByte(char)
				continue
			}

			// Check for comment start
			if i < len(sql)-1 && sql[i] == '-' && sql[i+1] == '-' {
				// Skip to end of line, but add newline to preserve structure
				for i < len(sql) && sql[i] != '\n' {
					i++
				}
				if i < len(sql) && sql[i] == '\n' {
					current.WriteByte('\n')
				}
				continue
			}

			// Check for statement delimiter
			if char == ';' {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" {
					statements = append(statements, stmt)
				}
				current.Reset()
				continue
			}
		} else {
			// Inside string
			if char == stringChar {
				inString = false
				stringChar = 0
			}
		}

		current.WriteByte(char)
	}

	// Add final statement if any
	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		statements = append(statements, stmt)
	}

	return statements
}

// truncateString truncates a string to maxLen characters for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// executeSpecBatch executes a batch of Specification insertions using a single INSERT statement
func executeSpecBatch(tx *sql.Tx, batch []struct {
	adCategoryID, makeID, yearID, modelID, engineID int
}) {
	// Use a single INSERT statement with multiple VALUES for better performance
	if len(batch) == 0 {
		return
	}

	// Build the VALUES clause with PostgreSQL placeholders
	values := make([]string, len(batch))
	args := make([]interface{}, len(batch)*5)

	for i, spec := range batch {
		argStart := i*5 + 1
		values[i] = fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", argStart, argStart+1, argStart+2, argStart+3, argStart+4)
		args[i*5] = spec.adCategoryID
		args[i*5+1] = spec.makeID
		args[i*5+2] = spec.yearID
		args[i*5+3] = spec.modelID
		args[i*5+4] = spec.engineID
	}

	query := fmt.Sprintf("INSERT INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id) VALUES %s ON CONFLICT DO NOTHING", strings.Join(values, ","))
	_, err := tx.Exec(query, args...)
	if err != nil {
		logger.Warn("Failed to insert Specification batch", "error", err)
	}
}

func getOrInsert(db *sql.DB, table, col, val string) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=$1", table, col), val).Scan(&id)
	if err == sql.ErrNoRows {
		err = db.QueryRow(fmt.Sprintf("INSERT INTO %s (%s) VALUES ($1) RETURNING id", table, col), val).Scan(&id)
		if err != nil {
			logger.Fatal("Failed to insert into table", "table", table, "error", err)
		}
		return id
	} else if err != nil {
		logger.Fatal("Failed to lookup table", "table", table, "error", err)
	}
	return id
}

func getOrInsertWithCategory(db *sql.DB, table, col, val string, categoryID int) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=$1 AND ad_category_id=$2", table, col), val, categoryID).Scan(&id)
	if err == sql.ErrNoRows {
		err = db.QueryRow(fmt.Sprintf("INSERT INTO %s (%s, ad_category_id) VALUES ($1, $2) RETURNING id", table, col), val, categoryID).Scan(&id)
		if err != nil {
			logger.Fatal("Failed to insert into table", "table", table, "error", err)
		}
		return id
	} else if err != nil {
		logger.Fatal("Failed to lookup table", "table", table, "error", err)
	}
	return id
}

// Helper functions for embedding generation
func interfaceSlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return fmt.Sprintf("%s", ss)
}

// importSpecs loads spec data from a JSON file and imports it into the database
func importSpecs(db *sql.DB, jsonFile string, categoryID int, makeParents []struct {
	Make          string
	ParentCompany string
}, parentCompanyMap map[string]int) error {
	f, err := os.Open(jsonFile)
	if err != nil {
		logger.Debug("Failed to open spec file, skipping", "error", err, "file", jsonFile)
		return nil // Skip missing files
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		logger.Warn("Failed to read spec file", "error", err, "file", jsonFile)
		return err
	}

	var mym MakeYearModel
	if err := json.Unmarshal(data, &mym); err != nil {
		logger.Warn("Failed to parse spec file", "error", err, "file", jsonFile)
		return err
	}

	// Collect all specs first, then batch insert
	type specEntry struct {
		adCategoryID, makeID, yearID, modelID, engineID int
	}
	var allSpecs []specEntry

	// Build map of make -> parentCompanyID for faster lookup
	makeToParentMap := make(map[string]*int)
	for _, mp := range makeParents {
		if id, exists := parentCompanyMap[mp.ParentCompany]; exists {
			makeToParentMap[mp.Make] = &id
		}
	}

	// First pass: collect all unique values
	type makeKey struct {
		name     string
		parentID *int
	}

	makeSet := make(map[makeKey]bool)
	yearSet := make(map[string]bool)
	modelSet := make(map[string]bool)
	engineSet := make(map[string]bool)

	for make, years := range mym {
		parentCompanyID := makeToParentMap[make]
		makeSet[makeKey{name: make, parentID: parentCompanyID}] = true

		for year, models := range years {
			yearSet[year] = true
			for model, engines := range models {
				modelSet[model] = true
				for _, engine := range engines {
					engineSet[engine] = true
				}
			}
		}
	}

	// Bulk load existing Makes, Years, Models, Engines and build lookup maps
	makeIDMap := make(map[makeKey]int)
	yearIDMap := make(map[string]int)
	modelIDMap := make(map[string]int)
	engineIDMap := make(map[string]int)

	// Load existing Makes
	for mk := range makeSet {
		var query string
		var args []interface{}
		if mk.parentID != nil {
			query = `SELECT id FROM Make WHERE name=$1 AND parent_company_id=$2 AND ad_category_id=$3`
			args = []interface{}{mk.name, *mk.parentID, categoryID}
		} else {
			query = `SELECT id FROM Make WHERE name=$1 AND parent_company_id IS NULL AND ad_category_id=$2`
			args = []interface{}{mk.name, categoryID}
		}
		var id int
		err := db.QueryRow(query, args...).Scan(&id)
		if err == nil {
			makeIDMap[mk] = id
		}
	}

	// Load existing Years
	if len(yearSet) > 0 {
		yearList := make([]interface{}, 0, len(yearSet))
		for year := range yearSet {
			yearList = append(yearList, year)
		}
		placeholders := make([]string, len(yearList))
		for i := range placeholders {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
		}
		query := fmt.Sprintf(`SELECT id, year FROM Year WHERE year IN (%s) AND ad_category_id=$1`, strings.Join(placeholders, ","))
		args := append([]interface{}{categoryID}, yearList...)
		rows, err := db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int
				var year string
				if rows.Scan(&id, &year) == nil {
					yearIDMap[year] = id
				}
			}
		}
	}

	// Load existing Models
	if len(modelSet) > 0 {
		modelList := make([]interface{}, 0, len(modelSet))
		for model := range modelSet {
			modelList = append(modelList, model)
		}
		placeholders := make([]string, len(modelList))
		for i := range placeholders {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
		}
		query := fmt.Sprintf(`SELECT id, name FROM Model WHERE name IN (%s) AND ad_category_id=$1`, strings.Join(placeholders, ","))
		args := append([]interface{}{categoryID}, modelList...)
		rows, err := db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int
				var model string
				if rows.Scan(&id, &model) == nil {
					modelIDMap[model] = id
				}
			}
		}
	}

	// Load existing Engines
	if len(engineSet) > 0 {
		engineList := make([]interface{}, 0, len(engineSet))
		for engine := range engineSet {
			engineList = append(engineList, engine)
		}
		placeholders := make([]string, len(engineList))
		for i := range placeholders {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
		}
		query := fmt.Sprintf(`SELECT id, name FROM Engine WHERE name IN (%s) AND ad_category_id=$1`, strings.Join(placeholders, ","))
		args := append([]interface{}{categoryID}, engineList...)
		rows, err := db.Query(query, args...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id int
				var engine string
				if rows.Scan(&id, &engine) == nil {
					engineIDMap[engine] = id
				}
			}
		}
	}

	// Batch insert missing Makes
	tx, err := db.Begin()
	if err == nil {
		for mk := range makeSet {
			if _, exists := makeIDMap[mk]; !exists {
				var id int
				var query string
				var args []interface{}
				if mk.parentID != nil {
					query = `INSERT INTO Make (name, parent_company_id, ad_category_id) VALUES ($1, $2, $3) RETURNING id`
					args = []interface{}{mk.name, *mk.parentID, categoryID}
				} else {
					query = `INSERT INTO Make (name, ad_category_id) VALUES ($1, $2) RETURNING id`
					args = []interface{}{mk.name, categoryID}
				}
				err := tx.QueryRow(query, args...).Scan(&id)
				if err == nil {
					makeIDMap[mk] = id
				}
			}
		}
		tx.Commit()
	}

	// Batch insert missing Years using VALUES clause
	missingYears := make([]string, 0)
	for year := range yearSet {
		if _, exists := yearIDMap[year]; !exists {
			missingYears = append(missingYears, year)
		}
	}
	if len(missingYears) > 0 {
		tx, err := db.Begin()
		if err == nil {
			values := make([]string, len(missingYears))
			args := make([]interface{}, len(missingYears)*2)
			for i, year := range missingYears {
				argStart := i*2 + 1
				values[i] = fmt.Sprintf("($%d, $%d)", argStart, argStart+1)
				args[i*2] = year
				args[i*2+1] = categoryID
			}
			query := fmt.Sprintf(`INSERT INTO Year (year, ad_category_id) VALUES %s RETURNING id, year`, strings.Join(values, ","))
			rows, err := tx.Query(query, args...)
			if err == nil {
				for rows.Next() {
					var id int
					var year string
					if rows.Scan(&id, &year) == nil {
						yearIDMap[year] = id
					}
				}
				rows.Close()
			}
			tx.Commit()
		}
	}

	// Batch insert missing Models using VALUES clause
	missingModels := make([]string, 0)
	for model := range modelSet {
		if _, exists := modelIDMap[model]; !exists {
			missingModels = append(missingModels, model)
		}
	}
	if len(missingModels) > 0 {
		tx, err := db.Begin()
		if err == nil {
			values := make([]string, len(missingModels))
			args := make([]interface{}, len(missingModels)*2)
			for i, model := range missingModels {
				argStart := i*2 + 1
				values[i] = fmt.Sprintf("($%d, $%d)", argStart, argStart+1)
				args[i*2] = model
				args[i*2+1] = categoryID
			}
			query := fmt.Sprintf(`INSERT INTO Model (name, ad_category_id) VALUES %s RETURNING id, name`, strings.Join(values, ","))
			rows, err := tx.Query(query, args...)
			if err == nil {
				for rows.Next() {
					var id int
					var model string
					if rows.Scan(&id, &model) == nil {
						modelIDMap[model] = id
					}
				}
				rows.Close()
			}
			tx.Commit()
		}
	}

	// Batch insert missing Engines using VALUES clause
	missingEngines := make([]string, 0)
	for engine := range engineSet {
		if _, exists := engineIDMap[engine]; !exists {
			missingEngines = append(missingEngines, engine)
		}
	}
	if len(missingEngines) > 0 {
		tx, err := db.Begin()
		if err == nil {
			values := make([]string, len(missingEngines))
			args := make([]interface{}, len(missingEngines)*2)
			for i, engine := range missingEngines {
				argStart := i*2 + 1
				values[i] = fmt.Sprintf("($%d, $%d)", argStart, argStart+1)
				args[i*2] = engine
				args[i*2+1] = categoryID
			}
			query := fmt.Sprintf(`INSERT INTO Engine (name, ad_category_id) VALUES %s RETURNING id, name`, strings.Join(values, ","))
			rows, err := tx.Query(query, args...)
			if err == nil {
				for rows.Next() {
					var id int
					var engine string
					if rows.Scan(&id, &engine) == nil {
						engineIDMap[engine] = id
					}
				}
				rows.Close()
			}
			tx.Commit()
		}
	}

	// Second pass: build specs using lookup maps (no DB queries)
	for make, years := range mym {
		parentCompanyID := makeToParentMap[make]
		makeKey := makeKey{name: make, parentID: parentCompanyID}
		makeID := makeIDMap[makeKey]

		for year, models := range years {
			yearID := yearIDMap[year]

			for model, engines := range models {
				modelID := modelIDMap[model]

				for _, engine := range engines {
					engineID := engineIDMap[engine]

					allSpecs = append(allSpecs, specEntry{
						adCategoryID: categoryID,
						makeID:       makeID,
						yearID:       yearID,
						modelID:      modelID,
						engineID:     engineID,
					})
				}
			}
		}
	}

	// Batch insert specs using transaction
	const batchSize = 1000
	tx, err = db.Begin()
	if err != nil {
		logger.Warn("Failed to begin transaction for spec batch", "error", err)
		return err
	}
	defer tx.Rollback()

	for i := 0; i < len(allSpecs); i += batchSize {
		end := i + batchSize
		if end > len(allSpecs) {
			end = len(allSpecs)
		}
		batch := allSpecs[i:end]

		// Build batch for executeSpecBatch
		specBatch := make([]struct {
			adCategoryID, makeID, yearID, modelID, engineID int
		}, len(batch))
		for j, spec := range batch {
			specBatch[j] = struct {
				adCategoryID, makeID, yearID, modelID, engineID int
			}{
				adCategoryID: spec.adCategoryID,
				makeID:       spec.makeID,
				yearID:       spec.yearID,
				modelID:      spec.modelID,
				engineID:     spec.engineID,
			}
		}

		executeSpecBatch(tx, specBatch)
	}

	if err := tx.Commit(); err != nil {
		logger.Warn("Failed to commit spec batch transaction", "error", err)
		return err
	}

	logger.Info("Imported specs", "file", jsonFile, "count", len(allSpecs))
	return nil
}

// importParts loads part data from a JSON file and imports it into the database
func importParts(db *sql.DB, jsonFile string, categoryID int) error {
	partData, err := os.ReadFile(jsonFile)
	if err != nil {
		logger.Debug("Failed to read part file, skipping", "error", err, "file", jsonFile)
		return nil // Skip missing files
	}

	var partMap map[string][]string
	if err := json.Unmarshal(partData, &partMap); err != nil {
		logger.Warn("Failed to parse part file", "error", err, "file", jsonFile)
		return err
	}

	for cat, subcats := range partMap {
		// Insert PartCategory (needed for Specification table)
		getOrInsertWithCategory(db, "PartCategory", "name", cat, categoryID)
		for _, subcat := range subcats {
			var subcatID int
			// PartSubCategory now uses ad_category_id directly, not part_category_id
			err := db.QueryRow(`SELECT id FROM PartSubCategory WHERE ad_category_id=$1 AND name=$2`, categoryID, subcat).Scan(&subcatID)
			if err == sql.ErrNoRows {
				_, err := db.Exec(`INSERT INTO PartSubCategory (ad_category_id, name) VALUES ($1, $2)`, categoryID, subcat)
				if err != nil {
					logger.Warn("Failed to insert PartSubCategory", "error", err)
				}
			} else if err != nil {
				logger.Warn("PartSubCategory lookup error", "error", err)
			}
		}
	}

	logger.Info("Imported parts", "file", jsonFile)
	return nil
}

// importAds loads ad data from a JSON file and imports it into the database
func importAds(db *sql.DB, jsonFile string, categoryID int, categoryMap map[string]int, userIDMap map[int]int) error {
	adData, err := os.ReadFile(jsonFile)
	if err != nil {
		logger.Debug("Failed to read ad file, skipping", "error", err, "file", jsonFile)
		return nil // Skip missing files
	}

	type AdImport struct {
		Make        string   `json:"make"`
		Years       []string `json:"years"`
		Models      []string `json:"models"`
		Engines     []string `json:"engines"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Price       float64  `json:"price"`
		CreatedAt   string   `json:"created_at"`
		UserID      int      `json:"user_id"`
		Category    string   `json:"category"`
		Subcategory string   `json:"subcategory"`
		Location    struct {
			City      string   `json:"city"`
			AdminArea string   `json:"admin_area"`
			Country   string   `json:"country"`
			Latitude  *float64 `json:"latitude"`
			Longitude *float64 `json:"longitude"`
		} `json:"location"`
	}

	var ads []AdImport
	if err := json.Unmarshal(adData, &ads); err != nil {
		logger.Warn("Failed to parse ad file", "error", err, "file", jsonFile)
		return err
	}

	// Get part categories and subcategories
	partCategoryMap := make(map[string]int)
	subcategoryMap := make(map[string]int)

	rows, err := db.Query(`SELECT id, name FROM PartCategory WHERE ad_category_id=$1`, categoryID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int
			var name string
			if err := rows.Scan(&id, &name); err == nil {
				partCategoryMap[name] = id
			}
		}
	}

	rows2, err := db.Query(`SELECT id, name FROM PartSubCategory WHERE ad_category_id=$1`, categoryID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id int
			var name string
			if err := rows2.Scan(&id, &name); err == nil {
				subcategoryMap[name] = id
			}
		}
	}

	adCount := 0
	for _, ad := range ads {
		// Insert or get location
		locationKey := fmt.Sprintf("%s, %s, %s", ad.Location.City, ad.Location.AdminArea, ad.Location.Country)
		var locationID int
		err := db.QueryRow(`SELECT id FROM Location WHERE raw_text=$1`, locationKey).Scan(&locationID)
		if err == sql.ErrNoRows {
			err = db.QueryRow(`INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
				locationKey, ad.Location.City, ad.Location.AdminArea, ad.Location.Country, ad.Location.Latitude, ad.Location.Longitude).Scan(&locationID)
			if err != nil {
				logger.Warn("Failed to insert Location", "error", err)
				continue
			}
		} else if err != nil {
			logger.Warn("Location lookup error", "error", err)
			continue
		}

		// Get part_category_id and part_subcategory_id (NULL for whole spec ads, IDs for parts ads)
		var partCategoryID *int
		var subcategoryID *int
		if ad.Category != "" {
			// This is a parts ad - get part_category_id
			partCatID, exists := partCategoryMap[ad.Category]
			if !exists {
				logger.Warn("No valid part category found for ad, skipping", "title", ad.Title, "category", ad.Category)
				continue
			}
			partCategoryID = &partCatID

			// Get part_subcategory_id
			subcatID := 0
			if ad.Subcategory != "" {
				// PartSubCategory now uses ad_category_id directly, not part_category_id
				subcatRows, err := db.Query(`SELECT id FROM PartSubCategory WHERE ad_category_id=$1 AND name=$2`, categoryID, ad.Subcategory)
				if err == nil {
					if subcatRows.Next() {
						subcatRows.Scan(&subcatID)
					}
					subcatRows.Close()
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
				logger.Warn("No valid subcategory found for ad, skipping", "title", ad.Title)
				continue
			}
			subcategoryID = &subcatID
		}
		// For whole spec ads (ad.Category == ""), partCategoryID and subcategoryID stay nil

		// Map user ID from JSON to actual user ID
		actualUserID, exists := userIDMap[ad.UserID]
		if !exists {
			logger.Warn("User ID not found in userIDMap, skipping ad", "userID", ad.UserID, "title", ad.Title)
			continue
		}

		// Generate between 0 and 5 images per ad
		numImages := rand.Intn(6)

		// Insert ad
		var adID int
		err = db.QueryRow(`INSERT INTO Ad (title, description, price, created_at, user_id, location_id, image_count, has_vector, ad_category_id) VALUES ($1, $2, $3, $4, $5, $6, $7, 0, $8) RETURNING id`,
			ad.Title, ad.Description, ad.Price, ad.CreatedAt, actualUserID, locationID, numImages, categoryID).Scan(&adID)
		if err != nil {
			logger.Warn("Failed to insert Ad", "error", err, "title", ad.Title)
			continue
		}

		// Link to specs (only if ad has years/engines fields)
		if len(ad.Years) > 0 && len(ad.Engines) > 0 {
			for _, year := range ad.Years {
				for _, model := range ad.Models {
					for _, engine := range ad.Engines {
						// Query for spec with part_category_id and part_subcategory_id (may be NULL)
						var specID int
						var query string
						var args []interface{}
						if partCategoryID != nil && subcategoryID != nil {
							query = `SELECT spec.id FROM Specification spec 
								LEFT JOIN Make m ON spec.make_id=m.id
								LEFT JOIN Year y ON spec.year_id=y.id
								LEFT JOIN Model mdl ON spec.model_id=mdl.id
								LEFT JOIN Engine e ON spec.engine_id=e.id
								WHERE m.name=$1 AND y.year=$2 AND mdl.name=$3 AND e.name=$4 AND spec.ad_category_id=$5 AND spec.part_category_id=$6 AND spec.part_subcategory_id=$7`
							args = []interface{}{ad.Make, year, model, engine, categoryID, *partCategoryID, *subcategoryID}
						} else if partCategoryID != nil {
							query = `SELECT spec.id FROM Specification spec 
								LEFT JOIN Make m ON spec.make_id=m.id
								LEFT JOIN Year y ON spec.year_id=y.id
								LEFT JOIN Model mdl ON spec.model_id=mdl.id
								LEFT JOIN Engine e ON spec.engine_id=e.id
								WHERE m.name=$1 AND y.year=$2 AND mdl.name=$3 AND e.name=$4 AND spec.ad_category_id=$5 AND spec.part_category_id=$6 AND spec.part_subcategory_id IS NULL`
							args = []interface{}{ad.Make, year, model, engine, categoryID, *partCategoryID}
						} else {
							query = `SELECT spec.id FROM Specification spec 
								LEFT JOIN Make m ON spec.make_id=m.id
								LEFT JOIN Year y ON spec.year_id=y.id
								LEFT JOIN Model mdl ON spec.model_id=mdl.id
								LEFT JOIN Engine e ON spec.engine_id=e.id
								WHERE m.name=$1 AND y.year=$2 AND mdl.name=$3 AND e.name=$4 AND spec.ad_category_id=$5 AND spec.part_category_id IS NULL AND spec.part_subcategory_id IS NULL`
							args = []interface{}{ad.Make, year, model, engine, categoryID}
						}
						err := db.QueryRow(query, args...).Scan(&specID)
						if err != nil {
							// Spec not found - need to create it
							// First get the IDs for make, year, model, engine
							var makeID, yearID, modelID, engineID int
							err = db.QueryRow(`SELECT id FROM Make WHERE name=$1 AND ad_category_id=$2`, ad.Make, categoryID).Scan(&makeID)
							if err != nil {
								logger.Warn("Failed to find make", "make", ad.Make, "error", err)
								continue
							}
							err = db.QueryRow(`SELECT id FROM Year WHERE year=$1 AND ad_category_id=$2`, year, categoryID).Scan(&yearID)
							if err != nil {
								logger.Warn("Failed to find year", "year", year, "error", err)
								continue
							}
							err = db.QueryRow(`SELECT id FROM Model WHERE name=$1 AND ad_category_id=$2`, model, categoryID).Scan(&modelID)
							if err != nil {
								logger.Warn("Failed to find model", "model", model, "error", err)
								continue
							}
							// Use getOrInsertWithCategory to create engine if it doesn't exist
							engineID = getOrInsertWithCategory(db, "Engine", "name", engine, categoryID)

							// Insert spec with part_category_id and part_subcategory_id (may be NULL)
							if partCategoryID != nil && subcategoryID != nil {
								err = db.QueryRow(`INSERT INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id, part_category_id, part_subcategory_id) VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT DO NOTHING RETURNING id`,
									categoryID, makeID, yearID, modelID, engineID, *partCategoryID, *subcategoryID).Scan(&specID)
							} else if partCategoryID != nil {
								err = db.QueryRow(`INSERT INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id, part_category_id, part_subcategory_id) VALUES ($1, $2, $3, $4, $5, $6, NULL) ON CONFLICT DO NOTHING RETURNING id`,
									categoryID, makeID, yearID, modelID, engineID, *partCategoryID).Scan(&specID)
							} else {
								err = db.QueryRow(`INSERT INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id, part_category_id, part_subcategory_id) VALUES ($1, $2, $3, $4, $5, NULL, NULL) ON CONFLICT DO NOTHING RETURNING id`,
									categoryID, makeID, yearID, modelID, engineID).Scan(&specID)
							}
							if err == sql.ErrNoRows {
								// Spec already exists, fetch it
								var lookupQuery string
								var lookupArgs []interface{}
								if partCategoryID != nil && subcategoryID != nil {
									lookupQuery = `SELECT id FROM Specification WHERE ad_category_id=$1 AND make_id=$2 AND year_id=$3 AND model_id=$4 AND engine_id=$5 AND part_category_id=$6 AND part_subcategory_id=$7`
									lookupArgs = []interface{}{categoryID, makeID, yearID, modelID, engineID, *partCategoryID, *subcategoryID}
								} else if partCategoryID != nil {
									lookupQuery = `SELECT id FROM Specification WHERE ad_category_id=$1 AND make_id=$2 AND year_id=$3 AND model_id=$4 AND engine_id=$5 AND part_category_id=$6 AND part_subcategory_id IS NULL`
									lookupArgs = []interface{}{categoryID, makeID, yearID, modelID, engineID, *partCategoryID}
								} else {
									lookupQuery = `SELECT id FROM Specification WHERE ad_category_id=$1 AND make_id=$2 AND year_id=$3 AND model_id=$4 AND engine_id=$5 AND part_category_id IS NULL AND part_subcategory_id IS NULL`
									lookupArgs = []interface{}{categoryID, makeID, yearID, modelID, engineID}
								}
								err = db.QueryRow(lookupQuery, lookupArgs...).Scan(&specID)
								if err != nil {
									logger.Warn("Failed to find created spec", "error", err)
									continue
								}
							} else if err != nil {
								logger.Warn("Failed to insert spec", "error", err)
								continue
							}
						}

						// Insert AdSpecification relationship
						db.Exec(`INSERT INTO AdSpecification (ad_id, specification_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, adID, specID)
					}
				}
			}
		}

		adCount++
	}

	logger.Info("Imported ads", "count", adCount, "file", jsonFile)
	return nil
}

// getOrInsertWithParentAndCategoryHelper helper that supports optional parentID
func getOrInsertWithParentAndCategoryHelper(db *sql.DB, table, col, val string, categoryID int, parentID *int) int {
	if parentID != nil && *parentID > 0 {
		return getOrInsertWithParentAndCategoryFull(db, table, col, val, categoryID, *parentID)
	}
	return getOrInsertWithCategory(db, table, col, val, categoryID)
}

func getOrInsertWithParentAndCategoryFull(db *sql.DB, table, col, val string, categoryID, parentID int) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=$1 AND parent_company_id=$2 AND ad_category_id=$3", table, col), val, parentID, categoryID).Scan(&id)
	if err == sql.ErrNoRows {
		err = db.QueryRow(fmt.Sprintf("INSERT INTO %s (%s, parent_company_id, ad_category_id) VALUES ($1, $2, $3) RETURNING id", table, col), val, parentID, categoryID).Scan(&id)
		if err != nil {
			logger.Fatal("Failed to insert into table", "table", table, "error", err)
		}
		return id
	} else if err != nil {
		logger.Fatal("Failed to lookup table", "table", table, "error", err)
	}
	return id
}
