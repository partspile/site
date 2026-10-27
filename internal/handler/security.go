package handler

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/partspile/site/internal/config"
)

func minioPublicOrigin() string {
	raw := config.MinIOPublicURL
	if raw == "" {
		raw = config.MinIOAPIURL
	}
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme == "http" {
		return "http://" + u.Host
	}
	return "https://" + u.Host
}

// ConfigureHelmet returns a configured Helmet middleware with custom CSP,
// DENY for X-Frame-Options, and strict-origin-when-cross-origin for
// Referrer-Policy.
func ConfigureHelmet() fiber.Handler {
	imgSrc := "'self' data:"
	connectSrc := "'self'"
	if origin := minioPublicOrigin(); origin != "" {
		imgSrc += " " + origin
		connectSrc += " " + origin
	}

	csp := "default-src 'self'; " +
		"script-src 'self' 'unsafe-inline' https://challenges.cloudflare.com; " +
		"style-src 'self' 'unsafe-inline'; " +
		"img-src " + imgSrc + "; " +
		"font-src 'self' data:; " +
		"connect-src " + connectSrc + " https://challenges.cloudflare.com; " +
		"frame-src https://challenges.cloudflare.com; " +
		"frame-ancestors 'none';" +
		"base-uri 'self';" +
		"form-action 'self';"

	cfg := helmet.Config{
		ContentSecurityPolicy:     csp,
		XFrameOptions:             "DENY",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		ContentTypeNosniff:        "nosniff",
		PermissionPolicy:          "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
		CrossOriginEmbedderPolicy: "unsafe-none",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "cross-origin",
		OriginAgentCluster:        "?1",
		XDNSPrefetchControl:       "off",
		XDownloadOptions:          "noopen",
		XPermittedCrossDomain:     "none",
	}

	if config.CookieSecure {
		cfg.HSTSMaxAge = 31536000
		cfg.HSTSExcludeSubdomains = false
	}

	return helmet.New(cfg)
}

// CSRFMiddleware is the CSRF protection middleware configured for
// Double-Submit Cookie Pattern with cookie-based storage.
var CSRFMiddleware = csrf.New(csrf.Config{
	KeyLookup:      "header:X-Csrf-Token",
	ContextKey:     "csrf-token",
	CookieName:     "_csrf",
	CookieHTTPOnly: true,
	CookieSecure:   config.CookieSecure,
	CookieSameSite: "Strict",
	CookiePath:     "/",
	ErrorHandler: func(c *fiber.Ctx, err error) error {
		if c.Get("HX-Request") != "" {
			return ErrorHandler(c, fiber.NewError(fiber.StatusForbidden,
				"CSRF token missing or invalid. Please refresh the page and try again."))
		}
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "CSRF token missing or invalid",
		})
	},
	Next: func(c *fiber.Ctx) bool {
		path := c.Path()
		return path == "/api/sms/webhook"
	},
})
