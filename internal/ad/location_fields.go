package ad

import (
	"strings"

	"github.com/partspile/site/internal/location"
)

func resolveLocationFields(locText string) (
	locationID *int, lat, lon *float64, err error,
) {
	if strings.TrimSpace(locText) == "" {
		return nil, nil, nil, nil
	}
	id, latitude, longitude, ok, err := location.FindLocation(locText)
	if err != nil || !ok {
		return nil, nil, nil, err
	}
	return &id, &latitude, &longitude, nil
}
