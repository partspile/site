package ad

import (
	"database/sql"
	"fmt"

	"github.com/parts-pile/site/db"
)

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
			part_subcategory_id, user_id, location_id, image_count, has_vector
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ad.AdCategoryID, ad.Title, ad.Description, ad.Price,
		ad.PartSubcategoryID, ad.UserID, ad.LocationID, ad.ImageCount, ad.HasVector)
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
func findVehicle(tx *sql.Tx, makeName, yearStr, modelName, engineName string, adCategoryID int) (int, error) {
	// Use the unified Vehicle table
	query := `
		SELECT v.id FROM Vehicle v
		JOIN Make m ON v.make_id = m.id
		LEFT JOIN Year y ON v.year_id = y.id
		JOIN Model mo ON v.model_id = mo.id
		LEFT JOIN Engine e ON v.engine_id = e.id
		WHERE m.name = ? AND mo.name = ? AND m.ad_category_id = ?
		AND (? IS NULL OR y.year = ?) AND (? IS NULL OR e.name = ?)
	`
	var vehicleID int
	var err error

	if yearStr == "" && engineName == "" {
		err = tx.QueryRow(query, makeName, modelName, adCategoryID, nil, nil, nil, nil).Scan(&vehicleID)
	} else if yearStr == "" {
		err = tx.QueryRow(query, makeName, modelName, adCategoryID, nil, nil, engineName, engineName).Scan(&vehicleID)
	} else if engineName == "" {
		err = tx.QueryRow(query, makeName, modelName, adCategoryID, yearStr, yearStr, nil, nil).Scan(&vehicleID)
	} else {
		err = tx.QueryRow(query, makeName, modelName, adCategoryID, yearStr, yearStr, engineName, engineName).Scan(&vehicleID)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("vehicle not found for make=%s, year=%s, model=%s, engine=%s",
				makeName, yearStr, modelName, engineName)
		}
		return 0, fmt.Errorf("error looking up vehicle: %w", err)
	}
	return vehicleID, nil
}

// addCarAssociations handles Cars: make, year, model, engine
func addCarAssociations(tx *sql.Tx, ad AdDetail) error {
	for _, yearStr := range ad.Years {
		for _, modelName := range ad.Models {
			for _, engineName := range ad.Engines {
				vehicleID, err := findVehicle(tx, ad.Make, yearStr, modelName, engineName, ad.AdCategoryID)
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
	for _, yearStr := range ad.Years {
		for _, modelName := range ad.Models {
			for _, engineName := range ad.Engines {
				vehicleID, err := findVehicle(tx, ad.Make, yearStr, modelName, engineName, ad.AdCategoryID)
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
	for _, yearStr := range ad.Years {
		for _, modelName := range ad.Models {
			vehicleID, err := findVehicle(tx, ad.Make, yearStr, modelName, "", ad.AdCategoryID)
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
	for _, modelName := range ad.Models {
		vehicleID, err := findVehicle(tx, ad.Make, "", modelName, "", ad.AdCategoryID)
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
