package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/jwt"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/user"
)

// JWTMiddleware is a middleware that validates a JWT token and sets the user in the context.
func JWTMiddleware(c *fiber.Ctx) error {
	// Get JWT token from cookie
	tokenString := cookie.GetJWT(c)
	if tokenString == "" {
		local.SetUserID(c, 0)
		local.SetUserName(c, "")
		return c.Next()
	}

	// Validate JWT token
	claims, err := jwt.ValidateToken(tokenString)
	if err != nil {
		// Invalid token, clear cookie
		cookie.ClearJWT(c)
		local.SetUserID(c, 0)
		local.SetUserName(c, "")
		return c.Next()
	}

	// Set user ID and username in context
	local.SetUserID(c, jwt.GetUserID(claims))
	local.SetUserName(c, jwt.GetUserName(claims))
	return c.Next()
}

// AuthRequired is a middleware that requires a user to be logged in.
func AuthRequired(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if userID == 0 {
		return redirectToLogin(c)
	}

	// Verify that the user still exists and is not archived
	u, err := user.GetUser(userID)
	if err != nil || u.IsArchived() {
		// User no longer exists or is archived, clear cookie and redirect to login
		cookie.ClearJWT(c)
		local.SetUserID(c, 0)
		local.SetUserName(c, "")
		return redirectToLogin(c)
	}

	return c.Next()
}

// AdminRequired is a middleware that requires a user to be an admin.
func AdminRequired(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if userID == 0 {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}

	// Fetch current admin status from database
	u, err := user.GetUser(userID)
	if err != nil || u.IsArchived() {
		// User no longer exists or is archived, clear cookie
		cookie.ClearJWT(c)
		local.SetUserID(c, 0)
		local.SetUserName(c, "")
		return c.Status(fiber.StatusUnauthorized).SendString("User not found")
	}

	if !u.IsAdmin {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}

	return c.Next()
}
