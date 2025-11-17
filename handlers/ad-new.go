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
	"github.com/parts-pile/site/spec"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// HandleNewAd shows the new ad form
func HandleNewAd(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)
	adCat := cookie.GetAdCategory(c)
	maxImagesPerAd := config.MaxImagesPerAd
	csrfToken := local.GetCSRFToken(c)
	fields := spec.GetCategoryFields(adCat)

	// Build map of first field of each chain -> values
	firstFieldValues := make(map[string][]string)

	for _, field := range fields {
		if field.IsFirstInChain() {
			// Get values for this first field (no parent filters)
			values, err := field.GetValues(map[string]string{})
			if err != nil {
				logger.Warn("Failed to get field values",
					"error", err, "component", "ad-new", "fieldName", field.Name(), "adCat", adCat)
				values = []string{}
			}
			firstFieldValues[field.Name()] = values
		}
	}

	return render(c, ui.NewAdPage(adCat, userID, userName, csrfToken, c.Path(), fields, firstFieldValues, maxImagesPerAd))
}

// HandleDuplicateAd shows the duplicate ad form with pre-filled data
func HandleDuplicateAd(c *fiber.Ctx) error {
	userID := local.GetUserID(c)
	userName := local.GetUserName(c)

	adID, err := AdID(c)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ad ID")
	}

	// Fetch the original ad
	adDetail, err := ad.GetAdDetailByID(adID, userID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Ad not found")
	}

	// Get category fields
	fields := spec.GetCategoryFields(adDetail.AdCategoryID)

	csrfToken := local.GetCSRFToken(c)

	return render(c, ui.DuplicateAdPage(
		userID, userName, csrfToken, c.Path(), *adDetail, fields, config.MaxImagesPerAd))
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

// HandlePartCategory handles the part category dropdown for new ad form
func HandlePartCategory(c *fiber.Ctx) error {
	return HandleFieldValues(c, "part_category")
}

// HandlePartSubcategory handles the part subcategory dropdown for new ad form
func HandlePartSubcategory(c *fiber.Ctx) error {
	return HandleFieldValues(c, "part_subcategory")
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
	fields := spec.GetCategoryFields(adCat)

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
			if !spec.FieldsInSameChain(parentField, currentField, fields) {
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

			endpoint := fmt.Sprintf("/api/%s", nextFieldName)
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
