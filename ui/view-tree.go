package ui

import (
	"encoding/base64"
	"encoding/binary"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/spec"
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

func TreeViewResults(adIDs []int, adCat int, q string, userID int, csrfToken string) g.Node {
	var viewContent = NoSearchResultsMessage()

	if q == "" && len(adIDs) == 0 {
		// No search query and no filtered results - show full browse tree
		viewContent = treeBrowseMakes(adCat)
	} else {
		// Either has search query or has filtered results - show filtered tree
		if len(adIDs) > 0 {
			viewContent = treeSearchMakes(adIDs)
		}
	}

	return Div(
		ID("searchResults"),
		ViewToggleButtons("tree", userID, csrfToken),
		viewContent,
	)
}

// treeBrowseMakes returns the initial tree view for browsing
func treeBrowseMakes(adCat int) g.Node {
	makes, err := spec.GetAdMakes(adCat)
	if err != nil {
		logger.Error("Error getting makes",
			"error", err, "component", "tree-view")
		return Div(Class("text-red-500"), g.Text("Error loading makes"))
	}

	return createBrowseMakeNodes(makes)
}

// treeSearchMakes returns the initial tree view for search results
func treeSearchMakes(adIDs []int) g.Node {
	makes, err := spec.GetAdMakesForAdIDs(adIDs)
	if err != nil {
		logger.Error("Error getting makes for ad IDs",
			"error", err, "component", "tree-search")
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
		nodes = append(nodes, CollapsedTreeNodeSearch(make, make))
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
