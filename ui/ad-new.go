package ui

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/spec"
)

// ---- New Ad Page ----

func NewAdPage(adCat int, userID int, userName, csrfToken string, path string, firstFieldValues map[string][]string, maxImagesPerAd int) g.Node {
	categoryName := ad.GetCategoryName(adCat)
	headerText := fmt.Sprintf("Create New %s Ad", categoryName)

	// Get category fields to dynamically render form
	fieldConfigs := spec.GetCategoryFields(adCat)

	// Build dynamic field nodes
	fieldNodes := []g.Node{
		titleInputField(""),
	}

	// Render spec fields dynamically
	// Render the first field of each chain independently
	// Subsequent fields in the same chain will be populated via HTMX
	for i, fieldConfig := range fieldConfigs {
		isFirstInChain := spec.IsFirstInChain(fieldConfig, fieldConfigs)

		if isFirstInChain {
			// Render the first field of this chain
			selected := []string{}
			var defaultValue string
			// Get pre-populated values for this field from the map
			fieldValues := firstFieldValues[fieldConfig.Name]
			fieldNodes = append(fieldNodes, renderFieldSelector(fieldConfig, i, fieldConfigs, selected, map[string][]string{}, fieldValues, defaultValue))
		} else {
			// Chained field - render empty div for HTMX to populate
			fieldID := fieldConfig.Name + "Div"
			if fieldConfig.QdrantType == "keyword_array" {
				fieldID = fieldConfig.Name + "sDiv"
			}
			fieldNodes = append(fieldNodes, Div(ID(fieldID)))
		}
	}

	// Add part category/subcategory (always shown)
	fieldNodes = append(fieldNodes,
		imagesInputField(maxImagesPerAd),
		descriptionTextareaField(),
		priceInputField(),
		locationInputField(),
		button("Submit", withType("submit")),
	)

	return Page(
		"New Ad - Parts Pile",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader(headerText),
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
	)
}

