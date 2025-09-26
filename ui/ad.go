package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/user"
)

func adID(ad ad.Ad) string {
	return fmt.Sprintf("ad-%d", ad.ID)
}

func adTarget(ad ad.Ad) string {
	return fmt.Sprintf("closest #%s", adID(ad))
}

// priceNode returns price text
func priceNode(ad ad.Ad) g.Node {
	return g.Text(fmt.Sprintf("$%.0f", ad.Price))
}

// titleNode returns title text without styling
func titleNode(ad ad.Ad) g.Node {
	return g.Text(ad.Title)
}

// Returns the Unicode flag for a given country code (e.g., "US" -> 🇺🇸)
func countryFlag(country string) string {
	if len(country) != 2 {
		return ""
	}
	code := strings.ToUpper(country)
	return string(rune(int32(code[0])-'A'+0x1F1E6)) + string(rune(int32(code[1])-'A'+0x1F1E6))
}

// locationFlagNode returns a Div containing flag and location text
func locationFlagNode(ad ad.Ad) g.Node {
	var city string
	if ad.City.Valid {
		city = ad.City.String
	}
	var adminArea string
	if ad.AdminArea.Valid {
		adminArea = ad.AdminArea.String
	}
	var country string
	if ad.Country.Valid {
		country = ad.Country.String
	}

	// Return nil if no location data
	if city == "" && adminArea == "" && country == "" {
		return nil
	}

	// Build location text
	var locationText string
	if city != "" && adminArea != "" {
		locationText = city + ", " + adminArea
	} else if city != "" {
		locationText = city
	} else if adminArea != "" {
		locationText = adminArea
	}

	// Return Div with flag and location text
	return Div(
		Class("flex items-center"),
		g.Text(countryFlag(country)),
		Span(Class("ml-1"), g.Text(locationText)),
	)
}

// Helper to format ad age as Xm, Xh, Xd, Xmo, or Xy Xmo
func formatAdAge(t time.Time) string {
	d := time.Since(t)
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}

	days := int(d.Hours() / 24)
	if days <= 31 {
		return fmt.Sprintf("%dd", days)
	}

	// Calculate months and years
	now := time.Now()
	years := now.Year() - t.Year()
	months := int(now.Month()) - int(t.Month())

	// Adjust for day of month
	if now.Day() < t.Day() {
		months--
	}

	// Adjust years if months went negative
	if months < 0 {
		years--
		months += 12
	}

	if years > 0 {
		if months > 0 {
			return fmt.Sprintf("%dy %dmo", years, months)
		}
		return fmt.Sprintf("%dy", years)
	}

	return fmt.Sprintf("%dmo", months)
}

// ageNode returns age text
func ageNode(ad ad.Ad, loc *time.Location) g.Node {
	agoStr := formatAdAge(ad.CreatedAt.In(loc))
	return g.Text(agoStr)
}

func bookmarkIcon(bookmarked bool) g.Node {
	icon := "/images/bookmark-false.svg"
	if bookmarked {
		icon = "/images/bookmark-true.svg"
	}
	return Img(
		Src(icon),
		Class("inline w-6 h-6 align-middle"),
		Alt("Bookmark"),
	)
}

// BookmarkButton returns the bookmark toggle button
func BookmarkButton(ad ad.Ad) g.Node {
	var hxMethod g.Node
	if ad.Bookmarked {
		hxMethod = hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID))
	} else {
		hxMethod = hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID))
	}

	return Button(
		Type("button"),
		Class("focus:outline-none"),
		hxMethod,
		hx.Target("this"),
		hx.Swap("outerHTML"),
		ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
		g.Attr("onclick", "event.stopPropagation()"),
		bookmarkIcon(ad.Bookmarked),
	)
}

func AdPage(adObj ad.Ad, currentUser *user.User, userID int, path string, loc *time.Location, view string) g.Node {
	return Page(
		fmt.Sprintf("Ad %d - Parts Pile", adObj.ID),
		currentUser,
		path,
		[]g.Node{
			AdDetail(adObj, loc, userID, view),
		},
	)
}

// ---- Ad Components ----

