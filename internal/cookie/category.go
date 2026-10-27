package cookie

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
)

func GetCategoryID(c *fiber.Ctx) int {
	// Prefer a category set earlier in this request (e.g. /c/:id) so a
	// later GetCategoryID does not overwrite it with the default.
	if v, ok := c.Locals("categoryID").(int); ok && v > 0 {
		return v
	}
	category := c.Cookies("category")
	categoryID := ad.ParseCategory(category)
	if category == "" {
		SetCategoryID(c, categoryID)
	}
	return categoryID
}

func SetCategoryID(c *fiber.Ctx, category int) {
	c.Locals("categoryID", category)
	c.Cookie(&fiber.Cookie{
		Name:     "category",
		Value:    strconv.Itoa(category),
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HTTPOnly: true,
		Secure:   config.CookieSecure,
		Path:     "/",
		SameSite: "Lax",
	})
}
