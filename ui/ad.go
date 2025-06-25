package ui

import (
	"fmt"
	"sort"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
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
	return GridContainer(1,
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", sortedYears))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", sortedModels))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Engines: %v", sortedEngines))),
		P(Class("mt-4"), g.Text(ad.Description)),
		P(Class("text-2xl font-bold mt-4"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
	)
}

// Add a flag icon SVG component
func FlagIcon(flagged bool) g.Node {
	if flagged {
		// Heroicons filled flag, green
		return g.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="inline w-6 h-6 text-green-600" fill="currentColor" viewBox="0 0 24 24"><path d="M6.75 3.75v16.5m0-16.5h10.5a.75.75 0 0 1 .67 1.08l-2.1 4.2a.75.75 0 0 0 0 .67l2.1 4.2a.75.75 0 0 1-.67 1.08H6.75"/></svg>`)
	}
	// Heroicons outline flag, gray
	return g.Raw(`<svg xmlns="http://www.w3.org/2000/svg" class="inline w-6 h-6 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6.75 3.75v16.5m0-16.5h10.5a.75.75 0 0 1 .67 1.08l-2.1 4.2a.75.75 0 0 0 0 .67l2.1 4.2a.75.75 0 0 1-.67 1.08H6.75"/></svg>`)
}

// AdCard now takes a flagged parameter
func AdCardWithFlag(ad ad.Ad, loc *time.Location, flagged bool, userID int) g.Node {
	sortedYears := append([]string{}, ad.Years...)
	sort.Strings(sortedYears)
	sortedModels := append([]string{}, ad.Models...)
	sort.Strings(sortedModels)
	posted := ad.CreatedAt.In(loc).Format("Jan 2, 2006 3:04:05 PM MST")
	flagBtn := g.Node(nil)
	if userID > 0 {
		if flagged {
			flagBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none"),
				hx.Delete(fmt.Sprintf("/api/flag-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#flag-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("flag-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				FlagIcon(true),
			)
		} else {
			flagBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none"),
				hx.Post(fmt.Sprintf("/api/flag-ad/%d", ad.ID)),
				hx.Target(fmt.Sprintf("#flag-btn-%d", ad.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("flag-btn-%d", ad.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				FlagIcon(false),
			)
		}
	}
	return Div(
		Class("block border p-4 mb-4 rounded hover:bg-gray-50 relative"),
		A(
			Href(fmt.Sprintf("/ad/%d", ad.ID)),
			Class("absolute inset-0 z-0"),
			g.Attr("aria-label", "View ad details"),
		),
		Div(
			Class("flex items-center justify-between relative z-10"),
			H3(Class("text-xl font-bold"), g.Text(ad.Make)),
			flagBtn,
		),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", sortedYears))),
		P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", sortedModels))),
		P(Class("mt-2"), g.Text(ad.Description)),
		P(Class("text-xl font-bold mt-2"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
		P(
			Class("text-xs text-gray-400 mt-4"),
			g.Text(fmt.Sprintf("ID: %d • Posted: %s", ad.ID, posted)),
		),
	)
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

func ViewAdPage(currentUser *user.User, path string, adObj ad.Ad, flagged bool) g.Node {
	flagBtn := g.Node(nil)
	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	// Action buttons: flag, edit, delete
	actionButtons := []g.Node{flagBtn}
	isDeadAd := adObj.IsDead()
	if userID > 0 {
		if flagged {
			flagBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none"),
				hx.Delete(fmt.Sprintf("/api/flag-ad/%d", adObj.ID)),
				hx.Target(fmt.Sprintf("#flag-btn-%d", adObj.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("flag-btn-%d", adObj.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				FlagIcon(true),
			)
		} else {
			flagBtn = Button(
				Type("button"),
				Class("ml-2 focus:outline-none"),
				hx.Post(fmt.Sprintf("/api/flag-ad/%d", adObj.ID)),
				hx.Target(fmt.Sprintf("#flag-btn-%d", adObj.ID)),
				hx.Swap("outerHTML"),
				ID(fmt.Sprintf("flag-btn-%d", adObj.ID)),
				g.Attr("onclick", "event.stopPropagation()"),
				FlagIcon(false),
			)
		}
		actionButtons[0] = flagBtn
		if currentUser.ID == adObj.UserID && !isDeadAd {
			editButton := StyledLink("Edit Ad", fmt.Sprintf("/edit-ad/%d", adObj.ID), ButtonPrimary)
			deleteButton := DeleteButton(adObj.ID)
			actionButtons = append(actionButtons, editButton, deleteButton)
		}
	}
	statusIndicator := g.Node(nil)
	if isDeadAd {
		statusIndicator = Div(
			Class("bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded mb-4"),
			g.Text("DEAD - This ad has been deleted"),
		)
	}
	return Page(
		fmt.Sprintf("Ad %d - Parts Pile", adObj.ID),
		currentUser,
		path,
		[]g.Node{
			Div(
				Class("flex items-center justify-between mb-4"),
				H2(Class("text-2xl font-bold"), g.Text(adObj.Make)),
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
				FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
						g.Text(currentAd.Description),
					),
				),
				FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
						Value(fmt.Sprintf("%.2f", currentAd.Price)),
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
	return AdCardWithFlag(ad, loc, false, 0)
}

// FlagButton returns the flag toggle button for HTMX swaps
func FlagButton(flagged bool, adID int) g.Node {
	if flagged {
		return Button(
			Type("button"),
			Class("ml-2 focus:outline-none"),
			hx.Delete(fmt.Sprintf("/api/flag-ad/%d", adID)),
			hx.Target(fmt.Sprintf("#flag-btn-%d", adID)),
			hx.Swap("outerHTML"),
			ID(fmt.Sprintf("flag-btn-%d", adID)),
			g.Attr("onclick", "event.stopPropagation()"),
			FlagIcon(true),
		)
	}
	return Button(
		Type("button"),
		Class("ml-2 focus:outline-none"),
		hx.Post(fmt.Sprintf("/api/flag-ad/%d", adID)),
		hx.Target(fmt.Sprintf("#flag-btn-%d", adID)),
		hx.Swap("outerHTML"),
		ID(fmt.Sprintf("flag-btn-%d", adID)),
		g.Attr("onclick", "event.stopPropagation()"),
		FlagIcon(false),
	)
}
