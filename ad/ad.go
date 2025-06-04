package ad

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SearchQuery represents a structured query for filtering ads
type SearchQuery struct {
	Make        string   `json:"make,omitempty"`
	Years       []string `json:"years,omitempty"`
	Models      []string `json:"models,omitempty"`
	EngineSizes []string `json:"engine_sizes,omitempty"`
	Category    string   `json:"category,omitempty"`
	SubCategory string   `json:"sub_category,omitempty"`
}

// SearchCursor represents a point in the search results for pagination
type SearchCursor struct {
	Query      SearchQuery `json:"q"`           // The structured query
	LastID     int         `json:"last_id"`     // Last ID seen
	LastPosted time.Time   `json:"last_posted"` // Timestamp of last item
}

type Ad struct {
	ID          int       `json:"id"`
	Make        string    `json:"make"`
	Years       []string  `json:"years"`
	Models      []string  `json:"models"`
	Engines     []string  `json:"engines"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	CreatedAt   time.Time `json:"created_at"`
}

var db *sql.DB

func InitDB(path string) error {
	var err error
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}

	// Create the ads table with the schema
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ads (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		make TEXT,
		years TEXT,
		models TEXT,
		engines TEXT,
		description TEXT,
		price REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	return err
}

func GetAllAds() map[int]Ad {
	rows, err := db.Query("SELECT id, make, years, models, engines, description, price, created_at FROM ads")
	if err != nil {
		return map[int]Ad{}
	}
	defer rows.Close()
	ads := make(map[int]Ad)
	for rows.Next() {
		var ad Ad
		var years, models, engines string
		if err := rows.Scan(&ad.ID, &ad.Make, &years, &models, &engines, &ad.Description, &ad.Price, &ad.CreatedAt); err != nil {
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
	row := db.QueryRow("SELECT id, make, years, models, engines, description, price, created_at FROM ads WHERE id = ?", id)
	var ad Ad
	var years, models, engines string
	if err := row.Scan(&ad.ID, &ad.Make, &years, &models, &engines, &ad.Description, &ad.Price, &ad.CreatedAt); err != nil {
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
	res, err := db.Exec("INSERT INTO ads (make, years, models, engines, description, price, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", ad.Make, string(years), string(models), string(engines), ad.Description, ad.Price, time.Now())
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

// GetAdsPage returns a slice of ads for cursor-based pagination
func GetAdsPage(cursorID int, limit int) ([]Ad, bool) {
	query := "SELECT id, make, years, models, engines, description, price, created_at FROM ads"
	args := []interface{}{}
	if cursorID > 0 {
		query += " WHERE id < ?"
		args = append(args, cursorID)
	}
	query += " ORDER BY id DESC LIMIT ?"
	args = append(args, limit+1) // Fetch one extra to check for more

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Printf("Error querying ads: %v\n", err)
		return nil, false
	}
	defer rows.Close()

	ads := []Ad{}
	for rows.Next() {
		var ad Ad
		var years, models, engines string
		if err := rows.Scan(&ad.ID, &ad.Make, &years, &models, &engines, &ad.Description, &ad.Price, &ad.CreatedAt); err != nil {
			fmt.Printf("Error scanning ad: %v\n", err)
			continue
		}
		json.Unmarshal([]byte(years), &ad.Years)
		json.Unmarshal([]byte(models), &ad.Models)
		json.Unmarshal([]byte(engines), &ad.Engines)
		ads = append(ads, ad)
	}

	hasMore := false
	if len(ads) > limit {
		hasMore = true
		ads = ads[:limit]
	}
	return ads, hasMore
}

// GetFilteredAdsPage returns a slice of filtered ads for cursor-based pagination
// filtered: the filtered ads, already sorted by CreatedAt DESC, ID DESC
// cursorID, cursorCreatedAt: the last seen ad's ID and CreatedAt
// limit: number of ads to return
// Returns: page of ads, hasMore
func GetFilteredAdsPage(filtered []Ad, cursorID int, cursorCreatedAt time.Time, limit int) ([]Ad, bool) {
	start := 0
	if !cursorCreatedAt.IsZero() || cursorID > 0 {
		for i, ad := range filtered {
			if ad.CreatedAt.Before(cursorCreatedAt) ||
				(ad.CreatedAt.Equal(cursorCreatedAt) && ad.ID < cursorID) {
				start = i
				break
			}
		}
	}
	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}
	page := filtered[start:end]
	hasMore := end < len(filtered)
	return page, hasMore
}

// GetFilteredAdsPageDB returns a page of filtered ads directly from the database
func GetFilteredAdsPageDB(query SearchQuery, cursor *SearchCursor, limit int) ([]Ad, bool, error) {
	sqlQuery := "SELECT id, make, years, models, engines, description, price, created_at FROM ads WHERE 1=1"
	args := []interface{}{}

	// Apply filters
	if query.Make != "" {
		sqlQuery += " AND LOWER(make) = LOWER(?)"
		args = append(args, query.Make)
	}

	// Apply cursor pagination
	if cursor != nil {
		sqlQuery += " AND (created_at < ? OR (created_at = ? AND id < ?))"
		args = append(args, cursor.LastPosted, cursor.LastPosted, cursor.LastID)
	}

	// Order by created_at DESC, id DESC
	sqlQuery += " ORDER BY created_at DESC, id DESC LIMIT ?"
	args = append(args, limit+1) // Get one extra to check if there are more results

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	ads := []Ad{}
	for rows.Next() {
		var ad Ad
		var years, models, engines string
		if err := rows.Scan(&ad.ID, &ad.Make, &years, &models, &engines, &ad.Description, &ad.Price, &ad.CreatedAt); err != nil {
			continue
		}
		json.Unmarshal([]byte(years), &ad.Years)
		json.Unmarshal([]byte(models), &ad.Models)
		json.Unmarshal([]byte(engines), &ad.Engines)

		// Apply array filters in memory since SQLite doesn't handle array operations well
		if len(query.Years) > 0 || len(query.Models) > 0 || len(query.EngineSizes) > 0 {
			if len(query.Years) > 0 && !anyStringInSlice(ad.Years, query.Years) {
				continue
			}
			if len(query.Models) > 0 && !anyStringInSlice(ad.Models, query.Models) {
				continue
			}
			if len(query.EngineSizes) > 0 && !anyStringInSlice(ad.Engines, query.EngineSizes) {
				continue
			}
		}

		ads = append(ads, ad)
	}

	hasMore := false
	if len(ads) > limit {
		hasMore = true
		ads = ads[:limit]
	}

	return ads, hasMore, nil
}

// Helper function for case-insensitive string slice comparison
func anyStringInSlice(a, b []string) bool {
	for _, s := range a {
		for _, t := range b {
			if strings.EqualFold(s, t) {
				return true
			}
		}
	}
	return false
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
