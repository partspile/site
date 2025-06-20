package handlers

import (
	"encoding/json"
	"net/http"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"

	"github.com/parts-pile/site/components"
	"github.com/parts-pile/site/vehicle"
)

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
			components.Checkbox("years", year, year, false, false,
				hx.Trigger("change"),
				hx.Get("/api/models"),
				hx.Target("#modelsDiv"),
				hx.Include("[name='make'],[name='years']:checked"),
				hx.Swap("innerHTML"),
				g.Attr("onclick", "document.getElementById('enginesDiv').innerHTML = ''"),
			),
		)
	}

	components.GridContainer(5, checkboxes...).Render(w)
}

func HandleModels(w http.ResponseWriter, r *http.Request) {
	makeName := r.URL.Query().Get("make")
	if makeName == "" {
		http.Error(w, "Make is required", http.StatusBadRequest)
		return
	}

	years := r.URL.Query()["years"]
	if len(years) == 0 {
		http.Error(w, "At least one year is required", http.StatusBadRequest)
		return
	}

	modelAvailability := vehicle.GetModelsWithAvailability(makeName, years)
	checkboxes := []g.Node{}
	for model, isAvailable := range modelAvailability {
		checkboxes = append(checkboxes,
			components.Checkbox("models", model, model, false, !isAvailable,
				hx.Trigger("change"),
				hx.Get("/api/engines"),
				hx.Target("#enginesDiv"),
				hx.Include("[name='make'],[name='years']:checked,[name='models']:checked"),
				hx.Swap("innerHTML"),
			),
		)
	}

	components.GridContainer(5, checkboxes...).Render(w)
}

func HandleEngines(w http.ResponseWriter, r *http.Request) {
	makeName := r.URL.Query().Get("make")
	if makeName == "" {
		http.Error(w, "Make is required", http.StatusBadRequest)
		return
	}

	years := r.URL.Query()["years"]
	if len(years) == 0 {
		http.Error(w, "At least one year is required", http.StatusBadRequest)
		return
	}

	models := r.URL.Query()["models"]
	if len(models) == 0 {
		http.Error(w, "At least one model is required", http.StatusBadRequest)
		return
	}

	engineAvailability := vehicle.GetEnginesWithAvailability(makeName, years, models)
	checkboxes := []g.Node{}
	for engine, isAvailable := range engineAvailability {
		checkboxes = append(checkboxes,
			components.Checkbox("engines", engine, engine, false, !isAvailable),
		)
	}
	components.GridContainer(5, checkboxes...).Render(w)
}
