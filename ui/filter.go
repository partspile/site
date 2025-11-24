package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func filterControls(adCategory int, params map[string]string) g.Node {
	fields := ad.GetCategoryFields(adCategory)

	var filterNodes []g.Node

	for _, field := range fields {
		filterNode := field.RenderFilterField(params)
		filterNodes = append(filterNodes, filterNode)
	}

	return Div(
		Class("grid grid-cols-2 gap-4 mt-4"),
		g.Group(filterNodes),
	)
}

func filterActions(adCategory int) g.Node {
	return Div(
		Class("flex justify-end gap-2 mt-4"),
		clearFilters(adCategory),
		applyFilters(),
	)
}

func filtersButton() g.Node {
	return Button(
		Type("button"),
		Class("px-4 py-2 border border-blue-500 bg-white text-blue-500 rounded-full hover:bg-blue-50 whitespace-nowrap"),
		hx.Get("/search-widget?show-filters=true"),
		hx.Target("#searchWidget"),
		hx.Swap("outerHTML"),
		hx.Include("form"),
		g.Text("Filters"),
	)
}

func clearFilters(adCategory int) g.Node {
	return buttonSecondary("Clear Filters", withType("button"), withClass("px-4 py-2"),
		withAttributes(
			hx.On("click", "document.getElementById('searchWidget').reset();"),
			hx.Get("/search"),
		),
	)
}

func applyFilters() g.Node {
	return button("Apply Filters", withType("submit"), withClass("px-4 py-2"))
}
