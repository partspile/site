package ad

import (
	"database/sql"
	"fmt"
	"sort"
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
	ID            int       `json:"id"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Price         float64   `json:"price"`
	CreatedAt     time.Time `json:"created_at"`
	SubCategoryID *int      `json:"subcategory_id,omitempty"`
	UserID        int       `json:"user_id"`
	// Runtime fields populated via joins
	Year         string     `json:"year,omitempty"`
	Make         string     `json:"make"`
	Years        []string   `json:"years"`
	Models       []string   `json:"models"`
	Engines      []string   `json:"engines"`
	Category     string     `json:"category,omitempty"`
	SubCategory  string     `json:"subcategory,omitempty"`
	DeletionDate *time.Time `json:"deletion_date,omitempty"`
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

func GetAd(id int) (Ad, bool) {
	row := db.QueryRow(`
		SELECT a.id, a.description, a.price, a.created_at, a.subcategory_id,
		       a.user_id, psc.name as subcategory
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		WHERE a.id = ?
	`, id)

	var ad Ad
	var subcategory sql.NullString
	if err := row.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt,
		&ad.SubCategoryID, &ad.UserID, &subcategory); err != nil {
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

	createdAt := time.Now().UTC().Format(time.RFC3339)
	res, err := tx.Exec("INSERT INTO Ad (description, price, created_at, subcategory_id, user_id) VALUES (?, ?, ?, ?, ?)",
		ad.Description, ad.Price, createdAt, ad.SubCategoryID, ad.UserID)
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
			       psc.name as subcategory,
			       m.name as make_name, y.year, mo.name as model_name, e.name as engine_name
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
			       psc.name as subcategory,
			       NULL as make_name, NULL as year, NULL as model_name, NULL as engine_name
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
	sqlQuery += " ORDER BY a.created_at DESC, a.id DESC"

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	// Map to collect all vehicle data for each ad
	adMap := make(map[int]*Ad)
	makeSet := make(map[int]map[string]bool)
	yearSet := make(map[int]map[string]bool)
	modelSet := make(map[int]map[string]bool)
	engineSet := make(map[int]map[string]bool)

	for rows.Next() {
		var (
			id          int
			description string
			price       float64
			createdAt   time.Time
			subcatID    sql.NullInt64
			subcategory sql.NullString
			makeName    sql.NullString
			year        sql.NullInt64
			modelName   sql.NullString
			engineName  sql.NullString
		)

		if err := rows.Scan(&id, &description, &price, &createdAt,
			&subcatID, &subcategory, &makeName, &year, &modelName, &engineName); err != nil {
			continue
		}

		// Get or create ad
		ad, exists := adMap[id]
		if !exists {
			ad = &Ad{
				ID:          id,
				Description: description,
				Price:       price,
				CreatedAt:   createdAt,
			}
			if subcatID.Valid {
				intVal := int(subcatID.Int64)
				ad.SubCategoryID = &intVal
			}
			if subcategory.Valid {
				ad.SubCategory = subcategory.String
			}
			adMap[id] = ad

			// Initialize sets for this ad
			makeSet[id] = make(map[string]bool)
			yearSet[id] = make(map[string]bool)
			modelSet[id] = make(map[string]bool)
			engineSet[id] = make(map[string]bool)
		}

		// Collect vehicle data
		if makeName.Valid {
			makeSet[id][makeName.String] = true
		}
		if year.Valid {
			yearSet[id][fmt.Sprintf("%d", year.Int64)] = true
		}
		if modelName.Valid {
			modelSet[id][modelName.String] = true
		}
		if engineName.Valid {
			engineSet[id][engineName.String] = true
		}
	}

	// Convert map to sorted slice
	ads := make([]Ad, 0, len(adMap))
	for id, ad := range adMap {
		// Convert sets to sorted slices
		makes := make([]string, 0, len(makeSet[id]))
		for m := range makeSet[id] {
			makes = append(makes, m)
		}
		sort.Strings(makes)
		if len(makes) > 0 {
			ad.Make = makes[0]
		}

		years := make([]string, 0, len(yearSet[id]))
		for y := range yearSet[id] {
			years = append(years, y)
		}
		sort.Strings(years)
		ad.Years = years

		models := make([]string, 0, len(modelSet[id]))
		for m := range modelSet[id] {
			models = append(models, m)
		}
		sort.Strings(models)
		ad.Models = models

		engines := make([]string, 0, len(engineSet[id]))
		for e := range engineSet[id] {
			engines = append(engines, e)
		}
		sort.Strings(engines)
		ad.Engines = engines

		// If there are no vehicle filters, populate vehicle data using getAdVehicleData
		if !hasVehicleFilters {
			ad.Make, ad.Years, ad.Models, ad.Engines = getAdVehicleData(id)
		}

		ads = append(ads, *ad)
	}

	// Sort by created_at DESC, id DESC
	sort.Slice(ads, func(i, j int) bool {
		if ads[i].CreatedAt.Equal(ads[j].CreatedAt) {
			return ads[i].ID > ads[j].ID
		}
		return ads[i].CreatedAt.After(ads[j].CreatedAt)
	})

	// Apply limit and check for more results
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

// GetAllAds returns all ads in the system
func GetAllAds() ([]Ad, error) {
	rows, err := db.Query(`
		SELECT
			a.id, a.description, a.price, a.created_at, a.user_id,
			GROUP_CONCAT(DISTINCT m.name) as make,
			GROUP_CONCAT(DISTINCT y.year) as years,
			GROUP_CONCAT(DISTINCT mo.name) as models,
			GROUP_CONCAT(DISTINCT e.name) as engines
		FROM Ad a
		LEFT JOIN AdCar ac ON a.id = ac.ad_id
		LEFT JOIN Car c ON ac.car_id = c.id
		LEFT JOIN Make m ON c.make_id = m.id
		LEFT JOIN Year y ON c.year_id = y.id
		LEFT JOIN Model mo ON c.model_id = mo.id
		LEFT JOIN Engine e ON c.engine_id = e.id
		GROUP BY a.id
		ORDER BY a.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ads []Ad
	for rows.Next() {
		var ad Ad
		var make, years, models, engines sql.NullString
		if err := rows.Scan(
			&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt, &ad.UserID,
			&make, &years, &models, &engines,
		); err != nil {
			return nil, err
		}

		if make.Valid {
			ad.Make = make.String
		}
		if years.Valid {
			ad.Years = strings.Split(years.String, ",")
		}
		if models.Valid {
			ad.Models = strings.Split(models.String, ",")
		}
		if engines.Valid {
			ad.Engines = strings.Split(engines.String, ",")
		}

		ads = append(ads, ad)
	}

	return ads, nil
}

