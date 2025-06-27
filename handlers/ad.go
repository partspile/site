package handlers

import (
	"fmt"

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

	newAd, err := BuildAdFromForm(c, currentUser.ID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}
	ad.AddAd(newAd)
	return render(c, ui.SuccessMessage("Ad created successfully", "/"))
}

func HandleViewAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}

	// Get ad from either active or archived tables
	adObj, _, ok := ad.GetAdByID(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
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

	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}

	ad, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if err := ValidateOwnership(ad.UserID, currentUser.ID); err != nil {
		return err
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
	println("HandleUpdateAdSubmission")
	currentUser := c.Locals("user").(*user.User)

	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	existingAd, ok := ad.GetAd(adID)
	if !ok {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	if err := ValidateOwnership(existingAd.UserID, currentUser.ID); err != nil {
		return err
	}

	updatedAd, err := BuildAdFromForm(c, currentUser.ID, adID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	ad.UpdateAd(updatedAd)
	return render(c, ui.SuccessMessage("Ad updated successfully", fmt.Sprintf("/ad/%d", adID)))
}

// Handler to flag an ad
func HandleFlagAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.FlagAd(currentUser.ID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to flag ad")
	}
	// Return the flagged button HTML for HTMX swap
	return render(c, ui.FlagButton(true, adID))
}

// Handler to unflag an ad
func HandleUnflagAd(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.UnflagAd(currentUser.ID, adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to unflag ad")
	}
	// Return the unflagged button HTML for HTMX swap
	return render(c, ui.FlagButton(false, adID))
}

// Handler to get flagged ads for the current user (for settings page)
func HandleFlaggedAds(c *fiber.Ctx) error {
	currentUser := c.Locals("user").(*user.User)
	adIDs, err := ad.GetFlaggedAdIDsByUser(currentUser.ID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get flagged ad IDs")
	}
	ads, err := ad.GetAdsByIDs(adIDs)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get flagged ads")
	}
	return render(c, ui.FlaggedAdsSection(currentUser, ads))
}

func HandleArchiveAd(c *fiber.Ctx) error {
	adID, err := ParseIntParam(c, "id")
	if err != nil {
		return err
	}
	if err := ad.ArchiveAd(adID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to archive ad")
	}
	return render(c, ui.SuccessMessage("Ad archived successfully", "/"))
}
