package ui

import (
	"encoding/json"
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

type SearchSchema ad.SearchQuery

func ViewToggleButtons(activeView string) g.Node {
	icon := func(name, alt string) g.Node {
		return Img(
			Src(name+".svg"),
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
			hx.Include("[name='q'],[name='structured_query'],[name='view']"),
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

func SearchResultsContainerWithFlags(newAdButton g.Node, filters SearchSchema, ads []ad.Ad, _ interface{}, userID int, loc *time.Location, view string, query string, loaderURL string) g.Node {
	return SearchResultsContainer(newAdButton, ads, userID, loc, view, query, loaderURL)
}

func SearchResultsContainer(newAdButton g.Node, ads []ad.Ad, userID int, loc *time.Location, view string, query string, loaderURL string) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(newAdButton, view, query),
		ViewToggleButtons(view),
		createViewWithInfiniteScroll(ads, userID, loc, view, query, loaderURL),
	)
}

func createViewWithInfiniteScroll(ads []ad.Ad, userID int, loc *time.Location, view string, query string, loaderURL string) g.Node {
	var viewContent g.Node

	// Create the appropriate view
	switch view {
	case "tree":
		structuredQueryJSON, _ := json.Marshal(SearchSchema{})
		viewContent = TreeViewWithQuery(query, string(structuredQueryJSON))
	case "grid":
		if loaderURL != "" {
			viewContent = GridViewWithTrigger(ads, loc, userID, loaderURL)
		} else {
			viewContent = GridView(ads, loc, userID)
		}
	case "map":
		adsMap := make(map[int]ad.Ad, len(ads))
		for _, ad := range ads {
			adsMap[ad.ID] = ad
		}
		viewContent = MapView(adsMap, loc)
	default: // list
		viewContent = ListViewFromSlice(ads, userID, loc)
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
		Class("flex items-center justify-center py-4 bg-blue-100 text-blue-600 border"),
		g.Attr("hx-get", loaderURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
		g.Text("Loading more..."),
	)
}

func ListViewFromSlice(ads []ad.Ad, userID int, loc *time.Location) g.Node {
	adNodes := buildAdListNodesFromSlice(ads, userID, loc)

	return Div(
		ID("list-view"),
		AdCompactListContainer(
			g.Group(adNodes),
		),
	)
}

func buildAdListNodesFromSlice(ads []ad.Ad, userID int, loc *time.Location) []g.Node {
	nodes := make([]g.Node, 0, len(ads)*2) // *2 because we'll add separators between ads
	for _, ad := range ads {
		nodes = append(nodes, AdCardCompactList(ad, loc, ad.Bookmarked, userID))

		// Add separator after each ad
		nodes = append(nodes, Div(
			Class("border-b border-gray-200"),
		))
	}
	return nodes
}

func GridView(ads []ad.Ad, loc *time.Location, userID int) g.Node {
	// Preserve original order from backend (Qdrant order)
	adNodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, ad.Bookmarked, userID, "grid"),
		)
	}

	return Div(
		ID("grid-view"),
		Class("grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"),
		g.Group(adNodes),
	)
}

func GridViewWithTrigger(ads []ad.Ad, loc *time.Location, userID int, loaderURL string) g.Node {
	// Preserve original order from backend (Qdrant order)
	adNodes := make([]g.Node, 0, len(ads)+1) // +1 for trigger
	for _, ad := range ads {
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, ad.Bookmarked, userID, "grid"),
		)
	}

	// Add the trigger as a grid item
	trigger := Div(
		Class("border rounded-lg shadow-sm bg-blue-100 flex items-center justify-center cursor-pointer hover:shadow-md transition-shadow"),
		g.Attr("hx-get", loaderURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
		Div(
			Class("p-4 text-center text-blue-600"),
			g.Text("Loading more..."),
		),
	)
	adNodes = append(adNodes, trigger)

	return Div(
		ID("grid-view"),
		Class("grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"),
		g.Group(adNodes),
	)
}

func MapView(ads map[int]ad.Ad, loc *time.Location) g.Node {
	// Placeholder: show a message or static image
	return Div(
		ID("map-view"),
		Class("flex items-center justify-center h-64 bg-gray-100 rounded"),
		g.Text("Map view coming soon!"),
	)
}
