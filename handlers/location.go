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
	err := db.QueryRow("SELECT id FROM Location WHERE raw_text = ?", raw).Scan(&id)
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
	// Use INSERT OR IGNORE to handle race conditions where another request might have inserted it
	res, err := db.Exec("INSERT OR IGNORE INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)",
		raw, loc.City, loc.AdminArea, loc.Country, loc.Latitude, loc.Longitude)
	if err != nil {
		return 0, err
	}

	// Check if insert actually created a row
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 {
		// We inserted it, get the new ID
		lastID, _ := res.LastInsertId()
		return int(lastID), nil
	}

	// Another request inserted it first (race condition), query for the ID
	err = db.QueryRow("SELECT id FROM Location WHERE raw_text = ?", raw).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
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