// GetAllDeadAds returns all archived ads in the system
func GetAllDeadAds() ([]Ad, error) {
	rows, err := db.Query(`
		SELECT id, description, price, created_at, user_id, deletion_date
		FROM AdDead
		ORDER BY deletion_date DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ads []Ad
	for rows.Next() {
		var ad Ad
		var deletionDate string
		err := rows.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt, &ad.UserID, &deletionDate)
		if err != nil {
			return nil, err
		}
		// For dead ads, we'll fetch the basic info and the deletion date.
		// Vehicle info like make, model, etc., is not directly available in AdDead
		// and is not required for this admin view.
		if parsedTime, err := time.Parse(time.RFC3339Nano, deletionDate); err == nil {
			ad.DeletionDate = &parsedTime
		}
		ads = append(ads, ad)
	}
	return ads, nil
}

// GetAdsByUserID returns all ads for a given user
func GetAdsByUserID(userID int) ([]Ad, error) {
	rows, err := db.Query(`
		SELECT a.id, a.user_id, a.title, a.description, a.price, a.created_at,
		       ac.year, ac.make, ac.model, ac.category, ac.subcategory
		FROM Ad a
		LEFT JOIN AdCar ac ON a.id = ac.ad_id
		WHERE a.user_id = ?
		ORDER BY a.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ads []Ad
	for rows.Next() {
		var ad Ad
		var createdAt string
		var year, make, model, category, subcategory sql.NullString
		err := rows.Scan(&ad.ID, &ad.UserID, &ad.Title, &ad.Description, &ad.Price,
			&createdAt, &year, &make, &model, &category, &subcategory)
		if err != nil {
			return nil, err
		}
		ad.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
		if year.Valid {
			ad.Year = year.String
			ad.Make = make.String
			ad.Models = []string{model.String}
			ad.Category = category.String
			ad.SubCategory = subcategory.String
		}
		ads = append(ads, ad)
	}
	return ads, nil
}

// GetAdByID returns a single ad by ID
func GetAdByID(id int) (Ad, error) {
	var ad Ad
	var createdAt string
	var year, make, model, category, subcategory sql.NullString
	err := db.QueryRow(`
		SELECT a.id, a.user_id, a.title, a.description, a.price, a.created_at,
		       ac.year, ac.make, ac.model, ac.category, ac.subcategory
		FROM Ad a
		LEFT JOIN AdCar ac ON a.id = ac.ad_id
		WHERE a.id = ?`, id).Scan(&ad.ID, &ad.UserID, &ad.Title, &ad.Description, &ad.Price,
		&createdAt, &year, &make, &model, &category, &subcategory)
	if err != nil {
		return Ad{}, err
	}
	ad.CreatedAt, _ = time.Parse(time.RFC3339Nano, createdAt)
	if year.Valid {
		ad.Year = year.String
		ad.Make = make.String
		ad.Models = []string{model.String}
		ad.Category = category.String
		ad.SubCategory = subcategory.String
	}
	return ad, nil
}

// CreateAd creates a new ad
func CreateAd(ad Ad) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO Ad (user_id, title, description, price) VALUES (?, ?, ?, ?)`,
		ad.UserID, ad.Title, ad.Description, ad.Price)
	if err != nil {
		return err
	}

	id, _ := res.LastInsertId()
	ad.ID = int(id)

	if ad.Year != "" {
		_, err = tx.Exec(`INSERT INTO AdCar (ad_id, year, make, model, category, subcategory) VALUES (?, ?, ?, ?, ?, ?)`,
			ad.ID, ad.Year, ad.Make, ad.Models[0], ad.Category, ad.SubCategory)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// UpdateAd updates an existing ad
func UpdateAd(ad Ad) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE Ad SET description = ?, price = ?, subcategory_id = ? WHERE id = ?",
		ad.Description, ad.Price, ad.SubCategoryID, ad.ID)
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

// DeleteAd deletes an ad
func DeleteAd(id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM AdCar WHERE ad_id = ?`, id)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM Ad WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteAdsByUserID deletes all ads for a given user
func DeleteAdsByUserID(userID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get all ad IDs for the user
	rows, err := tx.Query(`SELECT id FROM Ad WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Delete each ad's car info
	for rows.Next() {
		var adID int
		if err := rows.Scan(&adID); err != nil {
			return err
		}
		_, err = tx.Exec(`DELETE FROM AdCar WHERE ad_id = ?`, adID)
		if err != nil {
			return err
		}
	}

	// Delete all ads for the user
	_, err = tx.Exec(`DELETE FROM Ad WHERE user_id = ?`, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetAdsByMakeModel returns all ads for a given make and model
func GetAdsByMakeModel(make, model string) ([]Ad, error) {
	// ... existing code ...
	return nil, nil
}

// GetAdsBySubCategory returns all ads for a given subcategory
func GetAdsBySubCategory(subCategoryID int) ([]Ad, error) {
	// ... existing code ...
	return nil, nil
}

// GetAdsByMakeModelYear returns all ads for a given make, model, and year
func GetAdsByMakeModelYear(make, model, year string) ([]Ad, error) {
	// ... existing code ...
	return nil, nil
}

// GetAdsByMakeModelYearEngine returns all ads for a given make, model, year, and engine
func GetAdsByMakeModelYearEngine(make, model, year, engine string) ([]Ad, error) {
	// ... existing code ...
	return nil, nil
}

// GetAdsByMakeModelYearEngineSubCategory returns all ads for a given make, model, year, engine, and subcategory
func GetAdsByMakeModelYearEngineSubCategory(make, model, year, engine string, subCategoryID int) ([]Ad, error) {
	// ... existing code ...
	return nil, nil
}

func ResurrectAd(adID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get ad data from archive
	var ad Ad
	err = tx.QueryRow(`SELECT id, description, price, created_at, subcategory_id, user_id 
		FROM AdDead WHERE id = ?`, adID).Scan(
		&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt, &ad.SubCategoryID, &ad.UserID)
	if err != nil {
		return err
	}

	// Restore ad
	_, err = tx.Exec(`INSERT INTO Ad (id, description, price, created_at, subcategory_id, user_id)
		VALUES (?, ?, ?, ?, ?, ?)`,
		ad.ID, ad.Description, ad.Price, ad.CreatedAt, ad.SubCategoryID, ad.UserID)
	if err != nil {
		return err
	}

	// Restore ad-car relationships
	_, err = tx.Exec(`INSERT INTO AdCar (ad_id, car_id)
		SELECT ad_id, car_id
		FROM AdCarDead WHERE ad_id = ?`, adID)
	if err != nil {
		return err
	}

	// Delete from archive tables
	_, err = tx.Exec(`DELETE FROM AdCarDead WHERE ad_id = ?`, adID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM AdDead WHERE id = ?`, adID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
