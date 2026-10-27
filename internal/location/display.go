package location

import (
	"fmt"
	"strings"

	"github.com/partspile/site/internal/db"
)

// DisplayText formats city/admin area with a country flag for UI display.
func DisplayText(city, adminArea, country string) string {
	if city == "" && adminArea == "" && country == "" {
		return ""
	}

	var locationText string
	if city != "" && adminArea != "" {
		locationText = city + ", " + adminArea
	} else if city != "" {
		locationText = city
	} else if adminArea != "" {
		locationText = adminArea
	}

	var flag string
	if len(country) == 2 {
		code := strings.ToUpper(country)
		flag = string(rune(int32(code[0])-'A'+0x1F1E6)) +
			string(rune(int32(code[1])-'A'+0x1F1E6))
	}

	return strings.TrimSpace(flag + " " + locationText)
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
