package handlers

import (
	"database/sql"
	"fmt"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/db"
)

// Helper to resolve location using Grok and upsert into Location table
func resolveAndStoreLocation(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}

	// Check if location already exists first to avoid expensive Grok API call
	var id int
	err := db.QueryRow("SELECT id FROM Location WHERE raw_text = $1", raw).Scan(&id)
	if err == nil {
		// Location already exists, return the ID
		return id, nil
	} else if err != sql.ErrNoRows {
		// Database error
		return 0, err
	}

	// Location doesn't exist, resolve using Grok API
	loc, err := ad.ResolveLocation(raw)
	if err != nil {
		return 0, err
	}

	// Insert new location into database
	// Use ON CONFLICT DO NOTHING to handle race conditions where another request might have inserted it
	var locationID int
	err = db.QueryRow(`INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (raw_text) DO NOTHING RETURNING id`,
		raw, loc.City, loc.AdminArea, loc.Country, loc.Latitude, loc.Longitude).Scan(&locationID)
	if err == nil {
		return locationID, nil
	}

	// If insert returned no rows (conflict), query for the existing ID
	if err == sql.ErrNoRows {
		err = db.QueryRow("SELECT id FROM Location WHERE raw_text = $1", raw).Scan(&id)
		if err != nil {
			return 0, err
		}
		return id, nil
	}

	// Some other error occurred
	return 0, err
}

// resolveLocationForFilter resolves a location text to lat/lon coordinates
// Uses the common resolveAndStoreLocation function, then retrieves coordinates
func resolveLocationForFilter(locationText string) (latitude, longitude float64, err error) {
	if locationText == "" {
		return 0, 0, fmt.Errorf("empty location text")
	}

	// Use common function to resolve and store location
	locationID, err := resolveAndStoreLocation(locationText)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve location: %w", err)
	}

	// Get lat/lon from the location ID
	lat, lon, err := ad.GetLatLon(locationID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get coordinates for location: %w", err)
	}

	return lat, lon, nil
}
