package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
)

// MapView implements the View interface for map view
type MapView struct {
	ctx    *fiber.Ctx
	bounds *ui.GeoBounds
}

// NewMapView creates a new map view
func NewMapView(ctx *fiber.Ctx, bounds *ui.GeoBounds) *MapView {
	return &MapView{ctx: ctx, bounds: bounds}
}

func (v *MapView) GetAds() ([]ad.Ad, string, error) {
	if v.bounds != nil {
		geoFilter := vector.BuildBoundingBoxGeoFilter(v.bounds.MinLat, v.bounds.MaxLat, v.bounds.MinLon, v.bounds.MaxLon)
		return getAds(v.ctx, geoFilter)
	}
	return getAds(v.ctx, nil)
}

func (v *MapView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL if there are more results
	loaderURL := ui.SearchCreateLoaderURL(userPrompt, nextCursor, "map", threshold, v.bounds)

	loc := getLocation(v.ctx)
	return render(v.ctx, ui.MapViewRenderResults(ads, userID, loc, userPrompt, loaderURL, threshold))
}

func (v *MapView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL for infinite scroll
	var loaderURL string
	if nextCursor != "" {
		userPrompt := getQueryParam(v.ctx, "q")
		threshold := getThreshold(v.ctx)
		loaderURL = ui.SearchCreateLoaderURL(userPrompt, nextCursor, "map", threshold, v.bounds)
	}

	// Render the page content using UI function
	return render(v.ctx, ui.MapViewRenderPage(ads, userID, loc, loaderURL))
}
