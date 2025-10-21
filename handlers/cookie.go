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

func getCookieAdCategory(c *fiber.Ctx) string {
	categoryStr := c.Cookies("ad_category", "CarPart")
	return ad.ParseCategoryFromQuery(categoryStr)
}

func saveCookieAdCategory(c *fiber.Ctx, category string) {
	c.Cookie(&fiber.Cookie{
		Name:     "ad_category",
		Value:    category,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
}
