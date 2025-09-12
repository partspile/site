package handlers

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
)

func getCookieLastView(c *fiber.Ctx) string {
	return c.Cookies("last_view", "list") // default to list
}

func saveCookieLastView(c *fiber.Ctx, view string) {
	c.Cookie(&fiber.Cookie{
		Name:     "last_view",
		Value:    view,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
}

func getCookieMapBounds(c *fiber.Ctx) (*ui.GeoBounds, bool) {
	return parseBounds(
		c.Cookies("map_min_lat", ""),
		c.Cookies("map_max_lat", ""),
		c.Cookies("map_min_lon", ""),
		c.Cookies("map_max_lon", ""))
}

func parseBounds(minLatStr, maxLatStr, minLonStr, maxLonStr string) (*ui.GeoBounds, bool) {
	if minLatStr == "" || maxLatStr == "" || minLonStr == "" || maxLonStr == "" {
		return nil, false
	}

	minLat, err1 := strconv.ParseFloat(minLatStr, 64)
	maxLat, err2 := strconv.ParseFloat(maxLatStr, 64)
	minLon, err3 := strconv.ParseFloat(minLonStr, 64)
	maxLon, err4 := strconv.ParseFloat(maxLonStr, 64)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil, false
	}

	// Bound coordinates to valid ranges
	minLat = boundLatitude(minLat)
	maxLat = boundLatitude(maxLat)
	minLon = boundLongitude(minLon)
	maxLon = boundLongitude(maxLon)

	// Ensure min < max
	if minLat > maxLat {
		minLat, maxLat = maxLat, minLat
	}
	if minLon > maxLon {
		minLon, maxLon = maxLon, minLon
	}

	return &ui.GeoBounds{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}, true
}

func boundLatitude(lat float64) float64 {
	if lat < -90 {
		return -90
	}
	if lat > 90 {
		return 90
	}
	return lat
}

func boundLongitude(lon float64) float64 {
	if lon < -180 {
		return -180
	}
	if lon > 180 {
		return 180
	}
	return lon
}

func saveCookieMapBounds(c *fiber.Ctx, bounds *ui.GeoBounds) {
	c.Cookie(&fiber.Cookie{
		Name:     "map_min_lat",
		Value:    strconv.FormatFloat(bounds.MinLat, 'f', -1, 64),
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "map_max_lat",
		Value:    strconv.FormatFloat(bounds.MaxLat, 'f', -1, 64),
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "map_min_lon",
		Value:    strconv.FormatFloat(bounds.MinLon, 'f', -1, 64),
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
	c.Cookie(&fiber.Cookie{
		Name:     "map_max_lon",
		Value:    strconv.FormatFloat(bounds.MaxLon, 'f', -1, 64),
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
}
