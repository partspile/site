package ui

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/user"
)

// ---- Edit Ad Page ----

func EditAdPage(currentUser *user.User, path string, currentAd ad.Ad, makes, years, models, engines, categories, subcategories []string) g.Node {
	// Define htmxTarget for this form
	htmxTarget := fmt.Sprintf("#ad-%d", currentAd.ID)

	return Page(
		"Edit Ad - Parts Pile",
		currentUser,
		path,
		[]g.Node{
			pageHeader("Edit Ad"),
			// Show current ad details (read-only)
			Div(
				Class("space-y-4 mb-6 p-4 bg-gray-50 rounded border"),
				H2(Class("text-xl font-bold mb-4"), g.Text("Current Ad Details")),
				Div(Class("grid grid-cols-2 gap-4"),
					Div(
						Class("space-y-2"),
						Div(Class("font-semibold"), g.Text("Title:")),
						Div(g.Text(currentAd.Title)),
					),
					Div(
						Class("space-y-2"),
						Div(Class("font-semibold"), g.Text("Make:")),
						Div(g.Text(currentAd.Make)),
					),
					Div(
						Class("space-y-2"),
						Div(Class("font-semibold"), g.Text("Years:")),
						Div(g.Text(strings.Join(currentAd.Years, ", "))),
					),
					Div(
						Class("space-y-2"),
						Div(Class("font-semibold"), g.Text("Models:")),
						Div(g.Text(strings.Join(currentAd.Models, ", "))),
					),
					Div(
						Class("space-y-2"),
						Div(Class("font-semibold"), g.Text("Engines:")),
						Div(g.Text(strings.Join(currentAd.Engines, ", "))),
					),
					func() g.Node {
						if currentAd.Category.Valid {
							return Div(
								Class("space-y-2"),
								Div(Class("font-semibold"), g.Text("Category:")),
								Div(g.Text(currentAd.Category.String)),
							)
						}
						return g.Text("")
					}(),
				),
				Div(
					Class("mt-4"),
					Div(Class("font-semibold mb-2"), g.Text("Current Description:")),
					Div(Class("whitespace-pre-wrap"), g.Text(currentAd.Description)),
				),
			),
			// Editable fields
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				hx.Post(fmt.Sprintf("/api/update-ad/%d", currentAd.ID)),
				hx.Encoding("multipart/form-data"),
				hx.Target(htmxTarget),
				hx.Swap("outerHTML"),
				H2(Class("text-xl font-bold"), g.Text("Edit Ad")),
				editPriceInputField(currentAd.Price),
				editLocationInputField(func() string {
					if currentAd.City.Valid {
						return currentAd.City.String
					}
					return ""
				}()),
				editDescriptionAdditionTextareaField(),
				styledButton("Submit", buttonPrimary,
					Type("submit"),
				),
			),
		},
	)
}

// ---- Edit Ad Form Field Components ----

func editTitleInputField(currentTitle string) g.Node {
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
			Value(currentTitle),
		),
	)
}

