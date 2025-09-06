package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

func getLastView(c *fiber.Ctx) string {
	return c.Cookies("last_view", "list") // default to list
}

func saveLastView(c *fiber.Ctx, viewType string) {
	c.Cookie(&fiber.Cookie{
		Name:     "last_view",
		Value:    viewType,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		HTTPOnly: false,
		Path:     "/",
		SameSite: "Strict",
	})
}
