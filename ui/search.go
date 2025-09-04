package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

type SearchSchema ad.SearchQuery

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
			hx.Indicator("#searchWaiting"),
			hx.Include("[name='q'],[name='threshold']"),
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

func SearchResultsContainer(userID int, ads []ad.Ad, currentUser *user.User, loc *time.Location, view string, query string, loaderURL string, threshold float64) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, view, query, threshold),
		ViewToggleButtons(view),
		createViewWithInfiniteScroll(ads, currentUser, loc, view, query, loaderURL, threshold),
	)
}

func SearchResultsEmpty(userID int, viewType string, query string, threshold float64) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, viewType, query, threshold),
		ViewToggleButtons(viewType),
		NoSearchResultsMessage(),
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
			userID := 0
			if currentUser != nil {
				userID = currentUser.ID
			}
			if loaderURL != "" {
				viewContent = GridViewWithTrigger(ads, loc, userID, loaderURL)
			} else {
				viewContent = GridView(ads, loc, userID)
			}
		default: // list
			userID := 0
			if currentUser != nil {
				userID = currentUser.ID
			}
			viewContent = ListViewContainer(ads, userID, loc, loaderURL)
		}
	}

	return viewContent
}

func createInfiniteScrollTrigger(loaderURL string) g.Node {
	return g.If(loaderURL != "", Div(
		Class("h-4"),
		g.Attr("hx-get", loaderURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
	))
}

func ListViewContainer(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	adNodes := buildAdListNodesFromSlice(ads, userID, loc)

	return Div(
		ID("list-view"),
		AdCompactListContainer(
			g.Group(adNodes),
		),
		createInfiniteScrollTrigger(loaderURL),
	)
}

func buildAdListNodesFromSlice(ads []ad.Ad, userID int, loc *time.Location) []g.Node {
	nodes := make([]g.Node, 0, len(ads)*2) // *2 because we'll add separators between ads
	for _, ad := range ads {
		// Create minimal user object for compatibility
		var currentUser *user.User
		if userID != 0 {
			currentUser = &user.User{ID: userID}
		}
		nodes = append(nodes, AdCardCompactList(ad, loc, currentUser))

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
		// Create minimal user object for compatibility
		var currentUser *user.User
		if userID != 0 {
			currentUser = &user.User{ID: userID}
		}
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

func GridViewWithTrigger(ads []ad.Ad, loc *time.Location, userID int, loaderURL string) g.Node {
	// Preserve original order from backend (Qdrant order)
	adNodes := make([]g.Node, 0, len(ads)+1) // +1 for trigger
	for _, ad := range ads {
		// Create minimal user object for compatibility
		var currentUser *user.User
		if userID != 0 {
			currentUser = &user.User{ID: userID}
		}
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

func RenderListViewEmpty(query string, threshold float64, userID int) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, "list", query, threshold),
		ViewToggleButtons("list"),
		NoSearchResultsMessage(),
	)
}

func RenderListViewResults(ads []ad.Ad, userID int, loc *time.Location, query string, loaderURL string, threshold float64) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, "list", query, threshold),
		ViewToggleButtons("list"),
		ListViewContainer(ads, userID, loc, loaderURL),
	)
}

func RenderListViewPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	// For pagination, we return the ads to be rendered by the handler
	// The handler will call the appropriate rendering functions
	return nil
}

// Grid view functions
func RenderGridViewEmpty(query string, threshold float64, userID int) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, "grid", query, threshold),
		ViewToggleButtons("grid"),
		NoSearchResultsMessage(),
	)
}

func RenderGridViewResults(ads []ad.Ad, userID int, loc *time.Location, query string, loaderURL string, threshold float64) g.Node {
	var viewContent g.Node
	if loaderURL != "" {
		viewContent = GridViewWithTrigger(ads, loc, userID, loaderURL)
	} else {
		viewContent = GridView(ads, loc, userID)
	}

	return Div(
		ID("searchResults"),
		SearchWidget(userID, "grid", query, threshold),
		ViewToggleButtons("grid"),
		viewContent,
	)
}

func RenderGridViewPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	return nil
}

// Map view functions
func RenderMapViewEmpty(query string, threshold float64, userID int) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, "map", query, threshold),
		ViewToggleButtons("map"),
		NoSearchResultsMessage(),
	)
}

func RenderMapViewResults(ads []ad.Ad, userID int, loc *time.Location, query string, loaderURL string, threshold float64) g.Node {
	// For map view, always show the map (even if empty)
	adsMap := make(map[int]ad.Ad, len(ads))
	for _, ad := range ads {
		adsMap[ad.ID] = ad
	}
	viewContent := MapView(adsMap, loc)

	return Div(
		ID("searchResults"),
		SearchWidget(userID, "map", query, threshold),
		ViewToggleButtons("map"),
		viewContent,
	)
}

func RenderMapViewPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	return nil
}

// Tree view functions
func RenderTreeViewEmpty(query string, threshold float64, userID int) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, "tree", query, threshold),
		ViewToggleButtons("tree"),
		NoSearchResultsMessage(),
	)
}

func RenderTreeViewResults(ads []ad.Ad, userID int, loc *time.Location, query string, loaderURL string, threshold float64) g.Node {
	structuredQueryJSON, _ := json.Marshal(SearchSchema{})
	viewContent := TreeViewWithQueryAndThreshold(query, string(structuredQueryJSON), threshold)

	return Div(
		ID("searchResults"),
		SearchWidget(userID, "tree", query, threshold),
		ViewToggleButtons("tree"),
		viewContent,
	)
}

func RenderTreeViewPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	return nil
}

// Helper function to render new ad button based on user login
func renderNewAdButton(userID int) g.Node {
	if userID != 0 {
		return StyledLink("New Ad", "/new-ad", ButtonPrimary)
	}
	return StyledLinkDisabled("New Ad", ButtonPrimary)
}

// View-specific loader URL creation functions
func CreateListViewLoaderURL(userPrompt, nextCursor string, threshold float64) string {
	if nextCursor == "" {
		return ""
	}
	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=list&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)
}

func CreateGridViewLoaderURL(userPrompt, nextCursor string, threshold float64) string {
	if nextCursor == "" {
		return ""
	}
	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=grid&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)
}

func CreateMapViewLoaderURL(userPrompt, nextCursor string, threshold float64, bounds *GeoBounds) string {
	if nextCursor == "" {
		return ""
	}
	loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s&view=map&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)

	// Add bounding box to loader URL for map view
	if bounds != nil {
		loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
			bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}
	return loaderURL
}

func CreateTreeViewLoaderURL(userPrompt, nextCursor string, threshold float64) string {
	if nextCursor == "" {
		return ""
	}
	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=tree&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)
}
