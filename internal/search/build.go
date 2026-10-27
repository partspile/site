package search

import (
	"github.com/partspile/site/internal/facet"
	"github.com/partspile/site/internal/location"
)

// BuildParams converts filter fields into search Params, resolving geo when applicable.
func BuildParams(categoryID, limit, offset int, q, locationText string,
	within int, withinUnit string, facetFilters map[string]facet.Filter) Params {
	p := Params{
		CategoryID:   categoryID,
		Limit:        limit,
		Offset:       offset,
		Q:            q,
		FacetFilters: facetFilters,
	}

	if locationText == "" || within <= 0 {
		return p
	}

	lat, lon, ok, err := location.ResolveLocation(locationText)
	if err != nil || !ok {
		return p
	}

	p.CenterLat = lat
	p.CenterLon = lon
	if withinUnit == location.UnitKm {
		p.WithinKm = float64(within)
	} else {
		p.WithinKm = location.MilesToKm(float64(within))
	}
	p.HasGeo = true
	return p
}
