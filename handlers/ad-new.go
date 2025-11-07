package handlers

import (
	"net/url"
	"strings"

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
	return HandleFieldValues(c, "year")
}

// HandleModels handles the models dropdown for new ad form
func HandleModels(c *fiber.Ctx) error {
	return HandleFieldValues(c, "model")
}

// HandleEngines handles the engines dropdown for new ad form
func HandleEngines(c *fiber.Ctx) error {
	return HandleFieldValues(c, "engine")
}

// HandleFieldValues handles field value requests generically for any field
func HandleFieldValues(c *fiber.Ctx, fieldName string) error {
	adCat := cookie.GetAdCategory(c)

	// Get parent field values from query parameters
	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}

	// Build parent values map
	parentValues := make(map[string]string)
	fieldConfigs, err := spec.GetCategoryFields(adCat)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get category fields")
	}

	// Find current field index
	currentFieldIndex := -1
	for i, f := range fieldConfigs {
		if f.Name == fieldName {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid field name")
	}

	// Get parent field values (fields that come before current field)
	for i := 0; i < currentFieldIndex; i++ {
		parentField := fieldConfigs[i]
		// Check for single value or multiple values
		if values, exists := q[parentField.Name]; exists && len(values) > 0 {
			// For multi-value fields (years, models, engines), join with comma
			// For single-value fields (make), use first value
			if parentField.QdrantType == "keyword_array" {
				parentValues[parentField.Name] = strings.Join(values, ",")
			} else {
				parentValues[parentField.Name] = values[0]
			}
		}
	}

	// Check if required parent fields are present
	for i := 0; i < currentFieldIndex; i++ {
		parentField := fieldConfigs[i]
		if parentField.IsRequired {
			if _, exists := parentValues[parentField.Name]; !exists {
				// Return empty selector
				return renderEmptyFieldSelector(c, fieldName)
			}
		}
	}

	// Get field values
	values, err := spec.GetFieldValues(adCat, fieldName, parentValues)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to get field values")
	}

	// Render appropriate selector based on field type
	fieldDef, exists := spec.GetFieldDefinition(fieldName)
	if !exists {
		return fiber.NewError(fiber.StatusBadRequest, "Field definition not found")
	}

	if fieldDef.QdrantType == "keyword_array" {
		// Multi-select (checkboxes) - years, models, engines
		return renderMultiSelectField(c, fieldName, values)
	} else {
		// Single select (dropdown) - make, breed
		return renderSingleSelectField(c, fieldName, values)
	}
}

// renderEmptyFieldSelector returns empty selector for a field
func renderEmptyFieldSelector(c *fiber.Ctx, fieldName string) error {
	fieldDef, exists := spec.GetFieldDefinition(fieldName)
	if !exists {
		return fiber.NewError(fiber.StatusBadRequest, "Field definition not found")
	}

	if fieldDef.QdrantType == "keyword_array" {
		return renderMultiSelectField(c, fieldName, []string{})
	} else {
		return renderSingleSelectField(c, fieldName, []string{})
	}
}

// renderMultiSelectField renders a multi-select field (checkboxes)
func renderMultiSelectField(c *fiber.Ctx, fieldName string, values []string) error {
	// For now, use existing UI functions for backward compatibility
	switch fieldName {
	case "year":
		return render(c, ui.YearsSelector(values))
	case "model":
		if len(values) == 0 {
			return render(c, ui.ModelsDivEmpty())
		}
		return render(c, ui.ModelsSelector(values))
	case "engine":
		if len(values) == 0 {
			return render(c, ui.EnginesDivEmpty())
		}
		return render(c, ui.EnginesSelector(values))
	default:
		// Generic multi-select - TODO: create generic UI component
		return render(c, ui.YearsSelector(values))
	}
}

// renderSingleSelectField renders a single-select field (dropdown)
func renderSingleSelectField(c *fiber.Ctx, fieldName string, values []string) error {
	// For now, return empty - single select fields are rendered statically
	// TODO: create generic single-select component
	return c.SendString("")
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
