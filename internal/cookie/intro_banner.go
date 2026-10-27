package cookie

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/config"
)

const introBannerCookie = "intro_banner_dismissed"

func IsIntroBannerDismissed(c *fiber.Ctx) bool {
	return c.Cookies(introBannerCookie) != ""
}

func SetIntroBannerDismissed(c *fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     introBannerCookie,
		Value:    "1",
		MaxAge:   365 * 24 * 60 * 60,
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
	})
}
