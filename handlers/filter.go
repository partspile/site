package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
)

// HandleFilterMakes returns makes that have existing ads for filter dropdowns
func HandleFilterMakes(c *fiber.Ctx) error {
	category := AdCategory(c)

	makes, err := vehicle.GetAdMakes(category)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to get makes"})
	}

	// Return HTML options for the select dropdown
	c.Set("Content-Type", "text/html")
	return render(c, ui.MakeFilterOptions(makes))
}
