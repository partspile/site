package handlers

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/user"
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

// getThreshold gets the threshold parameter with default value
func getThreshold(ctx *fiber.Ctx) float64 {
	return ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
}

// getLocation gets the timezone location from context
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

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}
