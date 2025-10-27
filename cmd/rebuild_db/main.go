package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/password"
)

type MakeYearModel map[string]map[string]map[string][]string

func main() {
	parentFile := "cmd/rebuild_db/parent.json"
	makeParentFile := "cmd/rebuild_db/make-parent.json"
	typeFile := "cmd/rebuild_db/ad-category.json"
	dbFile := config.DatabaseURL
	schemaFile := "schema.sql"

	dbFileOnDisk := strings.TrimPrefix(dbFile, "file:")
	// Remove old DB if exists
	if _, err := os.Stat(dbFileOnDisk); err == nil {
		log.Printf("Removing existing database file: %s", dbFileOnDisk)
		if err := os.Remove(dbFileOnDisk); err != nil {
			log.Fatalf("Failed to remove old DB: %v", err)
		}
	} else {
		log.Printf("No existing database file to remove: %s", dbFileOnDisk)
	}

	// Create new DB from schema.sql
	cmd := exec.Command("sqlite3", dbFile, fmt.Sprintf(".read %s", schemaFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to create DB from schema.sql: %v", err)
	}

	// Initialize database
	if err := db.Init(dbFile); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Open DB for direct access
	database, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer database.Close()

	// Enable WAL mode and other optimizations for bulk operations
	if _, err := database.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("Warning: Failed to enable WAL mode: %v", err)
	}
	if _, err := database.Exec("PRAGMA synchronous=NORMAL"); err != nil {
		log.Printf("Warning: Failed to set synchronous mode: %v", err)
	}
	if _, err := database.Exec("PRAGMA cache_size=10000"); err != nil {
		log.Printf("Warning: Failed to set cache size: %v", err)
	}
	if _, err := database.Exec("PRAGMA temp_store=MEMORY"); err != nil {
		log.Printf("Warning: Failed to set temp store: %v", err)
	}

	// Import ad-category.json (AdCategory)
	typeData, err := os.ReadFile(typeFile)
	if err != nil {
		log.Fatalf("Failed to read ad-category.json: %v", err)
	}
	type AdCategory struct {
		Name        string `json:"name"`
		VehicleFile string `json:"vehicle_file,omitempty"`
		PartFile    string `json:"part_file,omitempty"`
		AdFile      string `json:"ad_file,omitempty"`
	}
	var adCategories []AdCategory
	if err := json.Unmarshal(typeData, &adCategories); err != nil {
		log.Fatalf("Failed to parse type.json: %v", err)
	}

	// Create map for category lookups
	categoryMap := make(map[string]int)
	for _, cat := range adCategories {
		_, err := database.Exec(`INSERT INTO AdCategory (name) VALUES (?)`, cat.Name)
		if err != nil {
			log.Printf("Failed to insert AdCategory %s: %v", cat.Name, err)
		}
	}
	fmt.Printf("Inserted %d AdCategories\n", len(adCategories))

	// Populate category map
	categoryRows, err := database.Query(`SELECT id, name FROM AdCategory`)
	if err != nil {
		log.Fatalf("Failed to query AdCategory: %v", err)
	}
	defer categoryRows.Close()
	for categoryRows.Next() {
		var id int
		var name string
		if err := categoryRows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan ad_category_id row: %v", err)
		}
		categoryMap[name] = id
	}

	// Import parent.json
	parentData, err := os.ReadFile(parentFile)
	if err != nil {
		log.Fatalf("Failed to read parent.json: %v", err)
	}
	type ParentCompany struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	}
	var parentCompanies []ParentCompany
	if err := json.Unmarshal(parentData, &parentCompanies); err != nil {
		log.Fatalf("Failed to parse parent.json: %v", err)
	}
	for _, pc := range parentCompanies {
		_, err := database.Exec(`INSERT INTO ParentCompany (name, country) VALUES (?, ?)`, pc.Name, pc.Country)
		if err != nil {
			log.Printf("Failed to insert ParentCompany %s: %v", pc.Name, err)
		}
	}
	fmt.Printf("Inserted %d ParentCompanies\n", len(parentCompanies))

	// Import user.json
	userFile := "cmd/rebuild_db/user.json"
	userData, err := os.ReadFile(userFile)
	if err != nil {
		log.Fatalf("Failed to read user.json: %v", err)
	}
	type UserImport struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Phone    string `json:"phone"`
		IsAdmin  bool   `json:"is_admin"`
	}
	var users []UserImport
	if err := json.Unmarshal(userData, &users); err != nil {
		log.Fatalf("Failed to parse user.json: %v", err)
	}
	userCount := 0
	for _, u := range users {
		hash, salt, err := password.HashPassword(u.Password)
		if err != nil {
			log.Printf("Failed to hash password for user %s: %v", u.Name, err)
			continue
		}
		_, err = database.Exec(`INSERT INTO User (name, phone,
			password_hash, password_salt, password_algo, phone_verified,
			verification_code, notification_method, email_address, is_admin)
			VALUES (?, ?, ?, ?, ?, 0, '', 'sms', '', ?)`,
			u.Name, u.Phone, hash, salt, "argon2id", u.IsAdmin)
		if err != nil {
			log.Printf("Failed to insert user %s: %v", u.Name, err)
		} else {
			userCount++
		}
	}
	fmt.Printf("Inserted %d Users\n", userCount)

	// Import make-parent.json
	makeParentData, err := os.ReadFile(makeParentFile)
	if err != nil {
		log.Fatalf("Failed to read make-parent.json: %v", err)
	}
	type MakeParent struct {
		Make          string `json:"make"`
		ParentCompany string `json:"parent_company"`
	}
	var makeParents []MakeParent
	if err := json.Unmarshal(makeParentData, &makeParents); err != nil {
		log.Fatalf("Failed to parse make-parent.json: %v", err)
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
		log.Fatalf("Failed to query ParentCompany: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan ParentCompany row: %v", err)
		}
		parentCompanyMap[name] = id
	}

	// Create category to seed file mapping
	categorySeedMap := make(map[string]struct {
		vehicleFile string
		partFile    string
		adFile      string
	})

	// Process each category from ad-category.json and add to seed map if it has files
	for _, cat := range adCategories {
		if cat.VehicleFile != "" || cat.AdFile != "" {
			seedFiles := struct {
				vehicleFile string
				partFile    string
				adFile      string
			}{
				vehicleFile: "cmd/rebuild_db/" + cat.VehicleFile,
				partFile:    "",
				adFile:      "cmd/rebuild_db/" + cat.AdFile,
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
			log.Printf("Category %s not found, skipping", categoryName)
			continue
		}

		log.Printf("Processing category: %s", categoryName)

		// Import vehicles for this category
		if err := importVehicles(database, seedFiles.vehicleFile, categoryID, makeParentsData, parentCompanyMap); err != nil {
			log.Printf("Failed to import vehicles for %s: %v", categoryName, err)
		}

		// Import parts for this category
		if seedFiles.partFile != "" {
			if err := importParts(database, seedFiles.partFile, categoryID); err != nil {
				log.Printf("Failed to import parts for %s: %v", categoryName, err)
			}
		}

		// Import ads for this category
		if err := importAds(database, seedFiles.adFile, categoryID, categoryMap); err != nil {
			log.Printf("Failed to import ads for %s: %v", categoryName, err)
		}
	}

	fmt.Println("Database rebuild and import complete.")
	fmt.Println("Vector embeddings will be processed by the main application background processor.")
}

