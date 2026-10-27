package ads

import (
	"fmt"
	"strconv"

	"github.com/partspile/site/internal/facet"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// SearchFiltersPanel renders facet controls for the search widget.
func SearchFiltersPanel(facets []facet.Def, f SearchFilters) g.Node {
	return Div(
		Class("grid grid-cols-2 gap-4"),
		ID("search-filters"),
		hx.Get("/api/search/"),
		hx.Target("#search-results"),
		hx.Swap("outerHTML"),
		hx.Include("#search-widget"),
		// Scope to descendant filter controls via event bubbling. Avoid the
		// document-wide "from:input" selector so typing in the search box does
		// not trigger a search here (it searches on enter/blur instead).
		hx.Trigger("input delay:500ms, change delay:300ms"),
		g.Group(facetFilterNodes(facets, f)),
	)
}

func facetFilterNodes(facets []facet.Def, f SearchFilters) []g.Node {
	var nodes []g.Node
	var flagRun []facet.Def
	flushFlags := func() {
		if len(flagRun) == 0 {
			return
		}
		nodes = append(nodes, flagFilterGroup(flagRun, f))
		flagRun = nil
	}
	for _, d := range facets {
		if d.Filter == facet.FilterFlag {
			flagRun = append(flagRun, d)
			continue
		}
		flushFlags()
		nodes = append(nodes, facetFilterRow(d, f.Facets[d.Key]))
	}
	flushFlags()
	return nodes
}

func facetFilterRow(d facet.Def, filter facet.Filter) g.Node {
	switch d.Filter {
	case facet.FilterExact:
		return enumFilterRow(d, filter)
	case facet.FilterCheckboxes:
		return enumCheckboxesFilterRow(d, filter)
	case facet.FilterFlag:
		return flagFilterRow(d, filter)
	default:
		if d.Kind == facet.Date {
			return dateRangeFilterRow(d.Key, d.Label, filter.TextMin, filter.TextMax)
		}
		return rangeFilterRow(d.Key, d.Label, filter.Min, filter.Max)
	}
}

func enumFilterRow(d facet.Def, filter facet.Filter) g.Node {
	selected := ""
	if filter.Value != nil {
		selected = *filter.Value
	}
	id := "filter-" + d.Key
	opts := make([]g.Node, 0, len(d.Enum)+1)
	opts = append(opts, enumOption("", "Any", selected))
	for _, e := range d.Enum {
		opts = append(opts, enumOption(e, e, selected))
	}
	return Div(
		Class("col-span-2 field-group"),
		Label(For(id), Class("field-label"), g.Text(d.Label)),
		Select(
			Name(d.Key),
			ID(id),
			Class("w-full "+controlClass),
			g.Group(opts),
		),
	)
}

func enumCheckboxesFilterRow(d facet.Def, filter facet.Filter) g.Node {
	selected := make(map[string]bool, len(filter.Values))
	for _, v := range filter.Values {
		selected[v] = true
	}
	nodes := make([]g.Node, len(d.Enum))
	for i, e := range d.Enum {
		id := fmt.Sprintf("filter-%s-%d", d.Key, i)
		attrs := []g.Node{
			Type("checkbox"),
			Name(d.Key),
			Value(e),
			ID(id),
		}
		if selected[e] {
			attrs = append(attrs, g.Attr("checked", "checked"))
		}
		nodes[i] = Label(
			Class("field-option"),
			Input(attrs...),
			g.Text(e),
		)
	}
	return Div(
		Class("col-span-2 field-group"),
		Span(Class("field-label"), g.Text(d.Label)),
		Div(Class("field-options"), g.Group(nodes)),
	)
}

func flagFilterRow(d facet.Def, filter facet.Filter) g.Node {
	id := "filter-" + d.Key
	checked := filter.Value != nil && *filter.Value == "1"
	attrs := []g.Node{
		Type("checkbox"),
		Name(d.Key),
		Value("1"),
		ID(id),
	}
	if checked {
		attrs = append(attrs, g.Attr("checked", "checked"))
	}
	return Label(
		Class("field-option"),
		Input(attrs...),
		g.Text(d.Label),
	)
}

func flagFilterGroup(defs []facet.Def, f SearchFilters) g.Node {
	items := make([]g.Node, len(defs))
	for i, d := range defs {
		items[i] = flagFilterRow(d, f.Facets[d.Key])
	}
	return Div(
		Class("col-span-2 field-group"),
		Div(Class("field-options flex-col items-start gap-2"), g.Group(items)),
	)
}

func enumOption(value, label, selected string) g.Node {
	if value == selected {
		return Option(Value(value), g.Attr("selected", "selected"), g.Text(label))
	}
	return Option(Value(value), g.Text(label))
}

func rangeFilterRow(name, label string, min, max *int) g.Node {
	minID := "filter-" + name + "-min"
	maxID := "filter-" + name + "-max"
	return Div(
		Class("col-span-2 field-group"),
		Label(For(minID), Class("field-label"), g.Text(label)),
		Div(Class("ad-filter-price-range flex flex-wrap items-center gap-2"),
			Input(
				Type("number"),
				Name(name+"_min"),
				ID(minID),
				Class("w-36 "+controlClass),
				g.Attr("placeholder", "Min"),
				g.Attr("min", "0"),
				g.Attr("inputmode", "numeric"),
				g.If(priceAmount(min) != "", Value(priceAmount(min))),
			),
			Span(Class("ad-filter-range-sep"), g.Text("–")),
			Input(
				Type("number"),
				Name(name+"_max"),
				ID(maxID),
				Class("w-36 "+controlClass),
				g.Attr("placeholder", "Max"),
				g.Attr("min", "0"),
				g.Attr("inputmode", "numeric"),
				g.If(priceAmount(max) != "", Value(priceAmount(max))),
			),
		),
	)
}

func dateRangeFilterRow(name, label string, min, max *string) g.Node {
	minID := "filter-" + name + "-min"
	maxID := "filter-" + name + "-max"
	minVal := ""
	if min != nil {
		minVal = *min
	}
	maxVal := ""
	if max != nil {
		maxVal = *max
	}
	return Div(
		Class("col-span-2 field-group"),
		Label(For(minID), Class("field-label"), g.Text(label)),
		Div(Class("ad-filter-price-range flex flex-wrap items-center gap-2"),
			Input(
				Type("date"),
				Name(name+"_min"),
				ID(minID),
				Class(controlClass),
				g.If(minVal != "", Value(minVal)),
			),
			Span(Class("ad-filter-range-sep"), g.Text("–")),
			Input(
				Type("date"),
				Name(name+"_max"),
				ID(maxID),
				Class(controlClass),
				g.If(maxVal != "", Value(maxVal)),
			),
		),
	)
}

func priceAmount(p *int) string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(*p)
}
