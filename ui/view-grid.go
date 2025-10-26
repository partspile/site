package ui

import (
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func GridViewResults(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	var viewContent = NoSearchResultsMessage()

	if len(ads) > 0 {
		nodes := gridNodes(ads, userID, loc)
		viewContent = Div(
			ID("grid-view"),
			Class("grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"),
			g.Group(append(nodes,
				createInfiniteScrollTrigger(loaderURL))),
		)
	}

	return Div(
		ID("searchResults"),
		ViewToggleButtons("grid", userID),
		viewContent,
	)
}

func GridViewPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	nodes := gridNodes(ads, userID, loc)
	return g.Group(append(nodes, createInfiniteScrollTrigger(loaderURL)))
}

func gridNodes(ads []ad.Ad, userID int, loc *time.Location) []g.Node {
	nodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		nodes = append(nodes, AdGridNode(ad, userID, loc))
	}
	return nodes
}
