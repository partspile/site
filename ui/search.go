package ui

import (
	"encoding/json"
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

type SearchSchema ad.SearchQuery

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
			hx.Indicator("#searchWaiting"),
			hx.Include("[name='q'],[name='structured_query'],[name='view'],[name='threshold']"),
			hx.Vals(fmt.Sprintf(`{"selected_view":"%s"}`, view)),
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

func ListViewFromMap(ads map[int]ad.Ad, loc *time.Location) g.Node {
	return Div(
		ID("list-view"),
		AdCompactListContainer(
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

func TreeViewWithQueryAndThreshold(query, structuredQuery string, threshold float64) g.Node {
	thresholdStr := fmt.Sprintf("%.1f", threshold)
	return Div(
		ID("tree-view"),
		hx.Get("/tree?q="+query+"&structured_query="+structuredQuery+"&threshold="+thresholdStr),
		hx.Trigger("load"),
		hx.Swap("innerHTML"),
	)
}

func InitialSearchResults(view string) g.Node {
	return Div(
		ID("searchResults"),
		Div(
			hx.Get("/search?q=&view="+view),
			hx.Trigger("load"),
			hx.Target("this"),
			hx.Swap("outerHTML"),
		),
	)
}

func SearchWidget(newAdButton g.Node, view string, query string, threshold float64) g.Node {
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
				// Add bounding box inputs for map view
				g.Group(boundingBoxInputs),
				Input(
					Type("search"),
					ID("searchBox"),
					Name("q"),
					Value(query),
					Class("w-full p-2 border rounded"),
					Placeholder("Search by make, year, model, or description..."),
					hx.Trigger("keyup changed delay:500ms, search"),
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
						hx.Indicator("#searchWaiting"),
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
		Div(
			ID("searchWaiting"),
			Class("htmx-indicator absolute inset-0 flex items-center justify-center bg-white bg-opacity-60 z-10 pointer-events-none"),
			Img(
				Src("/images/spinner.gif"),
				Alt("Loading..."),
				Class("w-12 h-12 pointer-events-auto"),
			),
		),
	)
}

func SearchResultsContainerWithFlags(newAdButton g.Node, filters SearchSchema, ads []ad.Ad, _ interface{}, currentUser *user.User, loc *time.Location, view string, query string, loaderURL string, threshold float64) g.Node {
	return SearchResultsContainer(newAdButton, ads, currentUser, loc, view, query, loaderURL, threshold)
}

func SearchResultsContainer(newAdButton g.Node, ads []ad.Ad, currentUser *user.User, loc *time.Location, view string, query string, loaderURL string, threshold float64) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(newAdButton, view, query, threshold),
		ViewToggleButtons(view),
		createViewWithInfiniteScroll(ads, currentUser, loc, view, query, loaderURL, threshold),
	)
}

func SearchResultsEmpty(viewType string, query string, threshold float64, newAdButton g.Node) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(newAdButton, viewType, query, threshold),
		ViewToggleButtons(viewType),
		Div(
			ID("view-wrapper"),
			NoSearchResultsMessage(),
		),
	)
}

func createViewWithInfiniteScroll(ads []ad.Ad, currentUser *user.User, loc *time.Location, view string, query string, loaderURL string, threshold float64) g.Node {
	var viewContent g.Node

	// Handle no results for list and grid views
	if len(ads) == 0 && (view == "list" || view == "grid") {
		viewContent = NoSearchResultsMessage()
	} else if view == "map" {
		// For map view, always show the map (even if empty)
		adsMap := make(map[int]ad.Ad, len(ads))
		for _, ad := range ads {
			adsMap[ad.ID] = ad
		}
		viewContent = MapView(adsMap, loc)
	} else {
		// Create the appropriate view
		switch view {
		case "tree":
			structuredQueryJSON, _ := json.Marshal(SearchSchema{})
			viewContent = TreeViewWithQueryAndThreshold(query, string(structuredQueryJSON), threshold)
		case "grid":
			if loaderURL != "" {
				viewContent = GridViewWithTrigger(ads, loc, currentUser, loaderURL)
			} else {
				viewContent = GridView(ads, loc, currentUser)
			}
		default: // list
			viewContent = ListViewFromSlice(ads, currentUser, loc)
		}
	}

	// Add infinite scroll trigger for list view only (grid has it built-in)
	if view == "list" && loaderURL != "" {
		return Div(
			ID("view-wrapper"),
			viewContent,
			createInfiniteScrollTrigger(loaderURL),
		)
	}

	return Div(
		ID("view-wrapper"),
		viewContent,
	)
}

