package handlers

import (
	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
)

// render sets the content type to HTML and renders the component.
func render(c *fiber.Ctx, component g.Node) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Response().BodyWriter())
}

// getQueryParam gets a parameter from either query string or form data
func getQueryParam(ctx *fiber.Ctx, key string) string {
	// Try query parameter first (for GET requests)
	if value := ctx.Query(key); value != "" {
		return value
	}
	// Fall back to form value (for POST requests)
	return ctx.FormValue(key)
}

func getView(c *fiber.Ctx) string {
	return c.Query("view", "list")
}
