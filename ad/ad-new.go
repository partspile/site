package ad

import (
	"database/sql"
	"fmt"

	"github.com/parts-pile/site/db"
)

// getPartSubcategoryID looks up the PartSubCategory ID from category and subcategory names
func getPartSubcategoryID(tx *sql.Tx, adCategoryID int, categoryName, subcategoryName string) (*int, error) {
	if categoryName == "" || subcategoryName == "" {
		return nil, nil
	}

	query := `
		SELECT psc.id
		FROM PartSubCategory psc
		JOIN PartCategory pc ON psc.part_category_id = pc.id
		WHERE pc.name = ? AND psc.name = ? AND pc.ad_category_id = ?
	`
	var id int
	err := tx.QueryRow(query, categoryName, subcategoryName, adCategoryID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("part subcategory not found: category=%s, subcategory=%s", categoryName, subcategoryName)
		}
		return nil, fmt.Errorf("failed to lookup part subcategory: %w", err)
	}
	return &id, nil
}

// AddAd creates a new ad in the database and returns the ad ID
func AddAd(ad AdDetail) (int, error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		INSERT INTO Ad (
			ad_category_id, title, description, price, 
			user_id, location_id, image_count, has_vector
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		ad.AdCategoryID, ad.Title, ad.Description, ad.Price,
		ad.UserID, ad.LocationID, ad.ImageCount, ad.HasVector)
	if err != nil {
		return 0, fmt.Errorf("failed to insert ad: %w", err)
	}
	adID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	ad.ID = int(adID)

	if err := addAdVehicleAssociations(tx, ad); err != nil {
		return 0, fmt.Errorf("failed to add vehicle associations: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return int(adID), nil
}

// addAdVehicleAssociations creates the normalized vehicle associations for an ad
func addAdVehicleAssociations(tx *sql.Tx, ad AdDetail) error {
	// Handle different vehicle schemas based on ad category
	switch ad.AdCategoryID {
	case AdCategoryCar, AdCategoryCarPart:
		return addCarAssociations(tx, ad)
	case AdCategoryMotorcycle, AdCategoryMotorcyclePart:
		return addMotorcycleAssociations(tx, ad)
	case AdCategoryAg, AdCategoryAgPart:
		return addAgAssociations(tx, ad)
	case AdCategoryBicycle, AdCategoryBicyclePart:
		return addBicycleAssociations(tx, ad)
	default:
		return nil
	}
}

// findVehicle finds a vehicle ID from the unified Vehicle table
func findVehicle(tx *sql.Tx, makeName, yearStr, modelName, engineName string, adCategoryID int, partSubcategoryID *int) (int, error) {
	// Use the unified Vehicle table
	var query string
	var vehicleID int
	var err error

	// Build query with appropriate part_subcategory_id check
	if partSubcategoryID != nil {
		query = `
			SELECT v.id FROM Vehicle v
			LEFT JOIN Make m ON v.make_id = m.id
			LEFT JOIN Year y ON v.year_id = y.id
			LEFT JOIN Model mo ON v.model_id = mo.id
			LEFT JOIN Engine e ON v.engine_id = e.id
			WHERE v.ad_category_id = ?
			AND (? IS NULL OR m.name = ?) AND (? IS NULL OR mo.name = ?)
			AND (? IS NULL OR y.year = ?) AND (? IS NULL OR e.name = ?)
			AND v.part_subcategory_id = ?
		`
	} else {
		query = `
			SELECT v.id FROM Vehicle v
			LEFT JOIN Make m ON v.make_id = m.id
			LEFT JOIN Year y ON v.year_id = y.id
			LEFT JOIN Model mo ON v.model_id = mo.id
			LEFT JOIN Engine e ON v.engine_id = e.id
			WHERE v.ad_category_id = ?
			AND (? IS NULL OR m.name = ?) AND (? IS NULL OR mo.name = ?)
			AND (? IS NULL OR y.year = ?) AND (? IS NULL OR e.name = ?)
			AND v.part_subcategory_id IS NULL
		`
	}

	// Build args: category, make (nullable), make name, model (nullable), model name, year (nullable), year value, engine (nullable), engine name, part_subcategory_id
	var makeArg interface{} = makeName
	if makeName == "" {
		makeArg = nil
	}
	var modelArg interface{} = modelName
	if modelName == "" {
		modelArg = nil
	}
	var yearArg interface{} = yearStr
	if yearStr == "" {
		yearArg = nil
	}
	var engineArg interface{} = engineName
	if engineName == "" {
		engineArg = nil
	}

	if partSubcategoryID != nil {
		err = tx.QueryRow(query, adCategoryID, makeArg, makeArg, modelArg, modelArg, yearArg, yearArg, engineArg, engineArg, *partSubcategoryID).Scan(&vehicleID)
	} else {
		err = tx.QueryRow(query, adCategoryID, makeArg, makeArg, modelArg, modelArg, yearArg, yearArg, engineArg, engineArg).Scan(&vehicleID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			partSubcatStr := "NULL"
			if partSubcategoryID != nil {
				partSubcatStr = fmt.Sprintf("%d", *partSubcategoryID)
			}
			return 0, fmt.Errorf("vehicle not found for make=%s, year=%s, model=%s, engine=%s, part_subcategory_id=%s",
				makeName, yearStr, modelName, engineName, partSubcatStr)
		}
		return 0, fmt.Errorf("error looking up vehicle: %w", err)
	}
	return vehicleID, nil
}

// addCarAssociations handles Cars: make, year, model, engine
func addCarAssociations(tx *sql.Tx, ad AdDetail) error {
	var partSubcatID *int
	var err error
	if IsPartCategory(ad.AdCategoryID) && ad.PartCategory != "" && ad.PartSubcategory != "" {
		partSubcatID, err = getPartSubcategoryID(tx, ad.AdCategoryID, ad.PartCategory, ad.PartSubcategory)
		if err != nil {
			return err
		}
	}
	for _, yearStr := range ad.Years {
		for _, modelName := range ad.Models {
			for _, engineName := range ad.Engines {
				vehicleID, err := findVehicle(tx, ad.Make, yearStr, modelName, engineName, ad.AdCategoryID, partSubcatID)
				if err != nil {
					return err
				}
				if err := insertVehicleAssociation(tx, ad.ID, vehicleID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// addMotorcycleAssociations handles Motorcycles: make, year, model, engine
func addMotorcycleAssociations(tx *sql.Tx, ad AdDetail) error {
	var partSubcatID *int
	var err error
	if IsPartCategory(ad.AdCategoryID) && ad.PartCategory != "" && ad.PartSubcategory != "" {
		partSubcatID, err = getPartSubcategoryID(tx, ad.AdCategoryID, ad.PartCategory, ad.PartSubcategory)
		if err != nil {
			return err
		}
	}
	for _, yearStr := range ad.Years {
		for _, modelName := range ad.Models {
			for _, engineName := range ad.Engines {
				vehicleID, err := findVehicle(tx, ad.Make, yearStr, modelName, engineName, ad.AdCategoryID, partSubcatID)
				if err != nil {
					return err
				}
				if err := insertVehicleAssociation(tx, ad.ID, vehicleID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// addAgAssociations handles Agricultural Equipment: make, year, model (no engine)
func addAgAssociations(tx *sql.Tx, ad AdDetail) error {
	var partSubcatID *int
	var err error
	if IsPartCategory(ad.AdCategoryID) && ad.PartCategory != "" && ad.PartSubcategory != "" {
		partSubcatID, err = getPartSubcategoryID(tx, ad.AdCategoryID, ad.PartCategory, ad.PartSubcategory)
		if err != nil {
			return err
		}
	}
	for _, yearStr := range ad.Years {
		for _, modelName := range ad.Models {
			vehicleID, err := findVehicle(tx, ad.Make, yearStr, modelName, "", ad.AdCategoryID, partSubcatID)
			if err != nil {
				return err
			}
			if err := insertVehicleAssociation(tx, ad.ID, vehicleID); err != nil {
				return err
			}
		}
	}
	return nil
}

// addBicycleAssociations handles Bicycles: make, model (no year, no engine)
func addBicycleAssociations(tx *sql.Tx, ad AdDetail) error {
	var partSubcatID *int
	var err error
	if IsPartCategory(ad.AdCategoryID) && ad.PartCategory != "" && ad.PartSubcategory != "" {
		partSubcatID, err = getPartSubcategoryID(tx, ad.AdCategoryID, ad.PartCategory, ad.PartSubcategory)
		if err != nil {
			return err
		}
	}
	for _, modelName := range ad.Models {
		vehicleID, err := findVehicle(tx, ad.Make, "", modelName, "", ad.AdCategoryID, partSubcatID)
		if err != nil {
			return err
		}
		if err := insertVehicleAssociation(tx, ad.ID, vehicleID); err != nil {
			return err
		}
	}
	return nil
}

// insertVehicleAssociation inserts the ad-vehicle association record
func insertVehicleAssociation(tx *sql.Tx, adID int, vehicleID int) error {
	_, err := tx.Exec("INSERT OR IGNORE INTO AdVehicle (ad_id, vehicle_id) VALUES (?, ?)",
		adID, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to insert AdVehicle association: %w", err)
	}
	return nil
}
