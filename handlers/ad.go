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

	// Get ad from either active or archived tables
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.ErrNotFound
	}

	currentUser, _ := c.Locals("user").(*user.User)
	flagged := false
	if currentUser != nil {
		flagged, _ = ad.IsAdFlaggedByUser(currentUser.ID, adID)
	}

	return render(c, ui.ViewAdPage(currentUser, c.Path(), adObj, flagged))
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

// Handler to flag an ad
func HandleFlagAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := ad.FlagAd(currentUser.ID, adID); err != nil {
		return fiber.ErrInternalServerError
	}
	// Return the flagged button HTML for HTMX swap
	return render(c, ui.FlagButton(true, adID))
}

// Handler to unflag an ad
func HandleUnflagAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	adID, err := c.ParamsInt("id")
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := ad.UnflagAd(currentUser.ID, adID); err != nil {
		return fiber.ErrInternalServerError
	}
	// Return the unflagged button HTML for HTMX swap
	return render(c, ui.FlagButton(false, adID))
}

// Handler to get flagged ads for the current user (for settings page)
func HandleFlaggedAds(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	adIDs, err := ad.GetFlaggedAdIDsByUser(currentUser.ID)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	ads, err := ad.GetAdsByIDs(adIDs)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	return render(c, ui.FlaggedAdsSection(currentUser, ads))
}

func HandleArchiveAd(c *fiber.Ctx) error {
	adID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).SendString("Invalid ad ID")
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return c.Status(500).SendString("Failed to archive ad")
	}
	return c.SendStatus(204)
}
