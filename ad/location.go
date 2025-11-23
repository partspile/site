package ad

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/grok"
)

// GetLatLon fetches latitude and longitude for a location by ID
func GetLatLon(id int) (latitude, longitude float64, err error) {
	if id == 0 {
		return
	}
	row := db.QueryRow("SELECT latitude, longitude FROM Location WHERE id = $1", id)
	err = row.Scan(&latitude, &longitude)
	return
}

// GetLatLonByRawText fetches latitude and longitude for a location by raw_text
func GetLatLonByRawText(rawText string) (latitude, longitude float64, err error) {
	row := db.QueryRow("SELECT latitude, longitude FROM Location WHERE raw_text = $1", rawText)
	err = row.Scan(&latitude, &longitude)
	return
}

// locationResolverPrompt is the system prompt for location resolution
const locationResolverPrompt = `You are a location resolver for an auto parts website.
Given a user input (which may be a address, city, zip code, or country),
return a JSON object with the best guess for city, admin_area (state,
province, or region), country, latitude, and longitude. The country field 
must be a 2-letter ISO country code (e.g., "US" for United States, "CA" 
for Canada, "GB" for United Kingdom). For US and Canada, the admin_area 
field must be the official 2-letter code (e.g., "OR" for Oregon, "NY" 
for New York, "BC" for British Columbia, "ON" for Ontario). For all 
other countries, use the full name for admin_area. Latitude and longitude 
should be decimal degrees (positive for North/East, negative for South/West).
If a field is unknown, leave it blank or null.
Example input: "97333" -> {"city": "Corvallis", "admin_area": "OR",
"country": "US", "latitude": 44.5646, "longitude": -123.2620}`

// LocationResponse represents the JSON response from Grok location resolution
type LocationResponse struct {
	City      string   `json:"city"`
	AdminArea string   `json:"admin_area"`
	Country   string   `json:"country"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
}

// ResolveLocation calls Grok to resolve a location and returns the parsed response
func ResolveLocation(locationText string) (*LocationResponse, error) {
	resp, err := grok.CallGrok(locationResolverPrompt, locationText)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve location with Grok: %w", err)
	}

	var loc LocationResponse
	err = json.Unmarshal([]byte(resp), &loc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Grok response: %w", err)
	}

	// Validate that coordinates were resolved
	if loc.Latitude == nil || loc.Longitude == nil {
		return nil, fmt.Errorf("grok could not resolve coordinates for location")
	}

	return &loc, nil
}

// ResolveAndStoreLocation resolves a location using Grok and upserts into Location table
func ResolveAndStoreLocation(raw string) (int, error) {
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
	loc, err := ResolveLocation(raw)
	if err != nil {
		return 0, err
	}

	// Insert new location into database
	// Use ON CONFLICT DO NOTHING to handle race conditions where another request might have inserted it
	var locationID int
	err = db.QueryRow(`INSERT INTO Location (raw_text, city, admin_area, country, latitude, longitude) VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (raw_text) DO NOTHING RETURNING id`,
		raw, loc.City, loc.AdminArea, loc.Country, *loc.Latitude, *loc.Longitude).Scan(&locationID)
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

// ResolveLocationToCoords resolves a location text to lat/lon coordinates
// Uses ResolveAndStoreLocation to resolve and store the location, then retrieves coordinates
func ResolveLocationToCoords(locationText string) (latitude, longitude float64, err error) {
	if locationText == "" {
		return 0, 0, fmt.Errorf("empty location text")
	}

	// Use common function to resolve and store location
	locationID, err := ResolveAndStoreLocation(locationText)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to resolve location: %w", err)
	}

	// Get lat/lon from the location ID
	lat, lon, err := GetLatLon(locationID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get coordinates for location: %w", err)
	}

	return lat, lon, nil
}
