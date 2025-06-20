package handlers

import (
	"fmt"
	"net/http"
	"sort"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/templates"
	"github.com/parts-pile/site/vehicle"
)

func HandleNewAd(w http.ResponseWriter, r *http.Request) {
	currentUser, err := GetCurrentUser(r)
	if err != nil || currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	makes := vehicle.GetMakes()
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		makeOptions = append(makeOptions,
			Option(Value(makeName), g.Text(makeName)),
		)
	}

	_ = templates.Page(
		"New Ad - Parts Pile",
		currentUser,
		r.URL.Path,
		[]g.Node{
			Div(
				Class("mb-4 flex items-center gap-4"),
				Span(Class("font-semibold"), g.Text(currentUser.Name)),
				Span(Class("text-green-700 font-bold"), g.Text(fmt.Sprintf("Balance: %.2f tokens", currentUser.TokenBalance))),
			),
			templates.PageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				templates.ValidationErrorContainer(),
				templates.FormGroup("Make", "make",
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
				templates.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
					),
				),
				templates.FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
					),
				),
				templates.StyledButton("Submit", templates.ButtonPrimary,
					Type("submit"),
					hx.Post("/api/new-ad"),
					hx.Target("#result"),
				),
				templates.ResultContainer(),
			),
		},
	).Render(w)
}

func HandleNewAdSubmission(w http.ResponseWriter, r *http.Request) {
	currentUser, err := GetCurrentUser(r)
	if err != nil || currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Validate make selection first
	if r.FormValue("make") == "" {
		_ = templates.ValidationError("Please select a make first").Render(w)
		return
	}

	// Validate required selections
	if len(r.Form["years"]) == 0 {
		_ = templates.ValidationError("Please select at least one year").Render(w)
		return
	}

	if len(r.Form["models"]) == 0 {
		_ = templates.ValidationError("Please select at least one model").Render(w)
		return
	}

	if len(r.Form["engines"]) == 0 {
		_ = templates.ValidationError("Please select at least one engine size").Render(w)
		return
	}

	price := 0.0
	fmt.Sscanf(r.FormValue("price"), "%f", &price)

	make := r.FormValue("make")
	years := r.Form["years"]
	models := r.Form["models"]
	engines := r.Form["engines"]
	description := r.FormValue("description")

	newAd := ad.Ad{
		ID:          ad.GetNextAdID(),
		Make:        make,
		Years:       years,
		Models:      models,
		Engines:     engines,
		Description: description,
		Price:       price,
		UserID:      currentUser.ID,
	}

	ad.AddAd(newAd)

	_ = templates.SuccessMessageWithRedirect("Ad created successfully!", "/").Render(w)
}

func HandleViewAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.PathValue("id"), "%d", &adID)

	ad, ok := ad.GetAd(adID)
	if !ok || ad.ID == 0 {
		http.NotFound(w, r)
		return
	}

	currentUser, _ := GetCurrentUser(r)
	var editButton, deleteButton g.Node
	if currentUser != nil {
		editButton = templates.StyledLink("Edit Ad", fmt.Sprintf("/edit-ad/%d", ad.ID), templates.ButtonPrimary)
		deleteButton = templates.DeleteButton(ad.ID)
	} else {
		editButton = templates.StyledLinkDisabled("Edit Ad", templates.ButtonPrimary)
		deleteButton = templates.StyledLinkDisabled("Delete Ad", templates.ButtonDanger)
	}

	_ = templates.Page(
		fmt.Sprintf("Ad %d - Parts Pile", ad.ID),
		currentUser,
		r.URL.Path,
		[]g.Node{
			Div(
				Class("max-w-2xl mx-auto"),
				templates.PageHeader(ad.Make),
				templates.AdDetails(ad),
				templates.ActionButtons(
					templates.BackToListingsButton(),
					editButton,
					deleteButton,
				),
				Div(
					ID("result"),
					Class("mt-4"),
				),
			),
		},
	).Render(w)
}

