package ads

import (
	"fmt"
	"strconv"
	"strings"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

const defaultWithin = 25

// SearchLocationBar renders the location summary link and hidden form fields.
func SearchLocationBar(f SearchFilters) g.Node {
	return searchLocationDiv(f, false)
}

// SearchLocationOOB swaps #search-location out-of-band after the modal saves.
func SearchLocationOOB(f SearchFilters) g.Node {
	return searchLocationDiv(f, true)
}

func searchLocationDiv(f SearchFilters, oob bool) g.Node {
	attrs := []g.Node{
		ID("search-location"),
		Class("min-w-0 text-right"),
		searchLocationHiddenFields(f),
		locationSummaryLink(f),
	}
	if oob {
		attrs = append([]g.Node{hx.SwapOOB("outerHTML")}, attrs...)
	}
	return Div(attrs...)
}

// LocationSummaryText formats the location summary link label.
func LocationSummaryText(f SearchFilters) string {
	if strings.TrimSpace(f.Location) == "" {
		return "No location set"
	}
	within := f.Within
	if within == 0 {
		within = defaultWithin
	}
	unit := "miles"
	if f.WithinUnit == "km" {
		unit = "kilometers"
	}
	label := f.LocationDisplay
	if label == "" {
		label = f.Location
	}
	return fmt.Sprintf("%s - Within %d %s", label, within, unit)
}

func locationSummaryLink(f SearchFilters) g.Node {
	hasLocation := strings.TrimSpace(f.Location) != ""
	return Button(
		Type("button"),
		Class("inline-flex items-center justify-end gap-1 max-w-full "+
			"text-blue-600 dark:text-blue-400 "+
			"cursor-pointer bg-transparent border-0 p-0 text-right"),
		hx.Get("/api/search-location-modal"),
		hx.Target("body"),
		hx.Swap("beforeend"),
		g.If(hasLocation, locationPin()),
		Span(
			Class("hover:underline min-w-0"),
			g.Text(LocationSummaryText(f)),
		),
	)
}

func locationPin() g.Node {
	return Span(
		Class("location-pin"),
		g.Attr("aria-hidden", "true"),
	)
}

func searchLocationHiddenFields(f SearchFilters) g.Node {
	withinVal := withinHiddenValue(f)
	nodes := []g.Node{
		Input(Type("hidden"), Name("location"), Value(f.Location)),
	}
	if withinVal != "" {
		nodes = append(nodes, Input(Type("hidden"), Name("within"), Value(withinVal)))
	}
	return g.Group(nodes)
}

func withinHiddenValue(f SearchFilters) string {
	if strings.TrimSpace(f.Location) == "" {
		return ""
	}
	within := f.Within
	if within == 0 {
		within = defaultWithin
	}
	return strconv.Itoa(within)
}

// WithinSelect renders the within-distance dropdown.
func WithinSelect(id string, selected int, options []int, suffix string,
	inputClass ...string) g.Node {
	class := fieldInputClass
	if len(inputClass) > 0 {
		class = inputClass[0]
	}
	if selected == 0 {
		selected = defaultWithin
	}
	opts := make([]g.Node, 0, len(options))
	for _, n := range options {
		label := strconv.Itoa(n) + suffix
		opt := Option(Value(strconv.Itoa(n)), g.Text(label))
		if n == selected {
			opt = Option(Value(strconv.Itoa(n)), g.Attr("selected", "selected"), g.Text(label))
		}
		opts = append(opts, opt)
	}
	return Select(
		Name("within"),
		ID(id),
		Class(class),
		g.Group(opts),
	)
}
