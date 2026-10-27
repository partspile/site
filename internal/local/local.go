package local

import (
	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/logger"
)

func GetUserID(c *fiber.Ctx) int {
	userID, _ := c.Locals("user-id").(int)
	return userID
}

// IsLoggedIn reports whether userID identifies an authenticated user.
func IsLoggedIn(userID int) bool {
	return userID != 0
}

func SetUserID(c *fiber.Ctx, userID int) {
	c.Locals("user-id", userID)
}

func GetUserName(c *fiber.Ctx) string {
	userName, _ := c.Locals("user-name").(string)
	return userName
}

func SetUserName(c *fiber.Ctx, userName string) {
	c.Locals("user-name", userName)
}

func GetUserIsAdmin(c *fiber.Ctx) bool {
	isAdmin, _ := c.Locals("user-is-admin").(bool)
	return isAdmin
}

func SetUserIsAdmin(c *fiber.Ctx, isAdmin bool) {
	c.Locals("user-is-admin", isAdmin)
}

// GetCSRFToken extracts the CSRF token from context (set by CSRF middleware)
// The middleware should set this for all requests, but if it's empty,
// the middleware may not be running or token generation failed
func GetCSRFToken(c *fiber.Ctx) string {
	token, _ := c.Locals("csrf-token").(string)
	if token == "" {
		logger.Warn("CSRF token is empty in context, middleware may not be running",
			"component", "csrf", "path", c.Path(), "method", c.Method())
	}
	return token
}
