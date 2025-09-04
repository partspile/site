package handlers

import (
	"fmt"
	"log"
	"strconv"

	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vector"
	"github.com/qdrant/go-client/qdrant"
)

// GeoBounds represents a geographic bounding box
type GeoBounds struct {
	MinLat float64
	MaxLat float64
	MinLon float64
	MaxLon float64
}

// View interface defines the contract for different view implementations
type View interface {
	// GetAds retrieves ads for this view with appropriate search strategy
	GetAds() ([]ad.Ad, string, error)

	// RenderSearchResults renders the complete search results including container, ads, and pagination
	RenderSearchResults(ads []ad.Ad, nextCursor string) error

	// RenderSearchPage renders just the ads and infinite scroll for pagination
	RenderSearchPage(ads []ad.Ad, nextCursor string) error
}

// ListView implements the View interface for list view
type ListView struct {
	ctx *fiber.Ctx
}

// NewListView creates a new list view
func NewListView(ctx *fiber.Ctx) *ListView {
	return &ListView{ctx: ctx}
}

func (v *ListView) GetAds() ([]ad.Ad, string, error) {
	return getAds(v.ctx, nil)
}

func (v *ListView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL for infinite scroll
	loaderURL := ui.ListViewCreateLoaderURL(userPrompt, nextCursor, threshold)

	loc := getLocation(v.ctx)
	return render(v.ctx, ui.ListViewRenderResults(ads, userID, loc, loaderURL))
}

func (v *ListView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	userPrompt := v.ctx.Query("q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)
	loc := getLocation(v.ctx)

	// Create loader URL for infinite scroll
	loaderURL := ui.ListViewCreateLoaderURL(userPrompt, nextCursor, threshold)

	return render(v.ctx, ui.ListViewRenderPage(ads, userID, loc, loaderURL))
}

// GridView implements the View interface for grid view
type GridView struct {
	ctx *fiber.Ctx
}

// NewGridView creates a new grid view
func NewGridView(ctx *fiber.Ctx) *GridView {
	return &GridView{ctx: ctx}
}

func (v *GridView) GetAds() ([]ad.Ad, string, error) {
	return getAds(v.ctx, nil)
}

func (v *GridView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL if there are more results
	loaderURL := ui.GridViewCreateLoaderURL(userPrompt, nextCursor, threshold)

	loc := getLocation(v.ctx)
	return render(v.ctx, ui.GridViewRenderResults(ads, userID, loc, userPrompt, loaderURL, threshold))
}

func (v *GridView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL for infinite scroll
	var loaderURL string
	if nextCursor != "" {
		userPrompt := getQueryParam(v.ctx, "q")
		threshold := getThreshold(v.ctx)
		loaderURL = ui.GridViewCreateLoaderURL(userPrompt, nextCursor, threshold)
	}

	// Render the page content using UI function
	return render(v.ctx, ui.GridViewRenderPage(ads, userID, loc, loaderURL))
}

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
	loaderURL := ui.MapViewCreateLoaderURL(userPrompt, nextCursor, threshold, v.bounds)

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
		loaderURL = ui.MapViewCreateLoaderURL(userPrompt, nextCursor, threshold, v.bounds)
	}

	// Render the page content using UI function
	return render(v.ctx, ui.MapViewRenderPage(ads, userID, loc, loaderURL))
}

// TreeView implements the View interface for tree view
type TreeView struct {
	ctx *fiber.Ctx
}

// NewTreeView creates a new tree view
func NewTreeView(ctx *fiber.Ctx) *TreeView {
	return &TreeView{ctx: ctx}
}

func (v *TreeView) GetAds() ([]ad.Ad, string, error) {
	return getAds(v.ctx, nil)
}

