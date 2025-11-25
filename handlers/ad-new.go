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
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	g "maragu.dev/gomponents"
)

// HandleNewAd shows the new ad form
func HandleNewAd(c *fiber.Ctx) error {
	adCat := cookie.GetAdCategory(c)
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	maxImagesPerAd := config.MaxImagesPerAd
	csrfToken := local.GetCSRFToken(c)
	fields := ad.GetCategoryFields(adCat)

	// Only render fields that are not in a chain or are the first field of a chain
	fieldNodes := []g.Node{}
	var inChain bool
	for _, field := range fields {
		if !inChain {
			fieldNodes = append(fieldNodes, field.RenderNewAdField(map[string]string{}))
		}
		inChain = field.NextField() != ""
	}

	return render(c, ui.NewAdPage(
		adCat,
		userID,
		userName,
		csrfToken,
		c.Path(),
		fieldNodes,
		maxImagesPerAd,
	))
}

// HandleNewAdSubmission processes the new ad form submission
func HandleNewAdSubmission(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	newAd, imageFiles, _, err := BuildAdFromForm(c, userID)
	if err != nil {
		return ValidationErrorResponse(c, err.Error())
	}

	err = ad.AddAd(&newAd)
	if err != nil {
		logger.Error("Failed to create ad",
			"component", "ad-new", "error", err, "userID", userID)
		return fiber.NewError(fiber.StatusInternalServerError, "Failed to create ad")
	}

	logger.Debug("Created ad",
		"component", "ad-new", "adID", newAd.ID, "imageCount", newAd.ImageCount)
	logger.Debug("Image files count",
		"component", "ad-new", "count", len(imageFiles))

	if len(imageFiles) > 0 {
		for i, file := range imageFiles {
			logger.Debug("Image file details",
				"component", "ad-new", "index", i+1, "filename", file.Filename, "size", file.Size)
		}
	}
	uploadAdImagesToB2(newAd.ID, imageFiles)

	// Attempt inline vector processing, fallback to queue if it fails
	logger.Debug("Attempting inline vector processing",
		"component", "embedding", "adID", newAd.ID)
	err = vector.BuildAdEmbedding(newAd)
	if err != nil {
		logger.Debug("Inline processing failed, queuing for background processing",
			"error", err, "component", "embedding", "adID", newAd.ID)
		vector.QueueAd(newAd.ID)
	} else {
		logger.Debug("Successfully processed ad inline",
			"component", "embedding", "adID", newAd.ID)
	}

	redirectDelayMs := config.ServerRedirectDelay.Milliseconds()
	return render(c, ui.SuccessMessage("Ad created successfully", "/", redirectDelayMs))
}

// HandleFieldValues handles field value requests generically for any field
// The field name is extracted from the route parameter
func HandleFieldValues(c *fiber.Ctx) error {
	fieldName := c.Params("fieldName")
	if fieldName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Field name is required")
	}
	adCat := cookie.GetAdCategory(c)

	// Get parent field values from query parameters
	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}

	// Get the current field
	currentField, err := ad.GetCategoryField(adCat, fieldName)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid field name")
	}

	// Build parentValues map from query parameters
	parentValues := buildParentValuesFromQuery(q, adCat, fieldName)

	// Render the field - GetValues will handle parent filtering internally
	fieldNode := currentField.RenderNewAdField(parentValues)

	return render(c, fieldNode)
}

// buildParentValuesFromQuery builds a parentValues map from query parameters
// It converts query parameter values to the format expected by Field.GetValues
func buildParentValuesFromQuery(q url.Values, adCat int, excludeFieldName string) map[string]string {
	parentValues := make(map[string]string)
	fields := ad.GetCategoryFields(adCat)

	for _, field := range fields {
		if field.Name() == excludeFieldName {
			// Skip the excluded field itself
			continue
		}
		if values, exists := q[field.Name()]; exists && len(values) > 0 {
			// For multi-value fields, join with comma; for single-value, use first value
			if field.IsMultiValue() {
				parentValues[field.Name()] = strings.Join(values, ",")
			} else {
				parentValues[field.Name()] = values[0]
			}
		}
	}

	return parentValues
}
