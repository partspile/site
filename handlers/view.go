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
