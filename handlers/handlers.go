package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/sfeldma/parts-pile/site/models"
	"github.com/sfeldma/parts-pile/site/templates"
	"github.com/sfeldma/parts-pile/site/vehicle"
)

func HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	adsList := []g.Node{}
	vehicle.AdsMutex.Lock()
	for _, ad := range vehicle.Ads {
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
			H1(Class("text-4xl font-bold mb-8"), g.Text("Parts Pile")),
			Div(
				Class("mb-8"),
				A(
					Href("/new-ad"),
					Class("bg-blue-500 text-white px-6 py-2 rounded hover:bg-blue-600"),
					g.Text("New Ad"),
				),
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
			H1(Class("text-4xl font-bold mb-8"), g.Text("Create New Ad")),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
				Div(
					ID("validationError"),
					Class("hidden bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded mb-4"),
				),
				Div(
					Class("space-y-2"),
					Label(For("make"), Class("block"), g.Text("Make")),
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
				Div(
					Class("space-y-2"),
					Label(For("description"), Class("block"), g.Text("Description")),
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
					),
				),
				Div(
					Class("space-y-2"),
					Label(For("price"), Class("block"), g.Text("Price")),
					Input(
						Type("number"),
						ID("price"),
						Name("price"),
						Class("w-full p-2 border rounded"),
						Step("0.01"),
					),
				),
				Button(
					Type("submit"),
					Class("bg-blue-500 text-white px-6 py-2 rounded hover:bg-blue-600"),
					hx.Post("/api/new-ad"),
					hx.Target("#result"),
					g.Text("Submit"),
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
			Div(
				Class("flex items-center space-x-2"),
				Input(
					Type("checkbox"),
					Name("years"),
					Value(year),
					ID("year-"+year),
					hx.Trigger("change"),
					hx.Get("/api/models"),
					hx.Target("#modelsDiv"),
					hx.Include("[name='make'],[name='years']:checked"),
					hx.Swap("innerHTML"),
					g.Attr("onclick", "document.getElementById('enginesDiv').innerHTML = ''"),
				),
				Label(For("year-"+year), g.Text(year)),
			),
		)
	}

	_ = Div(
		ID("yearsDiv"),
		Class("space-y-4"),
		Label(Class("block font-bold"), g.Text("Years")),
		Div(
			Class("grid grid-cols-4 gap-4"),
			g.Group(checkboxes),
		),
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
			Label(Class("block font-bold"), g.Text("Models")),
			P(
				Class("text-sm text-gray-600 mb-2"),
				g.Text("Select one or more years to see available models"),
			),
		).Render(w)

		// Also clear the engines div
		_ = Div(
			ID("enginesDiv"),
			Class("space-y-4"),
			Label(Class("block font-bold"), g.Text("Engines")),
			P(
				Class("text-sm text-gray-600 mb-2"),
				g.Text("Select one or more models to see available engines"),
			),
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
		inputAttrs := []g.Node{
			Type("checkbox"),
			Name("models"),
			Value(model),
			ID("model-" + model),
			hx.Trigger("change"),
			hx.Get("/api/engines"),
			hx.Target("#enginesDiv"),
			hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
			hx.Swap("innerHTML"),
		}
		if !isAvailable {
			inputAttrs = append(inputAttrs, Disabled())
			inputAttrs = append(inputAttrs, g.Attr("class", "opacity-50 cursor-not-allowed"))
		}

		checkboxes = append(checkboxes,
			Div(
				Class("flex items-center space-x-2"),
				Input(inputAttrs...),
				Label(
					For("model-"+model),
					func() g.Node {
						if !isAvailable {
							return Class("text-gray-400")
						}
						return g.Text("")
					}(),
					g.Text(model),
				),
			),
		)
	}

	_ = Div(
		ID("modelsDiv"),
		Class("space-y-4"),
		Label(Class("block font-bold"), g.Text("Models")),
		P(
			Class("text-sm text-gray-600 mb-2"),
			g.Text("Grayed out models are not available for all selected years"),
		),
		Div(
			Class("grid grid-cols-2 gap-4"),
			g.Group(checkboxes),
		),
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
			Label(Class("block font-bold"), g.Text("Engines")),
			P(
				Class("text-sm text-gray-600 mb-2"),
				g.Text("Select one or more models to see available engines"),
			),
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
		inputAttrs := []g.Node{
			Type("checkbox"),
			Name("engines"),
			Value(engine),
			ID("engine-" + engine),
		}
		if !isAvailable {
			inputAttrs = append(inputAttrs, Disabled())
			inputAttrs = append(inputAttrs, g.Attr("class", "opacity-50 cursor-not-allowed"))
		}

		checkboxes = append(checkboxes,
			Div(
				Class("flex items-center space-x-2"),
				Input(inputAttrs...),
				Label(
					For("engine-"+engine),
					func() g.Node {
						if !isAvailable {
							return Class("text-gray-400")
						}
						return g.Text("")
					}(),
					g.Text(engine),
				),
			),
		)
	}

	_ = Div(
		ID("enginesDiv"),
		Class("space-y-4"),
		Label(Class("block font-bold"), g.Text("Engines")),
		P(
			Class("text-sm text-gray-600 mb-2"),
			g.Text("Grayed out engines are not available for all selected year-model combinations"),
		),
		Div(
			Class("grid grid-cols-2 gap-4"),
			g.Group(checkboxes),
		),
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

	ad := models.Ad{
		ID:          vehicle.NextAdID,
		Make:        r.FormValue("make"),
		Years:       r.Form["years"],
		Models:      r.Form["models"],
		Engines:     r.Form["engines"],
		Description: r.FormValue("description"),
		Price:       price,
	}

	vehicle.AdsMutex.Lock()
	vehicle.Ads = append(vehicle.Ads, ad)
	vehicle.NextAdID++
	vehicle.AdsMutex.Unlock()

	_ = templates.SuccessMessage(
		"Ad created successfully! Redirecting...",
		"setTimeout(function() { window.location = '/' }, 1000)",
	).Render(w)
}

func HandleViewAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.URL.Path[4:], "%d", &adID)

	vehicle.AdsMutex.Lock()
	var ad models.Ad
	for _, a := range vehicle.Ads {
		if a.ID == adID {
			ad = a
			break
		}
	}
	vehicle.AdsMutex.Unlock()

	if ad.ID == 0 {
		http.NotFound(w, r)
		return
	}

	_ = templates.Page(
		fmt.Sprintf("Ad %d - Parts Pile", ad.ID),
		[]g.Node{
			Div(
				Class("max-w-2xl mx-auto"),
				H1(Class("text-4xl font-bold mb-8"), g.Text(ad.Make)),
				Div(
					Class("space-y-4"),
					P(Class("text-gray-600"), g.Text(fmt.Sprintf("Years: %v", ad.Years))),
					P(Class("text-gray-600"), g.Text(fmt.Sprintf("Models: %v", ad.Models))),
					P(Class("text-gray-600"), g.Text(fmt.Sprintf("Engines: %v", ad.Engines))),
					P(Class("mt-4"), g.Text(ad.Description)),
					P(Class("text-2xl font-bold mt-4"), g.Text(fmt.Sprintf("$%.2f", ad.Price))),
				),
				Div(
					Class("mt-8 space-x-4"),
					A(
						Href("/"),
						Class("text-blue-500 hover:underline"),
						g.Text("← Back to listings"),
					),
					A(
						Href(fmt.Sprintf("/edit-ad/%d", ad.ID)),
						Class("bg-blue-500 text-white px-4 py-2 rounded hover:bg-blue-600"),
						g.Text("Edit Ad"),
					),
				),
			),
		},
	).Render(w)
}

func HandleEditAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.URL.Path[9:], "%d", &adID)

	vehicle.AdsMutex.Lock()
	var ad models.Ad
	for _, a := range vehicle.Ads {
		if a.ID == adID {
			ad = a
			break
		}
	}
	vehicle.AdsMutex.Unlock()

	if ad.ID == 0 {
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
		attrs := []g.Node{
			Type("checkbox"),
			Name("years"),
			Value(year),
			ID("year-" + year),
			hx.Trigger("change"),
			hx.Get("/api/models"),
			hx.Target("#modelsDiv"),
			hx.Include("[name='make'],[name='years']:checked"),
			hx.Swap("innerHTML"),
		}
		if isChecked {
			attrs = append(attrs, Checked())
		}
		yearCheckboxes = append(yearCheckboxes,
			Div(
				Class("flex items-center space-x-2"),
				Input(attrs...),
				Label(For("year-"+year), g.Text(year)),
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
		inputAttrs := []g.Node{
			Type("checkbox"),
			Name("models"),
			Value(model),
			ID("model-" + model),
			hx.Trigger("change"),
			hx.Get("/api/engines"),
			hx.Target("#enginesDiv"),
			hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
			hx.Swap("innerHTML"),
		}
		if !isAvailable {
			inputAttrs = append(inputAttrs, Disabled())
			inputAttrs = append(inputAttrs, g.Attr("class", "opacity-50 cursor-not-allowed"))
		}
		if isChecked && isAvailable {
			inputAttrs = append(inputAttrs, Checked())
		}

		modelCheckboxes = append(modelCheckboxes,
			Div(
				Class("flex items-center space-x-2"),
				Input(inputAttrs...),
				Label(
					For("model-"+model),
					func() g.Node {
						if !isAvailable {
							return Class("text-gray-400")
						}
						return g.Text("")
					}(),
					g.Text(model),
				),
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
		inputAttrs := []g.Node{
			Type("checkbox"),
			Name("engines"),
			Value(engine),
			ID("engine-" + engine),
		}
		if !isAvailable {
			inputAttrs = append(inputAttrs, Disabled())
			inputAttrs = append(inputAttrs, g.Attr("class", "opacity-50 cursor-not-allowed"))
		}
		if isChecked && isAvailable {
			inputAttrs = append(inputAttrs, Checked())
		}

		engineCheckboxes = append(engineCheckboxes,
			Div(
				Class("flex items-center space-x-2"),
				Input(inputAttrs...),
				Label(
					For("engine-"+engine),
					func() g.Node {
						if !isAvailable {
							return Class("text-gray-400")
						}
						return g.Text("")
					}(),
					g.Text(engine),
				),
			),
		)
	}

	_ = templates.Page(
		"Edit Ad - Parts Pile",
		[]g.Node{
			H1(Class("text-4xl font-bold mb-8"), g.Text("Edit Ad")),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
				Div(
					ID("validationError"),
					Class("hidden bg-red-100 border-red-500 text-red-700 px-4 py-3 rounded mb-4"),
				),
				Div(
					Class("space-y-2"),
					Label(For("make"), Class("block"), g.Text("Make")),
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
					Label(Class("block font-bold"), g.Text("Years")),
					Div(
						Class("grid grid-cols-4 gap-4"),
						g.Group(yearCheckboxes),
					),
				),
				Div(
					ID("modelsDiv"),
					Class("space-y-4"),
					Label(Class("block font-bold"), g.Text("Models")),
					P(
						Class("text-sm text-gray-600 mb-2"),
						g.Text("Grayed out models are not available for all selected years"),
					),
					Div(
						Class("grid grid-cols-2 gap-4"),
						g.Group(modelCheckboxes),
					),
				),
				Div(
					ID("enginesDiv"),
					Class("space-y-4"),
					Label(Class("block font-bold"), g.Text("Engines")),
					P(
						Class("text-sm text-gray-600 mb-2"),
						g.Text("Grayed out engines are not available for all selected year-model combinations"),
					),
					Div(
						Class("grid grid-cols-2 gap-4"),
						g.Group(engineCheckboxes),
					),
				),
				Div(
					Class("space-y-2"),
					Label(For("description"), Class("block"), g.Text("Description")),
					Textarea(
						ID("description"),
						Name("description"),
						Class("w-full p-2 border rounded"),
						Rows("4"),
						g.Text(ad.Description),
					),
				),
				Div(
					Class("space-y-2"),
					Label(For("price"), Class("block"), g.Text("Price")),
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
				Button(
					Type("submit"),
					Class("bg-blue-500 text-white px-6 py-2 rounded hover:bg-blue-600"),
					hx.Post("/api/update-ad"),
					hx.Target("#result"),
					g.Text("Update"),
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

	updatedAd := models.Ad{
		ID:          adID,
		Make:        r.FormValue("make"),
		Years:       r.Form["years"],
		Models:      r.Form["models"],
		Engines:     r.Form["engines"],
		Description: r.FormValue("description"),
		Price:       price,
	}

	vehicle.AdsMutex.Lock()
	for i, ad := range vehicle.Ads {
		if ad.ID == adID {
			vehicle.Ads[i] = updatedAd
			break
		}
	}
	vehicle.AdsMutex.Unlock()

	_ = templates.SuccessMessage(
		"Ad updated successfully! Redirecting...",
		fmt.Sprintf("setTimeout(function() { window.location = '/ad/%d' }, 1000)", adID),
	).Render(w)
}
