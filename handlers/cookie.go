package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
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

func getCookieAdCategoryID(c *fiber.Ctx) ad.AdCategory {
	categoryStr := c.Cookies("ad_category", "CarParts") // default to CarParts
	return ad.ParseCategoryFromQuery(categoryStr)
}

func saveCookieAdCategoryID(c *fiber.Ctx, category ad.AdCategory) {
	c.Cookie(&fiber.Cookie{
		Name:     "ad_category",
		Value:    category.String(),
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
}
