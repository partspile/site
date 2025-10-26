package cookie

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/ui"
)

func GetView(c *fiber.Ctx) int {
	cookieValue := c.Cookies("view")

	view, err := strconv.Atoi(cookieValue)
	if err != nil {
		return ui.ViewList
	}

	return view
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

func GetAdCategory(c *fiber.Ctx) int {
	cookieValue := c.Cookies("ad_category")

	categoryID, err := strconv.Atoi(cookieValue)
	if err != nil {
		return ad.AdCategoryCarPart
	}

	// Check if the category ID exists in our cached map
	if _, exists := ad.AdCategoryNames[categoryID]; exists {
		return categoryID
	}

	return ad.AdCategoryCarPart
}

func SetAdCategory(c *fiber.Ctx, adCategory int) {
	c.Cookie(&fiber.Cookie{
		Name:     "ad_category",
		Value:    strconv.Itoa(adCategory),
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Strict",
	})
}

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
	c.ClearCookie("auth_token")
}

func GetJWT(c *fiber.Ctx) string {
	return c.Cookies("auth_token")
}