func createInfiniteScrollTrigger(loaderURL string) g.Node {
	return Div(
		Class("h-4"),
		g.Attr("hx-get", loaderURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
	)
}

func ListViewFromSlice(ads []ad.Ad, currentUser *user.User, loc *time.Location) g.Node {
	adNodes := buildAdListNodesFromSlice(ads, currentUser, loc)

	return Div(
		ID("list-view"),
		AdCompactListContainer(
			g.Group(adNodes),
		),
	)
}

func buildAdListNodesFromSlice(ads []ad.Ad, currentUser *user.User, loc *time.Location) []g.Node {
	nodes := make([]g.Node, 0, len(ads)*2) // *2 because we'll add separators between ads
	for _, ad := range ads {
		nodes = append(nodes, AdCardCompactList(ad, loc, currentUser))

		// Add separator after each ad
		nodes = append(nodes, Div(
			Class("border-b border-gray-200"),
		))
	}
	return nodes
}

func GridView(ads []ad.Ad, loc *time.Location, currentUser *user.User) g.Node {
	// Preserve original order from backend (Qdrant order)
	adNodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, currentUser, "grid"),
		)
	}

	return Div(
		ID("grid-view"),
		Class("grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"),
		g.Group(adNodes),
	)
}

func GridViewWithTrigger(ads []ad.Ad, loc *time.Location, currentUser *user.User, loaderURL string) g.Node {
	// Preserve original order from backend (Qdrant order)
	adNodes := make([]g.Node, 0, len(ads)+1) // +1 for trigger
	for _, ad := range ads {
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, currentUser, "grid"),
		)
	}

	// Add the trigger as a grid item
	trigger := Div(
		Class("h-4"),
		g.Attr("hx-get", loaderURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
	)
	adNodes = append(adNodes, trigger)

	return Div(
		ID("grid-view"),
		Class("grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"),
		g.Group(adNodes),
	)
}

func MapView(ads map[int]ad.Ad, loc *time.Location) g.Node {
	// Create hidden data elements for each ad with coordinates
	var adDataElements []g.Node
	for _, ad := range ads {
		if ad.Latitude != nil && ad.Longitude != nil {
			adDataElements = append(adDataElements,
				Div(
					Class("hidden"),
					g.Attr("data-ad-id", fmt.Sprintf("%d", ad.ID)),
					g.Attr("data-lat", fmt.Sprintf("%f", *ad.Latitude)),
					g.Attr("data-lon", fmt.Sprintf("%f", *ad.Longitude)),
					g.Attr("data-title", ad.Title),
					g.Attr("data-price", fmt.Sprintf("%.2f", ad.Price)),
				),
			)
		}
	}

	return Div(
		ID("map-view"),
		Class("h-96 w-full rounded border bg-gray-50"),
		// Map container with explicit styling
		Div(
			ID("map-container"),
			Class("h-full w-full z-10"),
			Style("min-height: 384px; position: relative;"),
		),
		// Hidden inputs for bounding box
		Input(Type("hidden"), ID("min-lat"), Name("minLat")),
		Input(Type("hidden"), ID("max-lat"), Name("maxLat")),
		Input(Type("hidden"), ID("min-lon"), Name("minLon")),
		Input(Type("hidden"), ID("max-lon"), Name("maxLon")),
		// Hidden ad data elements
		g.Group(adDataElements),
	)
}
