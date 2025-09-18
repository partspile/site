package ui

import (
	"fmt"
	"log"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/vehicle"
)

func TreeViewResults(adIDs []int, userPrompt string) g.Node {
	var viewContent = NoSearchResultsMessage()

	if userPrompt == "" {
		viewContent = treeBrowseNodes()
	} else {
		if len(adIDs) > 0 {
			viewContent = treeSearchNodes(adIDs, userPrompt)
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

// treeBrowseNodes returns the initial tree view for browsing
func treeBrowseNodes() g.Node {
	makes, err := vehicle.GetAdMakes()
	if err != nil {
		log.Printf("[tree-view] Error getting makes: %v", err)
		return Div(Class("text-red-500"), g.Text("Error loading makes"))
	}

	return createTreeNodes(makes, "")
}

// treeSearchNodes returns the initial tree view for search results
func treeSearchNodes(adIDs []int, userPrompt string) g.Node {
	makes, err := part.GetMakesForAdIDs(adIDs)
	if err != nil {
		log.Printf("[tree-search] Error getting makes for ad IDs: %v", err)
		return Div(Class("text-red-500"), g.Text("Error loading makes"))
	}

	return createTreeNodes(makes, userPrompt)
}

func createTreeNodes(makes []string, userPrompt string) g.Node {
	if len(makes) == 0 {
		return Div(Class("text-gray-500"), g.Text("No makes available"))
	}

	var nodes []g.Node
	for _, make := range makes {
		path := fmt.Sprintf("/%s", make)
		nodes = append(nodes, CollapsedTreeNode(make, path, userPrompt, 0))
	}

	return Div(
		Class("tree-contianer"),
		g.Group(nodes),
	)
}
