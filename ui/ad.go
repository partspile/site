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

// Returns the Unicode flag for a given country code (e.g., "US" -> 🇺🇸)
func countryFlag(country string) string {
	if len(country) != 2 {
		return ""
	}
	code := strings.ToUpper(country)
	return string(rune(int32(code[0])-'A'+0x1F1E6)) + string(rune(int32(code[1])-'A'+0x1F1E6))
}

// Returns the display string and flag for a location, per the table
func formatLocationDisplay(city, adminArea, country string) (string, g.Node) {
	city = strings.TrimSpace(city)
	adminArea = strings.TrimSpace(adminArea)
	country = strings.TrimSpace(country)

	flag := countryFlag(country)
	var flagNode g.Node
	if flag != "" {
		flagNode = Span(Style("font-size: 1.5em; vertical-align: middle;"), g.Text(flag))
	} else {
		flagNode = nil
	}

	if city == "" && adminArea == "" && country == "" {
		return "", nil
	}
	if city == "" && adminArea == "" && country != "" {
		return "", flagNode
	}
	if city == "" && adminArea != "" && country == "" {
		return adminArea, nil
	}
	if city == "" && adminArea != "" && country != "" {
		return adminArea, flagNode
	}
	if city != "" && adminArea == "" && country == "" {
		return city, nil
	}
	if city != "" && adminArea == "" && country != "" {
		return city, flagNode
	}
	if city != "" && adminArea != "" && country == "" {
		return city + ", " + adminArea, nil
	}
	if city != "" && adminArea != "" && country != "" {
		return city + ", " + adminArea, flagNode
	}
	return "", nil
}

// Helper to get display location string and flag for an ad
func getDisplayLocationAndFlag(adObj ad.Ad) (string, g.Node) {
	// Use the fields directly from adObj instead of DB lookup
	return formatLocationDisplay(adObj.City, adObj.AdminArea, adObj.Country)
}

