package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/grok"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/templates"
	"github.com/parts-pile/site/vehicle"
)

const sysPrompt = `You are an expert vehicle parts assistant.

Your job is to extract a structured query from a user's search request.

Extract the make, years, models, engine sizes, category, and subcategory from
the user's search request.  Use your best judgement as a vehicle parts export
to fill out the structured query as much as possible.  When filling out the
structured query, only use values from the lists below, and not the user's values.
For example, if user entered "Ford", the structure query would use "FORD".

<Makes>
%s
</Makes>

<Years>
%s
</Years>

<Models>
%s
</Models>

<EngineSizes>
%s
</EngineSizes>

<Categories>
%s
</Categories>

<SubCategories>
%s
</SubCategories>

Return JSON encoding this Go structure with the vehicle parts data:

struct {
	Make        string
	Years       []string
	Models      []string
	EngineSizes []string
	Category    string
	SubCategory string
}

Only return the JSON.  Nothing else.
`

func HandleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Build the initial page structure with search box
	_ = templates.Page(
		"Parts Pile - Auto Parts and Sales",
		[]g.Node{
			templates.PageHeader("Parts Pile"),
			Div(
				Class("flex items-start gap-4"),
				templates.StyledLink("New Ad", "/new-ad", templates.ButtonPrimary),
				Div(
					Class("flex-1 flex flex-col gap-4 relative"),
					Form(
						ID("searchForm"),
						Class("w-full"),
						hx.Get("/search"),
						hx.Target("#searchResults"),
						hx.Indicator("#searchWaiting"),
						hx.Swap("outerHTML"),
						Input(
							Type("search"),
							ID("searchBox"),
							Name("q"),
							Class("w-full p-2 border rounded"),
							Placeholder("Search by make, year, model, or description..."),
							hx.Trigger("search"),
						),
					),
				),
				Div(
					ID("searchWaiting"),
					Class("htmx-indicator absolute inset-0 flex items-center justify-center bg-white bg-opacity-60 z-10 pointer-events-none"),
					Img(
						Src("/static/spinner.gif"),
						Alt("Loading..."),
						Class("w-12 h-12 pointer-events-auto"),
					),
				),
			),
			// Initial search results with empty query
			Div(
				ID("searchResults"),
				g.Raw(`<div hx-get="/search?q=" hx-trigger="load" hx-target="this" hx-swap="outerHTML"></div>`),
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

	if !ad.UpdateAd(adID, updatedAd) {
		http.Error(w, "Ad not found", http.StatusNotFound)
		return
	}

	_ = templates.SuccessMessageWithRedirect("Ad updated successfully!", fmt.Sprintf("/ad/%d", adID)).Render(w)
}

func HandleDeleteAd(w http.ResponseWriter, r *http.Request) {
	var adID int
	fmt.Sscanf(r.PathValue("id"), "%d", &adID)

	if !ad.DeleteAd(adID) {
		http.Error(w, "Ad not found", http.StatusNotFound)
		return
	}

	_ = templates.SuccessMessageWithRedirect("Ad deleted successfully!", "/").Render(w)
}

// SearchQuery represents a structured query for filtering ads
type SearchQuery = ad.SearchQuery

// SearchCursor represents a point in the search results for pagination
type SearchCursor = ad.SearchCursor

// EncodeCursor converts a SearchCursor to a base64 string
func EncodeCursor(c SearchCursor) string {
	b, _ := json.Marshal(c)
	return base64.URLEncoding.EncodeToString(b)
}

// DecodeCursor converts a base64 string back to a SearchCursor
func DecodeCursor(s string) (SearchCursor, error) {
	var c SearchCursor
	b, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		return c, err
	}
	err = json.Unmarshal(b, &c)
	return c, err
}

// ParseSearchQuery converts a user's text query into a structured SearchQuery
func ParseSearchQuery(q string) (SearchQuery, error) {
	if q == "" {
		// Empty query matches everything
		return SearchQuery{}, nil
	}

	// Call Grok to parse the query
	makes := strings.Join(vehicle.GetMakes(), ",")
	years := strings.Join(vehicle.GetYearRange(), ",")
	models := strings.Join(vehicle.GetAllModels(), ",")
	engineSizes := strings.Join(vehicle.GetAllEngineSizes(), ",")
	categories := strings.Join(part.GetAllCategories(), ",")
	subCategories := strings.Join(part.GetAllSubCategories(), ",")
	systemPrompt := fmt.Sprintf(sysPrompt, makes, years, models, engineSizes, categories, subCategories)

	resp, err := grok.CallGrok(systemPrompt, q)
	if err != nil {
		return SearchQuery{}, err
	}

	var query SearchQuery
	err = json.Unmarshal([]byte(resp), &query)
	return query, err
}

// FilterAds applies a SearchQuery to a list of ads
func FilterAds(query SearchQuery, ads []ad.Ad) []ad.Ad {
	if query.Make == "" && len(query.Years) == 0 && len(query.Models) == 0 &&
		len(query.EngineSizes) == 0 && query.Category == "" && query.SubCategory == "" {
		return ads // Empty query matches everything
	}

	filtered := []ad.Ad{}
	for _, ad := range ads {
		if query.Make != "" && !strings.EqualFold(ad.Make, query.Make) {
			continue
		}
		if len(query.Years) > 0 && !anyStringInSlice(ad.Years, query.Years) {
			continue
		}
		if len(query.Models) > 0 && !anyStringInSlice(ad.Models, query.Models) {
			continue
		}
		if len(query.EngineSizes) > 0 && !anyStringInSlice(ad.Engines, query.EngineSizes) {
			continue
		}
		filtered = append(filtered, ad)
	}
	return filtered
}

// GetNextPage returns the next page of ads after the cursor
func GetNextPage(query SearchQuery, cursor *SearchCursor, limit int) ([]ad.Ad, *SearchCursor, error) {
	// Get filtered page from database
	ads, hasMore, err := ad.GetFilteredAdsPageDB(query, cursor, limit)
	if err != nil {
		return nil, nil, err
	}

	// Create next cursor if there are more results
	var nextCursor *SearchCursor
	if hasMore && len(ads) > 0 {
		last := ads[len(ads)-1]
		nextCursor = &SearchCursor{
			Query:      query,
			LastID:     last.ID,
			LastPosted: last.CreatedAt,
		}
	}

	return ads, nextCursor, nil
}

// Helper: check if any string in a is in b
func anyStringInSlice(a, b []string) bool {
	for _, s := range a {
		for _, t := range b {
			if strings.EqualFold(s, t) {
				return true
			}
		}
	}
	return false
}

// HandleSearch filters ads by query and returns the filtered list as HTML for HTMX
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	userPrompt := r.URL.Query().Get("q")
	cursorStr := r.URL.Query().Get("cursor")

	var query SearchQuery
	var cursor *SearchCursor
	var err error

	// If we have a cursor, use its query instead of parsing again
	if cursorStr != "" {
		c, err := DecodeCursor(cursorStr)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		cursor = &c
		query = c.Query
	} else {
		// Only parse search query if this is the initial request
		query, err = ParseSearchQuery(userPrompt)
		if err != nil {
			http.Error(w, "Error parsing search query", http.StatusBadRequest)
			return
		}
	}

	// Get the next page of results
	limit := 10
	page, nextCursor, err := GetNextPage(query, cursor, limit)
	if err != nil {
		http.Error(w, "Error getting results", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Search request - query: %s, results: %d, hasMore: %v\n",
		userPrompt, len(page), nextCursor != nil)

	// For initial search, return the full search results structure
	var searchQuery templates.SearchSchema
	if userPrompt != "" {
		makes := strings.Join(vehicle.GetMakes(), ",")
		years := strings.Join(vehicle.GetYearRange(), ",")
		models := strings.Join(vehicle.GetAllModels(), ",")
		engineSizes := strings.Join(vehicle.GetAllEngineSizes(), ",")
		categories := strings.Join(part.GetAllCategories(), ",")
		subCategories := strings.Join(part.GetAllSubCategories(), ",")
		systemPrompt := fmt.Sprintf(sysPrompt, makes, years, models, engineSizes, categories, subCategories)
		resp, err := grok.CallGrok(systemPrompt, userPrompt)
		if err == nil {
			_ = json.Unmarshal([]byte(resp), &searchQuery)
		}
	}

	// Build ad map for SearchResultsContainer
	adsMap := make(map[int]ad.Ad)
	for _, ad := range page {
		adsMap[ad.ID] = ad
	}

	// Add the loader if there are more results
	if nextCursor != nil {
		nextCursorStr := EncodeCursor(*nextCursor)
		loaderURL := fmt.Sprintf("/search?q=%s&cursor=%s",
			htmlEscape(userPrompt),
			htmlEscape(nextCursorStr))
		adsMap[-1] = ad.Ad{Description: fmt.Sprintf(`<div class=\"htmx-indicator\" id=\"loader\" hx-get=\"%s\" hx-trigger=\"revealed\" hx-swap=\"beforeend\" hx-target=\"#adsList\" hx-on=\"htmx:afterRequest: this.remove()\">Loading more ads...</div>`, loaderURL)}
	}

	_ = templates.SearchResultsContainer(searchQuery, adsMap).Render(w)
}

// HandleSearchPage serves a page of filtered ads for infinite scrolling
func HandleSearchPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	q := r.URL.Query().Get("q")
	cursorID := 0
	cursorCreatedAt := ""
	if r.URL.Query().Get("cursor_id") != "" {
		fmt.Sscanf(r.URL.Query().Get("cursor_id"), "%d", &cursorID)
	}
	if r.URL.Query().Get("cursor_created_at") != "" {
		cursorCreatedAt = r.URL.Query().Get("cursor_created_at")
	}
	var cursorTime time.Time
	if cursorCreatedAt != "" {
		cursorTime, _ = time.Parse(time.RFC3339Nano, cursorCreatedAt)
	}
	limit := 10

	fmt.Printf("Search request - query: %s, cursorID: %d, cursorTime: %v\n", q, cursorID, cursorTime)

	// Use the same filtering logic as HandleSearch
	userPrompt := q
	var searchQuery templates.SearchSchema
	if userPrompt != "" {
		makes := strings.Join(vehicle.GetMakes(), ",")
		years := strings.Join(vehicle.GetYearRange(), ",")
		models := strings.Join(vehicle.GetAllModels(), ",")
		engineSizes := strings.Join(vehicle.GetAllEngineSizes(), ",")
		categories := strings.Join(part.GetAllCategories(), ",")
		subCategories := strings.Join(part.GetAllSubCategories(), ",")
		systemPrompt := fmt.Sprintf(sysPrompt, makes, years, models, engineSizes, categories, subCategories)
		resp, err := grok.CallGrok(systemPrompt, userPrompt)
		if err == nil {
			_ = json.Unmarshal([]byte(resp), &searchQuery)
		}
	}

	allAds := ad.GetAllAds()
	filteredAds := []ad.Ad{}
	for _, ad := range allAds {
		if searchQuery.Make != "" && !strings.EqualFold(ad.Make, searchQuery.Make) {
			continue
		}
		if len(searchQuery.Years) > 0 && !anyStringInSlice(ad.Years, searchQuery.Years) {
			continue
		}
		if len(searchQuery.Models) > 0 && !anyStringInSlice(ad.Models, searchQuery.Models) {
			continue
		}
		if len(searchQuery.EngineSizes) > 0 && !anyStringInSlice(ad.Engines, searchQuery.EngineSizes) {
			continue
		}
		filteredAds = append(filteredAds, ad)
	}
	// Sort filteredAds by CreatedAt DESC, ID DESC
	sort.Slice(filteredAds, func(i, j int) bool {
		if filteredAds[i].CreatedAt.Equal(filteredAds[j].CreatedAt) {
			return filteredAds[i].ID > filteredAds[j].ID
		}
		return filteredAds[i].CreatedAt.After(filteredAds[j].CreatedAt)
	})

	page, hasMore := ad.GetFilteredAdsPage(filteredAds, cursorID, cursorTime, limit)
	fmt.Printf("Found %d ads, hasMore=%v\n", len(page), hasMore)

	for _, ad := range page {
		_ = templates.AdCard(ad).Render(w)
	}

	if hasMore && len(page) > 0 {
		last := page[len(page)-1]
		loaderHTML := fmt.Sprintf(`<div class="htmx-indicator" id="loader" hx-get="/search?q=%s&cursor_id=%d&cursor_created_at=%s" hx-trigger="revealed" hx-swap="beforeend" hx-target="#adsList" hx-on="htmx:afterRequest: this.remove()">Loading more ads...</div>`,
			htmlEscape(q), last.ID, last.CreatedAt.Format(time.RFC3339Nano))
		fmt.Printf("Adding loader with HTML: %s\n", loaderHTML)
		fmt.Fprint(w, loaderHTML)
	}
}

// htmlEscape escapes a string for safe use in HTML attributes
func htmlEscape(s string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		"'", "&#39;",
		`"`, "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(s)
}
