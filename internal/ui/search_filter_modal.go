package ui

import (
	"github.com/partspile/site/internal/facet"
	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// SearchFilterModal renders the popup editor for one search facet.
func SearchFilterModal(d facet.Def, filter facet.Filter) g.Node {
	name := uiads.FilterModalName(d.Key)
	formID := name + "-form"
	return g.Group([]g.Node{
		modalBackdrop(name),
		Div(
			ID(name+"-modal"),
			Class("fixed inset-0 flex items-center justify-center z-50 p-8 "+
				"pointer-events-none"),
			Div(
				Class("bg-white dark:bg-zinc-800 rounded-lg w-full shadow-2xl "+
					"border-2 border-zinc-300 dark:border-zinc-600 flex "+
					"flex-col pointer-events-auto"),
				Style("max-width: 400px"),
				Div(
					Class("flex items-center justify-between p-6 border-b "+
						"border-zinc-200 dark:border-zinc-700 flex-shrink-0"),
					H3(Class("text-xl font-bold text-zinc-900 dark:text-zinc-200"),
						g.Text(d.Label)),
					modalClose(name),
				),
				Div(
					Class("p-6 space-y-4 text-zinc-900 dark:text-zinc-200"),
					Form(
						ID(formID),
						Class("space-y-6"),
						g.Attr("onsubmit",
							"event.preventDefault(); return false;"),
						uiads.SearchFilterModalBody(d, filter),
						Div(
							Class("flex justify-end gap-2 pt-2"),
							Button(
								Type("button"),
								Class("py-2 px-4 border border-zinc-300 "+
									"dark:border-zinc-600 rounded-md "+
									"bg-white dark:bg-zinc-700 "+
									"text-zinc-900 dark:text-zinc-200 "+
									"hover:bg-zinc-100 dark:hover:bg-zinc-600 "+
									"cursor-pointer"),
								hx.Get("/api/modal-remove/"+name),
								hx.Swap("none"),
								g.Text("Cancel"),
							),
							Button(
								Type("button"),
								Class("py-2 px-4 border border-zinc-300 "+
									"dark:border-zinc-600 rounded-md "+
									"bg-white dark:bg-zinc-700 "+
									"text-zinc-900 dark:text-zinc-200 "+
									"hover:bg-zinc-100 dark:hover:bg-zinc-600 "+
									"cursor-pointer"),
								// Omit the form so save clears this facet.
								hx.Get("/api/filter-modal/"+d.Key+"/save"),
								hx.Include("#searchBox"),
								hx.Swap("none"),
								g.Text("Clear"),
							),
							Button(
								Type("button"),
								Class("py-2 px-4 bg-blue-600 text-white "+
									"rounded-md hover:bg-blue-700"),
								hx.Get("/api/filter-modal/"+d.Key+"/save"),
								hx.Include("#"+formID+", #searchBox"),
								hx.Swap("none"),
								g.Text("Apply"),
							),
						),
					),
				),
			),
		),
	})
}
