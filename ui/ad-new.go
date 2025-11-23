package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func NewAdPage(adCat int, userID int, userName, csrfToken string, path string,
	fieldNodes []g.Node, maxImagesPerAd int) g.Node {

	allFieldNodes := []g.Node{
		titleField(),
	}
	allFieldNodes = append(allFieldNodes, fieldNodes...)
	allFieldNodes = append(allFieldNodes,
		imagesField(maxImagesPerAd),
		descriptionField(),
		SubmitButton(),
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
				g.Group(allFieldNodes),
			),
			resultContainer(),
		},
	)
}

func titleField() g.Node {
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
		),
	)
}

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

func descriptionField() g.Node {
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
		),
	)
}

func SubmitButton() g.Node {
	return Button(
		Type("submit"),
		Class("px-4 py-2 rounded inline-block bg-blue-500 text-white hover:bg-blue-600"),
		g.Text("Submit"),
	)
}

// newAdButton renders the new ad button based on user login
func newAdButton(userID int) g.Node {
	if userID != 0 {
		return button("New Ad", withHref("/new-ad"))
	}
	return button("New Ad", withDisabled())
}
