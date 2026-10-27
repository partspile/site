package search

import "github.com/partspile/site/internal/facet"

// WithinMileOptions are valid search within-distance dropdown values (miles).
var WithinMileOptions = []int{1, 5, 10, 25, 50, 100, 500, 1000}

// WithinKmOptions are valid search within-distance dropdown values (kilometers).
var WithinKmOptions = []int{1, 5, 10, 25, 50, 100, 500, 1000}

// Params holds hard filters for listing search.
type Params struct {
	CategoryID   int
	UserID       int
	Expanded     bool
	Limit        int
	Offset       int
	Q            string
	FacetFilters map[string]facet.Filter
	// CenterLat/CenterLon and WithinKm apply when location text resolved and within > 0.
	CenterLat float64
	CenterLon float64
	WithinKm  float64
	HasGeo    bool
}

func (p Params) HasTextQuery() bool {
	return p.Q != ""
}

// Results holds vector search output. For geo searches, InAreaOnPage and
// HasAnyInArea drive the outside-area heading without a global in-area count.
type Results struct {
	IDs          []int
	InAreaOnPage int
	HasAnyInArea bool
}
