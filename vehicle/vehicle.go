package vehicle

import (
	"database/sql"
	"strconv"
	"strings"
	"time"
)

type Make struct {
	ID   int
	Name string
}

type Model struct {
	ID   int
	Name string
}

type Year struct {
	ID   int
	Year int
}

// ParentCompany represents a parent company of a make
type ParentCompany struct {
	ID      int
	Name    string
	Country string
}

var db *sql.DB
var (
	makesCache          []string
	yearsCache          = make(map[string][]string)
	allModelsCache      []string
	allEngineSizesCache []string
	yearRangeCache      []string
)

// InitDB sets the database connection for the vehicle package
func InitDB(database *sql.DB) {
	db = database
}

func GetMakes() []string {
	if makesCache != nil {
		return makesCache
	}
	rows, err := db.Query("SELECT name FROM Make ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var makes []string
	for rows.Next() {
		var make string
		rows.Scan(&make)
		makes = append(makes, make)
	}
	makesCache = makes
	return makes
}

func GetAllMakes() ([]Make, error) {
	rows, err := db.Query("SELECT id, name FROM Make ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var makes []Make
	for rows.Next() {
		var make Make
		if err := rows.Scan(&make.ID, &make.Name); err != nil {
			return nil, err
		}
		makes = append(makes, make)
	}
	return makes, nil
}

func GetAllYears() ([]Year, error) {
	rows, err := db.Query("SELECT id, year FROM Year ORDER BY year")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []Year
	for rows.Next() {
		var year Year
		if err := rows.Scan(&year.ID, &year.Year); err != nil {
			return nil, err
		}
		years = append(years, year)
	}
	return years, nil
}

func GetYears(makeName string) []string {
	if years, ok := yearsCache[makeName]; ok {
		return years
	}
	query := `SELECT DISTINCT Year.year FROM Car
	JOIN Make ON Car.make_id = Make.id
	JOIN Year ON Car.year_id = Year.id
	WHERE Make.name = ? ORDER BY Year.year`
	rows, err := db.Query(query, makeName)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var years []string
	for rows.Next() {
		var year int
		rows.Scan(&year)
		years = append(years, strconv.Itoa(year))
	}
	yearsCache[makeName] = years
	return years
}

func GetAllModels() []string {
	if allModelsCache != nil {
		return allModelsCache
	}
	rows, err := db.Query("SELECT DISTINCT name FROM Model ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var models []string
	for rows.Next() {
		var model string
		rows.Scan(&model)
		models = append(models, model)
	}
	allModelsCache = models
	return models
}

func GetAllModelsWithID() ([]Model, error) {
	rows, err := db.Query("SELECT id, name FROM Model ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []Model
	for rows.Next() {
		var model Model
		if err := rows.Scan(&model.ID, &model.Name); err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

func GetModelsWithAvailability(makeName string, years []string) map[string]bool {
	allModels := make(map[string]bool)
	availableInAllYears := make(map[string]bool)

	if len(years) == 0 {
		return allModels
	}

	// Build placeholders for IN clause
	yearPlaceholders := make([]string, len(years))
	args := make([]interface{}, len(years)+1)
	args[0] = makeName
	for i, year := range years {
		yearPlaceholders[i] = "?"
		args[i+1] = year
	}
	query := `SELECT DISTINCT Model.name FROM Car
	JOIN Make ON Car.make_id = Make.id
	JOIN Model ON Car.model_id = Model.id
	JOIN Year ON Car.year_id = Year.id
	WHERE Make.name = ? AND Year.year IN (` + strings.Join(yearPlaceholders, ",") + `) ORDER BY Model.name`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var model string
		rows.Scan(&model)
		allModels[model] = true
		if _, exists := availableInAllYears[model]; !exists {
			availableInAllYears[model] = true
		}
	}

	// Second pass: check if each model exists in all selected years
	for model := range allModels {
		for _, year := range years {
			var count int
			err := db.QueryRow(`SELECT COUNT(*) FROM Car
			JOIN Make ON Car.make_id = Make.id
			JOIN Model ON Car.model_id = Model.id
			JOIN Year ON Car.year_id = Year.id
			WHERE Make.name = ? AND Model.name = ? AND Year.year = ?`, makeName, model, year).Scan(&count)
			if err != nil {
				return nil
			}
			if count == 0 {
				availableInAllYears[model] = false
				break
			}
		}
	}
	return availableInAllYears
}

func GetEnginesWithAvailability(makeName string, years []string, models []string) map[string]bool {
	allEngines := make(map[string]bool)
	availableInAllCombos := make(map[string]bool)

	if len(years) == 0 || len(models) == 0 {
		return allEngines
	}

	// Build placeholders for IN clauses
	yearPlaceholders := make([]string, len(years))
	modelPlaceholders := make([]string, len(models))
	args := make([]interface{}, 0, 1+len(years)+len(models))
	args = append(args, makeName)
	for i, year := range years {
		yearPlaceholders[i] = "?"
		args = append(args, year)
	}
	for i, model := range models {
		modelPlaceholders[i] = "?"
		args = append(args, model)
	}
	query := `SELECT DISTINCT Engine.name FROM Car
	JOIN Make ON Car.make_id = Make.id
	JOIN Model ON Car.model_id = Model.id
	JOIN Year ON Car.year_id = Year.id
	JOIN Engine ON Car.engine_id = Engine.id
	WHERE Make.name = ? AND Year.year IN (` + strings.Join(yearPlaceholders, ",") + `) AND Model.name IN (` + strings.Join(modelPlaceholders, ",") + `) ORDER BY Engine.name`
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var engine string
		rows.Scan(&engine)
		allEngines[engine] = true
		if _, exists := availableInAllCombos[engine]; !exists {
			availableInAllCombos[engine] = true
		}
	}

	// Second pass: check if each engine exists for all selected year-model combinations
	for engine := range allEngines {
		for _, year := range years {
			for _, model := range models {
				var count int
				err := db.QueryRow(`SELECT COUNT(*) FROM Car
				JOIN Make ON Car.make_id = Make.id
				JOIN Model ON Car.model_id = Model.id
				JOIN Year ON Car.year_id = Year.id
				JOIN Engine ON Car.engine_id = Engine.id
				WHERE Make.name = ? AND Model.name = ? AND Year.year = ? AND Engine.name = ?`, makeName, model, year, engine).Scan(&count)
				if err != nil {
					return nil
				}
				if count == 0 {
					availableInAllCombos[engine] = false
					break
				}
			}
			if !availableInAllCombos[engine] {
				break
			}
		}
	}
	return availableInAllCombos
}

func GetAllEngineSizes() []string {
	if allEngineSizesCache != nil {
		return allEngineSizesCache
	}
	rows, err := db.Query("SELECT DISTINCT name FROM Engine ORDER BY name")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var engines []string
	for rows.Next() {
		var engine string
		rows.Scan(&engine)
		engines = append(engines, engine)
	}
	allEngineSizesCache = engines
	return engines
}

func GetYearRange() []string {
	if yearRangeCache != nil {
		return yearRangeCache
	}
	currentYear := time.Now().Year() + 1
	years := make([]string, 0, currentYear-1900+1)
	for year := 1900; year <= currentYear; year++ {
		years = append(years, strconv.Itoa(year))
	}
	yearRangeCache = years
	return years
}

// GetAllParentCompanies returns all parent companies in the system
func GetAllParentCompanies() ([]ParentCompany, error) {
	rows, err := db.Query("SELECT id, name, country FROM ParentCompany ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var pcs []ParentCompany
	for rows.Next() {
		var pc ParentCompany
		if err := rows.Scan(&pc.ID, &pc.Name, &pc.Country); err != nil {
			return nil, err
		}
		pcs = append(pcs, pc)
	}
	return pcs, nil
}

// AddParentCompany inserts a new parent company
func AddParentCompany(name, country string) (int, error) {
	res, err := db.Exec("INSERT INTO ParentCompany (name, country) VALUES (?, ?)", name, country)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	return int(id), err
}

// UpdateParentCompanyCountry updates the country for a parent company
func UpdateParentCompanyCountry(id int, country string) error {
	_, err := db.Exec("UPDATE ParentCompany SET country = ? WHERE id = ?", country, id)
	return err
}

func GetDB() *sql.DB {
	return db
}
