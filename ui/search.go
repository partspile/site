package ui

import (
	"fmt"
	"strconv"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

func createAdCategoryItems(activeCat int) []g.Node {
	var adCatItems []g.Node
	for _, adCat := range ad.GetCategoryIDs() {
		isActive := adCat == activeCat
		itemClass := "flex items-center gap-3 p-3 hover:bg-gray-50 cursor-pointer rounded-lg transition-colors "
		if isActive {
			itemClass += "bg-blue-50 border border-blue-200"
		}

		item := Div(
			Class(itemClass),
			hx.Get("/switch-ad-category/"+strconv.Itoa(adCat)),
			hx.Target("#searchContainer"),
			hx.Swap("outerHTML"),
			Div(
				Class("p-2 bg-gray-200 rounded-full flex items-center justify-center"),
				Img(
					Src(adCategoryIcon(adCat)),
					Alt("Category icon"),
					Class("w-6 h-6"),
				),
			),
			Span(Class("text-gray-700 flex-1"), g.Text(ad.GetCategoryName(adCat))),
		)
		adCatItems = append(adCatItems, item)
	}
	return adCatItems
}

func AdCategoryModal(activeCat int) g.Node {
	adCatItems := createAdCategoryItems(activeCat)

	return Div(
		ID("category-select-modal"),
		Class("fixed inset-0 bg-black/30 flex items-center justify-center z-50 p-8"),
		g.Attr("onclick", "this.remove()"),
		Div(
			Class("bg-white rounded-lg w-full shadow-2xl border-2 border-gray-300 flex flex-col"),
			Style("max-width: 400px; max-height: 80vh"),
			g.Attr("onclick", "event.stopPropagation()"),
			Div(
				Class("flex items-center justify-between p-6 border-b border-gray-200 flex-shrink-0"),
				H3(Class("text-xl font-bold text-gray-900"), g.Text("Select Category")),
				Button(
					Type("button"),
					Class("bg-white border-2 border-gray-800 rounded-full w-8 h-8 flex items-center justify-center shadow-lg hover:bg-gray-100 focus:outline-none cursor-pointer"),
					g.Attr("onclick", "this.closest('.fixed').remove()"),
					Img(
						Src("/images/close.svg"),
						Alt("Close"),
						Class("w-4 h-4"),
					),
				),
			),
			Div(
				Class("flex-1 overflow-y-auto p-6 pt-4"),
				Div(
					Class("space-y-2"),
					g.Group(adCatItems),
				),
			),
		),
	)
}

func ViewToggleButtons(activeView string, userID int) g.Node {
	icon := func(name, alt string) g.Node {
		return Img(
			Src("/images/"+name+".svg"),
			Alt(alt),
			Class("w-6 h-6 inline align-middle"),
		)
	}
	button := func(view, alt string) g.Node {
		active := activeView == view
		cls := "p-2 rounded-full border-2 "
		if active {
			cls += "border-blue-500 bg-blue-100"
		} else {
			cls += "border-transparent hover:bg-gray-100"
		}
		return Button(
			Class(cls),
			hx.Get("/view/"+view),
			hx.Target("#searchResults"),
			hx.Swap("outerHTML"),
			hx.Include("#searchWidget"),
			hx.Trigger("click"),
			icon(view, alt),
		)
	}
	return Div(
		Class("flex justify-between items-center gap-2 mb-4 mt-6"),
		newAdButton(userID),
		Div(
			Class("flex gap-2"),
			button("list", "List View"),
			button("tree", "Tree View"),
			button("grid", "Grid View"),
		),
	)
}

func SearchContainer(userID, view, adCategory int, params map[string]string) g.Node {
	return Div(
		ID("searchContainer"),
		SearchWidget(userID, view, adCategory, params, false),
		SearchResults(),
		Script(g.Raw("document.getElementById('category-select-modal')?.remove();")),
	)
}

func searchBox(q string) g.Node {
	return Input(
		Class("w-full p-2 border rounded"),
		Type("search"),
		ID("searchBox"),
		Name("q"),
		Value(q),
		hx.Trigger("search"),
		Placeholder("What are you looking for?"),
	)
}

func searchWidgetWithFilters(adCategory int, params map[string]string) g.Node {
	return Div(
		Class("border rounded-lg p-4"),
		searchBox(params["q"]),
		filterControls(adCategory, params),
		filterActions(adCategory),
	)
}

func searchWidgetSimple(query string) g.Node {
	return Div(
		Class("flex gap-2 items-center"),
		searchBox(query),
		filtersButton(),
	)
}

func SearchWidget(userID, view, adCategory int, params map[string]string, showFilters bool) g.Node {
	return Form(
		ID("searchWidget"),
		Class("flex flex-col gap-4"),
		hx.Get("/search"),
		hx.Target("#searchResults"),
		hx.Swap("outerHTML"),
		hx.Include("form"),
		adCatButton(adCategory),
		g.If(showFilters, searchWidgetWithFilters(adCategory, params)),
		g.If(!showFilters, searchWidgetSimple(params["q"])),
	)
}

func SearchResults() g.Node {
	return Div(
		ID("searchResults"),
		hx.Get("/search"),
		hx.Trigger("load"),
		hx.Target("this"),
		hx.Include("#searchWidget"),
		hx.Swap("outerHTML"),
	)
}

func createInfiniteScrollTrigger(loaderURL string) g.Node {
	return g.If(loaderURL != "", Div(
		Class("h-4"),
		hx.Get(loaderURL),
		hx.Trigger("revealed"),
		hx.Swap("outerHTML"),
		hx.Include("#searchWidget"),
	))
}

// SearchCreateLoaderURL creates the loader URL for pagination
func SearchCreateLoaderURL(cursor uint64) string {
	if cursor == 0 {
		return ""
	}
	return fmt.Sprintf("/search-page?cursor=%d", cursor)
}

// MakeFilterOptions returns HTML options for the make filter dropdown
func MakeFilterOptions(makes []string) g.Node {
	var options []g.Node

	// Add "All Makes" option first
	options = append(options, Option(Value(""), g.Text("All Makes")))

	// Add make options
	for _, make := range makes {
		options = append(options, Option(Value(make), g.Text(make)))
	}

	return g.Group(options)
}

// ModelFilterOptions returns HTML options for the model filter dropdown
func ModelFilterOptions(models []string) g.Node {
	var options []g.Node

	// Add "All Models" option first
	options = append(options, Option(Value(""), g.Text("All Models")))

	// Add model options
	for _, model := range models {
		options = append(options, Option(Value(model), g.Text(model)))
	}

	return g.Group(options)
}

// EngineFilterOptions returns HTML options for the engine filter dropdown
func EngineFilterOptions(engines []string) g.Node {
	var options []g.Node

	// Add "All Engines" option first
	options = append(options, Option(Value(""), g.Text("All Engines")))

	// Add engine options
	for _, engine := range engines {
		options = append(options, Option(Value(engine), g.Text(engine)))
	}

	return g.Group(options)
}

// BreedFilterOptions returns HTML options for the breed filter dropdown
func BreedFilterOptions(breeds []string) g.Node {
	var options []g.Node

	// Add "All Breeds" option first
	options = append(options, Option(Value(""), g.Text("All Breeds")))

	// Add breed options
	for _, breed := range breeds {
		options = append(options, Option(Value(breed), g.Text(breed)))
	}

	return g.Group(options)
}

// GenericFilterOptions returns HTML options for any field filter dropdown
func GenericFilterOptions(fieldName string, values []string, selectedValue string) g.Node {
	var options []g.Node

	// Add "All" option first
	allOption := Option(Value(""), g.Text("All"))
	if selectedValue == "" {
		allOption = Option(Value(""), g.Text("All"), Selected())
	}
	options = append(options, allOption)

	// Add field value options
	for _, value := range values {
		optionAttrs := []g.Node{Value(value), g.Text(value)}
		if value == selectedValue {
			optionAttrs = append(optionAttrs, Selected())
		}
		options = append(options, Option(optionAttrs...))
	}

	return g.Group(options)
}

// adCatButton renders the category selection button
func adCatButton(adCategory int) g.Node {
	return Div(
		Div(
			Class("flex items-center gap-5"),
			Label(
				Class("text-base font-bold text-gray-900 whitespace-nowrap"),
				g.Text("Category"),
			),
			Button(
				Type("button"),
				Class("py-2 px-5 flex items-center gap-2 rounded-full border-2 border-blue-500 bg-blue-100 hover:bg-blue-200"),
				hx.Get("/modal/category-select"),
				hx.Target("body"),
				hx.Swap("beforeend"),
				Img(
					Src(adCategoryIcon(adCategory)),
					Alt("Category icon"),
					Class("w-6 h-6"),
				),
				Span(g.Text(ad.GetCategoryName(adCategory))),
			),
		),
	)
}
