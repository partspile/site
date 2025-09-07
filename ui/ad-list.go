package ui

import (
	"fmt"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

// AdListNode renders a list view of ad
func AdListNode(ad ad.Ad, loc *time.Location, userID int) g.Node {
	return Div(
		ID(adID(ad)),
		Class("flex items-center py-2 px-3 hover:bg-gray-50 cursor-pointer"),

		hx.Get(fmt.Sprintf("/ad/detail/%d?view=list", ad.ID)),
		hx.Target(adTarget(ad)),
		hx.Swap("outerHTML"),

		g.If(userID != 0, BookmarkButton(ad)),

		Div(
			Class("flex-1 text-blue-600 hover:text-blue-800"),
			titleNode(ad),
		),
		Div(
			Class("mr-4 text-xs text-gray-500"),
			locationFlagNode(ad),
		),
		Div(
			Class("mr-4 text-xs text-gray-400"),
			ageNode(ad, loc),
		),
		Div(
			Class("text-green-600 font-semibold mr-4"),
			priceNode(ad),
		),
	)
}
