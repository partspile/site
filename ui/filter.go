package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/logger"
)

func filterControls(adCategory int, params map[string]string) g.Node {
	// Get category fields to determine which filters to show
	fields := ad.GetCategoryFields(adCategory)

	// Build filter nodes dynamically based on category fields
	var filterNodes []g.Node

	// Add spec field filters based on category configuration
	// Only show fields marked with ShowInFilters=true
	for _, field := range fields {
		fieldConfig := field.Config()
		if !fieldConfig.ShowInFilters {
			continue
		}

		// Use RenderFilterField to get the filter node(s)
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

// genericRangeFilter creates a range filter for numeric fields
func genericRangeFilter(field ad.Field, minValue, maxValue string) g.Node {
	minKey := "min_" + field.Name()
	maxKey := "max_" + field.Name()

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(field.DisplayName()+" Range")),
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
func genericDropdownFilter(adCategory int, field ad.Field, value string, hasChildren bool) g.Node {
	// Get all available values for this field (filter fields are independent)
	values, err := field.GetValues(map[string]string{})
	if err != nil {
		logger.Warn("Failed to get field values for filter",
			"error", err, "component", "filter-ui", "adCategory", adCategory, "fieldName", field.Name())
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
		Label(Class("block text-sm font-medium mb-1"), g.Text(field.DisplayName())),
		Select(
			Name(field.Name()),
			ID(field.Name()+"Filter"),
			Class("w-full p-2 border rounded-md"),
			g.Group(options),
		),
	)
}