// Helper to get raw location input for an ad
func getRawLocation(adObj ad.Ad) string {
	_, _, _, raw, _, _, err := ad.GetLocation(adObj.LocationID)
	if err != nil {
		return ""
	}
	return raw
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

// Helper function to return flag icon and country text in a div with text-xs
func LocationDisplayWithFlag(adObj ad.Ad) g.Node {
	locationStr, flagNode := getDisplayLocationAndFlag(adObj)
	if locationStr == "" && flagNode == nil {
		return nil
	}

	// Create smaller flag icon (same size as grid view)
	var smallFlagNode g.Node
	if flagNode != nil {
		smallFlagNode = Span(Style("font-size: 1em; vertical-align: middle;"), g.Text(countryFlag(adObj.Country)))
	}

	return Div(
		Class("text-xs text-gray-500 flex items-center gap-1"),
		smallFlagNode,
		g.If(locationStr != "" || smallFlagNode != nil, g.Text(locationStr)),
	)
}

// Helper function to return age div with text-xs
func AgeDisplay(posted time.Time) g.Node {
	agoStr := formatAdAge(posted)
	return Div(
		Class("text-xs text-gray-400"),
		g.Text(agoStr),
	)
}

// formatCategoryDisplay formats the category for display
func formatCategoryDisplay(ad ad.Ad) g.Node {
	if ad.Category == "" {
		return g.Node(nil)
	}

	return Div(
		Class("text-sm text-gray-500 -mt-1 mb-3"),
		g.Text(ad.Category),
	)
}

// ---- Ad Components ----

func AdDetails(adObj ad.Ad) g.Node {
	sortedYears := append([]string{}, adObj.Years...)
	sortedModels := append([]string{}, adObj.Models...)
	sortedEngines := append([]string{}, adObj.Engines...)
	sort.Strings(sortedYears)
	sort.Strings(sortedModels)
	sort.Strings(sortedEngines)

	locationStr, flagNode := getDisplayLocationAndFlag(adObj)
	var locationNode g.Node = nil
	if locationStr != "" || flagNode != nil {
		locationNode = P(Class("text-gray-600"),
			Span(Class("inline-flex items-center gap-1"), flagNode, g.Text(locationStr)),
		)
	}

	return Div(
		Class("mb-4"),
		P(Class("mt-4"), g.Text(adObj.Description)),
		P(Class("text-2xl font-bold mt-4"), g.Text(fmt.Sprintf("$%.2f", adObj.Price))),
		locationNode,
	)
}

// Add a bookmark icon SVG component
func BookmarkIcon(bookmarked bool) g.Node {
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

// AdGridWrapper wraps ad content in a grid item wrapper for grid view, applying col-span-full if expanded
func AdGridWrapper(ad ad.Ad, content g.Node, expanded bool) g.Node {
	className := "grid-item-wrapper"
	if expanded {
		className += " col-span-full"
	}
	return Div(
		ID(fmt.Sprintf("ad-grid-wrap-%d", ad.ID)),
		Class(className),
		content,
	)
}

// AdCardExpandable renders an ad card with a clickable area for details and a bookmark button
func AdCardExpandable(ad ad.Ad, loc *time.Location, currentUser *user.User, view ...string) g.Node {
	isGrid := len(view) > 0 && view[0] == "grid"
	htmxTarget := fmt.Sprintf("#ad-%d", ad.ID)
	if isGrid {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", ad.ID)
	}
	posted := ad.CreatedAt.In(loc)
	agoStr := formatAdAge(posted)
	sortedYears := append([]string{}, ad.Years...)
	sort.Strings(sortedYears)
	sortedModels := append([]string{}, ad.Models...)
	sort.Strings(sortedModels)
	sortedEngines := append([]string{}, ad.Engines...)
	sort.Strings(sortedEngines)
	bookmarkBtn := g.Node(nil)
	if currentUser != nil {
		if ad.Bookmarked {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(true),
			)
		} else {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(false),
			)
		}
	}
	// In AdCardExpandable, show the first image
	firstIdx := 1
	if len(ad.ImageOrder) > 0 {
		firstIdx = ad.ImageOrder[0]
	}
	imageNode := Div(
		Class("relative w-full h-48 bg-gray-100 overflow-hidden"),
		AdImageWithFallbackSrcSet(ad.ID, firstIdx, ad.Title, "grid"),
		Div(
			Class("absolute top-0 left-0 bg-white text-green-600 text-base font-normal px-2 rounded-br-md"),
			g.Text(fmt.Sprintf("$%.0f", ad.Price)),
		),
	)
	if isGrid {
		// Minimal grid card: image, price badge, title/bookmark, age/location
		locationStr, flagNode := getDisplayLocationAndFlag(ad)
		bookmarkBtnGrid := g.Node(nil)
		if currentUser != nil {
			if ad.Bookmarked {
				bookmarkBtnGrid = Button(
					Type("button"),
					Class("focus:outline-none mr-1 align-middle"),
					hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
					hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
					hx.Swap("outerHTML"),
					ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
					g.Attr("onclick", "event.stopPropagation()"),
					BookmarkIcon(true),
				)
			} else {
				bookmarkBtnGrid = Button(
					Type("button"),
					Class("focus:outline-none mr-1 align-middle"),
					hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
					hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
					hx.Swap("outerHTML"),
					ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
					g.Attr("onclick", "event.stopPropagation()"),
					BookmarkIcon(false),
				)
			}
		}
		return Div(
			ID(fmt.Sprintf("ad-grid-wrap-%d", ad.ID)),
			Class("border rounded-lg shadow-sm bg-white flex flex-col cursor-pointer hover:shadow-md transition-shadow"),
			hx.Get(fmt.Sprintf("/ad/detail/%d?view=grid", ad.ID)),
			hx.Target(fmt.Sprintf("#ad-grid-wrap-%d", ad.ID)),
			hx.Swap("outerHTML"),
			Div(
				Class("rounded-t-lg overflow-hidden"),
				imageNode,
			),
			Div(
				Class("p-2 flex flex-col gap-1"),
				// Title and bookmark row
				Div(
					Class("flex flex-row items-center justify-between"),
					Div(Class("font-semibold text-base truncate"), g.Text(ad.Title)),
					bookmarkBtnGrid,
				),
				// Age and location row
				Div(
					Class("flex flex-row items-center justify-between text-xs text-gray-500"),
					Div(Class("text-gray-400"), g.Text(agoStr)),
					Div(Class("flex flex-row items-center gap-1"),
						flagNode,
						g.If(locationStr != "" || flagNode != nil, Div(Class("text-xs text-gray-500"), g.Text(locationStr+""))),
					),
				),
			),
		)
	}
	card := Div(
		ID(fmt.Sprintf("ad-%d", ad.ID)),
		Class("block border p-4 mb-4 rounded hover:bg-gray-50 relative cursor-pointer group bg-white"),
		hx.Get(fmt.Sprintf("/ad/detail/%d?view=%s", ad.ID, func() string {
			if len(view) > 0 {
				return view[0]
			}
			return "list"
		}())),
		hx.Target(htmxTarget),
		hx.Swap("outerHTML"),
		Div(
			Class("flex items-center justify-between relative z-10"),
			H3(Class("text-xl font-bold"), g.Text(ad.Title)),
			Div(
				Class("flex flex-row items-center gap-2"),
				imageNode,
				bookmarkBtn,
			),
		),
		// Rock section - will be populated by HTMX
		Div(
			ID(fmt.Sprintf("rock-section-%d", ad.ID)),
			Class("mt-2"),
			hx.Get(fmt.Sprintf("/api/ad-rocks/%d", ad.ID)),
			hx.Trigger("load"),
		),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", sortedYears))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", sortedModels))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Engines: %v", sortedEngines))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Clicks: %d", ad.ClickCount))),
		P(Class("mt-2"), g.Text(ad.Description)),
		P(Class("text-xl font-bold mt-2"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
		P(
			Class("text-xs text-gray-400 mt-4"),
			g.Text(fmt.Sprintf("ID: %d • Posted: %s", ad.ID, posted.Format("Jan 2, 2006 3:04:05 PM MST"))),
		),
	)
	if isGrid {
		return AdGridWrapper(ad, card, false)
	}
	return card
}

