package handlers

import (
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
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HTTPOnly: true,
		Secure:   true,
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
		MaxAge:   30 * 24 * 60 * 60, // 30 days
		HTTPOnly: true,
		Secure:   true,
		Path:     "/",
		SameSite: "Strict",
	})
}

func setJWTCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   24 * 60 * 60, // 24 hours
	})
}

func clearJWTCookie(c *fiber.Ctx) {
	c.ClearCookie("auth_token")
}

func getJWTCookie(c *fiber.Ctx) string {
	return c.Cookies("auth_token")
}
