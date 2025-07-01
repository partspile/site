package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type MakeYearModel map[string]map[string]map[string][]string

func main() {
	jsonFile := "make-year-model.json"
	dbFile := "project.db"

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
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

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
	fmt.Println("Import complete.")
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
