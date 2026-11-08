package ui

import (
	"bytes"
	"strings"
	"testing"

	uiads "github.com/partspile/site/internal/ui/ads"
)

func TestHomeHasNoCategoryPicker(t *testing.T) {
	var buf bytes.Buffer
	n := SearchContainer(0, "", false, false, nil, uiads.SearchFilters{}, nil)
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		`id="search-container"`,
		`id="search-toggle"`,
		"/images/search.svg",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("home missing %q", want)
		}
	}
	for _, ban := range []string{
		"Change category",
		"category-select-open",
		"/api/category/",
	} {
		if strings.Contains(html, ban) {
			t.Errorf("home should not include %q", ban)
		}
	}
}

func TestSearchFiltersPanelHasCategorySelect(t *testing.T) {
	var buf bytes.Buffer
	n := uiads.SearchFiltersPanel(nil, uiads.SearchFilters{
		CategoryID: 0,
		Categories: []uiads.CategoryOption{
			{ID: 1, Name: "Car & Truck Parts"},
			{ID: 2, Name: "Bicycles"},
		},
	})
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		`id="filter-category"`,
		`name="category"`,
		"All categories",
		"Car &amp; Truck Parts",
		"Bicycles",
		"/api/show-filters",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("filters missing %q", want)
		}
	}
}
