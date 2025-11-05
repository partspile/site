package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/parts-pile/site/config"
)

// SecurityHeaders sets HTTP security headers on all responses
func SecurityHeaders(c *fiber.Ctx) error {

	// Content-Security-Policy
	// This policy restricts which resources can be loaded
	csp := "default-src 'self'; " +
		"script-src 'self' https://unpkg.com 'unsafe-inline'; " + // unpkg.com for HTMX library, 'unsafe-inline' for inline scripts (redirect scripts, category modal removal, MAX_IMAGES_PER_AD constant)
		"style-src 'self' 'unsafe-inline'; " + // 'unsafe-inline' needed for Tailwind CSS
		"img-src 'self' https://*.backblazeb2.com data:; " +
		"font-src 'self' data:; " +
		"connect-src 'self'; " +
		"frame-ancestors 'none';" +
		"base-uri 'self';" +
		"form-action 'self';"

	c.Set("Content-Security-Policy", csp)

	// Prevent clickjacking attacks
	c.Set("X-Frame-Options", "DENY")

	// Prevent MIME type sniffing
	c.Set("X-Content-Type-Options", "nosniff")

	// HTTP Strict Transport Security (only if using HTTPS)
	if config.CookieSecure {
		c.Set("Strict-Transport-Security",
			"max-age=31536000; includeSubDomains")
	}

	// Control referrer information leakage
	c.Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Disable browser features not needed by the application
	c.Set("Permissions-Policy",
		"geolocation=(), microphone=(), camera=(), payment=(), usb=(), "+
			"magnetometer=(), gyroscope=(), accelerometer=()")

	return c.Next()
}

// CSRFMiddleware is the CSRF protection middleware configured for
// Double-Submit Cookie Pattern with cookie-based storage.
// The token is stored in a secure cookie and validated against the header value.
// This pattern works across multiple instances without requiring shared session storage.
var CSRFMiddleware = csrf.New(csrf.Config{
	// Use cookie-based storage (double-submit cookie pattern)
	// No Session parameter means cookie-based storage is used
	// Check for token in header (used by HTMX requests via hx.Headers)
	KeyLookup: "header:X-Csrf-Token",
	// Store token in context so handlers can access it
	ContextKey: "csrf_token",
	// Configure CSRF cookie with secure settings
	Cookie: &fiber.Cookie{
		Name:     "_csrf",
		HTTPOnly: true,                // Prevent XSS attacks from reading token
		Secure:   config.CookieSecure, // HTTPS only when enabled
		SameSite: "Strict",            // Prevent cross-site cookie sending
		Path:     "/",
	},
	ErrorHandler: func(c *fiber.Ctx, err error) error {
		// For HTMX requests, return HTML error page; for API requests, return JSON
		if c.Get("HX-Request") != "" {
			return CustomErrorHandler(c, fiber.NewError(fiber.StatusForbidden,
				"CSRF token missing or invalid. Please refresh the page and try again."))
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "CSRF token missing or invalid",
		})
	},
	Next: func(c *fiber.Ctx) bool {
		// Exclude specific POST endpoints that don't need CSRF protection
		path := c.Path()
		return path == "/api/sms/webhook"
	},
})
