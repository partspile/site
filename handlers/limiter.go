package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/parts-pile/site/config"
)

// GlobalRateLimiter is the global rate limiter middleware
var GlobalRateLimiter = limiter.New(limiter.Config{
	Max:        config.ServerRateLimitMax,
	Expiration: config.ServerRateLimitExp,
})

// RegistrationRateLimiter is a strict rate limiter for registration (per IP)
var RegistrationRateLimiter = limiter.New(limiter.Config{
	Max:        config.RegistrationRateLimitMax,
	Expiration: config.RegistrationRateLimitExp,
	KeyGenerator: func(c *fiber.Ctx) string {
		// Rate limit per IP address
		return c.IP()
	},
	LimitReached: func(c *fiber.Ctx) error {
		return c.Status(429).
			SendString("Too many registration attempts. " +
				"Please try again later.")
	},
})
