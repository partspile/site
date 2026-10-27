package handler

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/local"
)

func loginURL(returnPath string) string {
	if p := safeReturnPath(returnPath); p != "" {
		return "/login?return=" + url.QueryEscape(p)
	}
	return "/login"
}

// redirectToLogin redirects to the login page
// For HTMX requests, uses HX-Redirect header to trigger a full page redirect
func redirectToLogin(c *fiber.Ctx) error {
	dest := "/login"
	if ref := c.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil {
			dest = loginURL(u.RequestURI())
		}
	}
	// Check if this is an HTMX request
	if c.Get("HX-Request") != "" {
		// Use HX-Redirect header for HTMX requests to trigger full page redirect
		c.Set("HX-Redirect", dest)
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	// Regular redirect for non-HTMX requests
	return c.Redirect(dest, fiber.StatusFound)
}

// AuthRequired is a middleware that requires a user to be logged in.
func AuthRequired(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if !local.IsLoggedIn(userID) {
		return redirectToLogin(c)
	}
	return c.Next()
}

// AdminRequired is a middleware that requires a user to be an admin.
func AdminRequired(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	if !local.IsLoggedIn(userID) {
		return c.Status(fiber.StatusUnauthorized).SendString("Unauthorized")
	}
	if !local.GetUserIsAdmin(c) {
		return c.Status(fiber.StatusForbidden).SendString("Forbidden")
	}
	return c.Next()
}
