package geoapify

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

// Result is one forward-geocode hit from Geoapify.
type Result struct {
	City        string  `json:"city"`
	County      string  `json:"county"`
	State       string  `json:"state"`
	StateCode   string  `json:"state_code"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
}

type searchResponse struct {
	Results []Result `json:"results"`
}

// Geocode resolves free-form location text via Geoapify forward geocoding.
// Returns nil, nil when the API finds no results.
func Geocode(text string) (*Result, error) {
	apiKey := config.GeoapifyAPIKey
	if apiKey == "" {
		return nil, fmt.Errorf("GEOAPIFY_API_KEY environment variable not set")
	}

	u, err := url.Parse(config.GeoapifyGeocodeURL)
	if err != nil {
		return nil, fmt.Errorf("parse geoapify url: %w", err)
	}
	q := u.Query()
	q.Set("text", text)
	q.Set("format", "json")
	q.Set("limit", "1")
	q.Set("apiKey", apiKey)
	u.RawQuery = q.Encode()

	logger.Debug("Geoapify geocode request", "text", text)

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("call geoapify: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"geoapify returned status %d: %s",
			resp.StatusCode, string(body),
		)
	}

	var parsed searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decode geoapify response: %w", err)
	}
	if len(parsed.Results) == 0 {
		return nil, nil
	}
	return &parsed.Results[0], nil
}
