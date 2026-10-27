package cookie

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
)

func SetJWT(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   24 * 60 * 60, // 24 hours
	})
}

func ClearJWT(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
		MaxAge:   -1,
	})
}

func GetJWT(c *fiber.Ctx) string {
	return c.Cookies("auth_token")
}
