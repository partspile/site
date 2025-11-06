package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/parts-pile/site/config"
)

// ConfigureHelmet returns a configured Helmet middleware with custom CSP,
// DENY for X-Frame-Options, and strict-origin-when-cross-origin for
// Referrer-Policy.
func ConfigureHelmet() fiber.Handler {
	// Custom Content-Security-Policy
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

	cfg := helmet.Config{
		ContentSecurityPolicy:     csp,
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		ContentTypeNosniff:        "nosniff",
		PermissionPolicy:          "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
	}

	// Set HSTS only if using HTTPS
	if config.CookieSecure {
		cfg.HSTSMaxAge = 31536000
		cfg.HSTSExcludeSubdomains = false // includeSubDomains
	}

	return helmet.New(cfg)
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
	// Configure CSRF cookie with secure settings using new API
	CookieName:     "_csrf",
	CookieHTTPOnly: true,                // Prevent XSS attacks from reading token
	CookieSecure:   config.CookieSecure, // HTTPS only when enabled
	CookieSameSite: "Strict",            // Prevent cross-site cookie sending
	CookiePath:     "/",
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
