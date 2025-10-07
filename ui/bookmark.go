package ui

import (
	"time"

	g "maragu.dev/gomponents"

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
