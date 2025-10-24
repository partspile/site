package ui

import (
	"fmt"
	"strings"
	"time"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
)

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

// AdAdCategoryPills renders the category selection pills above the search form
func AdAdCategoryPills(activeAdCategory string) g.Node {
	categories := ad.GetAllCategories()

	var pills []g.Node
	for _, category := range categories {
		isActive := category == activeAdCategory
		pillClass := "px-4 py-2 rounded-full text-sm font-medium transition-colors "
		if isActive {
			pillClass += "bg-blue-500 text-white"
		} else {
			pillClass += "bg-gray-200 text-gray-700 hover:bg-gray-300"
		}

		pill := Button(
			Class(pillClass),
			hx.Get("/search"),
			hx.Target("#searchContainer"),
			hx.Swap("outerHTML"),
			hx.Include("form"),
			hx.On("click", fmt.Sprintf("document.getElementById('ad-category-input').value = '%s'", category)),
			g.Text(ad.GetDisplayName(category)),
		)
		pills = append(pills, pill)
	}

	// Create a slice with all the nodes
	allNodes := []g.Node{Class("flex flex-wrap gap-2 mb-4")}
	allNodes = append(allNodes, pills...)
	return Div(allNodes...)
}

func ViewToggleButtons(activeView string) g.Node {
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
			hx.Post("/view/"+view),
			hx.Target("#searchResults"),
			hx.Swap("outerHTML"),
			hx.Include("form"),
			hx.Trigger("click"),
			hx.On("click", "document.getElementById('view-type-input').value = '"+view+"'"),
			icon(view, alt),
		)
	}
	return Div(
		Class("flex justify-end gap-2 mb-4 mt-6"),
		button("list", "List View"),
		button("tree", "Tree View"),
		button("grid", "Grid View"),
	)
}

func InitialSearchResults(userID int, view string, activeAdCategory string) g.Node {
	return Div(
		ID("searchContainer"),
		SearchWidget(userID, view, "", activeAdCategory),
		Div(
			ID("searchResults"),
			Class("h-96"),
			hx.Get("/search?q=&view="+view),
			hx.Trigger("load"),
			hx.Target("this"),
			hx.Swap("outerHTML"),
		),
	)
}

func SearchWidget(userID int, view string, query string, activeAdCategory string) g.Node {
	return Div(
		Class("flex items-start gap-4"),
		renderNewAdButton(userID),
		searchForm(view, query, activeAdCategory, Div(
			Class("flex gap-2 items-center"),
			searchBox(query),
			filtersButton(),
		)),
	)
}

func createInfiniteScrollTrigger(loaderURL string) g.Node {
	return g.If(loaderURL != "", Div(
		Class("h-4"),
		hx.Get(loaderURL),
		hx.Trigger("revealed"),
		hx.Swap("outerHTML"),
		hx.Include("#searchForm, #filtersArea"),
	))
}

// SearchCreateLoaderURL creates the loader URL for pagination
func SearchCreateLoaderURL(userPrompt, nextCursor, view string) string {
	if nextCursor == "" {
		return ""
	}

	return fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s",
		htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view))
}

// Helper function to render new ad button based on user login
func renderNewAdButton(userID int) g.Node {
	if userID != 0 {
		return button("New Ad", withHref("/new-ad"))
	}
	return button("New Ad", withDisabled())
}

// searchBox creates the search input field
func searchBox(query string) g.Node {
	return Input(
		Type("search"),
		ID("searchBox"),
		Name("q"),
		Value(query),
		hx.Trigger("search"),
		Class("w-full p-2 border rounded"),
		Placeholder("What are you looking for?"),
	)
}

// searchForm creates the common search form structure
func searchForm(view string, query string, activeAdCategory string, content g.Node) g.Node {
	return Div(
		Class("flex-1 flex flex-col gap-4"),
		AdAdCategoryPills(activeAdCategory),
		Form(
			ID("searchForm"),
			Class("w-full"),
			hx.Get("/search"),
			hx.Target("#searchResults"),
			hx.Swap("outerHTML"),
			hx.Include("form"),
			Input(Type("hidden"), Name("view"), Value(view), ID("view-type-input")),
			Input(Type("hidden"), Name("ad_category"), Value(activeAdCategory), ID("ad-category-input")),
			content,
		),
	)
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

// SearchPage renders a full search page with search widget and results
func SearchPage(userID int, userName string, query string, ads []ad.Ad, loc *time.Location, loaderURL string, activeAdCategory string) g.Node {
	return Div(
		ID("searchContainer"),
		SearchWidget(userID, "list", query, activeAdCategory),
		Div(
			ID("searchResults"),
			ListViewResults(ads, userID, userName, loc, loaderURL),
		),
	)
}
