package ui

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/spec"
)

func filterControls(adCategory int, params map[string]string) g.Node {
	// Get category fields to determine which filters to show
	fieldConfigs, err := spec.GetCategoryFields(adCategory)
	if err != nil {
		logger.Warn("Failed to get category fields for filter UI",
			"error", err, "component", "filter-ui", "adCategory", adCategory)
		// Fallback to basic filters only
		return Div(
			Class("grid grid-cols-2 gap-4 mt-4"),
			locationFilter(params["location"]),
			radiusFilter(params["radius"]),
			priceFilter(params["min_price"], params["max_price"]),
		)
	}

	// Build filter nodes dynamically based on category fields
	var filterNodes []g.Node

	// Always show location and radius filters
	filterNodes = append(filterNodes, locationFilter(params["location"]))
	filterNodes = append(filterNodes, radiusFilter(params["radius"]))

	// Add spec field filters based on category configuration
	// Use chain information to determine filter type and dependencies
	for _, fieldConfig := range fieldConfigs {
		// Determine filter type based on field type
		if fieldConfig.FieldType == "int" || fieldConfig.FieldType == "float64" {
			// Range filter for numeric fields (year, etc.)
			if fieldConfig.Name == "year" {
				filterNodes = append(filterNodes, yearFilter(params["min_year"], params["max_year"]))
			} else {
				// Generic range filter for other numeric fields
				minKey := "min_" + fieldConfig.Name
				maxKey := "max_" + fieldConfig.Name
				filterNodes = append(filterNodes, genericRangeFilter(fieldConfig, params[minKey], params[maxKey]))
			}
		} else {
			// Dropdown filter for string fields
			// Check if field is in a chain - if so, make it dynamic based on parent selections
			if fieldConfig.NextInChain != 0 {
				// Chained field - use dynamic filter that updates based on parent selections
				filterNodes = append(filterNodes, chainedDropdownFilter(adCategory, fieldConfig, params))
			} else {
				// Non-chained field - use static dropdown
				filterNodes = append(filterNodes, genericDropdownFilter(fieldConfig, params[fieldConfig.Name]))
			}
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

func breedFilter(value string) g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text("Breed")),
		Select(
			Name("breed"),
			ID("breedFilter"),
			Class("w-full p-2 border rounded-md"),
			hx.Get("/api/filter-breeds"),
			hx.Trigger("load"),
			hx.Target("this"),
			hx.Swap("innerHTML"),
			Option(Value(""), g.Text("All Breeds"), g.If(value == "", Selected())),
			g.If(value != "", Option(Value(value), g.Text(value), Selected())),
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
	// Build clear script dynamically based on category fields
	clearScript := []string{
		"document.getElementById('searchBox').value = ''",
		"document.getElementById('locationFilter').value = ''",
		"document.getElementById('radiusFilter').value = '25'",
		"document.getElementById('minPriceFilter').value = ''",
		"document.getElementById('maxPriceFilter').value = ''",
	}

	// Get category fields to determine which filter fields to clear
	fieldConfigs, err := spec.GetCategoryFields(adCategory)
	if err == nil {
		for _, fieldConfig := range fieldConfigs {
			if fieldConfig.FieldType == "int" || fieldConfig.FieldType == "float64" {
				// Range fields - clear min/max
				minKey := "min_" + fieldConfig.Name + "Filter"
				maxKey := "max_" + fieldConfig.Name + "Filter"
				clearScript = append(clearScript,
					fmt.Sprintf("document.getElementById('%s').value = ''", minKey),
					fmt.Sprintf("document.getElementById('%s').value = ''", maxKey))
			} else {
				// String fields - clear single value
				filterID := fieldConfig.Name + "Filter"
				clearScript = append(clearScript, fmt.Sprintf("document.getElementById('%s').value = ''", filterID))
			}
		}
	}

	clearScript = append(clearScript, "htmx.trigger('#searchWidget', 'submit')")

	return buttonSecondary("Clear Filters",
		withClass("px-4 py-2"),
		withAttributes(
			hx.On("click", strings.Join(clearScript, "; ")),
		),
	)
}

func applyFilters() g.Node {
	return button("Apply Filters", withType("submit"), withClass("px-4 py-2"))
}

// chainedDropdownFilter creates a dynamic dropdown filter that updates based on parent field selections
func chainedDropdownFilter(adCategory int, fieldConfig spec.FieldConfig, params map[string]string) g.Node {
	value := params[fieldConfig.Name]

	// Build query string with parent field values (only fields that come before this one in the chain)
	queryParts := []string{}
	fieldConfigs, err := spec.GetCategoryFields(adCategory)
	if err == nil {
		// Find current field index
		currentIndex := -1
		for i, f := range fieldConfigs {
			if f.Name == fieldConfig.Name {
				currentIndex = i
				break
			}
		}

		if currentIndex >= 0 {
			// Add parent field values to query
			for i := 0; i < currentIndex; i++ {
				parentField := fieldConfigs[i]
				if parentField.FieldType == "int" || parentField.FieldType == "float64" {
					// Range fields - add min/max
					minKey := "min_" + parentField.Name
					maxKey := "max_" + parentField.Name
					if minVal := params[minKey]; minVal != "" {
						queryParts = append(queryParts, fmt.Sprintf("%s=%s", minKey, minVal))
					}
					if maxVal := params[maxKey]; maxVal != "" {
						queryParts = append(queryParts, fmt.Sprintf("%s=%s", maxKey, maxVal))
					}
				} else {
					// String fields - add single value
					if val := params[parentField.Name]; val != "" {
						queryParts = append(queryParts, fmt.Sprintf("%s=%s", parentField.Name, val))
					}
				}
			}
		}
	}

	queryStr := ""
	if len(queryParts) > 0 {
		queryStr = "?" + strings.Join(queryParts, "&")
	}

	// Use generic handler
	endpoint := fmt.Sprintf("/api/filter-field/%s", fieldConfig.Name)

	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(fieldConfig.DisplayName)),
		Select(
			Name(fieldConfig.Name),
			ID(fieldConfig.Name+"Filter"),
			Class("w-full p-2 border rounded-md"),
			hx.Get(endpoint+queryStr),
			hx.Trigger("load, change from:#searchWidget"),
			hx.Target("this"),
			hx.Swap("innerHTML"),
			hx.Include("#searchWidget"),
			Option(Value(""), g.Text(fmt.Sprintf("All %s", fieldConfig.DisplayName)), g.If(value == "", Selected())),
			g.If(value != "", Option(Value(value), g.Text(value), Selected())),
		),
	)
}

// genericRangeFilter creates a range filter for numeric fields
func genericRangeFilter(fieldConfig spec.FieldConfig, minValue, maxValue string) g.Node {
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
func genericDropdownFilter(fieldConfig spec.FieldConfig, value string) g.Node {
	return Div(
		Label(Class("block text-sm font-medium mb-1"), g.Text(fieldConfig.DisplayName)),
		Select(
			Name(fieldConfig.Name),
			ID(fieldConfig.Name+"Filter"),
			Class("w-full p-2 border rounded-md"),
			hx.Get(fmt.Sprintf("/api/filter-%s", fieldConfig.Name)),
			hx.Trigger("load"),
			hx.Target("this"),
			hx.Swap("innerHTML"),
			Option(Value(""), g.Text(fmt.Sprintf("All %s", fieldConfig.DisplayName)), g.If(value == "", Selected())),
			g.If(value != "", Option(Value(value), g.Text(value), Selected())),
		),
	)
}
