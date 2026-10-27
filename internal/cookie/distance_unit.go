package cookie

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/location"
)

func GetDistanceUnit(c *fiber.Ctx) string {
	unit := c.Cookies("distance_unit")
	if unit == location.UnitMiles || unit == location.UnitKm {
		return unit
	}
	return location.DistanceUnitFromTimezone(c.Cookies("timezone"))
}

func SetDistanceUnit(c *fiber.Ctx, unit string) {
	if unit != location.UnitMiles && unit != location.UnitKm {
		unit = location.UnitMiles
	}
	c.Cookie(&fiber.Cookie{
		Name:     "distance_unit",
		Value:    unit,
		MaxAge:   30 * 24 * 60 * 60,
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
	})
}

func SetDistanceUnitForUser(c *fiber.Ctx, phoneE64 string) {
	unit := location.DistanceUnitFromTimezone(c.Cookies("timezone"))
	if phoneE64 != "" {
		unit = location.DistanceUnitFromPhone(phoneE64)
	}
	SetDistanceUnit(c, unit)
}

func ResetDistanceUnit(c *fiber.Ctx) {
	SetDistanceUnit(c, location.DistanceUnitFromTimezone(c.Cookies("timezone")))
}
