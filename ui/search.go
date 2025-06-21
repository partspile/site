package ui

import (
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

type SearchSchema ad.SearchQuery

func FilterCheckbox(value string) g.Node {
	return Div(
		Class("flex items-center space-x-2"),
		Input(
			Type("checkbox"),
			Checked(),
			Disabled(),
			Class("opacity-50 cursor-not-allowed"),
		),
		Label(Class("text-gray-600"), g.Text(value)),
	)
}

func SearchFilters(filters SearchSchema) g.Node {
	if filters.Make == "" && len(filters.Years) == 0 && len(filters.Models) == 0 &&
		len(filters.EngineSizes) == 0 && filters.Category == "" && filters.SubCategory == "" {
		return g.Text("")
	}

	checkboxes := []g.Node{}

	if filters.Make != "" {
		checkboxes = append(checkboxes, FilterCheckbox(filters.Make))
	}

	for _, year := range filters.Years {
		checkboxes = append(checkboxes, FilterCheckbox(year))
	}

	for _, model := range filters.Models {
		checkboxes = append(checkboxes, FilterCheckbox(model))
	}

	for _, engine := range filters.EngineSizes {
		checkboxes = append(checkboxes, FilterCheckbox(engine))
	}

	if filters.Category != "" {
		checkboxes = append(checkboxes, FilterCheckbox(filters.Category))
	}

	if filters.SubCategory != "" {
		checkboxes = append(checkboxes, FilterCheckbox(filters.SubCategory))
	}

	return Div(
		Class("flex flex-wrap gap-4 mt-2"),
		g.Group(checkboxes),
	)
}

func SearchResultsContainer(filters SearchSchema, ads map[int]ad.Ad, loc *time.Location) g.Node {
	return Div(
		ID("searchResults"),
		Div(
			ID("searchFilters"),
			Class("flex flex-wrap gap-4 mb-4"),
			SearchFilters(filters),
		),
		AdListContainer(
			g.Group(BuildAdListNodes(ads, loc)),
		),
	)
}

func InitialSearchResults() g.Node {
	return Div(
		ID("searchResults"),
		Div(
			hx.Get("/search?q="),
			hx.Trigger("load"),
			hx.Target("this"),
			hx.Swap("outerHTML"),
		),
	)
}

func SearchWidget(newAdButton g.Node) g.Node {
	return Div(
		Class("flex items-start gap-4"),
		newAdButton,
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
	)
}