// executeVehicleBatch executes a batch of Vehicle insertions using a single INSERT statement
func executeVehicleBatch(tx *sql.Tx, batch []struct {
	adCategoryID, makeID, yearID, modelID, engineID int
}) {
	// Use a single INSERT statement with multiple VALUES for better performance
	if len(batch) == 0 {
		return
	}

	// Build the VALUES clause
	values := make([]string, len(batch))
	args := make([]interface{}, len(batch)*5)

	for i, vehicle := range batch {
		values[i] = "(?, ?, ?, ?, ?)"
		args[i*5] = vehicle.adCategoryID
		args[i*5+1] = vehicle.makeID
		args[i*5+2] = vehicle.yearID
		args[i*5+3] = vehicle.modelID
		args[i*5+4] = vehicle.engineID
	}

	query := fmt.Sprintf("INSERT OR IGNORE INTO Vehicle (ad_category_id, make_id, year_id, model_id, engine_id) VALUES %s", strings.Join(values, ","))
	_, err := tx.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to insert Vehicle batch: %v", err)
	}
}

func getOrInsert(db *sql.DB, table, col, val string) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=?", table, col), val).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES (?)", table, col), val)
		if err != nil {
			log.Fatalf("Failed to insert into %s: %v", table, err)
		}
		id64, _ := res.LastInsertId()
		return int(id64)
	} else if err != nil {
		log.Fatalf("Failed to lookup %s: %v", table, err)
	}
	return id
}

func getOrInsertWithCategory(db *sql.DB, table, col, val string, categoryID int) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=? AND ad_category_id=?", table, col), val, categoryID).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s, ad_category_id) VALUES (?, ?)", table, col), val, categoryID)
		if err != nil {
			log.Fatalf("Failed to insert into %s: %v", table, err)
		}
		id64, _ := res.LastInsertId()
		return int(id64)
	} else if err != nil {
		log.Fatalf("Failed to lookup %s: %v", table, err)
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

// importVehicles loads vehicle data from a JSON file and imports it into the database
func importVehicles(db *sql.DB, jsonFile string, categoryID int, makeParents []struct {
	Make          string
	ParentCompany string
}, parentCompanyMap map[string]int) error {
	f, err := os.Open(jsonFile)
	if err != nil {
		log.Printf("Failed to open %s: %v", jsonFile, err)
		return nil // Skip missing files
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		log.Printf("Failed to read %s: %v", jsonFile, err)
		return err
	}

	var mym MakeYearModel
	if err := json.Unmarshal(data, &mym); err != nil {
		log.Printf("Failed to parse %s: %v", jsonFile, err)
		return err
	}

	// Import vehicles with the category
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

					// Insert vehicle
					db.Exec(`INSERT OR IGNORE INTO Vehicle (ad_category_id, make_id, year_id, model_id, engine_id) VALUES (?, ?, ?, ?, ?)`,
						categoryID, makeID, yearID, modelID, engineID)
				}
			}
		}
	}

	fmt.Printf("Imported vehicles from %s\n", jsonFile)
	return nil
}

