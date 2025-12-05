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
func AdListNode(adObj ad.Ad, userID int, loc *time.Location, csrfToken string) g.Node {
	// Determine styling based on deleted status
	var classes string
	if adObj.IsArchived() {
		classes = "flex flex-wrap items-center justify-between py-2 px-3 bg-red-100 cursor-pointer border border-red-300 rounded-lg my-2 mx-2"
	} else {
		classes = "flex flex-wrap items-center justify-between py-2 px-3 hover:bg-gray-50 cursor-pointer"
	}

	return Div(
		ID(adID(adObj)),
		Class(classes),

		hx.Get(fmt.Sprintf("/ad/detail/%d?view=list", adObj.ID)),
		hx.Target(adTarget(adObj)),
		hx.Swap("outerHTML show:top"),
		//hx.Swap("outerHTML show:bottom"),

		Div(
			Class("text-blue-600 hover:text-blue-800"),
			g.If(userID != 0, BookmarkButton(adObj, csrfToken)),
			titleNode(adObj),
		),
		Div(
			Class("flex items-center gap-2 ml-auto"),
			Div(
				Class("flex items-center gap-2 text-xs text-gray-500"),
				ageNode(adObj, loc, false),
				location(adObj),
			),
			Div(
				Class("text-green-600 font-semibold"),
				priceNode(adObj),
			),
		),
	)
}
