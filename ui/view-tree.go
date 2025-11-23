package ui

import (
	"encoding/base64"
	"encoding/binary"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/logger"
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
		viewContent = treeBrowseFirstField(adCat)
	} else {
		// Either has search query or has filtered results - show filtered tree
		if len(adIDs) > 0 {
			viewContent = treeSearchFirstField(adIDs, adCat)
		}
	}

	return Div(
		ID("searchResults"),
		ViewToggleButtons("tree", userID, csrfToken),
		viewContent,
	)
}

// treeBrowseFirstField returns the initial tree view for browsing
// Uses the first field in the first chain for the ad category
func treeBrowseFirstField(adCat int) g.Node {
	// Get first field from first chain
	chains := ad.GetFieldChains(adCat)
	if len(chains) == 0 || len(chains[0]) == 0 {
		logger.Error("No field chains found for category",
			"adCat", adCat, "component", "tree-view")
		return Div(Class("text-red-500"), g.Text("Error loading tree"))
	}

	firstField := chains[0][0]
	values, err := firstField.GetValues(map[string]string{})
	if err != nil {
		logger.Error("Error getting field values",
			"error", err, "field", firstField.Name(),
			"component", "tree-view")
		return Div(Class("text-red-500"), g.Text("Error loading tree"))
	}

	return createBrowseFieldNodes(values, firstField.Name())
}

// treeSearchFirstField returns the initial tree view for search results
// Uses the first field in the first chain for the ad category
func treeSearchFirstField(adIDs []int, adCat int) g.Node {
	// Get first field from first chain
	chains := ad.GetFieldChains(adCat)
	if len(chains) == 0 || len(chains[0]) == 0 {
		logger.Error("No field chains found for category",
			"adCat", adCat, "component", "tree-search")
		return Div(Class("text-red-500"), g.Text("Error loading tree"))
	}

	firstField := chains[0][0]
	values, err := ad.GetValuesForAdIDs(firstField, adIDs)
	if err != nil {
		logger.Error("Error getting field values for ad IDs",
			"error", err, "field", firstField.Name(),
			"component", "tree-search")
		return Div(Class("text-red-500"), g.Text("Error loading tree"))
	}

	return createSearchFieldNodes(values, firstField.Name(), adIDs)
}

func createBrowseFieldNodes(values []string, fieldName string) g.Node {
	if len(values) == 0 {
		return Div(Class("text-gray-500"), g.Text("No values available"))
	}

	var nodes []g.Node
	for _, value := range values {
		nodes = append(nodes, CollapsedTreeNodeBrowse(value, value))
	}

	return Div(
		Class("tree-contianer"),
		g.Group(nodes),
	)
}

func createSearchFieldNodes(values []string, fieldName string, adIDs []int) g.Node {
	if len(values) == 0 {
		return Div(Class("text-gray-500"), g.Text("No values available"))
	}

	var nodes []g.Node
	for _, value := range values {
		nodes = append(nodes, CollapsedTreeNodeSearch(value, value))
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
