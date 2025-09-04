package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

func RenderGridViewEmpty(query string, threshold float64, userID int) g.Node {
	return Div(
		ID("searchResults"),
		SearchWidget(userID, "grid", query, threshold),
		ViewToggleButtons("grid"),
		NoSearchResultsMessage(),
	)
}

func RenderGridViewResults(ads []ad.Ad, userID int, loc *time.Location, query string, loaderURL string, threshold float64) g.Node {
	var viewContent g.Node
	if loaderURL != "" {
		viewContent = GridViewWithTrigger(ads, loc, userID, loaderURL)
	} else {
		viewContent = GridView(ads, loc, userID)
	}

	return Div(
		ID("searchResults"),
		SearchWidget(userID, "grid", query, threshold),
		ViewToggleButtons("grid"),
		viewContent,
	)
}

func RenderGridViewPage(ads []ad.Ad, userID int, loc *time.Location, loaderURL string) g.Node {
	// For pagination, render just the ads and infinite scroll trigger
	adNodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		// Create minimal user object for compatibility
		var currentUser *user.User
		if userID != 0 {
			currentUser = &user.User{ID: userID}
		}
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, currentUser, "grid"),
		)
	}

	// Add infinite scroll trigger if there are more results
	if loaderURL != "" {
		trigger := Div(
			Class("h-4"),
			g.Attr("hx-get", loaderURL),
			g.Attr("hx-trigger", "revealed"),
			g.Attr("hx-swap", "outerHTML"),
		)
		adNodes = append(adNodes, trigger)
	}

	return g.Group(adNodes)
}

func GridView(ads []ad.Ad, loc *time.Location, userID int) g.Node {
	// Preserve original order from backend (Qdrant order)
	adNodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		// Create minimal user object for compatibility
		var currentUser *user.User
		if userID != 0 {
			currentUser = &user.User{ID: userID}
		}
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, currentUser, "grid"),
		)
	}

	return Div(
		ID("grid-view"),
		Class("grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"),
		g.Group(adNodes),
	)
}

func GridViewWithTrigger(ads []ad.Ad, loc *time.Location, userID int, loaderURL string) g.Node {
	// Preserve original order from backend (Qdrant order)
	adNodes := make([]g.Node, 0, len(ads)+1) // +1 for trigger
	for _, ad := range ads {
		// Create minimal user object for compatibility
		var currentUser *user.User
		if userID != 0 {
			currentUser = &user.User{ID: userID}
		}
		adNodes = append(adNodes,
			AdCardExpandable(ad, loc, currentUser, "grid"),
		)
	}

	// Add the trigger as a grid item
	trigger := Div(
		Class("h-4"),
		g.Attr("hx-get", loaderURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
	)
	adNodes = append(adNodes, trigger)

	return Div(
		ID("grid-view"),
		Class("grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4"),
		g.Group(adNodes),
	)
}

// View-specific loader URL creation function
func CreateGridViewLoaderURL(userPrompt, nextCursor string, threshold float64) string {
	if nextCursor == "" {
		return ""
	}
	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=grid&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), threshold)
}