// DuplicateAdPage renders the new ad form pre-filled with data from an existing ad
func DuplicateAdPage(
	userID int,
	userName, csrfToken string,
	path string,
	originalAd ad.AdDetail,
	maxImagesPerAd int,
) g.Node {
	adCat := originalAd.AdCategoryID

	// Get category fields to dynamically render form
	fieldConfigs := spec.GetCategoryFields(adCat)

	// Initialize SpecData if nil
	if originalAd.SpecData == nil {
		originalAd.SpecData = make(map[string][]string)
	}

	// Build selected values map from AdDetail SpecData
	selectedValues := make(map[string][]string)
	for fieldName, fieldValues := range originalAd.SpecData {
		if len(fieldValues) > 0 {
			selectedValues[fieldName] = fieldValues
		}
	}

	// Build dynamic field nodes
	fieldNodes := []g.Node{
		titleInputField(originalAd.Title),
	}

	// Render spec fields dynamically
	// Build map of field values for first fields of each chain
	firstFieldValues := make(map[string][]string)
	for _, fieldConfig := range fieldConfigs {
		if spec.IsFirstInChain(fieldConfig, fieldConfigs) {
			// Get values for this first field (no parent filters)
			values, err := spec.GetFieldValues(adCat, fieldConfig.Name, map[string]string{})
			if err != nil {
				firstFieldValues[fieldConfig.Name] = []string{}
			} else {
				firstFieldValues[fieldConfig.Name] = values
			}
		}
	}

	for i, fieldConfig := range fieldConfigs {
		selected := []string{}
		if sv, ok := selectedValues[fieldConfig.Name]; ok {
			selected = sv
		}
		var defaultValue string
		// Get default value for single-select fields
		if fieldConfig.QdrantType != "keyword_array" {
			if fieldValues, exists := originalAd.SpecData[fieldConfig.Name]; exists && len(fieldValues) > 0 {
				defaultValue = fieldValues[0]
			}
		}
		// Get field values - use pre-loaded for first fields, empty for chained fields
		fieldValues := firstFieldValues[fieldConfig.Name]
		if len(fieldValues) == 0 && fieldConfig.NextInChain == 0 {
			// For standalone fields without pre-loaded values, use empty
			fieldValues = []string{}
		}
		fieldNodes = append(fieldNodes, renderFieldSelector(fieldConfig, i, fieldConfigs, selected, selectedValues, fieldValues, defaultValue))
	}

	// Add part category/subcategory
	fieldNodes = append(fieldNodes,
		imagesInputField(maxImagesPerAd),
		descriptionTextareaField(),
		priceInputField(),
		locationInputField(),
		button("Submit", withType("submit")),
	)

	return Page(
		"Duplicate Ad - Parts Pile",
		userID,
		userName,
		path,
		csrfToken,
		[]g.Node{
			pageHeader("Create New Ad"),
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
	)
}

func titleInputField(defaultValue string) g.Node {
	return formGroup("Title", "title",
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

// renderFieldSelector renders a field selector dynamically based on field configuration
func renderFieldSelector(fieldConfig spec.FieldConfig, fieldIndex int, allFields []spec.FieldConfig, selectedValues []string, allSelectedValues map[string][]string, fieldValues []string, defaultMake string) g.Node {
	fieldID := fieldConfig.Name + "Div"
	fieldName := fieldConfig.Name

	// Build HTMX include selector for parent fields
	var includeSelectors []string
	for i := 0; i < fieldIndex; i++ {
		parentField := allFields[i]
		if parentField.QdrantType == "keyword_array" {
			includeSelectors = append(includeSelectors, fmt.Sprintf("[name='%s']:checked", parentField.Name))
		} else {
			includeSelectors = append(includeSelectors, fmt.Sprintf("[name='%s']", parentField.Name))
		}
	}

	// Determine next field to update using chain information
	var nextFieldName string
	var nextFieldTarget string
	if fieldConfig.NextInChain != 0 {
		// This field is chained to another field
		nextFieldName = spec.GetNextFieldName(fieldConfig, allFields)
		// Use plural form for multi-value fields
		nextFieldDef, exists := spec.GetFieldDefinition(nextFieldName)
		if exists && nextFieldDef.QdrantType == "keyword_array" {
			nextFieldTarget = nextFieldName + "sDiv"
		} else {
			nextFieldTarget = nextFieldName + "Div"
		}
	} else if fieldIndex+1 < len(allFields) {
		// No chain, but there's a next field - check if it's in a different chain
		nextField := allFields[fieldIndex+1]
		if !spec.FieldsInSameChain(fieldConfig, nextField, allFields) {
			// Next field is in a different chain
			if nextField.QdrantType == "keyword_array" {
				nextFieldTarget = nextField.Name + "sDiv"
			} else {
				nextFieldTarget = nextField.Name + "Div"
			}
			nextFieldName = nextField.Name
		} else {
			// Next field is in the same chain (shouldn't happen if chain logic is correct)
			if nextField.QdrantType == "keyword_array" {
				nextFieldTarget = nextField.Name + "sDiv"
			} else {
				nextFieldTarget = nextField.Name + "Div"
			}
			nextFieldName = nextField.Name
		}
	}

	if fieldConfig.QdrantType == "keyword_array" {
		// Multi-select (checkboxes)
		selected := make(map[string]bool)
		for _, val := range selectedValues {
			selected[val] = true
		}

		checkboxes := []g.Node{}
		for _, value := range fieldValues {
			attrs := []g.Node{
				Checkbox(fieldName, value, value, selected[value], false),
			}

			// Add HTMX trigger if there's a next field
			if nextFieldTarget != "" && nextFieldName != "" {
				includeStr := strings.Join(append(includeSelectors, fmt.Sprintf("[name='%s']:checked", fieldName)), ",")

				// Use singular field name for endpoint (consistent with field config)
				endpoint := fmt.Sprintf("/api/%s", nextFieldName)

				attrs = append(attrs,
					hx.Trigger("change"),
					hx.Get(endpoint),
					hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
					hx.Include(includeStr),
					hx.Swap("innerHTML"),
				)
				// HTMX will automatically replace content, no need to manually clear
			}

			checkboxes = append(checkboxes, g.Group(attrs))
		}

		return Div(
			ID(fieldID),
			Class("space-y-2"),
			formGroup(fieldConfig.DisplayName, fieldName, GridContainer4(checkboxes...)),
		)
	} else {
		// Single-select (dropdown)
		// Use generic selector for all single-select fields
		options := []g.Node{
			Option(Value(""), g.Text(fmt.Sprintf("Select a %s", fieldConfig.DisplayName))),
		}

		// Add options from fieldValues
		for _, value := range fieldValues {
			optionAttrs := []g.Node{Value(value), g.Text(value)}
			if value == defaultMake {
				optionAttrs = append(optionAttrs, Selected())
			}
			options = append(options, Option(optionAttrs...))
		}

		attrs := []g.Node{
			ID(fieldName),
			Name(fieldName),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
		}

		if fieldConfig.IsRequired {
			attrs = append(attrs, Required())
		}

		// Add HTMX trigger if there's a next field in chain
		if nextFieldTarget != "" && nextFieldName != "" {
			includeStr := strings.Join(append(includeSelectors, fmt.Sprintf("[name='%s']", fieldName)), ",")

			// Use singular field name for endpoint (consistent with field config)
			endpoint := fmt.Sprintf("/api/%s", nextFieldName)

			attrs = append(attrs,
				hx.Trigger("change"),
				hx.Get(endpoint),
				hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
				hx.Include(includeStr),
				hx.Swap("innerHTML"),
			)
			// HTMX will automatically replace content, no need to manually clear
		}

		return formGroup(fieldConfig.DisplayName, fieldName,
			Select(append(attrs, g.Group(options))...),
		)
	}
}

func imagesInputField(maxImagesPerAd int) g.Node {
	return formGroup("Images", "images",
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

func descriptionTextareaField() g.Node {
	return formGroup("Description", "description",
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
		),
	)
}

func priceInputField() g.Node {
	return formGroup("Price", "price",
		Input(
			Type("number"),
			ID("price"),
			Name("price"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Required(),
			Step("1"),
			Min("0"),
			g.Attr("title", "Price must be >= 0"),
			g.Attr("oninput", "this.checkValidity()"),
		),
	)
}

func locationInputField() g.Node {
	return formGroup("Location", "location",
		Input(
			Type("text"),
			ID("location"),
			Name("location"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Placeholder("(Optional)"),
		),
	)
}

// GenericMultiSelectField creates a generic multi-select field (checkboxes) for any field
func GenericMultiSelectField(fieldName, displayName string, values []string) g.Node {
	fieldID := fieldName + "sDiv"
	checkboxes := []g.Node{}
	for _, value := range values {
		checkboxes = append(checkboxes,
			Checkbox(fieldName, value, value, false, false),
		)
	}
	return Div(
		ID(fieldID),
		Class("space-y-2"),
		formGroup(displayName, fieldName, GridContainer4(checkboxes...)),
	)
}

func categoriesSelector(categories []string, selectedCategory string) g.Node {
	options := []g.Node{
		Option(Value(""), g.Text("Select a category")),
	}

	for _, category := range categories {
		attrs := []g.Node{Value(category), g.Text(category)}
		if category == selectedCategory {
			attrs = append(attrs, Selected())
		}
		options = append(options, Option(attrs...))
	}

	return formGroup("Category", "category",
		Select(
			ID("category"),
			Name("category"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Required(),
			hx.Trigger("change"),
			hx.Get("/api/subcategories"),
			hx.Target("#subcategoriesDiv"),
			hx.Include("this"),
			g.Group(options),
		),
	)
}

func SubCategoriesSelector(subCategoryNames []string, selectedSubCategory string) g.Node {
	options := []g.Node{
		Option(Value(""), g.Text("Select a subcategory")),
	}

	for _, subCategoryName := range subCategoryNames {
		attrs := []g.Node{Value(subCategoryName), g.Text(subCategoryName)}
		if subCategoryName == selectedSubCategory {
			attrs = append(attrs, Selected())
		}
		options = append(options, Option(attrs...))
	}

	return Div(
		ID("part_subcategoryDiv"),
		Class("space-y-2"),
		formGroup("Subcategory", "part_subcategory",
			Select(
				ID("part_subcategory"),
				Name("part_subcategory"),
				Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
				Required(),
				g.Group(options),
			),
		),
	)
}

// newAdButton renders the new ad button based on user login
func newAdButton(userID int) g.Node {
	if userID != 0 {
		return button("New Ad", withHref("/new-ad"))
	}
	return button("New Ad", withDisabled())
}
