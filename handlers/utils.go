package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	g "maragu.dev/gomponents"
)

// render sets the content type to HTML and renders the component.
func render(c *fiber.Ctx, component g.Node) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Response().BodyWriter())
}

func getView(c *fiber.Ctx) string {
	return c.Query("view", "list")
}

// AdCategory returns the ad category ID from the ad_category cookie
func AdCategory(c *fiber.Ctx) int {
	return cookie.GetAdCategory(c)
}
