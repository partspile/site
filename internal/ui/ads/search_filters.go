package ads

import (
	"fmt"
	"strconv"

	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/facet"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// SearchFilterPills renders always-visible filter pills under the search box.
// categoryControl is the shared icon category picker from package ui.
func SearchFilterPills(facets []facet.Def, f SearchFilters,
	categoryControl g.Node) g.Node {
	pills := []g.Node{categoryControl}
	for _, d := range facets {
		pills = append(pills, facetFilterPill(d, f.Facets[d.Key]))
	}
	return Div(
		Class("flex flex-wrap gap-2 items-center"),
		ID("search-filters"),
		g.Group(pills),
	)
}

func facetFilterPill(d facet.Def, filter facet.Filter) g.Node {
	switch d.Filter {
	case facet.FilterFlag:
		return flagFilterPill(d, filter)
	default:
		return facetEditorPill(d, filter)
	}
}

func facetEditorPill(d facet.Def, filter facet.Filter) g.Node {
	active := filter.Active()
	return Div(
		Class("contents"),
		facetHiddenFields(d, filter),
		Button(
			Type("button"),
			ID("filter-"+d.Key),
			Class(filterPillClass(active)),
			hx.Get("/api/filter-modal/"+d.Key),
			hx.Target("body"),
			hx.Swap("beforeend"),
			g.Attr("aria-label", "Edit "+d.Label+" filter"),
			g.Attr("title", "Edit "+d.Label+" filter"),
			g.Text(facetPillLabel(d, filter)),
		),
	)
}

func flagFilterPill(d facet.Def, filter facet.Filter) g.Node {
	checked := filter.Value != nil && *filter.Value == "1"
	inputAttrs := []g.Node{
		Type("checkbox"),
		Name(d.Key),
		Value("1"),
		ID("filter-" + d.Key),
		Class("shrink-0"),
		hx.Get("/api/search-filters"),
		hx.Include("#search-widget"),
		hx.Swap("none"),
		hx.Trigger("change"),
	}
	if checked {
		inputAttrs = append(inputAttrs, g.Attr("checked", "checked"))
	}
	return Label(
		Class(filterPillClass(checked)+" cursor-pointer gap-1.5"),
		g.Attr("title", d.Label),
		Input(inputAttrs...),
		Span(g.Text(d.Label)),
	)
}

func facetHiddenFields(d facet.Def, filter facet.Filter) g.Node {
	switch d.Filter {
	case facet.FilterCheckboxes:
		nodes := make([]g.Node, len(filter.Values))
		for i, v := range filter.Values {
			nodes[i] = Input(Type("hidden"), Name(d.Key), Value(v))
		}
		return g.Group(nodes)
	case facet.FilterFlag:
		return nil
	default:
		if priceFilterWanted(d, filter) {
			return Input(
				Type("hidden"),
				Name(d.Key+"_wanted"),
				Value("1"),
			)
		}
		var nodes []g.Node
		if filter.Min != nil {
			nodes = append(nodes, Input(
				Type("hidden"),
				Name(d.Key+"_min"),
				Value(strconv.Itoa(*filter.Min)),
			))
		}
		if filter.Max != nil {
			nodes = append(nodes, Input(
				Type("hidden"),
				Name(d.Key+"_max"),
				Value(strconv.Itoa(*filter.Max)),
			))
		}
		return g.Group(nodes)
	}
}

func facetPillLabel(d facet.Def, filter facet.Filter) string {
	switch d.Filter {
	case facet.FilterFlag:
		return d.Label
	case facet.FilterCheckboxes:
		if len(filter.Values) == 0 {
			return d.Label
		}
		if len(filter.Values) == 1 {
			return d.Label + ": " + filter.Values[0]
		}
		return fmt.Sprintf("%s: %s +%d",
			d.Label, filter.Values[0], len(filter.Values)-1)
	default:
		if priceFilterWanted(d, filter) {
			return "Wanted"
		}
		return rangePillLabel(d.Label, filter.Min, filter.Max)
	}
}

func priceFilterWanted(d facet.Def, filter facet.Filter) bool {
	return d.Kind == facet.Money && filter.Value != nil &&
		*filter.Value == currency.WantedFilter
}

func rangePillLabel(label string, min, max *int) string {
	switch {
	case min != nil && max != nil:
		return fmt.Sprintf("%s: %d-%d", label, *min, *max)
	case min != nil:
		return fmt.Sprintf("%s: %d+", label, *min)
	case max != nil:
		return fmt.Sprintf("%s: ≤%d", label, *max)
	default:
		return label
	}
}

func filterPillClass(active bool) string {
	base := "filter-pill inline-flex items-center max-w-full " +
		"px-2.5 py-1 rounded-full text-xs whitespace-nowrap"
	if active {
		return base + " filter-pill-active"
	}
	return base
}

// FilterModalName is the modal id prefix for a facet editor.
func FilterModalName(key string) string {
	return "filter-" + key
}

// SearchFilterModalBody is the editable form fields for a facet modal.
func SearchFilterModalBody(d facet.Def, filter facet.Filter) g.Node {
	switch d.Filter {
	case facet.FilterCheckboxes:
		return enumCheckboxesEditor(d, filter)
	default:
		if d.Kind == facet.Money {
			return priceFilterEditor(d.Key, filter)
		}
		return rangeEditor(d.Key, filter.Min, filter.Max)
	}
}

func priceFilterEditor(name string, filter facet.Filter) g.Node {
	wanted := filter.Value != nil && *filter.Value == currency.WantedFilter
	id := "filter-" + name + "-wanted"
	attrs := []g.Node{
		Type("checkbox"),
		Name(name + "_wanted"),
		Value("1"),
		ID(id),
	}
	if wanted {
		attrs = append(attrs, g.Attr("checked", "checked"))
	}
	var min, max *int
	if !wanted {
		min, max = filter.Min, filter.Max
	}
	return Div(
		Class("space-y-4"),
		Label(
			Class("field-option"),
			Input(attrs...),
			g.Text("Wanted"),
		),
		rangeEditor(name, min, max),
	)
}

func enumCheckboxesEditor(d facet.Def, filter facet.Filter) g.Node {
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
		Class("field-group"),
		Div(Class("field-options flex-col items-start gap-2"), g.Group(nodes)),
	)
}

func rangeEditor(name string, min, max *int) g.Node {
	minID := "filter-" + name + "-min"
	maxID := "filter-" + name + "-max"
	return Div(
		Class("field-group"),
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

func priceAmount(p *int) string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(*p)
}
