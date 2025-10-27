package handlers

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/db"
)

// HandleHealth returns the health status of the application
func HandleHealth(c *fiber.Ctx) error {
	health := map[string]string{
		"status": "ok",
	}

	// Check database connectivity
	if err := db.Get().Ping(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = "down"
		c.Status(fiber.StatusServiceUnavailable)
	} else {
		health["database"] = "up"
	}

	// Return JSON response
	c.Set("Content-Type", "application/json")
	return json.NewEncoder(c).Encode(health)
}
