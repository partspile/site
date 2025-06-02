package ad

import (
	"database/sql"
	"encoding/json"

	_ "github.com/mattn/go-sqlite3"
)

type Ad struct {
	ID          int      `json:"id"`
	Make        string   `json:"make"`
	Years       []string `json:"years"`
	Models      []string `json:"models"`
	Engines     []string `json:"engines"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
}

var db *sql.DB

func InitDB(path string) error {
	var err error
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		make TEXT,
		years TEXT,
		models TEXT,
		engines TEXT,
		description TEXT,
		price REAL
	)`)
	return err
}

func GetAllAds() map[int]Ad {
	rows, err := db.Query("SELECT id, make, years, models, engines, description, price FROM ads")
	if err != nil {
		return map[int]Ad{}
	}
	defer rows.Close()
	ads := make(map[int]Ad)
	for rows.Next() {
		var ad Ad
		var years, models, engines string
		if err := rows.Scan(&ad.ID, &ad.Make, &years, &models, &engines, &ad.Description, &ad.Price); err != nil {
			continue
		}
		json.Unmarshal([]byte(years), &ad.Years)
		json.Unmarshal([]byte(models), &ad.Models)
		json.Unmarshal([]byte(engines), &ad.Engines)
		ads[ad.ID] = ad
	}
	return ads
}

func GetAd(id int) (Ad, bool) {
	row := db.QueryRow("SELECT id, make, years, models, engines, description, price FROM ads WHERE id = ?", id)
	var ad Ad
	var years, models, engines string
	if err := row.Scan(&ad.ID, &ad.Make, &years, &models, &engines, &ad.Description, &ad.Price); err != nil {
		return Ad{}, false
	}
	json.Unmarshal([]byte(years), &ad.Years)
	json.Unmarshal([]byte(models), &ad.Models)
	json.Unmarshal([]byte(engines), &ad.Engines)
	return ad, true
}

func AddAd(ad Ad) int {
	years, _ := json.Marshal(ad.Years)
	models, _ := json.Marshal(ad.Models)
	engines, _ := json.Marshal(ad.Engines)
	res, err := db.Exec("INSERT INTO ads (make, years, models, engines, description, price) VALUES (?, ?, ?, ?, ?, ?)", ad.Make, string(years), string(models), string(engines), ad.Description, ad.Price)
	if err != nil {
		return 0
	}
	id, _ := res.LastInsertId()
	return int(id)
}

func UpdateAd(id int, ad Ad) bool {
	years, _ := json.Marshal(ad.Years)
	models, _ := json.Marshal(ad.Models)
	engines, _ := json.Marshal(ad.Engines)
	_, err := db.Exec("UPDATE ads SET make=?, years=?, models=?, engines=?, description=?, price=? WHERE id=?", ad.Make, string(years), string(models), string(engines), ad.Description, ad.Price, id)
	return err == nil
}

func DeleteAd(id int) bool {
	_, err := db.Exec("DELETE FROM ads WHERE id=?", id)
	return err == nil
}

func GetNextAdID() int {
	row := db.QueryRow("SELECT seq FROM sqlite_sequence WHERE name='ads'")
	var seq int
	if err := row.Scan(&seq); err != nil {
		return 1
	}
	return seq + 1
}
