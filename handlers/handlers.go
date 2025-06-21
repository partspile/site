package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	g "maragu.dev/gomponents"
)

// render sets the content type to HTML and renders the component.
func render(c *fiber.Ctx, component g.Node) error {
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return component.Render(c.Response().BodyWriter())
}

func anyStringInSlice(a, b []string) bool {
	for _, aVal := range a {
		for _, bVal := range b {
			if aVal == bVal {
				return true
			}
		}
	}
	return false
}

func htmlEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
