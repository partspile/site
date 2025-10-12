package ad

import (
	"math"

	"github.com/parts-pile/site/db"
)

// GetLocation fetches a Location by its ID
func GetLocation(id int) (city, adminArea, country, raw string, latitude, longitude float64, err error) {
	row := db.QueryRow("SELECT city, admin_area, country, raw_text, latitude, longitude FROM Location WHERE id = ?", id)
	err = row.Scan(&city, &adminArea, &country, &raw, &latitude, &longitude)
	return
}

// CalculateExtent calculates the geographic bounding box for a list of ads
// Returns minLat, maxLat, minLon, maxLon, and a boolean indicating if any
// valid locations were found
func CalculateExtent(ads []Ad) (
	minLat, maxLat, minLon, maxLon float64, found bool,
) {
	minLat = math.MaxFloat64
	maxLat = -math.MaxFloat64
	minLon = math.MaxFloat64
	maxLon = -math.MaxFloat64

	for _, ad := range ads {
		if ad.Latitude.Valid && ad.Longitude.Valid {
			found = true
			lat := ad.Latitude.Float64
			lon := ad.Longitude.Float64

			if lat < minLat {
				minLat = lat
			}
			if lat > maxLat {
				maxLat = lat
			}
			if lon < minLon {
				minLon = lon
			}
			if lon > maxLon {
				maxLon = lon
			}
		}
	}

	return minLat, maxLat, minLon, maxLon, found
}
