package handlers

import (
	"fmt"
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
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// HandleNewAd shows the new ad form
func HandleNewAd(c *fiber.Ctx) error {
	adCat := cookie.GetAdCategory(c)
	return renderNewAdPage(c, adCat, nil)
}

// HandleDuplicateAd shows the duplicate ad form with pre-filled data
func HandleDuplicateAd(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Fetch the original ad
	adDetail, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Switch to the duplicated ad's category
	cookie.SetAdCategory(c, adDetail.AdCategoryID)

	return renderNewAdPage(c, adDetail.AdCategoryID, adDetail.SpecData)
}

// renderNewAdPage renders the new ad form page
// adCat: the ad category ID
// specData: pre-filled spec data (nil for new ad, map[string][]string for duplicate)
func renderNewAdPage(c *fiber.Ctx, adCat int, specData map[string][]string) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	maxImagesPerAd := config.MaxImagesPerAd
	csrfToken := local.GetCSRFToken(c)
	fields := ad.GetCategoryFields(adCat)

	fieldNodes := []g.Node{
		titleField(""),
	}

	// Render fields with values from specData if provided
	for _, field := range fields {
		selected := make(map[string]bool)
		var defaultValue string

		if specData != nil {
			// Get values from specData if they exist
			if fieldValues, exists := specData[field.Name()]; exists {
				// For multi-value fields, populate selected map
				for _, val := range fieldValues {
					selected[val] = true
				}
				// For single-value fields, set defaultValue
				if !field.IsMultiValue() && len(fieldValues) > 0 {
					defaultValue = fieldValues[0]
				}
			}
		}

		fieldNodes = append(fieldNodes, field.RenderNewAdField(selected, defaultValue, map[string]string{}))
	}

	fieldNodes = append(fieldNodes,
		imagesField(maxImagesPerAd),
		descriptionField(),
		submitButton(),
	)

	categoryName := ad.GetCategoryName(adCat)

	return render(c, ui.Page(
		"New Ad - Parts Pile",
		userID,
		userName,
		c.Path(),
		csrfToken,
		[]g.Node{
			pageHeaderWithSubtitle("Create New Ad", categoryName),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				hx.Post("/api/new-ad"),
				hx.Encoding("multipart/form-data"),
				hx.Target("#result"),
				g.Group(fieldNodes),
			),
			resultContainer(),
		},
	))
}

// titleField creates the title input field
func titleField(defaultValue string) g.Node {
	return Div(
		Class("space-y-2"),
		Label(For("title"), Class("block text-sm font-medium text-gray-700"), g.Text("Title")),
		Input(
			Type("text"),
			ID("title"),
			Name("title"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Required(),
			MaxLength("35"),
			Pattern("[\\x20-\\x7E]+"),
			Title("Title must be 1-35 characters, printable ASCII characters only"),
			g.Attr("oninput", "this.checkValidity()"),
			g.If(defaultValue != "", Value(defaultValue)),
		),
	)
}

// imagesField creates the images upload field
func imagesField(maxImagesPerAd int) g.Node {
	return Div(
		Class("space-y-2"),
		Label(Class("block text-sm font-medium text-gray-700"), g.Text("Images")),
		Div(
			Input(
				Type("file"),
				ID("images"),
				Name("images"),
				Class("hidden"),
				g.Attr("accept", "image/*"),
				g.Attr("multiple"),
				g.Attr("onchange", "previewImages(this)"),
			),
			Div(
				ID("upload-area"),
				Class("border rounded p-6 hover:border-blue-400 hover:bg-blue-50 transition-colors duration-200 cursor-pointer"),
				g.Attr("onclick", "handleUploadClick()"),
				g.Attr("ondragover", "event.preventDefault(); this.classList.add('border-blue-400', 'bg-blue-50')"),
				g.Attr("ondragleave", "this.classList.remove('border-blue-400', 'bg-blue-50')"),
				g.Attr("ondrop", "event.preventDefault(); this.classList.remove('border-blue-400', 'bg-blue-50'); handleDrop(event)"),
				Div(
					ID("upload-content"),
					Class("flex flex-col items-center space-y-4"),
					Div(
						Class("flex flex-col items-center space-y-2"),
						Div(
							Class("w-12 h-12 bg-blue-100 rounded-full flex items-center justify-center"),
							g.Raw(`<svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
							</svg>`),
						),
						Div(
							Class("text-lg font-medium text-gray-700"),
							g.Text("Upload Images"),
						),
						Div(
							Class("text-sm text-gray-500"),
							g.Text("Click to browse or drag and drop"),
						),
					),
				),
				Div(
					ID("image-preview"),
					Class("hidden image-preview flex flex-row flex-wrap gap-2 justify-center mt-4"),
				),
			),
			g.Raw(fmt.Sprintf(`<script>const MAX_IMAGES_PER_AD = %d;</script>`, maxImagesPerAd)),
			g.Raw(`<script src="/js/image-preview.js" defer></script>`),
		),
	)
}

// descriptionField creates the description textarea field
func descriptionField() g.Node {
	return descriptionFieldWithValue("")
}

// descriptionFieldWithValue creates the description textarea field with a value
func descriptionFieldWithValue(value string) g.Node {
	return Div(
		Class("space-y-2"),
		Label(For("description"), Class("block text-sm font-medium text-gray-700"), g.Text("Description")),
		Textarea(
			ID("description"),
			Name("description"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Required(),
			MaxLength("500"),
			Rows("4"),
			Pattern("[\\x20-\\x7E]+"),
			Title("Description must contain printable ASCII characters only"),
			g.Attr("oninput", "this.checkValidity()"),
			g.If(value != "", g.Text(value)),
		),
	)
}

// submitButton creates the submit button
func submitButton() g.Node {
	return Button(
		Type("submit"),
		Class("px-4 py-2 rounded inline-block bg-blue-500 text-white hover:bg-blue-600"),
		g.Text("Submit"),
	)
}

// pageHeaderWithSubtitle creates a page header with subtitle
func pageHeaderWithSubtitle(title string, subtitle string) g.Node {
	return Div(
		Class("mb-8"),
		H1(Class("text-4xl font-bold"), g.Text(title)),
		g.If(subtitle != "", P(Class("text-lg text-gray-600 mt-2 italic"), g.Text(subtitle))),
	)
}

// resultContainer creates the result container div
func resultContainer() g.Node {
	return Div(
		ID("result"),
		Class("mt-4"),
	)
}

// HandleNewAdSubmission processes the new ad form submission
func HandleNewAdSubmission(c *fiber.Ctx) error {
	userID := local.GetUserID(c)

	newAd, imageFiles, _, err := BuildAdFromForm(c, userID)
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
		vector.QueueAd(newAd.ID)
	} else {
		logger.Debug("Successfully processed ad inline",
			"component", "embedding", "adID", adID)
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
	fieldNode := currentField.RenderNewAdField(map[string]bool{}, "", parentValues)

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
