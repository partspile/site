package handler

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/currency"
	"github.com/partspile/site/internal/facet"
)

func TestParseSearchParamsAppliesCategory(t *testing.T) {
	p := parseSearchParams(
		cookie.SearchState{Q: "Honda", CategoryID: 6},
		1, 0, "",
	)
	if p.CategoryID != 6 {
		t.Fatalf("category = %d", p.CategoryID)
	}
	if p.Q != "Honda" {
		t.Fatalf("q = %q", p.Q)
	}
}

func TestSaveSearchStateAlwaysPersistsLocation(t *testing.T) {
	app := fiber.New()
	var state cookie.SearchState
	app.Get("/search", func(c *fiber.Ctx) error {
		state = saveSearchStateFromRequest(c, true)
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest("GET", "/search?location=Denver&within=50&q=test", nil)
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	if state.Location != "Denver" || state.Within != 50 || state.Q != "test" {
		t.Fatalf("got state %+v", state)
	}
}

func TestFilterableInRegistryOrder(t *testing.T) {
	// Cars & Trucks lists price last. Pills follow the registry.
	allowed := map[string]bool{
		"condition":    true,
		"title_status": true,
		"year":         true,
		"mileage":      true,
		"local_pickup": true,
		"price":        true,
		"location":     true,
	}
	got := filterableInRegistryOrder(allowed)
	want := []string{
		"price", "mileage", "year", "condition",
		"title_status", "local_pickup",
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i, d := range got {
		if d.Key != want[i] {
			t.Errorf("[%d] = %s, want %s", i, d.Key, want[i])
		}
	}

	parts := filterableInRegistryOrder(map[string]bool{
		"condition": true, "local_pickup": true, "price": true,
	})
	partsWant := []string{"price", "condition", "local_pickup"}
	if len(parts) != len(partsWant) {
		t.Fatalf("parts len = %d, want %d", len(parts), len(partsWant))
	}
	for i, d := range parts {
		if d.Key != partsWant[i] {
			t.Errorf("parts[%d] = %s, want %s", i, d.Key, partsWant[i])
		}
	}
}

func TestParseFacetFiltersConditionCheckboxes(t *testing.T) {
	defs := filterableFacets(ad.Category{FacetKeys: []string{"condition"}}.Facets())

	app := fiber.New()
	var filters map[string]facet.Filter
	app.Get("/search", func(c *fiber.Ctx) error {
		filters = parseFacetFilters(c, defs)
		return c.SendStatus(fiber.StatusOK)
	})

	q := url.Values{}
	q.Add("condition", "New")
	q.Add("condition", "Used - Good")
	req := httptest.NewRequest("GET", "/search?"+q.Encode(), nil)
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	got := filters["condition"]
	if len(got.Values) != 2 || got.Values[0] != "New" || got.Values[1] != "Used - Good" {
		t.Fatalf("condition filter = %+v", got)
	}
}

func TestParseFacetFiltersPriceWantedIgnoresRange(t *testing.T) {
	price, ok := facet.Get("price")
	if !ok {
		t.Fatal("price facet missing")
	}
	app := fiber.New()
	var filters map[string]facet.Filter
	app.Get("/search", func(c *fiber.Ctx) error {
		filters = parseFacetFilters(c, []facet.Def{price})
		return c.SendStatus(fiber.StatusOK)
	})
	req := httptest.NewRequest("GET",
		"/search?price_wanted=1&price_min=10&price_max=50", nil)
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	got := filters["price"]
	if got.Value == nil || *got.Value != currency.WantedFilter {
		t.Fatalf("wanted filter = %+v", got)
	}
	if got.Min != nil || got.Max != nil {
		t.Fatalf("wanted filter should ignore range: %+v", got)
	}
}
