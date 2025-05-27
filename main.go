package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

type VehicleData map[string]map[string]map[string][]string

type Ad struct {
	ID          int      `json:"id"`
	Make        string   `json:"make"`
	Years       []string `json:"years"`
	Models      []string `json:"models"`
	Engines     []string `json:"engines"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
}

var (
	vehicleData VehicleData
	ads         []Ad
	adsMutex    sync.Mutex
	nextAdID    = 1
)

func main() {
	// Load vehicle data
	data, err := os.ReadFile("make-year-model.json")
	if err != nil {
		log.Fatal("Error reading vehicle data:", err)
	}

	if err := json.Unmarshal(data, &vehicleData); err != nil {
		log.Fatal("Error parsing vehicle data:", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/new-ad", handleNewAd)
	mux.HandleFunc("/edit-ad/", handleEditAd)
	mux.HandleFunc("/api/makes", handleMakes)
	mux.HandleFunc("/api/years", handleYears)
	mux.HandleFunc("/api/models", handleModels)
	mux.HandleFunc("/api/engines", handleEngines)
	mux.HandleFunc("/api/new-ad", handleNewAdSubmission)
	mux.HandleFunc("/api/update-ad", handleUpdateAdSubmission)
	mux.HandleFunc("/ad/", handleViewAd)

	fmt.Printf("Starting server on port %s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	adsList := []g.Node{}
	adsMutex.Lock()
	for _, ad := range ads {
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
	adsMutex.Unlock()

	_ = Page(
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

func handleNewAd(w http.ResponseWriter, r *http.Request) {
	makes := getMakes()
	makeOptions := []g.Node{}
	for _, makeName := range makes {
		makeOptions = append(makeOptions,
			Option(Value(makeName), g.Text(makeName)),
		)
	}

	_ = Page(
		"New Ad - Parts Pile",
		[]g.Node{
			H1(Class("text-4xl font-bold mb-8"), g.Text("Create New Ad")),
			Form(
				ID("newAdForm"),
				Class("space-y-6"),
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

func handleMakes(w http.ResponseWriter, r *http.Request) {
	makes := getMakes()
	json.NewEncoder(w).Encode(makes)
}

func handleYears(w http.ResponseWriter, r *http.Request) {
	makeName := r.URL.Query().Get("make")
	if makeName == "" {
		http.Error(w, "Make is required", http.StatusBadRequest)
		return
	}

	years := getYears(makeName)
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

func getModelsWithAvailability(makeName string, years []string) map[string]bool {
	// Get all possible models first
	allModels := make(map[string]bool)
	availableInAllYears := make(map[string]bool)

	if makeData, ok := vehicleData[makeName]; ok {
		// First pass: collect all models and mark them as potentially available
		for _, year := range years {
			if yearData, ok := makeData[year]; ok {
				for model := range yearData {
					allModels[model] = true
					if _, exists := availableInAllYears[model]; !exists {
						availableInAllYears[model] = true
					}
				}
			}
		}

		// Second pass: check if each model exists in all selected years
		for model := range allModels {
			for _, year := range years {
				if yearData, ok := makeData[year]; ok {
					if _, hasModel := yearData[model]; !hasModel {
						availableInAllYears[model] = false
						break
					}
				}
			}
		}
	}
	return availableInAllYears
}

func getEnginesWithAvailability(makeName string, years []string, models []string) map[string]bool {
	// Get all possible engines first
	allEngines := make(map[string]bool)
	availableInAllCombos := make(map[string]bool)

	if makeData, ok := vehicleData[makeName]; ok {
		// First pass: collect all engines
		for _, year := range years {
			if yearData, ok := makeData[year]; ok {
				for _, model := range models {
					if engines, ok := yearData[model]; ok {
						for _, engine := range engines {
							allEngines[engine] = true
							if _, exists := availableInAllCombos[engine]; !exists {
								availableInAllCombos[engine] = true
							}
						}
					}
				}
			}
		}

		// Second pass: check if each engine exists for all selected year-model combinations
		for engine := range allEngines {
			for _, year := range years {
				if yearData, ok := makeData[year]; ok {
					for _, model := range models {
						if engines, ok := yearData[model]; ok {
							engineFound := false
							for _, e := range engines {
								if e == engine {
									engineFound = true
									break
								}
							}
							if !engineFound {
								availableInAllCombos[engine] = false
								break
							}
						} else {
							availableInAllCombos[engine] = false
							break
						}
					}
					if !availableInAllCombos[engine] {
						break
					}
				}
			}
		}
	}
	return availableInAllCombos
}

func handleModels(w http.ResponseWriter, r *http.Request) {
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

	modelAvailability := getModelsWithAvailability(makeName, years)
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

func handleEngines(w http.ResponseWriter, r *http.Request) {
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

	engineAvailability := getEnginesWithAvailability(makeName, years, models)
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

func handleNewAdSubmission(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	price := 0.0
	fmt.Sscanf(r.FormValue("price"), "%f", &price)

	ad := Ad{
		ID:          nextAdID,
		Make:        r.FormValue("make"),
		Years:       r.Form["years"],
		Models:      r.Form["models"],
		Engines:     r.Form["engines"],
		Description: r.FormValue("description"),
		Price:       price,
	}

	adsMutex.Lock()
	ads = append(ads, ad)
	nextAdID++
	adsMutex.Unlock()

	_ = Div(
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Text("Ad created successfully! Redirecting..."),
		Script(g.Raw("setTimeout(function() { window.location = '/' }, 1000)")),
	).Render(w)
}

func handleViewAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.URL.Path[4:], "%d", &adID)

	adsMutex.Lock()
	var ad Ad
	for _, a := range ads {
		if a.ID == adID {
			ad = a
			break
		}
	}
	adsMutex.Unlock()

	if ad.ID == 0 {
		http.NotFound(w, r)
		return
	}

	_ = Page(
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

func handleEditAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.URL.Path[9:], "%d", &adID)

	adsMutex.Lock()
	var ad Ad
	for _, a := range ads {
		if a.ID == adID {
			ad = a
			break
		}
	}
	adsMutex.Unlock()

	if ad.ID == 0 {
		http.NotFound(w, r)
		return
	}

	// Prepare make options
	makes := getMakes()
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
	years := getYears(ad.Make)
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
	modelAvailability := getModelsWithAvailability(ad.Make, ad.Years)
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
	engineAvailability := getEnginesWithAvailability(ad.Make, ad.Years, ad.Models)
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

	_ = Page(
		"Edit Ad - Parts Pile",
		[]g.Node{
			H1(Class("text-4xl font-bold mb-8"), g.Text("Edit Ad")),
			Form(
				ID("editAdForm"),
				Class("space-y-6"),
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

func handleUpdateAdSubmission(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	var adID int
	fmt.Sscanf(r.FormValue("id"), "%d", &adID)

	price := 0.0
	fmt.Sscanf(r.FormValue("price"), "%f", &price)

	updatedAd := Ad{
		ID:          adID,
		Make:        r.FormValue("make"),
		Years:       r.Form["years"],
		Models:      r.Form["models"],
		Engines:     r.Form["engines"],
		Description: r.FormValue("description"),
		Price:       price,
	}

	adsMutex.Lock()
	for i, ad := range ads {
		if ad.ID == adID {
			ads[i] = updatedAd
			break
		}
	}
	adsMutex.Unlock()

	_ = Div(
		Class("bg-green-100 border-green-500 text-green-700 px-4 py-3 rounded"),
		g.Text("Ad updated successfully! Redirecting..."),
		Script(g.Raw(fmt.Sprintf("setTimeout(function() { window.location = '/ad/%d' }, 1000)", adID))),
	).Render(w)
}

// Helper function to convert a slice of strings to a JavaScript array literal
func toJSArray(items []string) string {
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = fmt.Sprintf("%q", item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func getMakes() []string {
	makesList := make([]string, 0, len(vehicleData))
	for makeName := range vehicleData {
		makesList = append(makesList, makeName)
	}
	sort.Strings(makesList)
	return makesList
}

func getYears(makeName string) []string {
	years := make([]string, 0)
	if makeData, ok := vehicleData[makeName]; ok {
		for year := range makeData {
			years = append(years, year)
		}
	}
	sort.Strings(years)
	return years
}

func getModels(makeName string, years []string) []string {
	modelSet := make(map[string]bool)
	if makeData, ok := vehicleData[makeName]; ok {
		for _, year := range years {
			if yearData, ok := makeData[year]; ok {
				for model := range yearData {
					modelSet[model] = true
				}
			}
		}
	}

	models := make([]string, 0, len(modelSet))
	for model := range modelSet {
		models = append(models, model)
	}
	sort.Strings(models)
	return models
}

func getEngines(makeName string, years []string, models []string) []string {
	engineSet := make(map[string]bool)
	if makeData, ok := vehicleData[makeName]; ok {
		for _, year := range years {
			if yearData, ok := makeData[year]; ok {
				for _, model := range models {
					if engines, ok := yearData[model]; ok {
						for _, engine := range engines {
							engineSet[engine] = true
						}
					}
				}
			}
		}
	}

	engines := make([]string, 0, len(engineSet))
	for engine := range engineSet {
		engines = append(engines, engine)
	}
	sort.Strings(engines)
	return engines
}

func Page(title string, content []g.Node) g.Node {
	return HTML(
		Head(
			Meta(Charset("utf-8")),
			Meta(Name("viewport"), Content("width=device-width, initial-scale=1")),
			Title(title),
			Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css")),
			Script(Src("https://unpkg.com/htmx.org@1.9.10")),
		),
		Body(
			Div(
				Class("container mx-auto px-4 py-8"),
				g.Group(content),
			),
		),
	)
}
