package ui

import (
	"fmt"
	"sort"
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
	// Prepare make options
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		attrs := []g.Node{Value(makeName)}
		if makeName == currentAd.Make {
			attrs = append(attrs, Selected())
		}
		attrs = append(attrs, g.Text(makeName))
		makeOptions = append(makeOptions, Option(attrs...))
	}

	// Prepare year checkboxes
	yearCheckboxes := []g.Node{}
	for _, year := range years {
		isChecked := false
		for _, adYear := range currentAd.Years {
			if year == adYear {
				isChecked = true
				break
			}
		}
		yearCheckboxes = append(yearCheckboxes,
			Checkbox("years", year, year, isChecked, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
			),
		)
	}

	// Prepare model checkboxes
	modelCheckboxes := []g.Node{}
	sortedModels := make([]string, len(models))
	copy(sortedModels, models)
	sort.Strings(sortedModels)

	for _, modelName := range sortedModels {
		isChecked := false
		for _, adModel := range currentAd.Models {
			if modelName == adModel {
				isChecked = true
				break
			}
		}
		modelCheckboxes = append(modelCheckboxes,
			Checkbox("models", modelName, modelName, isChecked, false,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
			),
		)
	}

	// Prepare engine checkboxes
	engineCheckboxes := []g.Node{}
	sortedEngines := make([]string, len(engines))
	copy(sortedEngines, engines)
	sort.Strings(sortedEngines)

	for _, engineName := range sortedEngines {
		isChecked := false
		for _, adEngine := range currentAd.Engines {
			if engineName == adEngine {
				isChecked = true
				break
			}
		}
		engineCheckboxes = append(engineCheckboxes,
			Checkbox("engines", engineName, engineName, isChecked, false),
		)
	}

	// Define htmxTarget for this form
	htmxTarget := fmt.Sprintf("#ad-%d", currentAd.ID)

	return Page(
		"Edit Ad - Parts Pile",
		currentUser,
		path,
		[]g.Node{
			pageHeader("Edit Ad"),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				hx.Post(fmt.Sprintf("/api/update-ad/%d", currentAd.ID)),
				hx.Encoding("multipart/form-data"),
				hx.Target(htmxTarget),
				hx.Swap("outerHTML"),
				editTitleInputField(currentAd.Title),
				editMakeSelectField(makes, currentAd.Make),
				formGroup("Years", "years", Div(ID("yearsDiv"), GridContainer4(yearCheckboxes...))),
				formGroup("Models", "models", Div(ID("modelsDiv"), GridContainer4(modelCheckboxes...))),
				formGroup("Engines", "engines", Div(ID("enginesDiv"), GridContainer4(engineCheckboxes...))),
				catogorySelector(categories, func() string {
					if currentAd.Category.Valid {
						return currentAd.Category.String
					}
					return ""
				}()),
				Div(
					ID("subcategoriesDiv"),
					Class("space-y-2"),
					// Show subcategories if they exist
					func() g.Node {
						if len(subcategories) > 0 {
							return SubCategoriesFormGroup(subcategories, "")
						}
						return g.Text("")
					}(),
				),
				editImagesInputField(currentAd.ID, currentAd.ImageCount),
				editDescriptionTextareaField(currentAd.Description),
				editPriceInputField(currentAd.Price),
				editLocationInputField(func() string {
					if currentAd.City.Valid {
						return currentAd.City.String
					}
					return ""
				}()),
				styledButton("Submit", buttonPrimary,
					Type("submit"),
				),
				g.Raw(`<script src="/js/image-preview.js" defer></script>`),
				g.Raw(`<script src="/js/image-edit.js" defer></script>`),
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
			),
			Div(ID("image-preview")),
		),
	)
}

