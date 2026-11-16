package ui

import (
	"bytes"
	"strings"
	"testing"

	uiads "github.com/partspile/site/internal/ui/ads"
)

func TestHomeShowsFilterPills(t *testing.T) {
	var buf bytes.Buffer
	n := SearchContainer("", nil, uiads.SearchFilters{}, nil)
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
}

func TestSearchFilterPillsHasCategorySelect(t *testing.T) {
	var buf bytes.Buffer
	cats := []CategoryOption{
		{ID: 1, Name: "Car & Truck Parts", ImageFile: "car.svg"},
		{ID: 2, Name: "Bicycles", ImageFile: "bicycle.svg"},
	}
	n := uiads.SearchFilterPills(nil, uiads.SearchFilters{
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
		"/api/search-filters?category=",
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
