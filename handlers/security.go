package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/config"
)

// SecurityHeaders sets HTTP security headers on all responses
func SecurityHeaders(c *fiber.Ctx) error {

	// Content-Security-Policy
	// This policy restricts which resources can be loaded
	csp := "default-src 'self'; " +
		"script-src 'self' https://unpkg.com 'unsafe-eval'; " + // 'unsafe-eval' required for HTMX dynamic JS evaluation
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
