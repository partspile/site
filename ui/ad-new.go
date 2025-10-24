package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
)

// ---- New Ad Page ----

func NewAdPage(userID int, userName string, path string, makes []string, categories []string) g.Node {
	return Page(
		"New Ad - Parts Pile",
		userID,
		userName,
		path,
		[]g.Node{
			pageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				hx.Post("/api/new-ad"),
				hx.Encoding("multipart/form-data"),
				hx.Target("#result"),
				titleInputField(""),
				makeSelector(makes, ""),
				YearsSelector([]string{}),
				ModelsSelector([]string{}),
				EnginesSelector([]string{}),
				categoriesSelector(categories, ""),
				SubCategoriesSelector([]string{}, ""),
				imagesInputField(),
				descriptionTextareaField(),
				priceInputField(),
				locationInputField(),
				button("Submit", withType("submit")),
			),
			resultContainer(),
		},
	)
}

// DuplicateAdPage renders the new ad form pre-filled with data from an existing ad
func DuplicateAdPage(
	userID int,
	userName string,
	path string,
	makes []string,
	categories []string,
	originalAd ad.AdDetail,
	years []string,
	models []string,
	engines []string,
	subcategoryNames []string,
	selectedSubcategory string,
) g.Node {
	categoryName := ""
	if originalAd.PartCategory.Valid {
		categoryName = originalAd.PartCategory.String
	}

	return Page(
		"Duplicate Ad - Parts Pile",
		userID,
		userName,
		path,
		[]g.Node{
			pageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				hx.Post("/api/new-ad"),
				hx.Encoding("multipart/form-data"),
				hx.Target("#result"),
				titleInputField(originalAd.Title),
				makeSelector(makes, originalAd.Make),
				YearsSelector(years, originalAd.Years),
				ModelsSelector(models, originalAd.Models),
				EnginesSelector(engines, originalAd.Engines),
				categoriesSelector(categories, categoryName),
				SubCategoriesSelector(subcategoryNames, selectedSubcategory),
				imagesInputField(),
				descriptionTextareaField(),
				priceInputField(),
				locationInputField(),
				button("Submit", withType("submit")),
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

func makeSelector(makes []string, defaultMake string) g.Node {
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		attrs := []g.Node{Value(makeName), g.Text(makeName)}
		if makeName == defaultMake {
			attrs = append(attrs, Selected())
		}
		makeOptions = append(makeOptions, Option(attrs...))
	}

	return formGroup("Make", "make",
		Select(
			ID("make"),
			Name("make"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Required(),
			hx.Trigger("change"),
			hx.Get("/api/years"),
			hx.Target("#yearsDiv"),
			hx.Include("this"),
			g.Attr("onchange", "this.checkValidity(); document.getElementById('modelsDiv').innerHTML = ''; document.getElementById('enginesDiv').innerHTML = ''; document.getElementById('subcategoriesDiv').innerHTML = '';"),
			Option(Value(""), g.Text("Select a make")),
			g.Group(makeOptions),
		),
	)
}

func imagesInputField() g.Node {
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
			g.Raw(fmt.Sprintf(`<script>const MAX_IMAGES_PER_AD = %d;</script>`, config.MaxImagesPerAd)),
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
