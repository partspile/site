package ad

import (
	"database/sql"
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
	ID            int       `json:"id"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	CreatedAt     time.Time `json:"created_at"`
	SubCategoryID *int      `json:"subcategory_id,omitempty"`
	// Runtime fields populated via joins
	Make        string   `json:"make"`
	Years       []string `json:"years"`
	Models      []string `json:"models"`
	Engines     []string `json:"engines"`
	SubCategory string   `json:"subcategory,omitempty"`
}

var db *sql.DB

// Exported for use by other packages
var DB *sql.DB

func InitDB(path string) error {
	var err error
	db, err = sql.Open("sqlite3", path)
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func GetAllAds() map[int]Ad {
	ads := make(map[int]Ad)

	// Get basic ad data
	rows, err := db.Query(`
		SELECT a.id, a.description, a.price, a.created_at, a.subcategory_id,
		       psc.name as subcategory
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
	`)
	if err != nil {
		return ads
	}
	defer rows.Close()

	for rows.Next() {
		var ad Ad
		var subcategory sql.NullString
		if err := rows.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt,
			&ad.SubCategoryID, &subcategory); err != nil {
			continue
		}
		if subcategory.Valid {
			ad.SubCategory = subcategory.String
		}
		ads[ad.ID] = ad
	}

	// Get vehicle data for each ad
	for id, ad := range ads {
		ad.Make, ad.Years, ad.Models, ad.Engines = getAdVehicleData(id)
		ads[id] = ad
	}

	return ads
}

// getAdVehicleData retrieves vehicle information for an ad
func getAdVehicleData(adID int) (makeName string, years []string, models []string, engines []string) {
	rows, err := db.Query(`
		SELECT DISTINCT m.name, y.year, mo.name, e.name
		FROM AdCar ac
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE ac.ad_id = ?
		ORDER BY m.name, y.year, mo.name, e.name
	`, adID)
	if err != nil {
		return "", nil, nil, nil
	}
	defer rows.Close()

	makeSet := make(map[string]bool)
	yearSet := make(map[string]bool)
	modelSet := make(map[string]bool)
	engineSet := make(map[string]bool)

	for rows.Next() {
		var makeName, modelName, engineName string
		var year int
		if err := rows.Scan(&makeName, &year, &modelName, &engineName); err != nil {
			continue
		}
		makeSet[makeName] = true
		yearSet[fmt.Sprintf("%d", year)] = true
		modelSet[modelName] = true
		engineSet[engineName] = true
	}

	// Convert sets to slices
	for m := range makeSet {
		makeName = m // Assuming single make per ad
		break
	}
	for y := range yearSet {
		years = append(years, y)
	}
	for m := range modelSet {
		models = append(models, m)
	}
	for e := range engineSet {
		engines = append(engines, e)
	}

	return makeName, years, models, engines
}

func GetAd(id int) (Ad, bool) {
	row := db.QueryRow(`
		SELECT a.id, a.description, a.price, a.created_at, a.subcategory_id,
		       psc.name as subcategory
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		WHERE a.id = ?
	`, id)

	var ad Ad
	var subcategory sql.NullString
	if err := row.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt,
		&ad.SubCategoryID, &subcategory); err != nil {
		return Ad{}, false
	}

	if subcategory.Valid {
		ad.SubCategory = subcategory.String
	}

	// Get vehicle data
	ad.Make, ad.Years, ad.Models, ad.Engines = getAdVehicleData(id)

	return ad, true
}

func AddAd(ad Ad) int {
	tx, err := db.Begin()
	if err != nil {
		return 0
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO Ad (description, price, created_at, subcategory_id) VALUES (?, ?, ?, ?)",
		ad.Description, ad.Price, time.Now(), ad.SubCategoryID)
	if err != nil {
		return 0
	}
	adID, _ := res.LastInsertId()

	if ad.Make != "" || len(ad.Years) > 0 || len(ad.Models) > 0 || len(ad.Engines) > 0 {
		if err := addAdVehicleAssociations(tx, int(adID), ad.Make, ad.Years, ad.Models, ad.Engines); err != nil {
			return 0
		}
	}

	if err := tx.Commit(); err != nil {
		return 0
	}

	return int(adID)
}

// addAdVehicleAssociations creates the normalized vehicle associations for an ad
func addAdVehicleAssociations(tx *sql.Tx, adID int, makeName string, years []string, models []string, engines []string) error {
	if makeName == "" && len(years) == 0 && len(models) == 0 && len(engines) == 0 {
		return nil
	}
	for _, yearStr := range years {
		for _, modelName := range models {
			for _, engineName := range engines {
				var carID int
				err := tx.QueryRow(`
					SELECT c.id FROM Car c
					JOIN Make m ON c.make_id = m.id
					JOIN Year y ON c.year_id = y.id
					JOIN Model mo ON c.model_id = mo.id
					JOIN Engine e ON c.engine_id = e.id
					WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ?
				`, makeName, yearStr, modelName, engineName).Scan(&carID)
				if err == sql.ErrNoRows {
					var makeID, yearID, modelID, engineID int
					err = tx.QueryRow("SELECT id FROM Make WHERE name = ?", makeName).Scan(&makeID)
					if err != nil {
						continue
					}
					err = tx.QueryRow("SELECT id FROM Year WHERE year = ?", yearStr).Scan(&yearID)
					if err != nil {
						continue
					}
					err = tx.QueryRow("SELECT id FROM Model WHERE name = ?", modelName).Scan(&modelID)
					if err != nil {
						continue
					}
					err = tx.QueryRow("SELECT id FROM Engine WHERE name = ?", engineName).Scan(&engineID)
					if err != nil {
						continue
					}
					res, err := tx.Exec("INSERT INTO Car (make_id, year_id, model_id, engine_id) VALUES (?, ?, ?, ?)", makeID, yearID, modelID, engineID)
					if err != nil {
						continue
					}
					id, _ := res.LastInsertId()
					carID = int(id)
				} else if err != nil {
					continue
				}
				_, err = tx.Exec("INSERT OR IGNORE INTO AdCar (ad_id, car_id) VALUES (?, ?)", adID, carID)
				if err != nil {
					continue
				}
			}
		}
	}
	return nil
}

func UpdateAd(id int, ad Ad) bool {
	tx, err := db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE Ad SET description=?, price=?, subcategory_id=? WHERE id=?",
		ad.Description, ad.Price, ad.SubCategoryID, id)
	if err != nil {
		return false
	}

	_, err = tx.Exec("DELETE FROM AdCar WHERE ad_id = ?", id)
	if err != nil {
		return false
	}

	if ad.Make != "" || len(ad.Years) > 0 || len(ad.Models) > 0 || len(ad.Engines) > 0 {
		if err := addAdVehicleAssociations(tx, id, ad.Make, ad.Years, ad.Models, ad.Engines); err != nil {
			return false
		}
	}

	return tx.Commit() == nil
}

func DeleteAd(id int) bool {
	tx, err := db.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM AdCar WHERE ad_id = ?", id)
	if err != nil {
		return false
	}

	_, err = tx.Exec("DELETE FROM Ad WHERE id=?", id)
	if err != nil {
		return false
	}

	return tx.Commit() == nil
}

func GetNextAdID() int {
	row := db.QueryRow("SELECT seq FROM sqlite_sequence WHERE name='Ad'")
	var seq int
	if err := row.Scan(&seq); err != nil {
		return 1
	}
	return seq + 1
}

// GetAdsPage returns a slice of ads for cursor-based pagination
func GetAdsPage(cursorID int, limit int) ([]Ad, bool) {
	query := `
		SELECT a.id, a.description, a.price, a.created_at, a.subcategory_id,
		       psc.name as subcategory
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
	`
	args := []interface{}{}
	if cursorID > 0 {
		query += " WHERE a.id < ?"
		args = append(args, cursorID)
	}
	query += " ORDER BY a.id DESC LIMIT ?"
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
		var subcategory sql.NullString
		if err := rows.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt,
			&ad.SubCategoryID, &subcategory); err != nil {
			fmt.Printf("Error scanning ad: %v\n", err)
			continue
		}

		if subcategory.Valid {
			ad.SubCategory = subcategory.String
		}

		// Get vehicle data
		ad.Make, ad.Years, ad.Models, ad.Engines = getAdVehicleData(ad.ID)
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
	// Check if we have any vehicle filters
	hasVehicleFilters := query.Make != "" || len(query.Years) > 0 || len(query.Models) > 0 || len(query.EngineSizes) > 0

	var sqlQuery string
	var args []interface{}

	if hasVehicleFilters {
		// Use JOIN-based query when we have vehicle filters
		sqlQuery = `
			SELECT DISTINCT a.id, a.description, a.price, a.created_at, a.subcategory_id,
			       psc.name as subcategory
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			JOIN AdCar ac ON a.id = ac.ad_id
			JOIN Car c ON ac.car_id = c.id
			JOIN Make m ON c.make_id = m.id
			JOIN Year y ON c.year_id = y.id
			JOIN Model mo ON c.model_id = mo.id
			JOIN Engine e ON c.engine_id = e.id
			WHERE 1=1
		`

		// Apply vehicle filters
		if query.Make != "" {
			sqlQuery += " AND m.name = ?"
			args = append(args, strings.ToUpper(query.Make))
		}

		if len(query.Years) > 0 {
			placeholders := make([]string, len(query.Years))
			for i, year := range query.Years {
				placeholders[i] = "?"
				args = append(args, year)
			}
			sqlQuery += " AND y.year IN (" + strings.Join(placeholders, ",") + ")"
		}

		if len(query.Models) > 0 {
			placeholders := make([]string, len(query.Models))
			for i, model := range query.Models {
				placeholders[i] = "?"
				args = append(args, strings.ToUpper(model))
			}
			sqlQuery += " AND mo.name IN (" + strings.Join(placeholders, ",") + ")"
		}

		if len(query.EngineSizes) > 0 {
			placeholders := make([]string, len(query.EngineSizes))
			for i, engine := range query.EngineSizes {
				placeholders[i] = "?"
				args = append(args, engine)
			}
			sqlQuery += " AND e.name IN (" + strings.Join(placeholders, ",") + ")"
		}
	} else {
		// Use simple query when no vehicle filters - this includes ALL ads
		sqlQuery = `
			SELECT a.id, a.description, a.price, a.created_at, a.subcategory_id,
			       psc.name as subcategory
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			WHERE 1=1
		`
	}

	// Apply category filters (works for both queries)
	if query.Category != "" {
		sqlQuery += " AND LOWER(psc.name) = LOWER(?)"
		args = append(args, query.Category)
	}

	// Apply cursor pagination
	if cursor != nil {
		sqlQuery += " AND (a.created_at < ? OR (a.created_at = ? AND a.id < ?))"
		timeStr := cursor.LastPosted.Format(time.RFC3339Nano)
		args = append(args, timeStr, timeStr, cursor.LastID)
	}

	// Order by created_at DESC, id DESC
	sqlQuery += " ORDER BY a.created_at DESC, a.id DESC LIMIT ?"
	args = append(args, limit+1) // Get one extra to check if there are more results

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	ads := []Ad{}
	seenIDs := make(map[int]bool)

	for rows.Next() {
		var ad Ad
		var subcategory sql.NullString
		if err := rows.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt,
			&ad.SubCategoryID, &subcategory); err != nil {
			continue
		}

		// Skip if we've already processed this ad (due to potential duplicates in JOIN query)
		if seenIDs[ad.ID] {
			continue
		}
		seenIDs[ad.ID] = true

		if subcategory.Valid {
			ad.SubCategory = subcategory.String
		}

		// Get vehicle data
		ad.Make, ad.Years, ad.Models, ad.Engines = getAdVehicleData(ad.ID)
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