func (v *TreeView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	userPrompt := getQueryParam(v.ctx, "q")
	threshold := getThreshold(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL if there are more results
	loaderURL := ui.TreeViewCreateLoaderURL(userPrompt, nextCursor, threshold)

	loc := getLocation(v.ctx)
	return render(v.ctx, ui.TreeViewRenderResults(ads, userID, loc, userPrompt, loaderURL, threshold))
}

func (v *TreeView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	_, userID := getUser(v.ctx)

	// Create loader URL for infinite scroll
	var loaderURL string
	if nextCursor != "" {
		userPrompt := getQueryParam(v.ctx, "q")
		threshold := getThreshold(v.ctx)
		loaderURL = ui.TreeViewCreateLoaderURL(userPrompt, nextCursor, threshold)
	}

	// Render the page content using UI function
	return render(v.ctx, ui.TreeViewRenderPage(ads, userID, loc, loaderURL))
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

// createLoaderURL creates the loader URL for pagination
func createLoaderURL(userPrompt, nextCursor, view string, threshold float64, bounds *GeoBounds) string {
	if nextCursor == "" {
		return ""
	}

	loaderURL := fmt.Sprintf("/search-page?q=%s&cursor=%s&view=%s&threshold=%.1f",
		htmlEscape(userPrompt), htmlEscape(nextCursor), htmlEscape(view), threshold)

	// Add bounding box to loader URL for map view
	if view == "map" && bounds != nil {
		loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
			bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)
	}

	return loaderURL
}

// saveUserSearchAndQueue saves user search and queues user for embedding update
func saveUserSearchAndQueue(userPrompt string, userID int) {
	if userPrompt != "" {
		_ = search.SaveUserSearch(sql.NullInt64{Int64: int64(userID), Valid: userID != 0}, userPrompt)
		if userID != 0 {
			// Queue user for background embedding update
			vector.QueueUserForUpdate(userID)
		}
	}
}

// performGeoBoxSearch performs search with geo bounding box filtering
func performGeoBoxSearch(userPrompt string, currentUser *user.User, cursorStr string, bounds *ui.GeoBounds, threshold float64) ([]ad.Ad, string, error) {
	if bounds == nil {
		return nil, "", fmt.Errorf("bounds cannot be nil for geo box search")
	}

	userID := 0
	if currentUser != nil {
		userID = currentUser.ID
	}
	log.Printf("[performGeoBoxSearch] userPrompt='%s', userID=%d, cursorStr='%s', bounds=%+v", userPrompt, userID, cursorStr, bounds)

	// Build geo filter
	geoFilter := vector.BuildBoundingBoxGeoFilter(bounds.MinLat, bounds.MaxLat, bounds.MinLon, bounds.MaxLon)

	// Use performSearch with the geo filter
	return performSearch(userPrompt, currentUser, cursorStr, threshold, config.QdrantSearchInitialK, geoFilter)
}

// getAds performs the common ad retrieval logic
func getAds(ctx *fiber.Ctx, geoFilter *qdrant.Filter) ([]ad.Ad, string, error) {
	userPrompt := getQueryParam(ctx, "q")
	cursor := getQueryParam(ctx, "cursor")
	threshold := getThreshold(ctx)
	currentUser, _ := CurrentUser(ctx)

	var ads []ad.Ad
	var nextCursor string
	var err error

	// Use QdrantSearchInitialK for geo searches, QdrantSearchPageSize for regular searches
	limit := config.QdrantSearchPageSize
	if geoFilter != nil {
		limit = config.QdrantSearchInitialK
	}

	ads, nextCursor, err = performSearch(userPrompt, currentUser, cursor, threshold, limit, geoFilter)

	if err == nil {
		log.Printf("[getAdsCommon] ads returned: %d", len(ads))
		log.Printf("[getAdsCommon] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

// NewView creates the appropriate view implementation based on view type
func NewView(ctx *fiber.Ctx, viewType string) (View, error) {
	switch viewType {
	case "list":
		return NewListView(ctx), nil
	case "grid":
		return NewGridView(ctx), nil
	case "map":
		bounds := extractMapBounds(ctx)
		return NewMapView(ctx, bounds), nil
	case "tree":
		return NewTreeView(ctx), nil
	default:
		return nil, fmt.Errorf("invalid view type: %s", viewType)
	}
}
