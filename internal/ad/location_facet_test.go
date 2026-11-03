package ad

import (
	"strings"
	"testing"

	"github.com/partspile/site/internal/facet"
)

func TestLocationFacetHelpers(t *testing.T) {
	withLoc := Category{FacetKeys: []string{"location"}}
	if !HasLocationFacet(withLoc) {
		t.Fatal("expected location facet")
	}
	if LocationFacetKey(withLoc) != "location" {
		t.Fatalf("key = %q", LocationFacetKey(withLoc))
	}

	car := Category{FacetKeys: []string{"mileage", "price"}}
	if HasLocationFacet(car) {
		t.Fatal("car category should not have location facet")
	}

	loc := "Portland, OR"
	facets := map[string]facet.Value{
		"location": {Text: &loc},
	}
	if got := LocationTextFromFacets(withLoc, facets); got != loc {
		t.Fatalf("LocationTextFromFacets = %q", got)
	}
}

func TestAdLocationDisplay(t *testing.T) {
	a := Ad{
		City:      "Portland",
		AdminArea: "OR",
		Country:   "US",
	}
	got := AdLocationDisplay(a)
	if !strings.Contains(got, "Portland") {
		t.Fatalf("location = %q, want city", got)
	}
}
