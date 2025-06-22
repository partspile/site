package handlers

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
)

func HandleMakes(c *fiber.Ctx) error {
	makes := vehicle.GetMakes()
	return c.JSON(makes)
}

func HandleYears(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	years := vehicle.GetYears(makeName)
	return render(c, ui.YearsFormGroup(years))
}

func HandleModels(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one year is required")
	}

	modelAvailability := vehicle.GetModelsWithAvailability(makeName, years)
	return render(c, ui.ModelsFormGroup(modelAvailability))
}

func HandleEngines(c *fiber.Ctx) error {
	makeName := c.Query("make")
	if makeName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Make is required")
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one year is required")
	}

	models := q["models"]
	if len(models) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "At least one model is required")
	}

	engineAvailability := vehicle.GetEnginesWithAvailability(makeName, years, models)
	return render(c, ui.EnginesFormGroup(engineAvailability))
}
