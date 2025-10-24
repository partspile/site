package cookie

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
)

func GetLastView(c *fiber.Ctx) string {
	return c.Cookies("last_view", "list") // default to list
}

func SetLastView(c *fiber.Ctx, view string) {
	c.Cookie(&fiber.Cookie{
		Name:     "last_view",
		Value:    view,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HTTPOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: "Strict",
	})
}

func GetAdCategory(c *fiber.Ctx) string {
	categoryStr := c.Cookies("ad_category", "CarPart")
	return ad.ParseCategoryFromQuery(categoryStr)
}

func SetAdCategory(c *fiber.Ctx, category string) {
	c.Cookie(&fiber.Cookie{
		Name:     "ad_category",
		Value:    category,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HTTPOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: "Strict",
	})
}

func SetJWT(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   24 * 60 * 60, // 24 hours
	})
}

func ClearJWT(c *fiber.Ctx) {
	c.ClearCookie("auth_token")
}

func GetJWT(c *fiber.Ctx) string {
	return c.Cookies("auth_token")
}
