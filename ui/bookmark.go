package ui

import (
	"time"

	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
)

func BuildAdListNodesFromSlice(currentUser *user.User, ads []ad.Ad) []g.Node {
	loc := time.Local
	nodes := make([]g.Node, 0, len(ads))
	for _, ad := range ads {
		nodes = append(nodes, AdListNode(ad, loc, currentUser.ID))
	}
	return nodes
}

func BookmarksPage(ads []ad.Ad, currentUser *user.User, path string, loc *time.Location) g.Node {
	var viewContent g.Node

	if len(ads) == 0 {
		viewContent = Div(Class("text-center py-12"),
			Div(Class("text-gray-500 text-lg mb-4"), g.Text("No bookmarked ads yet.")),
			Div(Class("text-gray-400 text-sm"), g.Text("Start browsing ads and bookmark the ones you're interested in!")),
		)
	} else {
		adNodes := adListNodes(ads, currentUser.ID, loc)
		viewContent = Div(
			g.Group(adNodes),
		)
	}

	return Page(
		"Bookmarked Ads",
		currentUser,
		path,
		[]g.Node{
			pageHeader("Bookmarked Ads"),
			Div(Class("text-gray-600 text-sm mb-6"), g.Text("Ads you have bookmarked for later.")),
			viewContent,
		},
	)
}
