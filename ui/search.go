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

func ListView(ads map[int]ad.Ad, loc *time.Location) g.Node {
	return Div(
		ID("list-view"),
		AdListContainer(
			g.Group(BuildAdListNodesWithView(ads, loc, "list")),
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

func SearchResultsContainerWithFlags(newAdButton g.Node, filters SearchSchema, ads []ad.Ad, _ interface{}, userID int, loc *time.Location, view string, query string) g.Node {
	structuredQueryJSON, _ := json.Marshal(filters)
	adsMap := make(map[int]ad.Ad, len(ads))
	for _, ad := range ads {
		adsMap[ad.ID] = ad
	}
	return Div(
		ID("searchResults"),
		SearchWidget(newAdButton, view, query),
		Input(
			Type("hidden"),
			Name("structured_query"),
			Value(string(structuredQueryJSON)),
		),
		Div(
			ID("searchFilters"),
			Class("flex flex-wrap gap-4 mb-4"),
			SearchFilters(filters),
		),
		ViewToggleButtons(view),
		Div(
			ID("view-wrapper"),
			func() g.Node {
				switch view {
				case "tree":
					return TreeViewWithQuery(query, string(structuredQueryJSON))
				case "grid":
					return GridView(ads, loc, userID)
				case "map":
					return MapView(adsMap, loc)
				default:
					return ListViewWithFlags(ads, userID, loc)
				}
			}(),
		),
	)
}

func ListViewWithFlags(ads []ad.Ad, userID int, loc *time.Location) g.Node {
	return Div(
		ID("list-view"),
		AdListContainer(
			g.Group(BuildAdListNodesWithBookmarks(ads, userID, loc)),
		),
	)
}

func BuildAdListNodesWithBookmarks(ads []ad.Ad, userID int, loc *time.Location) []g.Node {
	nodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		nodes = append(nodes, AdCardExpandable(ad, loc, ad.Bookmarked, userID))
	}
	return nodes
}

func BuildAdListNodesWithView(ads map[int]ad.Ad, loc *time.Location, view string) []g.Node {
	nodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		nodes = append(nodes, AdCardExpandable(ad, loc, ad.Bookmarked, 0, view))
	}
	return nodes
}

func GridView(ads []ad.Ad, loc *time.Location, userID ...int) g.Node {
	// Preserve original order from backend (Pinecone order)
	adNodes := make([]g.Node, 0, len(ads))
	uid := 0
	if len(userID) > 0 {
		uid = userID[0]
	}
	for _, ad := range ads {
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, ad.Bookmarked, uid, "grid"),
		)
	}
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
