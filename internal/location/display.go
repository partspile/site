package location

import (
	"fmt"
	"strings"

	"github.com/partspile/site/internal/db"
)

// DisplayText formats city and admin area for UI display.
// country is accepted so callers can pass the stored record; it is not shown.
func DisplayText(city, adminArea, country string) string {
	if city != "" && adminArea != "" {
		return city + ", " + adminArea
	}
	if city != "" {
		return city
	}
	return adminArea
}

// DisplayTextForInput resolves raw user input and returns formatted display text.
func DisplayTextForInput(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false, nil
	}

	id, ok, err := FindLocationID(raw)
	if err != nil {
		return "", false, err
	}
	if !ok {
		return "", false, nil
	}

	var city, adminArea, country string
	err = db.QueryRow(
		`SELECT city, admin_area, country FROM locations WHERE id = $1`, id,
	).Scan(&city, &adminArea, &country)
	if err != nil {
		return "", false, fmt.Errorf("lookup location display: %w", err)
	}
	return DisplayText(city, adminArea, country), true, nil
}
