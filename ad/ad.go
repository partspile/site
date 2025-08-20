package ad

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/user"
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

func (sq SearchQuery) IsEmpty() bool {
	return sq.Make == "" && len(sq.Years) == 0 && len(sq.Models) == 0 &&
		len(sq.EngineSizes) == 0 && sq.Category == "" && sq.SubCategory == ""
}

// SearchCursor represents a point in the search results for pagination
type SearchCursor struct {
	Query      SearchQuery `json:"q"`           // The structured query
	LastID     int         `json:"last_id"`     // Last ID seen
	LastPosted time.Time   `json:"last_posted"` // Timestamp of last item
}

// Ad represents an advertisement in the system
type Ad struct {
	// Core database fields (matching schema order)
	ID            int        `json:"id"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Price         float64    `json:"price"`
	CreatedAt     time.Time  `json:"created_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	SubCategoryID int        `json:"subcategory_id"`
	UserID        int        `json:"user_id"`
	ImageOrder    []int      `json:"image_order"`
	LocationID    int        `json:"location_id"`
	ClickCount    int        `json:"click_count"`
	LastClickedAt *time.Time `json:"last_clicked_at,omitempty"`
	HasVector     bool       `json:"has_vector"`

	// Computed/derived fields from joins
	City      string   `json:"city,omitempty"`
	AdminArea string   `json:"admin_area,omitempty"`
	Country   string   `json:"country,omitempty"`
	Category  string   `json:"category,omitempty"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`

	// Vehicle compatibility fields from AdCar join
	Make    string   `json:"make"`
	Years   []string `json:"years"`
	Models  []string `json:"models"`
	Engines []string `json:"engines"`

	// User-specific computed fields
	Bookmarked bool `json:"bookmarked"`
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

// buildAdQuery builds the complete query for fetching ads with IDs and user context
func buildAdQuery(ids []int, currentUser *user.User) (string, []interface{}) {
	var query string
	var args []interface{}

	if currentUser != nil {
		// Query with bookmark status
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
			       a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_order,
			       l.city, l.admin_area, l.country, l.latitude, l.longitude,
			       CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
			LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
			WHERE a.deleted_at IS NULL
		`
		args = append(args, currentUser.ID)
	} else {
		// Query without bookmark status (default to false)
		query = `
			SELECT a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
			       a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_order,
			       l.city, l.admin_area, l.country, l.latitude, l.longitude,
			       0 as is_bookmarked
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			LEFT JOIN PartCategory pc ON psc.category_id = pc.id
			LEFT JOIN Location l ON a.location_id = l.id
			WHERE a.deleted_at IS NULL
		`
	}

	placeholders := make([]string, len(ids))
	for i := range ids {
		placeholders[i] = "?"
	}
	query += " AND a.id IN (" + strings.Join(placeholders, ",") + ")"
	for _, id := range ids {
		args = append(args, id)
	}

	return query, args
}

// scanAdRows scans database rows into Ad structs
func scanAdRows(rows *sql.Rows) ([]Ad, error) {
	var ads []Ad
	for rows.Next() {
		var ad Ad
		var subcategory, category sql.NullString
		var lastClickedAt sql.NullTime
		var locationID sql.NullInt64
		var imageOrder sql.NullString
		var city, adminArea, country sql.NullString
		var latitude, longitude sql.NullFloat64
		var createdAt string
		var isBookmarked int

		if err := rows.Scan(&ad.ID, &ad.Title, &ad.Description, &ad.Price, &createdAt,
			&ad.SubCategoryID, &ad.UserID, &subcategory, &category, &ad.ClickCount, &lastClickedAt, &locationID, &imageOrder,
			&city, &adminArea, &country, &latitude, &longitude, &isBookmarked); err != nil {
			continue
		}

		// Parse the created_at string into time.Time
		ad.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)

		if category.Valid {
			ad.Category = category.String
		}

		if lastClickedAt.Valid {
			ad.LastClickedAt = &lastClickedAt.Time
		}

		if locationID.Valid {
			ad.LocationID = int(locationID.Int64)
			if city.Valid {
				ad.City = city.String
			}
			if adminArea.Valid {
				ad.AdminArea = adminArea.String
			}
			if country.Valid {
				ad.Country = country.String
			}
			if latitude.Valid && longitude.Valid {
				ad.Latitude = &latitude.Float64
				ad.Longitude = &longitude.Float64
			}
		}

		if imageOrder.Valid {
			_ = json.Unmarshal([]byte(imageOrder.String), &ad.ImageOrder)
		}

		ad.Bookmarked = isBookmarked == 1

		ads = append(ads, ad)
	}
	return ads, nil
}

// GetAd retrieves an ad by ID from the Ad table
func GetAd(id int, currentUser *user.User) (Ad, bool) {
	ads, err := GetAdsByIDs([]int{id}, currentUser)
	if err != nil || len(ads) == 0 {
		return Ad{}, false
	}
	return ads[0], true
}

// GetAdWithVehicle retrieves an ad by ID from the active ads table with vehicle data
func GetAdWithVehicle(id int, currentUser *user.User) (Ad, bool) {
	ad, ok := GetAd(id, currentUser)
	if !ok {
		return Ad{}, false
	}

	// Get vehicle data
	ad.Make, ad.Years, ad.Models, ad.Engines = GetVehicleData(id)

	return ad, true
}

func AddAd(ad Ad) int {
	tx, err := db.Begin()
	if err != nil {
		return 0
	}
	defer tx.Rollback()

	createdAt := time.Now().UTC().Format(time.RFC3339)
	imgOrderJSON, _ := json.Marshal(ad.ImageOrder)
	res, err := tx.Exec("INSERT INTO Ad (title, description, price, created_at, subcategory_id, user_id, location_id, image_order) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		ad.Title, ad.Description, ad.Price, createdAt, ad.SubCategoryID, ad.UserID, ad.LocationID, string(imgOrderJSON))
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

// UpdateAd updates an existing ad
func UpdateAd(ad Ad) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	imgOrderJSON, _ := json.Marshal(ad.ImageOrder)
	_, err = tx.Exec("UPDATE Ad SET title = ?, description = ?, price = ?, subcategory_id = ?, location_id = ?, image_order = ? WHERE id = ?",
		ad.Title, ad.Description, ad.Price, ad.SubCategoryID, ad.LocationID, string(imgOrderJSON), ad.ID)
	if err != nil {
		return err
	}

	// First, remove existing vehicle associations for this ad
	_, err = tx.Exec("DELETE FROM AdCar WHERE ad_id = ?", ad.ID)
	if err != nil {
		return err
	}

	// Then, add the new ones
	if ad.Make != "" || len(ad.Years) > 0 || len(ad.Models) > 0 || len(ad.Engines) > 0 {
		if err := addAdVehicleAssociations(tx, ad.ID, ad.Make, ad.Years, ad.Models, ad.Engines); err != nil {
			return err
		}
	}

	return tx.Commit()
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
	if len(ids) == 0 {
		return nil, nil
	}

	query, args := buildAdQuery(ids, currentUser)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ads, err := scanAdRows(rows)
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

	return result, nil
}

// GetLocationByID fetches a Location by its ID
func GetLocationByID(id int) (city, adminArea, country, raw string, err error) {
	row := db.QueryRow("SELECT city, admin_area, country, raw_text FROM Location WHERE id = ?", id)
	err = row.Scan(&city, &adminArea, &country, &raw)
	return
}

// GetLocationWithCoords returns location data including coordinates
func GetLocationWithCoords(id int) (city, adminArea, country, raw string, latitude, longitude *float64, err error) {
	row := db.QueryRow("SELECT city, admin_area, country, raw_text, latitude, longitude FROM Location WHERE id = ?", id)
	err = row.Scan(&city, &adminArea, &country, &raw, &latitude, &longitude)
	return
}

// GetMostPopularAds returns the top n ads by popularity using SQL
func GetMostPopularAds(n int) []Ad {
	log.Printf("[GetMostPopularAds] Querying for top %d popular ads", n)
	query := `
		SELECT 
			a.id, a.title, a.description, a.price, a.created_at, 
			a.subcategory_id, a.user_id, a.location_id, a.click_count,
			a.last_clicked_at, a.image_order,
			psc.name as subcategory,
			l.city, l.admin_area, l.country
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN Location l ON a.location_id = l.id
		WHERE a.deleted_at IS NULL
		ORDER BY (
			a.click_count * 2 + 
			COALESCE((SELECT COUNT(*) FROM BookmarkedAd ba WHERE ba.ad_id = a.id), 0) * 3 + 
			100.0 / (julianday('now') - julianday(a.created_at))
		) DESC
		LIMIT ?
	`

	rows, err := db.Query(query, n)
	if err != nil {
		log.Printf("[GetMostPopularAds] SQL error: %v", err)
		return nil
	}
	defer rows.Close()

	var ads []Ad
	rowCount := 0
	for rows.Next() {
		rowCount++
		var ad Ad
		var subcategory sql.NullString
		var lastClickedAt sql.NullTime
		var locationID sql.NullInt64
		var imageOrder sql.NullString
		var city, adminArea, country sql.NullString

		err := rows.Scan(
			&ad.ID, &ad.Title, &ad.Description, &ad.Price, &ad.CreatedAt,
			&ad.SubCategoryID, &ad.UserID, &locationID, &ad.ClickCount,
			&lastClickedAt, &imageOrder, &subcategory,
			&city, &adminArea, &country,
		)
		if err != nil {
			log.Printf("[GetMostPopularAds] Row scan error: %v", err)
			continue
		}

		if lastClickedAt.Valid {
			ad.LastClickedAt = &lastClickedAt.Time
		}
		if locationID.Valid {
			ad.LocationID = int(locationID.Int64)
		}
		if imageOrder.Valid {
			_ = json.Unmarshal([]byte(imageOrder.String), &ad.ImageOrder)
		}
		if city.Valid {
			ad.City = city.String
		}
		if adminArea.Valid {
			ad.AdminArea = adminArea.String
		}
		if country.Valid {
			ad.Country = country.String
		}

		// Get vehicle data
		ad.Make, ad.Years, ad.Models, ad.Engines = GetVehicleData(ad.ID)
		ads = append(ads, ad)
	}
	log.Printf("[GetMostPopularAds] Found %d ads from SQL query", rowCount)
	return ads
}

// GetAdsWithoutVectors returns ads that don't have vector embeddings
func GetAdsWithoutVectors() ([]Ad, error) {
	log.Printf("[GetAdsWithoutVectors] Querying for ads without vectors")
	query := `
		SELECT 
			a.id, a.title, a.description, a.price, a.created_at, 
			a.subcategory_id, a.user_id, a.location_id, a.click_count,
			a.last_clicked_at, a.image_order, a.has_vector,
			psc.name as subcategory,
			l.city, l.admin_area, l.country
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN Location l ON a.location_id = l.id
		WHERE a.has_vector = 0 AND a.deleted_at IS NULL
		ORDER BY a.created_at DESC
		LIMIT 50
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("[GetAdsWithoutVectors] SQL error: %v", err)
		return nil, err
	}
	defer rows.Close()

	var ads []Ad
	rowCount := 0
	for rows.Next() {
		rowCount++
		var ad Ad
		var subcategory sql.NullString
		var lastClickedAt sql.NullTime
		var locationID sql.NullInt64
		var imageOrder sql.NullString
		var city, adminArea, country sql.NullString
		var hasVector int

		err := rows.Scan(
			&ad.ID, &ad.Title, &ad.Description, &ad.Price, &ad.CreatedAt,
			&ad.SubCategoryID, &ad.UserID, &locationID, &ad.ClickCount,
			&lastClickedAt, &imageOrder, &hasVector, &subcategory,
			&city, &adminArea, &country,
		)
		if err != nil {
			log.Printf("[GetAdsWithoutVectors] Row scan error: %v", err)
			continue
		}

		ad.HasVector = hasVector == 1

		if lastClickedAt.Valid {
			ad.LastClickedAt = &lastClickedAt.Time
		}
		if locationID.Valid {
			ad.LocationID = int(locationID.Int64)
		}
		if imageOrder.Valid {
			_ = json.Unmarshal([]byte(imageOrder.String), &ad.ImageOrder)
		}
		if city.Valid {
			ad.City = city.String
		}
		if adminArea.Valid {
			ad.AdminArea = adminArea.String
		}
		if country.Valid {
			ad.Country = country.String
		}

		// Get vehicle data
		ad.Make, ad.Years, ad.Models, ad.Engines = GetVehicleData(ad.ID)
		ads = append(ads, ad)
	}
	log.Printf("[GetAdsWithoutVectors] Found %d ads without vectors from SQL query", rowCount)
	return ads, nil
}

// MarkAdAsHavingVector marks an ad as having a vector embedding
func MarkAdAsHavingVector(adID int) error {
	_, err := db.Exec("UPDATE Ad SET has_vector = 1 WHERE id = ?", adID)
	if err != nil {
		log.Printf("[MarkAdAsHavingVector] Failed to mark ad %d as having vector: %v", adID, err)
		return err
	}
	log.Printf("[MarkAdAsHavingVector] Successfully marked ad %d as having vector", adID)
	return nil
}

// GetAdsByIDsOptimized returns ads for a list of IDs with all data in a single query
func GetAdsByIDsOptimized(ids []int) ([]Ad, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// Build query with IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `
		SELECT 
			a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
			a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_order,
			l.city, l.admin_area, l.country, l.latitude, l.longitude,
			GROUP_CONCAT(DISTINCT m.name ORDER BY m.name) as makes,
			GROUP_CONCAT(DISTINCT y.year ORDER BY y.year) as years,
			GROUP_CONCAT(DISTINCT mo.name ORDER BY mo.name) as models,
			GROUP_CONCAT(DISTINCT e.name ORDER BY e.name) as engines
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		LEFT JOIN Location l ON a.location_id = l.id
		LEFT JOIN AdCar ac ON a.id = ac.ad_id
		LEFT JOIN Car c ON ac.car_id = c.id
		LEFT JOIN Make m ON c.make_id = m.id
		LEFT JOIN Year y ON c.year_id = y.id
		LEFT JOIN Model mo ON c.model_id = mo.id
		LEFT JOIN Engine e ON c.engine_id = e.id
		WHERE a.id IN (` + strings.Join(placeholders, ",") + `) AND a.deleted_at IS NULL
		GROUP BY a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
			a.user_id, psc.name, pc.name, a.click_count, a.last_clicked_at, a.location_id, a.image_order,
			l.city, l.admin_area, l.country, l.latitude, l.longitude`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	adMap := make(map[int]Ad)
	for rows.Next() {
		var ad Ad
		var subcategory, category sql.NullString
		var lastClickedAt sql.NullTime
		var locationID sql.NullInt64
		var imageOrder sql.NullString
		var city, adminArea, country sql.NullString
		var latitude, longitude sql.NullFloat64
		var makes, years, models, engines sql.NullString
		var createdAt string
		if err := rows.Scan(&ad.ID, &ad.Title, &ad.Description, &ad.Price, &createdAt,
			&ad.SubCategoryID, &ad.UserID, &subcategory, &category, &ad.ClickCount, &lastClickedAt, &locationID, &imageOrder,
			&city, &adminArea, &country, &latitude, &longitude, &makes, &years, &models, &engines); err != nil {
			continue
		}

		// Parse the created_at string into time.Time
		ad.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)

		if category.Valid {
			ad.Category = category.String
		}

		if lastClickedAt.Valid {
			ad.LastClickedAt = &lastClickedAt.Time
		}

		if locationID.Valid {
			ad.LocationID = int(locationID.Int64)
		}

		if latitude.Valid && longitude.Valid {
			ad.Latitude = &latitude.Float64
			ad.Longitude = &longitude.Float64
		}

		if imageOrder.Valid {
			_ = json.Unmarshal([]byte(imageOrder.String), &ad.ImageOrder)
		}
		if city.Valid {
			ad.City = city.String
		}
		if adminArea.Valid {
			ad.AdminArea = adminArea.String
		}
		if country.Valid {
			ad.Country = country.String
		}

		// Set bookmark status to false for anonymous users
		ad.Bookmarked = false

		// Parse vehicle data from GROUP_CONCAT results
		if makes.Valid && makes.String != "" {
			makeList := strings.Split(makes.String, ",")
			if len(makeList) > 0 {
				ad.Make = makeList[0] // Use first make as primary
			}
		}
		if years.Valid && years.String != "" {
			ad.Years = strings.Split(years.String, ",")
		}
		if models.Valid && models.String != "" {
			ad.Models = strings.Split(models.String, ",")
		}
		if engines.Valid && engines.String != "" {
			ad.Engines = strings.Split(engines.String, ",")
		}

		adMap[ad.ID] = ad
	}
	// Preserve order of ids
	ads := make([]Ad, 0, len(ids))
	for _, id := range ids {
		if ad, ok := adMap[id]; ok {
			ads = append(ads, ad)
		}
	}
	return ads, nil
}

// GetAdsByIDsOptimizedWithBookmarks returns ads for a list of IDs with all data and bookmark status in a single query
func GetAdsByIDsOptimizedWithBookmarks(ids []int, userID int) ([]Ad, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	// Build query with IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	args = append(args, userID) // Add userID for bookmark check
	query := `
		SELECT 
			a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
			a.user_id, psc.name as subcategory, pc.name as category, a.click_count, a.last_clicked_at, a.location_id, a.image_order,
			l.city, l.admin_area, l.country, l.latitude, l.longitude,
			GROUP_CONCAT(DISTINCT m.name ORDER BY m.name) as makes,
			GROUP_CONCAT(DISTINCT y.year ORDER BY y.year) as years,
			GROUP_CONCAT(DISTINCT mo.name ORDER BY mo.name) as models,
			GROUP_CONCAT(DISTINCT e.name ORDER BY e.name) as engines,
			CASE WHEN ba.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_bookmarked
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN PartCategory pc ON psc.category_id = pc.id
		LEFT JOIN Location l ON a.location_id = l.id
		LEFT JOIN BookmarkedAd ba ON a.id = ba.ad_id AND ba.user_id = ?
		LEFT JOIN AdCar ac ON a.id = ac.ad_id
		LEFT JOIN Car c ON ac.car_id = c.id
		LEFT JOIN Make m ON c.make_id = m.id
		LEFT JOIN Year y ON c.year_id = y.id
		LEFT JOIN Model mo ON c.model_id = mo.id
		LEFT JOIN Engine e ON c.engine_id = e.id
		WHERE a.id IN (` + strings.Join(placeholders, ",") + `) AND a.deleted_at IS NULL
		GROUP BY a.id, a.title, a.description, a.price, a.created_at, a.subcategory_id,
			a.user_id, psc.name, pc.name, a.click_count, a.last_clicked_at, a.location_id, a.image_order,
			l.city, l.admin_area, l.country, l.latitude, l.longitude, ba.ad_id`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	adMap := make(map[int]Ad)
	for rows.Next() {
		var ad Ad
		var subcategory, category sql.NullString
		var lastClickedAt sql.NullTime
		var locationID sql.NullInt64
		var imageOrder sql.NullString
		var city, adminArea, country sql.NullString
		var latitude, longitude sql.NullFloat64
		var makes, years, models, engines sql.NullString
		var isBookmarked int
		var createdAt string
		if err := rows.Scan(&ad.ID, &ad.Title, &ad.Description, &ad.Price, &createdAt,
			&ad.SubCategoryID, &ad.UserID, &subcategory, &category, &ad.ClickCount, &lastClickedAt, &locationID, &imageOrder,
			&city, &adminArea, &country, &latitude, &longitude, &makes, &years, &models, &engines, &isBookmarked); err != nil {
			continue
		}

		// Parse the created_at string into time.Time
		ad.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)

		if category.Valid {
			ad.Category = category.String
		}

		if lastClickedAt.Valid {
			ad.LastClickedAt = &lastClickedAt.Time
		}

		if locationID.Valid {
			ad.LocationID = int(locationID.Int64)
		}

		if latitude.Valid && longitude.Valid {
			ad.Latitude = &latitude.Float64
			ad.Longitude = &longitude.Float64
		}

		if imageOrder.Valid {
			_ = json.Unmarshal([]byte(imageOrder.String), &ad.ImageOrder)
		}
		if city.Valid {
			ad.City = city.String
		}
		if adminArea.Valid {
			ad.AdminArea = adminArea.String
		}
		if country.Valid {
			ad.Country = country.String
		}

		// Set bookmark status
		ad.Bookmarked = isBookmarked == 1

		// Parse vehicle data from GROUP_CONCAT results
		if makes.Valid && makes.String != "" {
			makeList := strings.Split(makes.String, ",")
			if len(makeList) > 0 {
				ad.Make = makeList[0] // Use first make as primary
			}
		}
		if years.Valid && years.String != "" {
			ad.Years = strings.Split(years.String, ",")
		}
		if models.Valid && models.String != "" {
			ad.Models = strings.Split(models.String, ",")
		}
		if engines.Valid && engines.String != "" {
			ad.Engines = strings.Split(engines.String, ",")
		}

		adMap[ad.ID] = ad
	}
	// Preserve order of ids
	ads := make([]Ad, 0, len(ids))
	for _, id := range ids {
		if ad, ok := adMap[id]; ok {
			ads = append(ads, ad)
		}
	}
	return ads, nil
}
