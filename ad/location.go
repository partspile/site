package ad

import (
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
	row := db.QueryRow("SELECT latitude, longitude FROM Location WHERE id = ?", id)
	err = row.Scan(&latitude, &longitude)
	return
}

// GetLatLonByRawText fetches latitude and longitude for a location by raw_text
func GetLatLonByRawText(rawText string) (latitude, longitude float64, err error) {
	row := db.QueryRow("SELECT latitude, longitude FROM Location WHERE raw_text = ?", rawText)
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
