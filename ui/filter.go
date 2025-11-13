package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/spec"
)

func filterControls(adCategory int, params map[string]string) g.Node {
	// Get category fields to determine which filters to show
	fields := spec.GetCategoryFields(adCategory)

	// Build filter nodes dynamically based on category fields
	var filterNodes []g.Node

	// Always show location and radius filters
	filterNodes = append(filterNodes, locationFilter(params["location"]))
	filterNodes = append(filterNodes, radiusFilter(params["radius"]))

	// Add spec field filters based on category configuration
	// Only show fields marked with ShowInFilters=true
	for _, field := range fields {
		fieldConfig := field.Config()
		if !fieldConfig.ShowInFilters {
			continue
		}

		// Determine filter type based on field type
		if fieldConfig.FieldType == "int" || fieldConfig.FieldType == "float64" {
			// Range filter for numeric fields (year, etc.)
			if fieldConfig.Name == "year" {
				filterNodes = append(filterNodes, yearFilter(params["min_year"], params["max_year"]))
			} else {
				// Generic range filter for other numeric fields
				minKey := "min_" + fieldConfig.Name
				maxKey := "max_" + fieldConfig.Name
				filterNodes = append(filterNodes, genericRangeFilter(field, params[minKey], params[maxKey]))
			}
		} else {
			// Dropdown filter for string fields - show all available values (not filtered by parents)
			filterNodes = append(filterNodes, genericDropdownFilter(adCategory, field, params[fieldConfig.Name], false))
		}
	}

	// Always show price filter
	filterNodes = append(filterNodes, priceFilter(params["min_price"], params["max_price"]))

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

// genericRangeFilter creates a range filter for numeric fields
func genericRangeFilter(field spec.Field, minValue, maxValue string) g.Node {
	fieldConfig := field.Config()
	minKey := "min_" + fieldConfig.Name
	maxKey := "max_" + fieldConfig.Name

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(fieldConfig.DisplayName+" Range")),
		Div(
			Class("flex gap-2 flex-nowrap"),
			Input(
				Type("number"),
				Name(minKey),
				ID(minKey+"Filter"),
				Class("w-24 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Min"),
				Value(minValue),
			),
			Input(
				Type("number"),
				Name(maxKey),
				ID(maxKey+"Filter"),
				Class("w-24 flex-shrink-0 p-2 border rounded-md"),
				Placeholder("Max"),
				Value(maxValue),
			),
		),
	)
}

// genericDropdownFilter creates a static dropdown filter for non-chained string fields
func genericDropdownFilter(adCategory int, field spec.Field, value string, hasChildren bool) g.Node {
	fieldConfig := field.Config()
	// Get all available values for this field (filter fields are independent)
	values, err := spec.GetFieldValues(adCategory, fieldConfig.Name, map[string]string{})
	if err != nil {
		logger.Warn("Failed to get field values for filter",
			"error", err, "component", "filter-ui", "adCategory", adCategory, "fieldName", fieldConfig.Name)
		values = []string{}
	}

	// Build options list
	var options []g.Node
	options = append(options, Option(Value(""), g.Text("All"), g.If(value == "", Selected())))
	for _, val := range values {
		optionAttrs := []g.Node{Value(val), g.Text(val)}
		if val == value {
			optionAttrs = append(optionAttrs, Selected())
		}
		options = append(options, Option(optionAttrs...))
	}

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(fieldConfig.DisplayName)),
		Select(
			Name(fieldConfig.Name),
			ID(fieldConfig.Name+"Filter"),
			Class("w-full p-2 border rounded-md"),
			g.Group(options),
		),
	)
}
