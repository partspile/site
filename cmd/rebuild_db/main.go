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
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
	"golang.org/x/crypto/bcrypt"
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
	dbFile := "project.db"
	schemaFile := "schema.sql"

	// Remove old DB if exists
	if _, err := os.Stat(dbFile); err == nil {
		if err := os.Remove(dbFile); err != nil {
			log.Fatalf("Failed to remove old DB: %v", err)
		}
	}

	// Create new DB from schema.sql
	cmd := exec.Command("sqlite3", dbFile, fmt.Sprintf(".read %s", schemaFile))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to create DB from schema.sql: %v", err)
	}

	// Open DB
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	// Initialize packages that need the DB
	if err := ad.InitDB(dbFile); err != nil {
		log.Fatalf("Failed to initialize ad package: %v", err)
	}
	vehicle.InitDB(ad.DB)

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
		_, err := db.Exec(`INSERT INTO ParentCompany (name, country) VALUES (?, ?)`, pc.Name, pc.Country)
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
	rows, err := db.Query(`SELECT id, name FROM ParentCompany`)
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

	// Import make-year-model.json
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
			makeID = getOrInsertWithParent(db, "Make", "name", make, *parentCompanyID)
		} else {
			makeID = getOrInsert(db, "Make", "name", make)
		}
		for year, models := range years {
			yearID := getOrInsert(db, "Year", "year", year)
			for model, engines := range models {
				modelID := getOrInsert(db, "Model", "name", model)
				for _, engine := range engines {
					engineID := getOrInsert(db, "Engine", "name", engine)
					// Insert Car if not exists
					var carID int
					err := db.QueryRow(`SELECT id FROM Car WHERE make_id=? AND year_id=? AND model_id=? AND engine_id=?`, makeID, yearID, modelID, engineID).Scan(&carID)
					if err == sql.ErrNoRows {
						res, err := db.Exec(`INSERT INTO Car (make_id, year_id, model_id, engine_id) VALUES (?, ?, ?, ?)`, makeID, yearID, modelID, engineID)
						if err != nil {
							log.Printf("Failed to insert Car: %v", err)
							continue
						}
						id, _ := res.LastInsertId()
						carID = int(id)
						fmt.Printf("Inserted Car: %s %s %s %s\n", make, year, model, engine)
					} else if err == nil {
						// Already exists
					} else {
						log.Printf("Car lookup error: %v", err)
					}
				}
			}
		}
	}

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
		catID := getOrInsert(db, "PartCategory", "name", cat)
		for _, subcat := range subcats {
			// Insert subcategory if not exists
			var subcatID int
			err := db.QueryRow(`SELECT id FROM PartSubCategory WHERE category_id=? AND name=?`, catID, subcat).Scan(&subcatID)
			if err == sql.ErrNoRows {
				_, err := db.Exec(`INSERT INTO PartSubCategory (category_id, name) VALUES (?, ?)`, catID, subcat)
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
	}
	var users []UserImport
	if err := json.Unmarshal(userData, &users); err != nil {
		log.Fatalf("Failed to parse user.json: %v", err)
	}
	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Failed to hash password for user %s: %v", u.Name, err)
			continue
		}
		_, err = db.Exec(`INSERT INTO User (name, phone, password_hash) VALUES (?, ?, ?)`, u.Name, u.Phone, string(hash))
		if err != nil {
			log.Printf("Failed to insert user %s: %v", u.Name, err)
		} else {
			fmt.Printf("Inserted user: %s\n", u.Name)
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
			City      string `json:"city"`
			AdminArea string `json:"admin_area"`
			Country   string `json:"country"`
		} `json:"location"`
	}
	var ads []AdImport
	if err := json.Unmarshal(adData, &ads); err != nil {
		log.Fatalf("Failed to parse ad.json: %v", err)
	}

	// Create maps for efficient lookups
	makeMap := make(map[string]int)
	yearMap := make(map[string]int)
	modelMap := make(map[string]int)
	engineMap := make(map[string]int)
	userMap := make(map[int]int)
	categoryMap := make(map[string]int)
	subcategoryMap := make(map[string]int)

	// Declare variables for database queries
	var makeRows, yearRows, modelRows, engineRows, userRows, categoryRows, subcategoryRows *sql.Rows

	// Populate maps
	makeRows, err = db.Query(`SELECT id, name FROM Make`)
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

	yearRows, err = db.Query(`SELECT id, year FROM Year`)
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

	modelRows, err = db.Query(`SELECT id, name FROM Model`)
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

	engineRows, err = db.Query(`SELECT id, name FROM Engine`)
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

	userRows, err = db.Query(`SELECT id FROM User`)
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

	categoryRows, err = db.Query(`SELECT id, name FROM PartCategory`)
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

	subcategoryRows, err = db.Query(`SELECT id, name FROM PartSubCategory`)
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
		err := db.QueryRow(`SELECT id FROM Location WHERE raw_text=?`, locationKey).Scan(&locationID)
		if err == sql.ErrNoRows {
			res, err := db.Exec(`INSERT INTO Location (raw_text, city, admin_area, country) VALUES (?, ?, ?, ?)`,
				locationKey, ad.Location.City, ad.Location.AdminArea, ad.Location.Country)
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

		// Insert ad with image_order
		res, err := db.Exec(`INSERT INTO Ad (title, description, price, created_at, subcategory_id, user_id, location_id, image_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			ad.Title, ad.Description, ad.Price, ad.CreatedAt, subcategoryID, ad.UserID, locationID, string(imageOrderJSON))
		if err != nil {
			log.Printf("Failed to insert Ad: %v", err)
			continue
		}
		adID, _ := res.LastInsertId()

		// Generate and upload images for this ad
		if err := uploadAdImagesToB2(int(adID), numImages, ad.Title); err != nil {
			log.Printf("Failed to upload images for ad %d: %v", adID, err)
		}

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

					err := db.QueryRow(`SELECT id FROM Car WHERE make_id=? AND year_id=? AND model_id=? AND engine_id=?`,
						makeID, yearID, modelID, engineID).Scan(&carID)
					if err == sql.ErrNoRows {
						// Car doesn't exist, skip this combination
						continue
					} else if err != nil {
						log.Printf("Car lookup error: %v", err)
						continue
					}

					// Insert AdCar relationship
					_, err = db.Exec(`INSERT INTO AdCar (ad_id, car_id) VALUES (?, ?)`, adID, carID)
					if err != nil {
						log.Printf("Failed to insert AdCar: %v", err)
					}
				}
			}
		}

		fmt.Printf("Inserted ad: %s\n", ad.Title)
	}

	fmt.Println("Database rebuild and import complete.")

	// Initialize vector embedding services
	fmt.Println("Initializing vector embedding services...")
	if err := vector.InitGeminiClient(""); err != nil {
		log.Printf("Failed to init Gemini: %v", err)
	} else {
		if err := vector.InitPineconeClient("", ""); err != nil {
			log.Printf("Failed to init Pinecone: %v", err)
		} else {
			// Generate embeddings for all ads
			fmt.Println("Generating embeddings for all ads...")
			ads, err := ad.GetAllAds()
			if err != nil {
				log.Printf("Failed to get ads for embedding: %v", err)
			} else {
				fmt.Printf("Found %d ads to generate embeddings for\n", len(ads))
				failures := 0
				for i, adObj := range ads {
					prompt := buildAdEmbeddingPrompt(adObj)
					log.Printf("[embedding] Generating embedding for ad %d", adObj.ID)
					embedding, err := vector.EmbedText(prompt)
					if err != nil {
						log.Printf("[embedding] failed for ad %d: %v", adObj.ID, err)
						failures++
						continue
					}
					meta := buildAdEmbeddingMetadata(adObj)
					err = vector.UpsertAdEmbedding(adObj.ID, embedding, meta)
					if err != nil {
						log.Printf("[pinecone] upsert failed for ad %d: %v", adObj.ID, err)
						failures++
						continue
					}
					if (i+1)%10 == 0 || i == len(ads)-1 {
						fmt.Printf("%d/%d ads processed for embeddings\n", i+1, len(ads))
					}
					// Sleep to avoid rate limits
					time.Sleep(100 * time.Millisecond)
				}
				fmt.Printf("Embedding generation complete. %d ads processed, %d failures.\n", len(ads), failures)
			}
		}
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

// buildAdEmbeddingPrompt creates a prompt for generating embeddings
func buildAdEmbeddingPrompt(adObj ad.Ad) string {
	// Get parent company information for the make
	var parentCompanyStr, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := vehicle.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
			parentCompanyStr = pcInfo.Name
			parentCompanyCountry = pcInfo.Country
		}
	}

	return fmt.Sprintf(`Encode the following ad for semantic search. Focus on what the part is, what vehicles it fits, and any relevant details for a buyer. Return only the embedding vector.\n\nTitle: %s\nDescription: %s\nMake: %s\nParent Company: %s\nParent Company Country: %s\nYears: %s\nModels: %s\nEngines: %s\nCategory: %s\nSubCategory: %s\nLocation: %s, %s, %s`,
		adObj.Title,
		adObj.Description,
		adObj.Make,
		parentCompanyStr,
		parentCompanyCountry,
		joinStrings(adObj.Years),
		joinStrings(adObj.Models),
		joinStrings(adObj.Engines),
		adObj.Category,
		adObj.SubCategory,
		adObj.City,
		adObj.AdminArea,
		adObj.Country,
	)
}

// buildAdEmbeddingMetadata creates metadata for embeddings
func buildAdEmbeddingMetadata(adObj ad.Ad) map[string]interface{} {
	// Get parent company information for the make
	var parentCompanyName, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := vehicle.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
			parentCompanyName = pcInfo.Name
			parentCompanyCountry = pcInfo.Country
		}
	}

	return map[string]interface{}{
		"ad_id":                  adObj.ID,
		"created_at":             adObj.CreatedAt.Format(time.RFC3339),
		"click_count":            adObj.ClickCount,
		"make":                   adObj.Make,
		"parent_company":         parentCompanyName,
		"parent_company_country": parentCompanyCountry,
		"years":                  interfaceSlice(adObj.Years),
		"models":                 interfaceSlice(adObj.Models),
		"engines":                interfaceSlice(adObj.Engines),
		"category":               adObj.Category,
		"subcategory":            adObj.SubCategory,
		"city":                   adObj.City,
		"admin_area":             adObj.AdminArea,
		"country":                adObj.Country,
	}
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
