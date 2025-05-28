package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/sfeldma/parts-pile/site/ad"
	"github.com/sfeldma/parts-pile/site/templates"
	"github.com/sfeldma/parts-pile/site/vehicle"
)

// saveAdsToFile saves the current ads to ads.json
func saveAdsToFile() {
	vehicle.AdsMutex.Lock()
	defer vehicle.AdsMutex.Unlock()
	f, err := os.Create("ads.json")
	if err != nil {
		log.Printf("Error creating ads.json: %v", err)
		return
	}
	defer f.Close()

	adsJSON, err := json.MarshalIndent(vehicle.Ads, "", "\t")
	if err != nil {
		log.Printf("Error encoding ads to ads.json: %v", err)
		return
	}
	if _, err := f.Write(adsJSON); err != nil {
		log.Printf("Error writing ads to ads.json: %v", err)
	}
}

func HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	adsList := []g.Node{}
	vehicle.AdsMutex.Lock()
	// Collect and sort ad IDs
	adIDs := make([]int, 0, len(vehicle.Ads))
	for id := range vehicle.Ads {
		adIDs = append(adIDs, id)
	}
	sort.Ints(adIDs)
	for _, id := range adIDs {
		ad := vehicle.Ads[id]
		adsList = append(adsList,
			A(
				Href(fmt.Sprintf("/ad/%d", ad.ID)),
				Class("block border p-4 mb-4 rounded hover:bg-gray-50"),
				Div(
					H3(Class("text-xl font-bold"), g.Text(ad.Make)),
					P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", ad.Years))),
					P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", ad.Models))),
					P(Class("mt-2"), g.Text(ad.Description)),
					P(Class("text-xl font-bold mt-2"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
				),
			),
		)
	}
	vehicle.AdsMutex.Unlock()

	_ = templates.Page(
		"Parts Pile - Auto Parts and Sales",
		[]g.Node{
			templates.PageHeader("Parts Pile"),
			Div(
				Class("mb-8"),
				templates.StyledLink("New Ad", "/new-ad", templates.ButtonPrimary),
			),
			Div(
				Class("space-y-4"),
				g.Group(adsList),
			),
		},
	).Render(w)
}

func HandleNewAd(w http.ResponseWriter, r *http.Request) {
	makes := vehicle.GetMakes()
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		makeOptions = append(makeOptions,
			Option(Value(makeName), g.Text(makeName)),
		)
	}

	_ = templates.Page(
		"New Ad - Parts Pile",
		[]g.Node{
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
				Div(
					ID("result"),
					Class("mt-4"),
				),
			),
		},
	).Render(w)
}

func HandleMakes(w http.ResponseWriter, r *http.Request) {
	makes := vehicle.GetMakes()
	json.NewEncoder(w).Encode(makes)
}

func HandleYears(w http.ResponseWriter, r *http.Request) {
	makeName := r.URL.Query().Get("make")
	if makeName == "" {
		http.Error(w, "Make is required", http.StatusBadRequest)
		return
	}

	years := vehicle.GetYears(makeName)
	checkboxes := []g.Node{}

	for _, year := range years {
		checkboxes = append(checkboxes,
			templates.Checkbox("years", year, year, false, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
				hx.Swap("innerHTML"),
				g.Attr("onclick", "document.getElementById('enginesDiv').innerHTML = ''"),
			),
		)
	}

	_ = Div(
		ID("yearsDiv"),
		Class("space-y-4"),
		templates.SectionHeader("Years", ""),
		templates.GridContainer(4, checkboxes...),
	).Render(w)
}

