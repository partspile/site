package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
)

// MapView implements the View interface for map view
type MapView struct {
	ctx       *fiber.Ctx
	bounds    *ui.GeoBounds
	geoFilter *qdrant.Filter
}

// NewMapView creates a new map view
func NewMapView(ctx *fiber.Ctx) *MapView {
	var bounds *ui.GeoBounds = extractMapBounds(ctx)
	var geoFilter *qdrant.Filter

	// If no bounds provided in query, try to load from cookies
	if bounds == nil {
		bounds, _ = getCookieMapBounds(ctx)
		if bounds == nil {
			return &MapView{ctx, nil, nil}
		}
	} else {
		// Save bounds to cookies
		saveCookieMapBounds(ctx, bounds)
	}

	geoFilter = vector.BuildBoundingBoxGeoFilter(bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)

	return &MapView{ctx, bounds, geoFilter}
}

func (v *MapView) GetAdIDs() ([]int, string, error) {
	return getAdIDs(v.ctx, v.geoFilter)
}

func (v *MapView) RenderSearchResults(adIDs []int, _ string) error {
	currentUser, userID := CurrentUser(v.ctx)
	loc := getLocation(v.ctx)
	hadBounds := v.bounds != nil

	// If no results found and geo filter is present, retry without geo filter
	if len(adIDs) == 0 && v.geoFilter != nil {
		var err error
		adIDs, _, err = getAdIDs(v.ctx, nil)
		if err != nil {
			return err
		}
		v.geoFilter = nil
	}

	// Convert ad IDs to full ad objects
	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
	if err != nil {
		return err
	}

	// Only calculate extent and update bounds if we had NO bounds to begin with
	// (first-time visitor). If user has explicitly set bounds (zoom/pan),
	// preserve those even if no results found in that area.
	if !hadBounds && len(ads) > 0 {
		minLat, maxLat, minLon, maxLon, found := ad.CalculateExtent(ads)
		if found {
			v.bounds = &ui.GeoBounds{
				MinLat: minLat,
				MaxLat: maxLat,
				MinLon: minLon,
				MaxLon: maxLon,
			}
		}
	}

	return render(v.ctx, ui.MapViewResults(ads, userID, loc, v.bounds))
}

func (v *MapView) RenderSearchPage(adIDs []int, _ string) error {
	currentUser, userID := CurrentUser(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects
	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
	if err != nil {
		return err
	}

	// For map view, return only the map data for HTMX updates
	return render(v.ctx, ui.MapDataOnly(ads, userID, loc))
}
