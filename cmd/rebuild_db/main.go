package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
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
	dbFile := config.DatabaseURL
	schemaFile := "schema.sql"

	dbFileOnDisk := strings.TrimPrefix(dbFile, "file:")
	// Remove old DB if exists
	if _, err := os.Stat(dbFileOnDisk); err == nil {
		logger.Info("Removing existing database file", "file", dbFileOnDisk)
		if err := os.Remove(dbFileOnDisk); err != nil {
			logger.Fatal("Failed to remove old DB", "error", err, "file", dbFileOnDisk)
		}
	} else {
		logger.Info("No existing database file to remove", "file", dbFileOnDisk)
	}

	// Create new DB from schema.sql
	cmd := exec.Command("sqlite3", dbFile, fmt.Sprintf(".read %s", schemaFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Fatal("Failed to create DB from schema.sql", "error", err, "schemaFile", schemaFile)
	}

	// Initialize database
	if err := db.Init(dbFile); err != nil {
		logger.Fatal("Failed to initialize database", "error", err, "databaseURL", dbFile)
	}

	// Open DB for direct access
	database, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		logger.Fatal("Failed to open DB", "error", err, "databaseURL", dbFile)
	}
	defer database.Close()

	// Enable WAL mode and other optimizations for bulk operations
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		logger.Warn("Failed to enable WAL mode", "error", err)
	}
	if _, err := database.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		logger.Warn("Failed to set synchronous mode", "error", err)
	}
	if _, err := database.Exec("PRAGMA cache_size=10000"); err != nil {
		logger.Warn("Failed to set cache size", "error", err)
	}
	if _, err := database.Exec("PRAGMA temp_store=MEMORY"); err != nil {
		logger.Warn("Failed to set temp store", "error", err)
	}

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
		_, err := database.Exec(`INSERT INTO AdCategory (name) VALUES (?)`, cat.Name)
		if err != nil {
			logger.Warn("Failed to insert AdCategory", "name", cat.Name, "error", err)
		}
	}
	logger.Info("Inserted AdCategories", "count", len(adCategories))

	// Populate category map
	categoryRows, err := database.Query(`SELECT id, name FROM AdCategory`)
	if err != nil {
		logger.Fatal("Failed to query AdCategory", "error", err)
	}
	defer categoryRows.Close()
	for categoryRows.Next() {
		var id int
		var name string
		if err := categoryRows.Scan(&id, &name); err != nil {
			logger.Fatal("Failed to scan ad_category_id row", "error", err)
		}
		categoryMap[name] = id
	}

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
		res, err := database.Exec(`INSERT INTO SpecificationField (name, display_name, field_type, table_name, table_field_name, value_column_name, qdrant_type) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fieldDef.Name, fieldDef.DisplayName, fieldDef.FieldType, fieldDef.TableName, fieldDef.TableFieldName, fieldDef.ValueColumnName, fieldDef.QdrantType)
		if err != nil {
			logger.Warn("Failed to insert SpecificationField", "name", fieldDef.Name, "error", err)
			continue
		}
		fieldID, _ := res.LastInsertId()
		fieldIDMap[fieldDef.Name] = int(fieldID)
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

			_, err := database.Exec(`INSERT INTO AdCategoryField (ad_category_id, specification_field_id, priority_order, next_in_chain, is_required, show_in_filters) VALUES (?, ?, ?, ?, ?, ?)`,
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
		_, err := database.Exec(`INSERT INTO ParentCompany (name, country) VALUES (?, ?)`, pc.Name, pc.Country)
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
	userCount := 0
	for _, u := range users {
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
		res, err := database.Exec(`INSERT INTO User (encrypted_name, name_nonce, name_hash, encrypted_phone, phone_nonce, phone_hash,
			password_hash, password_salt, password_algo, phone_verified,
			verification_code, notification_method, is_admin)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, '', 'sms', ?)`,
			encryptedName, nameNonce, nameHash, encryptedPhone, phoneNonce, phoneHash, hash, salt, "argon2id", u.IsAdmin)
		if err != nil {
			logger.Warn("Failed to insert user", "name", u.Name, "error", err)
			continue
		}

		// Get the user ID and re-encrypt with the real ID
		userID64, _ := res.LastInsertId()
		userID := int(userID64)

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
		_, err = database.Exec(`UPDATE User SET encrypted_name = ?, name_nonce = ?, encrypted_phone = ?, phone_nonce = ? WHERE id = ?`,
			encryptedName, nameNonce, encryptedPhone, phoneNonce, userID)
		if err != nil {
			logger.Warn("Failed to update encrypted data for user", "name", u.Name, "error", err)
			continue
		}

		// Initialize rock inventory
		_, err = database.Exec(`INSERT INTO UserRock (user_id, rock_count) VALUES (?, 3)`, userID)
		if err != nil {
			logger.Warn("Failed to initialize rock inventory for user", "name", u.Name, "error", err)
			continue
		}

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
		if err := importAds(database, seedFiles.adFile, categoryID, categoryMap); err != nil {
			logger.Warn("Failed to import ads for category", "category", categoryName, "error", err)
		}
	}

	logger.Info("Database rebuild and import complete")
	logger.Info("Vector embeddings will be processed by the main application background processor")
}

// executeSpecBatch executes a batch of Specification insertions using a single INSERT statement
func executeSpecBatch(tx *sql.Tx, batch []struct {
	adCategoryID, makeID, yearID, modelID, engineID int
}) {
	// Use a single INSERT statement with multiple VALUES for better performance
	if len(batch) == 0 {
		return
	}

	// Build the VALUES clause
	values := make([]string, len(batch))
	args := make([]interface{}, len(batch)*5)

	for i, spec := range batch {
		values[i] = "(?, ?, ?, ?, ?)"
		args[i*5] = spec.adCategoryID
		args[i*5+1] = spec.makeID
		args[i*5+2] = spec.yearID
		args[i*5+3] = spec.modelID
		args[i*5+4] = spec.engineID
	}

	query := fmt.Sprintf("INSERT OR IGNORE INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id) VALUES %s", strings.Join(values, ","))
	_, err := tx.Exec(query, args...)
	if err != nil {
		logger.Warn("Failed to insert Specification batch", "error", err)
	}
}

func getOrInsert(db *sql.DB, table, col, val string) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=?", table, col), val).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (?)", table, col), val)
		if err != nil {
			logger.Fatal("Failed to insert into table", "table", table, "error", err)
		}
		id64, _ := res.LastInsertId()
		return int(id64)
	} else if err != nil {
		logger.Fatal("Failed to lookup table", "table", table, "error", err)
	}
	return id
}

func getOrInsertWithCategory(db *sql.DB, table, col, val string, categoryID int) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=? AND ad_category_id=?", table, col), val, categoryID).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s, ad_category_id) VALUES (?, ?)", table, col), val, categoryID)
		if err != nil {
			logger.Fatal("Failed to insert into table", "table", table, "error", err)
		}
		id64, _ := res.LastInsertId()
		return int(id64)
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

	// Import specs with the category
	for make, years := range mym {
		// Find parent company ID for this make
		var parentCompanyID *int
		for _, mp := range makeParents {
			if mp.Make == make {
				if id, exists := parentCompanyMap[mp.ParentCompany]; exists {
					parentCompanyID = &id
				}
				break
			}
		}

		for year, models := range years {
			for model, engines := range models {
				for _, engine := range engines {
					makeID := getOrInsertWithParentAndCategoryHelper(db, "Make", "name", make, categoryID, parentCompanyID)
					yearID := getOrInsertWithCategory(db, "Year", "year", year, categoryID)
					modelID := getOrInsertWithCategory(db, "Model", "name", model, categoryID)
					engineID := getOrInsertWithCategory(db, "Engine", "name", engine, categoryID)

					// Insert spec
					db.Exec(`INSERT OR IGNORE INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id) VALUES (?, ?, ?, ?, ?)`,
						categoryID, makeID, yearID, modelID, engineID)
				}
			}
		}
	}

	logger.Info("Imported specs", "file", jsonFile)
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
			err := db.QueryRow(`SELECT id FROM PartSubCategory WHERE ad_category_id=? AND name=?`, categoryID, subcat).Scan(&subcatID)
			if err == sql.ErrNoRows {
				_, err := db.Exec(`INSERT INTO PartSubCategory (ad_category_id, name) VALUES (?, ?)`, categoryID, subcat)
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
func importAds(db *sql.DB, jsonFile string, categoryID int, categoryMap map[string]int) error {
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

	rows, err := db.Query(`SELECT id, name FROM PartCategory WHERE ad_category_id=?`, categoryID)
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

	rows2, err := db.Query(`SELECT id, name FROM PartSubCategory WHERE ad_category_id=?`, categoryID)
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
		err := db.QueryRow(`SELECT id FROM Location WHERE raw_text=?`, locationKey).Scan(&locationID)
		if err == sql.ErrNoRows {
			res, err := db.Exec(`INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)`,
				locationKey, ad.Location.City, ad.Location.AdminArea, ad.Location.Country, ad.Location.Latitude, ad.Location.Longitude)
			if err != nil {
				logger.Warn("Failed to insert Location", "error", err)
				continue
			}
			id64, _ := res.LastInsertId()
			locationID = int(id64)
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
				subcatRows, err := db.Query(`SELECT id FROM PartSubCategory WHERE ad_category_id=? AND name=?`, categoryID, ad.Subcategory)
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

		// Generate between 0 and 5 images per ad
		numImages := rand.Intn(6)

		// Insert ad
		res, err := db.Exec(`INSERT INTO Ad (title, description, price, created_at, user_id, location_id, image_count, has_vector, ad_category_id) VALUES (?, ?, ?, ?, ?, ?, ?, 0, ?)`,
			ad.Title, ad.Description, ad.Price, ad.CreatedAt, ad.UserID, locationID, numImages, categoryID)
		if err != nil {
			logger.Warn("Failed to insert Ad", "error", err, "title", ad.Title)
			continue
		}
		adID, _ := res.LastInsertId()

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
								WHERE m.name=? AND y.year=? AND mdl.name=? AND e.name=? AND spec.ad_category_id=? AND spec.part_category_id=? AND spec.part_subcategory_id=?`
							args = []interface{}{ad.Make, year, model, engine, categoryID, *partCategoryID, *subcategoryID}
						} else if partCategoryID != nil {
							query = `SELECT spec.id FROM Specification spec 
								LEFT JOIN Make m ON spec.make_id=m.id
								LEFT JOIN Year y ON spec.year_id=y.id
								LEFT JOIN Model mdl ON spec.model_id=mdl.id
								LEFT JOIN Engine e ON spec.engine_id=e.id
								WHERE m.name=? AND y.year=? AND mdl.name=? AND e.name=? AND spec.ad_category_id=? AND spec.part_category_id=? AND spec.part_subcategory_id IS NULL`
							args = []interface{}{ad.Make, year, model, engine, categoryID, *partCategoryID}
						} else {
							query = `SELECT spec.id FROM Specification spec 
								LEFT JOIN Make m ON spec.make_id=m.id
								LEFT JOIN Year y ON spec.year_id=y.id
								LEFT JOIN Model mdl ON spec.model_id=mdl.id
								LEFT JOIN Engine e ON spec.engine_id=e.id
								WHERE m.name=? AND y.year=? AND mdl.name=? AND e.name=? AND spec.ad_category_id=? AND spec.part_category_id IS NULL AND spec.part_subcategory_id IS NULL`
							args = []interface{}{ad.Make, year, model, engine, categoryID}
						}
						err := db.QueryRow(query, args...).Scan(&specID)
						if err != nil {
							// Spec not found - need to create it
							// First get the IDs for make, year, model, engine
							var makeID, yearID, modelID, engineID int
							err = db.QueryRow(`SELECT id FROM Make WHERE name=? AND ad_category_id=?`, ad.Make, categoryID).Scan(&makeID)
							if err != nil {
								logger.Warn("Failed to find make", "make", ad.Make, "error", err)
								continue
							}
							err = db.QueryRow(`SELECT id FROM Year WHERE year=? AND ad_category_id=?`, year, categoryID).Scan(&yearID)
							if err != nil {
								logger.Warn("Failed to find year", "year", year, "error", err)
								continue
							}
							err = db.QueryRow(`SELECT id FROM Model WHERE name=? AND ad_category_id=?`, model, categoryID).Scan(&modelID)
							if err != nil {
								logger.Warn("Failed to find model", "model", model, "error", err)
								continue
							}
							// Use getOrInsertWithCategory to create engine if it doesn't exist
							engineID = getOrInsertWithCategory(db, "Engine", "name", engine, categoryID)

							// Insert spec with part_category_id and part_subcategory_id (may be NULL)
							var specRes sql.Result
							if partCategoryID != nil && subcategoryID != nil {
								specRes, err = db.Exec(`INSERT OR IGNORE INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id, part_category_id, part_subcategory_id) VALUES (?, ?, ?, ?, ?, ?, ?)`,
									categoryID, makeID, yearID, modelID, engineID, *partCategoryID, *subcategoryID)
							} else if partCategoryID != nil {
								specRes, err = db.Exec(`INSERT OR IGNORE INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id, part_category_id, part_subcategory_id) VALUES (?, ?, ?, ?, ?, ?, NULL)`,
									categoryID, makeID, yearID, modelID, engineID, *partCategoryID)
							} else {
								specRes, err = db.Exec(`INSERT OR IGNORE INTO Specification (ad_category_id, make_id, year_id, model_id, engine_id, part_category_id, part_subcategory_id) VALUES (?, ?, ?, ?, ?, NULL, NULL)`,
									categoryID, makeID, yearID, modelID, engineID)
							}
							if err != nil {
								logger.Warn("Failed to insert spec", "error", err)
								continue
							}
							specID64, _ := specRes.LastInsertId()
							if specID64 == 0 {
								// Spec already exists, fetch it
								var lookupQuery string
								var lookupArgs []interface{}
								if partCategoryID != nil && subcategoryID != nil {
									lookupQuery = `SELECT id FROM Specification WHERE ad_category_id=? AND make_id=? AND year_id=? AND model_id=? AND engine_id=? AND part_category_id=? AND part_subcategory_id=?`
									lookupArgs = []interface{}{categoryID, makeID, yearID, modelID, engineID, *partCategoryID, *subcategoryID}
								} else if partCategoryID != nil {
									lookupQuery = `SELECT id FROM Specification WHERE ad_category_id=? AND make_id=? AND year_id=? AND model_id=? AND engine_id=? AND part_category_id=? AND part_subcategory_id IS NULL`
									lookupArgs = []interface{}{categoryID, makeID, yearID, modelID, engineID, *partCategoryID}
								} else {
									lookupQuery = `SELECT id FROM Specification WHERE ad_category_id=? AND make_id=? AND year_id=? AND model_id=? AND engine_id=? AND part_category_id IS NULL AND part_subcategory_id IS NULL`
									lookupArgs = []interface{}{categoryID, makeID, yearID, modelID, engineID}
								}
								err = db.QueryRow(lookupQuery, lookupArgs...).Scan(&specID)
								if err != nil {
									logger.Warn("Failed to find created spec", "error", err)
									continue
								}
							} else {
								specID = int(specID64)
							}
						}

						// Insert AdSpecification relationship
						db.Exec(`INSERT OR IGNORE INTO AdSpecification (ad_id, specification_id) VALUES (?, ?)`, adID, specID)
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
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=? AND parent_company_id=? AND ad_category_id=?", table, col), val, parentID, categoryID).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s, parent_company_id, ad_category_id) VALUES (?, ?, ?)", table, col), val, parentID, categoryID)
		if err != nil {
			logger.Fatal("Failed to insert into table", "table", table, "error", err)
		}
		id64, _ := res.LastInsertId()
		return int(id64)
	} else if err != nil {
		logger.Fatal("Failed to lookup table", "table", table, "error", err)
	}
	return id
}
