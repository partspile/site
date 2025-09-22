package ui

import (
	"encoding/base64"
	"encoding/binary"
	"log"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/vehicle"
)

// encodeAdIDs encodes a slice of integers to a base64 string
func encodeAdIDs(adIDs []int) string {
	if len(adIDs) == 0 {
		return ""
	}

	// Convert to bytes using binary encoding
	buf := make([]byte, len(adIDs)*4) // 4 bytes per int32
	for i, id := range adIDs {
		binary.LittleEndian.PutUint32(buf[i*4:], uint32(id))
	}

	// Encode to base64
	return base64.URLEncoding.EncodeToString(buf)
}

func TreeViewResults(adIDs []int, userPrompt string) g.Node {
	var viewContent = NoSearchResultsMessage()

	if userPrompt == "" {
		viewContent = treeBrowseMakes()
	} else {
		if len(adIDs) > 0 {
			viewContent = treeSearchMakes(adIDs)
		}
	}

	return Div(
		ID("searchResults"),
		ViewToggleButtons("tree"),
		viewContent,
	)
}

// treeBrowseMakes returns the initial tree view for browsing
func treeBrowseMakes() g.Node {
	makes, err := vehicle.GetAdMakes()
	if err != nil {
		log.Printf("[tree-view] Error getting makes: %v", err)
		return Div(Class("text-red-500"), g.Text("Error loading makes"))
	}

	return createBrowseMakeNodes(makes)
}

// treeSearchMakes returns the initial tree view for search results
func treeSearchMakes(adIDs []int) g.Node {
	makes, err := vehicle.GetAdMakesForAdIDs(adIDs)
	if err != nil {
		log.Printf("[tree-search] Error getting makes for ad IDs: %v", err)
		return Div(Class("text-red-500"), g.Text("Error loading makes"))
	}

	return createSearchMakeNodes(makes, adIDs)
}

func createBrowseMakeNodes(makes []string) g.Node {
	if len(makes) == 0 {
		return Div(Class("text-gray-500"), g.Text("No makes available"))
	}

	var nodes []g.Node
	for _, make := range makes {
		nodes = append(nodes, CollapsedTreeNodeBrowse(make, make))
	}

	return Div(
		Class("tree-contianer"),
		g.Group(nodes),
	)
}

func createSearchMakeNodes(makes []string, adIDs []int) g.Node {
	if len(makes) == 0 {
		return Div(Class("text-gray-500"), g.Text("No makes available"))
	}

	var nodes []g.Node
	for _, make := range makes {
		nodes = append(nodes, CollapsedTreeNodeSearch(make, 0, make))
	}

	// Convert adIDs to base64 encoded string for DOM storage
	adIDsStr := encodeAdIDs(adIDs)

	return Div(
		Class("tree-contianer"),
		g.Group(nodes),
		// Hidden input to store adIDs for HTMX requests
		Input(Type("hidden"), Name("adIDs"), Value(adIDsStr)),
	)
}
