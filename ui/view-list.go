package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

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
	// For pagination, render just the ads and infinite scroll trigger
	adNodes := buildAdListNodesFromSlice(ads, userID, loc)

	return g.Group(append(adNodes, createInfiniteScrollTrigger(loaderURL)))
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

func ListViewFromMap(ads map[int]ad.Ad, loc *time.Location) g.Node {
	return Div(
		ID("list-view"),
		AdCompactListContainer(
			g.Group(BuildAdListNodes(ads, loc)),
		),
	)
}

// View-specific loader URL creation function
func CreateListViewLoaderURL(userPrompt, nextCursor string, threshold float64) string {
	if nextCursor == "" {
		return ""
	}
	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=list&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)
}
