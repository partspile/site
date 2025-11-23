package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

// ---- New Ad Page ----

func NewAdPage(adCat int, userID int, userName, csrfToken string, path string,
	fields []ad.Field, maxImagesPerAd int) g.Node {

	// Build dynamic field nodes
	fieldNodes := []g.Node{
		titleField(""),
	}

	// Render spec fields dynamically
	// Render the first field of each chain independently
	// Subsequent fields in the same chain will be populated via HTMX
	for _, field := range fields {
		selected := make(map[string]bool)
		fieldNodes = append(fieldNodes, field.RenderNewAdField(selected, ""))
	}

	fieldNodes = append(fieldNodes,
		imagesField(maxImagesPerAd),
		descriptionField(),
		button("Submit", withType("submit")),
	)

	categoryName := ad.GetCategoryName(adCat)

	return Page(
		"New Ad - Parts Pile",
		userID,
		userName,
		path,
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
	)
}

// DuplicateAdPage renders the new ad form pre-filled with data from an existing ad
func DuplicateAdPage(userID int, userName, csrfToken string, path string,
	originalAd ad.AdDetail, fields []ad.Field, maxImagesPerAd int) g.Node {

	// Build dynamic field nodes
	fieldNodes := []g.Node{
		titleField(originalAd.Title),
	}

	// Render spec fields dynamically
	// RenderNewAdField handles determining if field is first in chain and renders accordingly
	for _, field := range fields {
		selected := make(map[string]bool)
		if sv, ok := originalAd.SpecData[field.Name()]; ok {
			for _, val := range sv {
				selected[val] = true
			}
		}
		var defaultValue string
		// Get default value for single-select fields
		if !field.IsMultiValue() {
			if fieldValues, exists := originalAd.SpecData[field.Name()]; exists && len(fieldValues) > 0 {
				defaultValue = fieldValues[0]
			}
		}
		fieldNodes = append(fieldNodes, field.RenderNewAdField(selected, defaultValue))
	}

	fieldNodes = append(fieldNodes,
		imagesField(maxImagesPerAd),
		descriptionField(),
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

func titleField(defaultValue string) g.Node {
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

func imagesField(maxImagesPerAd int) g.Node {
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

func descriptionField() g.Node {
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
