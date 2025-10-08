package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/user"
)

// ---- New Ad Page ----

func NewAdPage(currentUser *user.User, path string, makes []string, categories []string) g.Node {
	return Page(
		"New Ad - Parts Pile",
		currentUser,
		path,
		[]g.Node{
			pageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				hx.Post("/api/new-ad"),
				hx.Encoding("multipart/form-data"),
				hx.Target("#result"),
				titleInputField(),
				makeSelector(makes),
				YearsDiv(),
				ModelsDiv(),
				EnginesDiv(),
				categoriesSelector(categories, ""),
				SubcategoriesDiv(),
				imagesInputField(),
				descriptionTextareaField(),
				priceInputField(),
				locationInputField(),
				styledButton("Submit", buttonPrimary,
					Type("submit"),
				),
			),
			resultContainer(),
		},
	)
}

func titleInputField() g.Node {
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
		),
	)
}

func makeSelector(makes []string) g.Node {
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		makeOptions = append(makeOptions,
			Option(Value(makeName), g.Text(makeName)),
		)
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
				Class("border-2 border-dashed border-gray-300 rounded-lg p-6 hover:border-blue-400 hover:bg-blue-50 transition-colors duration-200 cursor-pointer"),
				g.Attr("onclick", "document.getElementById('images').click()"),
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

func YearsDiv() g.Node {
	return Div(
		ID("yearsDiv"),
		Class("space-y-2"),
	)
}

func ModelsDiv() g.Node {
	return Div(
		ID("modelsDiv"),
		Class("space-y-2"),
	)
}

func EnginesDiv() g.Node {
	return Div(
		ID("enginesDiv"),
		Class("space-y-2"),
	)
}

func SubcategoriesDiv() g.Node {
	return Div(
		ID("subcategoriesDiv"),
		Class("space-y-2"),
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