// importParts loads part data from a JSON file and imports it into the database
func importParts(db *sql.DB, jsonFile string, categoryID int) error {
	partData, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Printf("Failed to read %s: %v", jsonFile, err)
		return nil // Skip missing files
	}

	var partMap map[string][]string
	if err := json.Unmarshal(partData, &partMap); err != nil {
		log.Printf("Failed to parse %s: %v", jsonFile, err)
		return err
	}

	for cat, subcats := range partMap {
		catID := getOrInsertWithCategory(db, "PartCategory", "name", cat, categoryID)
		for _, subcat := range subcats {
			var subcatID int
			err := db.QueryRow(`SELECT id FROM PartSubCategory WHERE part_category_id=? AND name=?`, catID, subcat).Scan(&subcatID)
			if err == sql.ErrNoRows {
				_, err := db.Exec(`INSERT INTO PartSubCategory (part_category_id, name) VALUES (?, ?)`, catID, subcat)
				if err != nil {
					log.Printf("Failed to insert PartSubCategory: %v", err)
				}
			} else if err != nil {
				log.Printf("PartSubCategory lookup error: %v", err)
			}
		}
	}

	fmt.Printf("Imported parts from %s\n", jsonFile)
	return nil
}

// importAds loads ad data from a JSON file and imports it into the database
func importAds(db *sql.DB, jsonFile string, categoryID int, categoryMap map[string]int) error {
	adData, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Printf("Failed to read %s: %v", jsonFile, err)
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
		log.Printf("Failed to parse %s: %v", jsonFile, err)
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

	rows2, err := db.Query(`SELECT id, name FROM PartSubCategory ps JOIN PartCategory pc ON ps.part_category_id=pc.id WHERE pc.ad_category_id=?`, categoryID)
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
				log.Printf("Failed to insert Location: %v", err)
				continue
			}
			id64, _ := res.LastInsertId()
			locationID = int(id64)
		} else if err != nil {
			log.Printf("Location lookup error: %v", err)
			continue
		}

		// Get subcategory ID
		subcategoryID := 0
		if ad.Subcategory != "" && partCategoryMap[ad.Category] > 0 {
			subcatRows, err := db.Query(`SELECT id FROM PartSubCategory WHERE part_category_id=? AND name=?`, partCategoryMap[ad.Category], ad.Subcategory)
			if err == nil {
				if subcatRows.Next() {
					subcatRows.Scan(&subcategoryID)
				}
				subcatRows.Close()
			}
		}

		// If no subcategory found, use the first available one for this category (only for parts ads)
		if subcategoryID == 0 && len(subcategoryMap) > 0 && ad.Category != "" {
			for _, id := range subcategoryMap {
				subcategoryID = id
				break
			}
		}

		// For non-parts ads (empty category), allow subcategoryID to be 0
		if subcategoryID == 0 && ad.Category != "" {
			log.Printf("No valid subcategory found for ad: %s, skipping", ad.Title)
			continue
		}

		// Generate between 0 and 5 images per ad
		numImages := rand.Intn(6)

		// Insert ad
		res, err := db.Exec(`INSERT INTO Ad (title, description, price, created_at, part_subcategory_id, user_id, location_id, image_count, has_vector, ad_category_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`,
			ad.Title, ad.Description, ad.Price, ad.CreatedAt, subcategoryID, ad.UserID, locationID, numImages, categoryID)
		if err != nil {
			log.Printf("Failed to insert Ad: %v", err)
			continue
		}
		adID, _ := res.LastInsertId()

		// Link to vehicles (only if ad has years/engines fields)
		if len(ad.Years) > 0 && len(ad.Engines) > 0 {
			for _, year := range ad.Years {
				for _, model := range ad.Models {
					for _, engine := range ad.Engines {
						// Query for vehicle
						var vehicleID int
						err := db.QueryRow(`SELECT v.id FROM Vehicle v 
							JOIN Make m ON v.make_id=m.id
							JOIN Year y ON v.year_id=y.id
							JOIN Model mdl ON v.model_id=mdl.id
							JOIN Engine e ON v.engine_id=e.id
							WHERE m.name=? AND y.year=? AND mdl.name=? AND e.name=? AND v.ad_category_id=?`,
							ad.Make, year, model, engine, categoryID).Scan(&vehicleID)
						if err != nil {
							continue
						}

						// Insert AdVehicle relationship
						db.Exec(`INSERT OR IGNORE INTO AdVehicle (ad_id, vehicle_id) VALUES (?, ?)`, adID, vehicleID)
					}
				}
			}
		}

		adCount++
	}

	fmt.Printf("Imported %d ads from %s\n", adCount, jsonFile)
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
			log.Fatalf("Failed to insert into %s: %v", table, err)
		}
		id64, _ := res.LastInsertId()
		return int(id64)
	} else if err != nil {
		log.Fatalf("Failed to lookup %s: %v", table, err)
	}
	return id
}
