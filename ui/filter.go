package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func filterControls(params map[string]string) g.Node {
	return Div(
		Class("grid grid-cols-1 md:grid-cols-3 gap-4 mt-4"),
		locationFilter(params["location"]),
		radiusFilter(params["radius"]),
		makeFilter(params["make"]),
		Div(
			Class("grid grid-cols-1 md:grid-cols-2 gap-4 mt-4"),
			yearFilter(params["min_year"], params["max_year"]),
			priceFilter(params["min_price"], params["max_price"]),
		),
	)
}

func filterActions() g.Node {
	return Div(
		Class("flex justify-end gap-2 mt-4"),
		clearFilters(),
		applyFilters(),
	)
}

func filtersButton() g.Node {
	return Button(
		Type("button"),
		Class("px-4 py-2 border border-blue-500 bg-white text-blue-500 rounded-full hover:bg-blue-50 whitespace-nowrap"),
		hx.Get("/search-widget/true"),
		hx.Target("#searchWidget"),
		hx.Swap("outerHTML"),
		hx.Include("form"),
		g.Text("Filters"),
	)
}

func locationFilter(value string) g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Location")),
		Input(
			Type("text"),
			Name("location"),
			ID("locationFilter"),
			Class("w-full p-2 border rounded-md"),
			Placeholder("City, State or ZIP"),
			Value(value),
		),
	)
}

func radiusFilter(value string) g.Node {
	// Default to "25" if no value provided
	if value == "" {
		value = "25"
	}

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Radius")),
		Select(
			Name("radius"),
			ID("radiusFilter"),
			Class("w-full p-2 border rounded-md"),
			Option(Value("25"), g.Text("25 miles"), g.If(value == "25", Selected())),
			Option(Value("50"), g.Text("50 miles"), g.If(value == "50", Selected())),
			Option(Value("100"), g.Text("100 miles"), g.If(value == "100", Selected())),
			Option(Value("250"), g.Text("250 miles"), g.If(value == "250", Selected())),
			Option(Value("500"), g.Text("500 miles"), g.If(value == "500", Selected())),
		),
	)
}

func makeFilter(value string) g.Node {
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
			Option(Value(""), g.Text("All Makes"), g.If(value == "", Selected())),
			g.If(value != "", Option(Value(value), g.Text(value), Selected())),
		),
	)
}

func yearFilter(minYear, maxYear string) g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Year Range")),
		Div(
			Class("flex gap-2 flex-nowrap"),
			Input(
				Type("number"),
				Name("min_year"),
				ID("minYearFilter"),
				Class("w-20 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Min"),
				Min("1900"),
				Max("2030"),
				Value(minYear),
			),
			Input(
				Type("number"),
				Name("max_year"),
				ID("maxYearFilter"),
				Class("w-20 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Max"),
				Min("1900"),
				Max("2030"),
				Value(maxYear),
			),
		),
	)
}

func priceFilter(minPrice, maxPrice string) g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Price Range")),
		Div(
			Class("flex gap-2 flex-nowrap"),
			Input(
				Type("number"),
				Name("min_price"),
				ID("minPriceFilter"),
				Class("w-24 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Min $"),
				Min("0"),
				Step("0.01"),
				Value(minPrice),
			),
			Input(
				Type("number"),
				Name("max_price"),
				ID("maxPriceFilter"),
				Class("w-24 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Max $"),
				Min("0"),
				Step("0.01"),
				Value(maxPrice),
			),
		),
	)
}

func clearFilters() g.Node {
	return buttonSecondary("Clear Filters",
		withClass("px-4 py-2"),
		withAttributes(
			hx.On("click", "document.getElementById('searchBox').value = ''; document.getElementById('locationFilter').value = ''; document.getElementById('radiusFilter').value = '25'; document.getElementById('makeFilter').value = ''; document.getElementById('minYearFilter').value = ''; document.getElementById('maxYearFilter').value = ''; document.getElementById('minPriceFilter').value = ''; document.getElementById('maxPriceFilter').value = ''; htmx.trigger('#searchWidget', 'submit')"),
		),
	)
}

func applyFilters() g.Node {
	return button("Apply Filters", withType("submit"), withClass("px-4 py-2"))
}
