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
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// View interface defines the contract for different view implementations
type View interface {
	// GetAds retrieves ads for this view with appropriate search strategy
	GetAds() ([]ad.Ad, string, error)

	// RenderSearchResults renders the complete search results including container, ads, and pagination
	RenderSearchResults(ads []ad.Ad, nextCursor string) error

	// RenderSearchPage renders just the ads and infinite scroll for pagination
	RenderSearchPage(ads []ad.Ad, nextCursor string) error

	// SaveUserSearch saves user search and queues user for embedding update
	SaveUserSearch()

	// ShouldShowNoResults determines if this view should show a no-results message
	ShouldShowNoResults() bool
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
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	ads, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchPageSize, nil)
	if err == nil {
		log.Printf("[ListView.GetAds] ads returned: %d", len(ads))
		log.Printf("[ListView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *ListView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, userID := getUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "list", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "list", threshold, nil)
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "list", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *ListView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in list view format
	for _, ad := range ads {
		render(v.ctx, ui.AdCardCompactList(ad, loc, currentUser))
		// Add separator after each ad
		render(v.ctx, Div(Class("border-b border-gray-200")))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "list", threshold, nil)
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "list")
		}
	}

	return nil
}

func (v *ListView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	_, userID := getUser(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *ListView) ShouldShowNoResults() bool {
	return true
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
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	ads, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchPageSize, nil)
	if err == nil {
		log.Printf("[GridView.GetAds] ads returned: %d", len(ads))
		log.Printf("[GridView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *GridView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, userID := getUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "grid", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "grid", threshold, nil)
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "grid", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *GridView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in grid view format
	for _, ad := range ads {
		render(v.ctx, ui.AdCardExpandable(ad, loc, currentUser, "grid"))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "grid", threshold, nil)
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "grid")
		}
	}

	return nil
}

func (v *GridView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	_, userID := getUser(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *GridView) ShouldShowNoResults() bool {
	return true
}

// MapView implements the View interface for map view
type MapView struct {
	ctx    *fiber.Ctx
	bounds *GeoBounds
}

// NewMapView creates a new map view
func NewMapView(ctx *fiber.Ctx, bounds *GeoBounds) *MapView {
	return &MapView{ctx: ctx, bounds: bounds}
}

func (v *MapView) GetAds() ([]ad.Ad, string, error) {
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	var ads []ad.Ad
	var nextCursor string
	var err error
	if v.bounds != nil {
		ads, nextCursor, err = performGeoBoxSearch(userPrompt, currentUser, cursor, v.bounds, threshold)
	} else {
		ads, nextCursor, err = performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchInitialK, nil)
	}
	if err == nil {
		log.Printf("[MapView.GetAds] ads returned: %d", len(ads))
		log.Printf("[MapView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *MapView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	_, userID := getUser(v.ctx)
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "map", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "map", threshold, v.bounds)
		// Add bounding box to loader URL for map view
		if v.bounds != nil {
			loaderURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
				v.bounds.MinLat, v.bounds.MaxLat, v.bounds.MinLon, v.bounds.MaxLon)
		}
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "map", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *MapView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in map view format (same as list for now)
	for _, ad := range ads {
		render(v.ctx, ui.AdCardCompactList(ad, loc, currentUser))
		// Add separator after each ad
		render(v.ctx, Div(Class("border-b border-gray-200")))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "map", threshold, v.bounds)
		// Add bounding box to loader URL for map view
		if v.bounds != nil {
			nextPageURL += fmt.Sprintf("&minLat=%.6f&maxLat=%.6f&minLon=%.6f&maxLon=%.6f",
				v.bounds.MinLat, v.bounds.MaxLat, v.bounds.MinLon, v.bounds.MaxLon)
		}
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "map")
		}
	}

	return nil
}

