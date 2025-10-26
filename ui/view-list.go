package ui

import (
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func ListViewResults(ads []ad.Ad, userID int, userName string, loc *time.Location, loaderURL string) g.Node {
	var viewContent = NoSearchResultsMessage()

	if len(ads) > 0 {
		nodes := listNodes(ads, userID, loc)
		viewContent = Div(
			ID("list-view"),
			g.Group(append(nodes,
				createInfiniteScrollTrigger(loaderURL))),
		)
	}

	return Div(
		ID("searchResults"),
		ViewToggleButtons("list", userID),
		viewContent,
	)
}

func ListViewPage(ads []ad.Ad, userID int, userName string, loc *time.Location, loaderURL string) g.Node {
	nodes := listNodes(ads, userID, loc)
	return g.Group(append(nodes, createInfiniteScrollTrigger(loaderURL)))
}

func listNodes(ads []ad.Ad, userID int, loc *time.Location) []g.Node {
	nodes := make([]g.Node, 0, len(ads)*2) // *2 because we'll add separators between ads
	for _, ad := range ads {
		nodes = append(nodes, AdListNode(ad, userID, loc))
		// Add separator after each ad
		nodes = append(nodes, Div(
			Class("border-b border-gray-200"),
		))
	}
	return nodes
}