// AdEditPartial renders the ad edit form for inline editing
func AdEditPartial(adObj ad.Ad, makes, years []string, modelAvailability, engineAvailability map[string]bool, categories, subcategories []string, cancelTarget, htmxTarget string, view ...string) g.Node {
	isGrid := len(view) > 0 && view[0] == "grid"
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
			ValidationErrorContainer(),
			FormGroup("Title", "title",
				Input(
					Type("text"),
					ID("title"),
					Name("title"),
					Class("w-full p-2 border rounded"),
					Required(),
					Value(adObj.Title),
				),
			),
			FormGroup("Make", "make",
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
			FormGroup("Years", "years", Div(ID("yearsDiv"), GridContainer(5, func() []g.Node {
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
			FormGroup("Models", "models", Div(ID("modelsDiv"), GridContainer(5, func() []g.Node {
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
			FormGroup("Engines", "engines", Div(ID("enginesDiv"), GridContainer(5, func() []g.Node {
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
			CategoriesFormGroup(categories, adObj.Category),
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
			FormGroup("Images", "images",
				Div(
					// New: Image gallery for existing images
					Div(
						ID("image-gallery"),
						Class("flex flex-row gap-2 mb-2"),
						g.Group(func() []g.Node {
							imageNodes := []g.Node{}
							for i, idx := range adObj.ImageOrder {
								imageNodes = append(imageNodes,
									Div(
										Class("relative group"),
										g.Attr("data-image-idx", fmt.Sprintf("%d", idx)),
										AdImageWithFallbackSrcSet(adObj.ID, idx, fmt.Sprintf("Image %d", i+1), "grid"),
										Button(
											Type("button"),
											Class("absolute top-0 right-0 bg-white bg-opacity-80 rounded-full p-1 text-red-600 hover:text-red-800 z-10 delete-image-btn"),
											g.Attr("onclick", fmt.Sprintf("deleteImage(this, %d)", idx)),
											Img(Src("/images/trashcan.svg"), Alt("Delete"), Class("w-4 h-4")),
										),
									),
								)
							}
							return imageNodes
						}()),
					),
					// Hidden input for image order (comma-separated indices)
					Input(
						Type("hidden"),
						ID("image_order"),
						Name("image_order"),
						Value(func() string {
							order := ""
							for i, idx := range adObj.ImageOrder {
								if i > 0 {
									order += ","
								}
								order += fmt.Sprintf("%d", idx)
							}
							return order
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
			FormGroup("Description", "description",
				Textarea(
					ID("description"),
					Name("description"),
					Class("w-full p-2 border rounded"),
					Rows("4"),
					g.Text(adObj.Description),
				),
			),
			FormGroup("Price", "price",
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
			FormGroup("Location", "location",
				Input(
					Type("text"),
					ID("location"),
					Name("location"),
					Class("w-full p-2 border rounded"),
					Placeholder("(Optional)"),
					Value(getRawLocation(adObj)),
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
				StyledButton("Save", ButtonPrimary,
					Type("submit"),
				),
			),
		),
	)
	if isGrid {
		return AdGridWrapper(adObj, editForm, true)
	}
	return editForm
}

// AdDetailPartial renders the ad detail view (with collapse button)
func AdDetailPartial(ad ad.Ad, userID int, view ...string) g.Node {
	viewType := "list"
	if len(view) > 0 {
		viewType = view[0]
	}
	return AdDetailUnified(ad, userID, viewType)
}

// Update AdCardWithBookmark to use AdCardExpandable for in-place expand/collapse
func AdCardWithBookmark(ad ad.Ad, loc *time.Location, currentUser *user.User) g.Node {
	return AdCardExpandable(ad, loc, currentUser)
}

func AdListContainer(children ...g.Node) g.Node {
	return Div(
		ID("adsList"),
		Class("space-y-4"),
		g.Group(children),
	)
}

// AdCompactListContainer provides a container for the compact list view
func AdCompactListContainer(children ...g.Node) g.Node {
	return Div(
		ID("adsList"),
		Class("bg-white"),
		g.Group(children),
	)
}

func BuildAdListNodes(ads map[int]ad.Ad, loc *time.Location) []g.Node {
	// Convert map to slice
	adSlice := make([]ad.Ad, 0, len(ads))
	for _, ad := range ads {
		adSlice = append(adSlice, ad)
	}
	// Sort by CreatedAt DESC, ID DESC
	sort.Slice(adSlice, func(i, j int) bool {
		if adSlice[i].CreatedAt.Equal(adSlice[j].CreatedAt) {
			return adSlice[i].ID > adSlice[j].ID
		}
		return adSlice[i].CreatedAt.After(adSlice[j].CreatedAt)
	})
	// Build nodes
	adsList := []g.Node{}
	for _, ad := range adSlice {
		adsList = append(adsList, AdCard(ad, loc))
	}
	return adsList
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
			PageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				hx.Post("/api/new-ad"),
				hx.Encoding("multipart/form-data"),
				hx.Target("#result"),
				ValidationErrorContainer(),
				FormGroup("Title", "title",
					Input(
						Type("text"),
						ID("title"),
						Name("title"),
						Class("w-full p-2 border rounded"),
						Required(),
					),
				),
				FormGroup("Make", "make",
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
				FormGroup("Images", "images",
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
				FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
					),
				),
				FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
						Min("0"),
					),
				),
				FormGroup("Location", "location",
					Input(
						Type("text"),
						ID("location"),
						Name("location"),
						Class("w-full p-2 border rounded"),
						Placeholder("(Optional)"),
					),
				),
				StyledButton("Submit", ButtonPrimary,
					Type("submit"),
				),
				g.Raw(`<script src="/js/image-preview.js" defer></script>`),
			),
		},
	)
}

func ViewAdPage(currentUser *user.User, path string, adObj ad.Ad) g.Node {
	bookmarkBtn := g.Node(nil)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	// Action buttons: bookmark, delete (edit removed)
	actionButtons := []g.Node{bookmarkBtn}
	isArchivedAd := adObj.IsArchived()
	if userID > 0 {
		if adObj.Bookmarked {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", adObj.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", adObj.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", adObj.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(true),
			)
		} else {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", adObj.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", adObj.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", adObj.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(false),
			)
		}
		actionButtons[0] = bookmarkBtn
		if currentUser.ID == adObj.UserID && !isArchivedAd {
			deleteButton := DeleteButton(adObj.ID)
			actionButtons = append(actionButtons, deleteButton)
		}
	}
	statusIndicator := g.Node(nil)
	if isArchivedAd {
		statusIndicator = Div(
			Class("bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded mb-4"),
			g.Text("ARCHIVED - This ad has been archived"),
		)
	}
	return Page(
		fmt.Sprintf("Ad %d - Parts Pile", adObj.ID),
		currentUser,
		path,
		[]g.Node{
			Div(
				Class("flex items-center justify-between mb-4"),
				H2(Class("text-2xl font-bold"), g.Text(adObj.Title)),
				Div(
					append([]g.Node{Class("flex flex-row items-center gap-2")}, actionButtons...)...,
				),
			),
			statusIndicator,
			AdDetails(adObj),
			// Rock section - will be populated by HTMX
			Div(
				ID(fmt.Sprintf("rock-section-%d", adObj.ID)),
				Class("mt-4"),
				hx.Get(fmt.Sprintf("/api/ad-rocks/%d", adObj.ID)),
				hx.Trigger("load"),
			),
			ActionButtons(
				BackToListingsButton(),
			),
			g.Raw(`<script src="/js/image-preview.js" defer></script>`),
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
			PageHeader("Edit Ad"),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				hx.Post(fmt.Sprintf("/api/update-ad/%d", currentAd.ID)),
				hx.Encoding("multipart/form-data"),
				hx.Target(htmxTarget),
				hx.Swap("outerHTML"),
				ValidationErrorContainer(),
				FormGroup("Title", "title",
					Input(
						Type("text"),
						ID("title"),
						Name("title"),
						Class("w-full p-2 border rounded"),
						Required(),
					),
				),
				FormGroup("Make", "make",
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
				FormGroup("Years", "years", Div(ID("yearsDiv"), GridContainer(5, yearCheckboxes...))),
				FormGroup("Models", "models", Div(ID("modelsDiv"), GridContainer(5, modelCheckboxes...))),
				FormGroup("Engines", "engines", Div(ID("enginesDiv"), GridContainer(5, engineCheckboxes...))),
				CategoriesFormGroup(categories, currentAd.Category),
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
				FormGroup("Images", "images",
					Div(
						// New: Image gallery for existing images
						Div(
							ID("image-gallery"),
							Class("flex flex-row gap-2 mb-2"),
							g.Group(func() []g.Node {
								imageNodes := []g.Node{}
								imageURLs := AdImageURLs(currentAd.ID, currentAd.ImageOrder)
								for i, url := range imageURLs {
									imageNodes = append(imageNodes,
										Div(
											Class("relative group"),
											g.Attr("data-image-idx", fmt.Sprintf("%d", currentAd.ImageOrder[i])),
											Img(
												Src(url),
												Alt(fmt.Sprintf("Image %d", i+1)),
												Class("object-cover w-24 h-24 rounded border cursor-move"),
												g.Attr("draggable", "true"),
											),
											Button(
												Type("button"),
												Class("absolute top-0 right-0 bg-white bg-opacity-80 rounded-full p-1 text-red-600 hover:text-red-800 z-10 delete-image-btn"),
												g.Attr("onclick", fmt.Sprintf("deleteImage(this, %d)", currentAd.ImageOrder[i])),
												Img(Src("/images/trashcan.svg"), Alt("Delete"), Class("w-4 h-4")),
											),
										),
									)
								}
								return imageNodes
							}()),
						),
						// Hidden input for image order (comma-separated indices)
						Input(
							Type("hidden"),
							ID("image_order"),
							Name("image_order"),
							Value(func() string {
								order := ""
								for i, idx := range currentAd.ImageOrder {
									if i > 0 {
										order += ","
									}
									order += fmt.Sprintf("%d", idx)
								}
								return order
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
				FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
					),
				),
				FormGroup("Price", "price",
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
				FormGroup("Location (Zipcode)", "location",
					Input(
						Type("text"),
						ID("location"),
						Name("location"),
						Class("w-full p-2 border rounded"),
						Placeholder("Optional zipcode, e.g. 90210"),
						Value(getRawLocation(currentAd)),
					),
				),
				StyledButton("Submit", ButtonPrimary,
					Type("submit"),
				),
				g.Raw(`<script src="/js/image-preview.js" defer></script>`),
				g.Raw(`<script src="/js/image-edit.js" defer></script>`),
			),
		},
	)
}

// For compatibility, keep the original AdCard
func AdCard(ad ad.Ad, loc *time.Location) g.Node {
	return AdCardWithBookmark(ad, loc, nil)
}

// BookmarkButton returns the bookmark toggle button for HTMX swaps
func BookmarkButton(ad ad.Ad) g.Node {
	if ad.Bookmarked {
		return Button(
			Type("button"),
			Class("ml-2 focus:outline-none z-20"),
			hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
			hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
			hx.Swap("outerHTML"),
			ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
			g.Attr("onclick", "event.stopPropagation()"),
			BookmarkIcon(true),
		)
	}
	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none z-20"),
		hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
		hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
		hx.Swap("outerHTML"),
		ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
		g.Attr("onclick", "event.stopPropagation()"),
		BookmarkIcon(false),
	)
}

// Helper to generate signed B2 image URLs for an ad
func AdImageURLs(adID int, order []int) []string {
	urls := []string{}

	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	if err != nil || token == "" {
		// Return empty strings when B2 images aren't available - browser will show broken images
		for range order {
			urls = append(urls, "")
		}
		return urls
	}

	for _, idx := range order {
		// Use 160w size for gallery thumbnails
		urls = append(urls, fmt.Sprintf(
			"https://f004.backblazeb2.com/file/parts-pile/%d/%d-160w.webp?Authorization=%s",
			adID, idx, token,
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
	case "grid":
		src = fmt.Sprintf("%s-480w.webp?Authorization=%s", base, token)
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
	case "grid":
		sizes = "(max-width: 640px) 160px, (max-width: 768px) 200px, (max-width: 1024px) 240px, 300px"
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

// AdCardTreeView renders an ad card in tree view format (same as grid expanded but without close button)
func AdCardTreeView(ad ad.Ad, loc *time.Location, userID int) g.Node {
	// Use the unified function with "tree" view (no close button)
	return AdDetailUnified(ad, userID, "tree")
}

// AdCardCompactTree renders a compact single-line ad card for tree view (collapsed state)
func AdCardCompactTree(ad ad.Ad, loc *time.Location, currentUser *user.User) g.Node {
	posted := ad.CreatedAt.In(loc)

	// Bookmark button
	bookmarkBtn := g.Node(nil)
	if currentUser != nil {
		if ad.Bookmarked {
			bookmarkBtn = Button(
				Type("button"),
				Class("focus:outline-none mr-2"),
				hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(true),
			)
		} else {
			bookmarkBtn = Button(
				Type("button"),
				Class("focus:outline-none mr-2"),
				hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(false),
			)
		}
	}

	// Check if ad has images
	hasImages := len(ad.ImageOrder) > 0
	picLink := g.Node(nil)
	if hasImages {
		picLink = Span(
			Class("text-orange-500 hover:text-orange-700 cursor-pointer"),
			g.Text("pic"),
		)
	}

	return Div(
		ID(fmt.Sprintf("ad-tree-%d", ad.ID)),
		Class("flex items-center py-2 px-3 border-b border-gray-200 hover:bg-gray-50 cursor-pointer last:border-b-0"),
		hx.Get(fmt.Sprintf("/ad/expand-tree/%d", ad.ID)),
		hx.Target(fmt.Sprintf("#ad-tree-%d", ad.ID)),
		hx.Swap("outerHTML"),
		// Bookmark icon
		bookmarkBtn,
		// Description (blue text)
		Div(
			Class("flex-1 text-blue-600 hover:text-blue-800"),
			g.Text(ad.Title),
		),
		// Location with flag (using new helper function)
		Div(
			Class("mr-4"),
			LocationDisplayWithFlag(ad),
		),
		// Time posted (using new helper function)
		Div(
			Class("mr-4"),
			AgeDisplay(posted),
		),
		// Price (green text)
		Div(
			Class("text-green-600 font-semibold mr-4"),
			g.Text(fmt.Sprintf("$%.0f", ad.Price)),
		),
		// Pic link (orange text)
		picLink,
	)
}

// AdCardExpandedTree renders an expanded ad card for tree view (with close button)
func AdCardExpandedTree(ad ad.Ad, loc *time.Location, currentUser *user.User) g.Node {
	// Create the expanded content similar to AdDetailUnified but without close button
	htmxTarget := fmt.Sprintf("#ad-tree-%d", ad.ID)

	bookmarkBtn := g.Node(nil)
	if currentUser != nil {
		if ad.Bookmarked {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(true),
			)
		} else {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(false),
			)
		}
	}

	deleteBtn := g.Node(nil)
	editBtn := g.Node(nil)
	if currentUser != nil && currentUser.ID == ad.UserID {
		deleteBtn = Button(
			Type("button"),
			Class("ml-2 focus:outline-none z-20"),
			hx.Delete(fmt.Sprintf("/delete-ad/%d", ad.ID)),
			hx.Target(htmxTarget),
			hx.Swap("delete"),
			hx.Confirm("Are you sure you want to delete this ad? This action cannot be undone."),
			Img(
				Src("/images/trashcan.svg"),
				Alt("Delete"),
				Class("w-6 h-6 inline align-middle text-red-500 hover:text-red-700"),
			),
		)
		editBtn = Button(
			Type("button"),
			Class("ml-2 focus:outline-none z-20"),
			hx.Get(fmt.Sprintf("/ad/edit-partial/%d?view=tree", ad.ID)),
			hx.Target(htmxTarget),
			hx.Swap("outerHTML"),
			Img(
				Src("/images/edit.svg"),
				Alt("Edit"),
				Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
			),
		)
	}

	// Use tile view layout for expanded tree view
	firstIdx := 1
	if len(ad.ImageOrder) > 0 {
		firstIdx = ad.ImageOrder[0]
	}
	agoStr := formatAdAge(ad.CreatedAt)
	locationStr, flagNode := getDisplayLocationAndFlag(ad)

	// Carousel main image area
	mainImageArea := Div(
		Class("relative w-full aspect-square bg-gray-100 overflow-hidden rounded-t-lg flex items-center justify-center"),
		Div(
			ID(fmt.Sprintf("ad-carousel-img-%d", ad.ID)),
			AdImageWithFallbackSrcSet(ad.ID, firstIdx, ad.Title, "carousel"),
			Div(
				Class("absolute top-0 left-0 bg-white text-green-600 text-base font-normal px-2 rounded-br-md"),
				g.Text(fmt.Sprintf("$%.0f", ad.Price)),
			),
		),
	)

	// Carousel thumbnails
	thumbnails := Div(
		Class("flex flex-row gap-2 mt-2 px-4 justify-center"),
		g.Group(func() []g.Node {
			nodes := []g.Node{}
			for i, idx := range ad.ImageOrder {
				nodes = append(nodes, Button(
					Type("button"),
					Class("border rounded w-16 h-16 overflow-hidden p-0 focus:outline-none"),
					g.Attr("hx-get", fmt.Sprintf("/ad/image/%d/%d", ad.ID, idx)),
					g.Attr("hx-target", fmt.Sprintf("#ad-carousel-img-%d", ad.ID)),
					g.Attr("hx-swap", "innerHTML"),
					AdImageWithFallbackSrcSet(ad.ID, idx, fmt.Sprintf("Image %d", i+1), "thumbnail"),
				))
			}
			return nodes
		}()),
	)

	// Add close button to collapse back to compact view
	closeBtn := Button(
		Type("button"),
		Class("absolute -top-2 -right-2 bg-gray-800 bg-opacity-80 text-white text-2xl font-bold rounded-full w-10 h-10 flex items-center justify-center shadow-lg z-30 hover:bg-gray-700 focus:outline-none"),
		hx.Get(fmt.Sprintf("/ad/collapse-tree/%d", ad.ID)),
		hx.Target(fmt.Sprintf("#ad-tree-%d", ad.ID)),
		hx.Swap("outerHTML"),
		g.Text("×"),
	)

	content := Div(
		Class("border rounded-lg shadow-lg bg-white flex flex-col relative"),
		closeBtn,
		mainImageArea,
		thumbnails,
		Div(
			Class("p-4 flex flex-col gap-2"),
			// Title and buttons row
			Div(
				Class("flex flex-row items-center justify-between mb-2"),
				Div(Class("font-semibold text-xl truncate"), g.Text(ad.Title)),
				Div(Class("flex flex-row items-center gap-2 ml-2"),
					bookmarkBtn,
					MessageButton(ad.ID, ad.UserID, currentUser.ID, "tree"),
					editBtn,
					deleteBtn,
				),
			),
			// Category display
			formatCategoryDisplay(ad),
			// Age and location row
			Div(
				Class("flex flex-row items-center justify-between text-xs text-gray-500 mb-2"),
				Div(Class("text-gray-400"), g.Text(agoStr)),
				Div(Class("flex flex-row items-center gap-1"),
					flagNode,
					g.If(locationStr != "" || flagNode != nil, Div(Class("text-xs text-gray-500"), g.Text(locationStr+""))),
				),
			),
			// Description
			Div(Class("text-base mt-2"), g.Text(ad.Description)),
			// Rock section - will be populated by HTMX
			Div(
				ID(fmt.Sprintf("rock-section-%d", ad.ID)),
				Class("mt-2"),
				hx.Get(fmt.Sprintf("/api/ad-rocks/%d", ad.ID)),
				hx.Trigger("load"),
			),
		),
	)

	return Div(
		ID(fmt.Sprintf("ad-tree-%d", ad.ID)),
		Class("relative my-4 mx-2"),
		content,
	)
}

// AdDetailUnified renders the ad detail view for all three views (expanded grid, list, tree)
// Uses the expanded grid view layout as reference, with close button only for grid view
func AdDetailUnified(ad ad.Ad, userID int, view string) g.Node {
	htmxTarget := fmt.Sprintf("#ad-%d", ad.ID)
	if view == "grid" {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", ad.ID)
	}
	// For tree and list views, use the same ID as the content (#ad-{id})

	bookmarkBtn := g.Node(nil)
	if userID > 0 {
		if ad.Bookmarked {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(true),
			)
		} else {
			bookmarkBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none z-20"),
				hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				BookmarkIcon(false),
			)
		}
	}

	deleteBtn := g.Node(nil)
	editBtn := g.Node(nil)
	if userID == ad.UserID {
		deleteBtn = Button(
			Type("button"),
			Class("ml-2 focus:outline-none z-20"),
			hx.Delete(fmt.Sprintf("/delete-ad/%d", ad.ID)),
			hx.Target(htmxTarget),
			hx.Swap("delete"),
			hx.Confirm("Are you sure you want to delete this ad? This action cannot be undone."),
			Img(
				Src("/images/trashcan.svg"),
				Alt("Delete"),
				Class("w-6 h-6 inline align-middle text-red-500 hover:text-red-700"),
			),
		)
		editBtn = Button(
			Type("button"),
			Class("ml-2 focus:outline-none z-20"),
			hx.Get(fmt.Sprintf("/ad/edit-partial/%d?view=%s", ad.ID, view)),
			hx.Target(htmxTarget),
			hx.Swap("outerHTML"),
			Img(
				Src("/images/edit.svg"),
				Alt("Edit"),
				Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
			),
		)
	}

	// Use tile view layout for all views (expanded grid, list, tree)
	firstIdx := 1
	if len(ad.ImageOrder) > 0 {
		firstIdx = ad.ImageOrder[0]
	}
	agoStr := formatAdAge(ad.CreatedAt)
	locationStr, flagNode := getDisplayLocationAndFlag(ad)

	// Carousel main image area (HTMX target is the child, not the container)
	mainImageArea := Div(
		Class("relative w-full aspect-square bg-gray-100 overflow-hidden rounded-t-lg flex items-center justify-center"),
		Div(
			ID(fmt.Sprintf("ad-carousel-img-%d", ad.ID)),
			AdImageWithFallbackSrcSet(ad.ID, firstIdx, ad.Title, "carousel"),
			Div(
				Class("absolute top-0 left-0 bg-white text-green-600 text-base font-normal px-2 rounded-br-md"),
				g.Text(fmt.Sprintf("$%.0f", ad.Price)),
			),
		),
	)

	// Carousel thumbnails
	thumbnails := Div(
		Class("flex flex-row gap-2 mt-2 px-4 justify-center"),
		g.Group(func() []g.Node {
			nodes := []g.Node{}
			for i, idx := range ad.ImageOrder {
				nodes = append(nodes, Button(
					Type("button"),
					Class("border rounded w-16 h-16 overflow-hidden p-0 focus:outline-none"),
					g.Attr("hx-get", fmt.Sprintf("/ad/image/%d/%d", ad.ID, idx)),
					g.Attr("hx-target", fmt.Sprintf("#ad-carousel-img-%d", ad.ID)),
					g.Attr("hx-swap", "innerHTML"),
					AdImageWithFallbackSrcSet(ad.ID, idx, fmt.Sprintf("Image %d", i+1), "thumbnail"),
				))
			}
			return nodes
		}()),
	)

	// Show close button for all views (grid, list, and tree)
	var closeBtn g.Node
	closeBtn = Button(
		Type("button"),
		Class("absolute -top-2 -right-2 bg-gray-800 bg-opacity-80 text-white text-2xl font-bold rounded-full w-10 h-10 flex items-center justify-center shadow-lg z-30 hover:bg-gray-700 focus:outline-none"),
		hx.Get(fmt.Sprintf("/ad/card/%d?view=%s", ad.ID, view)),
		hx.Target(htmxTarget),
		hx.Swap("outerHTML"),
		g.Text("×"),
	)

	// For grid view, don't add ID to content since wrapper has it
	// For list view, add ID to content so close button can target it
	var content g.Node
	if view == "grid" {
		content = Div(
			Class("border rounded-lg shadow-lg bg-white flex flex-col relative"),
			closeBtn,
			mainImageArea,
			thumbnails,
			Div(
				Class("p-4 flex flex-col gap-2"),
				// Title and buttons row
				Div(
					Class("flex flex-row items-center justify-between mb-2"),
					Div(Class("font-semibold text-xl truncate"), g.Text(ad.Title)),
					Div(Class("flex flex-row items-center gap-2 ml-2"),
						bookmarkBtn,
						MessageButton(ad.ID, ad.UserID, userID, view),
						editBtn,
						deleteBtn,
					),
				),
				// Category display
				formatCategoryDisplay(ad),
				// Age and location row
				Div(
					Class("flex flex-row items-center justify-between text-xs text-gray-500 mb-2"),
					Div(Class("text-gray-400"), g.Text(agoStr)),
					Div(Class("flex flex-row items-center gap-1"),
						flagNode,
						g.If(locationStr != "" || flagNode != nil, Div(Class("text-xs text-gray-500"), g.Text(locationStr+""))),
					),
				),
				// Description
				Div(Class("text-base mt-2"), g.Text(ad.Description)),
				// Rock section - will be populated by HTMX
				Div(
					ID(fmt.Sprintf("rock-section-%d", ad.ID)),
					Class("mt-2"),
					hx.Get(fmt.Sprintf("/api/ad-rocks/%d", ad.ID)),
					hx.Trigger("load"),
				),
			),
		)
	} else {
		content = Div(
			ID(fmt.Sprintf("ad-%d", ad.ID)),
			Class("border rounded-lg shadow-lg bg-white flex flex-col relative my-4 mx-2"),
			closeBtn,
			mainImageArea,
			thumbnails,
			Div(
				Class("p-4 flex flex-col gap-2"),
				// Title and buttons row
				Div(
					Class("flex flex-row items-center justify-between mb-2"),
					Div(Class("font-semibold text-xl truncate"), g.Text(ad.Title)),
					Div(Class("flex flex-row items-center gap-2 ml-2"),
						bookmarkBtn,
						MessageButton(ad.ID, ad.UserID, userID, view),
						editBtn,
						deleteBtn,
					),
				),
				// Category display
				formatCategoryDisplay(ad),
				// Age and location row
				Div(
					Class("flex flex-row items-center justify-between text-xs text-gray-500 mb-2"),
					Div(Class("text-gray-400"), g.Text(agoStr)),
					Div(Class("flex flex-row items-center gap-1"),
						flagNode,
						g.If(locationStr != "" || flagNode != nil, Div(Class("text-xs text-gray-500"), g.Text(locationStr+""))),
					),
				),
				// Description
				Div(Class("text-base mt-2"), g.Text(ad.Description)),
				// Rock section - will be populated by HTMX
				Div(
					ID(fmt.Sprintf("rock-section-%d", ad.ID)),
					Class("mt-2"),
					hx.Get(fmt.Sprintf("/api/ad-rocks/%d", ad.ID)),
					hx.Trigger("load"),
				),
			),
		)
	}

	// Only wrap in AdGridWrapper for grid view
	if view == "grid" {
		return AdGridWrapper(ad, content, true)
	}
	return content
}

// AdCardCompactList renders a compact single-line ad card for list view
func AdCardCompactList(ad ad.Ad, loc *time.Location, currentUser *user.User) g.Node {
	posted := ad.CreatedAt.In(loc)

	// Bookmark button
	bookmarkBtn := g.Node(nil)
	if currentUser != nil {
		if ad.Bookmarked {
			bookmarkBtn = Button(
				Type("button"),
				Class("focus:outline-none mr-2"),
				hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation(); event.preventDefault();"),
				BookmarkIcon(true),
			)
		} else {
			bookmarkBtn = Button(
				Type("button"),
				Class("focus:outline-none mr-2"),
				hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#bookmark-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("bookmark-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation(); event.preventDefault();"),
				BookmarkIcon(false),
			)
		}
	}

	// Check if ad has images
	hasImages := len(ad.ImageOrder) > 0
	picLink := g.Node(nil)
	if hasImages {
		picLink = Span(
			Class("text-orange-500 hover:text-orange-700 cursor-pointer"),
			g.Text("pic"),
		)
	}

	return Div(
		ID(fmt.Sprintf("ad-%d", ad.ID)),
		Class("flex items-center py-2 px-3 hover:bg-gray-50 cursor-pointer"),
		hx.Get(fmt.Sprintf("/ad/detail/%d?view=list", ad.ID)),
		hx.Target(fmt.Sprintf("#ad-%d", ad.ID)),
		hx.Swap("outerHTML"),
		// Bookmark icon
		bookmarkBtn,
		// Description (blue text)
		Div(
			Class("flex-1 text-blue-600 hover:text-blue-800"),
			g.Text(ad.Title),
		),
		// Location with flag (using new helper function)
		Div(
			Class("mr-4"),
			LocationDisplayWithFlag(ad),
		),
		// Time posted (using new helper function)
		Div(
			Class("mr-4"),
			AgeDisplay(posted),
		),
		// Price (green text)
		Div(
			Class("text-green-600 font-semibold mr-4"),
			g.Text(fmt.Sprintf("$%.0f", ad.Price)),
		),
		// Pic link (orange text)
		picLink,
	)
}

// MessageButton renders a message button for an ad
func MessageButton(adID, adUserID, currentUserID int, view ...string) g.Node {
	// Don't show message button if user is viewing their own ad
	if currentUserID == adUserID {
		return g.Node(nil)
	}

	// Don't show message button if user is not logged in
	if currentUserID == 0 {
		return g.Node(nil)
	}

	// Default to tree view if not specified
	viewType := "tree"
	if len(view) > 0 {
		viewType = view[0]
	}

	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none z-20"),
		Title("Message seller"),
		hx.Get(fmt.Sprintf("/messages/inline/%d?view=%s", adID, viewType)),
		hx.Target(fmt.Sprintf("#ad-%d", adID)),
		hx.Swap("outerHTML"),
		Img(
			Src("/images/message.svg"),
			Alt("Message"),
			Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
		),
	)
}
