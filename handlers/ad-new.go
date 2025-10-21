package handlers

import (
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
)

// HandleNewAd shows the new ad form
func HandleNewAd(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	adCat := AdCategory(c)
	makes := vehicle.GetMakes(adCat)
	partCategories := part.GetCategories(adCat)
	return render(c, ui.NewAdPage(currentUser, c.Path(), makes, partCategories))
}

// HandleDuplicateAd shows the duplicate ad form with pre-filled data
func HandleDuplicateAd(c *fiber.Ctx) error {
	currentUser, _ := CurrentUser(c)
	adCat := AdCategory(c)

	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Fetch the original ad
	adDetail, err := ad.GetAdDetailByID(adID, currentUser)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Pre-fetch all the dropdown/checkbox data needed for the form
	makes := vehicle.GetMakes(adCat)
	categoryNames := part.GetCategories(adCat)
	years := vehicle.GetYears(adCat, adDetail.Make)
	models := vehicle.GetModels(adCat, adDetail.Make, adDetail.Years)
	engines := vehicle.GetEngines(adCat, adDetail.Make, adDetail.Years, adDetail.Models)
	var subcategoryNames []string
	if adDetail.PartCategory.Valid {
		subcategoryNames = part.GetSubCategories(adCat, adDetail.PartCategory.String)
	}

	return render(c, ui.DuplicateAdPage(
		currentUser, c.Path(), makes, categoryNames,
		*adDetail, years, models, engines, subcategoryNames, adDetail.PartSubcategory.String))
}

// HandleNewAdSubmission processes the new ad form submission
func HandleNewAdSubmission(c *fiber.Ctx) error {
	_, userID := CurrentUser(c)

	// Resolve and store location first
	locationRaw := c.FormValue("location")
	locID, err := resolveAndStoreLocation(locationRaw)
	if err != nil {
		return ValidationErrorResponse(c, "Could not resolve location.")
	}

	newAd, imageFiles, _, err := BuildAdFromForm(c, userID, locID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}
	adID, err := ad.AddAd(newAd)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create ad")
	}
	fmt.Printf("[DEBUG] Created ad ID=%d with ImageCount=%d\n", adID, newAd.ImageCount)
	fmt.Printf("[DEBUG] Image files count: %d\n", len(imageFiles))
	if len(imageFiles) > 0 {
		for i, file := range imageFiles {
			fmt.Printf("[DEBUG] Image file %d: %s (size: %d bytes)\n", i+1, file.Filename, file.Size)
		}
	}
	uploadAdImagesToB2(adID, imageFiles)

	// Attempt inline vector processing, fallback to queue if it fails
	log.Printf("[embedding] Attempting inline vector processing for ad %d", adID)
	err = vector.BuildAdEmbedding(newAd)
	if err != nil {
		log.Printf("[embedding] Inline processing failed for ad %d: %v, queuing for background processing", adID, err)
		vector.QueueAd(newAd.Ad)
	} else {
		log.Printf("[embedding] Successfully processed ad %d inline", adID)
	}

	return render(c, ui.SuccessMessage("Ad created successfully", "/"))
}
