package handlers

import (
	"strconv"

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

// extractMapBounds extracts geographic bounding box parameters for map view
func extractMapBounds(c *fiber.Ctx) *ui.GeoBounds {
	minLatStr := c.Query("minLat")
	maxLatStr := c.Query("maxLat")
	minLonStr := c.Query("minLon")
	maxLonStr := c.Query("maxLon")

	if minLatStr == "" || maxLatStr == "" || minLonStr == "" || maxLonStr == "" {
		return nil
	}

	minLat, err1 := strconv.ParseFloat(minLatStr, 64)
	maxLat, err2 := strconv.ParseFloat(maxLatStr, 64)
	minLon, err3 := strconv.ParseFloat(minLonStr, 64)
	maxLon, err4 := strconv.ParseFloat(maxLonStr, 64)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return nil
	}

	return &ui.GeoBounds{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLon: minLon,
		MaxLon: maxLon,
	}
}

// NewMapView creates a new map view
func NewMapView(ctx *fiber.Ctx) *MapView {
	var bounds *ui.GeoBounds = extractMapBounds(ctx)
	var geoFilter *qdrant.Filter
	if bounds != nil {
		geoFilter = vector.BuildBoundingBoxGeoFilter(bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}
	return &MapView{ctx, bounds, geoFilter}
}

func (v *MapView) GetAds() ([]ad.Ad, string, error) {
	return getAds(v.ctx, v.geoFilter)
}

func (v *MapView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	_, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	return render(v.ctx, ui.MapViewResults(ads, userID, loc))
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
