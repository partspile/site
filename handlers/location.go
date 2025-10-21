package handlers

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
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
	res, err := db.Exec("INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES (?, ?, ?, ?, ?, ?)",
		raw, loc.City, loc.AdminArea, loc.Country, loc.Latitude, loc.Longitude)
	if err != nil {
		return 0, err
	}
	lastID, _ := res.LastInsertId()
	return int(lastID), nil
}

// resolveLocationForFilter resolves a location text to lat/lon coordinates
// Checks database first, then uses Grok if not found, but doesn't store the result
func resolveLocationForFilter(locationText string) (latitude, longitude float64, err error) {
	if locationText == "" {
		return 0, 0, fmt.Errorf("empty location text")
	}

	// First try to find existing location in database
	lat, lon, err := ad.GetLatLonByRawText(locationText)
	if err != nil {
		return 0, 0, fmt.Errorf("database error looking up location: %w", err)
	}
	found := lat != 0 || lon != 0
	if found {
		return lat, lon, nil
	}

	// If not found in database, use Grok to resolve it but don't store
	loc, err := ad.ResolveLocation(locationText)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve location: %w", err)
	}

	return *loc.Latitude, *loc.Longitude, nil
}

// getLocation gets the timezone location from context
func getLocation(c *fiber.Ctx) *time.Location {
	loc, _ := time.LoadLocation(c.Get("X-Timezone"))
	return loc
}