func (v *MapView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	_, userID := getUser(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *MapView) ShouldShowNoResults() bool {
	return false // Map view continues to show empty map
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
	userPrompt := v.ctx.Query("q")
	cursor := v.ctx.Query("cursor")
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
	currentUser, _ := CurrentUser(v.ctx)
	ads, nextCursor, err := performSearch(userPrompt, currentUser, cursor, threshold, config.QdrantSearchPageSize, nil)
	if err == nil {
		log.Printf("[TreeView.GetAds] ads returned: %d", len(ads))
		log.Printf("[TreeView.GetAds] Final ad order: %v", func() []int {
			result := make([]int, len(ads))
			for i, ad := range ads {
				result[i] = ad.ID
			}
			return result
		}())
	}
	return ads, nextCursor, err
}

func (v *TreeView) RenderSearchResults(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)
	userPrompt := v.ctx.Query("q")
	_, userID := getUser(v.ctx)
	newAdButton := renderNewAdButton(userID)
	threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)

	// Check if we should show no results message
	if len(ads) == 0 && v.ShouldShowNoResults() {
		return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), nil, nil, currentUser, loc, "tree", userPrompt, "", fmt.Sprintf("%.1f", threshold)))
	}

	// Create loader URL if there are more results
	var loaderURL string
	if nextCursor != "" {
		loaderURL = createLoaderURL(userPrompt, nextCursor, "tree", threshold, nil)
	}

	return render(v.ctx, ui.SearchResultsContainerWithFlags(newAdButton, ui.SearchSchema(ad.SearchQuery{}), ads, nil, currentUser, loc, "tree", userPrompt, loaderURL, fmt.Sprintf("%.1f", threshold)))
}

func (v *TreeView) RenderSearchPage(ads []ad.Ad, nextCursor string) error {
	loc := getLocation(v.ctx)
	currentUser, _ := CurrentUser(v.ctx)

	// Render ads in tree view format (same as list for now)
	for _, ad := range ads {
		render(v.ctx, ui.AdCardCompactTree(ad, loc, currentUser))
		// Add separator after each ad
		render(v.ctx, Div(Class("border-b border-gray-200")))
	}

	// Add infinite scroll trigger if there are more results
	if nextCursor != "" {
		userPrompt := v.ctx.Query("q")
		threshold := v.ctx.QueryFloat("threshold", config.QdrantSearchThreshold)
		nextPageURL := createLoaderURL(userPrompt, nextCursor, "tree", threshold, nil)
		if nextPageURL != "" {
			renderInfiniteScrollTrigger(v.ctx, nextPageURL, "tree")
		}
	}

	return nil
}

func (v *TreeView) SaveUserSearch() {
	userPrompt := v.ctx.Query("q")
	_, userID := getUser(v.ctx)
	saveUserSearchAndQueue(userPrompt, userID)
}

func (v *TreeView) ShouldShowNoResults() bool {
	return true
}

// extractMapBounds extracts geographic bounding box parameters for map view
func extractMapBounds(c *fiber.Ctx) *GeoBounds {
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

	return &GeoBounds{
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

// renderInfiniteScrollTrigger renders the infinite scroll trigger for pagination
func renderInfiniteScrollTrigger(c *fiber.Ctx, nextPageURL, view string) {
	if nextPageURL == "" {
		log.Printf("[renderInfiniteScrollTrigger] No infinite scroll trigger - no more results")
		return
	}

	log.Printf("[renderInfiniteScrollTrigger] Adding infinite scroll trigger with URL: %s", nextPageURL)

	// Create trigger that matches the view style
	render(c, Div(
		Class("h-4"),
		g.Attr("hx-get", nextPageURL),
		g.Attr("hx-trigger", "revealed"),
		g.Attr("hx-swap", "outerHTML"),
	))
}

// Render new ad button based on user login
func renderNewAdButton(userID int) g.Node {
	if userID != 0 {
		return ui.StyledLink("New Ad", "/new-ad", ui.ButtonPrimary)
	}
	return ui.StyledLinkDisabled("New Ad", ui.ButtonPrimary)
}

// saveUserSearchAndQueue saves user search and queues user for embedding update
func saveUserSearchAndQueue(userPrompt string, userID int) {
	if userPrompt != "" {
		_ = search.SaveUserSearch(sql.NullInt64{Int64: int64(userID), Valid: userID != 0}, userPrompt)
		if userID != 0 {
			// Queue user for background embedding update
			vector.GetEmbeddingQueue().QueueUserForUpdate(userID)
		}
	}
}

// performGeoBoxSearch performs search with geo bounding box filtering
func performGeoBoxSearch(userPrompt string, currentUser *user.User, cursorStr string, bounds *GeoBounds, threshold float64) ([]ad.Ad, string, error) {
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