// AdEditPartial renders the ad edit form for inline editing
func AdEditPartial(adObj ad.Ad, makes, years []string, modelAvailability, engineAvailability map[string]bool, categories, subcategories []string, cancelTarget, htmxTarget string, view ...string) g.Node {
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
			validationErrorContainer(),
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
			formGroup("Years", "years", Div(ID("yearsDiv"), GridContainer(5, func() []g.Node {
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
				return yearCheckboxes
			}()...))),
			formGroup("Models", "models", Div(ID("modelsDiv"), GridContainer(5, func() []g.Node {
				modelCheckboxes := []g.Node{}
				models := make([]string, 0, len(modelAvailability))
				for m := range modelAvailability {
					models = append(models, m)
				}
				sort.Strings(models)
				for _, modelName := range models {
					isAvailable := modelAvailability[modelName]
					isChecked := false
					for _, adModel := range adObj.Models {
						if modelName == adModel {
							isChecked = true
							break
						}
					}
					modelCheckboxes = append(modelCheckboxes,
						Checkbox("models", modelName, modelName, isChecked, !isAvailable,
							hx.Trigger("change"),
							hx.Get("/api/engines"),
							hx.Target("#enginesDiv"),
							hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
						),
					)
				}
				return modelCheckboxes
			}()...))),
			formGroup("Engines", "engines", Div(ID("enginesDiv"), GridContainer(5, func() []g.Node {
				engineCheckboxes := []g.Node{}
				engines := make([]string, 0, len(engineAvailability))
				for e := range engineAvailability {
					engines = append(engines, e)
				}
				sort.Strings(engines)
				for _, engineName := range engines {
					isAvailable := engineAvailability[engineName]
					isChecked := false
					for _, adEngine := range adObj.Engines {
						if engineName == adEngine {
							isChecked = true
							break
						}
					}
					engineCheckboxes = append(engineCheckboxes,
						Checkbox("engines", engineName, engineName, isChecked, !isAvailable),
					)
				}
				return engineCheckboxes
			}()...))),
			CategoriesFormGroup(categories, func() string {
				if adObj.Category.Valid {
					return adObj.Category.String
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
			formGroup("Images", "images",
				Div(
					// New: Image gallery for existing images
					Div(
						ID("image-gallery"),
						Class("flex flex-row gap-2 mb-2"),
						g.Group(func() []g.Node {
							imageNodes := []g.Node{}
							for i := 1; i <= adObj.ImageCount; i++ {
								imageNodes = append(imageNodes,
									Div(
										Class("relative group"),
										g.Attr("data-image-idx", fmt.Sprintf("%d", i)),
										AdImageWithFallbackSrcSet(adObj.ID, i, fmt.Sprintf("Image %d", i), "grid"),
										Button(
											Type("button"),
											Class("absolute top-0 right-0 bg-white bg-opacity-80 rounded-full p-1 text-red-600 hover:text-red-800 z-10 delete-image-btn"),
											g.Attr("onclick", fmt.Sprintf("deleteImage(this, %d)", i)),
											Img(Src("/images/trashcan.svg"), Alt("Delete"), Class("w-4 h-4")),
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
			formGroup("Location", "location",
				Input(
					Type("text"),
					ID("location"),
					Name("location"),
					Class("w-full p-2 border rounded"),
					Placeholder("(Optional)"),
				),
			),
			Div(
				Class("flex flex-row gap-2 justify-end"),
				Button(
					Type("button"),
					Class("text-gray-400 hover:text-gray-700 text-2xl font-bold focus:outline-none z-20"),
					hx.Get(cancelTarget),
					hx.Target(htmxTarget),
					hx.Swap("outerHTML"),
					g.Text("×"),
				),
				styledButton("Save", buttonPrimary,
					Type("submit"),
				),
			),
		),
	)
	return editForm
}

// ---- Ad Pages ----

func NewAdPage(currentUser *user.User, path string, makes []string, categories []string) g.Node {
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		makeOptions = append(makeOptions,
			Option(Value(makeName), g.Text(makeName)),
		)
	}

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
				formGroup("Title", "title",
					Input(
						Type("text"),
						ID("title"),
						Name("title"),
						Class("w-full p-2 border rounded"),
						Required(),
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
						g.Group(makeOptions),
					),
				),
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
				formGroup("Images", "images",
					Div(
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
					),
				),
				formGroup("Location", "location",
					Input(
						Type("text"),
						ID("location"),
						Name("location"),
						Class("w-full p-2 border rounded"),
						Placeholder("(Optional)"),
					),
				),
				styledButton("Submit", buttonPrimary,
					Type("submit"),
				),
				g.Raw(`<script src="/js/image-preview.js" defer></script>`),
			),
			resultContainer(),
		},
	)
}

func EditAdPage(currentUser *user.User, path string, currentAd ad.Ad, makes []string, years []string, modelAvailability map[string]bool, engineAvailability map[string]bool, categories []string, subcategories []string) g.Node {
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
	models := make([]string, 0, len(modelAvailability))
	for m := range modelAvailability {
		models = append(models, m)
	}
	sort.Strings(models)

	for _, modelName := range models {
		isAvailable := modelAvailability[modelName]
		isChecked := false
		for _, adModel := range currentAd.Models {
			if modelName == adModel {
				isChecked = true
				break
			}
		}
		modelCheckboxes = append(modelCheckboxes,
			Checkbox("models", modelName, modelName, isChecked, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
			),
		)
	}

	// Prepare engine checkboxes
	engineCheckboxes := []g.Node{}
	engines := make([]string, 0, len(engineAvailability))
	for e := range engineAvailability {
		engines = append(engines, e)
	}
	sort.Strings(engines)

	for _, engineName := range engines {
		isAvailable := engineAvailability[engineName]
		isChecked := false
		for _, adEngine := range currentAd.Engines {
			if engineName == adEngine {
				isChecked = true
				break
			}
		}
		engineCheckboxes = append(engineCheckboxes,
			Checkbox("engines", engineName, engineName, isChecked, !isAvailable),
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
				validationErrorContainer(),
				formGroup("Title", "title",
					Input(
						Type("text"),
						ID("title"),
						Name("title"),
						Class("w-full p-2 border rounded"),
						Required(),
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
						g.Group(makeOptions),
					),
				),
				formGroup("Years", "years", Div(ID("yearsDiv"), GridContainer(5, yearCheckboxes...))),
				formGroup("Models", "models", Div(ID("modelsDiv"), GridContainer(5, modelCheckboxes...))),
				formGroup("Engines", "engines", Div(ID("enginesDiv"), GridContainer(5, engineCheckboxes...))),
				CategoriesFormGroup(categories, func() string {
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
				formGroup("Images", "images",
					Div(
						// New: Image gallery for existing images
						Div(
							ID("image-gallery"),
							Class("flex flex-row gap-2 mb-2"),
							g.Group(func() []g.Node {
								imageNodes := []g.Node{}
								imageURLs := AdImageURLs(currentAd.ID, currentAd.ImageCount)
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
												Img(Src("/images/trashcan.svg"), Alt("Delete"), Class("w-4 h-4")),
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
						Value(fmt.Sprintf("%.2f", currentAd.Price)),
					),
				),
				formGroup("Location (Zipcode)", "location",
					Input(
						Type("text"),
						ID("location"),
						Name("location"),
						Class("w-full p-2 border rounded"),
						Placeholder("Optional zipcode, e.g. 90210"),
					),
				),
				styledButton("Submit", buttonPrimary,
					Type("submit"),
				),
				g.Raw(`<script src="/js/image-preview.js" defer></script>`),
				g.Raw(`<script src="/js/image-edit.js" defer></script>`),
			),
		},
	)
}

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
		urls = append(urls, fmt.Sprintf(
			"https://f004.backblazeb2.com/file/parts-pile/%d/%d-160w.webp?Authorization=%s",
			adID, i, token,
		))
	}
	return urls
}

// Helper to generate signed B2 image URLs for an ad and all sizes
func AdImageSrcSet(adID int, idx int, context string) (src, srcset string) {
	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		// Return empty strings when B2 images aren't available - browser will show broken image
		return "", ""
	}

	base := fmt.Sprintf("https://f004.backblazeb2.com/file/parts-pile/%d/%d", adID, idx)
	src160 := fmt.Sprintf("%s-160w.webp?Authorization=%s 160w", base, token)
	src480 := fmt.Sprintf("%s-480w.webp?Authorization=%s 480w", base, token)
	src1200 := fmt.Sprintf("%s-1200w.webp?Authorization=%s 1200w", base, token)
	srcset = strings.Join([]string{src160, src480, src1200}, ", ")

	// Choose default src based on context
	switch context {
	case "thumbnail":
		src = fmt.Sprintf("%s-160w.webp?Authorization=%s", base, token)
	case "carousel":
		src = fmt.Sprintf("%s-1200w.webp?Authorization=%s", base, token)
	default:
		src = fmt.Sprintf("%s-480w.webp?Authorization=%s", base, token) // default
	}
	return src, srcset
}

// Helper to render an image with srcset and fallback
func AdImageWithFallbackSrcSet(adID int, idx int, alt string, context string) g.Node {
	src, srcset := AdImageSrcSet(adID, idx, context)

	// Set appropriate sizes attribute based on context
	var sizes string
	switch context {
	case "thumbnail":
		sizes = "64px"
	case "carousel":
		sizes = "(max-width: 640px) 300px, (max-width: 768px) 400px, (max-width: 1024px) 500px, 600px"
	default:
		sizes = "(max-width: 600px) 160px, (max-width: 900px) 480px, 1200px"
	}

	return Img(
		Src(src),
		Alt(alt),
		g.Attr("srcset", srcset),
		g.Attr("sizes", sizes),
		Class("object-contain w-full aspect-square bg-gray-100"),
	)
}
