package ui

import (
	"fmt"
	"strings"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// htmlEscape escapes HTML special characters
func htmlEscape(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
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

func InitialSearchResults(userID int, view string) g.Node {
	return Div(
		SearchWidget(userID, view, ""),
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

func SearchWidget(userID int, view string, query string) g.Node {
	return Div(
		Class("flex items-start gap-4"),
		renderNewAdButton(userID),
		searchForm(view, query, Div(
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
		return styledLink("New Ad", "/new-ad", buttonPrimary)
	}
	return styledLinkDisabled("New Ad", buttonPrimary)
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
func searchForm(view string, query string, content g.Node) g.Node {
	return Div(
		Class("flex-1 flex flex-col gap-4"),
		Form(
			ID("searchForm"),
			Class("w-full"),
			hx.Get("/search"),
			hx.Target("#searchResults"),
			hx.Swap("outerHTML"),
			hx.Include("form"),
			Input(Type("hidden"), Name("view"), Value(view), ID("view-type-input")),
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
