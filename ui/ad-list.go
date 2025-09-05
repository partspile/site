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
		ID(fmt.Sprintf("ad-%d", ad.ID)),
		Class("flex items-center py-2 px-3 hover:bg-gray-50 cursor-pointer"),

		hx.Get(fmt.Sprintf("/ad/detail/%d?view=list", ad.ID)),
		hx.Target(fmt.Sprintf("#ad-%d", ad.ID)),
		hx.Swap("outerHTML"),

		g.If(userID != 0, BookmarkButton(ad)),

		Div(
			Class("flex-1 text-blue-600 hover:text-blue-800"),
			g.Text(ad.Title),
		),
		Div(
			Class("mr-4"),
			LocationDisplayWithFlag(ad),
		),
		Div(
			Class("mr-4"),
			AgeDisplay(ad.CreatedAt.In(loc)),
		),
		Div(
			Class("text-green-600 font-semibold mr-4"),
			g.Text(fmt.Sprintf("$%.0f", ad.Price)),
		),
	)
}
