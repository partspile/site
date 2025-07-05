package ui

import (
	"fmt"
	"sort"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/user"
)

// ---- Ad Components ----

func AdDetails(ad ad.Ad) g.Node {
	sortedYears := append([]string{}, ad.Years...)
	sortedModels := append([]string{}, ad.Models...)
	sortedEngines := append([]string{}, ad.Engines...)
	sort.Strings(sortedYears)
	sort.Strings(sortedModels)
	sort.Strings(sortedEngines)

	var locationNode g.Node = nil
	if ad.Location != nil && *ad.Location != "" {
		locationNode = P(Class("text-gray-600"), g.Text(fmt.Sprintf("Location: %s", *ad.Location)))
	}

	return Div(
		Class("mb-4"),
		P(Class("mt-4"), g.Text(ad.Description)),
		P(Class("text-2xl font-bold mt-4"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
		locationNode,
	)
}

// Add a bookmark icon SVG component
func BookmarkIcon(bookmarked bool) g.Node {
	icon := "/bookmark-false.svg"
	if bookmarked {
		icon = "/bookmark-true.svg"
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
func AdCardExpandable(ad ad.Ad, loc *time.Location, bookmarked bool, userID int, view ...string) g.Node {
	isGrid := len(view) > 0 && view[0] == "grid"
	htmxTarget := fmt.Sprintf("#ad-%d", ad.ID)
	if isGrid {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", ad.ID)
	}
	sortedYears := append([]string{}, ad.Years...)
	sort.Strings(sortedYears)
	sortedModels := append([]string{}, ad.Models...)
	sort.Strings(sortedModels)
	sortedEngines := append([]string{}, ad.Engines...)
	sort.Strings(sortedEngines)
	posted := ad.CreatedAt.In(loc).Format("Jan 2, 2006 3:04:05 PM MST")
	bookmarkBtn := g.Node(nil)
	if userID > 0 {
		if bookmarked {
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
	firstImageURL := AdImageURLs(ad.ID, 1)[0]
	imageNode := AdImageWithFallback(firstImageURL, ad.Title)
	card := Div(
		ID(fmt.Sprintf("ad-%d", ad.ID)),
		Class("block border p-4 mb-4 rounded hover:bg-gray-50 relative cursor-pointer group bg-white"),
		hx.Get(fmt.Sprintf("/ad/detail/%d?view=%s", ad.ID, func() string {
			if isGrid {
				return "grid"
			} else {
				return "list"
			}
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
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", sortedYears))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", sortedModels))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Engines: %v", sortedEngines))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Clicks: %d", ad.ClickCount))),
		P(Class("mt-2"), g.Text(ad.Description)),
		P(Class("text-xl font-bold mt-2"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
		P(
			Class("text-xs text-gray-400 mt-4"),
			g.Text(fmt.Sprintf("ID: %d • Posted: %s", ad.ID, posted)),
		),
	)
	if isGrid {
		return AdGridWrapper(ad, card, false)
	}
	return card
}

// AdEditPartial renders the ad edit form for inline editing
func AdEditPartial(adObj ad.Ad, makes, years []string, modelAvailability, engineAvailability map[string]bool, cancelTarget, htmxTarget string, view ...string) g.Node {
	isGrid := len(view) > 0 && view[0] == "grid"
	editForm := Div(
		ID(fmt.Sprintf("ad-%d", adObj.ID)),
		Class("border p-4 mb-4 rounded bg-white shadow-lg relative"),
		Form(
			ID("editAdForm"),
			Class("space-y-6"),
			EncType("multipart/form-data"),
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
			FormGroup("Images", "images",
				Input(
					Type("file"),
					ID("images"),
					Name("images"),
					Class("w-full p-2 border rounded"),
					g.Attr("accept", ".webp"),
					g.Attr("multiple", ""),
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
					Value(func() string {
						if adObj.Location != nil {
							return *adObj.Location
						}
						return ""
					}()),
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
					hx.Post(fmt.Sprintf("/api/update-ad/%d", adObj.ID)),
					hx.Encoding("multipart/form-data"),
					hx.Target(htmxTarget),
					hx.Swap("outerHTML"),
				),
			),
			ResultContainer(),
		),
	)
	if isGrid {
		return AdGridWrapper(adObj, editForm, true)
	}
	return editForm
}

// AdDetailPartial renders the ad detail view (with collapse button)
func AdDetailPartial(ad ad.Ad, bookmarked bool, userID int, view ...string) g.Node {
	isGrid := len(view) > 0 && view[0] == "grid"
	htmxTarget := fmt.Sprintf("#ad-%d", ad.ID)
	if isGrid {
		htmxTarget = fmt.Sprintf("#ad-grid-wrap-%d", ad.ID)
	}
	bookmarkBtn := g.Node(nil)
	if userID > 0 {
		if bookmarked {
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
	// Delete icon button (only for ad owner)
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
				Src("/trashcan.svg"),
				Alt("Delete"),
				Class("w-6 h-6 inline align-middle text-red-500 hover:text-red-700"),
			),
		)
		editBtn = Button(
			Type("button"),
			Class("ml-2 focus:outline-none z-20"),
			hx.Get(fmt.Sprintf("/ad/edit-partial/%d?view=%s", ad.ID, func() string {
				if isGrid {
					return "grid"
				} else {
					return "list"
				}
			}())),
			hx.Target(htmxTarget),
			hx.Swap("outerHTML"),
			Img(
				Src("/edit.svg"),
				Alt("Edit"),
				Class("w-6 h-6 inline align-middle text-blue-500 hover:text-blue-700"),
			),
		)
	}
	// In AdDetailPartial, show all images as a gallery
	imageURLs := AdImageURLs(ad.ID, 10)
	gallery := Div(
		Class("flex flex-row gap-2 overflow-x-auto mt-4 mb-4"),
		g.Group(func() []g.Node {
			nodes := []g.Node{}
			for i, url := range imageURLs {
				nodes = append(nodes, AdImageWithFallback(url, fmt.Sprintf("Image %d", i+1)))
			}
			return nodes
		}()),
	)
	detail := Div(
		ID(fmt.Sprintf("ad-%d", ad.ID)),
		Class("border p-4 mb-4 rounded bg-white shadow-lg relative"),
		Div(
			Class("flex items-center justify-between relative z-10"),
			H2(Class("text-2xl font-bold"), g.Text(ad.Title)),
			Div(
				Class("flex flex-row items-center gap-2"),
				bookmarkBtn,
				deleteBtn,
				editBtn,
				Button(
					Type("button"),
					Class("ml-2 text-gray-400 hover:text-gray-700 text-2xl font-bold focus:outline-none z-20"),
					hx.Get(fmt.Sprintf("/ad/card/%d?view=%s", ad.ID, func() string {
						if isGrid {
							return "grid"
						} else {
							return "list"
						}
					}())),
					hx.Target(htmxTarget),
					hx.Swap("outerHTML"),
					g.Text("×"),
				),
			),
		),
		gallery,
		AdDetails(ad),
	)
	if isGrid {
		return AdGridWrapper(ad, detail, true)
	}
	return detail
}

// Update AdCardWithBookmark to use AdCardExpandable for in-place expand/collapse
func AdCardWithBookmark(ad ad.Ad, loc *time.Location, bookmarked bool, userID int) g.Node {
	return AdCardExpandable(ad, loc, bookmarked, userID)
}

func AdListContainer(children ...g.Node) g.Node {
	return Div(
		ID("adsList"),
		Class("space-y-4"),
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

func NewAdPage(currentUser *user.User, path string, makes []string) g.Node {
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
				EncType("multipart/form-data"),
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
				FormGroup("Images", "images",
					Input(
						Type("file"),
						ID("images"),
						Name("images"),
						Class("w-full p-2 border rounded"),
						g.Attr("accept", ".webp"),
						g.Attr("multiple", ""),
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
					hx.Post("/api/new-ad"),
					hx.Encoding("multipart/form-data"),
					hx.Target("#result"),
				),
				ResultContainer(),
			),
		},
	)
}

func ViewAdPage(currentUser *user.User, path string, adObj ad.Ad, bookmarked bool) g.Node {
	bookmarkBtn := g.Node(nil)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	// Action buttons: bookmark, delete (edit removed)
	actionButtons := []g.Node{bookmarkBtn}
	isArchivedAd := adObj.IsArchived()
	if userID > 0 {
		if bookmarked {
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
					append([]g.Node{Class("flex items-center gap-2")}, actionButtons...)...,
				),
			),
			statusIndicator,
			AdDetails(adObj),

			// Add action buttons at the bottom
			ActionButtons(
				BackToListingsButton(),
			),
		},
	)
}

func EditAdPage(currentUser *user.User, path string, currentAd ad.Ad, makes []string, years []string, modelAvailability map[string]bool, engineAvailability map[string]bool) g.Node {
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

	return Page(
		"Edit Ad - Parts Pile",
		currentUser,
		path,
		[]g.Node{
			PageHeader("Edit Ad"),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				EncType("multipart/form-data"),
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
				FormGroup("Images", "images",
					Input(
						Type("file"),
						ID("images"),
						Name("images"),
						Class("w-full p-2 border rounded"),
						g.Attr("accept", ".webp"),
						g.Attr("multiple", ""),
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
						Value(func() string {
							if currentAd.Location != nil {
								return *currentAd.Location
							}
							return ""
						}()),
					),
				),
				StyledButton("Submit", ButtonPrimary,
					Type("submit"),
					hx.Post(fmt.Sprintf("/api/update-ad/%d", currentAd.ID)),
					hx.Encoding("multipart/form-data"),
					hx.Target("#result"),
				),
				ResultContainer(),
			),
		},
	)
}

// For compatibility, keep the original AdCard
func AdCard(ad ad.Ad, loc *time.Location) g.Node {
	return AdCardWithBookmark(ad, loc, false, 0)
}

// BookmarkButton returns the bookmark toggle button for HTMX swaps
func BookmarkButton(bookmarked bool, adID int) g.Node {
	if bookmarked {
		return Button(
			Type("button"),
			Class("ml-2 focus:outline-none z-20"),
			hx.Delete(fmt.Sprintf("/api/bookmark-ad/%d", adID)),
			hx.Target(fmt.Sprintf("#bookmark-btn-%d", adID)),
			hx.Swap("outerHTML"),
			ID(fmt.Sprintf("bookmark-btn-%d", adID)),
			g.Attr("onclick", "event.stopPropagation()"),
			BookmarkIcon(true),
		)
	}
	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none z-20"),
		hx.Post(fmt.Sprintf("/api/bookmark-ad/%d", adID)),
		hx.Target(fmt.Sprintf("#bookmark-btn-%d", adID)),
		hx.Swap("outerHTML"),
		ID(fmt.Sprintf("bookmark-btn-%d", adID)),
		g.Attr("onclick", "event.stopPropagation()"),
		BookmarkIcon(false),
	)
}

// Helper to generate signed B2 image URLs for an ad
func AdImageURLs(adID int, max int) []string {
	prefix := fmt.Sprintf("%d/", adID)
	token, err := b2util.GetB2DownloadTokenForPrefixCached(prefix)
	urls := []string{}
	if err != nil || token == "" {
		for i := 1; i <= max; i++ {
			urls = append(urls, "/no-image.svg")
		}
		return urls
	}
	for i := 1; i <= max; i++ {
		urls = append(urls, fmt.Sprintf(
			"https://f004.backblazeb2.com/file/parts-pile/%d/%d.webp?Authorization=%s",
			adID, i, token,
		))
	}
	return urls
}

// Helper to render an image with fallback (no onerror)
func AdImageWithFallback(src, alt string) g.Node {
	return Img(
		Src(src),
		Alt(alt),
		Class("object-cover w-full h-48 rounded"),
	)
}
