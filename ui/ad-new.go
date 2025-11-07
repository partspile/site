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

func NewAdPage(adCat int, userID int, userName, csrfToken string, path string, makes []string, categories []string, maxImagesPerAd int) g.Node {
	categoryName := ad.GetCategoryName(adCat)
	headerText := fmt.Sprintf("Create New %s Ad", categoryName)

	// Get category fields to dynamically render form
	fieldConfigs, err := spec.GetCategoryFields(adCat)
	if err != nil {
		// Fallback to empty fields if error
		fieldConfigs = []spec.FieldConfig{}
	}

	// Build dynamic field nodes
	fieldNodes := []g.Node{
		titleInputField(""),
	}

	// Render spec fields dynamically
	for i, fieldConfig := range fieldConfigs {
		selected := []string{}
		var defaultMake string
		if fieldConfig.Name == "make" {
			defaultMake = ""
		}
		fieldNodes = append(fieldNodes, renderFieldSelector(fieldConfig, i, fieldConfigs, selected, map[string][]string{}, makes, defaultMake))
	}

	// Add part category/subcategory (always shown)
	fieldNodes = append(fieldNodes,
		categoriesSelector(categories, ""),
		SubCategoriesSelector([]string{}, ""),
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
	makes []string,
	categories []string,
	originalAd ad.AdDetail,
	years []string,
	models []string,
	engines []string,
	subcategoryNames []string,
	selectedSubcategory string,
	maxImagesPerAd int,
) g.Node {
	adCat := originalAd.AdCategoryID

	// Get category fields to dynamically render form
	fieldConfigs, err := spec.GetCategoryFields(adCat)
	if err != nil {
		// Fallback to empty fields if error
		fieldConfigs = []spec.FieldConfig{}
	}

	// Build selected values map from AdDetail (temporary compatibility layer)
	selectedValues := map[string][]string{
		"year":   originalAd.Years,
		"model":  originalAd.Models,
		"engine": originalAd.Engines,
	}

	// Build dynamic field nodes
	fieldNodes := []g.Node{
		titleInputField(originalAd.Title),
	}

	// Render spec fields dynamically
	for i, fieldConfig := range fieldConfigs {
		selected := []string{}
		if sv, ok := selectedValues[fieldConfig.Name]; ok {
			selected = sv
		}
		var defaultMake string
		if fieldConfig.Name == "make" && originalAd.Make != "" {
			defaultMake = originalAd.Make
		}
		fieldNodes = append(fieldNodes, renderFieldSelector(fieldConfig, i, fieldConfigs, selected, selectedValues, makes, defaultMake))
	}

	// Add part category/subcategory
	fieldNodes = append(fieldNodes,
		categoriesSelector(categories, originalAd.PartCategory),
		SubCategoriesSelector(subcategoryNames, selectedSubcategory),
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
func renderFieldSelector(fieldConfig spec.FieldConfig, fieldIndex int, allFields []spec.FieldConfig, selectedValues []string, allSelectedValues map[string][]string, makes []string, defaultMake string) g.Node {
	fieldID := fieldConfig.Name + "Div"
	fieldName := fieldConfig.Name
	fieldNamePlural := fieldConfig.Name + "s" // For multi-value fields

	// Build HTMX include selector for parent fields
	var includeSelectors []string
	for i := 0; i < fieldIndex; i++ {
		parentField := allFields[i]
		if parentField.QdrantType == "keyword_array" {
			includeSelectors = append(includeSelectors, fmt.Sprintf("[name='%ss']:checked", parentField.Name))
		} else {
			includeSelectors = append(includeSelectors, fmt.Sprintf("[name='%s']", parentField.Name))
		}
	}

	// Determine next field to update
	var nextFieldName string
	var nextFieldTarget string
	if fieldIndex+1 < len(allFields) {
		nextField := allFields[fieldIndex+1]
		nextFieldName = nextField.Name
		nextFieldTarget = nextField.Name + "Div"
	} else {
		// After last spec field, update subcategories
		nextFieldTarget = "subcategoriesDiv"
	}

	if fieldConfig.QdrantType == "keyword_array" {
		// Multi-select (checkboxes) - years, models, engines
		selected := make(map[string]bool)
		for _, val := range selectedValues {
			selected[val] = true
		}

		checkboxes := []g.Node{}
		for _, value := range selectedValues {
			attrs := []g.Node{
				Checkbox(fieldNamePlural, value, value, selected[value], false),
			}

			// Add HTMX trigger if there's a next field
			if nextFieldTarget != "" {
				includeStr := strings.Join(append(includeSelectors, fmt.Sprintf("[name='%s']:checked", fieldNamePlural)), ",")
				attrs = append(attrs,
					hx.Trigger("change"),
					hx.Get(fmt.Sprintf("/api/%s", nextFieldName)),
					hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
					hx.Include(includeStr),
					hx.Swap("innerHTML"),
				)
				// Clear fields after this one
				if fieldIndex+2 < len(allFields) {
					clearTargets := []string{}
					for j := fieldIndex + 2; j < len(allFields); j++ {
						clearTargets = append(clearTargets, fmt.Sprintf("document.getElementById('%sDiv').innerHTML = ''", allFields[j].Name))
					}
					clearTargets = append(clearTargets, "document.getElementById('subcategoriesDiv').innerHTML = ''")
					attrs = append(attrs, g.Attr("onclick", strings.Join(clearTargets, "; ")))
				}
			}

			checkboxes = append(checkboxes, g.Group(attrs))
		}

		// For now, return empty div - values will be populated via HTMX
		return Div(
			ID(fieldID),
			Class("space-y-2"),
			formGroup(fieldConfig.DisplayName, fieldNamePlural, GridContainer4(checkboxes...)),
		)
	} else {
		// Single-select (dropdown) - make, breed
		if fieldConfig.Name == "make" {
			// Build clear targets script
			clearTargets := []string{}
			if fieldIndex+1 < len(allFields) {
				for j := fieldIndex + 1; j < len(allFields); j++ {
					clearTargets = append(clearTargets, fmt.Sprintf("document.getElementById('%sDiv').innerHTML = ''", allFields[j].Name))
				}
			}
			clearTargets = append(clearTargets, "document.getElementById('subcategoriesDiv').innerHTML = ''")
			clearTargetsScript := strings.Join(clearTargets, "; ")

			// Use makes from parameter
			return makeSelector(makes, defaultMake, nextFieldName, nextFieldTarget, clearTargetsScript)
		}

		// Generic single-select dropdown
		options := []g.Node{
			Option(Value(""), g.Text(fmt.Sprintf("Select a %s", fieldConfig.DisplayName))),
		}

		attrs := []g.Node{
			ID(fieldName),
			Name(fieldName),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
		}

		if fieldConfig.IsRequired {
			attrs = append(attrs, Required())
		}

		// Add HTMX trigger if there's a next field
		if nextFieldTarget != "" {
			includeStr := strings.Join(append(includeSelectors, fmt.Sprintf("[name='%s']", fieldName)), ",")
			attrs = append(attrs,
				hx.Trigger("change"),
				hx.Get(fmt.Sprintf("/api/%s", nextFieldName)),
				hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
				hx.Include(includeStr),
			)
			// Clear fields after this one
			if fieldIndex+2 < len(allFields) {
				clearTargets := []string{}
				for j := fieldIndex + 2; j < len(allFields); j++ {
					clearTargets = append(clearTargets, fmt.Sprintf("document.getElementById('%sDiv').innerHTML = ''", allFields[j].Name))
				}
				clearTargets = append(clearTargets, "document.getElementById('subcategoriesDiv').innerHTML = ''")
				attrs = append(attrs, g.Attr("onchange", strings.Join(clearTargets, "; ")))
			}
		}

		return formGroup(fieldConfig.DisplayName, fieldName,
			Select(append(attrs, g.Group(options))...),
		)
	}
}

func makeSelector(makes []string, defaultMake string, nextFieldName string, nextFieldTarget string, clearTargetsScript string) g.Node {
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		attrs := []g.Node{Value(makeName), g.Text(makeName)}
		if makeName == defaultMake {
			attrs = append(attrs, Selected())
		}
		makeOptions = append(makeOptions, Option(attrs...))
	}

	attrs := []g.Node{
		ID("make"),
		Name("make"),
		Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
		Required(),
	}

	// Add HTMX trigger if there's a next field
	if nextFieldTarget != "" {
		attrs = append(attrs,
			hx.Trigger("change"),
			hx.Get(fmt.Sprintf("/api/%s", nextFieldName)),
			hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
			hx.Include("this"),
		)
		if clearTargetsScript != "" {
			attrs = append(attrs, g.Attr("onchange", fmt.Sprintf("this.checkValidity(); %s", clearTargetsScript)))
		} else {
			attrs = append(attrs, g.Attr("onchange", "this.checkValidity()"))
		}
	}

	return formGroup("Make", "make",
		Select(append(attrs, Option(Value(""), g.Text("Select a make")), g.Group(makeOptions))...),
	)
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

func YearsSelector(years []string, selectedYears ...[]string) g.Node {
	selected := make(map[string]bool)
	if len(selectedYears) > 0 {
		for _, year := range selectedYears[0] {
			selected[year] = true
		}
	}

	checkboxes := []g.Node{}
	for _, year := range years {
		checkboxes = append(checkboxes,
			Checkbox("years", year, year, selected[year], false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
				hx.Swap("innerHTML"),
				g.Attr("onclick", "document.getElementById('enginesDiv').innerHTML = ''"),
			),
		)
	}
	return Div(
		ID("yearsDiv"),
		Class("space-y-2"),
		formGroup("Years", "years", GridContainer4(checkboxes...)),
	)
}

func ModelsSelector(models []string, selectedModels ...[]string) g.Node {
	selected := make(map[string]bool)
	if len(selectedModels) > 0 {
		for _, model := range selectedModels[0] {
			selected[model] = true
		}
	}

	checkboxes := []g.Node{}
	for _, model := range models {
		checkboxes = append(checkboxes,
			Checkbox("models", model, model, selected[model], false,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}
	return Div(
		ID("modelsDiv"),
		Class("space-y-2"),
		formGroup("Models", "models", GridContainer4(checkboxes...)),
	)
}

func EnginesSelector(engines []string, selectedEngines ...[]string) g.Node {
	selected := make(map[string]bool)
	if len(selectedEngines) > 0 {
		for _, engine := range selectedEngines[0] {
			selected[engine] = true
		}
	}

	checkboxes := []g.Node{}
	for _, engine := range engines {
		checkboxes = append(checkboxes,
			Checkbox("engines", engine, engine, selected[engine], false),
		)
	}
	return Div(
		ID("enginesDiv"),
		Class("space-y-2"),
		formGroup("Engines", "engines", GridContainer4(checkboxes...)),
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
		ID("subcategoriesDiv"),
		Class("space-y-2"),
		formGroup("Subcategory", "subcategory",
			Select(
				ID("subcategory"),
				Name("subcategory"),
				Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
				Required(),
				g.Group(options),
			),
		),
	)
}

func ModelsDivEmpty() g.Node {
	return Div(
		ID("modelsDiv"),
		Class("space-y-2"),
		Div(
			Class("p-4 bg-gray-50 border border-gray-200 rounded-lg"),
			Div(
				Class("text-gray-600 text-sm italic"),
				g.Text("No models available for all selected years"),
			),
		),
	)
}

func EnginesDivEmpty() g.Node {
	return Div(
		ID("enginesDiv"),
		Class("space-y-2"),
		Div(
			Class("p-4 bg-gray-50 border border-gray-200 rounded-lg"),
			Div(
				Class("text-gray-600 text-sm italic"),
				g.Text("No engines available for all selected year-model combinations"),
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
