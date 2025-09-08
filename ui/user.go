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

func BookmarksPage(currentUser *user.User, ads []ad.Ad) g.Node {
	return Page(
		"Bookmarked Ads",
		currentUser,
		"/bookmarks",
		[]g.Node{
			pageHeader("Bookmarked Ads"),
			Div(Class("text-gray-600 text-sm mb-6"), g.Text("Ads you have bookmarked for later.")),
			g.If(len(ads) == 0,
				contentContainer(
					Div(Class("text-center py-12"),
						Div(Class("text-gray-500 text-lg mb-4"), g.Text("No bookmarked ads yet.")),
						Div(Class("text-gray-400 text-sm"), g.Text("Start browsing ads and bookmark the ones you're interested in!")),
					),
				),
			),
			g.If(len(ads) > 0,
				AdCompactListContainer(
					g.Group(BuildAdListNodesFromSlice(currentUser, ads)),
				),
			),
		},
	)
}
