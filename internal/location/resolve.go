package location

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/service/geoapify"
)

// LocationResponse is the normalized shape stored in locations.
type LocationResponse struct {
	City      string
	AdminArea string
	Country   string
	Latitude  float64
	Longitude float64
}

func resolveWithGeoapify(text string) (*LocationResponse, error) {
	result, err := geoapify.Geocode(text)
	if err != nil {
		return nil, fmt.Errorf("resolve location with geoapify: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("geoapify returned no results")
	}
	return locationFromGeoapify(result), nil
}

func locationFromGeoapify(r *geoapify.Result) *LocationResponse {
	country := strings.ToUpper(strings.TrimSpace(r.CountryCode))
	admin := strings.TrimSpace(r.StateCode)
	if country != "US" && country != "CA" {
		admin = strings.TrimSpace(r.State)
	} else if admin == "" {
		admin = strings.TrimSpace(r.State)
	}
	city := strings.TrimSpace(r.City)
	if city == "" {
		city = strings.TrimSpace(r.County)
	}
	return &LocationResponse{
		City:      city,
		AdminArea: admin,
		Country:   country,
		Latitude:  r.Lat,
		Longitude: r.Lon,
	}
}

type resolvedLoc struct {
	id       int
	lat, lon float64
}

func resolveAndStore(text string) (resolvedLoc, bool, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return resolvedLoc{}, false, nil
	}

	var loc resolvedLoc
	err := db.QueryRow(
		`SELECT id, latitude, longitude FROM locations
		 WHERE lower(raw_text) = lower($1) LIMIT 1`,
		text,
	).Scan(&loc.id, &loc.lat, &loc.lon)
	if err == nil {
		return loc, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return resolvedLoc{}, false, fmt.Errorf("resolve location: %w", err)
	}

	resolved, err := resolveWithGeoapify(text)
	if err != nil {
		logger.Warn("resolve location: geoapify failed",
			"text", text, "error", err)
		return resolvedLoc{}, false, nil
	}

	_, err = db.Exec(
		`INSERT INTO locations
		 (raw_text, city, admin_area, country, latitude, longitude)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT(raw_text) DO NOTHING`,
		text, resolved.City, resolved.AdminArea, resolved.Country,
		resolved.Latitude, resolved.Longitude,
	)
	if err != nil {
		return resolvedLoc{}, false, fmt.Errorf("resolve location: %w", err)
	}

	err = db.QueryRow(
		`SELECT id, latitude, longitude FROM locations
		 WHERE lower(raw_text) = lower($1) LIMIT 1`,
		text,
	).Scan(&loc.id, &loc.lat, &loc.lon)
	if err != nil {
		return resolvedLoc{}, false, fmt.Errorf("resolve location: %w", err)
	}
	return loc, true, nil
}
