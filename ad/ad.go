package ad

import (
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/user"
)

// Ad represents an advertisement in the system
type Ad struct {
	// Core database fields (matching schema order)
	ID            int        `json:"id" db:"id"`
	Title         string     `json:"title" db:"title"`
	Description   string     `json:"description" db:"description"`
	Price         float64    `json:"price" db:"price"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	SubCategoryID int        `json:"subcategory_id" db:"subcategory_id"`
	UserID        int        `json:"user_id" db:"user_id"`
	ImageCount    int        `json:"image_count" db:"image_count"`
	LocationID    int        `json:"location_id" db:"location_id"`
	ClickCount    int        `json:"click_count" db:"click_count"`
	LastClickedAt *time.Time `json:"last_clicked_at,omitempty" db:"last_clicked_at"`
	HasVector     bool       `json:"has_vector" db:"has_vector"`

	// Computed/derived fields from joins
	RawLocation sql.NullString  `json:"raw_location,omitempty" db:"raw_location"`
	City        sql.NullString  `json:"city,omitempty" db:"city"`
	AdminArea   sql.NullString  `json:"admin_area,omitempty" db:"admin_area"`
	Country     sql.NullString  `json:"country,omitempty" db:"country"`
	Category    sql.NullString  `json:"category,omitempty" db:"category"`
	SubCategory sql.NullString  `json:"subcategory,omitempty" db:"subcategory"`
	Latitude    sql.NullFloat64 `json:"latitude,omitempty" db:"latitude"`
	Longitude   sql.NullFloat64 `json:"longitude,omitempty" db:"longitude"`

	// Vehicle compatibility fields from AdCar join
	Make    string   `json:"make" db:"make"`
	Years   []string `json:"years" db:"years"`
	Models  []string `json:"models" db:"models"`
	Engines []string `json:"engines" db:"engines"`

	// User-specific computed fields
	Bookmarked bool `json:"bookmarked" db:"is_bookmarked"`
}

// IsArchived returns true if the ad has been archived
func (a Ad) IsArchived() bool {
	return a.DeletedAt != nil
}

// GetVehicleData retrieves vehicle information for an ad
func GetVehicleData(adID int) (makeName string, years []string, models []string, engines []string) {
	query := `
		SELECT DISTINCT m.name, y.year, mo.name, e.name
		FROM AdCar ac
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE ac.ad_id = ?
		ORDER BY m.name, y.year, mo.name, e.name
	`

	rows, err := db.Query(query, adID)
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
	makes := make([]string, 0, len(makeSet))
	for m := range makeSet {
		makes = append(makes, m)
	}
	sort.Strings(makes)
	if len(makes) > 0 {
		makeName = makes[0]
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

// buildAdQueryWithDeleted builds the complete query for fetching ads with IDs and user context, optionally including deleted ads
func buildAdQueryWithDeleted(ids []int, currentUser *user.User, includeDeleted bool) (string, []interface{}) {
	var query string
	var args []interface{}

	if currentUser != nil {
		// Query with bookmark status
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.deleted_at, a.subcategory_id,
			       a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_count,
			       l.raw_text as raw_location, l.city, l.admin_area, l.country, l.latitude, l.longitude,
			       CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
		`
		args = append(args, currentUser.ID)
	} else {
		// Query without bookmark status (default to false)
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.deleted_at, a.subcategory_id,
			       a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_count,
			       l.raw_text as raw_location, l.city, l.admin_area, l.country, l.latitude, l.longitude,
			       0 as is_bookmarked
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
		`
	}

	if !includeDeleted {
		query += " WHERE a.deleted_at IS NULL"
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}
	if includeDeleted {
		query += " WHERE a.id IN (" + strings.Join(placeholders, ",") + ")"
	} else {
		query += " AND a.id IN (" + strings.Join(placeholders, ",") + ")"
	}
	for _, id := range ids {
		args = append(args, id)
	}

	return query, args
}

// GetAd retrieves an ad by ID from the Ad table (includes deleted ads)
// Callers should check IsArchived() if they need to filter out deleted ads
func GetAd(id int, currentUser *user.User) (Ad, bool) {
	ads, err := GetAdsByIDsWithDeleted([]int{id}, currentUser, true)
	if err != nil || len(ads) == 0 {
		return Ad{}, false
	}
	return ads[0], true
}

func AddAd(ad Ad) int {
	tx, err := db.Begin()
	if err != nil {
		return 0
	}
	defer tx.Rollback()

	createdAt := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec("INSERT INTO Ad (title, description, price, created_at, subcategory_id, user_id, location_id, image_count) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		ad.Title, ad.Description, ad.Price, createdAt, ad.SubCategoryID, ad.UserID, ad.LocationID, ad.ImageCount)
	if err != nil {
		return 0
	}
	adID, _ := res.LastInsertId()

	if err := addAdVehicleAssociations(tx, int(adID), ad.Make, ad.Years, ad.Models, ad.Engines); err != nil {
		return 0
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
				if err != nil {
					if err == sql.ErrNoRows {
						return fmt.Errorf("car not found for make=%s, year=%s, model=%s, engine=%s", makeName, yearStr, modelName, engineName)
					}
					return fmt.Errorf("error looking up car: %w", err)
				}

				_, err = tx.Exec("INSERT OR IGNORE INTO AdCar (ad_id, car_id) VALUES (?, ?)", adID, carID)
				if err != nil {
					return fmt.Errorf("error inserting AdCar association: %w", err)
				}
			}
		}
	}
	return nil
}

// ArchiveAd archives an ad using soft delete
func ArchiveAd(id int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = ? WHERE id = ?",
		time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

// RestoreAd restores an archived ad by clearing the deleted_at field
func RestoreAd(adID int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = NULL WHERE id = ?", adID)
	return err
}

// ArchiveAdsByUserID archives all ads for a specific user
func ArchiveAdsByUserID(userID int) error {
	_, err := db.Exec("UPDATE Ad SET deleted_at = ? WHERE user_id = ? AND deleted_at IS NULL",
		time.Now().UTC().Format(time.RFC3339Nano), userID)
	return err
}

// GetAdsByIDs returns ads for a list of IDs
func GetAdsByIDs(ids []int, currentUser *user.User) ([]Ad, error) {
	return GetAdsByIDsWithDeleted(ids, currentUser, false)
}

// GetAdsByIDsWithDeleted returns ads for a list of IDs, optionally including deleted ads
func GetAdsByIDsWithDeleted(ids []int, currentUser *user.User, includeDeleted bool) ([]Ad, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args := buildAdQueryWithDeleted(ids, currentUser, includeDeleted)

	var ads []Ad
	err := db.Select(&ads, query, args...)
	if err != nil {
		return nil, err
	}

	// For single ID, just return the first result directly
	if len(ids) == 1 {
		return ads, nil
	}

	// For multiple IDs, create a map for quick lookup and preserve order
	adMap := make(map[int]Ad)
	for _, ad := range ads {
		adMap[ad.ID] = ad
	}

	// Preserve order of ids
	result := make([]Ad, 0, len(ids))
	for _, id := range ids {
		if ad, ok := adMap[id]; ok {
			result = append(result, ad)
		}
	}

	log.Printf("[GetAdsByIDs] Returning ads in order: %v", func() []int {
		debugResult := make([]int, len(result))
		for i, ad := range result {
			debugResult[i] = ad.ID
		}
		return debugResult
	}())
	return result, nil
}

// GetMostPopularAds returns the top n ads by popularity using SQL
func GetMostPopularAds(n int) []Ad {
	log.Printf("[GetMostPopularAds] Querying for top %d popular ads", n)
	query := `
		SELECT 
			a.id, a.title, a.description, a.price, a.created_at, 
			a.subcategory_id, a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_count,
			l.raw_text as raw_location, l.city, l.admin_area, l.country, l.latitude, l.longitude,
			0 as is_bookmarked
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		LEFT JOIN Location l ON a.location_id = l.id
		WHERE a.deleted_at IS NULL
		ORDER BY (
			a.click_count * 2 + 
			COALESCE((SELECT COUNT(*) FROM BookmarkedAd ba WHERE ba.ad_id = a.id), 0) * 3 + 
			100.0 / (julianday('now') - julianday(a.created_at))
		) DESC
		LIMIT ?
	`

	var ads []Ad
	err := db.Select(&ads, query, n)
	if err != nil {
		log.Printf("[GetMostPopularAds] SQL error: %v", err)
		return nil
	}

	// Get vehicle data for each ad
	for i := range ads {
		ads[i].Make, ads[i].Years, ads[i].Models, ads[i].Engines = GetVehicleData(ads[i].ID)
	}

	log.Printf("[GetMostPopularAds] Found %d ads from SQL query", len(ads))
	return ads
}

// getAdIDsForTreeCriteria returns ad IDs filtered by tree criteria
// If adIDs is nil/empty, searches all ads; otherwise filters by the provided ad IDs
func getAdIDsForTreeCriteria(adIDs []int, makeName, year, model, engine, category, subcategory string) ([]int, error) {
	// URL decode all parameters
	makeName, _ = url.QueryUnescape(makeName)
	year, _ = url.QueryUnescape(year)
	model, _ = url.QueryUnescape(model)
	engine, _ = url.QueryUnescape(engine)
	category, _ = url.QueryUnescape(category)
	subcategory, _ = url.QueryUnescape(subcategory)

	var query string
	var args []interface{}

	// Build the base query
	query = `
		SELECT DISTINCT a.id
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		JOIN AdCar ac ON a.id = ac.ad_id
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE a.deleted_at IS NULL
	`

	// Add tree criteria filters
	if makeName != "" {
		query += " AND m.name = ?"
		args = append(args, makeName)
	}
	if year != "" {
		query += " AND y.year = ?"
		args = append(args, year)
	}
	if model != "" {
		query += " AND mo.name = ?"
		args = append(args, model)
	}
	if engine != "" {
		query += " AND e.name = ?"
		args = append(args, engine)
	}
	if category != "" {
		query += " AND pc.name = ?"
		args = append(args, category)
	}
	if subcategory != "" {
		query += " AND psc.name = ?"
		args = append(args, subcategory)
	}

	// Add ad ID filtering if provided
	if len(adIDs) > 0 {
		placeholders := make([]string, len(adIDs))
		for i := range adIDs {
			placeholders[i] = "?"
		}
		query += fmt.Sprintf(" AND a.id IN (%s)", strings.Join(placeholders, ","))
		for _, id := range adIDs {
			args = append(args, id)
		}
	}

	query += " ORDER BY a.created_at DESC, a.id DESC"

	var ids []int
	err := db.Select(&ids, query, args...)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return ids, nil
}

// GetAdsForAll returns all ads for a specific make/year/model/engine/category/subcategory
func GetAdsForAll(makeName, year, model, engine, category, subcategory string) ([]Ad, error) {
	filteredIDs, err := getAdIDsForTreeCriteria(nil, makeName, year, model, engine, category, subcategory)
	if err != nil {
		return nil, err
	}

	if len(filteredIDs) == 0 {
		return nil, nil
	}

	return GetAdsByIDs(filteredIDs, nil) // user will be passed in actual usage
}

// GetAdsForAdIDs returns ads for a specific make/year/model/engine/category/subcategory, filtered by ad IDs
func GetAdsForAdIDs(adIDs []int, makeName, year, model, engine, category, subcategory string) ([]Ad, error) {
	if len(adIDs) == 0 {
		return nil, nil
	}

	filteredIDs, err := getAdIDsForTreeCriteria(adIDs, makeName, year, model, engine, category, subcategory)
	if err != nil {
		return nil, err
	}

	if len(filteredIDs) == 0 {
		return nil, nil
	}

	return GetAdsByIDs(filteredIDs, nil)
}

// GetUserActiveAdIDs returns a list of active ad IDs owned by the user
func GetUserActiveAdIDs(userID int) ([]int, error) {
	query := `SELECT id FROM Ad 
		WHERE user_id = ? AND deleted_at IS NULL 
		ORDER BY created_at DESC`

	var adIDs []int
	err := db.Select(&adIDs, query, userID)
	return adIDs, err
}

// GetUserDeletedAdIDs returns a list of deleted ad IDs owned by the user
func GetUserDeletedAdIDs(userID int) ([]int, error) {
	query := `SELECT id FROM Ad 
		WHERE user_id = ? AND deleted_at IS NOT NULL 
		ORDER BY deleted_at DESC`

	var adIDs []int
	err := db.Select(&adIDs, query, userID)
	return adIDs, err
}
