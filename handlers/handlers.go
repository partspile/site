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

// getUserID extracts the user ID from the current user context
// Returns 0 if no user is logged in, otherwise returns the user's ID
func getUserID(c *fiber.Ctx) int {
	currentUser, _ := CurrentUser(c)
	if currentUser != nil {
		return currentUser.ID
	}
	return 0
}

// getUserIDFromUser extracts the user ID from a user object
// Returns 0 if the user is nil, otherwise returns the user's ID
func getUserIDFromUser(currentUser *user.User) int {
	if currentUser != nil {
		return currentUser.ID
	}
	return 0
}
