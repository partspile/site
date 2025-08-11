package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/password"
)

type MakeYearModel map[string]map[string]map[string][]string

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	jsonFile := "cmd/rebuild_db/make-year-model.json"
	partFile := "cmd/rebuild_db/part.json"
	parentFile := "cmd/rebuild_db/parent.json"
	makeParentFile := "cmd/rebuild_db/make-parent.json"
	adFile := "cmd/rebuild_db/ad.json"
	dbFile := config.DatabaseURL
	schemaFile := "schema.sql"

	dbFileOnDisk := dbFile
	if strings.HasPrefix(dbFileOnDisk, "file:") {
		dbFileOnDisk = strings.TrimPrefix(dbFileOnDisk, "file:")
	}
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

	// Import parent.json
	parentData, err := ioutil.ReadFile(parentFile)
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
		} else {
			fmt.Printf("Inserted ParentCompany: %s (%s)\n", pc.Name, pc.Country)
		}
	}

	// Import make-parent.json
	makeParentData, err := ioutil.ReadFile(makeParentFile)
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

	// Import make-year-model.json with optimized Car insertion
	f, err := os.Open(jsonFile)
	if err != nil {
		log.Fatalf("Failed to open JSON: %v", err)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("Failed to read JSON: %v", err)
	}
	var mym MakeYearModel
	if err := json.Unmarshal(data, &mym); err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}

	// Create maps for efficient lookups
	makeMap := make(map[string]int)
	yearMap := make(map[string]int)
	modelMap := make(map[string]int)
	engineMap := make(map[string]int)

	// Pre-populate all makes, years, models, and engines
	fmt.Println("Pre-populating lookup maps...")

	// Get all makes
	makeRows, err := database.Query(`SELECT id, name FROM Make`)
	if err != nil {
		log.Fatalf("Failed to query Make: %v", err)
	}
	defer makeRows.Close()
	for makeRows.Next() {
		var id int
		var name string
		if err := makeRows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan Make row: %v", err)
		}
		makeMap[name] = id
	}

	// Get all years
	yearRows, err := database.Query(`SELECT id, year FROM Year`)
	if err != nil {
		log.Fatalf("Failed to query Year: %v", err)
	}
	defer yearRows.Close()
	for yearRows.Next() {
		var id int
		var year string
		if err := yearRows.Scan(&id, &year); err != nil {
			log.Fatalf("Failed to scan Year row: %v", err)
		}
		yearMap[year] = id
	}

	// Get all models
	modelRows, err := database.Query(`SELECT id, name FROM Model`)
	if err != nil {
		log.Fatalf("Failed to query Model: %v", err)
	}
	defer modelRows.Close()
	for modelRows.Next() {
		var id int
		var name string
		if err := modelRows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan Model row: %v", err)
		}
		modelMap[name] = id
	}

	// Get all engines
	engineRows, err := database.Query(`SELECT id, name FROM Engine`)
	if err != nil {
		log.Fatalf("Failed to query Engine: %v", err)
	}
	defer engineRows.Close()
	for engineRows.Next() {
		var id int
		var name string
		if err := engineRows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan Engine row: %v", err)
		}
		engineMap[name] = id
	}

	// First pass: Insert all makes, years, models, and engines
	fmt.Println("Inserting makes, years, models, and engines...")
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

		// Insert make with parent company relationship
		var makeID int
		if parentCompanyID != nil {
			makeID = getOrInsertWithParent(database, "Make", "name", make, *parentCompanyID)
		} else {
			makeID = getOrInsert(database, "Make", "name", make)
		}
		makeMap[make] = makeID

		for year, models := range years {
			yearID := getOrInsert(database, "Year", "year", year)
			yearMap[year] = yearID

			for model, engines := range models {
				modelID := getOrInsert(database, "Model", "name", model)
				modelMap[model] = modelID

				for _, engine := range engines {
					engineID := getOrInsert(database, "Engine", "name", engine)
					engineMap[engine] = engineID
				}
			}
		}
	}

	// Second pass: Insert all cars in batches
	fmt.Println("Inserting cars in optimized batches...")
	carCount := 0
	batchSize := 1000
	carBatch := make([]struct {
		makeID, yearID, modelID, engineID int
	}, 0, batchSize)

	// Start transaction for bulk Car operations
	tx, err := database.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	for make, years := range mym {
		makeID := makeMap[make]
		for year, models := range years {
			yearID := yearMap[year]
			for model, engines := range models {
				modelID := modelMap[model]
				for _, engine := range engines {
					engineID := engineMap[engine]

					// Add to batch for bulk insertion
					carBatch = append(carBatch, struct {
						makeID, yearID, modelID, engineID int
					}{makeID, yearID, modelID, engineID})
					carCount++

					// Execute batch when it reaches the batch size
					if len(carBatch) >= batchSize {
						executeCarBatch(tx, carBatch)
						carBatch = carBatch[:0] // Reset slice while keeping capacity
					}
				}
			}
		}
	}

	// Execute remaining cars in the batch
	if len(carBatch) > 0 {
		executeCarBatch(tx, carBatch)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		log.Fatalf("Failed to commit Car transaction: %v", err)
	}

	fmt.Printf("Processed %d cars in optimized batches\n", carCount)

	// Import part.json
	partData, err := ioutil.ReadFile(partFile)
	if err != nil {
		log.Fatalf("Failed to read part.json: %v", err)
	}
	var partMap map[string][]string
	if err := json.Unmarshal(partData, &partMap); err != nil {
		log.Fatalf("Failed to parse part.json: %v", err)
	}
	for cat, subcats := range partMap {
		catID := getOrInsert(database, "PartCategory", "name", cat)
		for _, subcat := range subcats {
			// Insert subcategory if not exists
			var subcatID int
			err := database.QueryRow(`SELECT id FROM PartSubCategory WHERE category_id=? AND name=?`, catID, subcat).Scan(&subcatID)
			if err == sql.ErrNoRows {
				_, err := database.Exec(`INSERT INTO PartSubCategory (category_id, name) VALUES (?, ?)`, catID, subcat)
				if err != nil {
					log.Printf("Failed to insert PartSubCategory: %v", err)
				}
			} else if err != nil {
				log.Printf("PartSubCategory lookup error: %v", err)
			}
		}
	}

	// Import user.json
	userFile := "cmd/rebuild_db/user.json"
	userData, err := ioutil.ReadFile(userFile)
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
	for _, u := range users {
		hash, salt, err := password.HashPassword(u.Password)
		if err != nil {
			log.Printf("Failed to hash password for user %s: %v", u.Name, err)
			continue
		}
		_, err = database.Exec(`INSERT INTO User (name, phone, password_hash, password_salt, password_algo, is_admin) VALUES (?, ?, ?, ?, ?, ?)`, u.Name, u.Phone, hash, salt, "argon2id", u.IsAdmin)
		if err != nil {
			log.Printf("Failed to insert user %s: %v", u.Name, err)
		} else {
			if u.IsAdmin {
				fmt.Printf("Inserted admin user: %s\n", u.Name)
			} else {
				fmt.Printf("Inserted user: %s\n", u.Name)
			}
		}
	}

	// Import ad.json
	adData, err := ioutil.ReadFile(adFile)
	if err != nil {
		log.Fatalf("Failed to read ad.json: %v", err)
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
		log.Fatalf("Failed to parse ad.json: %v", err)
	}

	// Create maps for efficient lookups
	userMap := make(map[int]int)
	categoryMap := make(map[string]int)
	subcategoryMap := make(map[string]int)

	// Declare variables for database queries
	var userRows, categoryRows, subcategoryRows *sql.Rows

	// Populate maps
	userRows, err = database.Query(`SELECT id FROM User`)
	if err != nil {
		log.Fatalf("Failed to query User: %v", err)
	}
	defer userRows.Close()
	for userRows.Next() {
		var id int
		if err := userRows.Scan(&id); err != nil {
			log.Fatalf("Failed to scan User row: %v", err)
		}
		userMap[id] = id
	}

	categoryRows, err = database.Query(`SELECT id, name FROM PartCategory`)
	if err != nil {
		log.Fatalf("Failed to query PartCategory: %v", err)
	}
	defer categoryRows.Close()
	for categoryRows.Next() {
		var id int
		var name string
		if err := categoryRows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan PartCategory row: %v", err)
		}
		categoryMap[name] = id
	}

	subcategoryRows, err = database.Query(`SELECT id, name FROM PartSubCategory`)
	if err != nil {
		log.Fatalf("Failed to query PartSubCategory: %v", err)
	}
	defer subcategoryRows.Close()
	for subcategoryRows.Next() {
		var id int
		var name string
		if err := subcategoryRows.Scan(&id, &name); err != nil {
			log.Fatalf("Failed to scan PartSubCategory row: %v", err)
		}
		subcategoryMap[name] = id
	}

	// Process ads
	for _, ad := range ads {
		// Insert or get location
		locationKey := fmt.Sprintf("%s, %s, %s", ad.Location.City, ad.Location.AdminArea, ad.Location.Country)
		var locationID int
		err := database.QueryRow(`SELECT id FROM Location WHERE raw_text=?`, locationKey).Scan(&locationID)
		if err == sql.ErrNoRows {
			res, err := database.Exec(`INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)`,
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

		// Get subcategory ID if specified
		var subcategoryID *int
		if ad.Subcategory != "" {
			if id, exists := subcategoryMap[ad.Subcategory]; exists {
				subcategoryID = &id
			}
		}

		// Generate between 1 and 5 images per ad
		numImages := 1 + rand.Intn(5) // 1 to 5 images

		// Create image_order as JSON array of integers starting from 1
		imageOrderIndices := make([]int, numImages)
		for i := 0; i < numImages; i++ {
			imageOrderIndices[i] = i + 1
		}
		imageOrderJSON, err := json.Marshal(imageOrderIndices)
		if err != nil {
			log.Printf("Failed to marshal image_order for ad: %v", err)
			continue
		}

		// Insert ad with image_order and has_vector (initially 0)
		res, err := database.Exec(`INSERT INTO Ad (title, description, price, created_at, subcategory_id, user_id, location_id, image_order, has_vector) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0)`,
			ad.Title, ad.Description, ad.Price, ad.CreatedAt, subcategoryID, ad.UserID, locationID, string(imageOrderJSON))
		if err != nil {
			log.Printf("Failed to insert Ad: %v", err)
			continue
		}
		adID, _ := res.LastInsertId()

		// Generate and upload images for this ad
		// Temporarily disabled for database rebuild
		// if err := uploadAdImagesToB2(int(adID), numImages, ad.Title); err != nil {
		// 	log.Printf("Failed to upload images for ad %d: %v", adID, err)
		// }

		// Create AdCar relationships for all combinations
		for _, year := range ad.Years {
			for _, model := range ad.Models {
				for _, engine := range ad.Engines {
					// Get car ID
					var carID int
					makeID := makeMap[ad.Make]
					yearID := yearMap[year]
					modelID := modelMap[model]
					engineID := engineMap[engine]

					err := database.QueryRow(`SELECT id FROM Car WHERE make_id=? AND year_id=? AND model_id=? AND engine_id=?`,
						makeID, yearID, modelID, engineID).Scan(&carID)
					if err == sql.ErrNoRows {
						// Car doesn't exist, skip this combination
						continue
					} else if err != nil {
						log.Printf("Car lookup error: %v", err)
						continue
					}

					// Insert AdCar relationship
					_, err = database.Exec(`INSERT INTO AdCar (ad_id, car_id) VALUES (?, ?)`, adID, carID)
					if err != nil {
						log.Printf("Failed to insert AdCar: %v", err)
					}
				}
			}
		}

		fmt.Printf("Inserted ad: %s\n", ad.Title)
	}

	fmt.Println("Database rebuild and import complete.")
	fmt.Println("Vector embeddings will be processed by the main application background processor.")
}

// executeCarBatch executes a batch of Car insertions using a single INSERT statement
func executeCarBatch(tx *sql.Tx, batch []struct {
	makeID, yearID, modelID, engineID int
}) {
	// Use a single INSERT statement with multiple VALUES for better performance
	if len(batch) == 0 {
		return
	}

	// Build the VALUES clause
	values := make([]string, len(batch))
	args := make([]interface{}, len(batch)*4)

	for i, car := range batch {
		values[i] = "(?, ?, ?, ?)"
		args[i*4] = car.makeID
		args[i*4+1] = car.yearID
		args[i*4+2] = car.modelID
		args[i*4+3] = car.engineID
	}

	query := fmt.Sprintf("INSERT OR IGNORE INTO Car (make_id, year_id, model_id, engine_id) VALUES %s", strings.Join(values, ","))
	_, err := tx.Exec(query, args...)
	if err != nil {
		log.Printf("Failed to insert Car batch: %v", err)
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

func getOrInsertWithParent(db *sql.DB, table, col, val string, parentID int) int {
	var id int
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE %s=?", table, col), val).Scan(&id)
	if err == sql.ErrNoRows {
		res, err := db.Exec(fmt.Sprintf("INSERT INTO %s (%s, parent_company_id) VALUES (?, ?)", table, col), val, parentID)
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
