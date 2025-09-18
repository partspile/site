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
		if bounds != nil {
			// Only create geo filter for saved bounds (user has previously interacted with map)
			geoFilter = vector.BuildBoundingBoxGeoFilter(bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
		}
		// If no saved bounds either, don't create geo filter (show all ads)
	} else {
		// Save bounds to cookies when they are provided (user interaction)
		saveCookieMapBounds(ctx, bounds)
		// Create geo filter for user interaction bounds
		geoFilter = vector.BuildBoundingBoxGeoFilter(bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}

	return &MapView{ctx, bounds, geoFilter}
}

func (v *MapView) GetAdIDs() ([]int, string, error) {
	return getAdIDs(v.ctx, v.geoFilter)
}

func (v *MapView) RenderSearchResults(adIDs []int, nextCursor string) error {
	currentUser, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
	if err != nil {
		return err
	}

	return render(v.ctx, ui.MapViewResults(ads, userID, loc, v.bounds))
}

func (v *MapView) RenderSearchPage(adIDs []int, nextCursor string) error {
	currentUser, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Convert ad IDs to full ad objects for UI rendering
	ads, err := ad.GetAdsByIDs(adIDs, currentUser)
	if err != nil {
		return err
	}

	// For map view, return only the map data for HTMX updates
	return render(v.ctx, ui.MapDataOnly(ads, userID, loc))
}
