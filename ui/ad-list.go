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
func AdListNode(adObj ad.Ad, userID int, loc *time.Location) g.Node {
	// Determine styling based on deleted status
	var classes string
	if adObj.IsArchived() {
		classes = "flex items-center py-2 px-3 bg-red-100 cursor-pointer border border-red-300 rounded-lg my-2 mx-2"
	} else {
		classes = "flex items-center py-2 px-3 hover:bg-gray-50 cursor-pointer"
	}

	return Div(
		ID(adID(adObj)),
		Class(classes),

		hx.Get(fmt.Sprintf("/ad/detail/%d?view=list", adObj.ID)),
		hx.Target(adTarget(adObj)),
		hx.Swap("outerHTML show:bottom"),

		g.If(userID != 0, BookmarkButton(adObj)),

		Div(
			Class("flex-1 text-blue-600 hover:text-blue-800"),
			titleNode(adObj),
		),
		Div(
			Class("mr-4 text-xs text-gray-500"),
			location(adObj),
		),
		Div(
			Class("mr-4 text-xs text-gray-400"),
			ageNode(adObj, loc),
		),
		Div(
			Class("text-green-600 font-semibold mr-4"),
			priceNode(adObj),
		),
	)
}
