package ad

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Table name constants
const (
	TableAd            = "Ad"
	TableArchivedAd    = "ArchivedAd"
	TableArchivedAdCar = "ArchivedAdCar"
	TableAdCar         = "AdCar"
)

// AdStatus represents the status of an ad
type AdStatus string

const (
	StatusActive   AdStatus = "active"
	StatusArchived AdStatus = "archived"
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
	Flagged      bool       `json:"flagged"` // true if flagged by current user
}

// IsArchived returns true if the ad has been archived
func (a Ad) IsArchived() bool {
	return a.DeletionDate != nil
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

// getVehicleData retrieves vehicle information for an ad from the specified table
func getVehicleData(adID int, adCarTable string) (makeName string, years []string, models []string, engines []string) {
	query := fmt.Sprintf(`
		SELECT DISTINCT m.name, y.year, mo.name, e.name
		FROM %s ac
		JOIN Car c ON ac.car_id = c.id
		JOIN Make m ON c.make_id = m.id
		JOIN Year y ON c.year_id = y.id
		JOIN Model mo ON c.model_id = mo.id
		JOIN Engine e ON c.engine_id = e.id
		WHERE ac.ad_id = ?
		ORDER BY m.name, y.year, mo.name, e.name
	`, adCarTable)

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

// getAdVehicleData retrieves vehicle information for an active ad
func getAdVehicleData(adID int) (makeName string, years []string, models []string, engines []string) {
	return getVehicleData(adID, TableAdCar)
}

// getArchivedAdVehicleData retrieves vehicle information for an archived ad
func getArchivedAdVehicleData(adID int) (makeName string, years []string, models []string, engines []string) {
	return getVehicleData(adID, TableArchivedAdCar)
}

// GetAdByID retrieves an ad by ID from either active or archived tables
// Returns the ad, its status, and whether it was found
func GetAdByID(id int) (Ad, AdStatus, bool) {
	// Try active ads first
	ad, ok := GetAd(id)
	if ok {
		return ad, StatusActive, true
	}

	// Try archived ads
	archivedAd, ok := GetArchivedAd(id)
	if ok {
		return archivedAd, StatusArchived, true
	}

	return Ad{}, StatusActive, false
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

// GetAdsPage returns a page of ads for cursor-based pagination
func GetAdsPage(cursorID int, limit int) ([]Ad, bool) {
	rows, err := db.Query(`
		SELECT a.id, a.description, a.price, a.created_at, a.subcategory_id,
		       psc.name as subcategory,
		       CASE WHEN fa.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_flagged
		FROM Ad a
		LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
		LEFT JOIN FlaggedAd fa ON a.id = fa.ad_id
		WHERE a.id < ?
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT ?
	`, cursorID, limit+1)
	if err != nil {
		return nil, false
	}
	defer rows.Close()

	var ads []Ad
	for rows.Next() {
		var ad Ad
		var subcatID sql.NullInt64
		var subcategory sql.NullString
		var isFlagged int
		if err := rows.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt,
			&subcatID, &subcategory, &isFlagged); err != nil {
			continue
		}
		if subcatID.Valid {
			intVal := int(subcatID.Int64)
			ad.SubCategoryID = &intVal
		}
		if subcategory.Valid {
			ad.SubCategory = subcategory.String
		}
		ad.Flagged = isFlagged == 1
		// Get vehicle data for this ad
		ad.Make, ad.Years, ad.Models, ad.Engines = getAdVehicleData(ad.ID)
		ads = append(ads, ad)
	}

	hasMore := len(ads) > limit
	if hasMore {
		ads = ads[:limit]
	}

	return ads, hasMore
}

// GetFilteredAdsPageDB returns a page of filtered ads directly from the database
func GetFilteredAdsPageDB(query SearchQuery, cursor *SearchCursor, limit int, userID int) ([]Ad, bool, error) {
	// Check if we have any vehicle filters
	hasVehicleFilters := query.Make != "" || len(query.Years) > 0 || len(query.Models) > 0 || len(query.EngineSizes) > 0

	var sqlQuery string
	var args []interface{}

	if hasVehicleFilters {
		// Use JOIN-based query when we have vehicle filters
		sqlQuery = `
			SELECT DISTINCT a.id, a.description, a.price, a.created_at, a.subcategory_id,
			       psc.name as subcategory,
			       m.name as make_name, y.year, mo.name as model_name, e.name as engine_name,
			       CASE WHEN fa.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_flagged
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			JOIN AdCar ac ON a.id = ac.ad_id
			JOIN Car c ON ac.car_id = c.id
			JOIN Make m ON c.make_id = m.id
			JOIN Year y ON c.year_id = y.id
			JOIN Model mo ON c.model_id = mo.id
			JOIN Engine e ON c.engine_id = e.id
			LEFT JOIN FlaggedAd fa ON a.id = fa.ad_id AND fa.user_id = ?
			WHERE 1=1
		`
		args = append(args, userID)

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
			       NULL as make_name, NULL as year, NULL as model_name, NULL as engine_name,
			       CASE WHEN fa.ad_id IS NOT NULL THEN 1 ELSE 0 END as is_flagged
			FROM Ad a
			LEFT JOIN PartSubCategory psc ON a.subcategory_id = psc.id
			LEFT JOIN FlaggedAd fa ON a.id = fa.ad_id AND fa.user_id = ?
			WHERE 1=1
		`
		args = append(args, userID)
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
			isFlagged   int
		)

		if err := rows.Scan(&id, &description, &price, &createdAt,
			&subcatID, &subcategory, &makeName, &year, &modelName, &engineName, &isFlagged); err != nil {
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

		ad.Flagged = isFlagged == 1
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

// GetAllArchivedAds returns all archived ads in the system
func GetAllArchivedAds() ([]Ad, error) {
	rows, err := db.Query(`
		SELECT id, description, price, created_at, user_id, deletion_date
		FROM ArchivedAd
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
		if parsedTime, err := time.Parse(time.RFC3339Nano, deletionDate); err == nil {
			ad.DeletionDate = &parsedTime
		}
		// Populate vehicle info for each archived ad
		ad.Make, ad.Years, ad.Models, ad.Engines = getArchivedAdVehicleData(ad.ID)
		ads = append(ads, ad)
	}
	return ads, nil
}

// GetArchivedAd retrieves an archived ad by ID
func GetArchivedAd(id int) (Ad, bool) {
	row := db.QueryRow(`
		SELECT id, description, price, created_at, subcategory_id, user_id, deletion_date
		FROM ArchivedAd
		WHERE id = ?
	`, id)

	var ad Ad
	var deletionDate string
	if err := row.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt,
		&ad.SubCategoryID, &ad.UserID, &deletionDate); err != nil {
		return Ad{}, false
	}

	// Parse deletion date
	if parsedTime, err := time.Parse(time.RFC3339Nano, deletionDate); err == nil {
		ad.DeletionDate = &parsedTime
	}

	// Get vehicle data from ArchivedAdCar
	ad.Make, ad.Years, ad.Models, ad.Engines = getArchivedAdVehicleData(id)

	return ad, true
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

// ArchiveAd archives an ad
func ArchiveAd(id int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Archive the ad to ArchivedAd with deletion_date
	_, err = tx.Exec(`INSERT INTO ArchivedAd (id, description, price, created_at, subcategory_id, user_id, deletion_date)
		SELECT id, description, price, created_at, subcategory_id, user_id, ? FROM Ad WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}

	// Archive ad-car relationships to ArchivedAdCar
	_, err = tx.Exec(`INSERT INTO ArchivedAdCar (ad_id, car_id)
		SELECT ad_id, car_id FROM AdCar WHERE ad_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete from AdCar
	_, err = tx.Exec(`DELETE FROM AdCar WHERE ad_id = ?`, id)
	if err != nil {
		return err
	}

	// Delete from Ad
	_, err = tx.Exec(`DELETE FROM Ad WHERE id = ?`, id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RestoreAd moves an ad from the ArchivedAd table back to the active Ad table
func RestoreAd(adID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get ad data from archive
	var ad Ad
	err = tx.QueryRow(`SELECT id, description, price, created_at, subcategory_id, user_id 
		FROM ArchivedAd WHERE id = ?`, adID).Scan(
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
		FROM ArchivedAdCar WHERE ad_id = ?`, adID)
	if err != nil {
		return err
	}

	// Delete from archive tables
	_, err = tx.Exec(`DELETE FROM ArchivedAdCar WHERE ad_id = ?`, adID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM ArchivedAd WHERE id = ?`, adID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Flag/unflag and flagged ads logic

// FlagAd flags an ad for a user
func FlagAd(userID, adID int) error {
	_, err := db.Exec(`INSERT OR IGNORE INTO FlaggedAd (user_id, ad_id) VALUES (?, ?)`, userID, adID)
	return err
}

// UnflagAd removes a flag for an ad by a user
func UnflagAd(userID, adID int) error {
	_, err := db.Exec(`DELETE FROM FlaggedAd WHERE user_id = ? AND ad_id = ?`, userID, adID)
	return err
}

// IsAdFlaggedByUser checks if a user has flagged an ad
func IsAdFlaggedByUser(userID, adID int) (bool, error) {
	row := db.QueryRow(`SELECT 1 FROM FlaggedAd WHERE user_id = ? AND ad_id = ?`, userID, adID)
	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// GetFlaggedAdIDsByUser returns a list of ad IDs flagged by the user
func GetFlaggedAdIDsByUser(userID int) ([]int, error) {
	rows, err := db.Query(`SELECT ad_id FROM FlaggedAd WHERE user_id = ? ORDER BY flagged_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var adIDs []int
	for rows.Next() {
		var adID int
		if err := rows.Scan(&adID); err != nil {
			continue
		}
		adIDs = append(adIDs, adID)
	}
	return adIDs, nil
}

// GetAdsByIDs returns ads for a list of IDs (order preserved as much as possible)
func GetAdsByIDs(ids []int) ([]Ad, error) {
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
	query := `SELECT id, description, price, created_at, subcategory_id, user_id FROM Ad WHERE id IN (` + strings.Join(placeholders, ",") + `)`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	adMap := make(map[int]Ad)
	for rows.Next() {
		var ad Ad
		if err := rows.Scan(&ad.ID, &ad.Description, &ad.Price, &ad.CreatedAt, &ad.SubCategoryID, &ad.UserID); err != nil {
			continue
		}
		ad.Make, ad.Years, ad.Models, ad.Engines = getAdVehicleData(ad.ID)
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
