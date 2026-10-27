package ui

import (
	"bytes"
	"strings"
	"testing"

	uiads "github.com/partspile/site/internal/ui/ads"
)

func TestCategorySelectLooksLikePicker(t *testing.T) {
	var buf bytes.Buffer
	n := SearchContainer(0, 0, "", false, false, CategoryOption{
		ID: 1, Name: "Car & Truck Parts", ImageFile: "car.svg",
	}, []CategoryOption{
		{ID: 1, Name: "Car & Truck Parts", ImageFile: "car.svg"},
		{ID: 2, Name: "Bicycles", ImageFile: "bicycle.svg"},
	}, nil, uiads.SearchFilters{}, nil)
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	html := buf.String()
	for _, want := range []string{
		"Change category",
		`aria-haspopup="listbox"`,
		"/images/expand.svg",
		"Car &amp; Truck Parts",
		"Bicycles",
		"<div",
		`role="listbox"`,
		`id="category-select-open"`,
		"category-select-dismiss",
		"/api/category/2/switch",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("category picker missing %q", want)
		}
	}
	if strings.Contains(html, "category-modal") {
		t.Error("category picker should not use a modal")
	}
}
