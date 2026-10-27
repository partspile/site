package cookie

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
)

func GetView(c *fiber.Ctx) string {
	return c.Cookies("view")
}

func SetView(c *fiber.Ctx, view int) {
	c.Cookie(&fiber.Cookie{
		Name:     "view",
		Value:    strconv.Itoa(view),
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
	})
}
