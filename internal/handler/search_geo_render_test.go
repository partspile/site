package handler

import (
	"strings"
	"testing"
	"time"

	"github.com/partspile/site/internal/ad"
)

func TestRenderSearchResultsGeoNoInArea(t *testing.T) {
	ads := []ad.Ad{{ID: 1, Title: "Far Away"}}
	nodes := renderSearchResults(
		ads,
		searchGeoMeta{InAreaOnPage: 0, HasAnyInArea: false},
		1, true, 0, 0, time.UTC, "",
		"Portland", 25, "mi",
	)
	html, err := renderToString(nodes[0])
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Portland") {
		t.Fatalf("expected no-in-area message, got %q", html)
	}
}

func TestRenderSearchResultsGeoStraddle(t *testing.T) {
	ads := []ad.Ad{
		{ID: 1, Title: "Near"},
		{ID: 2, Title: "Far"},
	}
	nodes := renderSearchResults(
		ads,
		searchGeoMeta{InAreaOnPage: 1, HasAnyInArea: true},
		1, true, 0, 0, time.UTC, "",
		"Portland", 25, "mi",
	)
	if len(nodes) < 3 {
		t.Fatalf("expected heading between cards, got %d nodes", len(nodes))
	}
	html, err := renderToString(nodes[1])
	if err != nil {
		t.Fatalf("render heading: %v", err)
	}
	if !strings.Contains(html, "Outside of area") {
		t.Fatalf("expected outside heading, got %q", html)
	}
}
