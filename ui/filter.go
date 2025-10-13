package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// filtersButton creates the Filters button
func filtersButton() g.Node {
	return Button(
		Type("button"),
		Class("px-4 py-2 border border-blue-500 bg-white text-blue-500 rounded-full hover:bg-blue-50 whitespace-nowrap"),
		hx.Get("/filters/show"),
		hx.Target("#searchForm"),
		hx.Swap("outerHTML"),
		hx.Vals("js:{q: document.getElementById('searchBox').value, view: document.getElementById('view-type-input').value}"),
		g.Text("Filters"),
	)
}

// locationFilter creates the location filter input
func locationFilter() g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Location")),
		Input(
			Type("text"),
			Name("location"),
			ID("locationFilter"),
			Class("w-full p-2 border rounded-md"),
			Placeholder("City, State or ZIP"),
		),
	)
}

// radiusFilter creates the radius filter select
func radiusFilter() g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Radius")),
		Select(
			Name("radius"),
			ID("radiusFilter"),
			Class("w-full p-2 border rounded-md"),
			Option(Value("25"), g.Text("25 miles"), Selected()),
			Option(Value("50"), g.Text("50 miles")),
			Option(Value("100"), g.Text("100 miles")),
			Option(Value("250"), g.Text("250 miles")),
			Option(Value("500"), g.Text("500 miles")),
		),
	)
}

// makeFilter creates the make filter select
func makeFilter() g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Make")),
		Select(
			Name("make"),
			ID("makeFilter"),
			Class("w-full p-2 border rounded-md"),
			hx.Get("/api/filter-makes"),
			hx.Trigger("load"),
			hx.Target("this"),
			hx.Swap("innerHTML"),
			Option(Value(""), g.Text("All Makes")),
		),
	)
}

// yearFilter creates the year range filter inputs
func yearFilter() g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Year Range")),
		Div(
			Class("flex gap-2"),
			Input(
				Type("number"),
				Name("min_year"),
				ID("minYearFilter"),
				Class("w-32 p-2 border rounded-md"),
				Placeholder("Min"),
				Min("1900"),
				Max("2030"),
			),
			Input(
				Type("number"),
				Name("max_year"),
				ID("maxYearFilter"),
				Class("w-32 p-2 border rounded-md"),
				Placeholder("Max"),
				Min("1900"),
				Max("2030"),
			),
		),
	)
}

// priceFilter creates the price range filter inputs
func priceFilter() g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Price Range")),
		Div(
			Class("flex gap-2"),
			Input(
				Type("number"),
				Name("min_price"),
				ID("minPriceFilter"),
				Class("w-32 p-2 border rounded-md"),
				Placeholder("Min $"),
				Min("0"),
				Step("0.01"),
			),
			Input(
				Type("number"),
				Name("max_price"),
				ID("maxPriceFilter"),
				Class("w-32 p-2 border rounded-md"),
				Placeholder("Max $"),
				Min("0"),
				Step("0.01"),
			),
		),
	)
}

// clearFilters creates the Clear Filters button
func clearFilters() g.Node {
	return Button(
		Type("button"),
		Class("px-4 py-2 text-gray-600 border border-gray-300 rounded-md hover:bg-gray-50"),
		hx.On("click", "document.getElementById('searchBox').value = ''; document.getElementById('locationFilter').value = ''; document.getElementById('radiusFilter').value = '25'; document.getElementById('makeFilter').value = ''; document.getElementById('minYearFilter').value = ''; document.getElementById('maxYearFilter').value = ''; document.getElementById('minPriceFilter').value = ''; document.getElementById('maxPriceFilter').value = ''; htmx.trigger('#searchForm', 'submit')"),
		g.Text("Clear Filters"),
	)
}

// applyFilters creates the Apply Filters button
func applyFilters() g.Node {
	return Button(
		Type("submit"),
		Class("px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"),
		g.Text("Apply Filters"),
	)
}

// FiltersShow renders the search form with filters
func FiltersShow(view string, query string) g.Node {
	// Return search form with filters panel
	return searchForm(view, query, Div(
		Class("border rounded-lg p-4"),
		searchBox(query),
		Div(
			Class("grid grid-cols-1 md:grid-cols-3 gap-4 mt-4"),
			locationFilter(),
			radiusFilter(),
			makeFilter(),
		),
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 gap-4 mt-4"),
			yearFilter(),
			priceFilter(),
		),
		Div(
			Class("flex justify-end gap-2 mt-4"),
			clearFilters(),
			applyFilters(),
		),
	))
}
