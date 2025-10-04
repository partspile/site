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
				validationErrorContainer(),
				titleInputField(),
				makeSelectField(makes),
				Div(
					ID("yearsDiv"),
					Class("space-y-2"),
				),
				Div(
					ID("modelsDiv"),
					Class("space-y-2"),
				),
				Div(
					ID("enginesDiv"),
					Class("space-y-2"),
				),
				CategoriesFormGroup(categories, ""),
				Div(
					ID("subcategoriesDiv"),
					Class("space-y-2"),
				),
				imagesInputField(),
				descriptionTextareaField(),
				priceInputField(),
				locationInputField(),
				styledButton("Submit", buttonPrimary,
					Type("submit"),
				),
				g.Raw(`<script src="/js/image-preview.js" defer></script>`),
			),
			resultContainer(),
		},
	)
}

// ---- New Ad Form Field Components ----

func titleInputField() g.Node {
	return formGroup("Title", "title",
		Input(
			Type("text"),
			ID("title"),
			Name("title"),
			Class("w-full p-2 border rounded invalid:border-red-500 invalid:bg-red-50 valid:border-emerald-500 valid:bg-emerald-50"),
			Required(),
			MaxLength("35"),
			Pattern("[\\x20-\\x7E]+"),
			Title("Title must be 1-35 characters, printable ASCII characters only"),
			g.Attr("oninput", "this.checkValidity()"),
		),
	)
}

func makeSelectField(makes []string) g.Node {
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
			Class("w-full p-2 border rounded invalid:border-red-500 invalid:bg-red-50 valid:border-emerald-500 valid:bg-emerald-50"),
			hx.Trigger("change"),
			hx.Get("/api/years"),
			hx.Target("#yearsDiv"),
			hx.Include("this"),
			g.Attr("onchange", "document.getElementById('modelsDiv').innerHTML = ''; document.getElementById('enginesDiv').innerHTML = '';"),
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
				Class("w-full p-2 border rounded invalid:border-red-500 invalid:bg-red-50 valid:border-emerald-500 valid:bg-emerald-50"),
				g.Attr("accept", "image/*"),
				g.Attr("multiple"),
			),
			Div(ID("image-preview")),
		),
	)
}

func descriptionTextareaField() g.Node {
	return formGroup("Description", "description",
		Textarea(
			ID("description"),
			Name("description"),
			Class("w-full p-2 border rounded invalid:border-red-500 invalid:bg-red-50 valid:border-emerald-500 valid:bg-emerald-50"),
			Rows("4"),
		),
	)
}

func priceInputField() g.Node {
	return formGroup("Price", "price",
		Input(
			Type("number"),
			ID("price"),
			Name("price"),
			Class("w-full p-2 border rounded invalid:border-red-500 invalid:bg-red-50 valid:border-emerald-500 valid:bg-emerald-50"),
			Step("0.01"),
			Min("0"),
		),
	)
}

func locationInputField() g.Node {
	return formGroup("Location", "location",
		Input(
			Type("text"),
			ID("location"),
			Name("location"),
			Class("w-full p-2 border rounded invalid:border-red-500 invalid:bg-red-50 valid:border-emerald-500 valid:bg-emerald-50"),
			Placeholder("(Optional)"),
		),
	)
}