func editDescriptionTextareaField(currentDescription string) g.Node {
	return formGroup("Description", "description",
		Textarea(
			ID("description"),
			Name("description"),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
			Rows("4"),
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

// ---- Edit Ad Partial (for inline editing) ----

func AdEditPartial(adObj ad.Ad, makes, years, models, engines, categories, subcategories []string, cancelTarget, htmxTarget string, view ...string) g.Node {
	editForm := Div(
		ID(fmt.Sprintf("ad-%d", adObj.ID)),
		Class("border p-4 mb-4 rounded bg-white shadow-lg relative"),
		Form(
			ID("editAdForm"),
			Class("space-y-6"),
			hx.Post(fmt.Sprintf("/api/update-ad/%d", adObj.ID)),
			hx.Encoding("multipart/form-data"),
			hx.Target(htmxTarget),
			hx.Swap("outerHTML"),
			formGroup("Title", "title",
				Input(
					Type("text"),
					ID("title"),
					Name("title"),
					Class("w-full p-2 border rounded"),
					Required(),
					Value(adObj.Title),
				),
			),
			formGroup("Make", "make",
				Select(
					ID("make"),
					Name("make"),
					Class("w-full p-2 border rounded"),
					hx.Trigger("change"),
					hx.Get("/api/years"),
					hx.Target("#yearsDiv"),
					hx.Include("this"),
					g.Attr("onchange", "document.getElementById('modelsDiv').innerHTML = ''; document.getElementById('enginesDiv').innerHTML = '';"),
					Option(Value(""), g.Text("Select a make")),
					g.Group(func() []g.Node {
						makeOptions := []g.Node{}
						for _, makeName := range makes {
							attrs := []g.Node{Value(makeName)}
							if makeName == adObj.Make {
								attrs = append(attrs, Selected())
							}
							attrs = append(attrs, g.Text(makeName))
							makeOptions = append(makeOptions, Option(attrs...))
						}
						return makeOptions
					}()),
				),
			),
			formGroup("Years", "years", Div(ID("yearsDiv"), func() g.Node {
				yearCheckboxes := []g.Node{}
				for _, year := range years {
					isChecked := false
					for _, adYear := range adObj.Years {
						if year == adYear {
							isChecked = true
							break
						}
					}
					yearCheckboxes = append(yearCheckboxes,
						Checkbox("years", year, year, isChecked, false,
							hx.Trigger("change"),
							hx.Get("/api/models"),
							hx.Target("#modelsDiv"),
							hx.Include("[name='make'],[name='years']:checked"),
						),
					)
				}
				return GridContainer4(yearCheckboxes...)
			}())),
			formGroup("Models", "models", Div(ID("modelsDiv"), func() g.Node {
				modelCheckboxes := []g.Node{}
				sortedModels := make([]string, len(models))
				copy(sortedModels, models)
				sort.Strings(sortedModels)

				for _, modelName := range sortedModels {
					isChecked := false
					for _, adModel := range adObj.Models {
						if modelName == adModel {
							isChecked = true
							break
						}
					}
					modelCheckboxes = append(modelCheckboxes,
						Checkbox("models", modelName, modelName, isChecked, false,
							hx.Trigger("change"),
							hx.Get("/api/engines"),
							hx.Target("#enginesDiv"),
							hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
						),
					)
				}
				return GridContainer4(modelCheckboxes...)
			}())),
			formGroup("Engines", "engines", Div(ID("enginesDiv"), func() g.Node {
				engineCheckboxes := []g.Node{}
				sortedEngines := make([]string, len(engines))
				copy(sortedEngines, engines)
				sort.Strings(sortedEngines)

				for _, engineName := range sortedEngines {
					isChecked := false
					for _, adEngine := range adObj.Engines {
						if engineName == adEngine {
							isChecked = true
							break
						}
					}
					engineCheckboxes = append(engineCheckboxes,
						Checkbox("engines", engineName, engineName, isChecked, false),
					)
				}
				return GridContainer4(engineCheckboxes...)
			}())),
			catogorySelector(categories, func() string {
				if adObj.Category.Valid {
					return adObj.Category.String
				}
				return ""
			}()),
			Div(
				ID("subcategoriesDiv"),
				Class("space-y-2"),
				func() g.Node {
					if len(subcategories) > 0 {
						return SubCategoriesFormGroup(subcategories, "")
					}
					return g.Text("")
				}(),
			),
			formGroup("Images", "images",
				Div(
					Div(
						ID("image-gallery"),
						Class("flex flex-row gap-2 mb-2"),
						g.Group(func() []g.Node {
							imageNodes := []g.Node{}
							imageURLs := AdImageURLs(adObj.ID, adObj.ImageCount)
							for i, url := range imageURLs {
								imageIdx := i + 1
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
					Input(
						Type("hidden"),
						ID("deleted_images"),
						Name("deleted_images"),
						Value(""),
					),
					Input(
						Type("file"),
						ID("images"),
						Name("images"),
						Class("w-full p-2 border rounded"),
						g.Attr("accept", "image/*"),
						g.Attr("multiple"),
					),
					Div(ID("image-preview")),
				),
			),
			formGroup("Description", "description",
				Textarea(
					ID("description"),
					Name("description"),
					Class("w-full p-2 border rounded"),
					Rows("4"),
					g.Text(adObj.Description),
				),
			),
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
			Div(
				Class("flex gap-2"),
				styledButton("Save", buttonPrimary, Type("submit")),
				styledButton("Cancel", ButtonSecondary,
					hx.Get(cancelTarget),
					hx.Target(htmxTarget),
					hx.Swap("outerHTML"),
				),
			),
			g.Raw(`<script src="/js/image-preview.js" defer></script>`),
			g.Raw(`<script src="/js/image-edit.js" defer></script>`),
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
