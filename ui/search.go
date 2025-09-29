package ui

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// GeoBounds represents a geographic bounding box
type GeoBounds struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

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
		button("map", "Map View"),
	)
}

func InitialSearchResults(userID int, view string) g.Node {
	return Div(
		SearchWidget(userID, view, "", 0),
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

func SearchWidget(userID int, view string, query string, threshold float64) g.Node {
	thresholdStr := fmt.Sprintf("%.1f", threshold)

	// Create bounding box inputs for map view
	var boundingBoxInputs []g.Node
	if view == "map" {
		boundingBoxInputs = []g.Node{
			Input(Type("hidden"), Name("minLat"), ID("form-min-lat")),
			Input(Type("hidden"), Name("maxLat"), ID("form-max-lat")),
			Input(Type("hidden"), Name("minLon"), ID("form-min-lon")),
			Input(Type("hidden"), Name("maxLon"), ID("form-max-lon")),
		}
	}

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
				// Add bounding box inputs for map view
				g.Group(boundingBoxInputs),
				Input(
					Type("search"),
					ID("searchBox"),
					Name("q"),
					Value(query),
					Class("w-full p-2 border rounded"),
					Placeholder("Search by make, year, model, or description..."),
					hx.Trigger("search"),
				),
				// Threshold slider - only show when there's a search query
				g.If(query != "", Div(
					Class("flex items-center gap-2"),
					Input(
						Type("range"),
						ID("thresholdSlider"),
						Name("threshold"),
						Min("0.0"),
						Max("1.0"),
						Step("0.1"),
						Value(thresholdStr),
						Class("flex-1 h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer"),
						hx.Get("/search"),
						hx.Target("#searchResults"),
						hx.Swap("outerHTML"),
						hx.Include("closest form"),
						hx.Trigger("change"),
					),
					Span(
						ID("thresholdValue"),
						Class("text-sm text-gray-600 min-w-[3rem]"),
						g.Text(thresholdStr),
					),
				)),
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
func SearchCreateLoaderURL(userPrompt, nextCursor, view string, threshold float64, bounds *GeoBounds) string {
	if nextCursor == "" {
		return ""
	}

	loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view), threshold)

	// Add bounding box to loader URL for map view
	if view == "map" && bounds != nil {
		loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
			bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}

	return loaderURL
}

// Helper function to render new ad button based on user login
func renderNewAdButton(userID int) g.Node {
	if userID != 0 {
		return StyledLink("New Ad", "/new-ad", buttonPrimary)
	}
	return StyledLinkDisabled("New Ad", buttonPrimary)
}
