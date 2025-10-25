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

// addCarAssociations handles Cars: make, year, model, engine
func addCarAssociations(tx *sql.Tx, ad AdDetail) error {
	for _, yearStr := range ad.Years {
		for _, modelName := range ad.Models {
			for _, engineName := range ad.Engines {
				vehicleID, err := findVehicleID(tx, "Car", ad.Make, yearStr,
					modelName, engineName)
				if err != nil {
					return err
				}
				if err := insertVehicleAssociation(tx, ad.ID, "AdCar",
					"car_id", vehicleID); err != nil {
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
				vehicleID, err := findVehicleID(tx, "Motorcycle", ad.Make, yearStr,
					modelName, engineName)
				if err != nil {
					return err
				}
				if err := insertVehicleAssociation(tx, ad.ID, "AdMotorcycle",
					"motorcycle_id", vehicleID); err != nil {
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
			vehicleID, err := findAgVehicleID(tx, "Ag", ad.Make, yearStr,
				modelName)
			if err != nil {
				return err
			}
			if err := insertVehicleAssociation(tx, ad.ID, "AdAg",
				"ag_id", vehicleID); err != nil {
				return err
			}
		}
	}
	return nil
}

// addBicycleAssociations handles Bicycles: make, model (no year, no engine)
func addBicycleAssociations(tx *sql.Tx, ad AdDetail) error {
	for _, modelName := range ad.Models {
		vehicleID, err := findBicycleVehicleID(tx, "Bicycle", ad.Make,
			modelName)
		if err != nil {
			return err
		}
		if err := insertVehicleAssociation(tx, ad.ID, "AdBicycle",
			"bicycle_id", vehicleID); err != nil {
			return err
		}
	}
	return nil
}

// findVehicleID finds a vehicle ID for Cars/Motorcycles with make, year, model, engine
func findVehicleID(tx *sql.Tx, vehicleTable, makeName, yearStr, modelName,
	engineName string) (int, error) {
	var vehicleID int
	err := tx.QueryRow(fmt.Sprintf(`
		SELECT v.id FROM %s v
		JOIN Make m ON v.make_id = m.id
		JOIN Year y ON v.year_id = y.id
		JOIN Model mo ON v.model_id = mo.id
		JOIN Engine e ON v.engine_id = e.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ? AND e.name = ?
	`, vehicleTable), makeName, yearStr, modelName, engineName).Scan(&vehicleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("vehicle not found for make=%s, year=%s, model=%s, engine=%s",
				makeName, yearStr, modelName, engineName)
		}
		return 0, fmt.Errorf("error looking up vehicle: %w", err)
	}
	return vehicleID, nil
}

// findAgVehicleID finds a vehicle ID for Agricultural Equipment with make, year, model
func findAgVehicleID(tx *sql.Tx, vehicleTable, makeName, yearStr, modelName string) (int, error) {
	var vehicleID int
	err := tx.QueryRow(fmt.Sprintf(`
		SELECT v.id FROM %s v
		JOIN Make m ON v.make_id = m.id
		JOIN Year y ON v.year_id = y.id
		JOIN Model mo ON v.model_id = mo.id
		WHERE m.name = ? AND y.year = ? AND mo.name = ?
	`, vehicleTable), makeName, yearStr, modelName).Scan(&vehicleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("vehicle not found for make=%s, year=%s, model=%s",
				makeName, yearStr, modelName)
		}
		return 0, fmt.Errorf("error looking up vehicle: %w", err)
	}
	return vehicleID, nil
}

// findBicycleVehicleID finds a vehicle ID for Bicycles with make, model
func findBicycleVehicleID(tx *sql.Tx, vehicleTable, makeName, modelName string) (int, error) {
	var vehicleID int
	err := tx.QueryRow(fmt.Sprintf(`
		SELECT v.id FROM %s v
		JOIN Make m ON v.make_id = m.id
		JOIN Model mo ON v.model_id = mo.id
		WHERE m.name = ? AND mo.name = ?
	`, vehicleTable), makeName, modelName).Scan(&vehicleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("vehicle not found for make=%s, model=%s",
				makeName, modelName)
		}
		return 0, fmt.Errorf("error looking up vehicle: %w", err)
	}
	return vehicleID, nil
}

// insertVehicleAssociation inserts the ad-vehicle association record
func insertVehicleAssociation(tx *sql.Tx, adID int, associationTable, columnName string,
	vehicleID int) error {
	_, err := tx.Exec(fmt.Sprintf("INSERT OR IGNORE INTO %s (ad_id, %s) VALUES (?, ?)",
		associationTable, columnName), adID, vehicleID)
	if err != nil {
		return fmt.Errorf("failed to insert %s association: %w", associationTable, err)
	}
	return nil
}
