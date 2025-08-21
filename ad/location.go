package ad

import (
	"github.com/parts-pile/site/db"
)

// GetLocation fetches a Location by its ID
func GetLocation(id int) (city, adminArea, country, raw string, latitude, longitude float64, err error) {
	row := db.QueryRow("SELECT city, admin_area, country, raw_text, latitude, longitude FROM Location WHERE id = ?", id)
	err = row.Scan(&city, &adminArea, &country, &raw, &latitude, &longitude)
	return
}
