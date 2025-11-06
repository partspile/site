package handlers

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/cookie"
	"github.com/parts-pile/site/local"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/spec"
)

// HandleNewAd shows the new ad form
func HandleNewAd(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	adCat := cookie.GetAdCategory(c)
	makes := spec.GetMakes(adCat)
	partCategories := part.GetCategories(adCat)
	maxImagesPerAd := config.MaxImagesPerAd
	csrfToken := local.GetCSRFToken(c)
	return render(c, ui.NewAdPage(adCat, userID, userName, csrfToken, c.Path(), makes, partCategories, maxImagesPerAd))
}

// HandleDuplicateAd shows the duplicate ad form with pre-filled data
func HandleDuplicateAd(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	adCat := cookie.GetAdCategory(c)

	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Fetch the original ad
	adDetail, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Pre-fetch all the dropdown/checkbox data needed for the form
	makes := spec.GetMakes(adCat)
	categoryNames := part.GetCategories(adCat)
	years := spec.GetYears(adCat, adDetail.Make)
	models := spec.GetModels(adCat, adDetail.Make, adDetail.Years)
	engines := spec.GetEngines(adCat, adDetail.Make, adDetail.Years, adDetail.Models)
	subcategoryNames := part.GetSubCategories(adCat, adDetail.PartCategory)
	csrfToken := local.GetCSRFToken(c)

	return render(c, ui.DuplicateAdPage(
		userID, userName, csrfToken, c.Path(), makes, categoryNames,
		*adDetail, years, models, engines, subcategoryNames, adDetail.PartSubcategory, config.MaxImagesPerAd))
}

// HandleNewAdSubmission processes the new ad form submission
func HandleNewAdSubmission(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

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
	logger.Debug("Created ad",
		"component", "ad-new", "adID", adID, "imageCount", newAd.ImageCount)
	logger.Debug("Image files count",
		"component", "ad-new", "count", len(imageFiles))
	if len(imageFiles) > 0 {
		for i, file := range imageFiles {
			logger.Debug("Image file details",
				"component", "ad-new", "index", i+1, "filename", file.Filename, "size", file.Size)
		}
	}
	uploadAdImagesToB2(adID, imageFiles)

	// Attempt inline vector processing, fallback to queue if it fails
	logger.Debug("Attempting inline vector processing",
		"component", "embedding", "adID", adID)
	err = vector.BuildAdEmbedding(newAd)
	if err != nil {
		logger.Debug("Inline processing failed, queuing for background processing",
			"error", err, "component", "embedding", "adID", adID)
		vector.QueueAd(newAd.Ad)
	} else {
		logger.Debug("Successfully processed ad inline",
			"component", "embedding", "adID", adID)
	}

	redirectDelayMs := config.ServerRedirectDelay.Milliseconds()
	return render(c, ui.SuccessMessage("Ad created successfully", "/", redirectDelayMs))
}

// HandleYears handles the years dropdown for new ad form
func HandleYears(c *fiber.Ctx) error {
	makeName := c.Query("make")
	adCat := cookie.GetAdCategory(c)
	if makeName == "" {
		// Return empty div when make is not selected
		return render(c, ui.YearsSelector([]string{}))
	}

	years := spec.GetYears(adCat, makeName)
	return render(c, ui.YearsSelector(years))
}

// HandleModels handles the models dropdown for new ad form
func HandleModels(c *fiber.Ctx) error {
	makeName := c.Query("make")
	adCat := cookie.GetAdCategory(c)
	if makeName == "" {
		// Return empty div when make is not selected
		return render(c, ui.ModelsSelector([]string{}))
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		// Return empty div instead of error when no years are selected
		return render(c, ui.ModelsSelector([]string{}))
	}

	models := spec.GetModels(adCat, makeName, years)
	if len(models) == 0 {
		// Return empty message when no models are available for all selected years
		return render(c, ui.ModelsDivEmpty())
	}
	return render(c, ui.ModelsSelector(models))
}

// HandleEngines handles the engines dropdown for new ad form
func HandleEngines(c *fiber.Ctx) error {
	makeName := c.Query("make")
	adCat := cookie.GetAdCategory(c)
	if makeName == "" {
		// Return empty div when make is not selected
		return render(c, ui.EnginesSelector([]string{}))
	}

	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}
	years := q["years"]
	if len(years) == 0 {
		// Return empty div instead of error when no years are selected
		return render(c, ui.EnginesSelector([]string{}))
	}

	models := q["models"]
	if len(models) == 0 {
		// Return empty div instead of error when no models are selected
		return render(c, ui.EnginesSelector([]string{}))
	}

	engines := spec.GetEngines(adCat, makeName, years, models)
	if len(engines) == 0 {
		// Return empty message when no engines are available for all selected year-model combinations
		return render(c, ui.EnginesDivEmpty())
	}
	return render(c, ui.EnginesSelector(engines))
}

// HandleSubCategories handles the subcategories dropdown for new ad form
func HandleSubCategories(c *fiber.Ctx) error {
	categoryName := c.Query("category")
	adCat := cookie.GetAdCategory(c)
	if categoryName == "" {
		// Return empty div when category is not selected
		return render(c, ui.SubCategoriesSelector([]string{}, ""))
	}

	subCategoryNames := part.GetSubCategories(adCat, categoryName)
	return render(c, ui.SubCategoriesSelector(subCategoryNames, ""))
}
