package handlers

import (
	"fmt"
	"net/http"
	"sort"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/components"
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

	_ = components.Page(
		"New Ad - Parts Pile",
		currentUser,
		r.URL.Path,
		[]g.Node{
			components.PageHeader("Create New Ad"),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				components.ValidationErrorContainer(),
				components.FormGroup("Make", "make",
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
				components.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
					),
				),
				components.FormGroup("Price", "price",
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
					),
				),
				components.StyledButton("Submit", components.ButtonPrimary,
					Type("submit"),
					hx.Post("/api/new-ad"),
					hx.Target("#result"),
				),
				components.ResultContainer(),
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
		_ = components.ValidationError("Please select a make first").Render(w)
		return
	}

	// Validate required selections
	if len(r.Form["years"]) == 0 {
		_ = components.ValidationError("Please select at least one year").Render(w)
		return
	}

	if len(r.Form["models"]) == 0 {
		_ = components.ValidationError("Please select at least one model").Render(w)
		return
	}

	if len(r.Form["engines"]) == 0 {
		_ = components.ValidationError("Please select at least one engine size").Render(w)
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

	_ = components.SuccessMessageWithRedirect("Ad created successfully!", "/").Render(w)
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
		editButton = components.StyledLink("Edit Ad", fmt.Sprintf("/edit-ad/%d", ad.ID), components.ButtonPrimary)
		deleteButton = components.DeleteButton(ad.ID)
	} else {
		editButton = components.StyledLinkDisabled("Edit Ad", components.ButtonPrimary)
		deleteButton = components.StyledLinkDisabled("Delete Ad", components.ButtonDanger)
	}

	_ = components.Page(
		fmt.Sprintf("Ad %d - Parts Pile", ad.ID),
		currentUser,
		r.URL.Path,
		[]g.Node{
			Div(
				Class("max-w-2xl mx-auto"),
				components.PageHeader(ad.Make),
				components.AdDetails(ad),
				components.ActionButtons(
					components.BackToListingsButton(),
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
			components.Checkbox("years", year, year, isChecked, false,
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
			components.Checkbox("models", model, model, isChecked, !isAvailable,
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
			components.Checkbox("engines", engine, engine, isChecked, !isAvailable),
		)
	}

	_ = components.Page(
		"Edit Ad - Parts Pile",
		currentUser,
		r.URL.Path,
		[]g.Node{
			components.PageHeader("Edit Ad"),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				components.ValidationErrorContainer(),
				components.FormGroup("Make", "make",
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
					components.SectionHeader("Years", ""),
					components.GridContainer(4, yearCheckboxes...),
				),
				Div(
					ID("modelsDiv"),
					Class("space-y-4"),
					components.SectionHeader("Models", "Grayed out models are not available for all selected years"),
					components.GridContainer(2, modelCheckboxes...),
				),
				Div(
					ID("enginesDiv"),
					Class("space-y-4"),
					components.SectionHeader("Engines", "Grayed out engines are not available for all selected year-model combinations"),
					components.GridContainer(2, engineCheckboxes...),
				),
				components.FormGroup("Description", "description",
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
						g.Text(ad.Description),
					),
				),
				components.FormGroup("Price", "price",
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
				components.StyledButton("Update", components.ButtonPrimary,
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
		_ = components.ValidationError("Please select a make first").Render(w)
		return
	}

	// Validate required selections
	if len(r.Form["years"]) == 0 {
		_ = components.ValidationError("Please select at least one year").Render(w)
		return
	}

	if len(r.Form["models"]) == 0 {
		_ = components.ValidationError("Please select at least one model").Render(w)
		return
	}

	if len(r.Form["engines"]) == 0 {
		_ = components.ValidationError("Please select at least one engine size").Render(w)
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

	_ = components.SuccessMessageWithRedirect("Ad updated successfully!", fmt.Sprintf("/ad/%d", adID)).Render(w)
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

	_ = components.SuccessMessageWithRedirect("Ad deleted successfully!", "/").Render(w)
}
