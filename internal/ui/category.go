package ui

import (
	"net/url"
	"strconv"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

const categorySelectOpenID = "category-select-open"

// CategorySelectOpts configures the shared category picker (new-ad + filters).
type CategorySelectOpts struct {
	Selected   CategoryOption
	Categories []CategoryOption
	// ReturnParam selects posting-category switch mode (e.g. "/auth/ad/new").
	// Empty means search-filter mode via /api/show-filters.
	ReturnParam string
	IncludeAll  bool
	ID          string
}

// categoryIcon paints an SVG as content-ink via CSS mask so it matches text.
func categoryIcon(imageFile, sizeClass string) g.Node {
	if imageFile == "" {
		return Span(Class("inline-block shrink-0 "+sizeClass),
			g.Attr("aria-hidden", "true"))
	}
	src := "/images/category/" + imageFile
	return Span(
		Class("icon-ink inline-block shrink-0 "+sizeClass),
		Style("-webkit-mask-image:url('"+src+"');mask-image:url('"+src+"')"),
		g.Attr("aria-hidden", "true"),
	)
}

func inkIcon(src, sizeClass, inkClass string) g.Node {
	return Span(
		Class(inkClass+" inline-block shrink-0 "+sizeClass),
		Style("-webkit-mask-image:url('"+src+"');mask-image:url('"+src+"')"),
		g.Attr("aria-hidden", "true"),
	)
}

// CategoryReadOnly shows the category with icon in the same brass control
// chrome as CategorySelect, without a dropdown.
func CategoryReadOnly(category CategoryOption) g.Node {
	return Div(
		Class("brand-metal brand-metal-control w-full py-2 px-3 flex "+
			"items-center gap-2 rounded-md min-w-0"),
		g.Attr("aria-label", "Category "+category.Name),
		categoryIcon(category.ImageFile, "w-6 h-6"),
		Span(Class("font-medium truncate"), g.Text(category.Name)),
	)
}

func CategorySelect(opts CategorySelectOpts) g.Node {
	action := "Change category"
	selected := opts.Selected
	if opts.IncludeAll && selected.ID == 0 && selected.Name == "" {
		selected = CategoryOption{ID: 0, Name: "All Categories"}
	}

	items := make([]g.Node, 0, len(opts.Categories)+1)
	if opts.IncludeAll {
		items = append(items, categorySelectItem(
			selected.ID, 0, "All Categories", "", opts.ReturnParam))
	}
	for _, cat := range opts.Categories {
		items = append(items, categorySelectItem(
			selected.ID, cat.ID, cat.Name, cat.ImageFile, opts.ReturnParam))
	}

	attrs := []g.Node{
		Class("relative w-full z-0"),
	}
	if opts.ID != "" {
		attrs = append(attrs, ID(opts.ID))
	}
	attrs = append(attrs,
		Input(
			Type("checkbox"),
			ID(categorySelectOpenID),
			Class("category-select-open"),
		),
		Label(
			For(categorySelectOpenID),
			Class("brand-metal brand-metal-control category-select-trigger "+
				"w-full py-2 px-3 flex items-center gap-2 rounded-md "+
				"cursor-pointer text-left min-w-0"),
			g.Attr("aria-haspopup", "listbox"),
			g.Attr("aria-label", action+", currently "+selected.Name),
			g.Attr("title", action),
			categoryIcon(selected.ImageFile, "w-6 h-6"),
			Span(Class("font-medium truncate flex-1"), g.Text(selected.Name)),
			inkIcon("/images/expand.svg", "w-5 h-5", "icon-ink-soft"),
		),
		Label(
			For(categorySelectOpenID),
			Class("category-select-dismiss fixed inset-0 z-40"),
			g.Attr("aria-label", "Close category list"),
		),
		Div(
			Class("category-select-menu absolute left-0 right-0 z-50 mt-1 py-1 "+
				"rounded-md overflow-y-auto max-h-[85vh]"),
			g.Attr("role", "listbox"),
			g.Group(items),
		),
	)
	return Div(attrs...)
}

// CategorySelectForNewAd is the posting-category picker on create-ad.
func CategorySelectForNewAd(selected CategoryOption,
	categories []CategoryOption) g.Node {
	return CategorySelect(CategorySelectOpts{
		Selected:    selected,
		Categories:  categories,
		ReturnParam: "/auth/ad/new",
	})
}

// CategorySelectForFilter is the search filter category picker (includes All).
func CategorySelectForFilter(selectedID int,
	categories []CategoryOption) g.Node {
	selected := CategoryOption{ID: 0, Name: "All Categories"}
	for _, cat := range categories {
		if cat.ID == selectedID {
			selected = cat
			break
		}
	}
	return CategorySelect(CategorySelectOpts{
		Selected:   selected,
		Categories: categories,
		IncludeAll: true,
		ID:         "filter-category",
	})
}

func categorySelectItem(currentID, categoryID int, name, imageFile,
	returnParam string) g.Node {
	itemClass := "category-select-item flex items-center gap-2 px-3 py-2 " +
		"cursor-pointer whitespace-nowrap"
	if categoryID == currentID {
		itemClass += " category-select-item-selected"
	}

	attrs := []g.Node{
		Class(itemClass),
		g.Attr("role", "option"),
		hx.Swap("none"),
	}
	if returnParam != "" {
		returnURL := "/api/category/" + strconv.Itoa(categoryID) +
			"/switch?return=" + url.QueryEscape(returnParam)
		attrs = append(attrs, hx.Get(returnURL))
	} else {
		attrs = append(attrs,
			hx.Get("/api/show-filters?category="+strconv.Itoa(categoryID)),
			hx.Include("#search-widget"),
		)
	}
	attrs = append(attrs,
		categoryIcon(imageFile, "w-6 h-6"),
		Span(g.Text(name)),
	)
	return Div(attrs...)
}
