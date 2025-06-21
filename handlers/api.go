package handlers

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vehicle"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
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
	checkboxes := []g.Node{}

	for _, year := range years {
		checkboxes = append(checkboxes,
			ui.Checkbox("years", year, year, false, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
				hx.Swap("innerHTML"),
				g.Attr("onclick", "document.getElementById('enginesDiv').innerHTML = ''"),
			),
		)
	}

	return render(c, ui.FormGroup("Years", "years", ui.GridContainer(5, checkboxes...)))
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
	checkboxes := []g.Node{}
	for model, isAvailable := range modelAvailability {
		checkboxes = append(checkboxes,
			ui.Checkbox("models", model, model, false, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}

	return render(c, ui.FormGroup("Models", "models", ui.GridContainer(5, checkboxes...)))
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
	checkboxes := []g.Node{}
	for engine, isAvailable := range engineAvailability {
		checkboxes = append(checkboxes,
			ui.Checkbox("engines", engine, engine, false, !isAvailable),
		)
	}
	return render(c, ui.FormGroup("Engines", "engines", ui.GridContainer(5, checkboxes...)))
}
