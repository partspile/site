package ui

import (
	"encoding/json"
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

func SearchResultsContainer(newAdButton g.Node, filters SearchSchema, ads map[int]ad.Ad, loc *time.Location, view string, query string) g.Node {
	// Marshal the structured query as JSON for the hidden input
	structuredQueryJSON, _ := json.Marshal(filters)
	return Div(
		ID("searchResults"),
		SearchWidget(newAdButton, view, query),
		Input(
			Type("hidden"),
			Name("structured_query"),
			Value(string(structuredQueryJSON)),
		),
		Div(
			Class("text-xs text-red-600 mb-2"),
			g.Text("[DEBUG] Current view: "+view),
		),
		Div(
			ID("searchFilters"),
			Class("flex flex-wrap gap-4 mb-4"),
			SearchFilters(filters),
		),

		// View toggle buttons
		ViewToggleButtons(view),

		// View Wrapper
		Div(
			ID("view-wrapper"),
			func() g.Node {
				structuredQueryJSON, _ := json.Marshal(filters)
				if view == "tree" {
					return TreeViewWithQuery(query, string(structuredQueryJSON))
				}
				return ListView(ads, loc)
			}(),
		),
	)
}

func ViewToggleButtons(activeView string) g.Node {
	listClass := "px-2 py-1 rounded text-sm"
	treeClass := "px-2 py-1 rounded text-sm"
	if activeView == "list" {
		listClass += " bg-blue-500 text-white"
		treeClass += " bg-gray-200"
	} else {
		listClass += " bg-gray-200"
		treeClass += " bg-blue-500 text-white"
	}

	return Div(
		Class("flex justify-end gap-2 my-4"),
		Button(
			Class(listClass),
			hx.Post("/view/list"),
			hx.Target("#searchResults"),
			hx.Indicator("#searchWaiting"),
			hx.Include("[name='q'],[name='structured_query'],[name='view']"),
			g.Text("List View"),
		),
		Button(
			Class(treeClass),
			hx.Post("/view/tree"),
			hx.Target("#searchResults"),
			hx.Indicator("#searchWaiting"),
			hx.Include("[name='q'],[name='structured_query'],[name='view']"),
			g.Text("Tree View"),
		),
	)
}

func ListView(ads map[int]ad.Ad, loc *time.Location) g.Node {
	return Div(
		ID("list-view"),
		AdListContainer(
			g.Group(BuildAdListNodes(ads, loc)),
		),
	)
}

func TreeView() g.Node {
	return Div(
		ID("tree-view"),
		hx.Get("/tree"),
		hx.Trigger("load"),
		hx.Swap("innerHTML"),
	)
}

func TreeViewWithQuery(query, structuredQuery string) g.Node {
	return Div(
		ID("tree-view"),
		hx.Get("/tree?q="+query+"&structured_query="+structuredQuery),
		hx.Trigger("load"),
		hx.Swap("innerHTML"),
	)
}

func TreeViewWithStructuredQuery(query, structuredQuery string) g.Node {
	return TreeViewWithQuery(query, structuredQuery)
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

func SearchWidget(newAdButton g.Node, view string, query string) g.Node {
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
				Input(Type("hidden"), Name("view"), Value(view), ID("view-type-input")),
				Input(
					Type("search"),
					ID("searchBox"),
					Name("q"),
					Value(query),
					Class("w-full p-2 border rounded"),
					Placeholder("Search by make, year, model, or description..."),
					hx.Trigger("keyup changed delay:500ms, search"),
				),
			),
		),
		Div(
			ID("searchWaiting"),
			Class("htmx-indicator absolute inset-0 flex items-center justify-center bg-white bg-opacity-60 z-10 pointer-events-none"),
			Img(
				Src("/spinner.gif"),
				Alt("Loading..."),
				Class("w-12 h-12 pointer-events-auto"),
			),
		),
	)
}
