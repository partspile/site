package ui

import (
	"strconv"

	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// SearchLocationModal renders the modal to edit location and within distance.
func SearchLocationModal(f uiads.SearchFilters) g.Node {
	unit := f.WithinUnit
	if unit == "" {
		unit = "mi"
	}
	suffix := " mi"
	if unit == "km" {
		suffix = " km"
	}
	return g.Group([]g.Node{
		modalBackdrop("search-location"),
		Div(
			ID("search-location-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-8 pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full shadow-2xl border-2 border-zinc-300 dark:border-zinc-600 flex flex-col pointer-events-auto"),
				Style("max-width: 400px"),
				Div(
					Class("flex items-center justify-between p-6 border-b border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					H3(Class("text-xl font-bold text-zinc-900 dark:text-zinc-200"), g.Text("Search location")),
					modalClose("search-location"),
				),
				Div(
					Class("p-6 space-y-4 text-zinc-900 dark:text-zinc-200"),
					Form(
						ID("search-location-form"),
						Class("space-y-8"),
						g.Attr("onsubmit", "event.preventDefault(); return false;"),
						modalLocationField(f.Location),
						modalWithinField(f.Within, f.WithinOptions, suffix),
						Div(
							Class("flex justify-end gap-2 pt-2"),
							Button(
								Type("button"),
								Class("py-2 px-4 border border-zinc-300 dark:border-zinc-600 rounded-md "+
									"bg-white dark:bg-zinc-700 text-zinc-900 dark:text-zinc-200 "+
									"hover:bg-zinc-100 dark:hover:bg-zinc-600 cursor-pointer"),
								hx.Get("/api/modal-remove/search-location"),
								hx.Swap("none"),
								g.Text("Cancel"),
							),
							Button(
								Type("button"),
								Class("py-2 px-4 bg-blue-600 text-white rounded-md hover:bg-blue-700"),
								hx.Get("/api/search-location/save"),
								// Include the active filter facets so saving a
								// location preserves them; the save handler
								// re-parses facets from the request when expanded.
								hx.Include("#search-location-form, #searchBox, #search-filters"),
								hx.Swap("none"),
								g.Text("Save"),
							),
						),
					),
				),
			),
		),
	})
}

func modalLocationField(value string) g.Node {
	return labeledTextInput("Location", "modal-search-location",
		Type("text"),
		Name("location"),
		g.Attr("placeholder", "City, State or ZIP"),
		g.If(value != "", Value(value)),
	)
}

func modalWithinField(selected int, options []int, suffix string) g.Node {
	if selected == 0 {
		selected = 25
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
	return labeledSelect("Within", "modal-search-within",
		Name("within"),
		g.Group(opts),
	)
}