func HandleEditAd(w http.ResponseWriter, r *http.Request) {
	currentUser, err := GetCurrentUser(r)
	if err != nil || currentUser == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	var adID int
	fmt.Sscanf(r.PathValue("id"), "%d", &adID)

	ad, ok := ad.GetAd(adID)
	if !ok || ad.ID == 0 {
		http.NotFound(w, r)
		return
	}

	// Prepare make options
	makes := vehicle.GetMakes()
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		attrs := []g.Node{Value(makeName)}
		if makeName == ad.Make {
			attrs = append(attrs, Selected())
		}
		attrs = append(attrs, g.Text(makeName))
		makeOptions = append(makeOptions, Option(attrs...))
	}

	// Prepare year checkboxes
	years := vehicle.GetYears(ad.Make)
	yearCheckboxes := []g.Node{}
	for _, year := range years {
		isChecked := false
		for _, adYear := range ad.Years {
			if year == adYear {
				isChecked = true
				break
			}
		}
		yearCheckboxes = append(yearCheckboxes,
			templates.Checkbox("years", year, year, isChecked, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
			),
		)
	}

	// Prepare model checkboxes
	modelAvailability := vehicle.GetModelsWithAvailability(ad.Make, ad.Years)
	modelCheckboxes := []g.Node{}
	models := make([]string, 0, len(modelAvailability))
	for model := range modelAvailability {
		models = append(models, model)
	}
	sort.Strings(models)

	for _, model := range models {
		isAvailable := modelAvailability[model]
		isChecked := false
		for _, adModel := range ad.Models {
			if model == adModel {
				isChecked = true
				break
			}
		}
		modelCheckboxes = append(modelCheckboxes,
			templates.Checkbox("models", model, model, isChecked, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}

	// Prepare engine checkboxes
	engineAvailability := vehicle.GetEnginesWithAvailability(ad.Make, ad.Years, ad.Models)
	engineCheckboxes := []g.Node{}
	engines := make([]string, 0, len(engineAvailability))
	for engine := range engineAvailability {
		engines = append(engines, engine)
	}
	sort.Strings(engines)

	for _, engine := range engines {
		isAvailable := engineAvailability[engine]
		isChecked := false
		for _, adEngine := range ad.Engines {
			if engine == adEngine {
				isChecked = true
				break
			}
		}
		engineCheckboxes = append(engineCheckboxes,
			templates.Checkbox("engines", engine, engine, isChecked, !isAvailable),
		)
	}

	_ = templates.Page(
		"Edit Ad - Parts Pile",
		currentUser,
		r.URL.Path,
		[]g.Node{
			templates.PageHeader("Edit Ad"),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				templates.ValidationErrorContainer(),
				templates.FormGroup("Make", "make",
					Select(
						ID("make"),
						Name("make"),
						Class("w-full p-2 border rounded"),
						hx.Trigger("change"),
						hx.Get("/api/years"),
						hx.Target("#yearsDiv"),
						hx.Include("this"),
						g.Group(makeOptions),
					),
				),
				Div(
					ID("yearsDiv"),
					Class("space-y-4"),
					templates.SectionHeader("Years", ""),
					templates.GridContainer(4, yearCheckboxes...),
				),
				Div(
					ID("modelsDiv"),
					Class("space-y-4"),
					templates.SectionHeader("Models", "Grayed out models are not available for all selected years"),
					templates.GridContainer(2, modelCheckboxes...),
				),
				Div(
					ID("enginesDiv"),
					Class("space-y-4"),
					templates.SectionHeader("Engines", "Grayed out engines are not available for all selected year-model combinations"),
					templates.GridContainer(2, engineCheckboxes...),
				),
				templates.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
						g.Text(ad.Description),
					),
				),
				templates.FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
						Value(fmt.Sprintf("%.2f", ad.Price)),
					),
				),
				Input(
					Type("hidden"),
					Name("id"),
					Value(fmt.Sprintf("%d", ad.ID)),
				),
				templates.StyledButton("Update", templates.ButtonPrimary,
					Type("submit"),
					hx.Post("/api/update-ad"),
					hx.Target("#result"),
				),
				Div(
					ID("result"),
					Class("mt-4"),
				),
			),
		},
	).Render(w)
}

func HandleUpdateAdSubmission(w http.ResponseWriter, r *http.Request) {
	currentUser, err := GetCurrentUser(r)
	if err != nil || currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Validate make selection first
	if r.FormValue("make") == "" {
		_ = templates.ValidationError("Please select a make first").Render(w)
		return
	}

	// Validate required selections
	if len(r.Form["years"]) == 0 {
		_ = templates.ValidationError("Please select at least one year").Render(w)
		return
	}

	if len(r.Form["models"]) == 0 {
		_ = templates.ValidationError("Please select at least one model").Render(w)
		return
	}

	if len(r.Form["engines"]) == 0 {
		_ = templates.ValidationError("Please select at least one engine size").Render(w)
		return
	}

	var adID int
	fmt.Sscanf(r.FormValue("id"), "%d", &adID)

	price := 0.0
	fmt.Sscanf(r.FormValue("price"), "%f", &price)

	make := r.FormValue("make")
	years := r.Form["years"]
	models := r.Form["models"]
	engines := r.Form["engines"]
	description := r.FormValue("description")

	updatedAd := ad.Ad{
		ID:          adID,
		Make:        make,
		Years:       years,
		Models:      models,
		Engines:     engines,
		Description: description,
		Price:       price,
	}

	if err := ad.UpdateAd(updatedAd); err != nil {
		http.Error(w, "Failed to update ad", http.StatusInternalServerError)
		return
	}

	_ = templates.SuccessMessageWithRedirect("Ad updated successfully!", fmt.Sprintf("/ad/%d", adID)).Render(w)
}

func HandleDeleteAd(w http.ResponseWriter, r *http.Request) {
	currentUser, err := GetCurrentUser(r)
	if err != nil || currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var adID int
	fmt.Sscanf(r.PathValue("id"), "%d", &adID)

	if err := ad.DeleteAd(adID); err != nil {
		http.Error(w, "Failed to delete ad", http.StatusInternalServerError)
		return
	}

	_ = templates.SuccessMessageWithRedirect("Ad deleted successfully!", "/").Render(w)
}
