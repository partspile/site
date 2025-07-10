package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type MakeYearModel map[string]map[string]map[string][]string

func main() {
	jsonFile := "cmd/rebuild_db/make-year-model.json"
	partFile := "cmd/rebuild_db/part.json"
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
		makeID := getOrInsert(db, "Make", "name", make)
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

	fmt.Println("Database rebuild and import complete.")
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
