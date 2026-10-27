package ad

import (
	"strings"
	"testing"

	"github.com/partspile/site/internal/facet"
)

func TestLocationFacetHelpers(t *testing.T) {
	garage := Category{FacetKeys: []string{
		"sale_type", "address",
	}}
	if !HasLocationFacet(garage) {
		t.Fatal("expected location facet")
	}
	if LocationFacetKey(garage) != "address" {
		t.Fatalf("key = %q", LocationFacetKey(garage))
	}
	if !UsesFullAddressDisplay(garage) {
		t.Fatal("garage sale should use full address display")
	}

	car := Category{FacetKeys: []string{"mileage", "price"}}
	if HasLocationFacet(car) {
		t.Fatal("car category should not have location facet")
	}

	addr := "123 Main St, Portland, OR"
	facets := map[string]facet.Value{
		"address": {Text: &addr},
	}
	if got := LocationTextFromFacets(garage, facets); got != addr {
		t.Fatalf("LocationTextFromFacets = %q", got)
	}
}

func TestAdLocationDisplayAddressPrivacy(t *testing.T) {
	garage := Category{FacetKeys: []string{"address"}}
	addr := "123 Main St, Portland, OR"
	a := Ad{
		City:      "Portland",
		AdminArea: "OR",
		Country:   "US",
		Facets: map[string]facet.Value{
			"address": {Text: &addr},
		},
	}

	if got := locationDisplayForCategory(a, garage, 0); got != addressLoginPrompt {
		t.Fatalf("logged out = %q, want %q", got, addressLoginPrompt)
	}
	if got := locationDisplayForCategory(a, garage, 1); got != addr {
		t.Fatalf("logged in = %q, want address", got)
	}

	car := Category{FacetKeys: []string{"mileage", "price"}}
	if got := locationDisplayForCategory(a, car, 0); got == "" {
		t.Fatal("non-address category should still show city for logged-out viewers")
	}
}

func TestAdFlyerLocationHidesLoginPrompt(t *testing.T) {
	garage := Category{FacetKeys: []string{"address"}}
	addr := "123 Main St, Portland, OR"
	a := Ad{
		City:      "Portland",
		AdminArea: "OR",
		Country:   "US",
		Facets: map[string]facet.Value{
			"address": {Text: &addr},
		},
	}

	got := flyerLocationForCategory(a, garage, 0)
	if got == addressLoginPrompt {
		t.Fatal("guest flyer should not print login prompt")
	}
	if !strings.Contains(got, "Portland") {
		t.Fatalf("guest flyer location %q, want city", got)
	}
	if got := flyerLocationForCategory(a, garage, 1); got != addr {
		t.Fatalf("logged-in flyer = %q, want address", got)
	}
}
