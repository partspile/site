package handlers

import (
	"fmt"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
)

func HandleNewAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	makes := vehicle.GetMakes()
	return render(c, ui.NewAdPage(currentUser, c.Path(), makes))
}

func HandleNewAdSubmission(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	// Validate make selection first
	if c.FormValue("make") == "" {
		return render(c, ui.ValidationError("Please select a make first"))
	}

	form, err := c.MultipartForm()
	if err != nil {
		return render(c, ui.ValidationError(err.Error()))
	}

	// Validate required selections
	if len(form.Value["years"]) == 0 {
		return render(c, ui.ValidationError("Please select at least one year"))
	}

	if len(form.Value["models"]) == 0 {
		return render(c, ui.ValidationError("Please select at least one model"))
	}

	if len(form.Value["engines"]) == 0 {
		return render(c, ui.ValidationError("Please select at least one engine size"))
	}

	price := 0.0
	fmt.Sscanf(c.FormValue("price"), "%f", &price)

	make := c.FormValue("make")
	years := form.Value["years"]
	models := form.Value["models"]
	engines := form.Value["engines"]
	description := c.FormValue("description")

	newAd := ad.Ad{
		ID:          ad.GetNextAdID(),
		Make:        make,
		Years:       years,
		Models:      models,
		Engines:     engines,
		Description: description,
		Price:       price,
		UserID:      currentUser.ID,
	}

	ad.AddAd(newAd)

	return render(c, ui.SuccessMessage("Ad created successfully", "/"))
}

func HandleViewAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	// Get ad from either active or dead tables
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.ErrNotFound
	}

	currentUser, _ := c.Locals("user").(*user.User)

	return render(c, ui.ViewAdPage(currentUser, c.Path(), adObj))
}

func HandleEditAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	ad, ok := ad.GetAd(adID)
	if !ok || ad.ID == 0 {
		return fiber.ErrNotFound
	}

	if ad.UserID != currentUser.ID {
		return fiber.ErrForbidden
	}

	// Prepare make options
	makes := vehicle.GetMakes()
	// Prepare year checkboxes
	years := vehicle.GetYears(ad.Make)
	// Prepare model checkboxes
	modelAvailability := vehicle.GetModelsWithAvailability(ad.Make, ad.Years)
	// Prepare engine checkboxes
	engineAvailability := vehicle.GetEnginesWithAvailability(ad.Make, ad.Years, ad.Models)

	return render(c, ui.EditAdPage(currentUser, c.Path(), ad, makes, years, modelAvailability, engineAvailability))
}

func HandleUpdateAdSubmission(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	existingAd, ok := ad.GetAd(adID)
	if !ok || existingAd.ID == 0 {
		return fiber.ErrNotFound
	}
	if existingAd.UserID != currentUser.ID {
		return fiber.ErrForbidden
	}

	form, err := c.MultipartForm()
	if err != nil {
		return render(c, ui.ValidationError(err.Error()))
	}

	if len(form.Value["years"]) == 0 || len(form.Value["models"]) == 0 || len(form.Value["engines"]) == 0 {
		return render(c, ui.ValidationError("Please make sure you have selected a year, model, and engine"))
	}

	price := 0.0
	fmt.Sscanf(c.FormValue("price"), "%f", &price)

	updatedAd := ad.Ad{
		ID:          adID,
		Make:        c.FormValue("make"),
		Years:       form.Value["years"],
		Models:      form.Value["models"],
		Engines:     form.Value["engines"],
		Description: c.FormValue("description"),
		Price:       price,
		UserID:      currentUser.ID,
	}

	ad.UpdateAd(updatedAd)

	return render(c, ui.SuccessMessage("Ad updated successfully", fmt.Sprintf("/ad/%d", adID)))
}

func HandleDeleteAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)

	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}

	existingAd, ok := ad.GetAd(adID)
	if !ok {
		return fiber.ErrNotFound
	}
	if existingAd.UserID != currentUser.ID {
		return fiber.ErrForbidden
	}

	ad.DeleteAd(adID)

	return render(c, ui.SuccessMessage("Ad deleted successfully", "/"))
}
