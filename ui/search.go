package ui

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func ViewToggleButtons(activeView string) g.Node {
	icon := func(name, alt string) g.Node {
		return Img(
			Src("/images/"+name+".svg"),
			Alt(alt),
			Class("w-6 h-6 inline align-middle"),
		)
	}
	button := func(view, alt string) g.Node {
		active := activeView == view
		cls := "p-2 rounded-full border-2 "
		if active {
			cls += "border-blue-500 bg-blue-100"
		} else {
			cls += "border-transparent hover:bg-gray-100"
		}
		return Button(
			Class(cls),
			hx.Post("/view/"+view),
			hx.Target("#searchResults"),
			hx.Swap("outerHTML"),
			hx.Include("[name='q'],[name='threshold']"),
			hx.Trigger("click"),
			hx.On("click", "document.getElementById('view-type-input').value = '"+view+"'"),
			icon(view, alt),
		)
	}
	return Div(
		Class("flex justify-end gap-2 my-4"),
		button("list", "List View"),
		button("tree", "Tree View"),
		button("grid", "Grid View"),
	)
}

func InitialSearchResults(userID int, view string) g.Node {
	return Div(
		SearchWidget(userID, view, ""),
		Div(
			ID("searchResults"),
			Class("h-96"),
			hx.Get("/search?q=&view="+view),
			hx.Trigger("load"),
			hx.Target("this"),
			hx.Swap("outerHTML"),
		),
	)
}

func SearchWidget(userID int, view string, query string) g.Node {
	return Div(
		Class("flex items-start gap-4"),
		renderNewAdButton(userID),
		Div(
			Class("flex-1 flex flex-col gap-4 relative"),
			Form(
				ID("searchForm"),
				Class("w-full"),
				hx.Get("/search"),
				hx.Target("#searchResults"),
				hx.Swap("outerHTML"),
				hx.Include("[name='view']"),
				Input(Type("hidden"), Name("view"), Value(view), ID("view-type-input")),
				Div(
					Class("flex gap-2 items-center"),
					Input(
						Type("search"),
						ID("searchBox"),
						Name("q"),
						Value(query),
						hx.Trigger("search"),
						Class("flex-1 p-2 border rounded"),
						Placeholder("Search by make, year, model, or description..."),
					),
					Button(
						Type("button"),
						Class("px-4 py-2 border border-blue-500 bg-white text-blue-500 rounded-full hover:bg-blue-50 whitespace-nowrap"),
						hx.Get("/filters/toggle"),
						hx.Target("#filtersArea"),
						hx.Swap("outerHTML"),
						hx.Vals("js:{show: document.getElementById('filtersArea').innerHTML.trim() === ''}"),
						g.Text("Filters"),
					),
				),
				emptyFiltersArea(),
			),
		),
	)
}

func createInfiniteScrollTrigger(loaderURL string) g.Node {
	return g.If(loaderURL != "", Div(
		Class("h-4"),
		g.Attr("hx-get", loaderURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
	))
}

// SearchCreateLoaderURL creates the loader URL for pagination
func SearchCreateLoaderURL(userPrompt, nextCursor, view string, threshold float64) string {
	if nextCursor == "" {
		return ""
	}

	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view), threshold)
}

// Helper function to render new ad button based on user login
func renderNewAdButton(userID int) g.Node {
	if userID != 0 {
		return styledLink("New Ad", "/new-ad", buttonPrimary)
	}
	return styledLinkDisabled("New Ad", buttonPrimary)
}

func emptyFiltersArea() g.Node {
	return Div(
		ID("filtersArea"),
	)
}

// FiltersToggleResponse renders the filters section that can be shown/hidden
func FiltersToggleResponse(showFilters bool) g.Node {
	if !showFilters {
		return emptyFiltersArea()
	}

	return Div(
		ID("filtersArea"),
		Class("bg-gray-50 border border-gray-200 rounded-lg p-4 my-4"),
		Div(
			Class("grid grid-cols-1 md:grid-cols-3 gap-4"),
			// Make filter
			Div(
				Label(Class("block text-sm font-medium text-gray-700 mb-1"), g.Text("Make")),
				Select(
					Name("make"),
					ID("makeFilter"),
					Class("w-full p-2 border border-gray-300 rounded-md"),
					Option(Value(""), g.Text("All Makes")),
					// TODO: Add dynamic makes from API
				),
			),
			// Year filter
			Div(
				Label(Class("block text-sm font-medium text-gray-700 mb-1"), g.Text("Year")),
				Select(
					Name("year"),
					ID("yearFilter"),
					Class("w-full p-2 border border-gray-300 rounded-md"),
					Option(Value(""), g.Text("All Years")),
					// TODO: Add dynamic years from API
				),
			),
			// Model filter
			Div(
				Label(Class("block text-sm font-medium text-gray-700 mb-1"), g.Text("Model")),
				Select(
					Name("model"),
					ID("modelFilter"),
					Class("w-full p-2 border border-gray-300 rounded-md"),
					Option(Value(""), g.Text("All Models")),
					// TODO: Add dynamic models from API
				),
			),
		),
		Div(
			Class("flex justify-end gap-2 mt-4"),
			Button(
				Type("button"),
				Class("px-4 py-2 text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50"),
				hx.On("click", "document.getElementById('makeFilter').value = ''; document.getElementById('yearFilter').value = ''; document.getElementById('modelFilter').value = ''; htmx.trigger('#searchForm', 'submit')"),
				g.Text("Clear Filters"),
			),
			Button(
				Type("submit"),
				Class("px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"),
				g.Text("Apply Filters"),
			),
		),
	)
}
