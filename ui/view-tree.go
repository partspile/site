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

func TreeViewRenderResults(ads []ad.Ad, userID int, loc *time.Location, query string, loaderURL string, threshold float64) g.Node {
	// Create the main search results container
	var viewContent g.Node

	if len(ads) == 0 {
		// Show empty state
		viewContent = NoSearchResultsMessage()
	} else {
		// Show tree view
		structuredQueryJSON, _ := json.Marshal(SearchSchema{})
		viewContent = TreeViewWithQueryAndThreshold(query, string(structuredQueryJSON), threshold)
	}

	return Div(
		ID("searchResults"),
		SearchWidget(userID, "tree", query, threshold),
		ViewToggleButtons("tree"),
		viewContent,
	)
}

func TreeViewRenderPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	// For tree view pagination, we would typically add new nodes to the existing tree
	// Since tree view is loaded via HTMX, we'll return a trigger to reload the tree
	// with the new data

	var nodes []g.Node

	// Add infinite scroll trigger if there are more results
	if loaderURL != "" {
		trigger := Div(
			Class("h-4"),
			g.Attr("hx-get", loaderURL),
			g.Attr("hx-trigger", "revealed"),
			g.Attr("hx-swap", "outerHTML"),
		)
		nodes = append(nodes, trigger)
	}

	return g.Group(nodes)
}

func TreeViewContainer() g.Node {
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

// View-specific loader URL creation function
func TreeViewCreateLoaderURL(userPrompt, nextCursor string, threshold float64) string {
	if nextCursor == "" {
		return ""
	}
	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=tree&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)
}
