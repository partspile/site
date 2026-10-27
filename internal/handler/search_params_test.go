package handler

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/cookie"
	"github.com/partspile/site/internal/facet"
)

func TestSearchVisibleToggle(t *testing.T) {
	if searchVisible(cookie.SearchState{}) {
		t.Fatal("expected search hidden on fresh state")
	}
	if !searchVisible(cookie.SearchState{SearchOpen: true}) {
		t.Fatal("expected search visible when SearchOpen")
	}
}

func TestParseSearchParamsCollapsedNoFacets(t *testing.T) {
	app := fiber.New()
	app.Get("/search", func(c *fiber.Ctx) error {
		p := parseSearchParams(cookie.SearchState{Q: "Honda"}, c.QueryInt("page", 1), 6, 0, cookie.GetDistanceUnit(c), cookie.GetTimezone(c))
		if p.Expanded {
			t.Fatal("expected Expanded false when filter panel collapsed")
		}
		if p.FacetFilters != nil {
			t.Fatal("expected no facet filters when collapsed")
		}
		if p.Q != "Honda" {
			t.Fatalf("q = %q", p.Q)
		}
		return c.SendStatus(fiber.StatusOK)
	})
	if _, err := app.Test(httptest.NewRequest("GET", "/search", nil)); err != nil {
		t.Fatal(err)
	}
}

func TestSaveSearchStateAlwaysPersistsLocation(t *testing.T) {
	app := fiber.New()
	var state cookie.SearchState
	app.Get("/search", func(c *fiber.Ctx) error {
		state = saveSearchStateFromRequest(c, nil, true)
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

func TestParseFacetFiltersConditionCheckboxes(t *testing.T) {
	category := ad.Category{FacetKeys: []string{"condition"}}

	app := fiber.New()
	var filters map[string]facet.Filter
	app.Get("/search", func(c *fiber.Ctx) error {
		filters = parseFacetFilters(c, category)
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

func TestParseFacetFiltersSaleWeek(t *testing.T) {
	category := ad.Category{FacetKeys: []string{"sale_start_date"}}

	app := fiber.New()
	var filters map[string]facet.Filter
	app.Get("/search", func(c *fiber.Ctx) error {
		filters = parseFacetFilters(c, category)
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/search?sale_start_date=Next+week", nil)
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	got := filters["sale_start_date"]
	if got.Value == nil || *got.Value != "Next week" {
		t.Fatalf("Value = %v", got.Value)
	}
}
