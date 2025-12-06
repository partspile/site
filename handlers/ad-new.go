package handlers

import (
	"fmt"

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

	uploadAdImagesToMinIO(newAd.ID, imageFiles)

	// Attempt inline vector processing, fallback to queue if it fails
	err = vector.BuildAdEmbedding(newAd)
	if err != nil {
		vector.QueueAd(newAd.ID)
	}

	redirectDelayMs := config.ServerRedirectDelay.Milliseconds()
	redirectURL := fmt.Sprintf("/ad/%d", newAd.ID)
	return render(c, ui.SuccessMessage("Ad created successfully", redirectURL, redirectDelayMs))
}

// HandleFieldValues handles field value requests generically for any field
// The field name is extracted from the route parameter
func HandleFieldValues(c *fiber.Ctx) error {
	fieldName := c.Params("fieldName")
	if fieldName == "" {
		return fiber.NewError(fiber.StatusBadRequest, "Field name is required")
	}
	adCat := cookie.GetAdCategory(c)

	// Debug: Log all query parameters
	allQueries := c.Queries()
	logger.Debug("HandleFieldValues request",
		"component", "ad-new",
		"fieldName", fieldName,
		"adCat", adCat,
		"allQueryParams", allQueries,
		"queryString", c.OriginalURL(),
	)

	// Get the current field
	currentField, err := ad.GetCategoryField(adCat, fieldName)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid field name")
	}

	// Build parentValues map from query parameters
	parentValues := ad.BuildParentValues(c, adCat, []string{fieldName})

	// Debug: Log parentValues
	logger.Debug("HandleFieldValues parentValues",
		"component", "ad-new",
		"fieldName", fieldName,
		"parentValues", parentValues,
	)

	// Render the field - GetValues will handle parent filtering internally
	fieldNode := currentField.RenderNewAdField(parentValues)

	return render(c, fieldNode)
}
