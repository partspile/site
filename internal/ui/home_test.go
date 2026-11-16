package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/partspile/site/internal/facet"
	uiads "github.com/partspile/site/internal/ui/ads"
)

func TestHomeShowsFilterPills(t *testing.T) {
	var buf bytes.Buffer
	n := SearchContainer(0, "", nil, uiads.SearchFilters{}, nil)
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		`id="search-container"`,
		`id="searchBox"`,
		`id="search-filters"`,
		`id="filter-category"`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("home missing %q", want)
		}
	}
	for _, ban := range []string{
		`id="filter-toggle"`,
		`id="search-toggle"`,
		"/images/search.svg",
		"/api/hide-filters",
	} {
		if strings.Contains(html, ban) {
			t.Errorf("home should not include %q", ban)
		}
	}
}

func TestSearchFiltersPanelHasCategorySelect(t *testing.T) {
	var buf bytes.Buffer
	cats := []CategoryOption{
		{ID: 1, Name: "Car & Truck Parts", ImageFile: "car.svg"},
		{ID: 2, Name: "Bicycles", ImageFile: "bicycle.svg"},
	}
	n := uiads.SearchFiltersPanel(nil, uiads.SearchFilters{
		CategoryID: 0,
		Categories: []uiads.CategoryOption{
			{ID: 1, Name: "Car & Truck Parts", ImageFile: "car.svg"},
			{ID: 2, Name: "Bicycles", ImageFile: "bicycle.svg"},
		},
	}, CategorySelectForFilter(0, cats))
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		`id="filter-category"`,
		"All Categories",
		"Car &amp; Truck Parts",
		"Bicycles",
		"/api/show-filters?category=",
		"/images/category/car.svg",
		"category-select-open",
		"filter-pill",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("filters missing %q", want)
		}
	}
	if strings.Contains(html, `name="category"`) {
		t.Error("filter category should use icon picker, not native select")
	}
}

func TestFacetPillLabels(t *testing.T) {
	price, ok := facet.Get("price")
	if !ok {
		t.Fatal("price facet missing")
	}
	min, max := 0, 50
	if got := uiads.FacetPillLabel(price, facet.Filter{}); got != "Price" {
		t.Fatalf("empty price pill = %q", got)
	}
	if got := uiads.FacetPillLabel(price, facet.Filter{Min: &min, Max: &max}); got != "Price: 0-50" {
		t.Fatalf("bounded price pill = %q", got)
	}
}
