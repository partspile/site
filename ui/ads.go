package ui

import (
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

// AdsPage renders the main ads page with navigation tabs
func AdsPage(currentUser *user.User, path string, activeTab string) g.Node {
	return Page(
		"My Ads",
		currentUser,
		path,
		[]g.Node{
			pageHeader("My Ads"),
			Div(Class("text-gray-600 text-sm mb-6"), g.Text("Manage your ads and bookmarks.")),
			Div(
				ID("ads-navigation"),
				adsNavigation(activeTab),
			),
			Div(
				ID("ads-content"),
				Class("mt-6"),
				hx.Get("/ads/bookmarked"),
				hx.Trigger("load"),
				hx.Target("#ads-navigation"),
				hx.Swap("outerHTML"),
			),
		},
	)
}

// AdsPageWithContent renders the ads page with navigation and content for HTMX updates
func AdsPageWithContent(currentUser *user.User, path string, activeTab string, content g.Node) g.Node {
	return Div(
		ID("ads-navigation"),
		adsNavigation(activeTab),
		Div(
			ID("ads-content"),
			Class("mt-6"),
			content,
		),
	)
}

// adsNavigation renders the tab navigation for the ads page
func adsNavigation(activeTab string) g.Node {
	tabs := []struct {
		id    string
		label string
		href  string
	}{
		{"bookmarked", "Bookmarked", "/ads/bookmarked"},
		{"active", "Active", "/ads/active"},
		{"deleted", "Deleted", "/ads/deleted"},
	}

	var tabNodes []g.Node
	for _, tab := range tabs {
		var classes string
		if activeTab == tab.id {
			classes = "px-4 py-2 text-sm font-medium text-blue-600 border-b-2 border-blue-600"
		} else {
			classes = "px-4 py-2 text-sm font-medium text-gray-500 hover:text-gray-700 hover:border-gray-300 border-b-2 border-transparent"
		}

		tabNodes = append(tabNodes,
			A(
				Href(tab.href),
				Class(classes),
				hx.Get(tab.href),
				hx.Target("#ads-navigation"),
				hx.Swap("outerHTML"),
				g.Text(tab.label),
			),
		)
	}

	return Div(
		Class("border-b border-gray-200"),
		Nav(
			Class("flex space-x-8"),
			g.Group(tabNodes),
		),
	)
}

// BookmarkedAdsPage renders the bookmarked ads sub-page
func BookmarkedAdsPage(ads []ad.Ad, currentUser *user.User, path string, loc *time.Location) g.Node {
	var viewContent g.Node

	if len(ads) == 0 {
		viewContent = Div(Class("text-center py-12"),
			Div(Class("text-gray-500 text-lg mb-4"), g.Text("No bookmarked ads yet.")),
			Div(Class("text-gray-400 text-sm"), g.Text("Start browsing ads and bookmark the ones you're interested in!")),
		)
	} else {
		adNodes := listNodes(ads, currentUser.ID, loc)
		viewContent = Div(
			g.Group(adNodes),
		)
	}

	return Div(
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Bookmarked Ads")),
		Div(Class("text-gray-600 text-sm"), g.Text("Ads you have bookmarked for later.")),
		viewContent,
	)
}

// ActiveAdsPage renders the active ads sub-page
func ActiveAdsPage(ads []ad.Ad, currentUser *user.User, path string, loc *time.Location) g.Node {
	var viewContent g.Node

	if len(ads) == 0 {
		viewContent = Div(Class("text-center py-12"),
			Div(Class("text-gray-500 text-lg mb-4"), g.Text("No active ads yet.")),
			Div(Class("text-gray-400 text-sm"), g.Text("Create your first ad to get started!")),
			A(
				Href("/new-ad"),
				Class("mt-4 inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700"),
				g.Text("Create New Ad"),
			),
		)
	} else {
		adNodes := listNodes(ads, currentUser.ID, loc)
		viewContent = Div(
			g.Group(adNodes),
		)
	}

	return Div(
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Active Ads")),
		Div(Class("text-gray-600 text-sm"), g.Text("Your currently active ads.")),
		viewContent,
	)
}

// DeletedAdsPage renders the deleted ads sub-page
func DeletedAdsPage(ads []ad.Ad, currentUser *user.User, path string, loc *time.Location) g.Node {
	var viewContent g.Node

	if len(ads) == 0 {
		viewContent = Div(Class("text-center py-12"),
			Div(Class("text-gray-500 text-lg mb-4"), g.Text("No deleted ads.")),
			Div(Class("text-gray-400 text-sm"), g.Text("Deleted ads will appear here.")),
		)
	} else {
		adNodes := listNodes(ads, currentUser.ID, loc)
		viewContent = Div(
			g.Group(adNodes),
		)
	}

	return Div(
		Class("space-y-4"),
		Div(Class("text-lg font-medium text-gray-900"), g.Text("Deleted Ads")),
		Div(Class("text-gray-600 text-sm"), g.Text("Your deleted ads. These can be restored if needed.")),
		viewContent,
	)
}