func editMakeSelectField(makes []string, currentMake string) g.Node {
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		attrs := []g.Node{Value(makeName)}
		if makeName == currentMake {
			attrs = append(attrs, Selected())
		}
		attrs = append(attrs, g.Text(makeName))
		makeOptions = append(makeOptions, Option(attrs...))
	}

	return formGroup("Make", "make",
		Select(
			ID("make"),
			Name("make"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
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

func editImagesInputField(adID int, imageCount int) g.Node {
	return formGroup("Images", "images",
		Div(
			// Image gallery for existing images
			Div(
				ID("image-gallery"),
				Class("flex flex-row gap-2 mb-2"),
				g.Group(func() []g.Node {
					imageNodes := []g.Node{}
					imageURLs := AdImageURLs(adID, imageCount)
					for i, url := range imageURLs {
						imageIdx := i + 1 // Images are numbered 1, 2, 3...
						imageNodes = append(imageNodes,
							Div(
								Class("relative group"),
								g.Attr("data-image-idx", fmt.Sprintf("%d", imageIdx)),
								Img(
									Src(url),
									Alt(fmt.Sprintf("Image %d", imageIdx)),
									Class("object-cover w-24 h-24 rounded border cursor-move"),
									g.Attr("draggable", "true"),
								),
								Button(
									Type("button"),
									Class("absolute top-0 right-0 bg-white bg-opacity-80 rounded-full p-1 text-red-600 hover:text-red-800 z-10 delete-image-btn"),
									g.Attr("onclick", fmt.Sprintf("deleteImage(this, %d)", imageIdx)),
									Img(Src("/images/trashcan.svg"), Alt("Delete"), Class("w-6 h-6")),
								),
							),
						)
					}
					return imageNodes
				}()),
			),
			// Hidden input for deleted images (comma-separated indices)
			Input(
				Type("hidden"),
				ID("deleted_images"),
				Name("deleted_images"),
				Value(""),
			),
			// File input for adding new images
			Input(
				Type("file"),
				ID("images"),
				Name("images"),
				Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
				g.Attr("accept", "image/*"),
				g.Attr("multiple"),
				g.Attr("onchange", "previewImages(this)"),
			),
			Div(ID("image-preview")),
			g.Raw(`<script src="/js/image-preview.js" defer></script>`),
		),
	)
}

func editDescriptionTextareaField(currentDescription string) g.Node {
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
			g.Text(currentDescription),
		),
	)
}

func editPriceInputField(currentPrice float64) g.Node {
	return formGroup("Price", "price",
		Input(
			Type("number"),
			ID("price"),
			Name("price"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Step("0.01"),
			Min("0"),
			Value(fmt.Sprintf("%.2f", currentPrice)),
		),
	)
}

func editLocationInputField(currentLocation string) g.Node {
	return formGroup("Location (Zipcode)", "location",
		Input(
			Type("text"),
			ID("location"),
			Name("location"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Placeholder("Optional zipcode, e.g. 90210"),
			Value(func() string {
				if currentLocation != "" {
					return currentLocation
				}
				return ""
			}()),
		),
	)
}

func editDescriptionAdditionTextareaField() g.Node {
	return formGroup("Add to Description", "description_addition",
		Div(
			Textarea(
				ID("description_addition"),
				Name("description_addition"),
				Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
				MaxLength("500"),
				Rows("4"),
				Pattern("[\\x20-\\x7E\\n]+"),
				Title("Addition must contain printable ASCII characters only"),
				Placeholder("Add additional information (will be timestamped and appended)"),
				g.Attr("oninput", "this.checkValidity()"),
			),
			Div(
				Class("text-sm text-gray-600 mt-1"),
				g.Text("Note: This will be appended to the current description with a timestamp. Total description must remain under 500 characters."),
			),
		),
	)
}

// ---- Edit Ad Partial (for inline editing) ----

func AdEditPartial(adObj ad.Ad, makes, years, models, engines, categories, subcategories []string, cancelTarget, htmxTarget string, view ...string) g.Node {
	editForm := Div(
		ID(fmt.Sprintf("ad-%d", adObj.ID)),
		Class("border p-4 mb-4 rounded bg-white shadow-lg relative"),
		// Show current ad details (read-only)
		Div(
			Class("space-y-4 mb-6 p-4 bg-gray-50 rounded border"),
			H3(Class("text-lg font-bold mb-4"), g.Text("Current Ad Details")),
			Div(Class("grid grid-cols-2 gap-4 text-sm"),
				Div(
					Class("space-y-1"),
					Div(Class("font-semibold"), g.Text("Title:")),
					Div(g.Text(adObj.Title)),
				),
				Div(
					Class("space-y-1"),
					Div(Class("font-semibold"), g.Text("Make:")),
					Div(g.Text(adObj.Make)),
				),
				Div(
					Class("space-y-1"),
					Div(Class("font-semibold"), g.Text("Years:")),
					Div(g.Text(strings.Join(adObj.Years, ", "))),
				),
				Div(
					Class("space-y-1"),
					Div(Class("font-semibold"), g.Text("Models:")),
					Div(g.Text(strings.Join(adObj.Models, ", "))),
				),
				Div(
					Class("space-y-1"),
					Div(Class("font-semibold"), g.Text("Engines:")),
					Div(g.Text(strings.Join(adObj.Engines, ", "))),
				),
				func() g.Node {
					if adObj.Category.Valid {
						return Div(
							Class("space-y-1"),
							Div(Class("font-semibold"), g.Text("Category:")),
							Div(g.Text(adObj.Category.String)),
						)
					}
					return g.Text("")
				}(),
			),
			Div(
				Class("mt-4"),
				Div(Class("font-semibold mb-2"), g.Text("Current Description:")),
				Div(Class("whitespace-pre-wrap text-sm"), g.Text(adObj.Description)),
			),
		),
		// Editable fields
		Form(
			ID("editAdForm"),
			Class("space-y-6"),
			hx.Post(fmt.Sprintf("/api/update-ad/%d", adObj.ID)),
			hx.Encoding("multipart/form-data"),
			hx.Target(htmxTarget),
			hx.Swap("outerHTML"),
			H3(Class("text-lg font-bold"), g.Text("Edit Ad")),
			formGroup("Price", "price",
				Input(
					Type("number"),
					ID("price"),
					Name("price"),
					Class("w-full p-2 border rounded"),
					Step("0.01"),
					Min("0"),
					Value(fmt.Sprintf("%.2f", adObj.Price)),
				),
			),
			formGroup("Location (Zipcode)", "location",
				Input(
					Type("text"),
					ID("location"),
					Name("location"),
					Class("w-full p-2 border rounded"),
					Placeholder("Optional zipcode, e.g. 90210"),
					Value(func() string {
						if adObj.City.Valid {
							return adObj.City.String
						}
						return ""
					}()),
				),
			),
			formGroup("Add to Description", "description_addition",
				Div(
					Textarea(
						ID("description_addition"),
						Name("description_addition"),
						Class("w-full p-2 border rounded"),
						MaxLength("500"),
						Rows("4"),
						Placeholder("Add additional information (will be timestamped and appended)"),
					),
					Div(
						Class("text-sm text-gray-600 mt-1"),
						g.Text("Note: This will be appended to the current description with a timestamp. Total description must remain under 500 characters."),
					),
				),
			),
			Div(
				Class("flex gap-2"),
				styledButton("Save", buttonPrimary, Type("submit")),
				styledButton("Cancel", ButtonSecondary,
					hx.Get(cancelTarget),
					hx.Target(htmxTarget),
					hx.Swap("outerHTML"),
				),
			),
		),
	)

	return editForm
}

// ---- Image Helper Functions ----

// Helper to generate signed B2 image URLs for an ad
func AdImageURLs(adID int, imageCount int) []string {
	urls := []string{}

	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		// Return empty strings when B2 images aren't available - browser will show broken images
		for i := 0; i < imageCount; i++ {
			urls = append(urls, "")
		}
		return urls
	}

	for i := 1; i <= imageCount; i++ {
		// Use 160w size for gallery thumbnails
		urls = append(urls, config.GetB2ImageURL(adID, i, "160w", token))
	}
	return urls
}

func AdImageSrcSet(adID int, idx int, context string) (src, srcset string) {
	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		return "", ""
	}

	src = config.GetB2ImageURL(adID, idx, "320w", token)

	// Generate srcset for different sizes
	sizes := []string{"320w", "640w", "1280w"}
	srcsetParts := []string{}
	for _, size := range sizes {
		srcsetParts = append(srcsetParts, fmt.Sprintf("%s %s", config.GetB2ImageURL(adID, idx, size, token), size))
	}
	srcset = strings.Join(srcsetParts, ", ")

	return src, srcset
}

func AdImageWithFallbackSrcSet(adID int, idx int, alt string, context string) g.Node {
	src, srcset := AdImageSrcSet(adID, idx, context)
	if src == "" {
		// Return a placeholder when B2 images aren't available
		return Div(
			Class("bg-gray-200 flex items-center justify-center text-gray-500"),
			g.Text("Image unavailable"),
		)
	}

	return Img(
		Src(src),
		SrcSet(srcset),
		Alt(alt),
		Class("w-full h-auto"),
	)
}