func HandleModels(w http.ResponseWriter, r *http.Request) {
	makeName := r.URL.Query().Get("make")
	years := r.URL.Query()["years"]
	if makeName == "" {
		http.Error(w, "Make is required", http.StatusBadRequest)
		return
	}

	// If no years are selected, just show an empty models div
	if len(years) == 0 {
		_ = Div(
			ID("modelsDiv"),
			Class("space-y-4"),
			templates.SectionHeader("Models", "Select one or more years to see available models"),
		).Render(w)

		// Also clear the engines div
		_ = Div(
			ID("enginesDiv"),
			Class("space-y-4"),
			templates.SectionHeader("Engines", "Select one or more models to see available engines"),
		).Render(w)
		return
	}

	modelAvailability := vehicle.GetModelsWithAvailability(makeName, years)
	checkboxes := []g.Node{}

	// Sort models for consistent display
	models := make([]string, 0, len(modelAvailability))
	for model := range modelAvailability {
		models = append(models, model)
	}
	sort.Strings(models)

	for _, model := range models {
		isAvailable := modelAvailability[model]
		checkboxes = append(checkboxes,
			templates.Checkbox("models", model, model, false, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}

	_ = Div(
		ID("modelsDiv"),
		Class("space-y-4"),
		templates.SectionHeader("Models", "Grayed out models are not available for all selected years"),
		templates.GridContainer(2, checkboxes...),
	).Render(w)
}

func HandleEngines(w http.ResponseWriter, r *http.Request) {
	makeName := r.URL.Query().Get("make")
	years := r.URL.Query()["years"]
	models := r.URL.Query()["models"]
	if makeName == "" {
		http.Error(w, "Make is required", http.StatusBadRequest)
		return
	}

	// If no years or models are selected, show an empty engines div
	if len(years) == 0 || len(models) == 0 {
		_ = Div(
			ID("enginesDiv"),
			Class("space-y-4"),
			templates.SectionHeader("Engines", "Select one or more models to see available engines"),
		).Render(w)
		return
	}

	engineAvailability := vehicle.GetEnginesWithAvailability(makeName, years, models)
	checkboxes := []g.Node{}

	// Sort engines for consistent display
	engines := make([]string, 0, len(engineAvailability))
	for engine := range engineAvailability {
		engines = append(engines, engine)
	}
	sort.Strings(engines)

	for _, engine := range engines {
		isAvailable := engineAvailability[engine]
		checkboxes = append(checkboxes,
			templates.Checkbox("engines", engine, engine, false, !isAvailable),
		)
	}

	_ = Div(
		ID("enginesDiv"),
		Class("space-y-4"),
		templates.SectionHeader("Engines", "Grayed out engines are not available for all selected year-model combinations"),
		templates.GridContainer(2, checkboxes...),
	).Render(w)
}

func HandleNewAdSubmission(w http.ResponseWriter, r *http.Request) {
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

	ad := ad.Ad{
		ID:          vehicle.NextAdID,
		Make:        r.FormValue("make"),
		Years:       r.Form["years"],
		Models:      r.Form["models"],
		Engines:     r.Form["engines"],
		Description: r.FormValue("description"),
		Price:       price,
	}

	vehicle.AdsMutex.Lock()
	vehicle.Ads[ad.ID] = ad
	vehicle.NextAdID++
	vehicle.AdsMutex.Unlock()

	saveAdsToFile()

	_ = templates.SuccessMessageWithRedirect("Ad created successfully!", "/").Render(w)
}

func HandleViewAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.PathValue("id"), "%d", &adID)

	vehicle.AdsMutex.Lock()
	ad, ok := vehicle.Ads[adID]
	vehicle.AdsMutex.Unlock()

	if !ok || ad.ID == 0 {
		http.NotFound(w, r)
		return
	}

	_ = templates.Page(
		fmt.Sprintf("Ad %d - Parts Pile", ad.ID),
		[]g.Node{
			Div(
				Class("max-w-2xl mx-auto"),
				templates.PageHeader(ad.Make),
				templates.AdDetails(ad),
				templates.ActionButtons(
					templates.BackToListingsButton(),
					templates.StyledLink("Edit Ad", fmt.Sprintf("/edit-ad/%d", ad.ID), templates.ButtonPrimary),
					templates.DeleteButton(ad.ID),
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
	var adID int
	fmt.Sscanf(r.PathValue("id"), "%d", &adID)

	vehicle.AdsMutex.Lock()
	ad, ok := vehicle.Ads[adID]
	vehicle.AdsMutex.Unlock()

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

	updatedAd := ad.Ad{
		ID:          adID,
		Make:        r.FormValue("make"),
		Years:       r.Form["years"],
		Models:      r.Form["models"],
		Engines:     r.Form["engines"],
		Description: r.FormValue("description"),
		Price:       price,
	}

	vehicle.AdsMutex.Lock()
	vehicle.Ads[adID] = updatedAd
	vehicle.AdsMutex.Unlock()

	saveAdsToFile()

	_ = templates.SuccessMessageWithRedirect("Ad updated successfully!", fmt.Sprintf("/ad/%d", adID)).Render(w)
}

func HandleDeleteAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.PathValue("id"), "%d", &adID)

	vehicle.AdsMutex.Lock()
	delete(vehicle.Ads, adID)
	vehicle.AdsMutex.Unlock()

	saveAdsToFile()

	_ = templates.SuccessMessageWithRedirect("Ad deleted successfully!", "/").Render(w)
}
