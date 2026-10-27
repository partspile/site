package ui

import (
	"fmt"

	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/local"
	uiads "github.com/partspile/site/internal/ui/ads"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func HomePage(userID, view int, q string, filtersExpanded, searchVisible bool,
	category CategoryOption, categories []CategoryOption,
	filterFacets []facet.Def, filters uiads.SearchFilters,
	results []g.Node) []g.Node {
	return []g.Node{SearchContainer(userID, view, q, filtersExpanded,
		searchVisible, category, categories, filterFacets, filters, results)}
}

func SearchContainer(userID, view int, q string, filtersExpanded,
	searchVisible bool, category CategoryOption, categories []CategoryOption,
	filterFacets []facet.Def, filters uiads.SearchFilters,
	results []g.Node) g.Node {
	return Div(
		ID("search-container"),
		Div(
			Class("flex flex-col gap-4"),
			categorySearchRow(userID, category, categories, "/",
				searchVisible),
			SearchWidget(userID, view, q, filtersExpanded, searchVisible, category, filterFacets, filters, results),
		),
	)
}

func SearchResults(view int, results []g.Node) g.Node {
	return searchResults(view, results, false)
}

// SearchResultsResponse is the HTMX body for /api/search/.
// Page 1: #search-results with sorry message or first page of ads.
// Page 2+: ad nodes only (replaces the scroll sentinel).
func SearchResultsResponse(view, page int, results []g.Node) g.Node {
	if page > 1 {
		return g.Group(results)
	}
	return SearchResults(view, results)
}

// SearchResultsOOB swaps #search-results out-of-band (show/hide filters).
func SearchResultsOOB(view int, results []g.Node) g.Node {
	return searchResults(view, results, true)
}

// SearchArea renders the search box, filter toggle, and optional expanded facets.
func SearchArea(q string, expanded bool, filterFacets []facet.Def,
	filters uiads.SearchFilters, searchVisible bool) g.Node {
	return searchArea(q, expanded, filterFacets, filters, searchVisible, false)
}

// SearchAreaOOB swaps #search-area when the filter panel is shown or hidden.
func SearchAreaOOB(q string, expanded bool, filterFacets []facet.Def,
	filters uiads.SearchFilters, searchVisible bool) g.Node {
	return searchArea(q, expanded, filterFacets, filters, searchVisible, true)
}

func searchArea(q string, expanded bool, filterFacets []facet.Def,
	filters uiads.SearchFilters, searchVisible, oob bool) g.Node {
	searchBlockClass := "flex flex-col gap-2"
	if !searchVisible {
		searchBlockClass += " hidden"
	}
	searchBlock := Div(
		Class(searchBlockClass),
		Div(
			Class("flex gap-2 items-center"),
			Div(Class("flex-1 min-w-0"), searchBox(q)),
			filterToggle(expanded),
		),
	)

	children := []g.Node{searchBlock}
	areaClass := "flex flex-col"
	if !searchVisible && !expanded {
		areaClass += " hidden"
	} else if expanded {
		areaClass += " border rounded-lg p-4 gap-4"
		children = append(children, uiads.SearchFiltersPanel(filterFacets, filters))
	}

	attrs := []g.Node{
		ID("search-area"),
		Class(areaClass),
		g.Group(children),
	}
	if oob {
		attrs = append([]g.Node{hx.SwapOOB("outerHTML")}, attrs...)
	}
	return Div(attrs...)
}

func categorySearchRow(userID int, category CategoryOption,
	categories []CategoryOption, returnParam string, searchVisible bool) g.Node {
	return Div(
		Class("flex items-center gap-2 justify-between"),
		CategorySelect(category, categories, returnParam),
		Div(
			Class("flex items-center gap-2 shrink-0"),
			g.Iff(local.IsLoggedIn(userID), func() g.Node { return newAdLink() }),
			searchToggle(searchVisible, false),
		),
	)
}

func newAdLink() g.Node {
	const label = "New ad"
	return A(
		Href("/auth/ad/new"),
		Class("flex-shrink-0 cursor-pointer"),
		g.Attr("aria-label", label),
		g.Attr("title", label),
		Img(
			Class("w-6 h-6 dark:invert dark:opacity-80"),
			Src("/images/post_add.svg"),
			Alt(label),
		),
	)
}

func searchToggle(searchVisible, oob bool) g.Node {
	label := "Show search"
	class := "flex-shrink-0 cursor-pointer"
	if searchVisible {
		class += " hidden"
	}

	attrs := []g.Node{
		Type("button"),
		ID("search-toggle"),
		Class(class),
		g.Attr("aria-label", label),
		g.Attr("title", label),
		hx.Get("/api/toggle-search"),
		hx.Include("#search-widget"),
		hx.Swap("none"),
		Img(
			Class("w-6 h-6 dark:invert dark:opacity-80"),
			Src("/images/search.svg"),
			Alt(label),
		),
	}
	if oob {
		attrs = append([]g.Node{hx.SwapOOB("outerHTML")}, attrs...)
	}
	return Button(attrs...)
}

// SearchToggleOOB swaps #search-toggle out-of-band after show/hide search.
func SearchToggleOOB(searchVisible bool) g.Node {
	return searchToggle(searchVisible, true)
}

func searchBox(q string) g.Node {
	return Input(
		Class(textFieldClass),
		Type("search"),
		ID("searchBox"),
		Name("q"),
		Value(q),
		g.Attr("aria-label", "Search"),
		hx.Get("/api/search/"),
		hx.Target("#search-results"),
		hx.Swap("outerHTML"),
		hx.Include("#search-widget"),
		hx.Trigger("keyup[key=='Enter'], blur changed, input[!event.target.value.length]"),
		Placeholder("What are you looking for?"),
	)
}

func FilterToggle(expanded bool) g.Node {
	return filterToggle(expanded)
}

func filterToggle(expanded bool) g.Node {
	label := "Expand filters"
	icon := "/images/expand.svg"
	var actionAttrs []g.Node
	if expanded {
		label = "Collapse filters"
		icon = "/images/collapse.svg"
		actionAttrs = []g.Node{
			hx.Get("/api/hide-filters"),
			hx.Target("#search-area"),
			hx.Swap("outerHTML"),
			hx.Include("#search-widget"),
		}
	} else {
		actionAttrs = []g.Node{
			hx.Get("/api/show-filters"),
			hx.Target("#search-area"),
			hx.Swap("outerHTML"),
		}
	}
	attrs := []g.Node{
		Type("button"),
		ID("filter-toggle"),
		Class("p-2 border border-zinc-300 dark:border-zinc-600 rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800"),
		g.Attr("aria-label", label),
		g.Attr("title", label),
	}
	attrs = append(attrs, actionAttrs...)
	attrs = append(attrs, Img(
		Class("w-6 h-6 dark:invert dark:opacity-80"),
		Src(icon),
		Alt(label),
	))
	return Button(attrs...)
}

func SearchView(view int, filters uiads.SearchFilters, results []g.Node) g.Node {
	return Div(
		Class("flex flex-col gap-4"),
		ID("search-view"),
		viewRow(filters, view),
		searchResults(view, results, false),
	)
}

func viewRow(filters uiads.SearchFilters, view int) g.Node {
	return Div(
		Class("flex justify-between items-center gap-2"),
		uiads.SearchLocationBar(filters),
		viewToggles(view),
	)
}

func SearchWidget(userID, view int, q string, filtersExpanded,
	searchVisible bool, category CategoryOption, filterFacets []facet.Def,
	filters uiads.SearchFilters, results []g.Node) g.Node {
	attrs := []g.Node{
		Class("flex flex-col gap-4"),
		ID("search-widget"),
		g.Attr("onsubmit", "event.preventDefault(); return false;"),
		SearchArea(q, filtersExpanded, filterFacets, filters, searchVisible),
		SearchView(view, filters, results),
	}
	return Form(attrs...)
}

func searchResults(view int, results []g.Node, oob bool) g.Node {
	var content g.Node
	if len(results) == 0 {
		content = searchResultsEmpty()
	} else {
		content = g.Group(results)
	}

	attrs := []g.Node{
		ID("search-results"),
		Class(resultsContainerClass(view)),
		content,
	}
	if oob {
		attrs = append([]g.Node{hx.SwapOOB("outerHTML")}, attrs...)
	}
	return Div(attrs...)
}

func searchResultsEmpty() g.Node {
	return SearchResultsEmptyMessage()
}

func searchResultsMessage(text string) g.Node {
	return P(
		Class("col-span-full py-8 text-center text-zinc-500 dark:text-zinc-400"),
		g.Text(text),
	)
}

// SearchResultsEmptyMessage is the standard no-results copy for search grids.
func SearchResultsEmptyMessage() g.Node {
	return searchResultsMessage("Sorry, no ads found matching that criteria.")
}

// NoInAreaMatchesMessage is shown when geo search has no matches in the within area.
func NoInAreaMatchesMessage(within int, unit, location string) g.Node {
	unitLabel := "miles"
	if unit == "km" {
		unitLabel = "kilometers"
	}
	msg := fmt.Sprintf("Sorry, no ads found within %d %s of %s.", within, unitLabel, location)
	return searchResultsMessage(msg)
}

// OutsideAreaHeading separates in-area from out-of-area search results.
func OutsideAreaHeading() g.Node {
	return Div(
		Class("col-span-full flex items-center gap-4 my-6"),
		Span(Class("flex-1 border-t border-blue-500 dark:border-blue-400")),
		Span(
			Class("shrink-0 text-base font-medium text-blue-600 dark:text-blue-400"),
			g.Text("Outside of area"),
		),
		Span(Class("flex-1 border-t border-blue-500 dark:border-blue-400")),
	)
}
