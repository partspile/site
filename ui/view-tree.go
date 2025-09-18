package ui

import (
	"fmt"
	"log"
	"sort"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/vehicle"
)

func TreeViewResults(ads []ad.Ad, userID int, loc *time.Location, userPrompt string, nextCursor string, threshold float64) g.Node {
	var viewContent = NoSearchResultsMessage()

	if userPrompt == "" {
		viewContent = treeBrowseNodes(userID, loc)
	} else {
		if len(ads) > 0 {
			viewContent = treeSearchNodes(ads, userID, loc, userPrompt)
		}
	}

	return Div(
		ID("searchResults"),
		ViewToggleButtons("tree"),
		viewContent,
	)
}

func TreeViewContainer() g.Node {
	return Div(
		ID("tree-view"),
		hx.Get("/tree"),
		hx.Trigger("load"),
		hx.Swap("innerHTML"),
	)
}

func TreeViewWithQuery(query string) g.Node {
	return Div(
		ID("tree-view"),
		hx.Get("/tree?q="+query),
		hx.Trigger("load"),
		hx.Swap("innerHTML"),
	)
}

func TreeViewWithQueryAndThreshold(query string, threshold float64) g.Node {
	thresholdStr := fmt.Sprintf("%.1f", threshold)
	return Div(
		ID("tree-view"),
		hx.Get("/tree?q="+query+"&threshold="+thresholdStr),
		hx.Trigger("load"),
		hx.Swap("innerHTML"),
	)
}

// treeBrowseNodes returns the initial tree view for browsing (no search query)
func treeBrowseNodes(userID int, loc *time.Location) g.Node {
	// Get makes with existing ads (cached)
	makes, err := vehicle.GetAdMakes()
	if err != nil {
		log.Printf("[tree-view] Error getting makes: %v", err)
		return Div(Class("text-red-500"), g.Text("Error loading makes"))
	}

	if len(makes) == 0 {
		return Div(Class("text-gray-500"), g.Text("No makes available"))
	}

	// Create tree nodes for each make
	var nodes []g.Node
	for _, make := range makes {
		path := fmt.Sprintf("/%s", make)
		nodes = append(nodes, CollapsedTreeNode(make, path, "", 0))
	}

	return Div(
		Class("tree-container"),
		g.Group(nodes),
	)
}

// treeSearchNodes returns tree view nodes for search results
func treeSearchNodes(ads []ad.Ad, userID int, loc *time.Location, userPrompt string) g.Node {
	// Extract unique makes from the search results
	makeSet := make(map[string]bool)
	for _, ad := range ads {
		if ad.Make != "" {
			makeSet[ad.Make] = true
		}
	}

	if len(makeSet) == 0 {
		return Div(Class("text-gray-500"), g.Text("No makes found in search results"))
	}

	// Convert map to slice and sort
	var makes []string
	for make := range makeSet {
		makes = append(makes, make)
	}

	// Sort makes alphabetically
	sort.Strings(makes)

	// Create tree nodes for each make found in search results
	var nodes []g.Node
	for _, make := range makes {
		path := fmt.Sprintf("/%s", make)
		nodes = append(nodes, CollapsedTreeNode(make, path, userPrompt, 0))
	}

	return Div(
		Class("tree-search-results"),
		g.Group(nodes),
	)
}
