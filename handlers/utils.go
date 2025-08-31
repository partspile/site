package handlers

import "github.com/gofiber/fiber/v2"

// getQueryParam gets a parameter from either query string or form data
func getQueryParam(ctx *fiber.Ctx, key string) string {
	// Try query parameter first (for GET requests)
	if value := ctx.Query(key); value != "" {
		return value
	}
	// Fall back to form value (for POST requests)
	return ctx.FormValue(key)
}
