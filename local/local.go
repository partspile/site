package local

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func GetUserID(c *fiber.Ctx) int {
	userID, _ := c.Locals("userID").(int)
	return userID
}

func SetUserID(c *fiber.Ctx, userID int) {
	c.Locals("userID", userID)
}

func SetUserName(c *fiber.Ctx, userName string) {
	c.Locals("userName", userName)
}

func GetUserName(c *fiber.Ctx) string {
	userName, _ := c.Locals("userName").(string)
	return userName
}

// GetCSRFToken extracts the CSRF token from context (set by CSRF middleware)
// The middleware should set this for all requests, but if it's empty,
// the middleware may not be running or token generation failed
func GetCSRFToken(c *fiber.Ctx) string {
	token, _ := c.Locals("csrf_token").(string)
	if token == "" {
		log.Printf("[CSRF] WARNING: CSRF token is empty in context for path %s, method %s. Middleware may not be running.", c.Path(), c.Method())
	}
	return token
}
