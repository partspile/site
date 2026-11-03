package ad

import (
	"strings"

	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/location"
)

// LocationFacetKey returns the facet key used for geo lookup on this category.
func LocationFacetKey(category Category) string {
	for _, key := range category.FacetKeys {
		if key == "location" {
			return key
		}
	}
	return ""
}

// HasLocationFacet reports whether the category uses a location facet.
func HasLocationFacet(category Category) bool {
	return LocationFacetKey(category) != ""
}

// LocationTextFromFacets returns trimmed location text from facets.
func LocationTextFromFacets(category Category,
	facets map[string]facet.Value) string {
	key := LocationFacetKey(category)
	if key == "" {
		return ""
	}
	v, ok := facets[key]
	if !ok || v.Text == nil {
		return ""
	}
	return strings.TrimSpace(*v.Text)
}

// AdLocationDisplay returns the location line for cards, detail, and flyers.
func AdLocationDisplay(a Ad) string {
	return location.DisplayText(a.City, a.AdminArea, a.Country)
}
