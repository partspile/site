package ads

import (
	"testing"

	"github.com/partspile/site/internal/facet"
)

func TestFacetPillLabels(t *testing.T) {
	price, ok := facet.Get("price")
	if !ok {
		t.Fatal("price facet missing")
	}
	min, max := 0, 50
	if got := facetPillLabel(price, facet.Filter{}); got != "Price" {
		t.Fatalf("empty price pill = %q", got)
	}
	if got := facetPillLabel(price, facet.Filter{Min: &min, Max: &max}); got != "Price: 0-50" {
		t.Fatalf("bounded price pill = %q", got)
	}
}
