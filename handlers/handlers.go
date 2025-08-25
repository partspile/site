package handlers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/user"
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

// Get location from context
func getLocation(c *fiber.Ctx) *time.Location {
	loc, _ := time.LoadLocation(c.Get("X-Timezone"))
	return loc
}

// getUser gets the current user and their ID
func getUser(c *fiber.Ctx) (*user.User, int) {
	currentUser, _ := CurrentUser(c)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	return currentUser, userID
}
