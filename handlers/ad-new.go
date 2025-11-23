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

		fieldNodes = append(fieldNodes, field.RenderNewAdField(selected, defaultValue))
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
	return handleFieldValues(c, fieldName)
}

// handleFieldValues handles field value requests generically for any field
func handleFieldValues(c *fiber.Ctx, fieldName string) error {
	adCat := cookie.GetAdCategory(c)

	// Get parent field values from query parameters
	q, err := url.ParseQuery(string(c.Request().URI().QueryString()))
	if err != nil {
		return err
	}

	// Build parent values map
	parentValues := make(map[string]string)
	fields := ad.GetCategoryFields(adCat)

	// Find current field index
	currentFieldIndex := -1
	for i, f := range fields {
		if f.Config().Name == fieldName {
			currentFieldIndex = i
			break
		}
	}
	if currentFieldIndex == -1 {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid field name")
	}

	// Get parent field values (fields that come before current field)
	// For chained fields, only include parents from the same chain
	currentField := fields[currentFieldIndex]
	currentConfig := currentField.Config()
	for i := 0; i < currentFieldIndex; i++ {
		parentField := fields[i]
		parentConfig := parentField.Config()

		// If current field is in a chain, only include parents from the same chain
		if currentConfig.NextInChain != 0 || currentField.IsFirstInChain() {
			if !ad.FieldsInSameChain(parentField, currentField, fields) {
				// Skip fields from different chains
				continue
			}
		}

		// Use field name directly (no plural form)
		queryKey := parentConfig.Name

		// Check for single value or multiple values
		if values, exists := q[queryKey]; exists && len(values) > 0 {
			// For multi-value fields (years, models, engines), join with comma
			// For single-value fields (make), use first value
			if parentConfig.FieldMultiValue {
				parentValues[parentConfig.Name] = strings.Join(values, ",")
			} else {
				parentValues[parentConfig.Name] = values[0]
			}
		}
	}

	// Check if required parent fields are present
	for i := 0; i < currentFieldIndex; i++ {
		parentField := fields[i]
		parentConfig := parentField.Config()
		if parentConfig.IsRequired {
			if _, exists := parentValues[parentConfig.Name]; !exists {
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

	// Find downstream fields in the same chain that need to be cleared
	downstreamFields := spec.GetDownstreamFields(currentField, fields)

	if fieldDef.FieldMultiValue {
		// Multi-select (checkboxes) - years, models, engines
		mainNode := buildMultiSelectFieldNode(fieldName, values)
		return renderWithDownstreamClears(c, mainNode, downstreamFields, fields)
	} else {
		// Single select (dropdown) - make, part_category, part_subcategory, breed
		// For part_category and part_subcategory, we need to render with HTMX triggers
		mainNode := buildSingleSelectFieldNode(fieldName, values, currentField, fields, currentFieldIndex)
		return renderWithDownstreamClears(c, mainNode, downstreamFields, fields)
	}
}

// buildMultiSelectFieldNode builds the node for a multi-select field
func buildMultiSelectFieldNode(fieldName string, values []string) g.Node {
	// Use generic multi-select component
	fieldDef, exists := spec.GetFieldDefinition(fieldName)
	displayName := fieldName
	if exists {
		displayName = fieldDef.DisplayName
	}
	return ui.GenericMultiSelectField(fieldName, displayName, values)
}

// buildSingleSelectFieldNode builds the node for a single-select field
func buildSingleSelectFieldNode(fieldName string, values []string, currentField spec.Field, allFields []spec.Field, currentFieldIndex int) g.Node {
	currentConfig := currentField.Config()
	// For chained single-select fields, render as dropdowns with HTMX triggers
	if currentConfig.NextInChain != 0 {
		// Create dropdown options
		options := []g.Node{
			Option(Value(""), g.Text(fmt.Sprintf("Select a %s", getFieldDisplayName(fieldName)))),
		}
		for _, value := range values {
			options = append(options, Option(Value(value), g.Text(value)))
		}

		fieldNameAttr := fieldName
		displayName := getFieldDisplayName(fieldName)

		// Build HTMX attributes if there's a next field in chain
		var selectAttrs []g.Node
		selectAttrs = append(selectAttrs,
			ID(fieldNameAttr),
			Name(fieldNameAttr),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
		)

		// Add HTMX trigger if there's a next field in chain
		if currentConfig.NextInChain != 0 {
			nextFieldName := spec.GetNextFieldName(currentField, allFields)
			nextFieldDef, exists := spec.GetFieldDefinition(nextFieldName)
			var nextFieldTarget string
			if exists && nextFieldDef.FieldMultiValue {
				nextFieldTarget = nextFieldName + "sDiv"
			} else {
				nextFieldTarget = nextFieldName + "Div"
			}

			// Build include selector for parent fields
			var includeSelectors []string
			for i := 0; i < currentFieldIndex; i++ {
				parentField := allFields[i]
				parentConfig := parentField.Config()
				if parentConfig.FieldMultiValue {
					includeSelectors = append(includeSelectors, fmt.Sprintf("[name='%ss']:checked", parentConfig.Name))
				} else {
					includeSelectors = append(includeSelectors, fmt.Sprintf("[name='%s']", parentConfig.Name))
				}
			}
			includeSelectors = append(includeSelectors, fmt.Sprintf("[name='%s']", fieldNameAttr))
			includeStr := strings.Join(includeSelectors, ",")

			endpoint := fmt.Sprintf("/api/field/%s", nextFieldName)
			selectAttrs = append(selectAttrs,
				hx.Trigger("change"),
				hx.Get(endpoint),
				hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
				hx.Include(includeStr),
				hx.Swap("innerHTML"),
			)
		}

		// Build form group structure similar to ui.formGroup
		// Note: Don't wrap in outer div with ID - HTMX swaps innerHTML into existing div
		return Div(
			Div(
				Class("mb-2"),
				Label(
					Class("block text-sm font-medium text-gray-700"),
					g.Text(displayName),
				),
			),
			Select(append(selectAttrs, g.Group(options)...)...),
		)
	}
	// For other single-select fields (like make), return empty div (they're rendered statically in ui)
	return Div()
}

// getFieldDisplayName returns the display name for a field
func getFieldDisplayName(fieldName string) string {
	fieldDef, exists := spec.GetFieldDefinition(fieldName)
	if exists {
		return fieldDef.DisplayName
	}
	// Fallback to capitalized field name
	return strings.Title(strings.ReplaceAll(fieldName, "_", " "))
}

// renderWithDownstreamClears returns the main content plus empty divs for downstream fields
// Uses HTMX's hx-swap-oob to swap into multiple targets
func renderWithDownstreamClears(c *fiber.Ctx, mainNode g.Node, downstreamFields []string, fields []spec.Field) error {
	if len(downstreamFields) == 0 {
		// No downstream fields to clear, just render main content
		return render(c, mainNode)
	}

	// Create a map of field names to their div IDs
	fieldIDMap := make(map[string]string)
	for _, field := range fields {
		fieldConfig := field.Config()
		divID := fieldConfig.Name + "Div"
		if fieldConfig.FieldMultiValue {
			divID = fieldConfig.Name + "sDiv"
		}
		fieldIDMap[fieldConfig.Name] = divID
	}

	// Build empty divs for downstream fields with hx-swap-oob
	var emptyDivs []g.Node
	for _, fieldName := range downstreamFields {
		divID, exists := fieldIDMap[fieldName]
		if !exists {
			continue
		}
		// Create empty div with hx-swap-oob attribute
		emptyDivs = append(emptyDivs, Div(
			g.Attr("id", divID),
			g.Attr("hx-swap-oob", "true"),
		))
	}

	// Combine main content with empty divs
	// HTMX will swap the main content into the target and the oob divs into their targets
	var allNodes []g.Node
	allNodes = append(allNodes, mainNode)
	allNodes = append(allNodes, emptyDivs...)
	return render(c, g.Group(allNodes))
}

// renderEmptyFieldSelector returns empty selector for a field
func renderEmptyFieldSelector(c *fiber.Ctx, fieldName string) error {
	fieldDef, exists := spec.GetFieldDefinition(fieldName)
	if !exists {
		return fiber.NewError(fiber.StatusBadRequest, "Field definition not found")
	}

	if fieldDef.FieldMultiValue {
		return renderMultiSelectField(c, fieldName, []string{})
	} else {
		return renderSingleSelectField(c, fieldName, []string{})
	}
}

// renderMultiSelectField renders a multi-select field (checkboxes)
func renderMultiSelectField(c *fiber.Ctx, fieldName string, values []string) error {
	fieldDef, exists := spec.GetFieldDefinition(fieldName)
	displayName := fieldName
	if exists {
		displayName = fieldDef.DisplayName
	}
	return render(c, ui.GenericMultiSelectField(fieldName, displayName, values))
}

// renderSingleSelectField renders a single-select field (dropdown)
func renderSingleSelectField(c *fiber.Ctx, fieldName string, values []string) error {
	// For now, return empty - single select fields are rendered statically
	// TODO: create generic single-select component
	return c.SendString("")
}

// HandleSubCategories handles the subcategories dropdown for new ad form
// Now uses dynamic field system - part_subcategory is chained from part_category
func HandleSubCategories(c *fiber.Ctx) error {
	// Get part_category from query parameter
	partCategory := c.Query("part_category")
	if partCategory == "" {
		// Also check for legacy "category" parameter for backward compatibility
		partCategory = c.Query("category")
	}

	if partCategory == "" {
		// Return empty div when part_category is not selected
		return render(c, ui.SubCategoriesSelector([]string{}, ""))
	}

	// Use dynamic field system to get part_subcategory values
	adCat := cookie.GetAdCategory(c)
	parentValues := map[string]string{
		"part_category": partCategory,
	}

	values, err := spec.GetFieldValues(adCat, "part_subcategory", parentValues)
	if err != nil {
		logger.Warn("Failed to get part subcategories",
			"error", err, "component", "subcategories", "adCat", adCat, "partCategory", partCategory)
		return render(c, ui.SubCategoriesSelector([]string{}, ""))
	}

	return render(c, ui.SubCategoriesSelector(values, ""))
}
